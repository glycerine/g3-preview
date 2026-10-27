package lp

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestLPExactStaticScopesIndependentDimensionOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	p := h.pair("2", "3")
	m := NewRationalStaticMatrix(h.m, 2, 2)
	t.Cleanup(func() { h.must(m.Close()) })
	want := map[IndexPair]*big.Rat{{0, 0}: big.NewRat(2, 1), {1, 1}: big.NewRat(3, 1)}
	h.must(m.Set(0, 0, &p.X))
	h.must(m.Set(1, 1, &p.Y))
	type dimensions struct{ rows, columns uint32 }
	var stack []dimensions
	rows, columns := uint32(2), uint32(2)
	r := rand.New(rand.NewPCG(791, 167))
	for step := 0; step < 1400; step++ {
		switch r.IntN(4) {
		case 0:
			if len(stack) >= 8 {
				continue
			}
			h.must(m.Push())
			stack = append(stack, dimensions{rows, columns})
			delta := uint32(r.IntN(3))
			h.must(m.AddColumns(delta))
			columns += delta
			for n := r.IntN(3) + 1; n > 0; n-- {
				h.must(m.AddRow())
				for j := uint32(0); j < columns; j++ {
					if r.IntN(4) != 0 {
						continue
					}
					x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(r.IntN(15)-7)), uint(r.IntN(90))), big.NewInt(int64(r.IntN(19)+1)))
					h.must(h.m.SetString(&p.X, x.RatString()))
					h.must(m.AddNewElement(rows, j, &p.X))
					want[IndexPair{rows, j}] = x
				}
				rows++
			}
		case 1:
			count := uint32(r.IntN(4))
			h.must(m.Pop(count))
			for ; count > 0 && len(stack) > 0; count-- {
				d := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				rows, columns = d.rows, d.columns
				for k := range want {
					if k.I >= rows {
						delete(want, k)
					}
				}
			}
		case 2:
			i, j := uint32(r.IntN(2)), uint32(r.IntN(2))
			h.must(m.TransposeRows(i, j))
			next := make(map[IndexPair]*big.Rat)
			for k, v := range want {
				if k.I == i {
					k.I = j
				} else if k.I == j {
					k.I = i
				}
				next[k] = v
			}
			want = next
		case 3:
			i := uint32(r.IntN(int(rows)))
			a := big.NewRat(int64(r.IntN(5)-2), int64(r.IntN(5)+1))
			h.must(h.m.SetString(&p.Y, a.RatString()))
			h.must(m.MultiplyRow(i, &p.Y))
			for k, v := range want {
				if k.I == i {
					v.Mul(v, a)
				}
			}
		}
		if m.RowCount() != rows || m.ColumnCount() != columns || len(m.stack) != len(stack) {
			t.Fatal("scope dimensions")
		}
		lpExactStaticCheck(t, h, m, want)
		ok, e := m.IsCorrect(nil)
		h.must(e)
		if !ok {
			t.Fatal("source diagnostic")
		}
		var domain []IndexPair
		for k := range want {
			domain = append(domain, k)
		}
		sort.Slice(domain, func(i, j int) bool {
			return domain[i].I < domain[j].I || domain[i].I == domain[j].I && domain[i].J < domain[j].J
		})
		if !reflect.DeepEqual(m.Domain(), domain) {
			t.Fatal("domain")
		}
	}
	h.must(m.Pop(99))
	if m.RowCount() != 2 || m.ColumnCount() != 2 {
		t.Fatal("excess pop")
	}
	// Scopes never restore baseline values or shrink the offset workspace.
	if len(m.RowOffsets) < 2 {
		t.Fatal("workspace")
	}
	h.must(m.Push())
	h.must(m.Clear())
	if len(m.stack) != 1 {
		t.Fatal("Clear removed scope")
	}
	h.must(m.Pop(1))
	if m.RowCount() != 0 || m.ColumnCount() != 0 {
		t.Fatal("Pop regrew cleared rows")
	}
	h.must(m.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactStaticTransposeAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"distinct", "self", "empty-first", "empty-second"} {
		failures, successes := 0, 0
		stages := [3]int{}
		for ordinal := int64(0); ordinal < 100; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				m := NewRationalStaticMatrix(h.m, 2, 4)
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(m.Close()) })
				x, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
				y, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/13")
				before := make(map[IndexPair]*big.Rat)
				if op != "empty-first" {
					h.must(m.Set(0, 0, &p.X))
					h.must(m.Set(0, 1, &p.Y))
					before[IndexPair{0, 0}] = x
					before[IndexPair{0, 1}] = y
				}
				if op != "empty-second" {
					h.must(m.Set(1, 2, &p.Y))
					h.must(m.Set(1, 3, &p.X))
					before[IndexPair{1, 2}] = y
					before[IndexPair{1, 3}] = x
				}
				j := uint32(1)
				if op == "self" {
					j = 0
				}
				h.budget.FailAfter(ordinal)
				err := m.TransposeRows(0, j)
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
					after := make(map[IndexPair]*big.Rat)
					for k, v := range before {
						if k.I == 0 {
							k.I = j
						} else if k.I == j {
							k.I = 0
						}
						after[k] = v
					}
					lpExactStaticCheck(t, h, m, after)
					if m.transposeRow != nil {
						t.Fatal("retained completed row temporary")
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if op == "distinct" {
						if m.Rows[1][0].Column == 0 {
							stages[2]++
						} else if m.Rows[0][0].Column == 2 {
							stages[1]++
						} else {
							stages[0]++
						}
					}
					if m.ready() != ErrExactVectorPending {
						t.Fatal("lost stopped copies")
					}
					if e := m.Push(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.Pop(0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.TransposeRows(0, 1); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if _, e := m.IsCorrect(nil); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				}
				h.must(m.Close())
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete campaign", op, failures, successes)
		}
		if op == "distinct" {
			for i, n := range stages {
				if n == 0 {
					t.Fatal("missing transpose copy stage", i, stages)
				}
			}
		}
	}
}

