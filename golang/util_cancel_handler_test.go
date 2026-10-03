package g3

import "testing"

// event_handler/cancel_eh have no dedicated source test suite. These added
// expectations freeze constructor, first-caller and nested destructor behavior.
func TestCancelHandlerSourceLifetime(t *testing.T) {
	var r utilReslimit
	defer r.close()
	h := newUtilCancelHandler(&r)
	if h.canceled() || h.callerID() != utilUnsetCaller {
		t.Fatal("initial state")
	}
	h.invoke(utilTimeoutCaller)
	h.invoke(utilAPIInterruptCaller)
	if !h.canceled() || h.callerID() != utilTimeoutCaller || r.cancelCount.Load() != 1 {
		t.Fatal("first invocation only")
	}
	h.close()
	h.close()
	if r.cancelFlag() || !h.canceled() || h.callerID() != utilTimeoutCaller {
		t.Fatal("one decrement, retained source fields")
	}
	quiet := newUtilCancelHandler(&r)
	r.cancel()
	quiet.close()
	if !r.cancelFlag() {
		t.Fatal("never invoked destructor must not decrement")
	}
	r.resetCancel()
}

func TestCancelHandlerSourceNestedAndReset(t *testing.T) {
	var r, child utilReslimit
	defer r.close()
	defer child.close()
	if err := r.pushChild(&child); err != nil {
		t.Fatal(err)
	}
	a, b := newUtilCancelHandler(&r), newUtilCancelHandler(&r)
	a.invoke(utilControlCCaller)
	b.invoke(utilResourceLimitCaller)
	if child.cancelCount.Load() != 2 {
		t.Fatal("nested propagation")
	}
	a.close()
	if child.cancelCount.Load() != 1 || b.callerID() != utilResourceLimitCaller {
		t.Fatal("nested cleanup")
	}
	r.resetCancel()
	b.close()
	if child.cancelFlag() || r.cancelFlag() {
		t.Fatal("source saturating decrement after reset")
	}
	// IDs retain the source enum's exact numeric values.
	if utilUnsetCaller != 0 || utilControlCCaller != 1 || utilTimeoutCaller != 2 || utilResourceLimitCaller != 3 || utilAPIInterruptCaller != 4 {
		t.Fatal("caller IDs")
	}
}

func TestCancelHandlerWarmNoAllocation(t *testing.T) {
	var r utilReslimit
	defer r.close()
	if a := testing.AllocsPerRun(100, func() {
		h := newUtilCancelHandler(&r)
		h.invoke(utilAPIInterruptCaller)
		h.invoke(utilTimeoutCaller)
		h.close()
	}); a != 0 {
		t.Fatal(a)
	}
}
