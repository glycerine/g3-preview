package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

type lpPropagationKey struct {
	column uint32
	low    bool
}
type lpPropagationOracle struct {
	key              lpPropagationKey
	value            *big.Rat
	positive, strict bool
	row              uint32
}

func lpPropagationModel(t *testing.T, h *lpPairHarness, columns []uint32) (*LARBounds, []*LARTerm) {
	t.Helper()
	b := NewLARBounds(h.m)
	for range columns {
		h.must(b.AppendFreeColumn(InvalidLARIndex))
	}
	terms := []*LARTerm{NewLARTerm(h.m), NewLARTerm(h.m)}
	p := h.pair("7/13", "-11/17")
	h.must(terms[0].SetValue(&p.X))
	h.must(terms[1].SetValue(&p.Y))
	return b, terms
}

func TestLARBoundPropagatorIndependentStrongestOracle(t *testing.T) {
	h := newLPPairHarness(t)
	columns := []uint32{41, LARTermStartIndex, 77, LARTermStartIndex + 1, LARTermStartIndex}
	b, terms := lpPropagationModel(t, h, columns)
	p := NewLARBoundPropagator(b, &columns, &terms)
	value := h.pair("0", "0")
	want := []lpPropagationOracle{}
	lookup := make(map[lpPropagationKey]int)
	r := rand.New(rand.NewPCG(55593, 71))
	for step := 0; step < 1200; step++ {
		j := uint32(r.IntN(len(columns)))
		external := j
		x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(r.IntN(101)-50)), uint(r.IntN(140))), big.NewInt(int64(r.IntN(47)+1)))
		low, positive, strict := r.IntN(2) == 0, r.IntN(2) == 0, r.IntN(2) == 0
		if columns[j] >= LARTermStartIndex {
			external = columns[j]
		}
		key := lpPropagationKey{external, low}
		if k, ok := lookup[key]; ok && r.IntN(3) == 0 {
			x.Set(want[k].value)
			if external == LARTermStartIndex {
				x.Sub(x, big.NewRat(7, 13))
			}
			if external == LARTermStartIndex+1 {
				x.Sub(x, big.NewRat(-11, 17))
			}
		}
		h.must(h.m.SetString(&value.X, x.RatString()))
		h.must(p.TryAddBound(&value.X, j, low, positive, uint32(step), strict))
		if external == LARTermStartIndex {
			x.Add(x, big.NewRat(7, 13))
		}
		if external == LARTermStartIndex+1 {
			x.Add(x, big.NewRat(-11, 17))
		}
		candidate := lpPropagationOracle{key, x, positive, strict, uint32(step)}
		if k, ok := lookup[key]; !ok {
			lookup[key] = len(want)
			want = append(want, candidate)
		} else {
			c := x.Cmp(want[k].value)
			if low && c > 0 || !low && c < 0 || c == 0 && strict && !want[k].strict {
				want[k] = candidate
			}
		}
		h.must(h.m.SetInt32(&value.X, 19))
		if int(p.NumberOfFoundBounds()) != len(want) {
			t.Fatal("entry count", step, p.NumberOfFoundBounds(), len(want))
		}
		for i, w := range want {
			got := p.FoundBounds[i]
			if got.Column != w.key.column || got.IsLowBound != w.key.low || got.CoefficientIsPositive != w.positive || got.Strict != w.strict || got.RowOrTermIndex != w.row {
				t.Fatal("evidence/order", step, i, got, w)
			}
			lpSparseRationalCheck(t, h, &got.Bound, w.value)
		}
	}
	// The borrowed directory is live: newly appended mappings and terms are
	// observed without recreating the propagator or copying a model owner.
	columns = append(columns, LARTermStartIndex+2)
	h.must(b.AppendFreeColumn(InvalidLARIndex))
	terms = append(terms, NewLARTerm(h.m))
	h.must(terms[2].SetValue(&value.X))
	h.must(p.TryAddBound(&value.X, 5, true, true, 91, true))
	last := p.FoundBounds[len(p.FoundBounds)-1]
	if last.Column != LARTermStartIndex+2 {
		t.Fatal("stale directory")
	}
	lpSparseRationalCheck(t, h, &last.Bound, big.NewRat(38, 1))
	h.must(p.Close())
	for _, term := range terms {
		h.must(term.Close())
	}
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

type lpPropagationHooks struct {
	h         *lpPairHarness
	accept    bool
	err       error
	columns   []uint32
	kinds     []LConstraintKind
	values    []string
	witnesses []uint32
}

func (h *lpPropagationHooks) BoundIsInteresting(j uint32, k LConstraintKind, v *numeral.Rational) (bool, error) {
	text, err := h.h.m.String(v)
	if err != nil {
		return false, err
	}
	h.columns = append(h.columns, j)
	h.kinds = append(h.kinds, k)
	h.values = append(h.values, text)
	return h.accept, h.err
}
func (h *lpPropagationHooks) Consume(v *numeral.Rational, w uint32) error {
	text, err := h.h.m.String(v)
	if err != nil {
		return err
	}
	h.values = append(h.values, text)
	h.witnesses = append(h.witnesses, w)
	return h.err
}

