package lp

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactLURowRefinementAllocationCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"dense", "indexed dense", "indexed sparse"} {
		for _, pair := range []bool{false, true} {
			for _, direct := range []bool{false, true} {
				t.Run(fmt.Sprint(op, "/", pair, "/", direct), func(t *testing.T) {
					failures, successes := 0, 0
					for ordinal := int64(0); ordinal < 800; ordinal++ {
						done := false
						t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
							var h *lpPairHarness
							if direct {
								h = lpNewDirectSparseHarness(t)
							} else {
								h = newLPPairHarness(t)
							}
							x := mustLPRat(t, "1"+strings.Repeat("0", 31)+"1/7")
							z := mustLPRat(t, "2"+strings.Repeat("0", 47)+"3/13")
							b := [][]*big.Rat{{x, z}, {z, x}}
							columns := uint32(60)
							if op == "indexed sparse" {
								columns = 61
							}
							a := NewRationalStaticMatrix(h.m, 2, columns)
							defer func() { h.must(a.Close()) }()
							v := h.pair("0", "0")
							for i := range b {
								for j := range b[i] {
									h.must(h.m.SetString(&v.X, b[i][j].RatString()))
									h.must(a.Set(uint32(i), uint32(j), &v.X))
								}
							}
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
							want := []*big.Rat{mustLPRat(t, "1"+strings.Repeat("0", 39)+"1/17"), mustLPRat(t, "-2"+strings.Repeat("0", 39)+"3/19")}
							y := lpExactLUVector(h, lpExactMatrixVector(b, want, true))
							defer func() {
								if r != nil {
									h.must(r.Close())
								}
								if p != nil {
									h.must(p.Close())
								}
								h.must(y.Close())
							}()
							h.budget.FailAfter(ordinal)
							if pair {
								if op == "dense" {
									err = p.SolveYBWithErrorCheck(y.Data, nil)
								} else {
									err = p.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
								}
							} else {
								if op == "dense" {
									err = r.SolveYBWithErrorCheck(y.Data, nil)
								} else {
									err = r.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
								}
							}
							h.budget.FailAfter(-1)
							if err == nil {
								done = true
								successes++
								lpExactLUCheckVector(t, h, y.Data, want)
							} else {
								failures++
								if !lpExactPriorityPhysicalRefusal(err) {
									t.Fatal(err)
								}
								if !state.stopped {
									t.Fatal("failed refinement did not stop owner")
								}
								if pair {
									err = p.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
								} else {
									err = r.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
								}
								if err != ErrLUStopped {
									t.Fatal(err)
								}
							}
							if r != nil {
								h.must(r.Close())
							}
							if p != nil {
								h.must(p.Close())
							}
							cells := lpExactLUStaticCells(a)
							for i := range y.Data {
								cells = append(cells, &y.Data[i])
							}
							lpExactVectorNamedCensus(t, h, cells)
							h.must(y.Close())
							lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
							lpExactLUCheckSource(t, h, a, b, []uint32{0, 1})
						})
						if done {
							break
						}
					}
					if successes != 1 || (direct && failures == 0) {
						t.Fatal("incomplete exact refinement failure walk", failures, successes)
					}
					t.Logf("physical refusals=%d", failures)
				})
			}
		}
	}
}

