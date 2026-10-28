package lp

// Source: src/util/lp/lar_solver.h:363-365,402-427;
// retained caller src/smt/theory_lra.cpp:1303-1314.

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

type lpTouchedInterval struct {
	low, high             *big.Rat
	lowStrict, highStrict bool
}

func lpLARTouchedRowsFixture(t *testing.T, h *lpPairHarness, world int, large bool) (*LARSolver, []lpTouchedInterval) {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	for _, ext := range []uint32{31, 67} {
		_, err = s.AddVar(ext)
		h.must(err)
	}
	var intervals []lpTouchedInterval
	for row := 0; row < 3; row++ {
		a, b := big.NewRat(int64(1+(world+row)%11), int64(1+world%5)), big.NewRat(int64(1+(world+2*row)%7), int64(1+world%3))
		if world&1 != 0 {
			a.Neg(a)
		}
		if world&2 != 0 {
			b.Neg(b)
		}
		if large {
			a = mustLPRat(t, strings.Repeat("1234567890", 20)+fmt.Sprint(1+row)+"/13")
		}
		constant := big.NewRat(int64(5+row), int64(1+world%7))
		input := NewSparseRationalVector(h.m)
		h.must(input.Append(0, &h.pair(a.RatString(), "0").X))
		h.must(input.Append(1, &h.pair(b.RatString(), "0").X))
		_, err = s.AddTerm(input.Data, &h.pair(constant.RatString(), "0").X)
		h.must(err)
		h.must(input.Close())
		lo, hi := new(big.Rat).Set(constant), new(big.Rat).Set(constant)
		for j, c := range []*big.Rat{a, b} {
			x, y := new(big.Rat).Mul(c, big.NewRat([]int64{-3, 2}[j], 1)), new(big.Rat).Mul(c, big.NewRat([]int64{7, 11}[j], 1))
			if x.Cmp(y) > 0 {
				x, y = y, x
			}
			lo.Add(lo, x)
			hi.Add(hi, y)
		}
		intervals = append(intervals, lpTouchedInterval{lo, hi, b.Sign() < 0, b.Sign() > 0})
	}
	for j := uint32(0); j < 2; j++ {
		_, err = s.AddVarBound(j, LConstraintGE, &h.pair(fmt.Sprint([]int{-3, 2}[j]), "0").X)
		h.must(err)
		kind := LConstraintLE
		if j == 1 {
			kind = LConstraintLT
		}
		_, err = s.AddVarBound(j, kind, &h.pair(fmt.Sprint([]int{7, 11}[j]), "0").X)
		h.must(err)
	}
	return s, intervals
}

func lpCheckTouchedIntervals(t *testing.T, h *lpPairHarness, p *LARBoundPropagator, intervals []lpTouchedInterval, order []uint32) {
	t.Helper()
	if len(p.FoundBounds) != 2*len(order) {
		t.Fatal("completed row publication", len(p.FoundBounds), order)
	}
	for k, row := range order {
		for side := 0; side < 2; side++ {
			b := p.FoundBounds[2*k+side]
			want := intervals[row].low
			strict := intervals[row].lowStrict
			if side == 1 {
				want = intervals[row].high
				strict = intervals[row].highStrict
			}
			if b.Column != LARTermStartIndex+row || b.RowOrTermIndex != row || b.IsLowBound != (side == 0) || b.Strict != strict || !b.CoefficientIsPositive {
				t.Fatal("row order and source evidence", k, side, b)
			}
			lpSparseRationalCheck(t, h, &b.Bound, want)
		}
	}
}

