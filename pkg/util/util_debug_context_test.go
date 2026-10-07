// Source: src/util/debug.cpp:28–79 and src/util/debug.h:63–65. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2006. Independent finite state and owning-prefix tests.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func debugTag(t *testing.T, r *arena.BytesRegion, s string) StrKey {
	t.Helper()
	ref, e := r.Allocate(uint32(len(s) + 1))
	serviceMust(t, e)
	for i := 0; i < len(s); i++ {
		serviceMust(t, r.Write(ref, uint32(i), s[i]))
	}
	serviceMust(t, r.Write(ref, uint32(len(s)), 0))
	key, e := BorrowCString(r, ref, 0)
	serviceMust(t, e)
	return key
}
func debugFixture(t *testing.T) (*DebugContext, *arena.BytesRegion, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("remaining debug graph", b.Used())
		}
	})
	d, e := NewDebugContext(r, b)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.Close()) })
	return d, r, b
}
func debugAudit(t *testing.T, d *DebugContext, r *arena.BytesRegion, b *arena.Budget) {
	t.Helper()
	x := r.Stats()
	a := x.Descriptors
	want := x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if !d.closed {
		want += uint64(reflect.TypeOf(DebugContext{}).Size())
	}
	if d.Tags != nil {
		h := d.Tags.Stats()
		want += h.OwnerBytes + h.TableBytes + h.DirectoryBytes
	}
	if b.Used() != want {
		t.Fatal("actual named debug graph", b.Used(), want)
	}
}
func TestDebugContextSourceLazyInitializationAndAssertions(t *testing.T) {
	d, r, b := debugFixture(t)
	if d.Tags != nil {
		t.Fatal("eager table")
	}
	enabled, e := d.AssertionsEnabled()
	serviceMust(t, e)
	if !enabled {
		t.Fatal("source initial true")
	}
	serviceMust(t, d.EnableAssertions(false))
	serviceMust(t, d.FinalizeDebug())
	enabled, e = d.AssertionsEnabled()
	serviceMust(t, e)
	if enabled {
		t.Fatal("finalizer reset assertion switch")
	}
	key := debugTag(t, r, "sat_solver")
	serviceMust(t, d.DisableDebug(key))
	if d.Tags == nil {
		t.Fatal("disable must initialize")
	}
	ok, e := d.IsDebugEnabled(key)
	serviceMust(t, e)
	if ok {
		t.Fatal("missing tag")
	}
	serviceMust(t, d.EnableDebug(key))
	ok, e = d.IsDebugEnabledCString("sat_solver\x00suffix")
	serviceMust(t, e)
	if !ok {
		t.Fatal("strcmp prefix")
	}
	serviceMust(t, d.FinalizeDebug())
	if d.Tags != nil {
		t.Fatal("source null table")
	}
	debugAudit(t, d, r, b)
	ok, e = d.IsDebugEnabled(key)
	serviceMust(t, e)
	if ok || d.Tags == nil {
		t.Fatal("query recreates empty table")
	}
	debugAudit(t, d, r, b)
}
func TestDebugContextFiniteActionOracle(t *testing.T) {
	for code := 0; code < 729; code++ {
		d, r, b := debugFixture(t)
		keys := [2]StrKey{debugTag(t, r, "a"), debugTag(t, r, "b")}
		mask := 0
		n := code
		for step := 0; step < 3; step++ {
			action := n % 9
			n /= 9
			switch {
			case action < 2:
				serviceMust(t, d.EnableDebug(keys[action]))
				mask |= 1 << action
			case action < 4:
				serviceMust(t, d.DisableDebug(keys[action-2]))
				mask &^= 1 << (action - 2)
			case action < 6:
				ok, e := d.IsDebugEnabled(keys[action-4])
				serviceMust(t, e)
				if ok != (mask&(1<<(action-4)) != 0) {
					t.Fatal(code, step, mask)
				}
			case action == 6:
				serviceMust(t, d.FinalizeDebug())
				mask = 0
			case action == 7:
				serviceMust(t, d.EnableAssertions(false))
			case action == 8:
				serviceMust(t, d.EnableAssertions(true))
			}
			for i := 0; i < 2; i++ {
				ok, e := d.IsDebugEnabled(keys[i])
				serviceMust(t, e)
				if ok != (mask&(1<<i) != 0) {
					t.Fatal("independent mask", code, step, i)
				}
			}
			debugAudit(t, d, r, b)
		}
		serviceMust(t, d.Close())
		debugAudit(t, d, r, b)
	}
}
func TestDebugContextEveryLazyConstructionRefusal(t *testing.T) {
	for operation := 0; operation < 3; operation++ {
		failures := 0
		done := false
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			d, r, b := debugFixture(t)
			key := debugTag(t, r, "cold")
			b.FailAfter(ordinal)
			var e error
			switch operation {
			case 0:
				e = d.EnableDebug(key)
			case 1:
				e = d.DisableDebug(key)
			case 2:
				_, e = d.IsDebugEnabled(key)
			}
			debugAudit(t, d, r, b)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				failures++
			} else {
				done = true
			}
			b.FailAfter(-1)
			serviceMust(t, d.Close())
			debugAudit(t, d, r, b)
			if done {
				break
			}
		}
		if !done || failures != 2 {
			t.Fatal("actual lazy header/table prefixes", operation, failures, done)
		}
	}
}
func TestDebugContextBorrowedLifetimeAndOwnerGuards(t *testing.T) {
	d, r, b := debugFixture(t)
	key := debugTag(t, r, "live")
	serviceMust(t, d.EnableDebug(key))
	clone := *d
	if e := clone.FinalizeDebug(); !errors.Is(e, ErrDebugContextOwner) {
		t.Fatal("copied owner", e)
	}
	serviceMust(t, r.Reset())
	if _, e := d.IsDebugEnabled(key); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("real dead caller key", e)
	}
	// Table finalization owns no tag bytes and must not dereference dead keys.
	serviceMust(t, d.FinalizeDebug())
	debugAudit(t, d, r, b)
	serviceMust(t, d.Close())
	if _, e := d.AssertionsEnabled(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestDebugContextConditionalAssertionReadOrder(t *testing.T) {
	d, r, b := debugFixture(t)
	serviceMust(t, d.EnableAssertions(false))
	b.FailAfter(0)
	enabled, e := d.ConditionalAssertionEnabled("sat_solver")
	serviceMust(t, e)
	if enabled || d.Tags != nil {
		t.Fatal("disabled assertions touched lazy table")
	}
	debugAudit(t, d, r, b)
	serviceMust(t, d.EnableAssertions(true))
	_, e = d.ConditionalAssertionEnabled("sat_solver")
	if !errors.Is(e, arena.ErrInjectedFailure) || d.Tags != nil {
		t.Fatal("actual lazy init refusal", e)
	}
	debugAudit(t, d, r, b)
	b.FailAfter(-1)
	tag := debugTag(t, r, "sat_solver")
	serviceMust(t, d.EnableDebug(tag))
	enabled, e = d.ConditionalAssertionEnabled("sat_solver")
	serviceMust(t, e)
	if !enabled {
		t.Fatal("selected tag")
	}
	serviceMust(t, d.EnableAssertions(false))
	enabled, e = d.ConditionalAssertionEnabled("sat_solver")
	serviceMust(t, e)
	if enabled {
		t.Fatal("assertions short circuit")
	}
}
func TestDebugContextColdOwnerAndExistingTableGrowthRefusal(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	before := b.Used()
	b.FailAfter(0)
	d, e := NewDebugContext(r, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || d != nil || b.Used() != before {
		t.Fatal("cold context prefix", d, e, b.Used(), before)
	}
	b.FailAfter(-1)
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	d, r, b = debugFixture(t)
	keys := make([]StrKey, 8)
	for i := 0; i < 8; i++ {
		keys[i] = debugTag(t, r, string(rune('a'+i)))
	}
	for i := 0; i < 7; i++ {
		serviceMust(t, d.EnableDebug(keys[i]))
	}
	if d.Tags.Capacity() != 8 {
		t.Fatal("source load boundary")
	}
	b.FailAfter(0)
	e = d.EnableDebug(keys[7])
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	debugAudit(t, d, r, b)
	for i := 0; i < 8; i++ {
		yes, e := d.IsDebugEnabled(keys[i])
		serviceMust(t, e)
		if yes != (i < 7) {
			t.Fatal("growth refusal changed old table", i)
		}
	}
	b.FailAfter(-1)
	serviceMust(t, d.EnableDebug(keys[7]))
	if d.Tags.Capacity() != 16 {
		t.Fatal("real source rehash")
	}
	debugAudit(t, d, r, b)
}
