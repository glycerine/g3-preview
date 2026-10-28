package control

// Source: src/util/scoped_ctrl_c.h:24-41; src/util/scoped_ctrl_c.cpp:24-66.

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestScopedCtrlCReceivesProcessInterrupt(t *testing.T) {
	handled := make(chan EventCaller, 1)
	scope, err := NewScopedCtrlC(EventHandlerFunc(func(caller EventCaller) { handled <- caller }), false, true)
	if err != nil {
		t.Fatal(err)
	}
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case caller := <-handled:
		if caller != EventCallerControlC {
			t.Fatalf("caller = %d", caller)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("process interrupt was not delivered")
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScopedCtrlCOnceResetAndForwardState(t *testing.T) {
	var calls []EventCaller
	scope, err := NewScopedCtrlC(EventHandlerFunc(func(caller EventCaller) {
		calls = append(calls, caller)
	}), true, true)
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	if dispatchScopedControlC() || !reflect.DeepEqual(calls, []EventCaller{EventCallerControlC}) {
		t.Fatal("first interrupt", calls)
	}
	if !dispatchScopedControlC() || len(calls) != 1 {
		t.Fatal("second interrupt was not forwarded", calls)
	}
	if err := scope.Reset(); err != nil || dispatchScopedControlC() || len(calls) != 2 {
		t.Fatal("reset interrupt", err, calls)
	}
}

func TestScopedCtrlCRepeatedModeAndDisabledScope(t *testing.T) {
	calls := 0
	outer, err := NewScopedCtrlC(EventHandlerFunc(func(EventCaller) { calls++ }), false, true)
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := NewScopedCtrlC(EventHandlerFunc(func(EventCaller) { t.Fatal("disabled handler invoked") }), true, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if dispatchScopedControlC() {
			t.Fatal("repeated scope forwarded interrupt")
		}
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if err := disabled.Close(); err != nil {
		t.Fatal(err)
	}
	if err := outer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScopedCtrlCNestedLIFOAndRestoration(t *testing.T) {
	var calls []string
	outer, _ := NewScopedCtrlC(EventHandlerFunc(func(EventCaller) { calls = append(calls, "outer") }), false, true)
	inner, _ := NewScopedCtrlC(EventHandlerFunc(func(EventCaller) { calls = append(calls, "inner") }), false, true)
	if err := outer.Close(); !errors.Is(err, ErrPrecondition) {
		t.Fatalf("out-of-order close = %v", err)
	}
	if dispatchScopedControlC() || !reflect.DeepEqual(calls, []string{"inner"}) {
		t.Fatal("inner dispatch", calls)
	}
	if err := inner.Close(); err != nil {
		t.Fatal(err)
	}
	if dispatchScopedControlC() || !reflect.DeepEqual(calls, []string{"inner", "outer"}) {
		t.Fatal("outer restoration", calls)
	}
	if err := outer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScopedCtrlCHandlerAndOwnerPreconditions(t *testing.T) {
	if scope, err := NewScopedCtrlC(nil, true, true); scope != nil || !errors.Is(err, ErrPrecondition) {
		t.Fatalf("nil handler = %#v, %v", scope, err)
	}
	var nilFunction EventHandlerFunc
	if scope, err := NewScopedCtrlC(nilFunction, true, false); scope != nil || !errors.Is(err, ErrPrecondition) {
		t.Fatalf("nil function = %#v, %v", scope, err)
	}
	scope, _ := NewScopedCtrlC(EventHandlerFunc(func(EventCaller) {}), true, false)
	copyOfScope := *scope
	if !errors.Is(copyOfScope.Reset(), ErrOwner) || !errors.Is(copyOfScope.Close(), ErrOwner) {
		t.Fatal("copied owner accepted")
	}
	if err := scope.Close(); err != nil || !scope.Closed() {
		t.Fatal("close", err)
	}
	if err := scope.Close(); err != nil || !errors.Is(scope.Reset(), ErrOwner) {
		t.Fatal("closed scope behavior", err)
	}
}
