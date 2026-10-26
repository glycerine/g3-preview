package lp

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPExactLUAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	etaStops, denseStops, basisStops, tailPrefixes := 0, 0, 0, 0
	for _, op := range []string{"constructor", "column", "row", "indexed-row"} {
		for _, dense := range []bool{false, true} {
			for _, pair := range []bool{false, true} {
				for _, direct := range []bool{false, true} {
					t.Run(fmt.Sprint(op, "/dense=", dense, "/pair=", pair, "/direct=", direct), func(t *testing.T) {
						failures, successes := 0, 0
						for ordinal := int64(0); ordinal < 1600; ordinal++ {
							done := false
							t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
								var h *lpPairHarness
								if direct {
									h = lpNewDirectSparseHarness(t)
								} else {
									h = newLPPairHarness(t)
								}
								// Distinct large numerators avoid a common scale
								// cancelling to inline ratios during elimination.
								x, ok := new(big.Rat).SetString("1" + strings.Repeat("0", 95) + "1/7")
								if !ok || x.Denom().Cmp(big.NewInt(7)) != 0 {
									t.Fatal("genuine x fraction")
								}
								y, ok := new(big.Rat).SetString("2" + strings.Repeat("0", 95) + "3/13")
								if !ok || y.Denom().Cmp(big.NewInt(13)) != 0 {
									t.Fatal("genuine y fraction")
								}
								n, offset, density := 2, 0, math.Inf(1)
								if dense {
									n, offset, density = 5, 1, 0
								}
								b := lpExactIdentity(n)
								b[offset][offset].Set(x)
								b[offset][offset+1].Set(y)
								b[offset+1][offset].Set(y)
								b[offset+1][offset+1].Set(x)
								basis := make([]uint32, n)
								want, wantY := make([]*big.Rat, n), make([]*big.Rat, n)
								for i := range basis {
									basis[i] = uint32(i)
									// Keep the transformed RHS large too, so refusals
									// reach Q, U and R after all tail applications.
									scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(80), nil)
									want[i] = new(big.Rat).SetFrac(new(big.Int).Add(new(big.Int).Mul(big.NewInt(int64(i+1)), scale), big.NewInt(1)), big.NewInt(7))
									wantY[i] = new(big.Rat).SetFrac(new(big.Int).Add(new(big.Int).Mul(big.NewInt(int64(2*i-3)), scale), big.NewInt(2)), big.NewInt(13))
								}
								a := lpExactLUStatic(t, h, b, basis)
								var r *RationalLU
								var p *PairLU
								var scalar *IndexedRationalVector
								var pairs *IndexedPairVector
								// Children own their arithmetic receipts; release them before
								// caller coordinates on every path, including a failing assertion.
								defer func() {
									if r != nil {
										h.must(r.Close())
									}
									if p != nil {
										h.must(p.Close())
									}
									if scalar != nil {
										h.must(scalar.Close())
									}
									if pairs != nil {
										h.must(pairs.Close())
									}
								}()
								s := lpFloatLUSettings(density)
								var err error
								if op != "constructor" {
									if pair {
										p, err = NewPairLU(a, basis, s)
									} else {
										r, err = NewRationalLU(a, basis, s)
									}
									h.must(err)
									rhs := lpExactMatrixVector(b, want, op != "column")
									if pair && op == "column" {
										pairs = NewIndexedPairVector(h.m, uint32(n))
										rhsY := lpExactMatrixVector(b, wantY, false)
										for i := range pairs.Data {
											h.must(h.m.SetString(&pairs.Data[i].X, rhs[i].RatString()))
											h.must(h.m.SetString(&pairs.Data[i].Y, rhsY[i].RatString()))
										}
									} else {
										scalar = lpExactLUVector(h, rhs)
									}
								}
								h.budget.FailAfter(ordinal)
								switch op {
								case "constructor":
									if pair {
										p, err = NewPairLU(a, basis, s)
									} else {
										r, err = NewRationalLU(a, basis, s)
									}
								case "column":
									if pair {
										err = p.SolveBy(pairs.Data)
									} else {
										err = r.SolveBy(scalar.Data)
									}
								case "row":
									if pair {
										err = p.SolveYB(scalar.Data)
									} else {
										err = r.SolveYB(scalar.Data)
									}
								case "indexed-row":
									if pair {
										err = p.SolveYBIndexed(scalar)
									} else {
										err = r.SolveYBIndexed(scalar)
									}
								}
								h.budget.FailAfter(-1)
								var state *exactLUState
								if pair {
									state = p.exactLUState
								} else {
									state = r.exactLUState
								}
								if err == nil {
									done = true
									successes++
									if state.GetStatus() != LUOK || state.Failure {
										t.Fatal("successful factorization status")
									}
									if (state.Dense != nil) != dense {
										t.Fatal("campaign did not reach selected factorization mode")
									}
									if op != "constructor" {
										if pairs != nil {
											for i := range pairs.Data {
												h.check(&pairs.Data[i], want[i], wantY[i])
											}
										} else {
											lpExactLUCheckVector(t, h, scalar.Data, want)
										}
									}
								} else {
									if !lpExactPriorityPhysicalRefusal(err) {
										t.Fatal(err)
									}
									failures++
									if state.ready() != ErrLUStopped {
										t.Fatal("failed LU not stopped")
									}
									if state.pendingEta != nil {
										etaStops++
									}
									if state.denseOwned {
										denseStops++
									}
									if state.rowEtaWork == nil {
										basisStops++
									}
									if pair {
										tailPrefixes += len(p.Tail)
										if p.SolveBy(nil) != ErrLUStopped || p.SolveYB(nil) != ErrLUStopped || p.SolveYBIndexed(nil) != ErrLUStopped {
											t.Fatal("stopped pair routes")
										}
									} else {
										tailPrefixes += len(r.Tail)
										if r.SolveBy(nil) != ErrLUStopped || r.SolveYB(nil) != ErrLUStopped || r.SolveYBIndexed(nil) != ErrLUStopped {
											t.Fatal("stopped scalar routes")
										}
									}
								}
								if r != nil {
									h.must(r.Close())
								}
								if p != nil {
									h.must(p.Close())
								}
								if scalar != nil {
									h.must(scalar.Close())
								}
								if pairs != nil {
									h.must(pairs.Close())
								}
								lpExactLUCheckSource(t, h, a, b, basis)
								lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
							})
							if done {
								break
							}
						}
						if successes != 1 || (direct && failures == 0) {
							t.Fatalf("failures=%d successes=%d", failures, successes)
						}
						t.Logf("%d real allocation refusals before first success", failures)
					})
				}
			}
		}
	}
	if etaStops == 0 || denseStops == 0 || basisStops == 0 || tailPrefixes == 0 {
		t.Fatalf("missing ownership stages eta=%d dense=%d basis=%d tails=%d", etaStops, denseStops, basisStops, tailPrefixes)
	}
}

