package lp

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactLUUpdateAllocationFailureCensus(t *testing.T) {
	argumentStops, arithmeticStops, rowEtaStops, diagonalStops, publishedDiagonalStops := 0, 0, 0, 0, 0
	for _, op := range []string{"no-bump", "bump", "dense"} {
		for _, pair := range []bool{false, true} {
			for _, direct := range []bool{false, true} {
				t.Run(fmt.Sprint(op, "/pair=", pair, "/direct=", direct), func(t *testing.T) {
					failures, successes := 0, 0
					for ordinal := int64(0); ordinal < 1800; ordinal++ {
						done := false
						t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
							var h *lpPairHarness
							if direct {
								h = lpNewDirectSparseHarness(t)
							} else {
								h = newLPPairHarness(t)
							}
							n := 3
							density := math.Inf(1)
							if op == "dense" {
								n = 5
								density = 0
							}
							x := mustLPRat(t, "1"+strings.Repeat("0", 31)+"1/7")
							y := mustLPRat(t, "2"+strings.Repeat("0", 47)+"3/13")
							if x.Denom().Cmp(big.NewInt(7)) != 0 || y.Denom().Cmp(big.NewInt(13)) != 0 {
								t.Fatal("genuine arbitrary-precision fractions")
							}
							b := lpExactIdentity(n)
							basis := make([]uint32, n)
							for i := range b {
								basis[i] = uint32(i)
								for j := range b[i] {
									b[i][j].Set(x)
									if (i+j)%2 == 0 {
										b[i][j].Set(y)
									}
									if i == j {
										b[i][j].Add(x, y)
										b[i][j].Mul(b[i][j], big.NewRat(int64(20*n+i), 1))
									}
									if op == "dense" {
										b[i][j].SetFrac64(int64(i+j+1), 7)
										if i == j {
											b[i][j].SetInt64(int64(50 + i))
										}
									}
								}
							}
							original := lpExactIdentity(n)
							for i := range b {
								for j := range b[i] {
									original[i][j].Set(b[i][j])
								}
							}
							a := lpExactLUStatic(t, h, b, basis)
							var r *RationalLU
							var p *PairLU
							var state *exactLUState
							var err error
							if pair {
								p, err = NewPairLU(a, basis, lpFloatLUSettings(density))
								state = p.exactLUState
							} else {
								r, err = NewRationalLU(a, basis, lpFloatLUSettings(density))
								state = r.exactLUState
							}
							h.must(err)
							w, d := NewIndexedRationalVector(h.m, uint32(n)), NewIndexedRationalVector(h.m, uint32(n))
							defer func() {
								if r != nil {
									h.must(r.Close())
								}
								if p != nil {
									h.must(p.Close())
								}
								h.must(w.Close())
								h.must(d.Close())
							}()
							pivot := h.pair(x.RatString(), "0")
							logical := uint32(0)
							if op == "no-bump" {
								logical = uint32(n - 1)
							}
							leaving := state.r.values[logical]
							comb := make([]*big.Rat, n)
							for i := range comb {
								comb[i] = new(big.Rat)
								if op != "no-bump" {
									if i%2 == 0 {
										comb[i].Set(x)
									} else {
										comb[i].Set(y)
									}
								}
							}
							comb[leaving].Set(y)
							column := lpExactMatrixVector(b, comb, false)
							entering := uint32(n)
							tmp := h.pair("0", "0")
							for i := range column {
								h.must(h.m.SetString(&tmp.X, column[i].RatString()))
								h.must(a.Set(uint32(i), entering, &tmp.X))
							}
							if pair {
								h.must(p.SolveBd(entering, d, w))
							} else {
								h.must(r.SolveBd(entering, d, w))
							}
							oldTails := 0
							if pair {
								oldTails = len(p.Tail)
							} else {
								oldTails = len(r.Tail)
							}
							h.budget.FailAfter(ordinal)
							if pair {
								err = p.ReplaceColumn(&pivot.X, w, leaving)
							} else {
								err = r.ReplaceColumn(&pivot.X, w, leaving)
							}
							h.budget.FailAfter(-1)
							if err == nil {
								done = true
								successes++
								if state.status != LUOK || state.RefactorCounter != 1 {
									t.Fatal("successful update", state.status, state.RefactorCounter)
								}
								for i := range b {
									b[i][leaving] = new(big.Rat).Set(column[i])
								}
								want, wantY := make([]*big.Rat, n), make([]*big.Rat, n)
								for i := range want {
									want[i] = big.NewRat(int64(i+1), 11)
									wantY[i] = big.NewRat(int64(2*i-1), 17)
								}
								lpExactLUUpdateCheckSolutions(t, h, r, p, b, want, wantY)
							} else {
								failures++
								if !lpExactPriorityPhysicalRefusal(err) {
									t.Fatal(err)
								}
								if state.RefactorCounter == 0 {
									argumentStops++
								}
								if state.updateWork != nil && state.updateWork.calls.pending() {
									arithmeticStops++
								}
								if state.pendingRowEta != nil {
									rowEtaStops++
								}
								if state.pendingDiagonal != nil {
									diagonalStops++
								}
								if pair {
									if len(p.Tail) > oldTails {
										if _, ok := p.Tail[len(p.Tail)-1].(*PairDiagonalMatrix); ok {
											publishedDiagonalStops++
										}
									}
								} else {
									if len(r.Tail) > oldTails {
										if _, ok := r.Tail[len(r.Tail)-1].(*RationalDiagonalMatrix); ok {
											publishedDiagonalStops++
										}
									}
								}
								if pair {
									err = p.ReplaceColumn(&pivot.X, w, leaving)
								} else {
									err = r.ReplaceColumn(&pivot.X, w, leaving)
								}
								if err != ErrLUStopped {
									t.Fatal("failed update not stopped", err)
								}
								if w.calls.pending() {
									t.Fatal("input adopted child receipt")
								}
							}
							if r != nil {
								h.must(r.Close())
							}
							if p != nil {
								h.must(p.Close())
							}
							// Source and caller cells remain owned independently of every
							// successfully published or partially constructed update tail.
							var cells []*numeral.Rational
							cells = append(cells, lpExactLUStaticCells(a)...)
							for i := range w.Data {
								cells = append(cells, &w.Data[i])
							}
							for i := range d.Data {
								cells = append(cells, &d.Data[i])
							}
							lpExactVectorNamedCensus(t, h, cells)
							h.must(w.Close())
							h.must(d.Close())
							lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
							source := make([][]*big.Rat, n)
							cols := make([]uint32, n+1)
							for j := range cols {
								cols[j] = uint32(j)
							}
							for i := range source {
								source[i] = make([]*big.Rat, n+1)
								for j := 0; j < n; j++ {
									source[i][j] = original[i][j]
								}
								source[i][n] = column[i]
							}
							lpExactLUCheckSource(t, h, a, source, cols)
							h.must(a.Close())
							lpExactVectorNamedCensus(t, h, nil)
						})
						if done {
							break
						}
					}
					if successes != 1 || (direct && failures == 0) {
						t.Fatal("incomplete allocation walk", failures, successes)
					}
					t.Logf("physical refusals=%d", failures)
				})
			}
		}
	}
	if argumentStops == 0 || arithmeticStops == 0 || rowEtaStops == 0 || diagonalStops == 0 || publishedDiagonalStops == 0 {
		t.Fatal("missing failure stages", argumentStops, arithmeticStops, rowEtaStops, diagonalStops, publishedDiagonalStops)
	}
	t.Log("actual failure stages", argumentStops, arithmeticStops, rowEtaStops, diagonalStops, publishedDiagonalStops)
}