func TestLARTouchedRowsIndependentIntervalsAndSolve(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 80; world++ {
		for _, costs := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%v", world, costs), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, intervals := lpLARTouchedRowsFixture(t, h, world, false)
				if costs {
					s.Settings.SimplexStrategy = StrategyTableauCosts
				}
				status, err := s.FindFeasibleSolution()
				h.must(err)
				if status != StatusOptimal {
					t.Fatal(status)
				}
				h.must(s.RowsWithChangedBounds.Clear())
				order := []uint32{2, 0, 1}
				for _, row := range order {
					h.must(s.RowsWithChangedBounds.Insert(row))
				}
				limit := &lpTestResourceLimit{}
				s.Settings.SetResourceLimit(limit)
				p := s.NewBoundPropagator()
				h.must(s.PropagateBoundsForTouchedRows(p))
				lpCheckTouchedIntervals(t, h, p, intervals, order)
				if limit.calls != 3 || !s.RowsWithChangedBounds.Empty() || s.pendingRowAnalyzer != nil || s.pendingRowIterator != nil {
					t.Fatal("post-row reads and final lexical cleanup", limit.calls)
				}
				h.must(s.PropagateBoundsForTouchedRows(p))
				if limit.calls != 3 {
					t.Fatal("empty row set read cancellation")
				}
				for _, row := range order {
					h.must(s.RowsWithChangedBounds.Insert(row))
				}
				h.must(s.PropagateBoundsForTouchedRows(p))
				lpCheckTouchedIntervals(t, h, p, intervals, order)
				h.must(s.Close())
				h.must(p.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARTouchedRowsCancellationPrefixesAndReplay(t *testing.T) {
	for _, failure := range []bool{false, true} {
		for after := 1; after <= 3; after++ {
			t.Run(fmt.Sprintf("%v/%d", failure, after), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, intervals := lpLARTouchedRowsFixture(t, h, 5, false)
				order := append([]uint32(nil), s.RowsWithChangedBounds.Indices()...)
				limit := &lpExactSolveCancelSequence{values: make([]bool, after)}
				if failure {
					limit.errorAt = after
				} else {
					limit.values[after-1] = true
				}
				s.Settings.SetResourceLimit(limit)
				p := s.NewBoundPropagator()
				err := s.PropagateBoundsForTouchedRows(p)
				want := error(nil)
				if failure {
					want = lpErrExactSolveCancel
				}
				if err != want || limit.calls != after || !reflect.DeepEqual(order, s.RowsWithChangedBounds.Indices()) || s.stopped != failure {
					t.Fatal("source cancellation after completed row", err, limit.calls)
				}
				lpCheckTouchedIntervals(t, h, p, intervals, order[:after])
				if failure {
					if e := s.PropagateBoundsForTouchedRows(p); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				} else {
					s.Settings.SetResourceLimit(&lpTestResourceLimit{})
					h.must(s.PropagateBoundsForTouchedRows(p))
					lpCheckTouchedIntervals(t, h, p, intervals, order)
					if !s.RowsWithChangedBounds.Empty() {
						t.Fatal("replay retained marks")
					}
				}
				h.must(s.Close())
				h.must(p.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

type lpTouchedInterest struct{ calls, failAt int }

func (p *lpTouchedInterest) BoundIsInteresting(_ uint32, _ LConstraintKind, _ *numeral.Rational) (bool, error) {
	p.calls++
	if p.calls == p.failAt {
		return false, lpErrExactSolveCancel
	}
	return true, nil
}

func TestLARTouchedRowsReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"LU", "undecided", "empty", "cutoff", "stopped", "nil-propagator", "late-row", "hook-error"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, _ := lpLARTouchedRowsFixture(t, h, 3, false)
			p := s.NewBoundPropagator()
			bp := p
			limit := &lpTestResourceLimit{err: lpErrExactSolveCancel}
			s.Settings.SetResourceLimit(limit)
			want := error(nil)
			switch phase {
			case "LU":
				s.Settings.SimplexStrategy = StrategyLU
				bp = nil
			case "undecided":
				s.Settings.SimplexStrategy = StrategyUndecided
				bp = nil
			case "empty":
				h.must(s.RowsWithChangedBounds.Clear())
				bp = nil
			case "cutoff":
				s.Settings.MaxRowLengthForBoundPropagation = 2
				bp = nil
				limit.err = nil
			case "stopped":
				s.stopped = true
				want = ErrExactVectorPending
			case "nil-propagator":
				bp = nil
				want = ErrBounds
			case "late-row":
				limit.err = nil
				h.must(s.RowsWithChangedBounds.Clear())
				h.must(s.RowsWithChangedBounds.Resize(4))
				h.must(s.RowsWithChangedBounds.Insert(0))
				h.must(s.RowsWithChangedBounds.Insert(3))
				want = ErrBounds
			case "hook-error":
				limit.err = nil
				p.Interest = &lpTouchedInterest{failAt: 3}
				want = lpErrExactSolveCancel
			}
			before := append([]uint32(nil), s.RowsWithChangedBounds.Indices()...)
			err := s.PropagateBoundsForTouchedRows(bp)
			if err != want {
				t.Fatal(err, want)
			}
			if phase == "cutoff" {
				if limit.calls != 3 || !s.RowsWithChangedBounds.Empty() {
					t.Fatal("cutoff still reads cancellation per row")
				}
			} else if limit.calls != 0 && phase != "late-row" && phase != "hook-error" {
				t.Fatal("unreached cancellation", limit.calls)
			}
			if phase == "late-row" || phase == "hook-error" {
				if len(p.FoundBounds) != 2 || limit.calls != 1 {
					t.Fatal("late failure lost completed row", len(p.FoundBounds), limit.calls)
				}
			}
			if phase != "cutoff" && phase != "empty" && !reflect.DeepEqual(before, s.RowsWithChangedBounds.Indices()) {
				t.Fatal("early/failure changed touched rows")
			}
			h.must(s.Close())
			h.must(p.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

type lpTouchedModeLimit struct {
	s     *LARSolver
	calls int
}

func (l *lpTouchedModeLimit) CancelFlag() (bool, error) {
	l.calls++
	if l.calls == 3 {
		l.s.Settings.SimplexStrategy = StrategyLU
	}
	return false, nil
}

func TestLARTouchedRowsLiveModeAndSourceTermBody(t *testing.T) {
	for _, unused := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/%v", unused, direct), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, intervals := lpLARTouchedRowsFixture(t, h, 7, false)
				if unused {
					s.Settings.SimplexStrategy = StrategyUndecided
					v := NewSparseRationalVector(h.m)
					h.must(v.Append(0, &h.pair("1", "0").X))
					_, err := s.AddTerm(v.Data, &h.pair("5", "0").X)
					h.must(err)
					h.must(v.Close())
					s.Settings.SimplexStrategy = StrategyTableauRows
				}
				p := s.NewBoundPropagator()
				limit := &lpTouchedModeLimit{s: s}
				s.Settings.SetResourceLimit(limit)
				var err error
				if direct {
					err = s.PropagateBoundsOnTerms(nil)
				} else {
					err = s.PropagateBoundsForTouchedRows(p)
				}
				want := error(nil)
				if unused && sourceDebugEnabled {
					want = ErrAssertion
				}
				if err != want {
					t.Fatal("retained term body is assertion/native no-op", err, want)
				}
				if err != nil {
					if e := s.PropagateBoundsOnTerms(nil); e != ErrExactVectorPending {
						t.Fatal("stopped term propagation", e)
					}
				}
				if direct {
					if limit.calls != 0 || s.RowsWithChangedBounds.Empty() || len(p.FoundBounds) != 0 {
						t.Fatal("term-only body touched rows/propagator")
					}
				} else {
					if limit.calls != 3 || s.Settings.SimplexStrategy != StrategyLU || !s.RowsWithChangedBounds.Empty() {
						t.Fatal("mode dispatch after final clearing", limit.calls)
					}
					lpCheckTouchedIntervals(t, h, p, intervals, []uint32{0, 1, 2})
				}
				h.must(s.Close())
				h.must(p.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARTouchedRowsPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		done, failures, published := false, 0, 0
		for ordinal := int64(0); ordinal < 1800 && !done; ordinal++ {
			t.Run(fmt.Sprintf("%v/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				s, intervals := lpLARTouchedRowsFixture(t, h, 5, true)
				p := s.NewBoundPropagator()
				before := append([]uint32(nil), s.RowsWithChangedBounds.Indices()...)
				h.budget.FailAfter(ordinal)
				err := s.PropagateBoundsForTouchedRows(p)
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
					lpCheckTouchedIntervals(t, h, p, intervals, before)
					if !s.RowsWithChangedBounds.Empty() {
						t.Fatal("success retained marks")
					}
				} else {
					failures++
					if len(p.FoundBounds) > 0 {
						published++
					}
					if !lpExactPriorityPhysicalRefusal(err) || s.calls.pending() {
						t.Fatal("parent adopted child failure", err)
					}
					if !reflect.DeepEqual(before, s.RowsWithChangedBounds.Indices()) {
						t.Fatal("failure erased marks")
					}
					if e := s.PropagateBoundsForTouchedRows(p); e != ErrExactVectorPending {
						t.Fatal("stopped retry", e)
					}
				}
				h.must(s.Close())
				h.must(p.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !done || failures == 0 || published == 0 {
			t.Fatal("incomplete real failure campaign", direct, done, failures, published)
		}
	}
}

func TestLARTouchedRowsPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, _ := lpLARTouchedRowsFixture(t, h, 5, true)
			p := s.NewBoundPropagator()

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
				err = s.PropagateBoundsForTouchedRows(p)
				if err == nil || s.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(s.Close())
			h.must(p.Close())
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
