package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactDenseBlockParentPublicationStages(t *testing.T) {
	h := newLPPairHarness(t)
	parent := lpNewExactSparse(t, h, 3)
	p := h.pair("9", "0")
	// Missing diagonal, common upper, cancelled upper, lower removal, and fill.
	for _, at := range []IndexPair{{0, 1}, {0, 2}, {1, 0}, {1, 1}, {2, 0}, {2, 1}} {
		h.must(parent.AddNewElement(at.I, at.J, &p.X))
	}
	m, err := NewRationalSquareDenseSubmatrix(parent, 0)
	h.must(err)
	input := []string{"1/2", "1/1000000000000000000000000000000", "0", "-3/2", "1/4", "7", "-2", "-5", "1/6"}
	for i, x := range input {
		h.must(h.m.SetString(&m.Data[i], x))
	}
	h.must(m.UpdateExistingParentRow(0, nil))
	// Existing pass clears the common upper coefficient but preserves new fill.
	lpSparseRationalCheck(t, h, &m.Data[1], new(big.Rat))
	lpSparseRationalCheck(t, h, &m.Data[5], big.NewRat(7, 1))
	h.must(m.UpdateExistingParentRow(1, nil))
	h.must(m.UpdateExistingParentRow(2, nil))
	h.must(m.PushNewParentElements(&Settings{DropTolerance: 1e100}))
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			lpSparseRationalCheck(t, h, &m.Data[i*3+j], new(big.Rat))
		}
	}
	for _, i := range []int{0, 3, 4, 6, 7, 8} {
		want, _ := new(big.Rat).SetString(input[i])
		lpSparseRationalCheck(t, h, &m.Data[i], want)
	}
	h.must(m.Close())
	tiny, _ := new(big.Rat).SetString(input[1])
	lpExactSparseCheck(t, h, parent, map[IndexPair]*big.Rat{{0, 0}: big.NewRat(1, 1), {0, 1}: tiny, {1, 1}: big.NewRat(1, 1), {1, 2}: big.NewRat(7, 1), {2, 2}: big.NewRat(1, 1)})
	h.must(parent.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactDenseBlockParentAllocationFailureCensus(t *testing.T) {
	denseStops, parentReceipts, rowBeforeColumn, fillRowBeforeColumn := 0, 0, 0, 0
	for _, op := range []string{"common", "remove", "fill", "update"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes := 0, 0
				for ordinal := int64(0); ordinal < 1000; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						p := h.pair(strings.Repeat("1234567890", 12)+"/7", strings.Repeat("9876543210", 12)+"/13")
						x, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 12) + "/7")
						y, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 12) + "/13")
						parent := lpNewExactSparse(t, h, 2)
						if op != "fill" {
							h.must(parent.AddNewElement(0, 0, &p.X))
							if op == "common" {
								one := h.pair("1", "0")
								h.must(parent.AddNewElement(0, 1, &one.X))
							} else {
								h.must(parent.AddNewElement(0, 1, &p.X))
							}
							if op == "remove" {
								one := h.pair("1", "0")
								h.must(parent.AddNewElement(1, 0, &one.X))
							} else {
								h.must(parent.AddNewElement(1, 0, &p.X))
							}
							h.must(parent.AddNewElement(1, 1, &p.Y))
						}
						m, err := NewRationalSquareDenseSubmatrix(parent, 0)
						h.must(err)
						h.must(h.m.Copy(&m.Data[1], &p.Y))
						h.budget.FailAfter(ordinal)
						switch op {
						case "common":
							err = m.UpdateExistingParentRow(0, nil)
						case "remove":
							err = m.UpdateExistingParentRow(1, nil)
						case "fill":
							err = m.PushNewParentElements(nil)
						case "update":
							err = m.UpdateParent(nil)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							done = true
							successes++
							if op != "remove" {
								lpSparseRationalCheck(t, h, &m.Data[1], new(big.Rat))
							}
							lpExactVectorNamedCensus(t, h, lpExactDenseCells(m))
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.work != nil && parent.work == nil {
								denseStops++
							}
							if parent.calls.pending() {
								parentReceipts++
							}
							if m.calls.pending() && parent.calls.pending() {
								t.Fatal("two owners adopted one invocation")
							}
							for _, f := range []func() error{
								func() error { return m.UpdateParent(nil) },
								func() error { return m.UpdateExistingParentRow(0, nil) },
								func() error { return m.PushNewParentElements(nil) },
								func() error { return m.Pivot(0, nil) },
							} {
								if e := f(); e != ErrExactVectorPending {
									t.Fatal("stopped dense owner", e)
								}
							}
							// Ordinary value inspection only after receipt cleanup, below.
						}
						h.must(m.Close())
						// Parent-owned failed helpers may retain numerical temporaries; release
						// only that parent's receipts before inspecting its published prefix.
						h.must(parent.calls.cleanupRaw(h.m))
						h.must(parent.calls.cleanupRoots(h.m))
						if err != nil && direct {
							if op == "common" && len(parent.Rows[0]) == 2 {
								r, er := h.m.String(&parent.Rows[0][1].Value)
								c, ec := h.m.String(&parent.Columns[1].Values[0].Value)
								rr, okr := new(big.Rat).SetString(r)
								cc, okc := new(big.Rat).SetString(c)
								if er == nil && ec == nil && okr && okc && rr.Cmp(y) == 0 && cc.Cmp(big.NewRat(1, 1)) == 0 {
									rowBeforeColumn++
								}
							}
							if op == "fill" && len(parent.Rows[0]) == 1 && len(parent.Columns[1].Values) == 0 {
								fillRowBeforeColumn++
							}
						}
						if err == nil {
							want := map[IndexPair]*big.Rat{}
							switch op {
							case "common":
								want = map[IndexPair]*big.Rat{{0, 0}: big.NewRat(1, 1), {0, 1}: y, {1, 0}: x, {1, 1}: y}
							case "remove":
								want = map[IndexPair]*big.Rat{{0, 0}: x, {0, 1}: x, {1, 1}: big.NewRat(1, 1)}
							case "fill":
								want = map[IndexPair]*big.Rat{{0, 1}: y}
							case "update":
								want = map[IndexPair]*big.Rat{{0, 0}: big.NewRat(1, 1), {0, 1}: y, {1, 1}: big.NewRat(1, 1)}
							}
							lpExactSparseCheck(t, h, parent, want)
						}
						h.must(parent.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if successes != 1 || (direct && failures == 0) {
					t.Fatal("incomplete failure campaign", failures, successes)
				}
			})
		}
	}
	if denseStops == 0 || parentReceipts == 0 || rowBeforeColumn == 0 || fillRowBeforeColumn == 0 {
		t.Fatal("missing actual ownership/publication prefixes", denseStops, parentReceipts, rowBeforeColumn, fillRowBeforeColumn)
	}
}

