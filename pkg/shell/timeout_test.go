package shell

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestShellTimeoutSourceCallbackOutputAndErrorOrder(t *testing.T) {
	var output bytes.Buffer
	s := NewTimeoutService(&output)
	defer func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	var callbackOutput string
	var callbackCaller control.EventCaller
	if err := s.RegisterOnTimeoutProc(func() {
		callbackOutput = output.String()
		callbackCaller = s.Caller()
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTimeout(5); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-s.Events():
		var timeout z3exception.Z3Error
		if !errors.As(err, &timeout) || timeout.ErrorCode() != z3exception.ErrorTimeout {
			t.Fatal("source timeout exception", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("source timer did not fire")
	}
	if callbackOutput != "timeout\n" || output.String() != "timeout\n" || callbackCaller != control.EventCallerTimeout {
		t.Fatal("source print/caller/callback order", callbackOutput, output.String(), callbackCaller)
	}
	s.mu.Lock()
	active := s.timer != nil
	s.mu.Unlock()
	if active {
		t.Fatal("fired global timer still owned")
	}
}

func TestShellTimeoutReplacementDisabledAndBorrowedCallback(t *testing.T) {
	var output bytes.Buffer
	s := NewTimeoutService(&output)
	defer func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	called := 0
	if err := s.RegisterOnTimeoutProc(func() { called++ }); err != nil {
		t.Fatal(err)
	}
	for _, milliseconds := range []int64{0, -1, int64(math.MaxUint32)} {
		if err := s.SetTimeout(milliseconds); err != nil {
			t.Fatal(err)
		}
		s.mu.Lock()
		owned := s.timer != nil
		s.mu.Unlock()
		if !owned {
			t.Fatal("source disabled interval still owns its scoped timer", milliseconds)
		}
	}
	if err := s.SetTimeout(250); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTimeout(5); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterOnTimeoutProc(nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("replacement timer did not fire")
	}
	if called != 0 || output.String() != "timeout\n" {
		t.Fatal("replaced callback or timer fired", called, output.String())
	}
	select {
	case err := <-s.Events():
		t.Fatal("replaced timer fired later", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := s.SetTimeout(100); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTimeout(1); !errors.Is(err, control.ErrOwner) {
		t.Fatal("closed global timeout owner", err)
	}
}

func TestShellTimeoutCallbackRearmIsDeletedBeforeError(t *testing.T) {
	s := NewTimeoutService(&bytes.Buffer{})
	defer func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	called := make(chan error, 2)
	if err := s.RegisterOnTimeoutProc(func() { called <- s.SetTimeout(5) }); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTimeout(5); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-called:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("source handler did not enter callback")
	}
	select {
	case <-s.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("source handler did not deliver timeout error")
	}
	select {
	case err := <-called:
		t.Fatal("callback's replacement timer survived source deletion", err)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestShellTimeoutFirstUnhandledErrorDoesNotBlockRearm(t *testing.T) {
	s := NewTimeoutService(&bytes.Buffer{})
	defer func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	called := make(chan struct{}, 2)
	if err := s.RegisterOnTimeoutProc(func() { called <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.SetTimeout(5); err != nil {
			t.Fatal(err)
		}
		select {
		case <-called:
		case <-time.After(2 * time.Second):
			t.Fatal("timer callback blocked")
		}
		deadline := time.After(2 * time.Second)
		for {
			s.mu.Lock()
			idle := s.timer == nil
			s.mu.Unlock()
			if idle {
				break
			}
			select {
			case <-deadline:
				t.Fatal("timer callback failed to clear owner")
			case <-time.After(time.Millisecond):
			}
		}
	}
	if len(s.events) != 1 {
		t.Fatal("first unhandled source timeout was not retained", len(s.events))
	}
}