func TestLARBoundPropagatorHooksBorrowedModelAndStreams(t *testing.T) {
	h := newLPPairHarness(t)
	columns := []uint32{17, LARTermStartIndex}
	b, terms := lpPropagationModel(t, h, columns)
	p := NewLARBoundPropagator(b, &columns, &terms)
	v := h.pair("1/13", "0")
	hook := &lpPropagationHooks{h: h}
	p.Interest = hook
	for _, low := range []bool{false, true} {
		for _, strict := range []bool{false, true} {
			h.must(p.TryAddBound(&v.X, 1, low, true, 31, strict))
		}
	}
	if p.NumberOfFoundBounds() != 0 || len(p.lower) != 0 || len(p.upper) != 0 {
		t.Fatal("rejected bound published")
	}
	for i, k := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT} {
		if hook.columns[i] != LARTermStartIndex || hook.kinds[i] != k || hook.values[i] != "8/13" {
			t.Fatal("filter preceded shift", hook)
		}
	}
	hook.accept = true
	h.must(p.TryAddBound(&v.X, 1, true, false, 32, false))
	if p.NumberOfFoundBounds() != 1 {
		t.Fatal("accepted bound")
	}
	// Source accessors borrow the current actual bounds, including strict Y.
	h.must(b.Update(0, LConstraintGT, &v.X, 33))
	h.must(h.m.SetInt32(&v.Y, 3))
	h.must(b.Update(0, LConstraintLE, &v.Y, 34))
	typ, err := p.ColumnType(0)
	h.must(err)
	if typ != ColumnBoxed {
		t.Fatal(typ)
	}
	lo, err := p.LowBound(0)
	h.must(err)
	h.check(lo, big.NewRat(1, 13), big.NewRat(1, 1))
	up, err := p.UpperBound(0)
	h.must(err)
	h.check(up, big.NewRat(3, 1), big.NewRat(0, 1))
	p.Evidence = hook
	h.must(p.Consume(&v.X, 72))
	if hook.witnesses[0] != 72 || hook.values[len(hook.values)-1] != "1/13" {
		t.Fatal("virtual evidence override")
	}
	failure := errors.New("evidence refused")
	hook.err = failure
	if err = p.Consume(&v.X, 73); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err = p.TryAddBound(&v.X, 0, true, true, 91, true); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if p.NumberOfFoundBounds() != 1 || p.ready() != ErrExactVectorPending {
		t.Fatal("hook error publication")
	}
	h.must(p.Close())
	// Closing the propagator does not destroy its borrowed model or hooks.
	lpSparseRationalCheck(t, h, &terms[0].Value, big.NewRat(7, 13))
	lo, err = p.LowBound(0)
	h.must(err)
	h.check(lo, big.NewRat(1, 13), big.NewRat(1, 1))
	if b.Types.Size() != 2 || terms[0].Size() != 0 || len(hook.columns) != 6 {
		t.Fatal("borrowed owners")
	}
	for _, term := range terms {
		h.must(term.Close())
	}
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARBoundPropagatorPhysicalFailurePrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"new-low", "new-upper", "replace-low", "replace-upper", "strict-low", "strict-upper", "term"} {
			failures, succeeded := 0, false
			for ordinal := int64(0); ordinal < 240; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					columns := []uint32{19, LARTermStartIndex}
					b, terms := lpPropagationModel(t, h, columns)
					p := NewLARBoundPropagator(b, &columns, &terms)
					x := strings.Repeat("1234567890", 100) + "/7"
					y := strings.Repeat("9876543210", 100) + "/13"
					v := h.pair(x, y)
					low := strings.HasSuffix(op, "low")
					j := uint32(0)
					strict := strings.HasPrefix(op, "strict")
					if strings.HasPrefix(op, "replace") || strict {
						old := &v.X
						if op == "replace-upper" {
							old = &v.Y
						}
						h.must(p.TryAddBound(old, j, low, true, 11, false))
					}
					candidate := &v.Y
					if strict || op == "replace-upper" {
						candidate = &v.X
					}
					if op == "term" {
						j = 1
						low = true
						h.must(terms[0].SetValue(&v.X))
					}
					before := p.NumberOfFoundBounds()
					h.budget.FailAfter(ordinal)
					err := p.TryAddBound(candidate, j, low, false, 71, strict)
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if p.ready() != ErrExactVectorPending || p.NumberOfFoundBounds() != before {
							t.Fatal("lost partial owner or early append", p.NumberOfFoundBounds(), before)
						}
						if e := p.TryAddBound(candidate, j, low, true, 72, false); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						lookup := p.upper
						if low {
							lookup = p.lower
						}
						key := j
						if j == 1 {
							key = LARTermStartIndex
						}
						if p.temporary != nil {
							if k, ok := lookup[key]; !ok || k != 0 {
								t.Fatal("map prefix missing", lookup)
							}
						}
					} else {
						done = true
						if p.NumberOfFoundBounds() != 1 {
							t.Fatal(p.NumberOfFoundBounds())
						}
						got := p.FoundBounds[0]
						if got.RowOrTermIndex != 71 || got.IsLowBound != low || got.Strict != strict || got.CoefficientIsPositive {
							t.Fatal("replacement evidence", got)
						}
						text, e := h.m.String(candidate)
						h.must(e)
						want, _ := new(big.Rat).SetString(text)
						if op == "term" {
							xx, _ := new(big.Rat).SetString(x)
							want.Add(want, xx)
						}
						lpSparseRationalCheck(t, h, &got.Bound, want)
					}
					h.must(p.Close())
					for _, term := range terms {
						h.must(term.Close())
					}
					h.must(b.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					succeeded = true
					break
				}
			}
			if failures == 0 || !succeeded {
				t.Fatal("incomplete physical campaign", direct, op, failures, succeeded)
			}
		}
	}
}

