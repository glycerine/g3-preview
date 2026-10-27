package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

// (I+N)^-1 = I-N+N^2-... for strictly upper triangular N. This independent
// dense polynomial oracle shares neither sparse traversal nor substitution.
func lpNilpotentSolve(a [][]*big.Rat, b []*big.Rat, left bool) []*big.Rat {
	n := len(b)
	off := make([][]*big.Rat, n)
	for i := range off {
		off[i] = make([]*big.Rat, n)
		for j := range off[i] {
			off[i][j] = new(big.Rat).Set(a[i][j])
			if i == j {
				off[i][j].SetInt64(0)
			}
		}
	}
	term, sum := make([]*big.Rat, n), make([]*big.Rat, n)
	for i := range b {
		term[i] = new(big.Rat).Set(b[i])
		sum[i] = new(big.Rat).Set(b[i])
	}
	for k := 1; k < n; k++ {
		term = lpExactMatrixVector(off, term, left)
		for i := range term {
			term[i].Neg(term[i])
			sum[i].Add(sum[i], term[i])
		}
	}
	return sum
}

func TestLPSparseIndexedSolveIndependentNilpotentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(12571, 1933))
	for world := 0; world < 240; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "0")
			n := uint32(rng.IntN(8))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			rows, cols := make([]uint32, n), make([]uint32, n)
			a := make([][]*big.Rat, n)
			edges := map[IndexPair]bool{}
			for i := range a {
				rows[i] = uint32(i)
				cols[i] = uint32(i)
				a[i] = make([]*big.Rat, n)
				for j := range a[i] {
					a[i][j] = new(big.Rat)
				}
			}
			if n > 1 {
				for k := 0; k < 6; k++ {
					i := rng.Uint32N(n)
					j := (i + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapRows(i, j))
					lpStaticMust(t, f.SwapRows(i, j))
					rows[i], rows[j] = rows[j], rows[i]
					i = rng.Uint32N(n)
					j = (i + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapColumns(i, j))
					lpStaticMust(t, f.SwapColumns(i, j))
					cols[i], cols[j] = cols[j], cols[i]
				}
			}
			for _, off := range rng.Perm(int(n * n)) {
				i, j := uint32(off)/n, uint32(off)%n
				if i > j || (i != j && rng.IntN(3) == 0) {
					continue
				}
				value := float64(rng.IntN(17)-8) / 4
				if i == j {
					value = 1
				} else if rng.IntN(7) == 0 {
					value = math.Ldexp(1, -80)
				}
				a[i][j] = new(big.Rat).SetFloat64(value)
				edges[IndexPair{i, j}] = true
				h.must(h.m.SetString(&p.X, a[i][j].RatString()))
				h.must(m.AddNewElement(rows[i], cols[j], &p.X))
				lpStaticMust(t, f.AddNewElement(rows[i], cols[j], value))
			}
			for _, left := range []bool{false, true} {
				b := make([]*big.Rat, n)
				for i := range b {
					b[i] = new(big.Rat)
					if rng.IntN(4) == 0 {
						b[i].SetFrac64(int64(rng.IntN(9)-4), 4)
					}
				}
				want := lpNilpotentSolve(a, b, left)
				v := lpExactVectorFromOracle(h, b)
				fv := NewIndexedFloatVector(n)
				fv.Index = append(fv.Index, v.Index...)
				for i := range b {
					fv.Data[i], _ = b[i].Float64()
				}
				reachable := map[uint32]bool{}
				for _, i := range v.Index {
					reachable[i] = true
				}
				for changed := true; changed; {
					changed = false
					for e := range edges {
						from, to := e.J, e.I
						if left {
							from, to = e.I, e.J
						}
						if reachable[from] && !reachable[to] {
							reachable[to] = true
							changed = true
						}
					}
				}
				var active, factive []uint32
				if left {
					h.must(m.SolveYUIndexed(v, &Settings{DropTolerance: 1e200}))
					lpStaticMust(t, f.SolveYUIndexed(fv, &Settings{DropTolerance: 1e-12}))
				} else {
					h.must(m.SolveUYIndexed(v, &Settings{DropTolerance: 1e200}, &active))
					lpStaticMust(t, f.SolveUYIndexed(fv, &Settings{DropTolerance: 1e-12}, &factive))
				}
				lpCheckExactSparse(t, h, v, want)
				for i, x := range want {
					expected, _ := x.Float64()
					if math.Abs(expected) < 1e-12 {
						expected = 0
					}
					if math.Abs(fv.Data[i]-expected) > 1e-11 {
						t.Fatal("floating nilpotent solve", left, i, fv.Data[i], expected)
					}
				}

				expectedIndex := map[uint32]bool{}
				for i, x := range want {
					v, _ := x.Float64()
					if math.Abs(v) >= 1e-12 {
						expectedIndex[uint32(i)] = true
					}
				}
				seen := map[uint32]bool{}
				for _, i := range fv.Index {
					if !expectedIndex[i] || seen[i] {
						t.Fatal("floating sparse membership", fv.Index, expectedIndex)
					}
					seen[i] = true
				}
				if len(seen) != len(expectedIndex) {
					t.Fatal("floating missing sparse index")
				}
				if !left {
					if len(active) != len(reachable) || !reflect.DeepEqual(active, factive) {
						t.Fatal("reachable graph domain", active, factive, reachable)
					}
					position := map[uint32]int{}
					for k, i := range active {
						if !reachable[i] {
							t.Fatal("unreachable output")
						}
						if _, ok := position[i]; ok {
							t.Fatal("duplicate graph output")
						}
						position[i] = k
					}
					for e := range edges {
						if e.I != e.J && reachable[e.I] && reachable[e.J] && position[e.I] >= position[e.J] {
							t.Fatal("non-topological postorder", e, active)
						}
					}
				}
				for _, done := range m.Processed {
					if done {
						t.Fatal("exact graph flag not cleared")
					}
				}
				for _, done := range f.Processed {
					if done {
						t.Fatal("floating graph flag not cleared")
					}
				}
				owned := lpSparseEtaOwningCells(m, nil)
				for i := range v.Data {
					owned = append(owned, &v.Data[i])
				}
				lpExactVectorNamedCensus(t, h, owned)
				h.must(v.Close())
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseIndexedGraphLiteralOrderAndFlags(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("1", "0")
	m := lpNewExactSparse(t, h, 6)
	f := newFloatSparseMatrix(6)
	for i := uint32(0); i < 6; i++ {
		h.must(m.AddNewElement(i, i, &p.X))
		lpStaticMust(t, f.AddNewElement(i, i, 1))
	}
	for _, e := range []IndexPair{{0, 2}, {0, 1}, {2, 4}, {1, 3}, {3, 4}} {
		h.must(m.AddNewElement(e.I, e.J, &p.Y))
		lpStaticMust(t, f.AddNewElement(e.I, e.J, 0))
	}
	var a, b []uint32
	h.must(m.extendActiveRows([]uint32{0, 0, 5}, &a))
	lpStaticMust(t, f.extendActiveRows([]uint32{0, 0, 5}, &b))
	if !reflect.DeepEqual(a, []uint32{4, 2, 3, 1, 0, 5}) || !reflect.DeepEqual(a, b) {
		t.Fatal("row postorder", a, b)
	}
	a = []uint32{5}
	b = []uint32{5}
	h.must(m.createGraph([]uint32{4, 4, 5}, &a))
	lpStaticMust(t, f.createGraph([]uint32{4, 4, 5}, &b))
	if !reflect.DeepEqual(a, []uint32{5, 0, 2, 1, 3, 4, 5}) || !reflect.DeepEqual(a, b) {
		t.Fatal("column postorder/prefix", a, b)
	}
	// Existing output entries are processed numerically and can appear twice in
	// the rebuilt sparse index; the source diagnostic is_OK returns true early.
	v := lpExactVectorFromOracle(h, []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat), big.NewRat(1, 1), big.NewRat(2, 1)})
	fv := NewIndexedFloatVector(6)
	fv.Data[4] = 1
	fv.Data[5] = 2
	fv.Index = []uint32{5, 4}
	a = []uint32{5}
	b = []uint32{5}
	h.must(m.SolveUYIndexed(v, nil, &a))
	lpStaticMust(t, f.SolveUYIndexed(fv, &Settings{DropTolerance: 1e-12}, &b))
	if !reflect.DeepEqual(v.Index, []uint32{5, 5, 4}) || !reflect.DeepEqual(v.Index, fv.Index) {
		t.Fatal("literal duplicate output index", v.Index, fv.Index)
	}
	h.must(v.Close())
	m.Processed[5] = true
	f.Processed[5] = true
	a = nil
	b = nil
	h.must(m.createGraph([]uint32{5}, &a))
	lpStaticMust(t, f.createGraph([]uint32{5}, &b))
	if len(a) != 0 || !m.Processed[5] || !f.Processed[5] {
		t.Fatal("unlisted processed flag was reset")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseIndexedSolveAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	receipts := 0
	for _, left := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(left, "/", direct), func(t *testing.T) {
				failures, successes := 0, 0
				for ordinal := int64(0); ordinal < 650; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						m := lpNewExactSparse(t, h, 2)
						xs, ys := strings.Repeat("1234567890", 50)+"/7", strings.Repeat("9876543210", 100)+"/13"
						p := h.pair(xs, ys)
						one := h.pair("1", "0")
						h.must(m.AddNewElement(0, 0, &one.X))
						h.must(m.AddNewElement(1, 1, &one.X))
						h.must(m.AddNewElement(0, 1, &p.X))
						values := []*big.Rat{big.NewRat(1, 1), mustLPRat(t, ys)}
						target := 0
						if left {
							values[0], values[1] = values[1], values[0]
							target = 1
						}
						v := lpExactVectorFromOracle(h, values)
						oldIndex := append([]uint32(nil), v.Index...)
						var active []uint32
						h.budget.FailAfter(ordinal)
						var err error
						if left {
							err = m.SolveYUIndexed(v, nil)
						} else {
							err = m.SolveUYIndexed(v, nil, &active)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							successes++
							done = true
							want := new(big.Rat).Sub(big.NewRat(1, 1), new(big.Rat).Mul(mustLPRat(t, xs), mustLPRat(t, ys)))
							lpSparseRationalCheck(t, h, &v.Data[target], want)
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.calls.pending() {
								receipts++
							}
							if v.calls.pending() {
								t.Fatal("caller adopted matrix invocation")
							}
							if !reflect.DeepEqual(v.Index, oldIndex) {
								t.Fatal("index rebuilt before arithmetic completed")
							}
							if e := m.SolveYUIndexed(v, nil); e != ErrExactVectorPending {
								t.Fatal("stopped left indexed solve", e)
							}
							if e := m.SolveUYIndexed(v, nil, &active); e != ErrExactVectorPending {
								t.Fatal("stopped right indexed solve", e)
							}
							for _, flag := range m.Processed {
								if flag {
									t.Fatal("numerical failure retained graph flag")
								}
							}
						}
						h.must(m.Close())
						var owned []*numeral.Rational
						for i := range v.Data {
							owned = append(owned, &v.Data[i])
						}
						lpExactVectorNamedCensus(t, h, owned)
						h.must(v.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if failures == 0 || successes != 1 {
					t.Fatal("incomplete indexed solve campaign", failures, successes)
				}
			})
		}
	}
	if receipts == 0 {
		t.Fatal("no actual indexed solve receipts")
	}
}

