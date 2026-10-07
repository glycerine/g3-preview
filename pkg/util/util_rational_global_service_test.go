// Source: src/util/rational.cpp:64-83; src/util/rational.h:67;
// src/util/inf_rational.cpp:185-198; src/util/inf_int_rational.cpp:46-59.
// Source authors: Leonardo de Moura (2006-09-18); Nikolaj Bjorner (2006-12-05).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func rationalGlobalFixture(t *testing.T) (*RationalGlobalService, *arena.Budget, *Runtime) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	cfg := RationalServiceConfig{Integers: IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}}
	g, e := NewRationalGlobalService(cfg)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		rationalGlobalFinish(t, g)
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Error("actual rational global roots", b.Used())
		}
	})
	return g, b, r
}
func TestRationalGlobalManagerPublicationNullAndEndedStaticReinitPrefix(t *testing.T) {
	g, b, _ := rationalGlobalFixture(t)
	if _, e := g.Manager(); !errors.Is(e, ErrRationalServiceNoManager) {
		t.Fatal("actual null global pointer", e)
	}
	paramsMust(t, g.InitializeSource())
	first, e := g.Manager()
	paramsMust(t, e)
	actualStaticContext := g.service
	b.FailAfter(0)
	paramsMust(t, g.InitializeSource())
	if current, e := g.Manager(); e != nil || current != first {
		t.Fatal("source pointer-present shortcut", current, e)
	}
	b.FailAfter(-1)
	paramsMust(t, g.FinalizeSource())
	if g.current != nil || g.service != actualStaticContext {
		t.Fatal("source pointer not null or original static context replaced")
	}
	mask, e := g.service.SourceEndedStaticMask()
	paramsMust(t, e)
	if mask != 0x1ff {
		t.Fatal("literal static destructor entry mask", mask)
	}
	e = g.InitializeSource()
	if !errors.Is(e, ErrRationalStaticLifetime) || g.current == nil || g.current == first || g.service != actualStaticContext {
		t.Fatal("reinit must publish real manager before ended zero access", e)
	}
	published := g.current
	b.FailAfter(0)
	paramsMust(t, g.InitializeSource())
	if g.current != published {
		t.Fatal("pointer-present shortcut after guarded ended constant")
	}
	b.FailAfter(-1)
	e = g.FinalizeSource()
	if !errors.Is(e, ErrRationalStaticLifetime) || g.current != published {
		t.Fatal("reinit finalize must reach ended Inf before deleting manager", e)
	}
}

func TestRationalGlobalNullConstructorFailureRetriesWithoutRetiringFailedOwner(t *testing.T) {
	for ordinal := 0; ordinal < 32; ordinal++ {
		g, b, _ := rationalGlobalFixture(t)
		b.FailAfter(int64(ordinal))
		e := g.InitializeSource()
		var failed *IntegerManager
		if !errors.As(e, &failed) {
			b.FailAfter(-1)
			paramsMust(t, g.Close())
			continue
		}
		if g.current != nil {
			t.Fatal("failed constructor published current")
		}
		used := b.Used()
		b.FailAfter(-1)
		if e = g.InitializeSource(); e != nil {
			t.Fatalf("null-pointer source retry retained old failure instead of attempting constructor: %v", e)
		}
		if g.current == nil || b.Used() <= used {
			t.Fatal("retry did not retain failed owner beside real completed manager")
		}
		found := false
		for n := g.acquisitions; n != nil; n = n.next {
			if n.failed == failed && n.acquired == 2 {
				found = true
			}
		}
		if !found {
			t.Fatal("exact returned failed child not a named retained owner")
		}
		paramsMust(t, g.Close())
		return
	}
	t.Fatal("no actual retained constructor failure reached")
}