func TestLARBoundPropagatorMappingAndOwnerGuards(t *testing.T) {
	for _, mode := range []string{"column", "missing-term", "nil-term", "foreign-term", "stopped-term"} {
		t.Run(mode, func(t *testing.T) {
			h := newLPPairHarness(t)
			columns := []uint32{17, LARTermStartIndex}
			b, terms := lpPropagationModel(t, h, columns)
			p := NewLARBoundPropagator(b, &columns, &terms)
			v := h.pair("2/7", "0")
			j := uint32(0)
			want := ErrBounds
			var foreign *lpPairHarness
			var foreignTerm *LARTerm
			var vector *IndexedRationalVector
			switch mode {
			case "column":
				j = 2
			case "missing-term":
				j = 1
				columns[1] = LARTermStartIndex + 9
			case "nil-term":
				j = 1
				columns[1] = LARTermStartIndex + 2
				terms = append(terms, nil)
			case "foreign-term":
				j = 1
				foreign = newLPPairHarness(t)
				foreignTerm = NewLARTerm(foreign.m)
				terms = append(terms, foreignTerm)
				columns[1] = LARTermStartIndex + 2
				want = ErrExactVectorOwner
			case "stopped-term":
				j = 1
				vector = NewIndexedRationalVector(h.m, 1)
				h.must(terms[0].AddCoefficient(0, &v.X))
				h.must(h.m.SetInt32(&v.Y, 1))
				h.must(vector.SetValue(&v.Y, 0))
				if e := terms[0].Substitute(0, vector); e != ErrLARTermPivotErased {
					t.Fatal(e)
				}
				want = ErrExactVectorPending
			}
			if err := p.TryAddBound(&v.X, j, true, true, 71, false); err != want {
				t.Fatal(mode, err, want)
			}
			h.must(p.Close())
			for _, term := range terms {
				if term != nil && term != foreignTerm {
					h.must(term.Close())
				}
			}
			if foreignTerm != nil {
				foreign.must(foreignTerm.Close())
			}
			if vector != nil {
				h.must(vector.Close())
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

// This hook owns an actual stopped arithmetic call, like the retained SMT
// virtual provider. The propagator must not acquire its invocation receipts.
type lpOwningInterest struct {
	h        *lpPairHarness
	operands *RationalPair
	ordinal  int64
	calls    exactEtaCalls
}

func (h *lpOwningInterest) BoundIsInteresting(_ uint32, _ LConstraintKind, _ *numeral.Rational) (bool, error) {
	before, err := h.calls.begin(h.h.m)
	if err != nil {
		return false, err
	}
	h.h.budget.FailAfter(h.ordinal)
	err = h.h.m.Div(&h.operands.X, &h.operands.Y, &h.operands.Y)
	h.h.budget.FailAfter(-1)
	return false, h.calls.after(h.h.m, before, err)
}

func TestLARBoundPropagatorLeavesHookInvocationOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 50 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			columns := []uint32{19}
			b, terms := lpPropagationModel(t, h, columns)
			p := NewLARBoundPropagator(b, &columns, &terms)
			v := h.pair("1/13", "0")
			hook := &lpOwningInterest{h: h, ordinal: ordinal, operands: h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")}
			p.Interest = hook
			err := p.TryAddBound(&v.X, 0, true, true, 71, false)
			state := h.m.DivisionInvocationState()
			if state.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) || p.calls.pending() || p.NumberOfFoundBounds() != 0 || len(p.lower) != 0 {
					t.Fatal("hook failure ownership/publication", err)
				}
				h.must(p.Close())
				if h.m.DivisionInvocationState() != state {
					t.Fatal("propagator consumed separately owned hook division")
				}
			} else {
				if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(p.Close())
			}
			h.must(hook.calls.cleanupRaw(h.m))
			h.must(hook.calls.cleanupRoots(h.m))
			for _, term := range terms {
				h.must(term.Close())
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual hook-owned stopped division")
	}
}
