package stopwatch

// Source: src/util/stopwatch.h.

import (
	"errors"
	"testing"
)

// Added independent finite clock profile; src/util/stopwatch.h has no driver.
func TestStopwatchSourceControls(t *testing.T) {
	samples := []uint64{100, 250, 400, 600, 750, 850, 1000}
	i := 0
	w := Stopwatch{clock: func() (uint64, error) { v := samples[i]; i++; return v, nil }}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	if i != 1 {
		t.Fatal("start must be idempotent")
	}
	if e := w.Stop(); e != nil {
		t.Fatal(e)
	}
	if e := w.Stop(); e != nil {
		t.Fatal(e)
	}
	if w.ElapsedNanoseconds() != 150 || i != 2 {
		t.Fatal(w, i)
	}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	w.Reset() // source resets only elapsed, leaving the old start while running.
	if e := w.Stop(); e != nil {
		t.Fatal(e)
	}
	if w.ElapsedNanoseconds() != 200 {
		t.Fatal(w.ElapsedNanoseconds())
	}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	v, e := w.Seconds()
	if e != nil || v != 300.0/1e9 || i != 7 || !w.Running() {
		t.Fatal(v, e, i)
	}
	other := Stopwatch{elapsed: 17}
	w.Add(&other)
	if w.ElapsedNanoseconds() != 317 {
		t.Fatal(w.ElapsedNanoseconds())
	}
}
func TestStopwatchActualClock(t *testing.T) {
	var w Stopwatch
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	// Actual clock path, not a timing threshold susceptible to scheduler noise.
	if e := w.Stop(); e != nil {
		t.Fatal(e)
	}
	if v, e := w.Seconds(); e != nil || v < 0 {
		t.Fatal(v, e)
	}
}

// Go clock-error adapter confirmations; C++ ignores clock_gettime's return.
func TestStopwatchClockFailurePrefixes(t *testing.T) {
	fail := errors.New("finite clock refusal")
	w := Stopwatch{clock: func() (uint64, error) { return 0, fail }}
	if e := w.Start(); !errors.Is(e, fail) || w.Running() || w.ElapsedNanoseconds() != 0 {
		t.Fatal(e, w)
	}
	i := 0
	w = Stopwatch{clock: func() (uint64, error) {
		i++
		if i == 1 {
			return 100, nil
		}
		return 0, fail
	}}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	if e := w.Stop(); !errors.Is(e, fail) || !w.Running() || w.start != 100 || w.elapsed != 0 {
		t.Fatal(e, w)
	}
	i = 0
	w = Stopwatch{clock: func() (uint64, error) {
		i++
		switch i {
		case 1:
			return 100, nil
		case 2:
			return 200, nil
		}
		return 0, fail
	}}
	if e := w.Start(); e != nil {
		t.Fatal(e)
	}
	if _, e := w.Seconds(); !errors.Is(e, fail) || w.Running() || w.ElapsedNanoseconds() != 100 {
		t.Fatal(e, w)
	}
}
func TestStopwatchUnsignedBorrowGuard(t *testing.T) {
	w := Stopwatch{start: 900, running: true, clock: func() (uint64, error) { return 800, nil }}
	if e := w.Stop(); e != nil || w.ElapsedNanoseconds() != 0 {
		t.Fatal(e, w)
	}
	// With nonzero accumulated time, source enters the unsigned subtraction.
	w = Stopwatch{elapsed: 50, start: 900, running: true, clock: func() (uint64, error) { return 800, nil }}
	if e := w.Stop(); e != nil || w.ElapsedNanoseconds() != ^uint64(49) {
		t.Fatal(e, w)
	}
}

func TestStopwatchNamedQueriesAndScopedWatch(t *testing.T) {
	samples := []uint64{100, 250, 400, 700, 900}
	index := 0
	exhausted := errors.New("clock samples exhausted")
	watch := Stopwatch{elapsed: 25, clock: func() (uint64, error) {
		if index == len(samples) {
			return 0, exhausted
		}
		value := samples[index]
		index++
		return value, nil
	}}
	scope, err := NewScopedWatch(&watch, true)
	if err != nil {
		t.Fatal(err)
	}
	if watch.ElapsedNanoseconds() != 0 || !watch.Running() || index != 1 {
		t.Fatal("scoped constructor", watch, index)
	}
	if err = scope.Close(); err != nil {
		t.Fatal(err)
	}
	if watch.ElapsedNanoseconds() != 150 || watch.Running() || index != 2 {
		t.Fatal("scoped close", watch, index)
	}
	if err = scope.Close(); err != nil || index != 2 {
		t.Fatal("repeated close", err, index)
	}

	if err = watch.Start(); err != nil {
		t.Fatal(err)
	}
	seconds, err := watch.GetSeconds()
	if err != nil || seconds != 450.0/1e9 || !watch.Running() || index != 5 {
		t.Fatal(seconds, err, watch, index)
	}
	current, err := watch.GetCurrentSeconds()
	if !errors.Is(err, exhausted) || current != 0 {
		// The finite clock is exhausted, proving this alias performs the same
		// running stop/start reads instead of returning cached state.
		t.Fatal(current, err)
	}
}

func TestScopedWatchPreconditionsAndFailure(t *testing.T) {
	if scope, err := NewScopedWatch(nil, false); scope != (ScopedWatch{}) || !errors.Is(err, ErrStopwatchOwner) {
		t.Fatal(scope, err)
	}
	var nilScope *ScopedWatch
	if err := nilScope.Close(); !errors.Is(err, ErrStopwatchOwner) {
		t.Fatal(err)
	}
	failure := errors.New("clock unavailable")
	watch := Stopwatch{elapsed: 9, clock: func() (uint64, error) { return 0, failure }}
	if scope, err := NewScopedWatch(&watch, true); scope != (ScopedWatch{}) || !errors.Is(err, failure) || watch.ElapsedNanoseconds() != 0 || watch.Running() {
		t.Fatal(scope, err, watch)
	}

	reads := 0
	watch = Stopwatch{clock: func() (uint64, error) {
		reads++
		if reads == 1 {
			return 10, nil
		}
		return 0, failure
	}}
	scope, err := NewScopedWatch(&watch, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = scope.Close(); !errors.Is(err, failure) || !scope.active || !watch.Running() {
		t.Fatal(err, scope, watch)
	}
	watch.clock = func() (uint64, error) { return 20, nil }
	if err = scope.Close(); err != nil || scope.active || watch.Running() || watch.ElapsedNanoseconds() != 10 {
		t.Fatal(err, scope, watch)
	}
}
