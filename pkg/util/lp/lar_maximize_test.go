package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpLARMaximizeBox(t *testing.T, h *lpPairHarness, strategy SimplexStrategy, world int, strict bool) *LARSolver {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	s.Settings.SimplexStrategy = strategy
	for j := uint32(0); j < 2; j++ {
		_, err = s.AddVar(j + 17)
		h.must(err)
		lo := big.NewRat(int64(world%7), 11)
		hi := new(big.Rat).Add(lo, big.NewRat(int64(1+world%5), 7))
		lowKind, upperKind := LConstraintGE, LConstraintLE
		if strict {
			lowKind, upperKind = LConstraintGT, LConstraintLT
		}
		_, err = s.AddVarBound(j, lowKind, &h.pair(lo.RatString(), "0").X)
		h.must(err)
		_, err = s.AddVarBound(j, upperKind, &h.pair(hi.RatString(), "0").X)
		h.must(err)
	}
	status, err := s.FindFeasibleSolution()
	h.must(err)
	if status != StatusOptimal {
		t.Fatal(status)
	}
	return s
}

func TestLARMaximizeIndependentBoxVertices(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
		for world := 0; world < 40; world++ {
			for _, strict := range []bool{false, true} {
				h := newLPPairHarness(t)
				s := lpLARMaximizeBox(t, h, strategy, world, strict)
				var term []RationalSparseEntry
				want := [2]*big.Rat{new(big.Rat), new(big.Rat)}
				lo := big.NewRat(int64(world%7), 11)
				hi := new(big.Rat).Add(lo, big.NewRat(int64(1+world%5), 7))
				for j := uint32(0); j < 2; j++ {
					a := big.NewRat(int64(1+(world+int(j))%11), int64(1+world%7))
					if (world+int(j))%2 != 0 {
						a.Neg(a)
					}
					term = append(term, RationalSparseEntry{Index: j, Value: h.pair(a.RatString(), "0").X})
					value, epsilon := hi, int64(0)
					if a.Sign() < 0 {
						value = lo
					}
					if strict {
						epsilon = -1
						if a.Sign() < 0 {
							epsilon = 1
						}
					}
					want[0].Add(want[0], new(big.Rat).Mul(a, value))
					want[1].Add(want[1], new(big.Rat).Mul(a, big.NewRat(epsilon, 1)))
				}
				out := h.pair("97", "101")
				bounded, err := s.MaximizeTerm(term, out)
				h.must(err)
				if !bounded {
					t.Fatal("bounded box reported unbounded", strategy, world)
				}
				h.check(out, want[0], want[1])
				if s.Settings.SimplexStrategy != strategy || s.Settings.BackupCosts || s.Core.RPrimal.LookForFeasibleSolutionOnly || s.Core.RPrimal.Status != StatusOptimal || !s.Core.RPrimal.InfSet.Empty() {
					t.Fatal("source completion state")
				}
				for _, cell := range term {
					zero, e := h.m.IsZero(&s.Core.Costs.Data[cell.Index])
					h.must(e)
					if !zero {
						t.Fatal("uncleared cost")
					}
				}
				for i := range s.Core.RPrimal.D.Data {
					zero, e := h.m.IsZero(&s.Core.RPrimal.D.Data[i])
					h.must(e)
					if !zero {
						t.Fatal("uncleared reduced cost")
					}
				}
				// Repeated optimization starts from the actual preceding optimum.
				again := h.pair("0", "0")
				bounded, err = s.MaximizeTerm(term, again)
				h.must(err)
				if !bounded {
					t.Fatal("repeated bounded objective")
				}
				h.check(again, want[0], want[1])
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLARMaximizeUnboundedAndUnsupportedModes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts, StrategyLU, StrategyUndecided} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		_, err = s.AddVar(17)
		h.must(err)
		// Keep source undecided as an unsupported corrected-solver entry; do not
		// invoke the feasibility front end that would select a strategy first.
		s.Settings.SimplexStrategy = strategy
		if strategy != StrategyUndecided {
			_, err = s.FindFeasibleSolution()
			h.must(err)
		}
		out := h.pair("97", "101")
		term := []RationalSparseEntry{{Index: 0, Value: h.pair("1", "0").X}}
		bounded, err := s.MaximizeTerm(term, out)
		want := error(nil)
		if sourceDebugEnabled && (strategy == StrategyLU || strategy == StrategyUndecided) {
			want = ErrAssertion
		}
		if bounded || err != want {
			t.Fatal(strategy, bounded, err, want)
		}
		h.check(out, big.NewRat(97, 1), big.NewRat(101, 1))
		if s.Settings.BackupCosts || s.Core.RPrimal.LookForFeasibleSolutionOnly {
			t.Fatal("source early settings publication")
		}
		if err == nil && (strategy == StrategyTableauRows || strategy == StrategyTableauCosts) && s.Core.RPrimal.Status != StatusOptimal {
			t.Fatal("source overwrites unbounded status after cleanup")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARMaximizeBasicTermAndPivot(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
		for world := 0; world < 60; world++ {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			s.Settings.SimplexStrategy = strategy
			for j := uint32(0); j < 2; j++ {
				_, err = s.AddVar(j + 17)
				h.must(err)
				_, err = s.AddVarBound(j, LConstraintGE, &h.pair("0", "0").X)
				h.must(err)
				_, err = s.AddVarBound(j, LConstraintLE, &h.pair("10", "0").X)
				h.must(err)
			}
			a, b := big.NewRat(int64(1+world%7), int64(1+world%5)), big.NewRat(int64(1+world%11), int64(1+world%3))
			input := []RationalSparseEntry{{Index: 0, Value: h.pair(a.RatString(), "0").X}, {Index: 1, Value: h.pair(b.RatString(), "0").X}}
			termID, err := s.AddTerm(input, &h.pair("0", "0").X)
			h.must(err)
			limit := big.NewRat(int64(1+world%9), 13)
			_, err = s.AddVarBound(termID, LConstraintLE, &h.pair(limit.RatString(), "0").X)
			h.must(err)
			_, err = s.FindFeasibleSolution()
			h.must(err)
			scalar := big.NewRat(int64(1+world%13), int64(1+world%7))
			term := []RationalSparseEntry{{Index: 2, Value: h.pair(scalar.RatString(), "0").X}}
			out := h.pair("97", "101")
			bounded, err := s.MaximizeTerm(term, out)
			h.must(err)
			if !bounded {
				t.Fatal("bounded term")
			}
			h.check(out, new(big.Rat).Mul(scalar, limit), big.NewRat(0, 1))
			// Independent original row equation and objective cap hold after a real pivot.
			values := [3]*big.Rat{}
			for j := range values {
				text, e := h.m.String(&s.Core.RX.Data[j].X)
				h.must(e)
				values[j] = mustLPRat(t, text)
			}
			sum := new(big.Rat).Add(new(big.Rat).Mul(a, values[0]), new(big.Rat).Mul(b, values[1]))
			if sum.Cmp(values[2]) != 0 || values[2].Cmp(limit) != 0 || s.Core.RHeading[2] >= 0 {
				t.Fatal("pivoted objective equation", world, values, s.Core.RHeading)
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARMaximizeCostPreparationIndependentOracle(t *testing.T) {
	for world := 0; world < 80; world++ {
		for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2/3", "-5/7", false)
			s.Settings.SimplexStrategy = strategy
			c, r := s.Core, s.Core.RPrimal
			h.must(c.PrefixR())
			r.UsingInfeasCosts = true
			scalars := []*big.Rat{big.NewRat(int64(world%9-4), 11), big.NewRat(int64(world%7-3), 13), big.NewRat(int64(world%11-5), 17)}
			var term []RationalSparseEntry
			for j, a := range scalars {
				term = append(term, RationalSparseEntry{Index: uint32(j), Value: h.pair(a.RatString(), "0").X})
			}
			h.must(s.PrepareCostsForRSolver(term))
			if r.UsingInfeasCosts {
				t.Fatal("did not leave feasibility costs")
			}
			for j, a := range scalars {
				lpSparseRationalCheck(t, h, &c.Costs.Data[j], a)
			}
			want := []*big.Rat{new(big.Rat).Add(scalars[0], new(big.Rat).Mul(big.NewRat(2, 3), scalars[2])), new(big.Rat).Sub(scalars[1], new(big.Rat).Mul(big.NewRat(5, 7), scalars[2])), big.NewRat(0, 1)}
			for j, a := range want {
				lpSparseRationalCheck(t, h, &r.D.Data[j], a)
			}
			h.must(s.SetCostsToZero(term))
			if !r.InfSet.Empty() {
				t.Fatal("scratch infeasibility set not cleared")
			}
			for j := 0; j < 3; j++ {
				lpSparseRationalCheck(t, h, &c.Costs.Data[j], big.NewRat(0, 1))
				lpSparseRationalCheck(t, h, &r.D.Data[j], big.NewRat(0, 1))
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARMaximizePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"prepare-nonbasic", "prepare-basic", "evaluate", "maximize"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 1800 && !done; ordinal++ {
				passed := t.Run(fmt.Sprintf("%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpLARMaximizeBox(t, h, StrategyTableauRows, 2, true)
					large := strings.Repeat("1234567890", 20) + "7/13"
					j := uint32(0)
					if op == "prepare-basic" {
						_, err := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: h.pair("3/7", "0").X}}, &h.pair("0", "0").X)
						h.must(err)
						h.must(s.Core.PrefixR())
						j = 2
					}
					term := []RationalSparseEntry{{Index: j, Value: h.pair(large, "0").X}}
					out := h.pair("97", "101")
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "prepare-nonbasic", "prepare-basic":
						err = s.PrepareCostsForRSolver(term)
					case "evaluate":
						_, err = s.MaximizeTermOnTableau(term, out)
					case "maximize":
						_, err = s.MaximizeTerm(term, out)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if _, e := s.MaximizeTerm(term, out); e != ErrExactVectorPending {
							t.Fatal("stopped retry", e)
						}
					}
					// Term input and output roots outlive Core/solver receipt cleanup.
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if !passed {
					t.FailNow()
				}
			}
			if !done || failures == 0 && direct {
				t.Fatal("incomplete physical campaign", direct, op, done, failures)
			}
		}
	}
}

func TestLARMaximizeEntryGuards(t *testing.T) {
	for _, op := range []string{"costs", "reduced", "prepare", "zero", "tableau", "corrected", "maximize"} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		s.stopped = true
		switch op {
		case "costs":
			_, err = s.CostsAreZerosForRSolver()
		case "reduced":
			_, err = s.ReducedCostsAreZeroesForRSolver()
		case "prepare":
			err = s.PrepareCostsForRSolver(nil)
		case "zero":
			err = s.SetCostsToZero(nil)
		case "tableau":
			_, err = s.MaximizeTermOnTableau(nil, nil)
		case "corrected":
			_, err = s.MaximizeTermOnCorrectedRSolver(nil, nil)
		case "maximize":
			_, err = s.MaximizeTerm(nil, nil)
		}
		if err != ErrExactVectorPending {
			t.Fatal(op, err)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARMaximizeZeroObjective(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpLARMaximizeBox(t, h, StrategyTableauRows, 1, false)
	out := h.pair("97", "101")
	bounded, err := s.MaximizeTerm(nil, out)
	h.must(err)
	if !bounded {
		t.Fatal("empty objective")
	}
	h.check(out, big.NewRat(0, 1), big.NewRat(0, 1))
	if len(s.Core.RBasis) != 0 {
		t.Fatal("empty objective changed basis")
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARMaximizeReachedPrefixes(t *testing.T) {
	for _, op := range []string{"cost-check", "reduced-check", "prepare-old-cost", "prepare-old-reduced", "prepare-column", "prepare-heading", "prepare-d", "prepare-basic-row", "zero-column", "zero-heading", "zero-row", "zero-domain", "zero-d", "zero-old-cost", "zero-old-reduced", "zero-inf", "nil-out", "evaluate-column", "entry-infeasible", "corrected-nil"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARMaximizeBox(t, h, StrategyTableauRows, 1, false)
			c, r := s.Core, s.Core.RPrimal
			term := []RationalSparseEntry{{Index: 0, Value: h.pair("3/7", "0").X}}
			out := h.pair("97", "101")
			want := error(ErrBounds)
			var err error
			switch op {
			case "cost-check":
				h.must(h.m.SetInt32(&c.Costs.Data[0], 1))
				_, err = s.CostsAreZerosForRSolver()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "reduced-check":
				h.must(h.m.SetInt32(&r.D.Data[0], 1))
				_, err = s.ReducedCostsAreZeroesForRSolver()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "prepare-old-cost":
				h.must(h.m.SetInt32(&c.Costs.Data[1], 1))
				r.UsingInfeasCosts = true
				err = s.PrepareCostsForRSolver(term)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				if r.UsingInfeasCosts {
					t.Fatal("cost flag precedes assertions")
				}
			case "prepare-old-reduced":
				h.must(h.m.SetInt32(&r.D.Data[1], 1))
				err = s.PrepareCostsForRSolver(term)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "prepare-column":
				term[0].Index = 99
				err = s.PrepareCostsForRSolver(term)
			case "prepare-heading":
				c.RHeading = nil
				err = s.PrepareCostsForRSolver(term)
				lpSparseRationalCheck(t, h, &c.Costs.Data[0], big.NewRat(3, 7))
			case "prepare-d":
				saved := r.D.Data
				r.D.Data = nil
				err = s.PrepareCostsForRSolver(term)
				r.D.Data = saved
			case "prepare-basic-row":
				c.RHeading[0] = 99
				err = s.PrepareCostsForRSolver(term)
			case "zero-column":
				term[0].Index = 99
				err = s.SetCostsToZero(term)
			case "zero-heading":
				c.RHeading = nil
				err = s.SetCostsToZero(term)
			case "zero-row":
				c.RHeading[0] = 99
				err = s.SetCostsToZero(term)
			case "zero-domain":
				h.must(r.InfSet.Resize(0))
				err = s.SetCostsToZero(term)
			case "zero-d":
				saved := r.D.Data
				r.D.Data = nil
				err = s.SetCostsToZero(term)
				r.D.Data = saved
				if !r.InfSet.Contains(0) {
					t.Fatal("reached scratch membership lost")
				}
			case "zero-old-cost":
				h.must(h.m.SetInt32(&c.Costs.Data[1], 1))
				err = s.SetCostsToZero(term)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "zero-old-reduced":
				h.must(h.m.SetInt32(&r.D.Data[1], 1))
				err = s.SetCostsToZero(term)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "zero-inf":
				h.must(r.InfSet.Insert(1))
				h.must(h.m.SetInt32(&r.D.Data[1], 1))
				err = s.SetCostsToZero(term)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				} else if !r.InfSet.Empty() {
					t.Fatal("native scratch reuse did not clear old membership")
				}
			case "nil-out":
				_, err = s.MaximizeTermOnTableau(nil, nil)
			case "evaluate-column":
				term[0].Index = 99
				_, err = s.MaximizeTermOnTableau(term, out)
				h.check(out, big.NewRat(0, 1), big.NewRat(0, 1))
			case "entry-infeasible":
				h.must(r.InfSet.Insert(0))
				_, err = s.MaximizeTerm(nil, out)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "corrected-nil":
				_, err = s.MaximizeTermOnCorrectedRSolver(term, nil)
				if s.Settings.SimplexStrategy != StrategyTableauCosts {
					t.Fatal("rows mode restored after reached error")
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARMaximizeDuplicateCostSourceAccumulation(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
		h := newLPPairHarness(t)
		s := lpLARMaximizeBox(t, h, strategy, 1, false)
		term := []RationalSparseEntry{{Index: 0, Value: h.pair("1", "0").X}, {Index: 0, Value: h.pair("2", "0").X}}
		var err error
		output := lpCaptureFeasibleStdout(t, func() { err = s.PrepareCostsForRSolver(term) })
		want := error(nil)
		if sourceDebugEnabled && strategy == StrategyTableauCosts {
			want = ErrAssertion
			if output == "" {
				t.Fatal("missing actual reduced-cost diagnostic")
			}
		}
		if err != want {
			t.Fatal(strategy, err, want)
		}
		lpSparseRationalCheck(t, h, &s.Core.Costs.Data[0], big.NewRat(2, 1))
		lpSparseRationalCheck(t, h, &s.Core.RPrimal.D.Data[0], big.NewRat(3, 1))
		if err == nil {
			h.must(s.SetCostsToZero(term))
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARMaximizeCancellationAndChildError(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
		for _, failure := range []bool{false, true} {
			h := newLPPairHarness(t)
			s := lpLARMaximizeBox(t, h, strategy, 2, false)
			term := []RationalSparseEntry{{Index: 0, Value: h.pair("3/7", "0").X}}
			out := h.pair("97", "101")
			limit := &lpExactSolveCancelSequence{values: []bool{true, true}}
			if failure {
				limit.errorAt = 1
			}
			s.Settings.SetResourceLimit(limit)
			bounded, err := s.MaximizeTerm(term, out)
			if failure {
				if err != lpErrExactSolveCancel || bounded {
					t.Fatal(bounded, err)
				}
				if s.Settings.SimplexStrategy != StrategyTableauCosts || s.Settings.BackupCosts || s.Core.RPrimal.LookForFeasibleSolutionOnly {
					t.Fatal("reached solve error settings")
				}
				h.check(out, big.NewRat(97, 1), big.NewRat(101, 1))
				lpSparseRationalCheck(t, h, &s.Core.Costs.Data[0], big.NewRat(3, 7))
			} else {
				h.must(err)
				if !bounded || limit.calls == 0 {
					t.Fatal(bounded, limit.calls)
				}
				// Core maps cancelled to optimal and the wrapper evaluates current X.
				// It does not establish the true upper vertex before returning.
				h.check(out, new(big.Rat).Mul(big.NewRat(3, 7), big.NewRat(2, 11)), big.NewRat(0, 1))
				if s.Settings.SimplexStrategy != strategy || s.Core.RPrimal.Status != StatusOptimal {
					t.Fatal("ordinary cancellation cleanup")
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARMaximizeDirectSourceStatuses(t *testing.T) {
	for _, mode := range []string{"undecided-empty", "infeasible"} {
		h := newLPPairHarness(t)
		var s *LARSolver
		var err error
		if mode == "infeasible" {
			s, _ = lpLARFullSolveFixture(t, h, StrategyTableauRows, 2, true, false)
			status, e := s.FindFeasibleSolution()
			h.must(e)
			if status != StatusInfeasible {
				t.Fatal(status)
			}
		} else {
			s, err = NewLARSolver(h.m)
			h.must(err)
			s.Settings.SimplexStrategy = StrategyUndecided
		}
		out := h.pair("97", "101")
		bounded, err := s.MaximizeTermOnTableau(nil, out)
		h.must(err)
		if !bounded {
			t.Fatal("source rejects only unbounded status", mode)
		}
		if mode == "infeasible" && s.Core.RPrimal.Status != StatusInfeasible {
			t.Fatal("direct source infeasible status changed", s.Core.RPrimal.Status)
		}
		h.check(out, big.NewRat(0, 1), big.NewRat(0, 1))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARMaximizePreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARMaximizeBox(t, h, StrategyTableauRows, 2, false)
			term := []RationalSparseEntry{{Index: 0, Value: h.pair("3/7", "0").X}}
			out := h.pair("97", "101")
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
				_, err = s.MaximizeTerm(term, out)
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
