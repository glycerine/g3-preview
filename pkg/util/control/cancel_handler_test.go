package control

import (
	"testing"
)

// event_handler/cancel_eh have no dedicated source test suite. These added
// expectations freeze constructor, first-caller and nested destructor behavior.
func TestCancelHandlerSourceLifetime(t *testing.T) {
	var r Reslimit
	defer r.Close()
	h := newUtilCancelHandler(&r)
	if h.canceled() || h.callerID() != utilUnsetCaller {
		t.Fatal("initial state")
	}
	h.invoke(utilTimeoutCaller)
	h.invoke(utilAPIInterruptCaller)
	if !h.canceled() || h.callerID() != utilTimeoutCaller || r.CancelCount() != 1 {
		t.Fatal("first invocation only")
	}
	h.close()
	h.close()
	if r.CancelFlag() || !h.canceled() || h.callerID() != utilTimeoutCaller {
		t.Fatal("one decrement, retained source fields")
	}
	quiet := newUtilCancelHandler(&r)
	r.Cancel()
	quiet.close()
	if !r.CancelFlag() {
		t.Fatal("never invoked destructor must not decrement")
	}
	r.ResetCancel()
}

func TestCancelHandlerSourceNestedAndReset(t *testing.T) {
	var r, child Reslimit
	defer r.Close()
	defer child.Close()
	if err := r.PushChild(&child); err != nil {
		t.Fatal(err)
	}
	a, b := newUtilCancelHandler(&r), newUtilCancelHandler(&r)
	a.invoke(utilControlCCaller)
	b.invoke(utilResourceLimitCaller)
	if child.CancelCount() != 2 {
		t.Fatal("nested propagation")
	}
	a.close()
	if child.CancelCount() != 1 || b.callerID() != utilResourceLimitCaller {
		t.Fatal("nested cleanup")
	}
	r.ResetCancel()
	b.close()
	if child.CancelFlag() || r.CancelFlag() {
		t.Fatal("source saturating decrement after reset")
	}
	// IDs retain the source enum's exact numeric values.
	if utilUnsetCaller != 0 || utilControlCCaller != 1 || utilTimeoutCaller != 2 || utilResourceLimitCaller != 3 || utilAPIInterruptCaller != 4 {
		t.Fatal("caller IDs")
	}
}

func TestCancelHandlerWarmNoAllocation(t *testing.T) {
	var r Reslimit
	defer r.Close()
	if a := testing.AllocsPerRun(100, func() {
		h := newUtilCancelHandler(&r)
		h.invoke(utilAPIInterruptCaller)
		h.invoke(utilTimeoutCaller)
		h.close()
	}); a != 0 {
		t.Fatal(a)
	}
}
