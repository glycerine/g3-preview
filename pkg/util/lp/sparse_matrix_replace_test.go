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

func TestLPSparseColumnReplacementIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	value := h.pair("0", "0")
	n := uint32(7)
	m := lpNewExactSparse(t, h, n)
	f := newFloatSparseMatrix(n)
	rng := rand.New(rand.NewPCG(8191, 331))
	exact := map[IndexPair]*big.Rat{}
	floating := map[IndexPair]float64{}
	rows, cols := make([]uint32, n), make([]uint32, n)
	for i := range rows {
		rows[i] = uint32(i)
		cols[i] = uint32(i)
		m.Columns[i].ShortenedMarkovitz = 77
		f.Columns[i].ShortenedMarkovitz = 77
	}
	for _, off := range rng.Perm(int(n * n)) {
		if rng.IntN(3) == 0 {
			continue
		}
		i, j := uint32(off)/n, uint32(off)%n
		v := float64(rng.IntN(17)-8) / 4
		x := new(big.Rat).SetFloat64(v)
		h.must(h.m.SetString(&value.X, x.RatString()))
		h.must(m.AddNewElement(i, j, &value.X))
		lpStaticMust(t, f.AddNewElement(i, j, v))
		exact[IndexPair{i, j}] = x
		floating[IndexPair{i, j}] = v
	}
	for i := uint32(0); i < n; i++ {
		h.must(m.SetMaxInRow(i))
		lpStaticMust(t, f.SetMaxInRow(i))
	}
	// Replacement does not update the active-pivot queue or shortened counters.
	h.must(m.pivotQueue.Enqueue(1, 2, 991))
	lpStaticMust(t, f.pivotQueue.Enqueue(1, 2, 991))
	for step := 0; step < 1600; step++ {
		if step%5 == 0 {
			a := rng.Uint32N(n)
			b := (a + 1 + rng.Uint32N(n-1)) % n
			h.must(m.SwapRows(a, b))
			lpStaticMust(t, f.SwapRows(a, b))
			rows[a], rows[b] = rows[b], rows[a]
			a = rng.Uint32N(n)
			b = (a + 1 + rng.Uint32N(n-1)) % n
			h.must(m.SwapColumns(a, b))
			lpStaticMust(t, f.SwapColumns(a, b))
			cols[a], cols[b] = cols[b], cols[a]
		}
		column := rng.Uint32N(n)
		physical := cols[column]
		w := NewIndexedRationalVector(h.m, n)
		fw := NewIndexedFloatVector(n)
		for _, i := range rng.Perm(int(n)) {
			v := float64(rng.IntN(17)-8) / 4
			if rng.IntN(5) == 0 {
				v = math.Ldexp(1, -40)
			}
			x := new(big.Rat).SetFloat64(v)
			h.must(h.m.SetString(&value.X, x.RatString()))
			h.must(h.m.Copy(&w.Data[i], &value.X))
			fw.Data[i] = v
			w.Index = append(w.Index, uint32(i))
			fw.Index = append(fw.Index, uint32(i))
			pair := IndexPair{rows[i], physical}
			_, had := floating[pair]
			if v == 0 || (!had && math.Abs(v) < 0.5) {
				delete(floating, pair)
			} else {
				floating[pair] = v
			}
			if v == 0 {
				delete(exact, pair)
			} else {
				exact[pair] = x
			}
		}
		h.must(m.ReplaceColumn(column, w, &Settings{DropTolerance: 1e200}))
		lpStaticMust(t, f.ReplaceColumn(column, fw, &Settings{DropTolerance: 0.5}))
		if len(w.Index) != 0 || len(fw.Index) != 0 {
			t.Fatal("input index not cleared")
		}
		for i := range w.Data {
			lpSparseRationalCheck(t, h, &w.Data[i], new(big.Rat))
			if fw.Data[i] != 0 {
				t.Fatal("input data not cleared")
			}
		}
		h.must(w.Close())
		lpExactSparseCheck(t, h, m, exact)
		lpFloatSparseCheck(t, f, floating)
		for i := uint32(0); i < n; i++ {
			if len(m.Rows[i]) > 0 {
				max := new(big.Rat).Abs(exact[IndexPair{i, m.Rows[i][0].Index}])
				for _, v := range m.Rows[i] {
					if new(big.Rat).Abs(exact[IndexPair{i, v.Index}]).Cmp(max) > 0 {
						t.Fatal("exact maximum", step, i)
					}
				}
			}
			if len(f.Rows[i]) > 0 {
				max := math.Abs(f.Rows[i][0].Value)
				for _, v := range f.Rows[i] {
					if math.Abs(v.Value) > max {
						t.Fatal("floating maximum", step, i)
					}
				}
			}
			if m.Columns[i].ShortenedMarkovitz != 77 || f.Columns[i].ShortenedMarkovitz != 77 {
				t.Fatal("replacement changed shortened counters")
			}
		}
		lpCheckActiveQueue(t, m.pivotQueue, map[IndexPair]uint32{{1, 2}: 991}, n)
		lpCheckActiveQueue(t, f.pivotQueue, map[IndexPair]uint32{{1, 2}: 991}, n)
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPSparseColumnReplacementLiteralInput(t *testing.T) {
	// The old column is traversed backward: its last duplicate consumes w first.
	// A common unindexed value is consumed; an absent unindexed value is retained.
	h := newLPPairHarness(t)
	v := h.pair("0", "0")
	m := lpNewExactSparse(t, h, 4)
	f := newFloatSparseMatrix(4)
	for _, e := range []struct {
		i, j uint32
		v    int64
	}{{0, 0, 8}, {0, 1, 2}, {0, 0, 9}, {1, 0, 7}, {1, 1, 6}, {2, 1, 10}} {
		h.must(h.m.SetInt64(&v.X, e.v))
		h.must(m.AddNewElement(e.i, e.j, &v.X))
		lpStaticMust(t, f.AddNewElement(e.i, e.j, float64(e.v)))
	}
	for i := uint32(0); i < 4; i++ {
		h.must(m.SetMaxInRow(i))
		lpStaticMust(t, f.SetMaxInRow(i))
	}
	w := NewIndexedRationalVector(h.m, 4)
	fw := NewIndexedFloatVector(4)
	for i, x := range []int64{3, 1, 15, 17} {
		h.must(h.m.SetInt64(&w.Data[i], x))
		fw.Data[i] = float64(x)
	}
	w.Index = []uint32{2, 2, 0}
	fw.Index = []uint32{2, 2, 0}
	h.must(m.ReplaceColumn(0, w, nil))
	lpStaticMust(t, f.ReplaceColumn(0, fw, &Settings{}))
	want := map[IndexPair]*big.Rat{{0, 0}: big.NewRat(3, 1), {0, 1}: big.NewRat(2, 1), {1, 0}: big.NewRat(1, 1), {1, 1}: big.NewRat(6, 1), {2, 0}: big.NewRat(15, 1), {2, 1}: big.NewRat(10, 1)}
	fwant := map[IndexPair]float64{}
	for p, x := range want {
		fwant[p], _ = x.Float64()
	}
	lpSparseRationalCheck(t, h, &w.Data[3], big.NewRat(17, 1))
	if fw.Data[3] != 17 {
		t.Fatal("unindexed new value consumed")
	}
	h.must(w.Close())
	lpExactSparseCheck(t, h, m, want)
	lpFloatSparseCheck(t, f, fwant)
	if m.Rows[1][0].Index != 1 || f.Rows[1][0].Index != 1 || m.Rows[2][0].Index != 0 || f.Rows[2][0].Index != 0 {
		t.Fatal("head decrease/new maximum")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseColumnReplacementAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	totalReceipts := 0
	for _, op := range []string{"existing-head", "existing-head-grow", "existing-other", "remove-head", "new"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes, prefixes := 0, 0, 0
				for ordinal := int64(0); ordinal < 600; ordinal++ {
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
						x, y := mustLPRat(t, xs), mustLPRat(t, ys)
						w := NewIndexedRationalVector(h.m, 2)
						w.Index = []uint32{0, 1}
						want := map[IndexPair]*big.Rat{}
						add := func(i, j uint32, value *numeral.Rational, rat *big.Rat) {
							h.must(m.AddNewElement(i, j, value))
							want[IndexPair{i, j}] = new(big.Rat).Set(rat)
						}
						switch op {
						case "existing-head":
							add(0, 0, &p.Y, y)
							add(0, 1, &one.X, big.NewRat(1, 1))
							h.must(h.m.Copy(&w.Data[0], &p.X))
							want[IndexPair{0, 0}] = x
						case "existing-head-grow":
							add(0, 0, &p.X, x)
							add(0, 1, &one.X, big.NewRat(1, 1))
							h.must(h.m.Copy(&w.Data[0], &p.Y))
							want[IndexPair{0, 0}] = y
						case "existing-other":
							add(0, 1, &p.X, x)
							add(0, 0, &one.X, big.NewRat(1, 1))
							h.must(h.m.Copy(&w.Data[0], &p.Y))
							want[IndexPair{0, 0}] = y
						case "remove-head":
							add(0, 0, &p.Y, y)
							add(0, 1, &p.X, x)
							add(1, 0, &one.X, big.NewRat(1, 1))
							h.must(h.m.Copy(&w.Data[1], &one.X))
							delete(want, IndexPair{0, 0})
						case "new":
							add(0, 1, &p.X, x)
							h.must(h.m.Copy(&w.Data[0], &p.Y))
							want[IndexPair{0, 0}] = y
						}
						h.budget.FailAfter(ordinal)
						err := m.ReplaceColumn(0, w, nil)
						h.budget.FailAfter(-1)
						if err == nil {
							successes++
							done = true
							if len(w.Index) != 0 {
								t.Fatal("successful input index")
							}
							h.must(w.Close())
							lpExactSparseCheck(t, h, m, want)
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.calls.pending() {
								totalReceipts++
							}
							if err = m.ReplaceColumn(0, w, nil); err != ErrExactVectorPending {
								t.Fatal("stopped replacement", err)
							}
							if _, _, err = m.FillEtaMatrix(0); err != ErrExactVectorPending {
								t.Fatal("stopped construction", err)
							}
							if w.calls.pending() {
								t.Fatal("borrowed input adopted matrix invocation")
							}
							if direct {
								switch op {
								case "new":
									if len(m.Rows[0]) == 2 && len(m.Columns[0].Values) == 0 {
										prefixes++
									}
								case "remove-head":
									if len(m.Rows[0]) == 1 && m.active == 2 {
										prefixes++
									}
								case "existing-other":
									if len(m.Columns[0].Values) > 0 && m.Columns[0].Values[0].Other == 0 && m.Rows[0][0].Index == 1 {
										prefixes++
									}
								case "existing-head", "existing-head-grow":
									if m.calls.comparison == 0 && m.calls.division == 0 && m.calls.numeric.raw == [3]uint64{} && m.calls.integerDivision.Invocation == 0 && m.calls.gcd.Invocation == 0 {
										rs, re := h.m.String(&m.Rows[0][0].Value)
										cs, ce := h.m.String(&m.Columns[0].Values[0].Value)
										r, rok := new(big.Rat).SetString(rs)
										c, cok := new(big.Rat).SetString(cs)
										old, next := y, x
										if op == "existing-head-grow" {
											old, next = x, y
										}
										if re == nil && ce == nil && rok && cok && r.Cmp(old) == 0 && c.Cmp(next) == 0 {
											prefixes++
										}
									}
								}
							}
						}
						h.must(m.Close())
						var owned []*numeral.Rational
						for i := range w.Data {
							owned = append(owned, &w.Data[i])
						}
						lpExactVectorNamedCensus(t, h, owned)
						h.must(w.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if failures == 0 || successes != 1 || (direct && prefixes == 0) {
					t.Fatal("incomplete replacement campaign", failures, successes, prefixes)
				}
			})
		}
	}
	if totalReceipts == 0 {
		t.Fatal("no actual replacement numerical receipts")
	}
}

func TestLPSparseColumnReplacementIEEEAndTolerance(t *testing.T) {
	m := newFloatSparseMatrix(4)
	lpStaticMust(t, m.AddNewElement(0, 0, 5))
	lpStaticMust(t, m.AddNewElement(0, 1, 4))
	lpStaticMust(t, m.AddNewElement(1, 1, 3))
	w := NewIndexedFloatVector(4)
	w.Data = []float64{0.1, 0.1, 0.5, math.Copysign(0, -1)}
	w.Index = []uint32{0, 1, 2, 3}
	lpStaticMust(t, m.ReplaceColumn(0, w, &Settings{DropTolerance: 0.5}))
	lpFloatSparseCheck(t, m, map[IndexPair]float64{{0, 0}: 0.1, {0, 1}: 4, {1, 1}: 3, {2, 0}: 0.5})
	if !math.Signbit(w.Data[3]) || w.Data[0] != 0 || w.Data[1] != 0 || len(w.Index) != 0 {
		t.Fatal("literal zero and tolerance clearing")
	}
	// NaN comparisons do not move a term to the head; infinity can become head.
	w.Data[0] = math.NaN()
	w.Data[1] = math.Inf(1)
	w.Index = []uint32{0, 1}
	lpStaticMust(t, m.ReplaceColumn(0, w, &Settings{DropTolerance: 0.5}))
	if m.Rows[0][0].Index != 1 || !math.IsNaN(m.Columns[0].Values[0].Value) || m.Rows[1][0].Index != 0 || !math.IsInf(m.Rows[1][0].Value, 1) {
		t.Fatal("IEEE maximum update")
	}
}

func TestLPSparseColumnReplacementBoundsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"logical column", "physical column", "row inverse", "input extent", "row extent", "row offset", "remove link", "remove maximum", "common maximum", "common swap", "new index", "new inverse", "new row", "new column", "new swap"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("5", "2")
			m := lpNewExactSparse(t, h, 3)
			f := newFloatSparseMatrix(3)
			for _, e := range []IndexPair{{0, 0}, {0, 1}} {
				h.must(m.AddNewElement(e.I, e.J, &p.X))
				lpStaticMust(t, f.AddNewElement(e.I, e.J, 5))
			}
			w := NewIndexedRationalVector(h.m, 3)
			fw := NewIndexedFloatVector(3)
			savedRows, savedCols := m.Rows, m.Columns
			column := uint32(0)
			var err, ferr error
			switch phase {
			case "logical column":
				column = 3
			case "physical column":
				m.Columns = nil
				f.Columns = nil
			case "row inverse":
				m.Columns[0].Values[0].Index = 3
				f.Columns[0].Values[0].Index = 3
			case "input extent":
				h.must(w.Resize(0))
				fw.Data = nil
			case "row extent":
				m.Rows = nil
				f.Rows = nil
			case "row offset":
				m.Columns[0].Values[0].Other = 3
				f.Columns[0].Values[0].Other = 3
			case "remove link", "remove maximum":
				if phase == "remove link" {
					m.Rows[0][1].Other = 3
					f.Rows[0][1].Other = 3
				} else {
					h.must(m.AddNewElement(0, 2, &p.Y))
					lpStaticMust(t, f.AddNewElement(0, 2, 2))
					h.must(h.m.SetInt64(&m.Rows[0][0].Value, 10))
					h.must(h.m.SetInt64(&m.Columns[0].Values[0].Value, 10))
					f.Rows[0][0].Value = 10
					f.Columns[0].Values[0].Value = 10
					h.must(h.m.SetInt64(&m.Rows[0][1].Value, 9))
					h.must(h.m.SetInt64(&m.Columns[1].Values[0].Value, 9))
					f.Rows[0][1].Value = 9
					f.Columns[1].Values[0].Value = 9
					m.Rows[0][1].Other = 3
					f.Rows[0][1].Other = 3
				}
			case "common maximum":
				h.must(h.m.Copy(&w.Data[0], &p.Y))
				fw.Data[0] = 2
				m.Rows[0][1].Other = 3
				f.Rows[0][1].Other = 3
				// Make the other term larger so head decrease actually selects it.
				h.must(h.m.SetInt64(&m.Rows[0][1].Value, 9))
				f.Rows[0][1].Value = 9
			case "common swap":
				// Replace the nonhead column0 term with a new maximum.
				h.must(m.putMaxForReplacementTest(0, 1))
				lpStaticMust(t, f.putMaxFirst(0, 1))
				h.must(h.m.SetInt64(&w.Data[0], 9))
				fw.Data[0] = 9
				m.Rows[0][1].Other = 3
				f.Rows[0][1].Other = 3
			case "new index", "new inverse", "new row", "new column", "new swap":
				index := uint32(2)
				if phase == "new index" || phase == "new inverse" {
					index = 3
				}
				if phase == "new inverse" {
					h.must(w.Resize(4))
					fw.Data = append(fw.Data, 0)
				}
				if index < uint32(len(w.Data)) {
					h.must(h.m.SetInt64(&w.Data[index], 9))
					fw.Data[index] = 9
				}
				w.Index = []uint32{index}
				fw.Index = []uint32{index}
				if phase == "new row" {
					m.Rows = m.Rows[:2]
					f.Rows = f.Rows[:2]
				}
				if phase == "new column" {
					m.Columns = m.Columns[:1]
					f.Columns = f.Columns[:1]
					column = 1
				}
				if phase == "new swap" {
					h.must(m.AddNewElement(2, 1, &p.Y))
					lpStaticMust(t, f.AddNewElement(2, 1, 2))
					m.Rows[2][0].Other = 3
					f.Rows[2][0].Other = 3
				}
			}
			if phase == "new column" { // Reach insertion's column guard after an empty old-column pass.
				b, e := m.beginWork(8)
				h.must(e)
				err = m.addNewElementsOfWAndClear(1, w)
				m.finishWork(b, &err)
				ferr = f.addNewElementsOfWAndClear(1, fw, &Settings{})
			} else {
				err = m.ReplaceColumn(column, w, nil)
				ferr = f.ReplaceColumn(column, fw, &Settings{})
			}
			if err != ErrBounds || ferr != ErrBounds {
				t.Fatal("replacement bounds", phase, err, ferr)
			}
			if phase == "common maximum" || phase == "common swap" {
				if w.Data[0].Num.Val == 0 || fw.Data[0] == 0 {
					t.Fatal("input cleared before maximum failure")
				}
			}
			if phase == "new swap" {
				if len(m.Columns[0].Values) != 1 || len(f.Columns[0].Values) != 1 {
					t.Fatal("new term missing before swap failure")
				}
			}
			m.Rows = savedRows
			m.Columns = savedCols
			h.must(m.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

// Use the existing owning maximum helper without introducing a second public
// operation or allowing source values to be shallow-swapped in test fixtures.
func (m *RationalSparseMatrix) putMaxForReplacementTest(row, index uint32) (err error) {
	b, err := m.beginWork(4)
	if err != nil {
		return err
	}
	defer m.finishWork(b, &err)
	return m.putMaxFirst(row, index)
}

func TestLPExactSparseColumnReplacementBorrowedOwnerGuards(t *testing.T) {
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 1)
	foreign := NewIndexedRationalVector(other.m, 1)
	if err := m.ReplaceColumn(0, foreign, nil); err != ErrExactVectorOwner {
		t.Fatal("foreign input", err)
	}
	if m.work != nil || m.calls.pending() {
		t.Fatal("foreign input started mutation")
	}
	h.must(foreign.Close())
	h.must(m.Close())
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 1)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			w := NewIndexedRationalVector(h.m, 1)
			h.must(w.SetValue(&p.X, 0))
			h.budget.FailAfter(ordinal)
			err := w.Add(0, &p.Y)
			h.budget.FailAfter(-1)
			if w.calls.pending() {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				before := h.m.RawInvocationState()
				if err = m.ReplaceColumn(0, w, nil); err != ErrExactVectorPending {
					t.Fatal("borrowed stopped input", err)
				}
				if m.work != nil || m.calls.pending() || m.active != 0 {
					t.Fatal("borrowed input changed matrix")
				}
				h.must(m.Close())
				if h.m.RawInvocationState() != before {
					t.Fatal("matrix consumed input receipt")
				}
			}
			h.must(m.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped input vector")
	}
}
