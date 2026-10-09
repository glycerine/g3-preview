// Source: src/util/rational.h; src/util/scoped_numeral.h;
// src/util/memory_manager.cpp:274-285.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// The signed-host accounting stop is an explicit guarded Go profile. The
// temporary is a real completed copy in Add's source body, not a forged frame.
func TestRationalServiceInvocationCompletedSourceStopAndOwnHistory(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 28)
	active, calls := false, 0
	var rt *Runtime
	var external Block
	var release error
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if active {
			active = false
			calls++
			contextTLSNearMinimum(t, rt)
			release = rt.Deallocate(&external)
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	a, e := s.NewValue()
	serviceMust(t, e)
	zero, e := s.NewValue()
	serviceMust(t, e)
	words := make([]uint32, 2000)
	for i := range words {
		words[i] = ^uint32(0)
	}
	serviceMust(t, s.Manager.Z.SetLimbs(&a.Raw.Num, words))
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	// First work-copy completes below the actual synchronization threshold;
	// the returned copy is the next reached allocator action.
	pad := int64(99990) - rt.State().LocalSize - 8 - 13000
	if pad <= 0 {
		t.Fatal("actual source prefix setup", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	var call RationalServiceInvocation
	out, body := s.AddWithInvocation(&a, &zero, &call)
	active = false
	if calls != 1 || !errors.Is(release, arena.ErrBounds) || external != (Block{}) {
		t.Fatal("actual callback frontier", calls, release)
	}
	if call.sourceCause == nil || !errors.Is(call.sourceCause, arena.ErrBounds) || !errors.Is(body, arena.ErrBounds) {
		t.Fatal("completed temporary source stop not reached", body, call.sourceCause, call.scope.State())
	}
	roots := []*Int{&a.Raw.Num, &a.Raw.Den, &zero.Raw.Num, &zero.Raw.Den, &out.Raw.Num, &out.Raw.Den}
	for i := uint8(0); i < call.scope.Count; i++ {
		roots = append(roots, &call.scope.Roots[i].Num, &call.scope.Roots[i].Den)
	}
	extra := []Block{held}
	if block := call.transferred.Block(); block != (Block{}) {
		extra = append(extra, block)
	}
	integerPhysicalAuditWithRaw(t, s.Manager.Z, extra, roots...)
	if want := s.HeaderBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z); b.Used() != want {
		t.Fatal("source-stop cold graph before recovery", b.Used(), want)
	}
	tuple, original, source := call.invocation, call.bodyCause, call.sourceCause
	if e := call.SourceReady(); e != source {
		t.Fatal("source retry changed sticky stop", e)
	}
	serviceMust(t, call.Cleanup(false))
	own := call.Cleanup(true)
	if !errors.Is(own, arena.ErrBounds) || call.ownCause != own {
		t.Fatal("real incomplete-child owning stop", own, call.ownCause, call.numeric, call.originalPending, call.originalCell, call.transferred.Block())
	}
	extra = []Block{held}
	if block := call.transferred.Block(); block != (Block{}) {
		extra = append(extra, block)
	}
	integerPhysicalAuditWithRaw(t, s.Manager.Z, extra, roots...)
	// A later actual inline invocation does not read or replace the retained
	// older child's source/own state, even before its owning recovery.
	var later RationalServiceInvocation
	three, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&three, 3))
	laterValue, e := s.ModWithInvocation(&zero, &three, &later)
	serviceMust(t, e)
	if laterValue.Raw.Num.Val != 0 || later.SourceReady() != nil {
		t.Fatal("source-valid later inline modulus", laterValue, later.SourceReady())
	}
	serviceMust(t, s.Del(&laterValue))
	serviceMust(t, s.Del(&three))
	if later.self != &later || later.invocation == call.invocation || call.invocation != tuple || call.bodyCause != original || call.sourceCause != source || call.ownCause != own {
		t.Fatal("independent later invocation overwrote old history")
	}
	serviceMust(t, later.Cleanup(false))
	serviceMust(t, later.Cleanup(true))
	serviceMust(t, later.EndContext())
	serviceMust(t, rt.SetMaxSize(0))
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return call.Cleanup(false) })
	if out.ownerID != 0 {
		contextTLSFinish(t, rt, func() error { return s.CleanupValue(&out) })
	}
	contextTLSFinish(t, rt, func() error { return call.Cleanup(true) })
	if call.invocation != tuple || call.bodyCause != original || call.sourceCause != source || call.ownCause != own || call.SourceReady() != source {
		t.Fatal("ACK replaced immutable source/body/first-own history")
	}
	serviceMust(t, call.EndContext())
	contextTLSFinish(t, rt, func() error { return s.CleanupValue(&a) })
	contextTLSFinish(t, rt, func() error { return s.CleanupValue(&zero) })
	contextTLSFinish(t, rt, s.CloseChecked)
	for _, p := range []*Block{&recovery, &held} {
		if e := rt.Deallocate(p); e != nil && (!errors.Is(e, arena.ErrBounds) || *p != (Block{})) {
			t.Fatal(e)
		}
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("actual enclosing owner end", b.Used())
	}
}
