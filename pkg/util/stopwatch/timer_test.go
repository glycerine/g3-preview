// Source: src/util/timer.h:27-36; src/util/timer.cpp:24-39.
package stopwatch

import (
	"errors"
	"testing"
)

func TestTimerOriginalStartQueriesAndStrictTimeouts(t *testing.T) {
	var now uint64 = 10_000_000_000
	reads := 0
	p, e := NewTimer(func() (uint64, error) { reads++; return now, nil })
	timerMust(t, e)
	defer func() { timerMust(t, p.Close()) }()
	if reads != 1 {
		t.Fatal("constructor start", reads)
	}
	timerMust(t, p.Start())
	if reads != 1 {
		t.Fatal("start while running reads clock")
	}
	for _, v := range []uint32{0, ^uint32(0)} {
		if yes, e := p.Timeout(v); e != nil || yes {
			t.Fatal(yes, e)
		}
		if yes, e := p.MSTimeout(v); e != nil || yes {
			t.Fatal(yes, e)
		}
	}
	if reads != 1 {
		t.Fatal("disabled timeouts read clock")
	}
	now += 2_000_000_000
	if yes, e := p.Timeout(2); e != nil || yes {
		t.Fatal("strict seconds boundary", yes, e)
	}
	now++
	if yes, e := p.Timeout(2); e != nil || !yes {
		t.Fatal("seconds after boundary", yes, e)
	}
	if seconds, e := p.GetSeconds(); e != nil || seconds != 2.000000001 {
		t.Fatal(seconds, e)
	}
	// Each query performs stop then start, retaining accumulated time.
	if reads != 7 {
		t.Fatal("query clock reads", reads)
	}
	w := p.watch
	if !w.Running() || w.ElapsedNanoseconds() != 2_000_000_001 || w.State().Start != now {
		t.Fatal(w)
	}
	q, e := NewTimer(func() (uint64, error) { return now, nil })
	timerMust(t, e)
	now += 1_000_000
	if yes, e := q.MSTimeout(1); e != nil || yes {
		t.Fatal("strict milliseconds boundary", yes, e)
	}
	now++
	if yes, e := q.MSTimeout(1); e != nil || !yes {
		t.Fatal("milliseconds after boundary", yes, e)
	}
	timerMust(t, q.Close())
	copy := *p
	if copy.Start() != ErrTimerOwner || copy.Close() != ErrTimerOwner {
		t.Fatal("copied owner")
	}
	timerMust(t, p.Close())
	timerMust(t, p.Close())
	if p.Start() != ErrTimerClosed {
		t.Fatal("closed timer")
	}
	var absent *Timer
	if absent.Start() != ErrTimerOwner {
		t.Fatal("nil timer")
	}
	if yes, e := absent.Timeout(0); yes || e != nil {
		t.Fatal("short-circuit reads absent owner")
	}
}

func TestTimerClockFailureAndReentryPrefixes(t *testing.T) {
	failure := errors.New("timer clock refusal")
	for _, refuse := range []int{1, 2, 3} {
		calls := 0
		p, e := NewTimer(func() (uint64, error) {
			calls++
			if calls == refuse {
				return 0, failure
			}
			return uint64(calls) * 100, nil
		})
		if refuse == 1 {
			if !errors.Is(e, failure) || p == nil {
				t.Fatal("retained constructor prefix", p, e)
			}
			if p.Start() != failure {
				t.Fatal("stopped constructor replay")
			}
		} else {
			timerMust(t, e)
			if _, e = p.GetSeconds(); !errors.Is(e, failure) {
				t.Fatal(e)
			}
			w := p.watch
			if refuse == 2 && (!w.Running() || w.ElapsedNanoseconds() != 0 || w.State().Start != 100) {
				t.Fatal("stop failure prefix", w)
			}
			if refuse == 3 && (w.Running() || w.ElapsedNanoseconds() != 100) {
				t.Fatal("restart failure prefix", w)
			}
		}
		timerMust(t, p.Close())
	}
	var p *Timer
	calls := 0
	p, e := NewTimer(func() (uint64, error) {
		calls++
		if p != nil {
			if p.Start() != ErrTimerPending || p.Close() != ErrTimerPending {
				t.Fatal("clock reentry")
			}
			w := p.watch
			if calls == 2 && (!w.Running() || w.ElapsedNanoseconds() != 0) {
				t.Fatal("stop clock prefix", w)
			}
			if calls == 3 && (w.Running() || w.ElapsedNanoseconds() != 100) {
				t.Fatal("restart clock prefix", w)
			}
		}
		return uint64(calls) * 100, nil
	})
	timerMust(t, e)
	_, e = p.GetSeconds()
	timerMust(t, e)
	timerMust(t, p.Close())
}

func TestTimerActualProcessClock(t *testing.T) {
	p, e := NewTimer(nil)
	timerMust(t, e)
	timerMust(t, p.Start())
	seconds, e := p.GetSeconds()
	timerMust(t, e)
	if seconds < 0 {
		t.Fatal(seconds)
	}
	timerMust(t, p.Close())
}

func timerMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