func TestLPSparseIndexedSolveToleranceAndHiddenInput(t *testing.T) {
	for _, left := range []bool{false, true} {
		f := newFloatSparseMatrix(3)
		lpStaticMust(t, f.AddNewElement(0, 1, 1))
		v := NewIndexedFloatVector(3)
		v.Data = []float64{math.Copysign(0, -1), 0.25, 17}
		v.Index = []uint32{0, 1}
		var active []uint32
		var err error
		if left {
			err = f.SolveYUIndexed(v, &Settings{DropTolerance: 0.25})
		} else {
			err = f.SolveUYIndexed(v, &Settings{DropTolerance: 0.25}, &active)
		}
		lpStaticMust(t, err)
		if v.Data[2] != 17 || v.Data[1] != 0.25 {
			t.Fatal("hidden input or strict threshold changed")
		}
		if !left && v.Data[0] != -0.25 {
			t.Fatal("right zero anchor/target")
		}
		if left && (v.Data[0] != 0 || math.Signbit(v.Data[0])) {
			t.Fatal("dropped zero not assigned positive zero")
		}
		// A zero tolerance does not drop zero, and visitation order is not a sort.
		v = NewIndexedFloatVector(3)
		v.Index = []uint32{0}
		active = nil
		if left {
			err = f.SolveYUIndexed(v, &Settings{})
		} else {
			err = f.SolveUYIndexed(v, &Settings{}, &active)
		}
		lpStaticMust(t, err)
		expected := []uint32{0}
		if left {
			expected = []uint32{1, 0}
		}
		if !reflect.DeepEqual(v.Index, expected) {
			t.Fatal("literal zero tolerance", left, v.Index, expected)
		}
	}
}