func TestLPExactStaticScopesLiteralStopsAndDiagnosticGaps(t *testing.T) {
	for _, op := range []string{"tail-mismatch", "column-bounds", "column-empty", "old-row-new-column", "transpose-bounds", "transpose-first-bounds", "transpose-second-bounds", "transpose-first-assert", "transpose-second-assert", "duplicate-column", "duplicate-row", "bad-row-column", "bad-row-offset", "bad-column-row", "bad-column-offset", "row-value-mismatch", "column-value-bounds", "orphan-column"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "3")
			m := NewRationalStaticMatrix(h.m, 2, 2)
			t.Cleanup(func() { h.must(m.Close()) })
			h.must(m.Set(0, 0, &p.X))
			h.must(m.Set(1, 1, &p.Y))
			var err error
			want := ErrBounds
			switch op {
			case "tail-mismatch":
				h.must(m.Push())
				h.must(m.AddRow())
				h.must(m.Set(2, 0, &p.X))
				h.must(m.Set(1, 0, &p.X))
				err = m.Pop(1)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "column-bounds", "column-empty":
				h.must(m.Push())
				h.must(m.AddRow())
				h.must(m.Set(2, 0, &p.X))
				if op == "column-bounds" {
					m.Rows[2][0].Column = 7
				} else {
					m.Columns[0] = nil
				}
				err = m.Pop(1)
			case "old-row-new-column":
				h.must(m.Push())
				h.must(m.AddColumn())
				h.must(m.Set(0, 2, &p.X))
				err = m.Pop(1)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
				if m.ColumnCount() != 2 || m.Size() != 3 {
					t.Fatal("scope literal orphan")
				}
			case "transpose-bounds":
				err = m.TransposeRows(0, 7)
			case "transpose-first-bounds":
				m.Rows[1][0].Column = 7
				err = m.TransposeRows(0, 1)
			case "transpose-second-bounds":
				m.Rows[0][0].Column = 7
				err = m.TransposeRows(0, 1)
				if m.Columns[1][0].Row != 0 {
					t.Fatal("first repair lost")
				}
			case "transpose-first-assert":
				m.Columns[1][0].Row = 0
				err = m.TransposeRows(0, 1)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "transpose-second-assert":
				m.Columns[0][0].Row = 1
				err = m.TransposeRows(0, 1)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
				if m.Columns[1][0].Row != 0 {
					t.Fatal("first repair lost")
				}
			default:
				want = nil
				expected := false
				var out bytes.Buffer
				switch op {
				case "duplicate-column":
					h.must(m.Set(0, 0, &p.Y))
				case "duplicate-row":
					m.Columns[0] = append(m.Columns[0], m.Columns[0][0])
				case "bad-row-column":
					m.Rows[0][0].Column = 7
				case "bad-row-offset":
					m.Rows[0][0].Offset = 7
				case "bad-column-row":
					m.Columns[0] = append(m.Columns[0], ColumnCell{7, 0})
				case "bad-column-offset":
					m.Columns[0] = append(m.Columns[0], ColumnCell{1, 7})
				case "row-value-mismatch":
					m.Columns[0][0] = ColumnCell{1, 0}
				case "column-value-bounds":
					m.Columns[0][0] = ColumnCell{7, 0}
					want = ErrBounds
				case "orphan-column":
					m.Columns[1] = append(m.Columns[1], ColumnCell{0, 0})
					expected = true
				}
				var ok bool
				ok, err = m.IsCorrect(&out)
				if ok != expected {
					t.Fatal("diagnostic", ok, expected, err)
				}
				if op == "duplicate-column" && out.String() != "found column 0 twice in a row\n" {
					t.Fatal(out.String())
				}
				if op == "duplicate-row" && out.String() != "found row 0 twice in a column\n" {
					t.Fatal(out.String())
				}
				if op == "duplicate-column" || op == "duplicate-row" {
					failure := fmt.Errorf("writer stopped")
					if _, e := m.IsCorrect(lpStaticRefusingWriter{failure}); e != failure {
						t.Fatal("ignored writer error")
					}
				}
				if op == "duplicate-column" && !reflect.DeepEqual(m.Domain(), []IndexPair{{0, 0}, {1, 1}}) {
					t.Fatal("domain did not deduplicate")
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
