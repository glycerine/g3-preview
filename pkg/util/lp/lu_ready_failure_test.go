package lp

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestLPLUPreparedSolvesFailurePrefixes(t *testing.T) {
	for _, kind := range []string{"float", "rational", "pair"} {
		for _, op := range []string{"X", "T", "indexed"} {
			for _, phase := range []string{"short", "R bounds", "closed", "stopped"} {
				t.Run(kind+"/"+op+"/"+phase, func(t *testing.T) {
					h := newLPPairHarness(t)
					basis := []uint32{0, 1}
					a := lpExactLUStatic(t, h, lpExactIdentity(2), basis)
					fa := lpFloatLUSource(t, lpExactIdentity(2), basis)
					s := lpFloatLUSettings(math.Inf(1))
					f, err := NewFloatLU(fa, basis, s)
					lpStaticMust(t, err)
					defer f.Close()
					r, err := NewRationalLU(a, basis, s)
					h.must(err)
					defer func() { h.must(r.Close()) }()
					p, err := NewPairLU(a, basis, s)
					h.must(err)
					defer func() { h.must(p.Close()) }()
					fi := NewIndexedFloatVector(2)
					lpStaticMust(t, fi.SetValue(2, 0))
					lpStaticMust(t, fi.SetValue(3, 1))
					vi := lpExactLUVector(h, []*big.Rat{big.NewRat(2, 1), big.NewRat(3, 1)})
					defer func() { h.must(vi.Close()) }()
					pairs := NewIndexedPairVector(h.m, 2)
					defer func() { h.must(pairs.Close()) }()
					for i := range pairs.Data {
						h.must(h.m.Copy(&pairs.Data[i].X, &vi.Data[i]))
					}
					want := ErrBounds
					index := []uint32{1}
					switch phase {
					case "short":
						fi.Data = fi.Data[:1]
						vi.Data = vi.Data[:1]
						pairs.Data = pairs.Data[:1]
					case "R bounds":
						f.R.values[0] = 2
						r.R.values[0] = 2
						p.R.values[0] = 2
					case "closed":
						lpStaticMust(t, f.Close())
						h.must(r.Close())
						h.must(p.Close())
						want = ErrLUClosed
					case "stopped":
						if kind == "float" {
							err = f.SolveByWhenReadyForX(nil)
						} else if kind == "rational" {
							err = r.SolveByWhenReadyForX(nil)
						} else {
							err = p.SolveByWhenReadyForX(nil)
						}
						if err != ErrBounds {
							t.Fatal("actual stopped solve", err)
						}
						want = ErrLUStopped
					}
					if kind == "float" {
						switch op {
						case "X":
							err = f.SolveByWhenReadyForX(fi.Data)
						case "T":
							err = f.SolveByWhenReadyForT(fi.Data, &index)
						case "indexed":
							err = f.SolveByForTIndexedOnly(fi, nil)
						}
					} else if kind == "rational" {
						switch op {
						case "X":
							err = r.SolveByWhenReadyForX(vi.Data)
						case "T":
							err = r.SolveByWhenReadyForT(vi.Data, &index)
						case "indexed":
							err = r.SolveByForTIndexedOnly(vi, nil)
						}
					} else {
						switch op {
						case "X":
							err = p.SolveByWhenReadyForX(pairs.Data)
						case "T":
							err = p.SolveByWhenReadyForT(vi.Data, &index)
						case "indexed":
							err = p.SolveByForTIndexedOnly(vi, nil)
						}
					}
					if err != want {
						t.Fatal(err, want)
					}
					if op == "T" && (len(index) != 1 || index[0] != 1) {
						t.Fatal("index appended before successful permutation", index)
					}
				})
			}
		}
	}
}

