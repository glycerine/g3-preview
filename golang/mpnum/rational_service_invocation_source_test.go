// Source: src/util/rational.cpp:50-62; src/util/rational.h; src/util/scoped_numeral.h.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestRationalServiceInvocationInlineHistoryAndExactOwner(t *testing.T) {
	s, b := serviceNew(t)
	a := serviceValue(t, s, "17")
	d := serviceValue(t, s, "8")
	warm, e := s.PowerOfTwo(2)
	serviceMust(t, e)
	serviceMust(t, s.Del(&warm))
	var first, second RationalServiceInvocation
	b.FailAfter(0)
	v, e := s.ModWithInvocation(&a, &d, &first)
	serviceMust(t, e)
	p, e := s.PowerOfTwoWithInvocation(2, &second)
	serviceMust(t, e)
	if first.self != &first || second.self != &second || first.invocation == 0 || second.invocation == first.invocation {
		t.Fatal("actual independent invocation")
	}
	firstTuple := first.invocation
	serviceMust(t, first.SourceReady())
	serviceMust(t, first.Cleanup(false))
	serviceMust(t, first.Cleanup(true))
	serviceMust(t, second.Cleanup(false))
	serviceMust(t, second.Cleanup(true))
	if first.invocation != firstTuple {
		t.Fatal("own ACK changed immutable tuple")
	}
	copied := first
	if e := copied.Cleanup(true); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("copied invocation accepted", e)
	}
	if _, e := s.ModWithInvocation(&a, &d, &first); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("active context overwritten", e)
	}
	b.FailAfter(-1)
	serviceMust(t, s.Del(&v))
	serviceMust(t, s.Del(&p))
	serviceMust(t, first.EndContext())
	serviceMust(t, second.EndContext())
	if e := first.Cleanup(true); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("ended tuple accepted", e)
	}
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&d))
}

// Each ordinal uses a fresh actual service and caller. The invocation owns only
// the exact lexical records transferred by its reached call; cache and caller
// values remain independently named before the refusal is lifted.
func TestRationalServiceInvocationEveryPowerReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 300; ordinal++ {
		b := arena.NewBudget(1 << 27)
		s, e := NewRationalService(serviceConfig(b))
		serviceMust(t, e)
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		out, body := s.PowerOfTwoWithInvocation(40, &call)
		roots := []*Int{&out.raw.num, &out.raw.den}
		for i := range s.cache {
			roots = append(roots, &s.cache[i].num, &s.cache[i].den)
		}
		for i := uint8(0); i < call.scope.count; i++ {
			roots = append(roots, &call.scope.roots[i].num, &call.scope.roots[i].den)
		}
		integerPhysicalAudit(t, s.manager.z, roots...)
		if want := s.headerBytes + s.cacheBytes + s.manager.headerBytes + integerActualOwnedBytes(s.manager.z); b.Used() != want {
			t.Fatal("before-lift actual cold graph", ordinal, b.Used(), want)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected source body", ordinal, body)
		}
		tuple, original, source := call.invocation, call.bodyCause, call.sourceCause
		b.FailAfter(-1)
		serviceMust(t, call.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		if call.invocation != tuple || call.bodyCause != original || call.sourceCause != source {
			t.Fatal("own ACK replaced history", ordinal)
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.CloseChecked())
		serviceMust(t, s.manager.z.memory.Close())
		if b.Used() != 0 {
			t.Fatal("context end", ordinal, b.Used())
		}
		if body == nil {
			if ordinal == 0 {
				t.Fatal("missing real constructor/copy allocations")
			}
			t.Log("first complete", ordinal)
			return
		}
	}
	t.Fatal("power prefix horizon did not complete")
}

func TestRationalServiceInvocationDetachedCellSourceEnd(t *testing.T) {
	for _, debug := range []bool{false, true} {
		found := false
		for ordinal := int64(0); ordinal < 100 && !found; ordinal++ {
			b := arena.NewBudget(1 << 27)
			cfg := serviceConfig(b)
			cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: debug}
			s, e := NewRationalService(cfg)
			serviceMust(t, e)
			var call RationalServiceInvocation
			b.FailAfter(ordinal)
			out, body := s.PowerOfTwoWithInvocation(40, &call)
			captured := call.numeric.cell
			roots := []*Int{&out.raw.num, &out.raw.den}
			for i := range s.cache {
				roots = append(roots, &s.cache[i].num, &s.cache[i].den)
			}
			for i := uint8(0); i < call.scope.count; i++ {
				roots = append(roots, &call.scope.roots[i].num, &call.scope.roots[i].den)
			}
			integerPhysicalAudit(t, s.manager.z, roots...)
			if want := s.headerBytes + s.cacheBytes + s.manager.headerBytes + integerActualOwnedBytes(s.manager.z); b.Used() != want {
				t.Fatal("immediate detached frontier", debug, ordinal, b.Used(), want)
			}
			b.FailAfter(-1)
			if captured.object != (memory.ObjectRef{}) {
				found = true
				if body == nil || call.originalCell != captured {
					t.Fatal("actual failed leaf missing immutable backing", ordinal, body)
				}
				serviceMust(t, s.SourceFinalize())
				if !debug && (!call.cellRetired || call.numeric.cell != (sourceCellBacking{})) {
					t.Fatal("actual pooled DT did not stamp exact child", call.cellRetired, call.numeric.cell)
				}
				if call.originalCell != captured {
					t.Fatal("source DT rewrote captured history")
				}
			}
			serviceMust(t, call.Cleanup(false))
			if out.ownerID != 0 {
				serviceMust(t, s.manager.z.cleanupRawNumeral(&out.raw.num))
				serviceMust(t, s.manager.z.cleanupRawNumeral(&out.raw.den))
			}
			serviceMust(t, call.Cleanup(true))
			if call.originalCell != captured {
				t.Fatal("ACK rewrote original cell")
			}
			serviceMust(t, call.EndContext())
			serviceMust(t, s.CloseChecked())
			serviceMust(t, s.manager.z.memory.Close())
			if b.Used() != 0 {
				t.Fatal("source then own context end", debug, ordinal, b.Used())
			}
		}
		if !found {
			t.Fatal("real descriptor failure frontier not reached", debug)
		}
	}
}

