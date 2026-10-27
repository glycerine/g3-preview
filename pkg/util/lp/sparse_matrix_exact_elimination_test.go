package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// The source debug allocator makes every object a real physical allocation;
// default pooled campaigns below remain separate. This exposes copy-action
// failure prefixes that can reuse warmed storage in the normal allocator.
func lpNewDirectSparseHarness(t *testing.T) *lpPairHarness {
	t.Helper()
	b := arena.NewBudget(8 << 20)
	m, e := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{PageSize: 4, Budget: b}, Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 4, Budget: b}},
		Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
	}, Synchronized: true})
	if e != nil {
		t.Fatal(e)
	}
	h := &lpPairHarness{t: t, m: m, budget: b}
	t.Cleanup(func() {
		b.FailAfter(-1)
		for i := len(h.pairs) - 1; i >= 0; i-- {
			if e := h.pairs[i].Del(m); e != nil {
				t.Error(e)
			}
		}
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("direct fixture retained bytes", b.Used())
		}
	})
	return h
}

func lpNewExactSparse(t *testing.T, h *lpPairHarness, n uint32) *RationalSparseMatrix {
	t.Helper()
	a := NewRationalStaticMatrix(h.m, n, n)
	basis := make([]uint32, n)
	for i := range basis {
		basis[i] = uint32(i)
	}
	m, e := NewRationalSparseBasisMatrix(a, basis)
	h.must(e)
	h.must(a.Close())
	t.Cleanup(func() { h.must(m.Close()) })
	return m
}
func lpExactSparseCheck(t *testing.T, h *lpPairHarness, m *RationalSparseMatrix, want map[IndexPair]*big.Rat) {
	t.Helper()
	seen := make(map[IndexPair]bool)
	var owned []*numeral.Rational
	for i := range m.Rows {
		for k := range m.Rows[i] {
			v := &m.Rows[i][k]
			p := IndexPair{uint32(i), v.Index}
			if seen[p] || want[p] == nil {
				t.Fatal("unexpected coefficient", p)
			}
			seen[p] = true
			lpSparseRationalCheck(t, h, &v.Value, want[p])
			owned = append(owned, &v.Value)
			if uint64(v.Index) >= uint64(len(m.Columns)) || uint64(v.Other) >= uint64(len(m.Columns[v.Index].Values)) {
				t.Fatal("row bounds")
			}
			c := &m.Columns[v.Index].Values[v.Other]
			if c.Index != uint32(i) || c.Other != uint32(k) {
				t.Fatal("column backlink")
			}
		}
	}
	count := 0
	for j := range m.Columns {
		for k := range m.Columns[j].Values {
			v := &m.Columns[j].Values[k]
			count++
			if uint64(v.Index) >= uint64(len(m.Rows)) || uint64(v.Other) >= uint64(len(m.Rows[v.Index])) {
				t.Fatal("column bounds")
			}
			r := &m.Rows[v.Index][v.Other]
			if r.Index != uint32(j) || r.Other != uint32(k) {
				t.Fatal("row backlink")
			}
			lpSparseRationalCheck(t, h, &v.Value, want[IndexPair{v.Index, uint32(j)}])
			owned = append(owned, &v.Value)
		}
	}
	if len(seen) != len(want) || count != len(want) || int(m.active) != len(want) {
		t.Fatal("counts", len(seen), count, len(want), m.active)
	}
	lpExactVectorNamedCensus(t, h, owned)
}

func TestLPExactSparseIndependentMutationOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 5)
	p := h.pair("0", "0")
	rng := rand.New(rand.NewPCG(879, 777))
	want := make(map[IndexPair]*big.Rat)
	for step := 0; step < 1200; step++ {
		i, j := rng.Uint32N(5), rng.Uint32N(5)
		key := IndexPair{i, j}
		switch rng.IntN(4) {
		case 0, 1:
			if want[key] != nil {
				continue
			}
			x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(19)-9)), uint(rng.IntN(120))), big.NewInt(int64(1+rng.IntN(29))))
			h.must(h.m.SetString(&p.X, x.RatString()))
			h.must(m.AddNewElement(i, j, &p.X))
			want[key] = x
		case 2:
			if len(m.Rows[i]) > 0 {
				k := uint32(rng.IntN(len(m.Rows[i])))
				delete(want, IndexPair{i, m.Rows[i][k].Index})
				h.must(m.RemoveElement(i, k))
			}
		case 3:
			divisor := big.NewRat(int64(1+rng.IntN(9)), int64(1+rng.IntN(7)))
			h.must(h.m.SetString(&p.Y, divisor.RatString()))
			h.must(m.DivideRow(i, &p.Y, nil))
			for p, v := range want {
				if p.I == i {
					v.Quo(v, divisor)
				}
			}
		}
		lpExactSparseCheck(t, h, m, want)
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseIndependentPivotOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(975, 67))
	fills, cancels, empty := 0, 0, 0
	for world := 0; world < 160; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 4)
			p := h.pair("0", "0")
			want := make(map[IndexPair]*big.Rat)
			for _, k := range rng.Perm(16) {
				if rng.IntN(3) == 0 {
					continue
				}
				x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(rng.IntN(7)-3)), uint(rng.IntN(100))), big.NewInt(int64(1+rng.IntN(11))))
				i, j := uint32(k/4), uint32(k%4)
				h.must(h.m.SetString(&p.X, x.RatString()))
				h.must(m.AddNewElement(i, j, &p.X))
				want[IndexPair{i, j}] = x
			}
			rows, cols := []uint32{0, 1, 2, 3}, []uint32{0, 1, 2, 3}
			for k := 0; k < 4; k++ {
				a := rng.Uint32N(4)
				b := (a + 1 + rng.Uint32N(3)) % 4
				h.must(m.SwapRows(a, b))
				rows[a], rows[b] = rows[b], rows[a]
				a = rng.Uint32N(4)
				b = (a + 1 + rng.Uint32N(3)) % 4
				h.must(m.SwapColumns(a, b))
				cols[a], cols[b] = cols[b], cols[a]
			}
			pivot := rng.Uint32N(4)
			target := (pivot + 1 + rng.Uint32N(3)) % 4
			alpha := big.NewRat(int64(rng.IntN(7)-3), int64(1+rng.IntN(7)))
			h.must(h.m.SetString(&p.Y, alpha.RatString()))
			delete(want, IndexPair{rows[target], cols[pivot]})
			for j := uint32(0); j < 4; j++ {
				if j == cols[pivot] {
					continue
				}
				v := want[IndexPair{rows[pivot], j}]
				if v == nil {
					continue
				}
				key := IndexPair{rows[target], j}
				old := want[key]
				x := new(big.Rat).Mul(alpha, v)
				if old != nil {
					x.Add(x, old)
				}
				if x.Sign() == 0 {
					if old != nil {
						cancels++
					}
					delete(want, key)
				} else {
					if old == nil {
						fills++
					}
					want[key] = x
				}
			}
			ok, e := m.PivotRowToRow(pivot, &p.Y, target, &Settings{DropTolerance: 1e200})
			h.must(e)
			count := 0
			for p := range want {
				if p.I == rows[target] {
					count++
				}
			}
			if ok != (count != 0) {
				t.Fatal("empty result", ok, count)
			}
			if !ok {
				empty++
			}
			lpExactSparseCheck(t, h, m, want)
			for _, offset := range m.WorkPivot {
				if offset != -1 {
					t.Fatal("workspace not cleared")
				}
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if fills == 0 || cancels == 0 || empty == 0 {
		t.Fatal("missing oracle paths", fills, cancels, empty)
	}
}