// This public TLS recipe reaches the guarded signed-overflow source domain,
// not ordinary exhaustion or a fabricated allocator free refusal.
func TestRationalGlobalActualStoppedFinalizePreservesPointerShortcut(t *testing.T) {
	g, b, r := rationalGlobalFixture(t)
	paramsMust(t, g.InitializeSource())
	old := g.current
	anchor := g.service.Manager.Z
	v, e := g.service.PowerOfTwo(3000)
	paramsMust(t, e)
	paramsMust(t, g.service.Del(&v))
	paramsDecTLSNearMinimum(t, r)
	e = g.FinalizeSource()
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("expected actual reached TLS release guard", e)
	}
	mask, e1 := g.service.SourceEndedStaticMask()
	paramsMust(t, e1)
	if mask != 0x3f || g.current != old || g.sourceCause != e {
		t.Fatal("actual Inf destructor prefix/current retention", mask)
	}
	numericAuditPhysical(t, anchor, numericPhysicalExtraOwners{RationalServices: []*RationalService{g.service}})
	rationalGlobalColdGraph(t, g, r)
	before := r.State()
	used := b.Used()
	b.FailAfter(0)
	paramsMust(t, g.InitializeSource())
	paramsMust(t, g.service.InitializeSource())
	if r.State() != before || b.Used() != used || g.current != old {
		t.Fatal("nonnull source shortcut changed stopped graph")
	}
	if again := g.FinalizeSource(); again != e {
		t.Fatal("source stop lost first cause", again)
	}
	b.FailAfter(-1)

	rationalGlobalFinish(t, g)
}

func rationalGlobalFinish(t *testing.T, g *RationalGlobalService) {
	t.Helper()
	// The actual 3000-power cache can reach more than 32 independent TLS guards.
	// These retries acknowledge consumed physical actions, never source DT replay.
	for i := 0; i < 8192; i++ {
		sv := reflect.ValueOf(g.service).Elem()
		before := [4]uint64{g.config.Integers.Cells.Budget.Used(), testFieldBySourceName(sv, "cleanupStage").Uint(), testFieldBySourceName(sv, "cleanupCacheIndex").Uint(), testFieldBySourceName(sv, "cleanupComponent").Uint()}
		oldNode := g.acquisitions
		e := g.Close()
		after := [4]uint64{g.config.Integers.Cells.Budget.Used(), testFieldBySourceName(sv, "cleanupStage").Uint(), testFieldBySourceName(sv, "cleanupCacheIndex").Uint(), testFieldBySourceName(sv, "cleanupComponent").Uint()}
		if e != nil && before == after && oldNode == g.acquisitions {
			t.Fatal("owning retry made no actual storage/cursor progress", e)
		}
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
	}
	t.Fatal("actual global owning cleanup failed to advance")
}