func TestLPLURowRefinementOwnerGuards(t *testing.T) {
	for _, pair := range []bool{false, true} {
		for _, phase := range []string{"foreign", "short", "empty short", "closed"} {
			t.Run(fmt.Sprint(pair, "/", phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				a := lpExactLUStatic(t, h, lpExactIdentity(2), []uint32{0, 1})
				var r *RationalLU
				var p *PairLU
				var err error
				if pair {
					p, err = NewPairLU(a, []uint32{0, 1}, lpFloatLUSettings(math.Inf(1)))
				} else {
					r, err = NewRationalLU(a, []uint32{0, 1}, lpFloatLUSettings(math.Inf(1)))
				}
				h.must(err)
				y := NewIndexedRationalVector(h.m, 0)
				if phase == "short" {
					y.Index = []uint32{0}
				}
				input := y
				if phase == "foreign" {
					input = NewIndexedRationalVector(nil, 0)
				}
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(y.Close())
				}()
				if phase == "closed" {
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
				}
				if pair {
					err = p.SolveYBWithErrorCheckIndexed(input, nil, nil, nil)
				} else {
					err = r.SolveYBWithErrorCheckIndexed(input, nil, nil, nil)
				}
				want := ErrBounds
				if phase == "short" && sourceDebugEnabled {
					want = ErrAssertion
				}
				if phase == "empty short" {
					want = nil
				}
				if phase == "foreign" {
					want = ErrExactVectorOwner
				}
				if phase == "closed" {
					want = ErrLUClosed
				}
				if err != want {
					t.Fatal(err, want)
				}
			})
		}
	}
	for _, indexed := range []bool{false, true} {
		a := NewFloatStaticMatrix(2, 2)
		lpStaticMust(t, a.Set(0, 0, 1))
		lpStaticMust(t, a.Set(1, 1, 1))
		m, err := NewFloatLU(a, []uint32{0, 1}, lpFloatLUSettings(math.Inf(1)))
		lpStaticMust(t, err)
		lpStaticMust(t, m.Close())
		if indexed {
			err = m.SolveYBWithErrorCheckIndexed(NewIndexedFloatVector(2), nil, nil, nil)
		} else {
			err = m.SolveYBWithErrorCheck(make([]float64, 2), nil)
		}
		if err != ErrLUClosed {
			t.Fatal(err)
		}
	}
}

func TestLPFloatLURowRefinementBoundsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"basis", "dot column", "descending residual", "residual extent", "dense correction", "marker clear", "correction marker clear", "row extent", "input extent", "heading", "heading value", "copy extent", "copy index", "orphan copy extent", "old solution index", "delta extent", "public dense solve", "public dense delegation", "public residual", "public sparse solve", "public sparse permutation", "public sparse to dense residual", "public indexed residual"} {
		t.Run(phase, func(t *testing.T) {
			a := NewFloatStaticMatrix(2, 241)
			lpStaticMust(t, a.Set(0, 0, 1))
			lpStaticMust(t, a.Set(1, 1, 1))
			s := lpFloatLUSettings(math.Inf(1))
			m, err := NewFloatLU(a, []uint32{0, 1}, s)
			lpStaticMust(t, err)
			defer m.Close()
			y := NewIndexedFloatVector(2)
			lpStaticMust(t, y.SetValue(2, 0))
			lpStaticMust(t, y.SetValue(3, 1))
			m.yCopy.Data = []float64{9, 7}
			m.yCopy.Index = []uint32{0, 1}
			heading := lpLUHeading(241, []uint32{0, 1})
			switch phase {
			case "basis":
				err = m.findRowError(m.yCopy.Data, y.Data, nil)
			case "dot column":
				err = m.findRowError(m.yCopy.Data, y.Data, []uint32{0, 241})
			case "descending residual":
				err = m.findRowError(m.yCopy.Data, y.Data, []uint32{241, 1})
				if m.yCopy.Data[0] != 9 || m.yCopy.Data[1] != 4 {
					t.Fatal("descending residual prefix")
				}
			case "residual extent":
				err = m.findRowError(nil, y.Data, []uint32{0, 1})
			case "dense correction":
				err = addFloatRowCorrection(nil, y.Data)
			case "marker clear":
				m.ii.Index = []uint32{2}
				err = m.findIndexedRowError(y, heading, s)
			case "correction marker clear":
				m.ii.Index = []uint32{2}
				err = m.addIndexedRowCorrection(y)
			case "row extent":
				y.Index = []uint32{2}
				err = m.findIndexedRowError(y, heading, s)
			case "input extent":
				y.Data = y.Data[:1]
				err = m.findIndexedRowError(y, heading, s)
			case "heading":
				err = m.findIndexedRowError(y, nil, s)
			case "heading value":
				heading[1] = 2
				err = m.findIndexedRowError(y, heading, s)
			case "copy extent":
				m.yCopy.Data = m.yCopy.Data[:1]
				err = m.findIndexedRowError(y, heading, s)
			case "copy index":
				m.yCopy.Index = []uint32{2}
				err = m.findIndexedRowError(y, heading, s)
			case "orphan copy extent":
				y.Index = nil
				m.yCopy.Data = m.yCopy.Data[:1]
				err = m.findIndexedRowError(y, heading, s)
			case "old solution index":
				y.Index = []uint32{2}
				err = m.addIndexedRowCorrection(y)
			case "delta extent":
				m.yCopy.Data = nil
				err = m.addIndexedRowCorrection(y)
			case "public dense solve":
				err = m.SolveYBWithErrorCheck(nil, []uint32{0, 1})
			case "public dense delegation":
				a.Columns = a.Columns[:1]
				err = m.SolveYBWithErrorCheckIndexed(y, heading, nil, s)
			case "public residual":
				err = m.SolveYBWithErrorCheck(y.Data, nil)
				if len(m.yCopy.Data) != 2 || m.yCopy.Data[0] != 2 || m.yCopy.Data[1] != 3 {
					t.Fatal("saved RHS prefix")
				}
			case "public sparse solve":
				y.Index = []uint32{2}
				err = m.SolveYBWithErrorCheckIndexed(y, heading, []uint32{0, 1}, s)
			case "public sparse permutation":
				m.R.reverse[0] = 2
				err = m.SolveYBWithErrorCheckIndexed(y, heading, []uint32{0, 1}, s)
			case "public sparse to dense residual":
				a.Columns = a.Columns[:121]
				lpStaticMust(t, a.Set(0, 1, 0.5))
				lpStaticMust(t, m.Close())
				m, err = NewFloatLU(a, []uint32{0, 1}, s)
				lpStaticMust(t, err)
				defer m.Close()
				y.Data[1] = 0
				y.Index = []uint32{0}
				err = m.SolveYBWithErrorCheckIndexed(y, heading, nil, s)
			case "public indexed residual":
				err = m.SolveYBWithErrorCheckIndexed(y, nil, []uint32{0, 1}, s)
			}
			want := ErrBounds
			if (phase == "dot column" || phase == "descending residual") && sourceDebugEnabled {
				want = ErrAssertion
			}
			if phase == "public dense solve" && sourceDebugEnabled {
				want = ErrAssertion
			}
			if err != want {
				t.Fatal(phase, err, want)
			}
			if len(phase) >= 7 && phase[:7] == "public " && !m.stopped {
				t.Fatal("public failure did not stop")
			}
		})
	}
}

