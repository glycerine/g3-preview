// Source: src/util/lp/lar_solver.h:433-454,851-869.
package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func lpLARFullSolveFixture(t *testing.T, h *lpPairHarness, strategy SimplexStrategy, world int, impossible, strict bool) (*LARSolver, *big.Rat) {
	return lpLARFullSolveNumericFixture(t, h, strategy, world, impossible, strict, false)
}

func lpLARFullSolveNumericFixture(t *testing.T, h *lpPairHarness, strategy SimplexStrategy, world int, impossible, strict, largeCoefficient bool) (*LARSolver, *big.Rat) {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	s.Settings.SimplexStrategy = strategy
	for _, ext := range []uint32{17, 31} {
		_, err = s.AddVar(ext)
		h.must(err)
	}
	a := big.NewRat(int64(1+world%7), int64(1+world%5))
	if largeCoefficient {
		a = mustLPRat(t, strings.Repeat("1234567890", 20)+"7/13")
	}
	b := big.NewRat(int64(1+world%11), int64(1+world%3))
	constant := big.NewRat(int64(world-10), 13)
	input := NewSparseRationalVector(h.m)
	h.must(input.Append(0, &h.pair(a.RatString(), "0").X))
	h.must(input.Append(1, &h.pair(b.RatString(), "0").X))
	term, err := s.AddTerm(input.Data, &h.pair(constant.RatString(), "0").X)
	h.must(err)
	h.must(input.Close())
	low := big.NewRat(int64(1+world%9), 7)
	cap := new(big.Rat).Add(new(big.Rat).Mul(a, low), constant)
	if impossible {
		cap.Sub(cap, big.NewRat(1, 17))
	} else {
		cap.Add(cap, big.NewRat(1, 17))
	}
	kind := LConstraintGE
	if strict {
		kind = LConstraintGT
	}
	_, err = s.AddVarBound(0, kind, &h.pair(low.RatString(), "0").X)
	h.must(err)
	_, err = s.AddVarBound(1, LConstraintGE, &h.pair("0", "0").X)
	h.must(err)
	_, err = s.AddVarBound(term, LConstraintLE, &h.pair(cap.RatString(), "0").X)
	h.must(err)
	return s, low
}

func TestLARFullSolveIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts, StrategyLU} {
		for world := 0; world < 30; world++ {
			for _, impossible := range []bool{false, true} {
				for _, strict := range []bool{false, true} {
					t.Run(fmt.Sprintf("%d/%d/%v/%v", strategy, world, impossible, strict), func(t *testing.T) {
						h := newLPPairHarness(t)
						s, low := lpLARFullSolveFixture(t, h, strategy, world, impossible, strict)
						status, err := s.FindFeasibleSolution()
						h.must(err)
						want := StatusOptimal
						if impossible {
							want = StatusInfeasible
						}
						if status != want || s.Bounds.Status.Value != want || !s.Core.RPrimal.LookForFeasibleSolutionOnly || !s.Bounds.ChangedColumns.Empty() {
							t.Fatal("source status and changed-column cleanup", status, want)
						}
						if impossible {
							if len(s.Core.Infeasibility.Data) == 0 {
								t.Fatal("missing real infeasibility evidence")
							}
						} else {
							model := NewLARRationalModel(h.m)
							h.must(s.GetModel(model))
							text, e := h.m.String(model.Values[0])
							h.must(e)
							v := mustLPRat(t, text)
							if v.Cmp(low) < 0 || strict && v.Cmp(low) == 0 {
								t.Fatal("independent strict lower bound", v, low)
							}
							text, e = h.m.String(model.Values[1])
							h.must(e)
							x1 := mustLPRat(t, text)
							a, b := big.NewRat(int64(1+world%7), int64(1+world%5)), big.NewRat(int64(1+world%11), int64(1+world%3))
							lhs := new(big.Rat).Add(new(big.Rat).Mul(a, v), new(big.Rat).Mul(b, x1))
							cap := new(big.Rat).Add(new(big.Rat).Mul(a, low), big.NewRat(1, 17))
							if x1.Sign() < 0 || lhs.Cmp(cap) > 0 {
								t.Fatal("independent nonnegative variable and term bound", x1, lhs, cap)
							}
							for _, con := range s.Constraints {
								holds, e := s.ConstraintHolds(con, model)
								h.must(e)
								if !holds {
									t.Fatal("actual retained constraint")
								}
							}
							h.must(model.Close())
							if s.RowsWithChangedBounds.Empty() {
								t.Fatal("missing touched row")
							}
						}
						h.must(s.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
				}
			}
		}
	}
}

func TestLARFullSolveOptimizationAndRepeatedCalls(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, propagate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%v", strategy, propagate), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, _ := lpLARFullSolveFixture(t, h, strategy, 4, false, true)
				s.Settings.BoundPropagation = propagate
				h.must(s.RowsWithChangedBounds.Clear())
				err := s.SolveWithCoreSolver()
				// The source leaves feasibility costs active here. With no
				// entering column, DecideOnStatusWhenCannotFindEntering
				// asserts need_to_switch_costs in the costs strategy.
				if strategy == StrategyTableauCosts && sourceDebugEnabled {
					if err != ErrAssertion || !s.Core.RPrimal.NeedToSwitchCosts() || !s.Core.stopped || s.Bounds.ChangedColumns.Empty() {
						t.Fatal("source optimization assertion", err)
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
					return
				}
				h.must(err)
				if s.Bounds.Status.Value != StatusOptimal || s.Bounds.ChangedColumns.Empty() || !s.RowsWithChangedBounds.Empty() || s.Core.RPrimal.LookForFeasibleSolutionOnly != (strategy == StrategyLU) {
					t.Fatal("core-only publication and source signature flag")
				}

				for n := 0; n < 3; n++ {
					status, err := s.Solve()
					h.must(err)
					if status != StatusOptimal || !s.Bounds.ChangedColumns.Empty() || s.RowsWithChangedBounds.Empty() == propagate {
						t.Fatal("repeat solve and propagation gate", status)
					}
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARFullSolveCancellationAndDiagnosticPrefixes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, phase := range []string{"head-cancel", "floating-timeout", "head-error", "false-constraint", "invalid-changed-column", "stopped-find", "row-domain"} {
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, _ := lpLARFullSolveFixture(t, h, strategy, 4, phase == "head-cancel" || phase == "floating-timeout" || phase == "head-error" || phase == "row-domain", false)
				before := append([]uint32(nil), s.Bounds.ChangedColumns.Indices()...)
				limit := &lpTestResourceLimit{}
				s.Settings.SetResourceLimit(limit)
				wantErr := error(nil)
				wantStatus := StatusOptimal
				switch phase {
				case "head-cancel":
					limit.cancel = true

				case "floating-timeout":
					_, e := s.AddVarBound(0, LConstraintGE, &h.pair("17", "0").X)
					h.must(e)
					_, e = s.AddVarBound(LARTermStartIndex, LConstraintEQ, &s.Constraints[2].Base().RightSide)
					h.must(e)
					before = append([]uint32(nil), s.Bounds.ChangedColumns.Indices()...)
					limit.cancel = true
					if strategy == StrategyLU {
						wantStatus = StatusTimeExhausted
					}
				case "head-error":
					limit.err = lpErrExactSolveCancel
					wantErr = lpErrExactSolveCancel
				case "false-constraint":
					con, err := NewLARVarConstraint(h.m, 1, LConstraintLT, &h.pair("-1", "0").X)
					h.must(err)
					s.Constraints = append(s.Constraints, con)
					if sourceDebugEnabled {
						wantErr = ErrAssertion
					}
				case "invalid-changed-column":
					h.must(s.Bounds.ChangedColumns.Resize(4))
					h.must(s.Bounds.ChangedColumns.Insert(3))
					before = append([]uint32(nil), s.Bounds.ChangedColumns.Indices()...)
					wantErr = ErrBounds
				case "stopped-find":
					s.stopped = true
					wantErr = ErrExactVectorPending
				case "row-domain":
					h.must(s.RowsWithChangedBounds.Clear())
					h.must(s.RowsWithChangedBounds.Resize(0))
					limit.cancel = true
					wantErr = ErrBounds

				}
				var status Status
				var err error
				output := lpCaptureFeasibleStdout(t, func() { status, err = s.FindFeasibleSolution() })
				if err != wantErr || status != wantStatus {
					t.Fatal("source status/error publication", err, wantErr, status, wantStatus, limit.calls, output, "double status", s.Core.DPrimal.Status, "double values", s.Core.DX)
				}
				if wantErr == nil {
					if !s.Bounds.ChangedColumns.Empty() {
						t.Fatal("ordinary return retained changed columns")
					}
				} else {
					if !reflect.DeepEqual(before, s.Bounds.ChangedColumns.Indices()) {
						t.Fatal("failure erased changed columns")
					}
					if _, e := s.Solve(); e != ErrExactVectorPending {
						t.Fatal("stopped retry", e)
					}
				}
				if phase == "false-constraint" && sourceDebugEnabled && output != "v31 < -1\n" {
					t.Fatal("actual violated diagnostic", output)
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARFullSolvePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, direct := range []bool{false, true} {
			for _, largeCoefficient := range []bool{false, true} {
				done, failures := false, 0
				for ordinal := int64(0); ordinal < 1800 && !done; ordinal++ {
					t.Run(fmt.Sprintf("%d/%v/%v/%d", strategy, direct, largeCoefficient, ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						s, _ := lpLARFullSolveNumericFixture(t, h, strategy, 4, false, true, largeCoefficient)
						// A real retained bound, rather than a detached numeric helper,
						// forces allocation during clamping and basis propagation.
						huge := strings.Repeat("1234567890", 20) + "7/13"
						_, err := s.AddVarBound(0, LConstraintGT, &h.pair(huge, "0").X)
						h.must(err)
						// The positive term cap is now impossible; the bound owner does
						// not detect this until the real linear solve runs.
						h.budget.FailAfter(ordinal)
						status, err := s.FindFeasibleSolution()
						h.budget.FailAfter(-1)
						if err == nil {
							done = true
							if status != StatusInfeasible {
								t.Fatal("independent impossible term", status)
							}
						} else {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal("unexpected physical failure", err)
							}
							if _, e := s.Solve(); e != ErrExactVectorPending {
								t.Fatal("stopped owner", e)
							}
						}
						h.must(s.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
				}
				if !done || failures == 0 {
					t.Fatal("incomplete allocation campaign", strategy, direct, done, failures)
				}
			}
		}
	}
}

func TestLARFullSolvePreparationAndValidationErrors(t *testing.T) {
	for _, phase := range []string{"pair-factor", "float-factor", "cost-clear", "validation"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			strategy := StrategyLU
			if phase == "cost-clear" || phase == "validation" {
				strategy = StrategyTableauCosts
			}
			s, _ := lpLARFullSolveFixture(t, h, strategy, 4, false, false)
			before := append([]uint32(nil), s.Bounds.ChangedColumns.Indices()...)
			want := error(ErrBounds)
			switch phase {
			case "pair-factor":
				s.Core.RBasis[0] = 999
			case "float-factor":
				s.Core.DBasis[0] = 999
			case "cost-clear":
				h.must(s.BasicColumnsWithChangedCost.Resize(3))
				h.must(s.BasicColumnsWithChangedCost.Insert(2))
				h.must(s.BasicColumnsWithChangedCost.Resize(0))
			case "validation":
				s.Bounds.Status.Value = StatusUnknown
				s.Settings.SetResourceLimit(&lpTestResourceLimit{err: lpErrExactSolveCancel})
				want = nil
				if sourceDebugEnabled {
					want = lpErrExactSolveCancel
				}
			}
			status, err := s.FindFeasibleSolution()
			if err != want {
				t.Fatal("reached preparation/validation error", err, want)
			}
			if phase == "validation" && status != StatusOptimal {
				t.Fatal("status must publish before debug validation", status)
			}
			if err != nil {
				if !reflect.DeepEqual(before, s.Bounds.ChangedColumns.Indices()) {
					t.Fatal("error erased changed-column prefix")
				}
				if _, e := s.Solve(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			// An invalid source set member is outside Clear's domain; its Go
			// guard leaves the member present. Restore its nonowning domain.
			if phase == "cost-clear" {
				h.must(s.BasicColumnsWithChangedCost.Resize(3))
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARFullSolveInfeasibleEarlyReturnAndSelection(t *testing.T) {
	for _, entry := range []string{"solve", "find", "core"} {
		t.Run(entry, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(17)
			h.must(err)
			_, err = s.AddVarBound(0, LConstraintGE, &h.pair("2", "0").X)
			h.must(err)
			_, err = s.AddVarBound(0, LConstraintLE, &h.pair("1", "0").X)
			h.must(err)
			before := append([]uint32(nil), s.Bounds.ChangedColumns.Indices()...)
			s.Settings.SimplexStrategy = StrategyUndecided
			limit := &lpTestResourceLimit{err: lpErrExactSolveCancel}
			s.Settings.SetResourceLimit(limit)
			if entry == "solve" {
				status, e := s.Solve()
				h.must(e)
				if status != StatusInfeasible || s.Settings.SimplexStrategy != StrategyUndecided || s.Core.RPrimal.LookForFeasibleSolutionOnly {
					t.Fatal("early solve changed prefix")
				}
			} else if entry == "find" {
				status, e := s.FindFeasibleSolution()
				h.must(e)
				if status != StatusInfeasible || s.Settings.SimplexStrategy != StrategyTableauRows || !s.Core.RPrimal.LookForFeasibleSolutionOnly {
					t.Fatal("selection and feasible-only flag before infeasible return")
				}
			} else {
				s.stopped = true
				if e := s.SolveWithCoreSolver(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			if !reflect.DeepEqual(before, s.Bounds.ChangedColumns.Indices()) || limit.calls != 0 {
				t.Fatal("early return erased changed columns or read cancellation")
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARFullSolvePreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, _ := lpLARFullSolveFixture(t, h, StrategyLU, 4, false, true)
			h.must(s.Bounds.Low.Set(0, h.pair(strings.Repeat("1234567890", 20)+"7/13", "1")))

			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = s.FindFeasibleSolution()
				if err == nil || s.calls.comparison != 0 {
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

func TestLARFullSolveUndecidedSelectionAndFailure(t *testing.T) {
	for _, phase := range []string{"rows", "LU", "deferred-error", "pending-core"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			s.Settings.SimplexStrategy = StrategyUndecided
			_, err = s.AddVar(17)
			h.must(err)
			if phase == "LU" {
				s.Settings.ColumnThresholdForLUInLAR = 0
			}
			if phase == "deferred-error" {
				input := NewSparseRationalVector(h.m)
				h.must(input.Append(0, &h.pair("1", "0").X))
				_, err = s.AddTerm(input.Data, &h.pair("0", "0").X)
				h.must(err)
				h.must(input.Close())
				_, err = s.Terms[0].beginWork(1)
				h.must(err)
			}
			if phase == "pending-core" {
				s.Core.RPrimal.stopped = true
			}
			status, err := s.FindFeasibleSolution()
			if phase == "deferred-error" || phase == "pending-core" {
				if err != ErrExactVectorPending || s.Settings.SimplexStrategy != StrategyTableauRows || s.Core.RPrimal.LookForFeasibleSolutionOnly != (phase == "pending-core") {
					t.Fatal("strategy publishes before adjustment; flag only after adjustment", err)
				}
				if _, err = s.FindFeasibleSolution(); err != ErrExactVectorPending {
					t.Fatal("stopped find", err)
				}
			} else {
				h.must(err)
				want := StrategyTableauRows
				if phase == "LU" {
					want = StrategyLU
				}
				if status != StatusOptimal || s.Settings.SimplexStrategy != want || !s.Core.RPrimal.LookForFeasibleSolutionOnly {
					t.Fatal("literal strategy threshold", status, s.Settings.SimplexStrategy)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
