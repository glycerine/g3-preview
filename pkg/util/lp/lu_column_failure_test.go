package lp

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPLUEnteringColumnAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	columnPrefixes, oldDestinationIndices, childStops := 0, 0, 0
	for _, op := range []string{"prepare", "dense", "sparse", "faster"} {
		for _, pair := range []bool{false, true} {
			if op == "faster" && !pair {
				continue
			}
			for _, denseTail := range []bool{false, true} {
				for _, direct := range []bool{false, true} {
					t.Run(fmt.Sprint(op, "/pair=", pair, "/dense-tail=", denseTail, "/direct=", direct), func(t *testing.T) {
						failures, successes := 0, 0
						for ordinal := int64(0); ordinal < 2400; ordinal++ {
							done := false
							t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
								var h *lpPairHarness
								if direct {
									h = lpNewDirectSparseHarness(t)
								} else {
									h = newLPPairHarness(t)
								}
								n, offset, density := 2, 0, math.Inf(1)
								if denseTail {
									n, offset, density = 5, 1, 0
								}
								x, _ := new(big.Rat).SetString("1" + strings.Repeat("0", 47) + "1/7")
								y, _ := new(big.Rat).SetString("2" + strings.Repeat("0", 47) + "3/13")
								b := lpExactIdentity(n)
								b[offset][offset].Set(x)
								b[offset][offset+1].Set(y)
								b[offset+1][offset].Set(y)
								b[offset+1][offset+1].Set(x)
								basis := make([]uint32, n)
								want := make([]*big.Rat, n)
								for i := range basis {
									basis[i] = uint32(i)
									want[i], _ = new(big.Rat).SetString(fmt.Sprint(i+1) + strings.Repeat("0", 40) + "1/17")
								}
								rhs := lpExactMatrixVector(b, want, false)
								a := lpExactLUStatic(t, h, b, basis)
								entering := uint32(n)
								value := h.pair("0", "0")
								source := make([][]*big.Rat, n)
								sourceBasis := append(append([]uint32(nil), basis...), entering)
								for i := range rhs {
									h.must(h.m.SetString(&value.X, rhs[i].RatString()))
									h.must(a.Set(uint32(i), entering, &value.X))
									source[i] = append(append([]*big.Rat(nil), b[i]...), rhs[i])
								}
								s := lpFloatLUSettings(density)
								var r *RationalLU
								var p *PairLU
								var err error
								if pair {
									p, err = NewPairLU(a, basis, s)
								} else {
									r, err = NewRationalLU(a, basis, s)
								}
								h.must(err)
								size := uint32(n)
								if op == "sparse" {
									size = uint32(10*n + 1)
								}
								d, w := NewIndexedRationalVector(h.m, size), NewIndexedRationalVector(h.m, uint32(n))
								defer func() {
									if pair {
										h.must(p.Close())
									} else {
										h.must(r.Close())
									}
									h.must(d.Close())
									h.must(w.Close())
								}()
								h.must(h.m.SetString(&value.Y, "99"))
								h.must(d.SetValue(&value.Y, 0))
								h.must(w.SetValue(&value.Y, 0))
								h.budget.FailAfter(ordinal)
								switch op {
								case "prepare":
									if pair {
										err = p.PrepareEntering(entering, w)
									} else {
										err = r.PrepareEntering(entering, w)
									}
								case "dense", "sparse":
									if pair {
										err = p.SolveBd(entering, d, w)
									} else {
										err = r.SolveBd(entering, d, w)
									}
								case "faster":
									err = p.SolveBdFaster(entering, d)
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
									if op == "prepare" {
										for i := range want {
											expected := new(big.Rat)
											for j := range want {
												u, e := state.U.Get(uint32(i), uint32(j))
												h.must(e)
												text, e := h.m.String(u)
												h.must(e)
												v, ok := new(big.Rat).SetString(text)
												if !ok {
													t.Fatal(text)
												}
												expected.Add(expected, new(big.Rat).Mul(v, want[state.r.values[j]]))
											}
											lpSparseRationalCheck(t, h, &w.Data[i], expected)
										}
									} else {
										lpExactLUCheckVector(t, h, d.Data, want)
										lpColumnIndexCheck(t, d.Index, want, op == "dense")
									}
								} else {
									if !lpExactPriorityPhysicalRefusal(err) {
										t.Fatal(err)
									}
									failures++
									if state.ready() != ErrLUStopped {
										t.Fatal("failed column operation stayed usable")
									}
									// A refused copy can leave an owned partial value
									// without an active raw arithmetic invocation.
									if w.Size() < uint32(n) && op != "faster" {
										columnPrefixes++
									}
									if op == "dense" && w.Size() == uint32(n) && reflect.DeepEqual(d.Index, []uint32{0}) {
										oldDestinationIndices++
									}
									if state.U.ready() != nil {
										childStops++
									}
									if pair {
										if p.PrepareEntering(entering, w) != ErrLUStopped || p.SolveBd(entering, d, w) != ErrLUStopped || p.SolveBdFaster(entering, d) != ErrLUStopped {
											t.Fatal("stopped pair columns")
										}
									} else {
										if r.PrepareEntering(entering, w) != ErrLUStopped || r.SolveBd(entering, d, w) != ErrLUStopped {
											t.Fatal("stopped scalar columns")
										}
									}
								}
								if pair {
									h.must(p.Close())
								} else {
									h.must(r.Close())
								}
								h.must(d.Close())
								h.must(w.Close())
								lpExactLUCheckSource(t, h, a, source, sourceBasis)
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
	if columnPrefixes == 0 || oldDestinationIndices == 0 || childStops == 0 {
		t.Fatalf("missing failure stages: column-prefix=%d old-index=%d triangular-child=%d", columnPrefixes, oldDestinationIndices, childStops)
	}
}

func TestLPLUEnteringColumnLiteralStopsAndOwnership(t *testing.T) {
	for _, scalar := range []string{"float", "rational", "pair"} {
		for _, kind := range []string{"closed", "clear", "column", "extent", "short-triangular", "destination"} {
			t.Run(scalar+"/"+kind, func(t *testing.T) {
				h := newLPPairHarness(t)
				b := lpExactIdentity(2)
				basis := []uint32{0, 1}
				s := lpFloatLUSettings(math.Inf(1))
				entering := uint32(2)
				if kind == "column" {
					entering = 99
				}
				if scalar == "float" {
					a := lpFloatLUSource(t, b, basis)
					lpStaticMust(t, a.Set(0, 2, 1))
					m, e := NewFloatLU(a, basis, s)
					lpStaticMust(t, e)
					defer m.Close()
					d, w := NewIndexedFloatVector(2), NewIndexedFloatVector(2)
					lpStaticMust(t, w.SetValue(9, 0))
					switch kind {
					case "closed":
						lpStaticMust(t, m.Close())
					case "clear":
						w.Index = append(w.Index, 2)
					case "extent":
						w.Data = nil
						w.Index = nil
					case "short-triangular":
						lpStaticMust(t, w.Resize(1))
					case "destination":
						lpStaticMust(t, d.Resize(241))
						d.Index = []uint32{241}
					}
					err := m.SolveBd(entering, d, w)
					if err == nil {
						t.Fatal("invalid column accepted")
					}
					if kind == "closed" {
						if err != ErrLUClosed || m.PrepareEntering(entering, w) != ErrLUClosed {
							t.Fatal(err)
						}
					} else if m.SolveBd(entering, d, w) != ErrLUStopped || m.PrepareEntering(entering, w) != ErrLUStopped {
						t.Fatal("failed column not stopped")
					}
					if kind == "column" && (w.Data[0] != 0 || len(w.Index) != 0) {
						t.Fatal("clear before column guard lost")
					}
					if kind == "short-triangular" && (len(d.Data) != 1 || len(d.Index) != 0 || d.Data[0] != 1) {
						t.Fatal("dense assignment/index-clear before solve failure lost", d)
					}
				} else {
					a := lpExactLUStatic(t, h, b, basis)
					v := h.pair("1", "9")
					h.must(a.Set(0, 2, &v.X))
					var r *RationalLU
					var p *PairLU
					var err error
					if scalar == "pair" {
						p, err = NewPairLU(a, basis, s)
					} else {
						r, err = NewRationalLU(a, basis, s)
					}
					h.must(err)
					d, w := NewIndexedRationalVector(h.m, 2), NewIndexedRationalVector(h.m, 2)
					defer func() {
						if p != nil {
							h.must(p.Close())
						} else {
							h.must(r.Close())
						}
						h.must(d.Close())
						h.must(w.Close())
					}()
					h.must(w.SetValue(&v.Y, 0))
					switch kind {
					case "closed":
						if p != nil {
							h.must(p.Close())
						} else {
							h.must(r.Close())
						}
					case "clear":
						w.Index = append(w.Index, 2)
					case "extent":
						h.must(w.Clear())
						h.must(w.Resize(0))
					case "short-triangular":
						h.must(w.Resize(1))
					case "destination":
						h.must(d.Resize(21))
						d.Index = []uint32{21}
					}
					if p != nil {
						err = p.SolveBd(entering, d, w)
					} else {
						err = r.SolveBd(entering, d, w)
					}
					if err == nil {
						t.Fatal("invalid exact column accepted")
					}
					if kind == "closed" {
						if err != ErrLUClosed {
							t.Fatal(err)
						}
					} else {
						if p != nil {
							if p.SolveBdFaster(entering, w) != ErrLUStopped {
								t.Fatal("stopped pair")
							}
						} else {
							if r.PrepareEntering(entering, w) != ErrLUStopped {
								t.Fatal("stopped scalar")
							}
						}
					}
					if kind == "column" {
						if !reflect.DeepEqual(w.Index, []uint32{}) {
							t.Fatal("clear before column guard lost", w.Index)
						}
						lpSparseRationalCheck(t, h, &w.Data[0], new(big.Rat))
					}
					if kind == "short-triangular" {
						if d.DataSize() != 1 || len(d.Index) != 0 {
							t.Fatal("exact dense assignment prefix", d.DataSize(), d.Index)
						}
						lpSparseRationalCheck(t, h, &d.Data[0], big.NewRat(1, 1))
					}
					if p != nil {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					h.must(d.Close())
					h.must(w.Close())
					lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
				}
			})
		}
	}
}

func TestLPLUColumnDenseAssignmentGuardAndCopyPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	foreign := newLPPairHarness(t)
	d, w := NewIndexedRationalVector(h.m, 3), NewIndexedRationalVector(h.m, 2)
	other := NewIndexedRationalVector(foreign.m, 1)
	defer func() { h.must(d.Close()); h.must(w.Close()); foreign.must(other.Close()) }()
	if d.assignDenseData(other) != ErrExactVectorOwner {
		t.Fatal("foreign assignment accepted")
	}
	for i := range w.Data {
		h.must(h.m.SetString(&w.Data[i], fmt.Sprint(i+2)))
	}
	d.Index = []uint32{2}
	h.must(d.assignDenseData(w))
	if d.DataSize() != 2 || !reflect.DeepEqual(d.Index, []uint32{2}) {
		t.Fatal("data assignment changed index before source algorithm")
	}
	lpExactLUCheckVector(t, h, d.Data, []*big.Rat{big.NewRat(2, 1), big.NewRat(3, 1)})
	h.must(d.Close())
	h.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPLUColumnDenseAssignmentLeavesStoppedCallerOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			v := h.pair("1"+strings.Repeat("0", 100)+"/7", "2"+strings.Repeat("0", 100)+"/13")
			d, w := NewIndexedRationalVector(h.m, 1), NewIndexedRationalVector(h.m, 1)
			defer func() { h.must(d.Close()); h.must(w.Close()) }()
			h.must(w.SetValue(&v.X, 0))
			h.budget.FailAfter(ordinal)
			err := w.Add(0, &v.Y)
			h.budget.FailAfter(-1)
			if w.calls.pending() {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				before := h.m.RawInvocationState()
				if d.assignDenseData(w) != ErrExactVectorPending || w.assignDenseData(d) != ErrExactVectorPending {
					t.Fatal("stopped assignment accepted")
				}
				if d.calls.pending() {
					t.Fatal("destination adopted caller receipt")
				}
				h.must(d.Close())
				if h.m.RawInvocationState() != before {
					t.Fatal("destination consumed caller receipt")
				}
			}
			h.must(d.Close())
			h.must(w.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual stopped vector invocation")
	}
}

func TestLPLUEnteringColumnTailReadAfterExtraction(t *testing.T) {
	a := NewFloatStaticMatrix(3, 4)
	for i, v := range []float64{1, 1, 2} {
		lpStaticMust(t, a.Set(uint32(i), uint32(i), v))
	}
	lpStaticMust(t, a.Set(0, 3, 1))
	m, err := NewFloatLU(a, []uint32{0, 1, 2}, lpFloatLUSettings(math.Inf(1)))
	lpStaticMust(t, err)
	defer m.Close()
	if len(m.Tail) != 1 {
		t.Fatal("fixture must reach a nontrivial eta")
	}
	w := NewIndexedFloatVector(1)
	lpStaticMust(t, w.SetValue(99, 0))
	if err = m.PrepareEntering(3, w); err != ErrBounds {
		t.Fatal("expected reached eta read past short caller", err)
	}
	if !reflect.DeepEqual(w.Data, []float64{1}) || !reflect.DeepEqual(w.Index, []uint32{0}) {
		t.Fatal("clear/extraction prefix lost", w)
	}
	if m.PrepareEntering(3, NewIndexedFloatVector(3)) != ErrLUStopped {
		t.Fatal("tail read failure not stopped")
	}
}
