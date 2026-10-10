package lp

import (
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/stopwatch"
)

type lpTestResourceLimit struct {
	cancel bool
	err    error
	calls  int
}

func (l *lpTestResourceLimit) CancelFlag() (bool, error) { l.calls++; return l.cancel, l.err }
func TestLPSettingsDefaultsAndServices(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.DebugOut != os.Stdout || s.MessageOut != os.Stdout || !s.watch.Running() {
		t.Fatal("source report streams/default watch")
	}
	for _, row := range []struct{ got, want float64 }{
		{s.PivotEpsilon, 1e-8}, {s.PositivePriceEpsilon, 1e-7}, {s.EnteringDiagEpsilon, 1e-8},
		{s.RefactorTolerance, 1e-4}, {s.PivotTolerance, 1e-6}, {s.ZeroTolerance, 1e-12}, {s.DropTolerance, 1e-14},
		{s.ToleranceForArtificials, 1e-4}, {s.CanBeTakenToBasisTolerance, 1e-5},
		{s.ScalingMaximum, 1}, {s.ScalingMinimum, 0.5}, {s.HarrisFeasibilityTolerance, 1e-7},
		{s.IgnoreEpsilonOfHarris, 1e-4}, {s.TimeLimit, math.MaxFloat64},
		{s.DualFeasibilityTolerance, 1e-7}, {s.PrimalFeasibilityTolerance, 1e-7},
		{s.RelativePrimalFeasibilityTolerance, 1e-9}, {s.DensityThreshold, 0.7},
	} {
		if row.got != row.want {
			t.Fatal(row)
		}
	}
	if s.RepsInScaler != 20 || s.PartialPivotingConstant != 10 || s.DepthOfRookSearch != 4 || s.PercentOfEnteringToCheck != 5 || s.MaxIterationsWithoutImprovements != 2000000 || s.MaxTotalIterations != 20000000 || s.ReportFrequency != 1000 || s.ColumnNormsUpdateFrequency != 12000 || s.MaxRowLengthForBoundPropagation != 300 || s.ColumnThresholdForLUInLAR != 4000 {
		t.Fatal("source integer defaults", s)
	}
	if !s.UsingPartialPivoting || !s.UseScaling || !s.BoundPropagation || !s.PresolveWithDoubleSolverForLAR || !s.ScaleWithRatio || !s.BackupCosts || s.PrintStatistics || s.UseBreakpointsInFeasibilitySearch {
		t.Fatal("source bool defaults", s)
	}
	if s.SimplexStrategy != StrategyTableauRows || !reflect.DeepEqual(s.Stats, Stats{}) {
		t.Fatal(s.SimplexStrategy, s.Stats)
	}
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts, StrategyLU, StrategyUndecided, 99} {
		s.SimplexStrategy = strategy
		if s.UseLU() != (strategy == StrategyLU) || s.UseTableau() != (strategy == StrategyTableauRows || strategy == StrategyTableauCosts) || s.UseTableauRows() != (strategy == StrategyTableauRows) {
			t.Fatal(strategy)
		}
	}
	for _, want := range []uint32{38, 7719, 21238, 2437, 8855} {
		if got := s.RandomNext(); got != want {
			t.Fatal(got, want)
		}
	}
	s.RandomSeed(^uint32(0))
	for _, want := range []uint32{35, 29739, 3374, 11141, 31308} {
		if got := s.RandomNext(); got != want {
			t.Fatal(got, want)
		}
	}
	s.Stats = Stats{1, 2, 3, 4, 5}
	s.Stats.Reset()
	if s.Stats != (Stats{}) {
		t.Fatal(s.Stats)
	}
	sentinel := errors.New("borrowed resource failure")
	limit := &lpTestResourceLimit{cancel: true, err: sentinel}
	s.SetResourceLimit(limit)
	if cancel, err := s.CancelFlag(); !cancel || err != sentinel || limit.calls != 1 {
		t.Fatal(cancel, err, limit.calls)
	}
	s.SetResourceLimit(nil)
	// Use the existing stopwatch's borrowed clock profile to test the exact CPU
	// elapsed > limit transition without sleeping or depending on host scheduling.
	ticks := []uint64{0, 1000000000, 1000000000, 1000000001, 1000000001}
	pos := 0
	s.watch = stopwatch.New(func() (uint64, error) { value := ticks[pos]; pos++; return value, nil })
	if err := s.watch.Start(); err != nil {
		t.Fatal(err)
	}
	s.TimeLimit = 1
	if cancel, err := s.CancelFlag(); cancel || err != nil {
		t.Fatal(cancel, err)
	}
	if cancel, err := s.CancelFlag(); !cancel || err != nil {
		t.Fatal(cancel, err)
	}

	reads := 0
	s.watch = stopwatch.New(func() (uint64, error) {
		reads++
		if reads > 1 {
			return 0, sentinel
		}
		return 0, nil
	})
	if err := s.watch.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CancelFlag(); err != sentinel {
		t.Fatal(err)
	}
	// Constructor uses the real host clock and has no fabricated failure hook.

}
func TestLPSettingsFloatToleranceEdges(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	predicates := []struct {
		f   func(float64) bool
		eps float64
	}{
		{s.FloatDualFeasibilitySmall, s.DualFeasibilityTolerance}, {s.FloatPrimalFeasibilitySmall, s.PrimalFeasibilityTolerance},
		{s.FloatCanBeTakenToBasisSmall, s.CanBeTakenToBasisTolerance}, {s.FloatDropSmall, s.DropTolerance},
		{s.FloatZeroSmall, s.ZeroTolerance}, {s.FloatRefactorSmall, s.RefactorTolerance}, {s.FloatPivotSmall, s.PivotTolerance},
		{s.FloatHarrisSmall, s.HarrisFeasibilityTolerance}, {s.FloatIgnoreHarrisSmall, s.IgnoreEpsilonOfHarris},
		{s.FloatArtificialSmall, s.ToleranceForArtificials},
	}
	for _, row := range predicates {
		for _, test := range []struct {
			value float64
			small bool
		}{
			{0, true}, {math.Copysign(0, -1), true}, {math.Nextafter(row.eps, 0), true}, {-math.Nextafter(row.eps, 0), true},
			{row.eps, false}, {-row.eps, false}, {math.NaN(), false}, {math.Inf(1), false}, {math.Inf(-1), false},
		} {
			if row.f(test.value) != test.small {
				t.Fatal(row.eps, test)
			}
		}
	}
	for _, eps := range []float64{0, -1, math.NaN()} {
		if FloatEpsilonSmall(0, eps) {
			t.Fatal(eps)
		}
	}
	s.DropTolerance = 1
	if !s.FloatDropSmall(0.5) || s.FloatDropSmall(1) {
		t.Fatal("live setting")
	}
}
