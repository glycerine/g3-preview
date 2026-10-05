// Source: src/util/debug.cpp:44-68; src/util/hashtable.h:243-272.
// Source author: Leonardo de Moura, 2006-09-11.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"reflect"
	"testing"
)

func TestDebugRuntimeLazyActualTableAndFinalize(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := NewDebugContext(region, b)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.BindTagRuntime(r); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 {
		t.Fatal("binding eagerly allocated table")
	}
	enabled, e := d.IsDebugEnabledCString("absent")
	if e != nil || enabled {
		t.Fatal(enabled, e)
	}
	roots, e := d.tags.RuntimeRoots()
	if e != nil || roots.Object == (memory.Block{}) || roots.Table == (memory.Block{}) {
		t.Fatal(roots, e)
	}
	if e = d.EnableAssertions(false); e != nil {
		t.Fatal(e)
	}
	if e = d.FinalizeDebug(); e != nil {
		t.Fatal(e)
	}
	if d.tags != nil || r.Stats().LiveBlocks != 0 || d.assertions {
		t.Fatal("same source table destroyed, assertion flag retained")
	}
	if e = d.Close(); e != nil {
		t.Fatal(e)
	}
	if e = region.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMemoryProcessTagRuntimeBindsOnlyAfterSuccessfulParentPreparation(t *testing.T) {
	p, b := memoryProcessFixture(t)
	d, e := NewDebugContext(p.config.Debug.region, b)
	paramsMust(t, e)
	t.Cleanup(func() { b.FailAfter(-1); paramsMust(t, d.Close()) })
	cfg := p.config
	cfg.Debug = d
	before := b.Used()
	b.FailAfter(0)
	failed, e := NewMemoryProcess(cfg)
	if e == nil || failed != nil || d.tagRuntime != nil || d.tags != nil || b.Used() != before {
		t.Fatal("failed parent changed borrowed debug owner", failed, e, b.Used(), before)
	}
	b.FailAfter(-1)
	next, e := NewMemoryProcess(cfg)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, next.Close()) })
	if d.tagRuntime != cfg.Runtime || d.tags != nil {
		t.Fatal("successful parent did not bind actual same Runtime lazily")
	}
	paramsMust(t, next.Close())
	// A live legacy table is not a source Runtime table and cannot be retagged.
	legacy, e := NewDebugContext(p.config.Debug.region, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, legacy.Close()) })
	_, e = legacy.IsDebugEnabledCString("unused")
	paramsMust(t, e)
	cfg.Debug = legacy
	before = b.Used()
	rejected, e := NewMemoryProcess(cfg)
	if e == nil || rejected != nil || legacy.tagRuntime != nil || b.Used() != before {
		t.Fatal("silently migrated legacy table", e)
	}
}

func TestDebugRuntimeEveryLazyConstructorPrefixAndFailedOwnerRetry(t *testing.T) {
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b := arena.NewBudget(1 << 26)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
		paramsMust(t, e)
		region := traceRegion(t, b)
		d, e := NewDebugContext(region, b)
		paramsMust(t, e)
		paramsMust(t, d.BindTagRuntime(r))
		b.FailAfter(ordinal)
		_, e = d.IsDebugEnabledCString("missing")
		live := map[memory.Block]bool{}
		cold := d.ownerBytes
		if d.currentTagConstruction != nil {
			cold += d.currentTagConstruction.bytes
		}
		add := func(h *utilhash.StrHashtable) {
			if h == nil {
				return
			}
			roots, e := h.RuntimeRoots()
			paramsMust(t, e)
			v := reflect.ValueOf(h).Elem()
			cold += v.FieldByName("rootBytes").Uint() + v.FieldByName("directoryBytes").Uint() + roots.AcquisitionBytes
			for _, token := range append([]memory.Block{roots.Object, roots.Table}, roots.Acquisitions...) {
				if token == (memory.Block{}) {
					continue
				}
				if live[token] {
					t.Fatal("duplicate actual lazy root")
				}
				live[token] = true
				_, e = r.Size(token)
				paramsMust(t, e)
			}
		}
		add(d.tags)
		for node := d.failedTags; node != nil; node = node.next {
			cold += node.bytes
			add(node.table)
		}
		if uint32(len(live)) != r.Stats().LiveBlocks {
			t.Fatal("lazy named/raw bijection", live, r.Stats())
		}
		s := region.Stats()
		a := s.Descriptors
		want := cold + r.Stats().TotalBytes + s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
		if b.Used() != want {
			t.Fatal("lazy exact cold graph", ordinal, b.Used(), want)
		}
		if e == nil {
			first = int(ordinal)
			if d.tags == nil {
				t.Fatal("current publication missing")
			}
		} else if d.tags != nil {
			t.Fatal("failed constructor published source pointer")
		}
		b.FailAfter(-1)
		if e != nil {
			_, e = d.IsDebugEnabledCString("missing")
			paramsMust(t, e)
		}
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if first >= 0 {
			break
		}
	}
	if first < 0 {
		t.Fatal("first lazy success missing")
	}
	t.Log("first lazy success", first)
}