func TestLPExactLUUpdateLeavesOlderComparisonOwned(t *testing.T) {
	for _, pair := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprint(pair, "/", ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				a := lpExactLUStatic(t, h, lpExactIdentity(2), []uint32{0, 1})
				large := h.pair("1"+strings.Repeat("0", 1000)+"/7", "2"+strings.Repeat("0", 1000)+"/13")
				one := h.pair("1", "0")
				var r *RationalLU
				var p *PairLU
				var state *exactLUState
				var err error
				if pair {
					p, err = NewPairLU(a, []uint32{0, 1}, lpFloatLUSettings(math.Inf(1)))
					state = p.exactLUState
				} else {
					r, err = NewRationalLU(a, []uint32{0, 1}, lpFloatLUSettings(math.Inf(1)))
					state = r.exactLUState
				}
				h.must(err)
				w := NewIndexedRationalVector(h.m, 2)
				h.must(w.SetValue(&one.X, 0))
				var caller exactPriorityCalls
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(w.Close())
					h.must(caller.cleanupRaw(h.m))
					h.must(caller.cleanupRoots(h.m))
				}()
				before, err := caller.begin(h.m)
				h.must(err)
				h.budget.FailAfter(ordinal)
				_, err = h.m.Compare(&large.X, &large.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if pair {
						err = p.ReplaceColumn(&one.X, w, 0)
					} else {
						err = r.ReplaceColumn(&one.X, w, 0)
					}
					if err != nil && err != numeral.ErrRationalRawArithmeticPending {
						t.Fatal(err)
					}
					if (state.updateWork != nil && state.updateWork.calls.comparison != 0) || state.U.calls.comparison != 0 {
						t.Fatal("adopted old comparison")
					}
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					if h.m.ComparisonInvocationState() != older {
						t.Fatal("closed foreign comparison")
					}
				}
				if pair {
					h.must(p.Close())
				} else {
					h.must(r.Close())
				}
				h.must(w.Close())
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
		if !matched {
			t.Fatal("no real older comparison")
		}
	}
}