func TestLPSparseIndexedGraphBoundsAndPartialFlags(t *testing.T) {
	for _, left := range []bool{false, true} {
		for _, phase := range []string{"input", "node", "mapping", "storage", "inverse", "processed", "partial", "cleanup", "assertion"} {
			t.Run(fmt.Sprint(left, "/", phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair("1", "0")
				m := lpNewExactSparse(t, h, 3)
				f := newFloatSparseMatrix(3)
				for i := uint32(0); i < 3; i++ {
					h.must(m.AddNewElement(i, i, &p.X))
					lpStaticMust(t, f.AddNewElement(i, i, 1))
				}
				var edges []IndexPair
				if left {
					edges = []IndexPair{{0, 1}, {0, 2}}
				} else {
					edges = []IndexPair{{0, 2}, {1, 2}}
				}
				for _, e := range edges {
					h.must(m.AddNewElement(e.I, e.J, &p.X))
					lpStaticMust(t, f.AddNewElement(e.I, e.J, 1))
				}
				seed := uint32(2)
				if left {
					seed = 0
				}
				index := []uint32{seed}
				var a, b []uint32
				savedRows, savedCols := m.Rows, m.Columns
				switch phase {
				case "input", "node":
					seed = 3
					index = []uint32{3}
				case "mapping":
					if left {
						m.RowPermutation.values = nil
						f.RowPermutation.values = nil
					} else {
						m.ColumnPermutation.reverse = nil
						f.ColumnPermutation.reverse = nil
					}
				case "storage":
					if left {
						m.Rows = nil
						f.Rows = nil
					} else {
						m.Columns = nil
						f.Columns = nil
					}
				case "inverse":
					if left {
						m.Rows[0][1].Index = 3
						f.Rows[0][1].Index = 3
					} else {
						m.Columns[2].Values[1].Index = 3
						f.Columns[2].Values[1].Index = 3
					}
				case "processed":
					if left {
						m.ColumnPermutation.values[1] = 3
						f.ColumnPermutation.values[1] = 3
					} else {
						m.RowPermutation.reverse[0] = 3
						f.RowPermutation.reverse[0] = 3
					}
				case "partial":
					if left {
						m.Rows[0][2].Index = 3
						f.Rows[0][2].Index = 3
					} else {
						m.Columns[2].Values[2].Index = 3
						f.Columns[2].Values[2].Index = 3
					}
				case "cleanup":
					a = []uint32{3}
					b = []uint32{3}
				case "assertion":
					m.Processed[seed] = true
					f.Processed[seed] = true
				}
				var err, ferr error
				if phase == "node" || phase == "assertion" {
					if left {
						err = m.processYIndex(seed, &a)
						ferr = f.processYIndex(seed, &b)
					} else {
						err = m.processColumn(seed, &a)
						ferr = f.processColumn(seed, &b)
					}
				} else if left {
					err = m.extendActiveRows(index, &a)
					ferr = f.extendActiveRows(index, &b)
				} else {
					err = m.createGraph(index, &a)
					ferr = f.createGraph(index, &b)
				}
				if phase == "assertion" {
					if sourceDebugTests {
						if err != ErrAssertion || ferr != ErrAssertion {
							t.Fatal("graph assertion", err, ferr)
						}
					} else {
						h.must(err)
						lpStaticMust(t, ferr)
					}
				} else if err != ErrBounds || ferr != ErrBounds {
					t.Fatal("graph bounds", phase, err, ferr)
				}
				if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(m.Processed, f.Processed) {
					t.Fatal("graph publication profiles", a, b, m.Processed, f.Processed)
				}
				if phase == "partial" {
					want := []uint32{0}
					flags := []bool{true, false, false}
					if left {
						want = []uint32{1}
						flags = []bool{true, true, false}
					}
					if !reflect.DeepEqual(a, want) || !reflect.DeepEqual(m.Processed, flags) {
						t.Fatal("source partial flags", a, m.Processed, want, flags)
					}
				}
				m.Rows = savedRows
				m.Columns = savedCols
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPSparseIndexedSolveBoundsAndPublication(t *testing.T) {
	for _, left := range []bool{false, true} {
		for _, phase := range []string{"input", "column mapping", "column storage", "row inverse", "target", "filter"} {
			t.Run(fmt.Sprint(left, "/", phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair("1", "0")
				m := lpNewExactSparse(t, h, 2)
				f := newFloatSparseMatrix(2)
				for i := uint32(0); i < 2; i++ {
					h.must(m.AddNewElement(i, i, &p.X))
					lpStaticMust(t, f.AddNewElement(i, i, 1))
				}
				v := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
				fv := NewIndexedFloatVector(2)
				fv.Data = []float64{1, 1}
				var a, b []uint32
				savedCols := m.Columns
				if !left {
					v.Index = nil
					fv.Index = nil
					a = []uint32{0}
					b = []uint32{0}
				} else {
					v.Index = []uint32{0}
					fv.Index = []uint32{0}
				}
				switch phase {
				case "input":
					h.must(v.Resize(0))
					fv.Data = nil
					if left {
						v.Index = []uint32{0}
						fv.Index = []uint32{0}
					}
				case "column mapping":
					m.ColumnPermutation.reverse = nil
					f.ColumnPermutation.reverse = nil
				case "column storage":
					m.Columns = nil
					f.Columns = nil
				case "row inverse":
					m.RowPermutation.reverse = nil
					f.RowPermutation.reverse = nil
				case "target":
					m.RowPermutation.reverse[0] = 2
					f.RowPermutation.reverse[0] = 2
				case "filter":
					// A preexisting graph output can name a coordinate outside the
					// borrowed vector and fail before sparse-index rebuilding.
					h.must(v.Resize(1))
					fv.Data = fv.Data[:1]
					h.must(h.m.SetInt64(&v.Data[0], 0))
					fv.Data[0] = 0
					if left { // Left solve always reads every visited j; this is an input error.
						v.Index = []uint32{1}
						fv.Index = []uint32{1}
					} else {
						a = []uint32{1, 0}
						b = []uint32{1, 0}
					}
				}
				var err, ferr error
				if left {
					err = m.SolveYUIndexed(v, nil)
					ferr = f.SolveYUIndexed(fv, &Settings{DropTolerance: 1e-12})
				} else {
					err = m.SolveUYIndexed(v, nil, &a)
					ferr = f.SolveUYIndexed(fv, &Settings{DropTolerance: 1e-12}, &b)
				}
				if err != ErrBounds || ferr != ErrBounds {
					t.Fatal("indexed solve bounds", phase, left, err, ferr)
				}
				m.Columns = savedCols
				h.must(m.Close())
				h.must(v.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactSparseIndexedSolveBorrowedOwnerGuards(t *testing.T) {
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 1)
	v := NewIndexedRationalVector(other.m, 1)
	var a []uint32
	if err := m.SolveYUIndexed(v, nil); err != ErrExactVectorOwner {
		t.Fatal("foreign left input", err)
	}
	if err := m.SolveUYIndexed(v, nil, &a); err != ErrExactVectorOwner {
		t.Fatal("foreign right input", err)
	}
	h.must(m.Close())
	other.must(v.Close())
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 1)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			v := NewIndexedRationalVector(h.m, 1)
			h.must(v.SetValue(&p.X, 0))
			h.budget.FailAfter(ordinal)
			err := v.Add(0, &p.Y)
			h.budget.FailAfter(-1)
			if v.calls.pending() {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				state := h.m.RawInvocationState()
				a := []uint32{77}
				if err = m.SolveYUIndexed(v, nil); err != ErrExactVectorPending {
					t.Fatal("stopped left input", err)
				}
				if err = m.SolveUYIndexed(v, nil, &a); err != ErrExactVectorPending {
					t.Fatal("stopped right input", err)
				}
				if m.work != nil || m.calls.pending() || !reflect.DeepEqual(a, []uint32{77}) || m.Processed[0] {
					t.Fatal("borrowed input changed caller or matrix")
				}
				h.must(m.Close())
				if h.m.RawInvocationState() != state {
					t.Fatal("consumed caller receipt")
				}
			}
			h.must(m.Close())
			h.must(v.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual stopped input")
	}
}
