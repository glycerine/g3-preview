// Source: src/util/lp/lar_core_solver.hpp:275-319.
package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func lpLAROuterSolveFixture(t *testing.T, h *lpPairHarness, strategy SimplexStrategy, lower, extent int, strict bool) *LARCoreSolver {
	t.Helper()
	c := lpLARFloatingSignatureFixture(t, h, lower, extent, strict)
	c.RPrimal.Names = lpFeasibleNames{}
	if strategy == StrategyLU {
		h.must(c.RPrimal.InfSet.Resize(3))
		h.must(c.RPrimal.InitInfSet())
	} else {
		lpPrepareLARSignatureTableau(t, h, c, strategy)
	}
	c.RPrimal.LookForFeasibleSolutionOnly = true
	return c
}

func TestLAROuterSolveIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for world := 0; world < 100; world++ {
			for _, profile := range []string{"feasible", "early", "optimize", "infeasible"} {
				t.Run(fmt.Sprintf("%d/%d/%s", strategy, world, profile), func(t *testing.T) {
					h := newLPPairHarness(t)
					lower, extent := 1+world%17, 20+world%7
					strict := world%2 == 0
					if profile == "early" || profile == "optimize" {
						lower = -lower
					}
					if profile == "infeasible" {
						lower = 2*extent + 1
					}
					c := lpLAROuterSolveFixture(t, h, strategy, lower, extent, strict)
					if profile == "optimize" {
						c.RPrimal.LookForFeasibleSolutionOnly = false
					}
					c.settings.Stats.NeedToSolveInf = 41
					oldFloatTrace := append([]uint32(nil), c.DPrimal.Trace...)
					h.must(c.Solve())
					wantCount := uint32(42)
					if profile == "early" {
						wantCount = 41
					}
					if c.settings.Stats.NeedToSolveInf != wantCount {
						t.Fatal("source counter", c.settings.Stats.NeedToSolveInf, wantCount)
					}
					wantX, wantY := big.NewRat(int64(lower), 1), new(big.Rat)
					if strict {
						wantY.SetInt64(1)
					}
					if profile == "early" || profile == "optimize" {
						wantX, wantY = new(big.Rat), new(big.Rat)
					}
					if profile == "infeasible" && strategy != StrategyTableauRows {
						wantX, wantY = big.NewRat(int64(2*extent), 1), new(big.Rat)
					}
					h.check(&c.RX.Data[0], wantX, wantY)
					h.check(&c.RX.Data[1], new(big.Rat).Quo(new(big.Rat).Neg(wantX), big.NewRat(2, 1)), new(big.Rat).Quo(new(big.Rat).Neg(wantY), big.NewRat(2, 1)))
					h.check(&c.RX.Data[2], new(big.Rat), new(big.Rat))
					if profile == "infeasible" {
						if c.RPrimal.Status != StatusInfeasible || len(c.Infeasibility.Data) == 0 {
							t.Fatal("independently impossible interval and actual explanation", c.RPrimal.Status)
						}
						// Any multiple of the original homogeneous row must annihilate two
						// independent witnesses; this checks the explanation as an equation.
						for _, k := range []int64{-7, 19} {
							x1, x2 := big.NewRat(k, 3), big.NewRat(k+2, 5)
							x0 := new(big.Rat).Neg(new(big.Rat).Add(new(big.Rat).Mul(big.NewRat(2, 1), x1), new(big.Rat).Mul(big.NewRat(3, 1), x2)))
							values := []*big.Rat{x0, x1, x2}
							sum := new(big.Rat)
							for i := range c.Infeasibility.Data {
								cell := &c.Infeasibility.Data[i]
								text, e := h.m.String(&cell.Value)
								h.must(e)
								sum.Add(sum, new(big.Rat).Mul(mustLPRat(t, text), values[cell.Index]))
							}
							if sum.Sign() != 0 {
								t.Fatal("independent explanation equation", sum)
							}
						}
					} else if c.RPrimal.Status != StatusOptimal || !c.RPrimal.CurrentXIsFeasible() {
						t.Fatal("outer feasibility/status", c.RPrimal.Status)
					}
					if profile == "early" && !reflect.DeepEqual(c.DPrimal.Trace, oldFloatTrace) {
						t.Fatal("early return reached presolver")
					}
					h.must(c.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
		}
	}
}

func TestLAROuterSolveCancellationPrefixes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, phase := range []string{"head-cancel", "head-error", "exit-error", "post-cancel"} {
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				c := lpLAROuterSolveFixture(t, h, strategy, 3, 7, false)
				limit := &lpExactSolveCancelSequence{}
				wantErr := error(nil)
				wantCalls := 1
				wantStatus := StatusOptimal
				pivot := false
				switch phase {
				case "head-cancel":
					limit.values = []bool{true, true}
					if strategy == StrategyLU {
						wantCalls = 2
						wantStatus = StatusTimeExhausted
					}
				case "head-error":
					limit.errorAt = 1
					wantErr = lpErrExactSolveCancel
				case "exit-error":
					limit.errorAt = 2
					wantErr = lpErrExactSolveCancel
					wantCalls = 2
					pivot = true
				case "post-cancel":
					if strategy == StrategyLU {
						limit.values = []bool{false, false, true}
						wantCalls = 3
					} else {
						limit.values = []bool{false, true}
						wantCalls = 2
					}
					pivot = true
				}
				c.settings.SetResourceLimit(limit)
				err := c.Solve()
				if err != wantErr || limit.calls != wantCalls || c.stopped != (wantErr != nil) || c.settings.Stats.NeedToSolveInf != 1 {
					t.Fatal("outer cancellation/error prefix", err, wantErr, limit.calls)
				}
				if wantErr == nil && c.RPrimal.Status != wantStatus {
					t.Fatal("literal outer status mapping", c.RPrimal.Status, wantStatus)
				}
				// With LU the second cancellation read occurs after a floating pivot,
				// before exact catch-up; an error leaves exact X and its basis untouched.
				wantX := int64(0)
				if pivot && (strategy != StrategyLU || wantErr == nil) {
					wantX = 3
				}
				h.check(&c.RX.Data[0], big.NewRat(wantX, 1), new(big.Rat))
				if phase == "head-cancel" && strategy != StrategyLU && (c.RPrimal.Status != StatusOptimal || !c.RPrimal.InfSet.Contains(0)) {
					t.Fatal("source maps cancelled infeasible tableau run to optimal")
				}
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

type lpLAROuterStrategyLimit struct {
	c        *LARCoreSolver
	strategy SimplexStrategy
	calls    int
}

func (l *lpLAROuterStrategyLimit) CancelFlag() (bool, error) {
	l.calls++
	if l.calls == 2 {
		l.c.settings.SimplexStrategy = l.strategy
	}
	return false, nil
}
func TestLAROuterSolveLiveStrategyAfterPresolve(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts} {
		h := newLPPairHarness(t)
		c := lpLAROuterSolveFixture(t, h, StrategyLU, 3, 7, true)
		oldR := c.RPrimal.Factorization
		limit := &lpLAROuterStrategyLimit{c: c, strategy: strategy}
		c.settings.SetResourceLimit(limit)
		h.must(c.Solve())
		if limit.calls != 3 || c.RPrimal.Status != StatusOptimal || c.settings.SimplexStrategy != strategy || c.RPrimal.Factorization != oldR || oldR.closed || c.RBasis[0] != 1 || c.DBasis[0] != 1 {
			t.Fatal("live settings dispatch", limit.calls, c.RPrimal.Status, c.RBasis, c.DBasis)
		}
		h.check(&c.RX.Data[0], big.NewRat(3, 1), big.NewRat(1, 1))
		h.check(&c.RX.Data[1], big.NewRat(-3, 2), big.NewRat(-1, 2))
		if len(c.RPrimal.Trace) != 0 || c.RPrimal.Tracing {
			t.Fatal("caught-up tableau is initially feasible")
		}
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLAROuterSolveUndecidedSnapAndReconstruction(t *testing.T) {
	for _, profile := range []string{"bound", "interior-upper", "interior-low"} {
		h := newLPPairHarness(t)
		c := lpLAROuterSolveFixture(t, h, StrategyLU, 3, 7, false)
		c.settings.SimplexStrategy = StrategyUndecided
		if profile != "bound" {
			h.must(c.RX.Data[0].Copy(h.m, h.pair("2", "0")))
			h.must(c.RX.Data[1].Copy(h.m, h.pair("-1", "0")))
		}
		seed := uint32(0)
		if profile == "interior-low" {
			seed = 1
		}
		c.settings.RandomSeed(seed)
		oldD, oldTrace := c.DPrimal.Factorization, append([]uint32(nil), c.DPrimal.Trace...)
		h.must(c.Solve())
		want := int64(3)
		if profile == "interior-low" {
			want = 14
		}
		h.check(&c.RX.Data[0], big.NewRat(want, 1), new(big.Rat))
		h.check(&c.RX.Data[1], big.NewRat(-want, 2), new(big.Rat))
		if c.RPrimal.Status != StatusOptimal || c.DPrimal.Factorization != oldD || !reflect.DeepEqual(c.DPrimal.Trace, oldTrace) {
			t.Fatal("nonpresolve source dispatch")
		}
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLAROuterSolvePreservesUnbounded(t *testing.T) {
	h := newLPPairHarness(t)
	c := lpLAROuterSolveFixture(t, h, StrategyTableauCosts, -1, 7, false)
	c.RPrimal.LookForFeasibleSolutionOnly = false
	c.Bounds.Types.values[0], c.Bounds.Types.values[1] = ColumnFree, ColumnFree
	h.must(h.m.SetInt64(&c.Costs.Data[1], 1))
	h.must(c.RPrimal.InitReducedCostsTableau())
	h.must(c.Solve())
	if c.RPrimal.Status != StatusUnbounded || c.settings.Stats.NeedToSolveInf != 1 {
		t.Fatal("unbounded objective x1, x0=-2*x1", c.RPrimal.Status)
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLAROuterSolvePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
		for _, direct := range []bool{false, true} {
			for _, infeasible := range []bool{false, true} {
				success, failures := false, 0
				for ordinal := int64(0); ordinal < 2000; ordinal++ {
					done := false
					t.Run(fmt.Sprintf("%d/%t/%t/%d", strategy, direct, infeasible, ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						lower := 3
						if infeasible {
							lower = 17
						}
						c := lpLAROuterSolveFixture(t, h, strategy, lower, 7, true)
						// A retained large coefficient reaches actual bound conversion,
						// reconstruction, tableau arithmetic and explanation copying.
						huge := strings.Repeat("1234567890", 20) + "7/13"
						lpSetLARSignatureBothMatrices(t, h, c, 0, 1, huge)
						if strategy == StrategyTableauCosts {
							h.must(c.RPrimal.InitReducedCostsTableau())
						}
						if infeasible {
							h.must(c.Bounds.Low.Set(1, h.pair("0", "0")))
						}
						h.budget.FailAfter(ordinal)
						err := c.Solve()
						h.budget.FailAfter(-1)
						if err == nil {
							success, done = true, true
							if infeasible && c.RPrimal.Status != StatusInfeasible {
								t.Fatal("impossible positive x0, nonnegative x1", c.RPrimal.Status)
							}
						} else {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							if c.Solve() != ErrExactVectorPending {
								t.Fatal("stopped retry")
							}
						}
						h.must(c.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if !success || direct && failures == 0 {
					t.Fatal("physical campaign", strategy, direct, infeasible, success, failures)
				}
			}
		}
	}
}

func TestLAROuterSolveReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "nil-r", "pending-r", "nil-d", "entry-nonbasic", "entry-membership", "residual", "basis"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			lower := 3
			if phase == "entry-nonbasic" || phase == "entry-membership" {
				lower = -1
			}
			strategy := StrategyTableauRows
			if phase == "nil-d" {
				strategy = StrategyLU
			}
			c := lpLAROuterSolveFixture(t, h, strategy, lower, 7, false)
			oldR, oldD := c.RPrimal, c.DPrimal
			limit := &lpTestResourceLimit{cancel: true}
			c.settings.SetResourceLimit(limit)
			want := error(nil)
			wantCount := uint32(1)
			switch phase {
			case "stopped":
				c.stopped = true
				want = ErrExactVectorPending
				wantCount = 0
			case "nil-r":
				c.RPrimal = nil
				want = ErrBounds
				wantCount = 0
			case "pending-r":
				c.RPrimal.stopped = true
				want = ErrExactVectorPending
				wantCount = 0
			case "nil-d":
				c.DPrimal = nil
				want = ErrBounds
			case "entry-nonbasic":
				h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
				wantCount = 0
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "entry-membership":
				h.must(c.RPrimal.InfSet.Insert(0))
				if sourceDebugEnabled {
					want = ErrAssertion
					wantCount = 0
				}
			case "residual":
				h.must(c.RX.Data[0].Copy(h.m, h.pair("1", "0")))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis":
				lpSetLARCatchUpExactCell(t, h, c, 0, 0, "2")
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			var err error
			output := lpCaptureFeasibleStdout(t, func() { err = c.Solve() })
			if err != want || c.settings.Stats.NeedToSolveInf != wantCount {
				t.Fatal("source stage", err, want, c.settings.Stats.NeedToSolveInf, wantCount)
			}
			if sourceDebugEnabled && phase == "entry-membership" && !strings.HasSuffix(output, "belongs_to_set = 1\nfeas\n") {
				t.Fatal("actual entry diagnostic", output)
			}
			if sourceDebugEnabled && phase == "residual" && !strings.Contains(output, "left side = (1, 0)") {
				t.Fatal("actual residual diagnostic", output)
			}
			if err != nil && c.Solve() != ErrExactVectorPending {
				t.Fatal("stopped retry")
			}
			c.RPrimal, c.DPrimal = oldR, oldD
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

type lpLAROuterStageLimit struct {
	c     *LARCoreSolver
	mode  string
	calls int
}

func (l *lpLAROuterStageLimit) CancelFlag() (bool, error) {
	l.calls++
	if l.calls == 2 {
		switch l.mode {
		case "exact-heading":
			l.c.RHeading = nil
		case "solve-domain":
			if err := l.c.RPrimal.InfSet.Resize(0); err != nil {
				return false, err
			}
			return true, nil
		case "final-membership":
			if err := l.c.RPrimal.InfSet.Insert(1); err != nil {
				return false, err
			}
		}
	}
	return false, nil
}
func TestLAROuterSolvePostSearchPrefixes(t *testing.T) {
	for _, mode := range []string{"exact-heading", "solve-domain", "final-membership"} {
		h := newLPPairHarness(t)
		strategy := StrategyLU
		if mode == "final-membership" {
			strategy = StrategyTableauRows
		}
		c := lpLAROuterSolveFixture(t, h, strategy, 3, 7, false)
		heading := c.RHeading
		limit := &lpLAROuterStageLimit{c: c, mode: mode}
		c.settings.SetResourceLimit(limit)
		want := error(ErrBounds)
		if mode == "final-membership" {
			want = nil
			if sourceDebugEnabled {
				want = ErrAssertion
			}
		}
		var err error
		output := lpCaptureFeasibleStdout(t, func() { err = c.Solve() })
		if err != want || limit.calls != 2 || c.settings.Stats.NeedToSolveInf != 1 {
			t.Fatal("post-search stage", mode, err, want, limit.calls)
		}
		if mode == "solve-domain" && (!c.RPrimal.Tracing || !c.DPrimal.Tracing || c.RBasis[0] != 0 || c.DBasis[0] != 1) {
			t.Fatal("solve fails after tracing start", c.RBasis, c.DBasis)
		}
		if mode == "final-membership" {
			if c.RPrimal.Status != StatusOptimal || !strings.Contains(output, "belongs_to_set = 1") && sourceDebugEnabled {
				t.Fatal("status maps before final assertion", output, c.RPrimal.Status)
			}
			h.check(&c.RX.Data[0], big.NewRat(3, 1), new(big.Rat))
		}
		c.RHeading = heading
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLAROuterSolvePreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			c := lpLAROuterSolveFixture(t, h, StrategyLU, 3, 7, false)
			h.must(c.Bounds.Upper.Set(1, h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")))
			h.must(c.RX.Data[1].Copy(h.m, h.pair("1", "0")))
			h.must(c.RX.Data[0].Copy(h.m, h.pair("-2", "0")))
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
				err = c.Solve()
				if err == nil || c.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(c.Close())
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

func TestLAROuterSolveCounterWrapAndRepeatedEarlyReturn(t *testing.T) {
	h := newLPPairHarness(t)
	c := lpLAROuterSolveFixture(t, h, StrategyLU, 3, 7, true)
	c.settings.Stats.NeedToSolveInf = ^uint32(0)
	h.must(c.Solve())
	if c.settings.Stats.NeedToSolveInf != 0 || c.RPrimal.Status != StatusOptimal {
		t.Fatal("unsigned source counter wrap")
	}
	before := append([]uint32(nil), c.DPrimal.Trace...)
	limit := &lpTestResourceLimit{err: lpErrExactSolveCancel}
	c.settings.SetResourceLimit(limit)
	h.must(c.Solve())
	if limit.calls != 0 || c.settings.Stats.NeedToSolveInf != 0 || !reflect.DeepEqual(c.DPrimal.Trace, before) {
		t.Fatal("repeat skips presolve and cancellation")
	}
	h.must(c.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLAROuterSolvePreparationAndReconstructionErrors(t *testing.T) {
	for _, phase := range []string{"floating-prefix", "snapped-reconstruction"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			c := lpLAROuterSolveFixture(t, h, StrategyLU, 3, 7, false)
			factor := c.RPrimal.Factorization
			if phase == "floating-prefix" {
				// Actual indexed scratch clearing fails after the ordinary
				// floating prefix has published, before cancellation is read.
				c.DPrimal.Ed.Index = []uint32{uint32(len(c.DPrimal.Ed.Data))}
			} else {
				c.settings.SimplexStrategy = StrategyUndecided
				c.settings.RandomSeed(0)
				h.must(c.RX.Data[0].Copy(h.m, h.pair("2", "0")))
				h.must(c.RX.Data[1].Copy(h.m, h.pair("-1", "0")))
				c.RPrimal.Factorization = nil
			}
			limit := &lpTestResourceLimit{err: lpErrExactSolveCancel}
			c.settings.SetResourceLimit(limit)
			if err := c.Solve(); err != ErrBounds || limit.calls != 0 || c.settings.Stats.NeedToSolveInf != 1 {
				t.Fatal("pre-search failure prefix", phase, err, limit.calls)
			}
			if c.Solve() != ErrExactVectorPending {
				t.Fatal("failed outer solve must stop")
			}
			if phase == "floating-prefix" {
				if len(c.DRightSides) != 1 || len(c.DPrimal.CopyXB) != 3 || len(c.DPrimal.Ed.Index) != 1 {
					t.Fatal("floating preparation publication")
				}
				c.DPrimal.Ed.Index = nil
			} else {
				h.check(&c.RX.Data[0], big.NewRat(2, 1), new(big.Rat))
				h.check(&c.RX.Data[1], new(big.Rat), new(big.Rat))
				if c.RPrimal.reconstructionWork == nil {
					t.Fatal("actual reconstruction must retain interrupted work")
				}
			}
			c.RPrimal.Factorization = factor
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