// Only actual service fields and acquired constructor nodes introduce owners.
// Runtime is counted once; its allocation directory is never an owning source.
func rationalGlobalColdGraph(t *testing.T, g *RationalGlobalService, r *Runtime) {
	t.Helper()
	n := r.Stats().TotalBytes + g.header
	sv := reflect.ValueOf(g.service).Elem()
	n += testFieldBySourceName(sv, "headerBytes").Uint() + testFieldBySourceName(sv, "cacheBytes").Uint()
	seen := map[uintptr]bool{}
	slab := func(v reflect.Value) uint64 {
		if v.IsNil() {
			return 0
		}
		s := v.Elem()
		out := testFieldBySourceName(s, "directoryBytes").Uint()
		if !testFieldBySourceName(s, "closed").Bool() {
			out += uint64(s.Type().Size())
		}
		pages := testFieldBySourceName(s, "pages")
		for i := 0; i < pages.Len(); i++ {
			p := pages.Index(i)
			for _, f := range []string{"records", "meta"} {
				a := testFieldBySourceName(p, f)
				out += uint64(a.Len()) * uint64(a.Type().Elem().Size())
			}
		}
		return out
	}
	addZ := func(v reflect.Value) {
		if v.IsNil() {
			return
		}
		if seen[v.Pointer()] {
			t.Fatal("duplicate owning Integer manager")
		}
		seen[v.Pointer()] = true
		z := v.Elem()
		n += testFieldBySourceName(z, "headerBytes").Uint()
		a := testFieldBySourceName(z, "allocator")
		if !a.IsNil() {
			a = a.Elem()
			loc := testFieldBySourceName(a, "locations")
			n += testFieldBySourceName(a, "header").Uint() + uint64(loc.Cap())*uint64(loc.Type().Elem().Size())
		}
		n += slab(testFieldBySourceName(z, "cells"))
		w := testFieldBySourceName(z, "words")
		if !w.IsNil() {
			w = w.Elem()
			if !testFieldBySourceName(w, "closed").Bool() {
				n += uint64(w.Type().Size())
			}
			n += testFieldBySourceName(w, "directoryBytes").Uint() + slab(testFieldBySourceName(w, "descriptors"))
			pages := testFieldBySourceName(w, "pages")
			for i := 0; i < pages.Len(); i++ {
				p := pages.Index(i)
				n += uint64(testFieldBySourceName(p, "data").Len())*4 + uint64(testFieldBySourceName(p, "next").Len())*4
			}
		}
		nat := testFieldBySourceName(z, "natural")
		if !nat.IsNil() {
			nat = nat.Elem()
			n += testFieldBySourceName(nat, "headerBytes").Uint()
			if !testFieldBySourceName(nat, "text").IsNil() {
				t.Fatal("undeclared Natural text owner")
			}
		}
	}
	addM := func(v reflect.Value) {
		if v.IsNil() {
			return
		}
		m := v.Elem()
		n += testFieldBySourceName(m, "headerBytes").Uint()
		addZ(testFieldBySourceName(m, "z"))
	}
	addM(testFieldBySourceName(sv, "manager"))
	addZ(testFieldBySourceName(sv, "initializationFailure"))
	for node := g.acquisitions; node != nil; node = node.next {
		if node.owner != g || node.invocation == 0 || node.bytes != uint64(reflect.TypeOf(*node).Size()) {
			t.Fatal("invalid acquisition receipt")
		}
		n += node.bytes
		switch node.acquired {
		case 0:
			if node.manager != nil || node.failed != nil {
				t.Fatal("unacquired node claims owner")
			}
		case 1:
			addM(reflect.ValueOf(node.manager))
		case 2:
			addZ(reflect.ValueOf(node.failed))
		default:
			t.Fatal("invalid constructor acquisition")
		}
	}

	if n != g.config.Integers.Cells.Budget.Used() {
		t.Fatal("exact named rational global cold/provider graph", n, g.config.Integers.Cells.Budget.Used())
	}
}

func TestRationalGlobalEveryActualConstructorReservationAndRetry(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		g, b, r := rationalGlobalFixture(t)
		b.FailAfter(ordinal)
		e := g.InitializeSource()
		rationalGlobalColdGraph(t, g, r)
		if e == nil {
			if g.current == nil {
				t.Fatal("successful source manager not published")
			}
			t.Log("first publication ordinal", ordinal)
			return
		}
		if !errors.Is(e, arena.ErrInjectedFailure) {
			var failure Failure
			if !errors.As(e, &failure) || failure.Kind != FailureMemout || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
				t.Fatal("unexpected actual constructor refusal", ordinal, e)
			}
		}
		if g.current != nil {
			t.Fatal("constructor refusal published source pointer")
		}
		old := g.acquisitions
		b.FailAfter(-1)
		paramsMust(t, g.InitializeSource())
		if g.current == nil || g.acquisitions == old {
			t.Fatal("null current retry skipped actual constructor")
		}
		rationalGlobalColdGraph(t, g, r)
		rationalGlobalFinish(t, g)
	}
	t.Fatal("actual constructor failure horizon never reached publication")
}

