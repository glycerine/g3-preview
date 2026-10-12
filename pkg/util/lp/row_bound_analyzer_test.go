package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

type lpRowEndpoint struct {
	low, upper             *big.Rat
	lowStrict, upperStrict bool
}

// Eliminate each target directly from the equation. This oracle does not use
// the implementation's total-minus-own-term identity or missing-column counters.
func lpRowOracle(coeff []*big.Rat, endpoints []lpRowEndpoint, row uint32) []lpPropagationOracle {
	var want []lpPropagationOracle
	for _, maximum := range []bool{true, false} {
		for j, aj := range coeff {
			sum := new(big.Rat)
			available, strict := true, false
			for k, ak := range coeff {
				if k == j {
					continue
				}
				ep := endpoints[k]
				value, str := ep.low, ep.lowStrict
				if (ak.Sign() > 0) == maximum {
					value, str = ep.upper, ep.upperStrict
				}
				if value == nil {
					available = false
					break
				}
				sum.Add(sum, new(big.Rat).Mul(ak, value))
				strict = strict || str
			}
			if !available {
				continue
			}
			value := new(big.Rat).Quo(new(big.Rat).Neg(sum), aj)
			want = append(want, lpPropagationOracle{lpPropagationKey{uint32(j), (aj.Sign() > 0) == maximum}, value, aj.Sign() > 0, strict, row})
		}
	}
	return want
}
func lpCheckRowBounds(t *testing.T, h *lpPairHarness, p *LARBoundPropagator, want []lpPropagationOracle) {
	t.Helper()
	if len(p.FoundBounds) != len(want) {
		t.Fatalf("bounds %d, want %d", len(p.FoundBounds), len(want))
	}
	for i, w := range want {
		got := p.FoundBounds[i]
		lpSparseRationalCheck(t, h, &got.Bound, w.value)
		if got.Column != w.key.column || got.IsLowBound != w.key.low || got.CoefficientIsPositive != w.positive || got.Strict != w.strict || got.RowOrTermIndex != w.row {
			t.Fatalf("bound %d evidence: %+v want %+v", i, got, w)
		}
	}
}
func TestLARRowBoundAnalyzerIndependentEliminationOracle(t *testing.T) {
	h := newLPPairHarness(t)
	r := rand.New(rand.NewPCG(9172, 319))
	for world := 0; world < 240; world++ {
		n := 1 + r.IntN(6)
		columns := make([]uint32, n)
		bounds := NewLARBounds(h.m)
		matrix := NewRationalStaticMatrix(h.m, 1, uint32(n))
		coeff := make([]*big.Rat, n)
		endpoints := make([]lpRowEndpoint, n)
		value := h.pair("0", "0")
		for j := 0; j < n; j++ {
			columns[j] = uint32(j)
			h.must(bounds.AppendFreeColumn(91))
			c := int64(1 + r.IntN(9))
			if r.IntN(2) == 0 {
				c = -c
			}
			coeff[j] = big.NewRat(c, int64(1+r.IntN(7)))
			h.must(h.m.SetString(&value.X, coeff[j].RatString()))
			h.must(matrix.Set(0, uint32(j), &value.X))
			typ := ColumnBoxed
			switch world % 5 {
			case 1:
				if j == 0 {
					typ = ColumnFree
				}
			case 2:
				if j < 2 {
					typ = ColumnFree
				}
			case 3:
				typ = []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed}[r.IntN(5)]
			case 4:
				if c > 0 {
					typ = ColumnLowBound
				} else {
					typ = ColumnUpperBound
				}
			}
			lo := big.NewRat(int64(r.IntN(21)-10), int64(1+r.IntN(11)))
			up := new(big.Rat).Add(lo, big.NewRat(int64(1+r.IntN(17)), int64(1+r.IntN(13))))
			ep := lpRowEndpoint{}
			if typ == ColumnFixed {
				h.must(h.m.SetString(&value.X, lo.RatString()))
				h.must(bounds.Update(uint32(j), LConstraintEQ, &value.X, 19))
				ep.low, ep.upper = lo, lo
			} else {
				if typ == ColumnBoxed || typ == ColumnLowBound {
					ep.low, ep.lowStrict = lo, r.IntN(2) == 0
					kind := LConstraintGE
					if ep.lowStrict {
						kind = LConstraintGT
					}
					h.must(h.m.SetString(&value.X, lo.RatString()))
					h.must(bounds.Update(uint32(j), kind, &value.X, 20))
				}
				if typ == ColumnBoxed || typ == ColumnUpperBound {
					ep.upper, ep.upperStrict = up, r.IntN(2) == 0
					kind := LConstraintLE
					if ep.upperStrict {
						kind = LConstraintLT
					}
					h.must(h.m.SetString(&value.X, up.RatString()))
					h.must(bounds.Update(uint32(j), kind, &value.X, 21))
				}
			}
			endpoints[j] = ep
		}
		it, err := NewRationalRowIterator(matrix, 0)
		h.must(err)
		var terms []*LARTerm
		p := NewLARBoundPropagator(bounds, &columns, &terms)
		rhs := h.pair("0", "0")
		a, err := AnalyzeLARRow(it, rhs, uint32(world), p)
		h.must(err)
		lpCheckRowBounds(t, h, p, lpRowOracle(coeff, endpoints, uint32(world)))
		h.must(a.Close())
		h.must(it.Close())
		h.must(p.Close())
		h.must(matrix.Close())
		h.must(bounds.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARRowBoundAnalyzerCursorRHSAndBorrowedLifetime(t *testing.T) {
	h := newLPPairHarness(t)
	columns := []uint32{0, 1, 2}
	b := NewLARBounds(h.m)
	for range columns {
		h.must(b.AppendFreeColumn(0))
	}
	var terms []*LARTerm
	p := NewLARBoundPropagator(b, &columns, &terms)
	m := NewRationalStaticMatrix(h.m, 1, 3)
	v := h.pair("2", "3")
	h.must(m.Set(0, 0, &v.X))
	h.must(m.Set(0, 1, &v.Y))
	h.must(m.Set(0, 2, &v.X))
	it, err := NewRationalRowIterator(m, 0)
	h.must(err)
	rhs := h.pair("7/11", "9/13")
	a, err := NewLARRowBoundAnalyzer(it, rhs, 51, p)
	h.must(err)
	h.must(h.m.SetInt32(&rhs.X, 100))
	h.must(a.Analyze())
	if it.offset != 2 || a.ColumnOfU != -2 || a.ColumnOfL != -2 || len(p.FoundBounds) != 0 {
		t.Fatal("scan must stop at second free column")
	}
	h.must(a.Analyze())
	if it.offset != 2 {
		t.Fatal("persistent missing counters should skip second scan")
	}
	lpSparseRationalCheck(t, h, &a.RightSide.X, big.NewRat(7, 11))
	h.must(a.Close())
	h.must(it.Reset())
	j, ok, err := it.Next(&v.X)
	h.must(err)
	if !ok || j != 0 {
		t.Fatal("borrowed iterator closed")
	}
	h.must(it.Close())
	h.must(p.Close())
	h.must(m.Close())
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARRowBoundAnalyzerPhysicalFailurePrefixes(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, mode := range []string{"construct", "all", "unique"} {
			succeeded, failures := false, 0
			conditionFailures := 0
			for ordinal := int64(0); ordinal < 800; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%s/%d", direct, mode, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					columns := []uint32{0, 1}
					b := NewLARBounds(h.m)
					for range columns {
						h.must(b.AppendFreeColumn(0))
					}
					var terms []*LARTerm
					p := NewLARBoundPropagator(b, &columns, &terms)
					matrix := NewRationalStaticMatrix(h.m, 1, 2)
					large := strings.Repeat("1234567890", 100)
					v := h.pair(large+"/7", "-"+large+"/13")
					h.must(matrix.Set(0, 0, &v.X))
					h.must(matrix.Set(0, 1, &v.Y))
					lo := h.pair("-"+large+"/17", large+"/19")
					for j := uint32(0); j < 2; j++ {
						if mode == "unique" && j == 0 {
							continue
						}
						h.must(b.Update(j, LConstraintGT, &lo.X, 11))
						h.must(b.Update(j, LConstraintLT, &lo.Y, 12))
					}
					rhs := h.pair("0", "0")
					if mode == "construct" {
						rhs = v
					}
					it, err := NewRationalRowIterator(matrix, 0)
					h.must(err)
					var a *LARRowBoundAnalyzer
					if mode == "construct" {
						h.budget.FailAfter(ordinal)
						a, err = NewLARRowBoundAnalyzer(it, rhs, 9, p)
					} else {
						a, err = NewLARRowBoundAnalyzer(it, rhs, 9, p)
						h.must(err)
						h.budget.FailAfter(ordinal)
						err = a.Analyze()
					}
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if a.condition != nil {
							conditionFailures++
							if mode != "unique" || p.NumberOfFoundBounds() != 0 {
								t.Fatal("implicit pair refusal followed bound publication")
							}
						}
						if !errors.Is(a.Analyze(), ErrExactVectorPending) {
							t.Fatal("failed owner accepted retry")
						}
					} else {
						done = true
						if mode == "construct" {
							lpSparseRationalCheck(t, h, &a.RightSide.X, mustLPRat(t, large+"/7"))
							lpSparseRationalCheck(t, h, &a.RightSide.Y, mustLPRat(t, "-"+large+"/13"))
						} else {
							coeff := []*big.Rat{mustLPRat(t, large+"/7"), mustLPRat(t, "-"+large+"/13")}
							ep := lpRowEndpoint{mustLPRat(t, "-"+large+"/17"), mustLPRat(t, large+"/19"), true, true}
							endpoints := []lpRowEndpoint{ep, ep}
							if mode == "unique" {
								endpoints[0] = lpRowEndpoint{}
							}
							lpCheckRowBounds(t, h, p, lpRowOracle(coeff, endpoints, 9))
						}
					}
					// Compare actual stopped numeric generations before cleanup.
					// Neither borrowed child's new calls may become our receipts.
					for k, generation := range a.calls.numeric.raw {
						if generation != 0 && (generation == it.calls.raw[k] || generation == p.calls.numeric.raw[k]) {
							t.Fatal("borrowed raw generation acquired by analyzer")
						}
					}
					if a.calls.division != 0 && a.calls.division == p.calls.division {
						t.Fatal("borrowed division generation acquired by analyzer")
					}
					// The analyzer releases only its own values/receipts. Iterator and
					// propagator stopped copies remain owned by those concrete children.
					h.must(a.Close())
					h.must(it.Close())
					h.must(p.Close())
					lpSparseRationalCheck(t, h, &matrix.Rows[0][0].Value, mustLPRat(t, large+"/7"))
					h.must(matrix.Close())
					h.must(b.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					succeeded = true
					break
				}
			}
			if !succeeded || failures == 0 {
				t.Fatal(direct, mode, succeeded, failures)
			}
			if mode == "unique" && conditionFailures == 0 {
				t.Fatal("implicit scalar-to-pair copy refusal not reached", direct)
			}
		}
	}
}

