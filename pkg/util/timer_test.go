// Source: src/util/timer.h:27-36; src/util/timer.cpp:24-39.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestTimerOriginalStartQueriesAndStrictTimeouts(t *testing.T) {
	r, _ := memoryRuntimeFixture(t, false, 64)
	var now uint64 = 10_000_000_000
	reads := 0
	p, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) { reads++; return now, nil }})
	memoryRuntimeRequire(t, e)
	defer func() { memoryRuntimeRequire(t, p.Close()) }()
	if reads != 1 || r.Stats().LiveBlocks != 1 {
		t.Fatal("constructor allocation/start", reads, r.Stats())
	}
	n, e := r.Size(p.watch)
	memoryRuntimeRequire(t, e)
	if n != 32 {
		t.Fatal("Linux stopwatch extent", n)
	}
	memoryRuntimeRequire(t, p.Start())
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
	w, e := p.readWatch()
	memoryRuntimeRequire(t, e)
	if !w.Running() || w.ElapsedNanoseconds() != 2_000_000_001 || w.start != now {
		t.Fatal(w)
	}
	q, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) { return now, nil }})
	memoryRuntimeRequire(t, e)
	now += 1_000_000
	if yes, e := q.MSTimeout(1); e != nil || yes {
		t.Fatal("strict milliseconds boundary", yes, e)
	}
	now++
	if yes, e := q.MSTimeout(1); e != nil || !yes {
		t.Fatal("milliseconds after boundary", yes, e)
	}
	memoryRuntimeRequire(t, q.Close())
	copy := *p
	if copy.Start() != ErrTimerOwner || copy.Close() != ErrTimerOwner {
		t.Fatal("copied owner")
	}
	memoryRuntimeRequire(t, p.Close())
	memoryRuntimeRequire(t, p.Close())
	if p.Start() != arena.ErrClosed {
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
	r, _ := memoryRuntimeFixture(t, false, 64)
	failure := errors.New("timer clock refusal")
	for _, refuse := range []int{1, 2, 3} {
		calls := 0
		p, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) {
			calls++
			if calls == refuse {
				return 0, failure
			}
			return uint64(calls) * 100, nil
		}})
		if refuse == 1 {
			if !errors.Is(e, failure) || p == nil {
				t.Fatal("retained constructor prefix", p, e)
			}
			if p.Start() != failure {
				t.Fatal("stopped constructor replay")
			}
		} else {
			memoryRuntimeRequire(t, e)
			if _, e = p.GetSeconds(); !errors.Is(e, failure) {
				t.Fatal(e)
			}
			w, e := p.readWatch()
			memoryRuntimeRequire(t, e)
			if refuse == 2 && (!w.Running() || w.ElapsedNanoseconds() != 0 || w.start != 100) {
				t.Fatal("stop failure prefix", w)
			}
			if refuse == 3 && (w.Running() || w.ElapsedNanoseconds() != 100) {
				t.Fatal("restart failure prefix", w)
			}
		}
		memoryRuntimeRequire(t, p.Close())
		if r.Stats().LiveBlocks != 0 {
			t.Fatal("retained stopwatch block")
		}
	}
	var p *Timer
	calls := 0
	p, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) {
		calls++
		if p != nil {
			if p.Start() != ErrTimerPending || p.Close() != ErrTimerPending {
				t.Fatal("clock reentry")
			}
			w, e := p.readWatch()
			memoryRuntimeRequire(t, e)
			if calls == 2 && (!w.Running() || w.elapsed != 0) {
				t.Fatal("stop clock prefix", w)
			}
			if calls == 3 && (w.Running() || w.elapsed != 100) {
				t.Fatal("restart clock prefix", w)
			}
		}
		return uint64(calls) * 100, nil
	}})
	memoryRuntimeRequire(t, e)
	_, e = p.GetSeconds()
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, p.Close())
}

func TestTimerEveryColdConstructorReservation(t *testing.T) {
	for _, tls := range []bool{false, true} {
		completed := false
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			r, b := memoryRuntimeFixture(t, tls, 64)
			b.FailAfter(ordinal)
			calls := 0
			p, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) { calls++; return 100, nil }})
			ownerBytes := uint64(0)
			if p != nil {
				ownerBytes = p.header
			}
			if b.Used() != r.Stats().TotalBytes+ownerBytes {
				t.Fatal("immediate constructor graph", ordinal, b.Used(), r.Stats())
			}
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				var memoryFailure Failure
				if !errors.As(e, &memoryFailure) || memoryFailure.Kind != FailureMemout {
					t.Fatal(e)
				}
			}
			if e != nil && calls != 0 {
				t.Fatal("clock before allocation completes")
			}
			if e == nil && (calls != 1 || r.Stats().LiveBlocks != 1) {
				t.Fatal(calls, r.Stats())
			}
			b.FailAfter(-1)
			if p != nil {
				memoryRuntimeRequire(t, p.Close())
			}
			if r.Stats().LiveBlocks != 0 {
				t.Fatal("constructor cleanup", ordinal)
			}
			memoryRuntimeRequire(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("budget refund", ordinal, b.Used())
			}
			if e == nil {
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("missing constructor completion horizon")
		}
	}
}

func TestTimerRuntimeLimitRetainsReachedAllocationPrefix(t *testing.T) {
	for _, tls := range []bool{false, true} {
		r, b := memoryRuntimeFixture(t, tls, 64)
		var preload Block
		if tls {
			var e error
			preload, e = r.Allocate(99960)
			memoryRuntimeRequire(t, e)
		}
		memoryRuntimeRequire(t, r.SetMaxSize(1))
		calls := 0
		p, e := NewTimer(TimerConfig{Runtime: r, Clock: func() (uint64, error) { calls++; return 100, nil }})
		memoryRuntimeCode(t, e, 101)
		if p == nil || p.Start() != e || calls != 0 {
			t.Fatal("failure before stopwatch construction", p, e, calls)
		}
		if tls {
			if p.watch == (Block{}) || r.Stats().LiveBlocks != 2 {
				t.Fatal("TLS allocation returned ownership", r.Stats())
			}
			if _, e = r.Read(p.watch, 0); e != ErrRuntimeUninitialized {
				t.Fatal("unreached stopwatch initialization", e)
			}
		} else if p.watch != (Block{}) || r.Stats().LiveBlocks != 0 {
			t.Fatal("non-TLS failure precedes allocation", r.Stats())
		}
		if b.Used() != r.Stats().TotalBytes+p.header {
			t.Fatal("limit failure graph")
		}
		memoryRuntimeRequire(t, p.Close())
		if preload != (Block{}) {
			memoryRuntimeRequire(t, r.Deallocate(&preload))
		}
		if r.Stats().LiveBlocks != 0 {
			t.Fatal("retained blocks")
		}
	}
}

func TestTimerActualProcessClock(t *testing.T) {
	r, _ := memoryRuntimeFixture(t, false, 64)
	p, e := NewTimer(TimerConfig{Runtime: r})
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, p.Start())
	seconds, e := p.GetSeconds()
	memoryRuntimeRequire(t, e)
	if seconds < 0 {
		t.Fatal(seconds)
	}
	memoryRuntimeRequire(t, p.Close())
	if _, e = NewTimer(TimerConfig{}); e != ErrTimerOwner {
		t.Fatal(e)
	}
}