func TestRationalGlobalEveryReinitReservationPublishesBeforeEndedZero(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		g, b, r := rationalGlobalFixture(t)
		paramsMust(t, g.InitializeSource())
		original := g.service
		paramsMust(t, g.FinalizeSource())
		b.FailAfter(ordinal)
		e := g.InitializeSource()
		rationalGlobalColdGraph(t, g, r)
		if errors.Is(e, ErrRationalStaticLifetime) {
			if g.current == nil || g.service != original || g.acquisitions.acquired != 1 || g.acquisitions.manager != g.current {
				t.Fatal("ended zero guard precedes real publication")
			}
			state := r.State()
			used := b.Used()
			b.FailAfter(0)
			paramsMust(t, g.InitializeSource())
			if r.State() != state || b.Used() != used {
				t.Fatal("published pointer retry performed constructor/body")
			}
			t.Log("first reinit publication ordinal", ordinal)
			return
		}
		if g.current != nil {
			t.Fatal("failed reinit constructor published pointer")
		}
		if !errors.Is(e, arena.ErrInjectedFailure) {
			var f Failure
			if !errors.As(e, &f) || f.Kind != FailureMemout || f.ErrorCode() != 101 || !r.State().OutOfMemory {
				t.Fatal("unexpected reinit refusal", e)
			}
		}
		b.FailAfter(-1)
		if e = g.InitializeSource(); !errors.Is(e, ErrRationalStaticLifetime) || g.current == nil {
			t.Fatal("real retry must reach ended zero after publication", e)
		}
		rationalGlobalColdGraph(t, g, r)
		rationalGlobalFinish(t, g)
	}
	t.Fatal("reinit reservation horizon never reached publication")
}

func TestRationalGlobalNeverInitializedFinalizeEndsFirstStaticBeforeNullGuard(t *testing.T) {
	g, b, r := rationalGlobalFixture(t)
	e := g.FinalizeSource()
	if !errors.Is(e, ErrRationalServiceNoManager) || g.current != nil {
		t.Fatal("first unconditional rational destructor did not reach null manager", e)
	}
	mask, e := g.service.SourceEndedStaticMask()
	paramsMust(t, e)
	if mask != 1 {
		t.Fatal("static lifetime must end at actual first DT entry", mask)
	}
	rationalGlobalColdGraph(t, g, r)
	e = g.InitializeSource()
	if !errors.Is(e, ErrRationalStaticLifetime) || g.current == nil {
		t.Fatal("reinit must publish then set live simple statics before ended Inf access", e)
	}
	mask, e = g.service.SourceEndedStaticMask()
	paramsMust(t, e)
	if mask != 1 {
		t.Fatal("reconstructed source static defaults", mask)
	}
	before := r.State()
	used := b.Used()
	b.FailAfter(0)
	paramsMust(t, g.InitializeSource())
	if r.State() != before || b.Used() != used {
		t.Fatal("nonnull pointer shortcut inspected stale static lifetime")
	}
}

func TestRationalGlobalOwnerCopiesAndForeignProviderRejected(t *testing.T) {
	g, b, r := rationalGlobalFixture(t)
	copied := reflect.New(reflect.TypeOf(*g))
	copied.Elem().Set(reflect.ValueOf(g).Elem())
	other := copied.Interface().(*RationalGlobalService)
	if !errors.Is(other.InitializeSource(), ErrRationalGlobalOwner) || !errors.Is(other.Close(), ErrRationalGlobalOwner) {
		t.Fatal("shallow wrapper copy accepted")
	}
	before := b.Used()
	cfg := g.config
	cfg.Integers.Cells.Budget = arena.NewBudget(1 << 24)
	if _, e := NewRationalGlobalService(cfg); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("foreign physical cohort accepted", e)
	}
	if b.Used() != before {
		t.Fatal("foreign cohort caused cold reservation")
	}
	paramsMust(t, g.InitializeSource())
	rationalGlobalColdGraph(t, g, r)
	rationalGlobalFinish(t, g)
	if _, e := g.Manager(); !errors.Is(e, arena.ErrClosed) || g.current != nil {
		t.Fatal("closed source binding remains published", e)
	}
}