func TestLPExactLURowRefinementLeavesStoppedCallerOwned(t *testing.T) {
	for _, pair := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
			t.Run(fmt.Sprint(pair, "/", ordinal), func(t *testing.T) {
				h := lpNewDirectSparseHarness(t)
				a := lpExactLUStatic(t, h, lpExactIdentity(1), []uint32{0})
				v := h.pair("1"+strings.Repeat("0", 100)+"/7", "2"+strings.Repeat("0", 100)+"/13")
				var r *RationalLU
				var p *PairLU
				var state *exactLUState
				var err error
				if pair {
					p, err = NewPairLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
					state = p.exactLUState
				} else {
					r, err = NewRationalLU(a, []uint32{0}, lpFloatLUSettings(math.Inf(1)))
					state = r.exactLUState
				}
				h.must(err)
				y := NewIndexedRationalVector(h.m, 1)
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(y.Close())
				}()
				h.must(y.SetValue(&v.X, 0))
				h.budget.FailAfter(ordinal)
				err = y.Add(0, &v.Y)
				h.budget.FailAfter(-1)
				if y.calls.pending() {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					before := h.m.RawInvocationState()
					if pair {
						err = p.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
					} else {
						err = r.SolveYBWithErrorCheckIndexed(y, nil, nil, nil)
					}
					if err != ErrExactVectorPending || state.yCopy.DataSize() != 0 {
						t.Fatal("stopped caller accepted", err)
					}
					if pair {
						h.must(p.Close())
					} else {
						h.must(r.Close())
					}
					if h.m.RawInvocationState() != before {
						t.Fatal("refinement consumed caller receipt")
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
		if !matched {
			t.Fatal("no real caller refusal")
		}
	}
}
