// Source: src/util/stopwatch.h. Licensed under the MIT license.
package util

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
