package lp

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/stopwatch"
)

func TestLPSettingsBorrowedLimitAndContinuingDefaultTimer(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	var now uint64
	reads := 0
	s.watch = stopwatch.New(func() (uint64, error) { reads++; return now, nil })
	if err := s.watch.Start(); err != nil {
		t.Fatal(err)
	}
	s.TimeLimit = 2
	first := &lpTestResourceLimit{}
	second := &lpTestResourceLimit{cancel: true, err: errors.New("borrowed cancellation failure")}
	s.SetResourceLimit(first)
	now = 3_000_000_000 // Stopwatch clock values are nanoseconds.
	if cancel, err := s.CancelFlag(); cancel || err != nil || reads != 1 || first.calls != 1 {
		t.Fatal("borrowed limit must replace clock dispatch", cancel, err, reads, first.calls)
	}
	// Borrowed provider mutations are observed; neither SetResourceLimit nor
	// cancellation copies, closes or restarts the default watch.
	first.cancel = true
	if cancel, err := s.CancelFlag(); !cancel || err != nil || reads != 1 || first.calls != 2 {
		t.Fatal("live borrowed provider", cancel, err, reads, first.calls)
	}
	s.SetResourceLimit(second)
	if cancel, err := s.CancelFlag(); !cancel || err != second.err || reads != 1 || second.calls != 1 || first.calls != 2 {
		t.Fatal("replacement provider/error transport", cancel, err, reads)
	}
	// nil is the documented Go selection of the existing default service.
	s.SetResourceLimit(nil)
	// A running stopwatch query reads once for Stop and once for Start.
	if cancel, err := s.CancelFlag(); !cancel || err != nil || reads != 3 {
		t.Fatal("default timer was restarted or paused", cancel, err, reads)
	}
	for _, tc := range []struct {
		limit float64
		want  bool
	}{
		{3, false}, {math.Nextafter(3, 0), true}, {math.Nextafter(3, math.Inf(1)), false},
		{0, true}, {-1, true}, {math.Inf(1), false}, {math.Inf(-1), true}, {math.NaN(), false},
	} {
		s.TimeLimit = tc.limit
		if cancel, err := s.CancelFlag(); cancel != tc.want || err != nil {
			t.Fatal("strict live time limit", tc.limit, cancel, err)
		}
	}
	if first.calls != 2 || second.calls != 1 {
		t.Fatal("default cancellation retained an old provider")
	}
}

func TestLPSettingsIndependentRandomOwnersAndReseeding(t *testing.T) {
	a, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []uint32{0, 1, 17, 723, 1 << 31, ^uint32(0)} {
		a.RandomSeed(seed)
		b.RandomSeed(seed ^ 0x12345678)
		x, y := seed, seed^0x12345678
		for step := 0; step < 200; step++ {
			x = x*214013 + 2531011
			if got := a.RandomNext(); got != (x>>16)&32767 {
				t.Fatal("first source RNG recurrence", seed, step, got, x)
			}
			if step%3 == 0 {
				// Mutating another setting owner must not reset a's generator.
				b.RandomSeed(uint32(step))
				y = uint32(step)
			}
			y = y*214013 + 2531011
			if got := b.RandomNext(); got != (y>>16)&32767 {
				t.Fatal("second source RNG recurrence", seed, step, got, y)
			}
		}
	}
}

func TestLPSettingsDefaultTimerRestartFailureRetainsElapsed(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("restart clock failure")
	reads := 0
	s.watch = stopwatch.New(func() (uint64, error) {
		reads++
		switch reads {
		case 1:
			return 0, nil
		case 2:
			return 3_000_000_000, nil
		default:
			return 0, sentinel
		}
	})
	if err := s.watch.Start(); err != nil {
		t.Fatal(err)
	}
	s.TimeLimit = 2
	if _, err := s.CancelFlag(); err != sentinel || reads != 3 {
		t.Fatal("restart error transport", err, reads)
	}
	if s.watch.Running() || s.watch.ElapsedNanoseconds() != 3_000_000_000 {
		t.Fatal("completed stop lost its elapsed prefix", s.watch.State())
	}
	if cancel, err := s.CancelFlag(); !cancel || err != nil || reads != 3 {
		t.Fatal("stopped timer should expose retained elapsed time", cancel, err, reads)
	}
}

func TestLPSettingsAllToleranceFieldsRemainLive(t *testing.T) {
	s, err := NewSettings()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		eps   *float64
		small func(float64) bool
	}{
		{&s.DualFeasibilityTolerance, s.FloatDualFeasibilitySmall},
		{&s.PrimalFeasibilityTolerance, s.FloatPrimalFeasibilitySmall},
		{&s.CanBeTakenToBasisTolerance, s.FloatCanBeTakenToBasisSmall},
		{&s.DropTolerance, s.FloatDropSmall}, {&s.ZeroTolerance, s.FloatZeroSmall},
		{&s.RefactorTolerance, s.FloatRefactorSmall}, {&s.PivotTolerance, s.FloatPivotSmall},
		{&s.HarrisFeasibilityTolerance, s.FloatHarrisSmall},
		{&s.IgnoreEpsilonOfHarris, s.FloatIgnoreHarrisSmall},
		{&s.ToleranceForArtificials, s.FloatArtificialSmall},
	} {
		for _, eps := range []float64{0, -1, math.SmallestNonzeroFloat64, 0.125, 2, math.MaxFloat64, math.Inf(1), math.NaN()} {
			*tc.eps = eps
			for _, value := range []float64{0, math.Copysign(0, -1), eps, -eps, math.Nextafter(eps, 0), -math.Nextafter(eps, 0), 1, -1, math.Inf(1), math.Inf(-1), math.NaN()} {
				if got, want := tc.small(value), math.Abs(value) < eps; got != want {
					t.Fatal("live symmetric strict interval", value, eps, got, want)
				}
			}
		}
	}
}
