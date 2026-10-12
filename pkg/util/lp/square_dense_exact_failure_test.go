package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactDenseBlockAllocationFailureCensus(t *testing.T) {
	receipts, publishedL := 0, 0
	for _, op := range []string{"constructor", "find", "divide", "row", "pivot"} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", direct), func(t *testing.T) {
				failures, successes := 0, 0
				for ordinal := int64(0); ordinal < 1200; ordinal++ {
					done := false
					t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						xText, yText := strings.Repeat("1234567890", 12)+"/7", strings.Repeat("9876543210", 12)+"/13"
						p := h.pair(xText, yText)
						x, _ := new(big.Rat).SetString(xText)
						y, _ := new(big.Rat).SetString(yText)
						parent := lpNewExactSparse(t, h, 2)
						h.must(parent.AddNewElement(0, 0, &p.X))
						h.must(parent.AddNewElement(0, 1, &p.Y))
						h.must(parent.AddNewElement(1, 0, &p.Y))
						h.must(parent.AddNewElement(1, 1, &p.X))
						var m *RationalSquareDenseSubmatrix
						var err error
						if op != "constructor" {
							m, err = NewRationalSquareDenseSubmatrix(parent, 0)
							h.must(err)
							if op == "row" {
								h.must(m.DivideRowByPivot(0))
							}
						}
						h.budget.FailAfter(ordinal)
						best := int32(-1)
						switch op {
						case "constructor":
							m, err = NewRationalSquareDenseSubmatrix(parent, 0)
						case "find":
							best, err = m.FindPivotColumn(0)
						case "divide":
							err = m.DivideRowByPivot(0)
						case "row":
							err = m.PivotRowToRow(0, 1, nil)
						case "pivot":
							err = m.Pivot(0, nil)
						}
						h.budget.FailAfter(-1)
						if err == nil {
							successes++
							done = true
							if op == "find" && best != 1 {
								t.Fatal("max pivot", best)
							}
							want := []*big.Rat{new(big.Rat).Set(x), new(big.Rat).Set(y), new(big.Rat).Set(y), new(big.Rat).Set(x)}
							if op == "divide" || op == "row" || op == "pivot" {
								want[0].Inv(x)
								want[1].Quo(y, x)
							}
							if op == "row" || op == "pivot" {
								want[2].Neg(new(big.Rat).Quo(y, x))
								want[3].Sub(x, new(big.Rat).Quo(new(big.Rat).Mul(y, y), x))
							}
							for i := range want {
								lpSparseRationalCheck(t, h, &m.Data[i], want[i])
							}
							lpExactVectorNamedCensus(t, h, lpExactDenseCells(m))
						} else {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if m.calls.pending() {
								receipts++
							}
							if e := m.SwapColumns(0, 1); e != ErrExactVectorPending {
								t.Fatal("stopped swap", e)
							}
							if _, e := m.FindPivotColumn(0); e != ErrExactVectorPending {
								t.Fatal("stopped max", e)
							}
							if e := m.DivideRowByPivot(0); e != ErrExactVectorPending {
								t.Fatal("stopped divide", e)
							}
							if e := m.PivotRowToRow(0, 1, nil); e != ErrExactVectorPending {
								t.Fatal("stopped row", e)
							}
							if parent.work != nil || parent.calls.pending() {
								t.Fatal("parent adopted dense operation")
							}
							// Read only when no arithmetic/physical receipt is active.
							// A later copy refusal must preserve the completed L entry.
							if op == "row" && direct && !m.calls.pending() {
								l, el := h.m.String(&m.Data[2])
								r, er := h.m.String(&m.Data[3])
								lv, okl := new(big.Rat).SetString(l)
								rv, okr := new(big.Rat).SetString(r)
								if el == nil && er == nil && okl && okr && lv.Cmp(new(big.Rat).Neg(new(big.Rat).Quo(y, x))) == 0 && rv.Cmp(x) == 0 {
									publishedL++
								}
							}
						}
						h.must(m.Close())
						// Dense construction and pivoting never mutate the borrowed parent.
						lpExactSparseCheck(t, h, parent, map[IndexPair]*big.Rat{{0, 0}: x, {0, 1}: y, {1, 0}: y, {1, 1}: x})
						h.must(parent.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if successes != 1 || (direct && failures == 0) {
					t.Fatalf("incomplete campaign %d %d", failures, successes)
				}
			})
		}
	}
	if receipts == 0 || publishedL == 0 {
		t.Fatal("missing real numerical receipt/L publication failure", receipts, publishedL)
	}
}

// A dense owner may observe an older stopped comparison but must never adopt it.
func TestLPExactDenseBlockLeavesOlderComparisonOwned(t *testing.T) {
	for _, direct := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				parent := lpNewExactSparse(t, h, 2)
				h.must(parent.AddNewElement(0, 0, &p.X))
				h.must(parent.AddNewElement(0, 1, &p.Y))
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
					matched = true
					// Construction copies coefficients without using the foreign comparison.
					m, err := NewRationalSquareDenseSubmatrix(parent, 0)
					h.must(err)
					_, err = m.FindPivotColumn(0)
					if err != numeral.ErrRationalRawArithmeticPending || m.calls.comparison != 0 {
						t.Fatal("older comparison ownership", err, m.calls.comparison)
					}
					h.must(m.Close())
					if h.m.ComparisonInvocationState() != older {
						t.Fatal("closed caller's comparison")
					}
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				h.must(parent.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no real comparison refusal")
		}
	}
}