func TestLPExactLUUpdateRowFillAllocationCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			failures, successes, fillCopies := 0, 0, 0
			for ordinal := int64(0); ordinal < 700; ordinal++ {
				done := false
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := lpExactLUStatic(t, h, lpExactIdentity(3), []uint32{0, 1, 2})
					p := h.pair("1"+strings.Repeat("0", 31)+"1/7", "2"+strings.Repeat("0", 47)+"3/13")
					one := h.pair("1", "0")
					m, err := NewRationalLU(a, []uint32{0, 1, 2}, lpFloatLUSettings(math.Inf(1)))
					h.must(err)
					input := NewIndexedRationalVector(h.m, 3)
					defer func() { h.must(m.Close()); h.must(input.Close()) }()
					m.U.RowPermutation.Init(3)
					m.U.ColumnPermutation.Init(3)
					h.must(m.beginUpdate(&one.X, input))
					h.must(m.U.AddNewElement(0, 1, &p.Y))
					h.must(m.U.AddNewElement(1, 2, &one.X))
					h.must(m.rowEtaWork.SetValue(&p.X, 0))
					h.must(m.rowEtaWork.SetValue(&one.X, 2))
					x := mustLPRat(t, "1"+strings.Repeat("0", 31)+"1/7")
					y := mustLPRat(t, "2"+strings.Repeat("0", 47)+"3/13")
					delta := new(big.Rat).Neg(new(big.Rat).Mul(x, y))
					h.budget.FailAfter(ordinal)
					err = m.solveExactUpdateRow(0, 2)
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
						successes++
						lpExactLUCheckVector(t, h, m.rowEtaWork.Data, []*big.Rat{x, delta, new(big.Rat).Add(big.NewRat(1, 1), delta)})
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if !m.updateWork.calls.pending() && len(m.rowEtaWork.Index) == 2 {
							s, e := h.m.String(&m.updateWork.Data[3])
							v, ok := new(big.Rat).SetString(s)
							if e == nil && ok && v.Cmp(delta) == 0 {
								fillCopies++
							}
						}
					}
					m.finishUpdate(&err)
					h.must(m.Close())
					h.must(input.Close())
					lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				})
				if done {
					break
				}
			}
			if successes != 1 || (direct && (failures == 0 || fillCopies == 0)) {
				t.Fatal("row fill campaign", failures, successes, fillCopies)
			}
			t.Logf("physical refusals=%d, new workspace copy stops=%d", failures, fillCopies)
		})
	}
}