func TestDebugRuntimePrephysicalFailedConstructionRetiresEmptyContexts(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	d, e := NewDebugContext(region, b)
	paramsMust(t, e)
	paramsMust(t, d.BindTagRuntime(r))
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	before := b.Used()
	for i := 0; i < 8; i++ {
		b.FailAfter(2)
		_, e = d.IsDebugEnabledCString("missing")
		if e == nil {
			t.Fatal("expected actual acquisition metadata refusal")
		}
		if d.tags != nil || r.Stats().LiveBlocks != 0 || b.Used() != before || d.failedTags != nil {
			t.Fatal("empty failed invocation retained owner charge", i, b.Used(), before)
		}
	}
	b.FailAfter(-1)
	_, e = d.IsDebugEnabledCString("missing")
	paramsMust(t, e)
}

func TestDebugRuntimeProcessColdGraphIncludesActualCurrentContext(t *testing.T) {
	p, _ := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(^uint64(0)))
	_, e := p.config.Debug.IsDebugEnabledCString("missing")
	paramsMust(t, e)
	memoryProcessColdAudit(t, p)
}

// Name the exact current construction context and returned/retired owners;
// directory scans never contribute a missing owner to this cold equation.
func debugTagColdBytes(d *DebugContext) uint64 {
	var n uint64
	add := func(h *utilhash.StrHashtable) {
		if h == nil {
			return
		}
		v := reflect.ValueOf(h).Elem()
		n += v.FieldByName("rootBytes").Uint() + v.FieldByName("tableBytes").Uint() + v.FieldByName("directoryBytes").Uint()
		if !v.FieldByName("config").FieldByName("Runtime").IsNil() {
			roots, e := h.RuntimeRoots()
			if e != nil {
				panic(e)
			}
			n += roots.AcquisitionBytes
		}
	}
	add(d.tags)
	if node := d.currentTagConstruction; node != nil {
		if node.table != nil || node.next != nil || node.bytes != uint64(reflect.TypeOf(*node).Size()) {
			panic(ErrDebugContextOwner)
		}
		n += node.bytes
	}
	for node := d.failedTags; node != nil; node = node.next {
		if node.bytes != uint64(reflect.TypeOf(*node).Size()) {
			panic(ErrDebugContextOwner)
		}
		n += node.bytes
		add(node.table)
	}
	return n
}

func TestDebugRuntimeConstructorExecutingTransportCannotRetireOuterContext(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var d *DebugContext
	active := false
	calls := 0
	var own, source error
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Transport: func(f memory.Failure) error {
		if active {
			calls++
			own = d.Close()
			source = d.FinalizeDebug()
		}
		return f
	}})
	paramsMust(t, e)
	region := traceRegion(t, b)
	d, e = NewDebugContext(region, b)
	paramsMust(t, e)
	paramsMust(t, d.BindTagRuntime(r))
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, r.SetMaxCount(0))
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	paramsMust(t, r.SetMaxCount(1))
	active = true
	_, cause := d.IsDebugEnabledCString("missing")
	active = false
	if cause == nil || calls != 1 {
		t.Fatal("actual count transport not reached", cause, calls)
	}
	if own != ErrDebugTagConstructionExecuting || source != nil || d.closed || d.tags != nil {
		t.Fatal("executing owning guard / actual NULL source-finalize branch", own, source, d.closed)
	}
	paramsMust(t, r.SetMaxCount(0))
	_, e = d.IsDebugEnabledCString("missing")
	paramsMust(t, e)
}