func TestLPExactDenseBlockParentErrorPrefixes(t *testing.T) {
	for _, kind := range []string{"row-map", "rows", "column-map", "cell", "other", "remove", "diagonal-map", "diagonal-add", "push-row-map", "push-cell", "push-column-map", "push-add", "stopped-parent"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			parent := m.Parent
			row := uint32(0)
			push := false
			switch kind {
			case "row-map":
				parent.RowPermutation.values = nil
			case "rows":
				parent.Rows = nil // fixture coefficients are inline
			case "column-map":
				parent.ColumnPermutation.values = nil
			case "cell":
				m.Data = m.Data[:1]
			case "other":
				parent.Columns[1].Values = nil
				h.must(h.m.SetInt64(&m.Data[1], 17))
			case "remove":
				row = 1
				parent.Rows[1][0].Other = 20
			case "diagonal-map":
				parent.Rows[0] = nil
				parent.ColumnPermutation.reverse = nil
			case "diagonal-add":
				parent.Rows[0] = nil
				parent.Columns = nil
			case "push-row-map":
				push = true
				parent.RowPermutation.values = nil
			case "push-cell":
				push = true
				m.Data = m.Data[:1]
			case "push-column-map":
				push = true
				parent.ColumnPermutation.reverse = nil
			case "push-add":
				push = true
				parent.Rows = nil
			case "stopped-parent":
				p := h.pair("1", "0")
				if err := parent.AddNewElement(2, 0, &p.X); err != ErrBounds {
					t.Fatal(err)
				}
			}
			var err error
			if kind == "stopped-parent" {
				err = m.UpdateParent(nil)
			} else if push {
				err = m.PushNewParentElements(nil)
			} else {
				err = m.UpdateExistingParentRow(row, nil)
			}
			want := ErrBounds
			if kind == "stopped-parent" {
				want = ErrExactVectorPending
			}
			if err != want {
				t.Fatal("publication guard", err, want)
			}
			if kind == "other" {
				lpSparseRationalCheck(t, h, &parent.Rows[0][1].Value, big.NewRat(17, 1))
				lpSparseRationalCheck(t, h, &m.Data[1], big.NewRat(17, 1))
			}
			if kind == "push-add" {
				lpSparseRationalCheck(t, h, &m.Data[1], big.NewRat(4, 1))
			}
			if kind == "remove" && (m.calls.pending() || parent.work == nil) {
				t.Fatal("delegated remove ownership")
			}
			if kind == "stopped-parent" && m.work != nil {
				t.Fatal("adopted already stopped parent")
			}
			h.must(m.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	// Empty suffixes execute no publication, preserving all earlier rows.
	for _, n := range []uint32{1, 3} {
		t.Run(fmt.Sprint("empty-suffix/", n), func(t *testing.T) {
			h := newLPPairHarness(t)
			parent := lpNewExactSparse(t, h, n)
			p := h.pair("2", "0")
			h.must(parent.AddNewElement(0, 0, &p.X))
			m, err := NewRationalSquareDenseSubmatrix(parent, n)
			h.must(err)
			h.must(m.UpdateParent(nil))
			h.must(m.PushNewParentElements(nil))
			h.must(m.Close())
			lpExactSparseCheck(t, h, parent, map[IndexPair]*big.Rat{{0, 0}: big.NewRat(2, 1)})
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactDenseBlockParentLeavesOlderComparisonOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			parent := lpNewExactSparse(t, h, 2)
			h.must(parent.AddNewElement(0, 0, &p.X))
			h.must(parent.AddNewElement(0, 1, &p.Y))
			m, err := NewRationalSquareDenseSubmatrix(parent, 0)
			h.must(err)
			var caller exactPriorityCalls
			b, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, b, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				err = m.UpdateParent(nil)
				if (err != numeral.ErrRationalRawArithmeticPending && err != memory.ErrAllocatorPending) || m.calls.pending() || parent.calls.pending() {
					t.Fatal("older comparison ownership", err, m.calls, parent.calls)
				}
				// A foreign physical root can stop an earlier by-value copy. Keep
				// walking real failure ordinals until publication reaches maxima.
				if err == numeral.ErrRationalRawArithmeticPending {
					matched = true
					if parent.work == nil {
						t.Fatal("missing stopped maximum workspace")
					}
					// Both publication passes precede maxima: the upper buffer has
					// cleared and the missing second diagonal was added.
					lpSparseRationalCheck(t, h, &m.Data[1], new(big.Rat))
					lpSparseRationalCheck(t, h, &parent.Rows[0][0].Value, big.NewRat(1, 1))
					lpSparseRationalCheck(t, h, &parent.Rows[1][0].Value, big.NewRat(1, 1))
				}
			}
			h.must(m.Close())
			h.must(parent.Close())
			if h.m.ComparisonInvocationState() != older {
				t.Fatal("closed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real older comparison refusal")
	}
}