func TestRationalServiceCopyInvocationEveryActualReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 30; ordinal++ {
		b := arena.NewBudget(1 << 27)
		cfg := serviceConfig(b)
		cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "18446744073709551617")
		out, e := s.NewValue()
		serviceMust(t, e)
		var c RationalServiceInvocation
		b.FailAfter(ordinal)
		body := s.CopyWithInvocation(&out, &a, &c)
		extra := []memory.Block{}
		if block := c.transferred.Block(); block != (memory.Block{}) {
			extra = append(extra, block)
		}
		integerPhysicalAuditWithRaw(t, s.manager.z, extra, &a.raw.num, &a.raw.den, &out.raw.num, &out.raw.den)
		if backing := c.numeric.cell; backing.object != (memory.ObjectRef{}) {
			if _, e := s.manager.z.allocator.ObjectExtent(backing.object); e != nil {
				t.Fatal("exact detached copy backing", e)
			}
		}
		if want := s.headerBytes + s.cacheBytes + s.manager.headerBytes + integerActualOwnedBytes(s.manager.z); b.Used() != want {
			t.Fatal("immediate Copy cold graph", ordinal, b.Used(), want)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(ordinal, body)
		}
		tuple, original := c.invocation, c.bodyCause
		b.FailAfter(-1)
		serviceMust(t, c.Cleanup(false))
		serviceMust(t, s.CleanupValue(&out))
		serviceMust(t, c.Cleanup(true))
		if c.invocation != tuple || c.bodyCause != original {
			t.Fatal("copy receipt/history overwritten")
		}
		serviceMust(t, c.EndContext())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, s.manager.z.memory.Close())
		if b.Used() != 0 {
			t.Fatal("actual copy caller end", ordinal, b.Used())
		}
		if body == nil {
			if ordinal == 0 {
				t.Fatal("copy allocator frontier erased")
			}
			t.Log("first completion", ordinal)
			return
		}
	}
	t.Fatal("copy prefix horizon")
}

// Source: src/util/rational.h compound operators used by
// src/ast/bv_decl_plugin.cpp:759-763. Same caller NUM, no by-value work copy.
func TestRationalServiceCompoundInvocationInlineExactReceiver(t *testing.T) {
	s, b := serviceNew(t)
	a := serviceValue(t, s, "17")
	d := serviceValue(t, s, "8")
	var subtract, add RationalServiceInvocation
	b.FailAfter(0)
	serviceMust(t, s.SubAssignWithInvocation(&a, &d, &subtract))
	serviceMust(t, s.AddAssignWithInvocation(&a, &d, &add))
	b.FailAfter(-1)
	got, e := s.String(&a)
	serviceMust(t, e)
	if got != "17" {
		t.Fatal("actual compound receiver", got)
	}
	for _, c := range []*RationalServiceInvocation{&add, &subtract} {
		serviceMust(t, c.SourceReady())
		serviceMust(t, c.Cleanup(false))
		serviceMust(t, c.Cleanup(true))
		serviceMust(t, c.EndContext())
	}
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&d))
}

// Source: src/util/rational.h:176-185, source BV digit loop uses % then div.
func TestRationalServiceDigitInvocationExactBody(t *testing.T) {
	s, b := serviceNew(t)
	a := serviceValue(t, s, "255")
	d := serviceValue(t, s, "16")
	var remCall, divCall RationalServiceInvocation
	b.FailAfter(0)
	digit, e := s.RemWithInvocation(&a, &d, &remCall)
	serviceMust(t, e)
	quotient, e := s.IDivWithInvocation(&a, &d, &divCall)
	serviceMust(t, e)
	b.FailAfter(-1)
	for _, v := range []*OwnedRational{&digit, &quotient} {
		got, e := s.String(v)
		serviceMust(t, e)
		if got != "15" {
			t.Fatal("literal remainder/division", got)
		}
	}
	for _, c := range []*RationalServiceInvocation{&divCall, &remCall} {
		serviceMust(t, c.SourceReady())
		serviceMust(t, c.Cleanup(false))
	}
	serviceMust(t, s.Del(&quotient))
	serviceMust(t, s.Del(&digit))
	for _, c := range []*RationalServiceInvocation{&divCall, &remCall} {
		serviceMust(t, c.Cleanup(true))
		serviceMust(t, c.EndContext())
	}
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&d))
}
