// Source: src/util/scoped_timer.h:24-30; src/util/scoped_timer.cpp:56-267.
package control

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestScopedTimerFiresWithTimeoutCaller(t *testing.T) {
	fired := make(chan EventCaller, 1)
	timer, err := NewScopedTimer(5, EventHandlerFunc(func(caller EventCaller) {
		fired <- caller
	}))
	serviceMust(t, err)
	defer func() { serviceMust(t, timer.Close()) }()
	select {
	case caller := <-fired:
		if caller != EventCallerTimeout {
			t.Fatal("wrong caller", caller)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timer did not fire")
	}
}

func TestScopedTimerDisabledIntervalsAndClose(t *testing.T) {
	for _, milliseconds := range []uint32{0, math.MaxUint32} {
		fired := make(chan EventCaller, 1)
		timer, err := NewScopedTimer(milliseconds, EventHandlerFunc(func(caller EventCaller) {
			fired <- caller
		}))
		serviceMust(t, err)
		serviceMust(t, timer.Close())
		serviceMust(t, timer.Close())
		select {
		case caller := <-fired:
			t.Fatal("disabled timer fired", milliseconds, caller)
		case <-time.After(20 * time.Millisecond):
		}
	}

	fired := make(chan EventCaller, 1)
	timer, err := NewScopedTimer(100, EventHandlerFunc(func(caller EventCaller) {
		fired <- caller
	}))
	serviceMust(t, err)
	serviceMust(t, timer.Close())
	select {
	case caller := <-fired:
		t.Fatal("closed timer fired", caller)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestScopedTimerAuthorityAndRunningCallbackClose(t *testing.T) {
	if timer, err := NewScopedTimer(1, nil); timer != nil || !errors.Is(err, ErrPrecondition) {
		t.Fatal("nil handler", timer, err)
	}
	var nilFunc EventHandlerFunc
	if timer, err := NewScopedTimer(1, nilFunc); timer != nil || !errors.Is(err, ErrPrecondition) {
		t.Fatal("typed nil handler", timer, err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	timer, err := NewScopedTimer(1, EventHandlerFunc(func(EventCaller) {
		close(entered)
		<-release
		close(done)
	}))
	serviceMust(t, err)
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not begin")
	}
	serviceMust(t, timer.Close())
	if !timer.Closed() {
		t.Fatal("closed state")
	}
	copy := ScopedTimer{self: timer}
	if !errors.Is(copy.Close(), ErrOwner) {
		t.Fatal("copied owner")
	}
	var absent *ScopedTimer
	if !errors.Is(absent.Close(), ErrOwner) {
		t.Fatal("nil owner")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("running callback did not finish")
	}

	selfClosed := make(chan error, 1)
	var self *ScopedTimer
	self, err = NewScopedTimer(20, EventHandlerFunc(func(EventCaller) {
		selfClosed <- self.Close()
	}))
	serviceMust(t, err)
	select {
	case err = <-selfClosed:
		serviceMust(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("handler-driven close deadlocked")
	}
}