func TestLPExactSparseLiteralDivisionAndPivot(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("2", "6")
	m := lpNewExactSparse(t, h, 3)
	h.must(m.AddNewElement(1, 0, &p.X))
	h.must(m.AddNewElement(1, 1, &p.Y))
	h.must(m.SwapRows(0, 1))
	h.must(m.DivideRow(0, &m.Rows[1][0].Value, nil))
	lpExactSparseCheck(t, h, m, map[IndexPair]*big.Rat{{1, 0}: big.NewRat(1, 1), {1, 1}: big.NewRat(6, 1)})
	h.must(m.Close())
	m = lpNewExactSparse(t, h, 3)
	h.must(m.AddNewElement(1, 0, &p.X))
	h.must(m.AddNewElement(1, 1, &p.Y))
	h.must(m.DivideRow(1, &m.Columns[0].Values[0].Value, nil))
	lpExactSparseCheck(t, h, m, map[IndexPair]*big.Rat{{1, 0}: big.NewRat(1, 1), {1, 1}: big.NewRat(6, 1)})
	h.must(m.Close())
	m = lpNewExactSparse(t, h, 3)
	tiny := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 400))
	h.must(h.m.SetString(&p.X, tiny.RatString()))
	h.must(m.AddNewElement(1, 1, &p.X))
	h.must(h.m.SetInt64(&p.Y, 100))
	h.must(m.AddNewElement(0, 2, &p.Y))
	h.must(h.m.SetInt64(&p.X, 1))
	ok, e := m.PivotRowToRow(0, &p.X, 1, &Settings{DropTolerance: 1e200})
	h.must(e)
	if !ok || m.Rows[1][0].Index != 1 {
		t.Fatal("precise pivot reordered maximum")
	}
	lpExactSparseCheck(t, h, m, map[IndexPair]*big.Rat{{0, 2}: big.NewRat(100, 1), {1, 1}: tiny, {1, 2}: big.NewRat(100, 1)})
	h.must(m.Close())
	// Duplicate cancellations retain the first deletion and then fail at the
	// repeated source offset, without repairing or deduplicating the list.
	m = lpNewExactSparse(t, h, 3)
	for _, entry := range []struct {
		i, j uint32
		v    int64
	}{{0, 2, 1}, {0, 2, 0}, {1, 1, 7}, {1, 2, -1}} {
		h.must(h.m.SetInt64(&p.Y, entry.v))
		h.must(m.AddNewElement(entry.i, entry.j, &p.Y))
	}
	ok, e = m.PivotRowToRow(0, &p.X, 1, nil)
	if ok || e != ErrBounds || len(m.Rows[1]) != 1 || m.active != 3 {
		t.Fatal("duplicate deletion prefix", ok, e)
	}
	h.must(m.Close())
	// Release follows the source self-pivot body; debug asserts first.
	m = lpNewExactSparse(t, h, 3)
	h.must(h.m.SetInt64(&p.Y, 3))
	h.must(m.AddNewElement(0, 1, &p.Y))
	ok, e = m.PivotRowToRow(0, &p.X, 0, nil)
	if sourceDebugTests {
		if ok || e != ErrAssertion {
			t.Fatal(e)
		}
	} else {
		h.must(e)
		if !ok {
			t.Fatal("self continuation")
		}
		lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, big.NewRat(6, 1))
	}
	h.must(m.Close())
	// Debug rejects a zero divisor. Release follows actual rational arithmetic:
	// this nonzero numerator normalizes to 1/0, with no invented early error.
	m = lpNewExactSparse(t, h, 1)
	h.must(m.AddNewElement(0, 0, &p.X))
	h.must(h.m.SetInt64(&p.Y, 0))
	e = m.DivideRow(0, &p.Y, nil)
	if sourceDebugTests {
		if e != ErrAssertion {
			t.Fatal("zero divisor assertion", e)
		}
	} else {
		h.must(e)
		for _, v := range []*numeral.Rational{&m.Rows[0][0].Value, &m.Columns[0].Values[0].Value} {
			if v.Num.Val != 1 || v.Den.Val != 0 {
				t.Fatal("literal release zero division", v)
			}
		}
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseMutationAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	totalReceipts := 0
	for _, campaign := range []string{"add", "remove-col", "remove-row", "remove-both", "divide", "pivot-new", "pivot-existing", "pivot-cancel", "divide-direct", "remove-both-direct", "pivot-existing-direct"} {
		op := strings.TrimSuffix(campaign, "-direct")
		failures, successes, receipts, prefixes, syncPrefixes := 0, 0, 0, 0, 0
		for ordinal := int64(0); ordinal < 240; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", campaign, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if strings.HasSuffix(campaign, "-direct") {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				m := lpNewExactSparse(t, h, 3)
				xs, ys := strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13"
				p := h.pair(xs, ys)
				one := h.pair("1", "-1")
				x, _ := new(big.Rat).SetString(xs)
				y, _ := new(big.Rat).SetString(ys)
				want := make(map[IndexPair]*big.Rat)
				add := func(i, j uint32, v *numeral.Rational, w *big.Rat) {
					h.must(m.AddNewElement(i, j, v))
					want[IndexPair{i, j}] = new(big.Rat).Set(w)
				}
				switch op {
				case "remove-col", "remove-row", "remove-both":
					add(0, 0, &one.X, big.NewRat(1, 1))
					if op != "remove-col" {
						add(0, 1, &p.Y, y)
					}
					if op != "remove-row" {
						add(1, 0, &p.X, x)
					}
					delete(want, IndexPair{0, 0})
				case "divide":
					add(0, 0, &p.X, x)
					add(0, 1, &p.Y, y)
					divisor := y
					if campaign == "divide-direct" {
						h.must(h.m.SetInt64(&one.X, 2))
						divisor = big.NewRat(2, 1)
					}
					for _, v := range want {
						v.Quo(v, divisor)
					}
				case "pivot-new", "pivot-existing", "pivot-cancel":
					add(0, 1, &p.X, x)
					if op == "pivot-existing" {
						add(1, 1, &p.Y, y)
					}
					if op == "pivot-cancel" {
						add(1, 1, &p.X, x)
						add(1, 2, &p.Y, y)
						delete(want, IndexPair{1, 1})
					} else {
						v := new(big.Rat).Mul(x, y)
						if op == "pivot-existing" {
							v.Add(v, y)
						}
						want[IndexPair{1, 1}] = v
					}
				case "add":
					want[IndexPair{0, 1}] = x
				}
				h.budget.FailAfter(ordinal)
				var e error
				switch op {
				case "add":
					e = m.AddNewElement(0, 1, &p.X)
				case "remove-col", "remove-row", "remove-both":
					e = m.RemoveElement(0, 0)
				case "divide":
					divisor := &p.Y
					if campaign == "divide-direct" {
						divisor = &one.X
					}
					e = m.DivideRow(0, divisor, nil)
				default:
					a := &p.Y
					if op == "pivot-cancel" {
						a = &one.Y
					}
					_, e = m.PivotRowToRow(0, a, 1, nil)
				}
				h.budget.FailAfter(-1)
				if e == nil {
					successes++
					done = true
					lpExactSparseCheck(t, h, m, want)
				} else {
					if !lpExactPriorityPhysicalRefusal(e) {
						t.Fatal(e)
					}
					failures++
					if m.calls.pending() {
						receipts++
					}
					if op == "add" && len(m.Rows[0]) == 1 && len(m.Columns[1].Values) == 0 {
						prefixes++
					}
					if op == "remove-both" && m.Columns[0].Values[0].Index == 1 && m.Rows[0][0].Index == 0 {
						prefixes++
					}
					if (campaign == "divide-direct" || campaign == "pivot-existing-direct") &&
						m.calls.division == 0 && m.calls.numeric.raw == [3]uint64{} &&
						m.calls.integerDivision.Invocation == 0 && m.calls.gcd.Invocation == 0 {
						// Arithmetic has finished. Observe independent row and
						// column values at the later by-value copy refusal.
						for i := range m.Rows {
							for k := range m.Rows[i] {
								v := &m.Rows[i][k]
								c := &m.Columns[v.Index].Values[v.Other]
								rs, re := h.m.String(&v.Value)
								cs, ce := h.m.String(&c.Value)
								r, rok := new(big.Rat).SetString(rs)
								col, cok := new(big.Rat).SetString(cs)
								w := want[IndexPair{uint32(i), v.Index}]
								if re == nil && ce == nil && rok && cok && w != nil && r.Cmp(w) == 0 && r.Cmp(col) != 0 {
									syncPrefixes++
								}
							}
						}
					}
					for _, f := range []func() error{
						func() error { return m.AddNewElement(2, 2, &one.X) }, func() error { return m.RemoveElement(0, 0) },
						func() error { return m.DivideRow(0, &one.X, nil) }, func() error { _, e := m.PivotRowToRow(0, &one.X, 1, nil); return e },
						func() error { _, e := m.Get(0, 0); return e }, func() error { return m.SwapRows(0, 1) },
					} {
						if !errors.Is(f(), ErrExactVectorPending) {
							t.Fatal("stopped guard")
						}
					}
				}
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.RawInvocationState().Active != [3]bool{} || h.m.DivisionInvocationState().Active || h.m.ComparisonInvocationState().Active {
					t.Fatal("orphan numerical frame")
				}
			})
			if done {
				break
			}
		}
		totalReceipts += receipts
		if failures == 0 || successes != 1 || ((op == "add" || op == "remove-both") && prefixes == 0) {
			t.Fatal("incomplete failure campaign", campaign, failures, successes, receipts, prefixes)
		}
		if (campaign == "divide-direct" || campaign == "pivot-existing-direct") && syncPrefixes == 0 {
			t.Fatal("no completed row before failed column copy", campaign)
		}
	}
	if totalReceipts == 0 {
		t.Fatal("no actual numerical receipts across campaigns")
	}
}

func TestLPExactSparseMutationBoundsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"add row", "add column", "remove row", "remove offset", "current link", "column tail link", "both tails link", "row tail link", "divide row", "divide link", "scan", "scan pivot removal", "product index", "new column", "paired update", "stale work", "invalid pivot", "invalid target"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 3)
			p := h.pair("2", "1")
			add := func(i, j uint32) { h.must(m.AddNewElement(i, j, &p.X)) }
			var e error
			switch phase {
			case "add row":
				e = m.AddNewElement(3, 0, &p.X)
			case "add column":
				e = m.AddNewElement(0, 3, &p.X)
			case "remove row":
				e = m.RemoveElement(3, 0)
			case "remove offset":
				e = m.RemoveElement(0, 0)
			case "current link":
				add(0, 0)
				m.Rows[0][0].Other = 99
				e = m.RemoveElement(0, 0)
			case "column tail link", "both tails link", "row tail link":
				add(0, 0)
				add(0, 1)
				if phase != "row tail link" {
					add(1, 0)
				}
				if phase == "column tail link" {
					m.Columns[0].Values[1].Other = 99
				} else {
					m.Rows[0][1].Other = 99
				}
				e = m.RemoveElement(0, 0)
				expectedCount := uint32(3)
				if phase == "row tail link" {
					expectedCount = 2
				}
				if m.active != expectedCount || len(m.Rows[0]) != 2 {
					t.Fatal("tail pop before repair")
				}
			case "divide row":
				e = m.DivideRow(3, &p.Y, nil)
			case "divide link":
				add(0, 0)
				m.Rows[0][0].Other = 99
				e = m.DivideRow(0, &p.X, nil)
				lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, big.NewRat(1, 1))
				lpSparseRationalCheck(t, h, &m.Columns[0].Values[0].Value, big.NewRat(2, 1))
			default:
				add(0, 2)
				switch phase {
				case "scan":
					add(1, 2)
					m.WorkPivot = m.WorkPivot[:2]
				case "scan pivot removal":
					add(1, 0)
					m.Rows[1][0].Other = 99
				case "product index":
					m.Rows[0][0].Index = 3
				case "new column":
					m.Rows[0][0].Index = 3
					m.WorkPivot = append(m.WorkPivot, -1)
				case "paired update":
					add(1, 2)
					m.Rows[1][0].Other = 99
				case "stale work":
					m.WorkPivot[2] = -2
				}
				pivot, target := uint32(0), uint32(1)
				if phase == "invalid pivot" {
					pivot = 3
				}
				if phase == "invalid target" {
					target = 3
				}
				_, e = m.PivotRowToRow(pivot, &p.Y, target, nil)
				if phase == "paired update" {
					lpSparseRationalCheck(t, h, &m.Rows[1][0].Value, big.NewRat(4, 1))
					lpSparseRationalCheck(t, h, &m.Columns[2].Values[1].Value, big.NewRat(2, 1))
				}
			}
			want := error(ErrBounds)
			if sourceDebugTests && (phase == "invalid pivot" || phase == "invalid target") {
				want = ErrAssertion
			}
			if e != want {
				t.Fatal("reached guard", e, want)
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	// Exact cancellation of the only target term returns false immediately.
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 2)
	p := h.pair("3/7", "-1")
	h.must(m.AddNewElement(0, 1, &p.X))
	h.must(m.AddNewElement(1, 1, &p.X))
	ok, e := m.PivotRowToRow(0, &p.Y, 1, nil)
	h.must(e)
	if ok || len(m.Rows[1]) != 0 {
		t.Fatal("empty cancellation")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseMutationLeavesOlderComparisonOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 3)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			one := h.pair("1", "0")
			h.must(m.AddNewElement(0, 0, &one.X))
			h.must(m.AddNewElement(0, 1, &p.X))
			var caller exactPriorityCalls
			b, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, b, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				matched = true
				h.must(m.AddNewElement(2, 2, &p.Y))
				h.must(m.RemoveElement(0, 0))
				h.must(m.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no stopped comparison")
	}
}
