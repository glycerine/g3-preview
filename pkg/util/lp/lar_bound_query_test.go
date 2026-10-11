package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLARBoundQueryActualConstraintIntersections(t *testing.T) {
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		_, err = s.AddVar(17)
		h.must(err)
		oracle := lpBoundOracle{lowWitness: InvalidLARIndex, upperWitness: InvalidLARIndex}
		for step := 0; step < 20; step++ {
			kind := []LConstraintKind{LConstraintGE, LConstraintGT, LConstraintLE, LConstraintLT, LConstraintEQ}[(world+step*3)%5]
			rhs := big.NewRat(int64((world*7+step*11)%31-15), int64(1+(world+step)%7))
			// Query feasible intersections. Contradictory boxed updates have a
			// separate source debug assertion, exercised by the bounds tests.
			candidate := oracle
			if conflict, _ := candidate.intersect(kind, rhs, 0); conflict {
				break
			}
			ci, err := s.AddVarBound(0, kind, &h.pair(rhs.RatString(), "0").X)
			if err != nil {
				t.Fatalf("world=%d step=%d kind=%v rhs=%s: %v", world, step, kind, rhs, err)
			}
			oracle.intersect(kind, rhs, ci)
			for _, upper := range []bool{false, true} {
				witness := uint32(97)
				strict := true
				out := h.pair("101", "0")
				var found bool
				if upper {
					found, err = s.HasUpperBound(0, &witness, &out.X, &strict)
				} else {
					found, err = s.HasLowerBound(0, &witness, &out.X, &strict)
				}
				h.must(err)
				value, wantWitness, wantStrict := oracle.low, oracle.lowWitness, oracle.lowStrict
				if upper {
					value, wantWitness, wantStrict = oracle.upper, oracle.upperWitness, oracle.upperStrict
				}
				if found != (value != nil) || witness != wantWitness {
					t.Fatal(world, step, upper, found, witness, wantWitness)
				}
				if found {
					lpSparseRationalCheck(t, h, &out.X, value)
					if strict != wantStrict {
						t.Fatal(world, step, upper, strict, wantStrict)
					}
				} else {
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(101, 1))
					if !strict {
						t.Fatal("missing bound changed strictness")
					}
				}
			}
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARBoundQueryPublicationAndTermIndices(t *testing.T) {
	for _, upper := range []bool{false, true} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		_, err = s.AddVar(17)
		h.must(err)
		witness := uint32(97)
		strict := true
		out := h.pair("101", "0")
		query := func(j uint32) (bool, error) {
			if upper {
				return s.HasUpperBound(j, &witness, &out.X, &strict)
			}
			return s.HasLowerBound(j, &witness, &out.X, &strict)
		}
		found, err := query(99)
		h.must(err)
		if found || witness != 97 || !strict {
			t.Fatal("missing variable prefix")
		}
		lpSparseRationalCheck(t, h, &out.X, big.NewRat(101, 1))
		found, err = query(0)
		h.must(err)
		if found || witness != InvalidLARIndex || !strict {
			t.Fatal("missing witness prefix")
		}
		lpSparseRationalCheck(t, h, &out.X, big.NewRat(101, 1))
		term, err := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: h.pair("2", "0").X}}, &h.pair("5", "0").X)
		h.must(err)
		kind := LConstraintGE
		if upper {
			kind = LConstraintLE
		}
		ci, err := s.AddVarBound(term, kind, &h.pair("8", "0").X)
		h.must(err)
		witness = 97
		found, err = query(term)
		h.must(err)
		if found || witness != 97 || !strict {
			t.Fatal("external term was translated")
		}
		found, err = query(1)
		h.must(err)
		if !found || witness != ci || strict {
			t.Fatal("actual internal term column", found, witness, strict)
		}
		lpSparseRationalCheck(t, h, &out.X, big.NewRat(3, 1))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARBoundQueryWitnessAndEpsilonSigns(t *testing.T) {
	for _, upper := range []bool{false, true} {
		for _, sign := range []string{"-3/7", "0", "5/11"} {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(17)
			h.must(err)
			w, err := s.Bounds.Witnesses.At(0)
			h.must(err)
			w.LowBoundWitness, w.UpperBoundWitness = 123, 127
			h.must(s.Bounds.Witnesses.Set(0, w))
			// A query follows witness presence and epsilon sign, independent of the
			// column type and constraint count. Source performs no constraint lookup.
			h.must(s.Bounds.Low.Set(0, h.pair("13/17", sign)))
			h.must(s.Bounds.Upper.Set(0, h.pair("19/23", sign)))
			witness := uint32(97)
			strict := false
			out := h.pair("101", "0")
			var found bool
			if upper {
				found, err = s.HasUpperBound(0, &witness, &out.X, &strict)
			} else {
				found, err = s.HasLowerBound(0, &witness, &out.X, &strict)
			}
			h.must(err)
			wantWitness, wantValue, wantStrict := uint32(123), big.NewRat(13, 17), sign == "5/11"
			if upper {
				wantWitness, wantValue, wantStrict = 127, big.NewRat(19, 23), sign == "-3/7"
			}
			if !found || witness != wantWitness || strict != wantStrict {
				t.Fatal(upper, sign, found, witness, strict)
			}
			lpSparseRationalCheck(t, h, &out.X, wantValue)
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARBoundQueryPhysicalOutputCopies(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, upper := range []bool{false, true} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 200 && !done; ordinal++ {
				passed := t.Run(fmt.Sprintf("%v/%v/%d", direct, upper, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, err := NewLARSolver(h.m)
					h.must(err)
					_, err = s.AddVar(17)
					h.must(err)
					kind := LConstraintGT
					if upper {
						kind = LConstraintLT
					}
					large := strings.Repeat("1234567890", 20) + "7/13"
					ci, err := s.AddVarBound(0, kind, &h.pair(large, "0").X)
					h.must(err)
					witness := uint32(97)
					strict := false
					out := h.pair("101", "0")
					h.budget.FailAfter(ordinal)
					var found bool
					if upper {
						found, err = s.HasUpperBound(0, &witness, &out.X, &strict)
					} else {
						found, err = s.HasLowerBound(0, &witness, &out.X, &strict)
					}
					h.budget.FailAfter(-1)
					if witness != ci {
						t.Fatal("witness must publish before numerical copy", witness, ci)
					}
					if err == nil {
						done = true
						if !found || !strict {
							t.Fatal(found, strict)
						}
						lpSparseRationalCheck(t, h, &out.X, mustLPRat(t, large))
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) || found || strict {
							t.Fatal("failed copy prefix", found, strict, err)
						}
						if _, e := s.HasLowerBound(0, &witness, &out.X, &strict); e != ErrExactVectorPending {
							t.Fatal("retry after unfinished copy", e)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if !passed {
					t.FailNow()
				}
			}
			if !done || failures == 0 && direct {
				t.Fatal("incomplete physical campaign", direct, upper, done, failures)
			}
		}
	}
}

func TestLARBoundQueryReachedGuards(t *testing.T) {
	for _, upper := range []bool{false, true} {
		for _, phase := range []string{"stopped", "nil-witness", "nil-value", "nil-strict", "absent-all-nil", "missing-value-nil", "short-bound"} {
			t.Run(fmt.Sprintf("%v/%s", upper, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, err := NewLARSolver(h.m)
				h.must(err)
				_, err = s.AddVar(17)
				h.must(err)
				kind := LConstraintGE
				if upper {
					kind = LConstraintLE
				}
				ci, err := s.AddVarBound(0, kind, &h.pair("3/7", "0").X)
				h.must(err)
				witness := uint32(97)
				strict := true
				out := h.pair("101", "0")
				wp, vp, sp := &witness, &out.X, &strict
				j := uint32(0)
				var shortened *StackedPairVector
				var savedBounds []RationalPair
				want := error(ErrBounds)
				switch phase {
				case "stopped":
					s.stopped = true
					want = ErrExactVectorPending
				case "nil-witness":
					wp = nil
				case "nil-value":
					vp = nil
				case "nil-strict":
					sp = nil
				case "absent-all-nil":
					j = 99
					wp, vp, sp = nil, nil, nil
					want = nil
				case "missing-value-nil":
					w, err := s.Bounds.Witnesses.At(0)
					h.must(err)
					w.LowBoundWitness, w.UpperBoundWitness = InvalidLARIndex, InvalidLARIndex
					h.must(s.Bounds.Witnesses.Set(0, w))
					vp, sp = nil, nil
					want = nil
				case "short-bound":
					bounds := s.Bounds.Low
					if upper {
						bounds = s.Bounds.Upper
					}
					shortened, savedBounds = bounds, bounds.values
					bounds.values = nil
				}
				var found bool
				if upper {
					found, err = s.HasUpperBound(j, wp, vp, sp)
				} else {
					found, err = s.HasLowerBound(j, wp, vp, sp)
				}
				if found || err != want {
					t.Fatal(found, err, want)
				}
				if phase == "nil-value" || phase == "nil-strict" || phase == "short-bound" {
					if witness != ci {
						t.Fatal("lost reached witness", witness, ci)
					}
				}
				if phase == "nil-strict" {
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(3, 7))
				}
				if shortened != nil {
					shortened.values = savedBounds
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARBoundQueryActualNestedScopes(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	_, err = s.AddVar(17)
	h.must(err)
	lower, err := s.AddVarBound(0, LConstraintGE, &h.pair("0", "0").X)
	h.must(err)
	_, err = s.FindFeasibleSolution()
	h.must(err)
	h.must(s.Push())
	nestedLower, err := s.AddVarBound(0, LConstraintGT, &h.pair("1", "0").X)
	h.must(err)
	nestedUpper, err := s.AddVarBound(0, LConstraintLE, &h.pair("5", "0").X)
	h.must(err)
	_, err = s.FindFeasibleSolution()
	h.must(err)
	h.must(s.Push())
	innerLower, err := s.AddVarBound(0, LConstraintGE, &h.pair("3", "0").X)
	h.must(err)
	innerUpper, err := s.AddVarBound(0, LConstraintLT, &h.pair("4", "0").X)
	h.must(err)
	_, err = s.FindFeasibleSolution()
	h.must(err)
	for depth := 2; depth >= 0; depth-- {
		for _, upper := range []bool{false, true} {
			witness := uint32(97)
			strict := true
			out := h.pair("101", "0")
			var found bool
			if upper {
				found, err = s.HasUpperBound(0, &witness, &out.X, &strict)
			} else {
				found, err = s.HasLowerBound(0, &witness, &out.X, &strict)
			}
			h.must(err)
			wantWitness, wantValue, wantStrict := lower, int64(0), false
			if depth == 2 {
				wantWitness, wantValue = innerLower, 3
				if upper {
					wantWitness, wantValue, wantStrict = innerUpper, 4, true
				}
			} else if depth == 1 {
				wantWitness, wantValue, wantStrict = nestedLower, 1, true
				if upper {
					wantWitness, wantValue, wantStrict = nestedUpper, 5, false
				}
			} else if upper {
				wantWitness = InvalidLARIndex
			}
			if found != (wantWitness != InvalidLARIndex) || witness != wantWitness {
				t.Fatal(depth, upper, found, witness, wantWitness)
			}
			if found {
				if strict != wantStrict {
					t.Fatal(depth, upper, strict)
				}
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(wantValue, 1))
			}
		}
		if depth > 0 {
			h.must(s.Pop(1))
		}
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARBoundQueryPreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(17)
			h.must(err)
			large := strings.Repeat("1234567890", 20) + "7/13"
			_, err = s.AddVarBound(0, LConstraintGT, &h.pair(large, "0").X)
			h.must(err)
			witness := uint32(97)
			strict := false
			out := h.pair("101", "0")
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			b, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, b, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = s.HasLowerBound(0, &witness, &out.X, &strict)
				if s.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed older comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}