func TestLPExactLUPreparedSolvesAllocationCensus(t *testing.T) {
	for _, pair := range []bool{false, true} {
		for _, op := range []string{"X", "T", "indexed"} {
			for _, direct := range []bool{false, true} {
				t.Run(fmt.Sprint(pair, "/", op, "/", direct), func(t *testing.T) {
					failures, success := 0, false
					for ordinal := int64(0); ordinal < 600 && !success; ordinal++ {
						t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
							var h *lpPairHarness
							if direct {
								h = lpNewDirectSparseHarness(t)
							} else {
								h = newLPPairHarness(t)
							}
							a := lpExactLUStatic(t, h, lpExactIdentity(2), []uint32{0, 1})
							s := lpFloatLUSettings(math.Inf(1))
							var r *RationalLU
							var p *PairLU
							var state *exactLUState
							var err error
							if pair {
								p, err = NewPairLU(a, []uint32{0, 1}, s)
								state = p.exactLUState
							} else {
								r, err = NewRationalLU(a, []uint32{0, 1}, s)
								state = r.exactLUState
							}
							h.must(err)
							v := h.pair("1"+strings.Repeat("0", 31)+"1/7", "2"+strings.Repeat("0", 47)+"3/13")
							// Directly install a unit upper triangle. Its independently known solution
							// for rhs=(0,z) is (-c*z,z), in the owner's retained R coordinates.
							h.must(state.U.AddNewElement(0, 1, &v.X))
							zero := new(big.Rat)
							z := mustLPRat(t, "3"+strings.Repeat("0", 39)+"1/17")
							rhs := []*big.Rat{zero, z}
							vi := lpExactLUVector(h, rhs)
							pairs := NewIndexedPairVector(h.m, 2)
							for i := range pairs.Data {
								h.must(h.m.Copy(&pairs.Data[i].X, &vi.Data[i]))
								h.must(h.m.Copy(&pairs.Data[i].Y, &vi.Data[i]))
							}
							defer func() {
								if pair {
									h.must(p.Close())
								} else {
									h.must(r.Close())
								}
								h.must(vi.Close())
								h.must(pairs.Close())
							}()
							index := []uint32{0}
							h.budget.FailAfter(ordinal)
							if pair {
								switch op {
								case "X":
									err = p.SolveByWhenReadyForX(pairs.Data)
								case "T":
									err = p.SolveByWhenReadyForT(vi.Data, &index)
								case "indexed":
									err = p.SolveByForTIndexedOnly(vi, nil)
								}
							} else {
								switch op {
								case "X":
									err = r.SolveByWhenReadyForX(vi.Data)
								case "T":
									err = r.SolveByWhenReadyForT(vi.Data, &index)
								case "indexed":
									err = r.SolveByForTIndexedOnly(vi, nil)
								}
							}
							h.budget.FailAfter(-1)
							if err == nil {
								success = true
								coefficient := mustLPRat(t, "1"+strings.Repeat("0", 31)+"1/7")
								internal := []*big.Rat{new(big.Rat).Neg(new(big.Rat).Mul(coefficient, z)), z}
								want := make([]*big.Rat, 2)
								var mapping []uint32
								if pair {
									mapping = p.R.values
								} else {
									mapping = r.R.values
								}
								for i, j := range mapping {
									want[j] = internal[i]
								}
								if pair && op == "X" {
									for i := range want {
										h.check(&pairs.Data[i], want[i], want[i])
									}
								} else {
									lpExactLUCheckVector(t, h, vi.Data, want)
								}
							} else {
								failures++
								if !lpExactPriorityPhysicalRefusal(err) || !state.stopped {
									t.Fatal(err, state.stopped)
								}
							}
							if pair {
								h.must(p.Close())
							} else {
								h.must(r.Close())
							}
							cells := lpExactLUStaticCells(a)
							for i := range vi.Data {
								cells = append(cells, &vi.Data[i])
							}
							for i := range pairs.Data {
								cells = append(cells, &pairs.Data[i].X, &pairs.Data[i].Y)
							}
							lpExactVectorNamedCensus(t, h, cells)
							h.must(vi.Close())
							h.must(pairs.Close())
							lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
						})
					}
					if !success || (direct && failures == 0) {
						t.Fatal("incomplete prepared solve refusal walk", success, failures)
					}
					t.Logf("physical refusals=%d", failures)
				})
			}
		}
	}
}

func TestLPExactLUPreparedIndexedCallerOwnership(t *testing.T) {
	for _, pair := range []bool{false, true} {
		reached := false
		for ordinal := int64(0); ordinal < 80 && !reached; ordinal++ {
			t.Run(fmt.Sprint(pair, "/", ordinal), func(t *testing.T) {
				h := lpNewDirectSparseHarness(t)
				a := lpExactLUStatic(t, h, lpExactIdentity(1), []uint32{0})
				s := lpFloatLUSettings(math.Inf(1))
				var r *RationalLU
				var p *PairLU
				var state *exactLUState
				var err error
				if pair {
					p, err = NewPairLU(a, []uint32{0}, s)
					state = p.exactLUState
				} else {
					r, err = NewRationalLU(a, []uint32{0}, s)
					state = r.exactLUState
				}
				h.must(err)
				v := h.pair("1"+strings.Repeat("0", 100)+"/7", "2"+strings.Repeat("0", 100)+"/13")
				y := NewIndexedRationalVector(h.m, 1)
				h.must(y.SetValue(&v.X, 0))
				defer func() {
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					h.must(y.Close())
				}()
				h.budget.FailAfter(ordinal)
				err = y.Add(0, &v.Y)
				h.budget.FailAfter(-1)
				if y.calls.pending() {
					reached = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					before := h.m.RawInvocationState()
					if pair {
						err = p.SolveByForTIndexedOnly(y, nil)
					} else {
						err = r.SolveByForTIndexedOnly(y, nil)
					}
					if err != ErrExactVectorPending || state.U.work != nil {
						t.Fatal("caller stop not checked before U work", err)
					}
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					if h.m.RawInvocationState() != before {
						t.Fatal("LU consumed caller receipt")
					}
				}
				if pair {
					h.must(p.Close())
				} else {
					h.must(r.Close())
				}
				h.must(y.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
		if !reached {
			t.Fatal("no actual caller-owned numerical refusal")
		}
	}
	for _, pair := range []bool{false, true} {
		h := newLPPairHarness(t)
		a := lpExactLUStatic(t, h, lpExactIdentity(1), []uint32{0})
		foreign := &IndexedRationalVector{}
		if pair {
			m, e := NewPairLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
			h.must(e)
			if e = m.SolveByForTIndexedOnly(foreign, nil); e != ErrExactVectorOwner || m.U.work != nil {
				t.Fatal(e)
			}
			h.must(m.Close())
		} else {
			m, e := NewRationalLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
			h.must(e)
			if e = m.SolveByForTIndexedOnly(foreign, nil); e != ErrExactVectorOwner || m.U.work != nil {
				t.Fatal(e)
			}
			h.must(m.Close())
		}
		lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
	}
}
