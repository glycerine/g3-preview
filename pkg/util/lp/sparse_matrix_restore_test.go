package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPSparseRowRestoreIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 7)
	f := newFloatSparseMatrix(7)
	want := map[IndexPair]*big.Rat{}
	fwant := map[IndexPair]float64{}
	cols := []uint32{0, 1, 2, 3, 4, 5, 6}
	rng := rand.New(rand.NewPCG(713, 991))
	for step := 0; step < 800; step++ {
		a := rng.Uint32N(7)
		b := (a + 1 + rng.Uint32N(6)) % 7
		h.must(m.SwapColumns(a, b))
		lpStaticMust(t, f.SwapColumns(a, b))
		cols[a], cols[b] = cols[b], cols[a]
		h.must(m.SwapRows(a, b))
		lpStaticMust(t, f.SwapRows(a, b))
		// The argument is physical, independent of the shuffled row map.
		row := rng.Uint32N(7)
		w := NewIndexedRationalVector(h.m, 7)
		fw := NewIndexedFloatVector(7)
		for _, j := range rng.Perm(7) {
			v := int64(rng.IntN(19) - 9)
			if step%19 == 0 {
				v = 0
			}
			h.must(h.m.SetInt64(&w.Data[j], v))
			fw.Data[j] = float64(v)
			w.Index = append(w.Index, uint32(j))
			fw.Index = append(fw.Index, uint32(j))
			key := IndexPair{row, cols[j]}
			if v == 0 {
				delete(want, key)
				delete(fwant, key)
			} else {
				want[key] = big.NewRat(v, 1)
				fwant[key] = float64(v)
			}
		}
		ok, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(row, w, nil)
		h.must(err)
		fok, err := f.SetRowFromWorkVectorAndCleanNotAdjusted(row, fw, &Settings{DropTolerance: 0.5})
		lpStaticMust(t, err)
		if ok != (len(m.Rows[row]) > 0) || fok != ok || len(w.Index) != 0 || len(fw.Index) != 0 {
			t.Fatal("result/index", step)
		}
		for j := range w.Data {
			lpSparseRationalCheck(t, h, &w.Data[j], new(big.Rat))
			if fw.Data[j] != 0 {
				t.Fatal("workspace")
			}
		}
		h.must(w.Close())
		lpExactSparseCheck(t, h, m, want)
		lpFloatSparseCheck(t, f, fwant)
		if ok {
			max := new(big.Rat).Abs(want[IndexPair{row, m.Rows[row][0].Index}])
			for _, iv := range m.Rows[row] {
				if new(big.Rat).Abs(want[IndexPair{row, iv.Index}]).Cmp(max) > 0 {
					t.Fatal("exact maximum")
				}
			}
			for _, iv := range f.Rows[row] {
				if math.Abs(iv.Value) > math.Abs(f.Rows[row][0].Value) {
					t.Fatal("float maximum")
				}
			}
		}
	}
}

func TestLPSparseRowRestoreLiteralWorkspace(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("5", "0")
	m := lpNewExactSparse(t, h, 4)
	f := newFloatSparseMatrix(4)
	for _, j := range []uint32{0, 0, 1} {
		h.must(m.AddNewElement(2, j, &p.X))
		lpStaticMust(t, f.AddNewElement(2, j, 5))
	}
	w := NewIndexedRationalVector(h.m, 4)
	defer func() { h.must(w.Close()) }()
	fw := NewIndexedFloatVector(4)
	for j, v := range []int64{3, 0, 7, 11} {
		h.must(h.m.SetInt64(&w.Data[j], v))
		fw.Data[j] = float64(v)
	}
	w.Index = []uint32{2, 2, 0}
	fw.Index = []uint32{2, 2, 0}
	_, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(2, w, nil)
	h.must(err)
	_, err = f.SetRowFromWorkVectorAndCleanNotAdjusted(2, fw, &Settings{DropTolerance: 0.5})
	lpStaticMust(t, err)
	// Last old duplicate consumes the hidden common value; missing unindexed
	// coordinate3 remains, and the repeated new index inserts only once.
	lpSparseRationalCheck(t, h, &w.Data[3], big.NewRat(11, 1))
	if fw.Data[3] != 11 || len(m.Rows[2]) != 2 {
		t.Fatal("literal input")
	}
	h.must(w.Close())
	lpExactSparseCheck(t, h, m, map[IndexPair]*big.Rat{{2, 0}: big.NewRat(3, 1), {2, 2}: big.NewRat(7, 1)})
	lpFloatSparseCheck(t, f, map[IndexPair]float64{{2, 0}: 3, {2, 2}: 7})
}