func TestLARRowBoundAnalyzerSourceRHSProfilesAndPivot(t *testing.T) {
	for _, unique := range []bool{false, true} {
		t.Run(fmt.Sprint(unique), func(t *testing.T) {
			h := newLPPairHarness(t)
			columns := []uint32{0, 1}
			b := NewLARBounds(h.m)
			for range columns {
				h.must(b.AppendFreeColumn(0))
			}
			v := h.pair("2", "5")
			for j := uint32(0); j < 2; j++ {
				if unique && j == 0 {
					continue
				}
				h.must(b.Update(j, LConstraintGE, &v.X, 1))
				h.must(b.Update(j, LConstraintLE, &v.Y, 2))
			}
			// Actual pivot iterator emits x0 - 2*x1; Size excludes the basis entry.
			vec := NewIndexedRationalVector(h.m, 2)
			h.must(h.m.SetInt32(&v.X, -2))
			h.must(vec.SetValue(&v.X, 1))
			it := NewRationalPivotRowIterator(vec, 0)
			var terms []*LARTerm
			p := NewLARBoundPropagator(b, &columns, &terms)
			rhs := h.pair("7", "19")
			a, err := NewLARRowBoundAnalyzer(it, rhs, 77, p)
			h.must(err)
			h.must(a.Analyze())
			ep := []lpRowEndpoint{{big.NewRat(2, 1), big.NewRat(5, 1), false, false}, {big.NewRat(2, 1), big.NewRat(5, 1), false, false}}
			if unique {
				ep[0] = lpRowEndpoint{}
			}
			want := lpRowOracle([]*big.Rat{big.NewRat(1, 1), big.NewRat(-2, 1)}, ep, 77)
			if unique {
				for i := range want {
					want[i].value.Sub(want[i].value, big.NewRat(7, 1))
				}
			}
			lpCheckRowBounds(t, h, p, want)
			h.must(a.Close())
			h.must(it.Close())
			h.must(p.Close())
			h.must(vec.Close())
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARRowBoundAnalyzerForeignIteratorAndCurrentCursor(t *testing.T) {
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	columns := []uint32{0}
	b := NewLARBounds(h.m)
	h.must(b.AppendFreeColumn(0))
	var terms []*LARTerm
	p := NewLARBoundPropagator(b, &columns, &terms)
	foreign := NewIndexedRationalVector(other.m, 1)
	it := NewRationalIndexedIterator(foreign)
	rhs := h.pair("0", "0")
	a, err := NewLARRowBoundAnalyzer(it, rhs, 0, p)
	if !errors.Is(err, ErrExactVectorOwner) || it.offset != 0 {
		t.Fatal("foreign numeric owner was read", err)
	}
	h.must(a.Close())
	h.must(it.Close())
	other.must(foreign.Close())
	// Consume the only free-column coefficient before construction. The initial
	// scan must stay exhausted; helper scans reset and derive both zero bounds.
	vec := NewIndexedRationalVector(h.m, 1)
	one := h.pair("1", "0")
	h.must(vec.SetValue(&one.X, 0))
	own := NewRationalIndexedIterator(vec)
	_, ok, err := own.NextIndex()
	h.must(err)
	if !ok {
		t.Fatal("missing coefficient")
	}
	a, err = NewLARRowBoundAnalyzer(own, rhs, 13, p)
	h.must(err)
	// In debug the source helper availability assertion rejects the skipped free
	// variable. Native uses its initialized zero endpoints after erased asserts.
	err = a.Analyze()
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) {
			t.Fatal(err)
		}
	} else {
		h.must(err)
		if len(p.FoundBounds) != 2 {
			t.Fatal("initial cursor was reset")
		}
	}
	h.must(a.Close())
	h.must(own.Close())
	h.must(p.Close())
	h.must(vec.Close())
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
	lpExactVectorNamedCensus(t, other, nil)
}

func TestLARRowBoundAnalyzerLeavesHookInvocationOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 50 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			columns := []uint32{0}
			b := NewLARBounds(h.m)
			h.must(b.AppendFreeColumn(0))
			var terms []*LARTerm
			p := NewLARBoundPropagator(b, &columns, &terms)
			matrix := NewRationalStaticMatrix(h.m, 1, 1)
			one := h.pair("1", "0")
			h.must(matrix.Set(0, 0, &one.X))
			it, err := NewRationalRowIterator(matrix, 0)
			h.must(err)
			rhs := h.pair("0", "0")
			a, err := NewLARRowBoundAnalyzer(it, rhs, 0, p)
			h.must(err)
			hook := &lpOwningInterest{h: h, ordinal: ordinal, operands: h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")}
			p.Interest = hook
			err = a.Analyze()
			state := h.m.DivisionInvocationState()
			if state.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) || a.calls.pending() || p.calls.pending() {
					t.Fatal("hook receipts acquired by borrowed callers", err)
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(a.Close())
			h.must(p.Close())
			h.must(it.Close())
			if state.Active && h.m.DivisionInvocationState() != state {
				t.Fatal("analyzer consumed hook division")
			}
			h.must(hook.calls.cleanupRaw(h.m))
			h.must(hook.calls.cleanupRoots(h.m))
			h.must(matrix.Close())
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no stopped provider division observed")
	}
}