func TestLPExactLULeavesOlderComparisonOwned(t *testing.T) {
	for _, pair := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprint(pair, "/", ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				v := h.pair("1"+strings.Repeat("0", 1000)+"/7", "2"+strings.Repeat("0", 1000)+"/13")
				x, _ := new(big.Rat).SetString("1" + strings.Repeat("0", 1000) + "/7")
				y, _ := new(big.Rat).SetString("2" + strings.Repeat("0", 1000) + "/13")
				b := [][]*big.Rat{{x, y}, {y, x}}
				a := lpExactLUStatic(t, h, b, []uint32{0, 1})
				var caller exactPriorityCalls
				before, err := caller.begin(h.m)
				h.must(err)
				h.budget.FailAfter(ordinal)
				_, err = h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					matched = true
					if pair {
						m, e := NewPairLU(a, []uint32{0, 1}, lpFloatLUSettings(1))
						if e != numeral.ErrRationalRawArithmeticPending {
							t.Fatal(e)
						}
						h.must(m.Close())
					} else {
						m, e := NewRationalLU(a, []uint32{0, 1}, lpFloatLUSettings(1))
						if e != numeral.ErrRationalRawArithmeticPending {
							t.Fatal(e)
						}
						h.must(m.Close())
					}
					if h.m.ComparisonInvocationState() != older {
						t.Fatal("LU consumed another owner's comparison")
					}
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactLUCheckSource(t, h, a, b, []uint32{0, 1})
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
		if !matched {
			t.Fatal("no real comparison refusal")
		}
	}
}

func TestLPExactLUBorrowedStoppedSource(t *testing.T) {
	for _, pair := range []bool{false, true} {
		t.Run(fmt.Sprint(pair), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			a := NewRationalStaticMatrix(h.m, 1, 1)
			v := h.pair("1"+strings.Repeat("0", 100)+"/7", "0")
			h.budget.FailAfter(0)
			err := a.Set(0, 0, &v.X)
			h.budget.FailAfter(-1)
			if !lpExactPriorityPhysicalRefusal(err) || a.ready() != ErrExactVectorPending {
				t.Fatal("expected actual stopped source", err)
			}
			older := h.m.RawInvocationState()
			if pair {
				m, e := NewPairLU(a, []uint32{0}, lpFloatLUSettings(1))
				if e != ErrExactVectorPending || m.U != nil || m.rowEtaWork != nil {
					t.Fatal("stopped source pair constructor", e)
				}
				h.must(m.Close())
			} else {
				m, e := NewRationalLU(a, []uint32{0}, lpFloatLUSettings(1))
				if e != ErrExactVectorPending || m.U != nil || m.rowEtaWork != nil {
					t.Fatal("stopped source scalar constructor", e)
				}
				h.must(m.Close())
			}
			if h.m.RawInvocationState() != older {
				t.Fatal("LU consumed source receipt")
			}
			h.must(a.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