func debugRuntimeNamedGraph(t *testing.T, d *DebugContext, r *memory.Runtime, region *arena.BytesRegion, b *arena.Budget) {
	t.Helper()
	roots := map[memory.Block]bool{}
	add := func(h *utilhash.StrHashtable) {
		if h == nil {
			return
		}
		rr, e := h.RuntimeRoots()
		paramsMust(t, e)
		for _, token := range append([]memory.Block{rr.Object, rr.Table}, rr.Acquisitions...) {
			if token == (memory.Block{}) {
				continue
			}
			if roots[token] {
				t.Fatal("duplicate named debug root", token)
			}
			roots[token] = true
			_, e = r.Size(token)
			paramsMust(t, e)
		}
	}
	add(d.tags)
	for n := d.failedTags; n != nil; n = n.next {
		add(n.table)
	}
	if uint32(len(roots)) != r.Stats().LiveBlocks {
		t.Fatal("debug physical bijection", roots, r.Stats())
	}
	s := region.Stats()
	a := s.Descriptors
	want := d.ownerBytes + debugTagColdBytes(d) + r.Stats().TotalBytes + s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if b.Used() != want {
		t.Fatal("debug exact cold graph", b.Used(), want)
	}
}

// Source: src/util/debug.cpp:44-68; src/util/hashtable.h:243-272.
func TestDebugRuntimePublicGrowthRetryFinalizeRetainsOlderAcquisition(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	paramsMust(t, e)
	region := traceRegion(t, b)
	d, e := NewDebugContext(region, b)
	paramsMust(t, e)
	paramsMust(t, d.BindTagRuntime(r))
	// Reach the actual next expansion through public EnableDebug, not a table
	// capacity mutation. 2048*3/4+1 entries trigger the next prepare_insert.
	for i := 0; i < 1537; i++ {
		paramsMust(t, d.EnableDebug(debugTag(t, region, fmt.Sprintf("public-tag-%d", i))))
	}
	if d.tags.Capacity() != 2048 {
		t.Fatal("actual growth frontier", d.tags.Capacity(), d.tags.Size())
	}
	key := debugTag(t, region, "after-real-failure")
	paramsMust(t, r.SetMaxSize(1))
	cause := d.EnableDebug(key)
	if cause == nil || !r.State().OutOfMemory {
		t.Fatal("real postphysical limit not reached", cause, r.State())
	}
	roots, e := d.tags.RuntimeRoots()
	paramsMust(t, e)
	if len(roots.Acquisitions) != 1 || d.tags.Capacity() != 2048 || d.tags.Size() != 1537 {
		t.Fatal("unjoined NEW and unchanged source table", roots, d.tags.Stats())
	}
	pending := roots.Acquisitions[0]
	debugRuntimeNamedGraph(t, d, r, region, b)
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, d.EnableDebug(key))
	roots, e = d.tags.RuntimeRoots()
	paramsMust(t, e)
	if d.tags.Capacity() != 4096 || d.tags.Size() != 1538 || len(roots.Acquisitions) != 1 || roots.Acquisitions[0] != pending {
		t.Fatal("valid source retry lost older acquisition", roots, d.tags.Stats())
	}
	debugRuntimeNamedGraph(t, d, r, region, b)
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, d.FinalizeDebug())
	if d.tags != nil || d.currentTagConstruction != nil || d.failedTags == nil || d.failedTags.table == nil || d.assertions {
		t.Fatal("source current NULL versus retained owner", d.tags, d.failedTags)
	}
	roots, e = d.failedTags.table.RuntimeRoots()
	paramsMust(t, e)
	if roots.Object != (memory.Block{}) || roots.Table != (memory.Block{}) || len(roots.Acquisitions) != 1 || roots.Acquisitions[0] != pending {
		t.Fatal("source destructor consumed only completed current fields", roots)
	}
	debugRuntimeNamedGraph(t, d, r, region, b)
	paramsMust(t, d.FinalizeDebug())
	debugRuntimeNamedGraph(t, d, r, region, b)
	paramsMust(t, d.Close())
	paramsMust(t, region.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("separate owning abort once-only refund", b.Used())
	}
}