func TestLPFloatSparseRowRestoreToleranceAndIEEE(t *testing.T) {
	for _, existing := range []bool{false, true} {
		m := newFloatSparseMatrix(2)
		if existing {
			lpStaticMust(t, m.AddNewElement(0, 0, 4))
		}
		w := NewIndexedFloatVector(2)
		w.Data[0] = 0.1
		w.Index = []uint32{0}
		_, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, &Settings{DropTolerance: 0.5})
		if err != sourceAssertionFailure() {
			t.Fatal(err)
		}
		if err != nil {
			if len(m.Rows[0]) != 0 || len(w.Index) != 1 || w.Data[0] != 0.1 {
				t.Fatal("assertion prefix")
			}
		} else {
			lpFloatSparseCheck(t, m, map[IndexPair]float64{{0, 0}: 0.1})
		}
	}
	for _, tolerance := range []float64{0, math.NaN()} {
		m := newFloatSparseMatrix(2)
		lpStaticMust(t, m.AddNewElement(0, 0, 4))
		w := NewIndexedFloatVector(2)
		w.Data[1] = math.Inf(1)
		w.Index = []uint32{0, 1}
		ok, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, &Settings{DropTolerance: tolerance})
		lpStaticMust(t, err)
		if !ok || len(m.Rows[0]) != 2 || m.Rows[0][0].Index != 1 {
			t.Fatal("strict zero/NaN tolerance")
		}
		lpFloatSparseCheck(t, m, map[IndexPair]float64{{0, 0}: 0, {0, 1}: math.Inf(1)})
	}
}

func TestLPExactSparseRowRestoreAllocationCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"existing", "new", "remove"} {
			t.Run(fmt.Sprint(direct, "/", op), func(t *testing.T) {
				failures, successes, rowFirst, indexFirst := 0, 0, 0, 0
				for ordinal := int64(0); ordinal < 700; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						p := h.pair("1"+strings.Repeat("0", 95)+"1/7", "2"+strings.Repeat("0", 191)+"3/13")
						m := lpNewExactSparse(t, h, 2)
						w := NewIndexedRationalVector(h.m, 2)
						defer func() { h.must(w.Close()) }()
						h.must(m.AddNewElement(0, 0, &p.X))
						h.must(m.AddNewElement(1, 0, &p.Y))
						w.Index = []uint32{0, 1}
						want := map[IndexPair]*big.Rat{{1, 0}: mustLPRat(t, "2"+strings.Repeat("0", 191)+"3/13")}
						switch op {
						case "existing":
							h.must(h.m.Copy(&w.Data[0], &p.Y))
							want[IndexPair{0, 0}] = new(big.Rat).Set(want[IndexPair{1, 0}])
						case "new":
							h.must(h.m.Copy(&w.Data[1], &p.Y))
							want[IndexPair{0, 1}] = new(big.Rat).Set(want[IndexPair{1, 0}])
						case "remove":
						}
						h.budget.FailAfter(ordinal)
						_, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, nil)
						h.budget.FailAfter(-1)
						if err == nil {
							successes++
							done = true
							h.must(w.Close())
							lpExactSparseCheck(t, h, m, want)
						} else {
							failures++
							if len(w.Index) == 0 {
								indexFirst++
							}
							if op == "existing" && !m.calls.pending() {
								rs, re := h.m.String(&m.Rows[0][0].Value)
								cs, ce := h.m.String(&m.Columns[0].Values[0].Value)
								r, rok := new(big.Rat).SetString(rs)
								c, cok := new(big.Rat).SetString(cs)
								if re == nil && ce == nil && rok && cok && r.Cmp(want[IndexPair{0, 0}]) == 0 && c.Cmp(want[IndexPair{0, 0}]) != 0 {
									rowFirst++
								}
							}
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							if _, err = m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, nil); err != ErrExactVectorPending {
								t.Fatal("stopped", err)
							}
							if w.calls.pending() {
								t.Fatal("borrowed invocation adopted")
							}
						}
						h.must(m.Close())
						var cells []*numeral.Rational
						for j := range w.Data {
							cells = append(cells, &w.Data[j])
						}
						lpExactVectorNamedCensus(t, h, cells)
						h.must(w.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if direct && op == "existing" && rowFirst == 0 {
					t.Fatal("missing row-before-column copy refusal")
				}
				if direct && op == "new" && indexFirst == 0 {
					t.Fatal("missing cleared-index maximum refusal")
				}
				t.Logf("physical refusals=%d, row-before-column=%d, cleared-index=%d", failures, rowFirst, indexFirst)
				if successes != 1 || (direct && failures == 0 && op != "remove") {
					t.Fatal("campaign", failures, successes)
				}
			})
		}
	}
}

func TestLPSparseRowRestoreBoundsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"row", "old inverse", "old workspace", "column link", "remove link", "new index", "new mapping", "new column"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("4", "0")
			m := lpNewExactSparse(t, h, 2)
			f := newFloatSparseMatrix(2)
			h.must(m.AddNewElement(0, 0, &p.X))
			lpStaticMust(t, f.AddNewElement(0, 0, 4))
			w := NewIndexedRationalVector(h.m, 2)
			fw := NewIndexedFloatVector(2)
			defer func() { h.must(w.Close()) }()
			h.must(h.m.SetInt64(&w.Data[0], 3))
			fw.Data[0] = 3
			w.Index = []uint32{0}
			fw.Index = []uint32{0}
			row := uint32(0)
			switch phase {
			case "row":
				row = 2
			case "old inverse":
				m.ColumnPermutation.values = nil
				f.ColumnPermutation.values = nil
			case "old workspace":
				m.ColumnPermutation.values[0] = 2
				f.ColumnPermutation.values[0] = 2
			case "column link":
				m.Rows[0][0].Other = 2
				f.Rows[0][0].Other = 2
			case "remove link":
				m.Rows[0][0].Other = 2
				f.Rows[0][0].Other = 2
				h.must(h.m.SetInt64(&w.Data[0], 0))
				fw.Data[0] = 0
			case "new index":
				w.Index = append(w.Index, 2)
				fw.Index = append(fw.Index, 2)
			case "new mapping", "new column":
				h.must(h.m.SetInt64(&w.Data[1], 5))
				fw.Data[1] = 5
				w.Index = append(w.Index, 1)
				fw.Index = append(fw.Index, 1)
				if phase == "new mapping" {
					m.ColumnPermutation.reverse = m.ColumnPermutation.reverse[:1]
					f.ColumnPermutation.reverse = f.ColumnPermutation.reverse[:1]
				} else {
					m.Columns = m.Columns[:1]
					f.Columns = f.Columns[:1]
				}
			}
			_, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(row, w, nil)
			_, ferr := f.SetRowFromWorkVectorAndCleanNotAdjusted(row, fw, &Settings{DropTolerance: 0.5})
			if err != ErrBounds || ferr != ErrBounds {
				t.Fatal(phase, err, ferr)
			}
			if phase == "new index" || phase == "new mapping" || phase == "new column" {
				lpSparseRationalCheck(t, h, &w.Data[0], new(big.Rat))
				if fw.Data[0] != 0 || len(w.Index) != 2 || len(fw.Index) != 2 {
					t.Fatal("completed existing prefix")
				}
			}
			h.must(m.Close())
		})
	}
}

func TestLPExactSparseRowRestoreLeavesStoppedCallerOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			p := h.pair("1"+strings.Repeat("0", 100)+"/7", "2"+strings.Repeat("0", 100)+"/13")
			m := lpNewExactSparse(t, h, 1)
			w := NewIndexedRationalVector(h.m, 1)
			defer func() { h.must(w.Close()) }()
			h.must(w.SetValue(&p.X, 0))
			h.budget.FailAfter(ordinal)
			err := w.Add(0, &p.Y)
			h.budget.FailAfter(-1)
			if w.calls.pending() {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				before := h.m.RawInvocationState()
				if _, err = m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, nil); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				if m.work != nil || m.calls.pending() || len(m.Rows[0]) != 0 {
					t.Fatal("rejected caller mutated matrix")
				}
				h.must(m.Close())
				if h.m.RawInvocationState() != before {
					t.Fatal("matrix consumed caller receipt")
				}
			}
			h.must(m.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped caller")
	}
}

func TestLPExactSparseRowRestoreTinyAndForeign(t *testing.T) {
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 2)
	p := h.pair("1/"+"1"+strings.Repeat("0", 120), "0")
	w := NewIndexedRationalVector(h.m, 2)
	defer func() { h.must(w.Close()) }()
	h.must(m.AddNewElement(0, 0, &p.X))
	h.must(h.m.Copy(&w.Data[0], &p.X))
	h.must(h.m.Copy(&w.Data[1], &p.X))
	w.Index = []uint32{1}
	ok, err := m.SetRowFromWorkVectorAndCleanNotAdjusted(0, w, &Settings{DropTolerance: 1e200})
	h.must(err)
	if !ok || len(m.Rows[0]) != 2 {
		t.Fatal("exact tiny values dropped")
	}
	foreign := NewIndexedRationalVector(nil, 0)
	if _, err = m.SetRowFromWorkVectorAndCleanNotAdjusted(0, foreign, nil); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(m.Close())
	h.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
