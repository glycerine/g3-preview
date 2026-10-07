// Source: src/util/heap.h:61-78,126,136-145; src/util/debug.h:60-65; src/test/heap.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-14.
package util

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntHeapDebugSourceConstructorGate(t *testing.T) {
	d, _, b := debugFixture(t)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	if (d.Tags != nil) != sourceDebug {
		t.Fatal("source CASSERT lazy lookup profile", d.Tags != nil)
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugSourceEnabledInvariantBeforeReset(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	// Actual initialized inverse slot corruption: source CASSERT before reset
	// must observe it; tag-disabled/release reset retains its ordinary body.
	heapMust(t, h.runtimeQueue.runtimeIndices.Write(1, 0))
	e = h.Reset()
	if sourceDebug {
		if !errors.Is(e, ErrRuntimeVectorAssertion) {
			t.Fatal("pre-reset invariant", e)
		}
		n, x := h.runtimeQueue.runtimeValues.Size()
		heapMust(t, x)
		if n != 2 {
			t.Fatal("reset preceded assertion", n)
		}
	} else {
		heapMust(t, e)
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugSourceDisabledAssertionsSkipTagLookup(t *testing.T) {
	d, _, b := debugFixture(t)
	heapMust(t, d.EnableAssertions(false))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	if d.Tags != nil {
		t.Fatal("disabled assertion evaluated tag")
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugDisabledIndexedAssertionRetainsPhysicalGuard(t *testing.T) {
	d, _, b := debugFixture(t)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	// Shrink changes only source SIZE; the initialized retained slot still holds1.
	heapMust(t, h.runtimeQueue.runtimeValues.Shrink(1))
	heapMust(t, d.EnableAssertions(false))
	v, e := h.Min()
	heapMust(t, e)
	if v != 1 {
		t.Fatal("disabled source empty/index assertions", v)
	}
	// Physical extent is never exempted by assertions_enabled=false.
	_, e = h.runtimeQueue.valueRuntime(1000)
	if e == nil {
		t.Fatal("disabled assertions bypassed actual payload bounds")
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugClosedBorrowReachedOnlyInDebug(t *testing.T) {
	d, _, b := debugFixture(t)
	heapMust(t, d.Close())
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("reached assertion context", e)
		}
		if h != nil {
			heapMust(t, h.Close())
		}
	} else {
		heapMust(t, e)
		heapMust(t, h.Close())
	}
	heapMust(t, r.Close())
}

func TestIntHeapDebugDisabledExpansionAssertionUsesLiteralOverflow(t *testing.T) {
	d, _, b := debugFixture(t)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, d.EnableAssertions(false))
	c := &h.runtimeQueue.runtimeIndices.Raw
	heapMust(t, WriteRuntimeWord(r, c.Block, 0, 0))
	e = h.SetBounds(5)
	var overflow DefaultException
	if !errors.As(e, &overflow) || overflow.Error() != "Overflow encountered when expanding vector" {
		t.Fatal("disabled nonnull-capacity SASSERT must reach literal geometric overflow", e)
	}
	n, x := c.Size()
	heapMust(t, x)
	if n != 4 {
		t.Fatal("size published before failed expansion", n)
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugResizePlacementBeforeReachedContext(t *testing.T) {
	if !sourceDebug {
		return
	}
	d, _, b := debugFixture(t)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	c := &h.runtimeQueue.runtimeIndices
	capacity, e := c.Capacity()
	heapMust(t, e)
	if capacity != 5 {
		t.Fatal("actual resize geometry", capacity)
	}
	heapMust(t, d.Close())
	e = h.SetBounds(5)
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("post-resize CASSERT reached context", e)
	}
	n, x := c.Size()
	heapMust(t, x)
	if n != 5 {
		t.Fatal("size publication", n)
	}
	v, x := ReadRuntimeWord(r, c.Raw.Block, 8+4*4)
	heapMust(t, x)
	if v != 0 {
		t.Fatal("ascending source placement before CASSERT", v)
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

// Runtime directory records detect unowned live objects; they never supply an
// owner. All actual malloc tokens come from declared heap columns/lexical state.
func intHeapDebugPhysical(t *testing.T, d *DebugContext, region *arena.BytesRegion, r *Runtime, b *arena.Budget, hs ...*IntHeap) {
	t.Helper()
	seen := map[Block]bool{}
	physical, bitmap, cold := uint64(0), uint64(0), uint64(0)
	add := func(c *RuntimeVectorColumn, consumed bool) {
		refs := []Block{c.Block, c.Pending}
		if consumed || c.PendingReplaces {
			refs[0] = Block{}
		}
		for _, ref := range refs {
			if ref == (Block{}) {
				continue
			}
			if seen[ref] {
				t.Fatal("duplicate named owner", ref)
			}
			seen[ref] = true
			n, e := r.Size(ref)
			heapMust(t, e)
			n += uint64(r.SizeTBits() / 8)
			physical += n
			bitmap += ((n + 63) / 64) * 8
		}
	}
	for _, h := range hs {
		if h == nil || h.closed {
			continue
		}
		cold += intHeapOwnerBytes
		add(&h.runtimeQueue.runtimeValues.Raw, h.destroyed&1 != 0)
		add(&h.runtimeQueue.runtimeIndices.Raw, h.destroyed&2 != 0)
		if h.findFrame != nil {
			cold += intHeapFindFrameBytes
			add(&h.findFrame.todo.Raw, false)
		}
	}
	st := r.Stats()
	if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
		t.Fatal("named actual Runtime graph", st, len(seen), physical, bitmap)
	}
	rs := region.Stats()
	a := rs.Descriptors
	debugBytes := rs.OwnerBytes + rs.PayloadBytes + rs.PageHeaderBytes + rs.DirectoryBytes + rs.LogBytes + rs.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if !d.closed {
		debugBytes += uint64(reflect.TypeOf(DebugContext{}).Size())
	}
	if d.Tags != nil {
		s := d.Tags.Stats()
		debugBytes += s.OwnerBytes + s.TableBytes + s.DirectoryBytes
	}
	want := st.TotalBytes + cold + debugBytes
	if b.Used() != want {
		t.Fatal("exact once-only borrowed debug/runtime/cold graph", b.Used(), want)
	}
}

func TestIntHeapDebugConstructorEveryActualRefusal(t *testing.T) {
	refusals, success, late := 0, 0, 0
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		passed := false
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			d, region, b := debugFixture(t)
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
			heapMust(t, e)
			b.FailAfter(ordinal)
			h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
			intHeapDebugPhysical(t, d, region, r, b, h) // BEFORE disabling failure/format/cleanup.
			if e != nil {
				var f Failure
				if errors.As(e, &f) {
					if f.ErrorCode() != 101 || !r.State().OutOfMemory {
						t.Fatal("real Runtime refusal", e, r.State())
					}
				} else if !errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("unexpected source/cold refusal", e)
				}
				refusals++
				if sourceDebug && r.State().GlobalCount == 2 {
					late++
				}
			} else {
				success++
				passed = true
				if h.Bounds() != 4 || !h.Empty() {
					t.Fatal("constructor output")
				}
			}
			b.FailAfter(-1)
			if h != nil {
				heapMust(t, h.Close())
			}
			intHeapDebugPhysical(t, d, region, r, b)
			heapMust(t, r.Close())
		})
		if passed {
			break
		}
	}
	if refusals == 0 || success != 1 || (sourceDebug && late == 0) {
		t.Fatal("complete real constructor horizon", refusals, success, late)
	}
	t.Logf("actual constructor refusals=%d successordinal=%d late-table-prefixes=%d", refusals, refusals, late)
}

func TestIntHeapDebugEnabledIndependentMutableOracle(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	for world := 0; world < 81; world++ {
		priorities := [4]int32{}
		x := world
		for i := range priorities {
			priorities[i] = int32(x % 3)
			x /= 3
		}
		h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d, Less: func(a, b int32) bool { return priorities[a] < priorities[b] }})
		heapMust(t, e)
		for _, v := range []int32{3, 1, 2, 0} {
			heapMust(t, h.Insert(v))
		}
		priorities[2] = -10
		heapMust(t, h.Decreased(2))
		v, e := h.Min()
		heapMust(t, e)
		if v != 2 {
			t.Fatal("mutable decrease", world, v)
		}
		priorities[2] = 10
		heapMust(t, h.Increased(2))
		present := [4]bool{true, true, true, true}
		for !h.Empty() {
			v, e := h.EraseMin()
			heapMust(t, e)
			if v < 0 || v >= 4 || !present[v] {
				t.Fatal("independent membership", v)
			}
			for i, yes := range present {
				if yes && priorities[i] < priorities[v] {
					t.Fatal("independent priority minimum", world, v, i)
				}
			}
			present[v] = false
		}
		intHeapDebugPhysical(t, d, region, r, b, h)
		heapMust(t, h.Close())
	}
	heapMust(t, r.Close())
}

func TestIntHeapDebugInvariantPerAssertionShortCircuit(t *testing.T) {
	if !sourceDebug {
		return
	}
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	armed, calls := false, 0
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d, Less: func(a, b int32) bool {
		if armed {
			calls++
			heapMust(t, d.EnableAssertions(false))
		}
		return a < b
	}})
	heapMust(t, e)
	for _, v := range []int32{1, 2, 3} {
		heapMust(t, h.Insert(v))
	}
	heapMust(t, WriteRuntimeWord(r, h.runtimeQueue.runtimeIndices.Raw.Block, 8+4*3, 0))
	armed = true
	ok, e := h.invariantRuntime()
	heapMust(t, e)
	if !ok || calls != 1 {
		t.Fatal("source later recursive assertions skipped after actual switch change", ok, calls)
	}
	armed = false
	heapMust(t, d.EnableAssertions(true))
	_, e = h.invariantRuntime()
	if !errors.Is(e, ErrRuntimeVectorAssertion) {
		t.Fatal("enabled traversal reaches retained corruption", e)
	}
	intHeapDebugPhysical(t, d, region, r, b, h)
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugWarmLookupAndHeapPhysicalPlateau(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	_, e = h.EraseMin()
	heapMust(t, e)
	before, used := r.State(), b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		heapMust(t, h.Insert(1))
		v, e := h.EraseMin()
		heapMust(t, e)
		if v != 1 {
			t.Fatal(v)
		}
	})
	intHeapDebugPhysical(t, d, region, r, b, h)
	if allocs != 0 || b.Used() != used || r.State() != before {
		t.Fatal("warmed real tag lookup and heap plateau", allocs, b.Used(), used, r.State(), before)
	}
	b.FailAfter(-1)
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugLazyLookupExactEntryAndPostResizeFrontiers(t *testing.T) {
	for _, route := range []string{"reset-entry", "resize-exit"} {
		t.Run(route, func(t *testing.T) {
			d, region, b := debugFixture(t)
			heapMust(t, d.EnableAssertions(false))
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
			heapMust(t, e)
			h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
			heapMust(t, e)
			if d.Tags != nil {
				t.Fatal("disabled ctor lookup")
			}
			before := r.State()
			heapMust(t, d.EnableAssertions(true))
			b.FailAfter(0)
			if route == "reset-entry" {
				e = h.Reset()
			} else {
				e = h.SetBounds(5)
			}
			intHeapDebugPhysical(t, d, region, r, b, h)
			if sourceDebug {
				if !errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("actual lazy table header refusal", e)
				}
			} else {
				heapMust(t, e)
			}
			if d.Tags != nil || r.State() != before {
				t.Fatal("source diagnostic cold reservation fabricated Runtime malloc", d.Tags, r.State(), before)
			}
			size, x := h.runtimeQueue.runtimeIndices.Size()
			heapMust(t, x)
			want := uint32(4)
			if route == "resize-exit" {
				want = 5
				v, x := ReadRuntimeWord(r, h.runtimeQueue.runtimeIndices.Raw.Block, 8+4*4)
				heapMust(t, x)
				if v != 0 {
					t.Fatal("source placement prefix", v)
				}
			}
			if size != want {
				t.Fatal("reached lookup source frontier", route, size, want)
			}
			b.FailAfter(-1)
			heapMust(t, h.Close())
			heapMust(t, r.Close())
		})
	}
}

func TestIntHeapDebugEnabledInsertInversePublicationBeforePhysicalRefusal(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	old := h.runtimeQueue.runtimeValues.Raw.Block
	b.FailAfter(0)
	e = h.Insert(2)
	var failure Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal("actual reached Runtime growth refusal", e, r.State())
	}
	n, x := h.runtimeQueue.runtimeValues.Size()
	heapMust(t, x)
	inverse, x := ReadRuntimeWord(r, h.runtimeQueue.runtimeIndices.Raw.Block, 8+4*2)
	heapMust(t, x)
	if n != 2 || inverse != 2 || h.runtimeQueue.runtimeValues.Raw.Block != old {
		t.Fatal("source index-before-push prefix", n, inverse)
	}
	intHeapDebugPhysical(t, d, region, r, b, h)
	b.FailAfter(-1)
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugBorrowedTagBytesGuardOnlyAtLookup(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	heapMust(t, region.Close())
	// SASSERT reads only the actual global switch, not borrowed tag bytes.
	v, e := h.Min()
	heapMust(t, e)
	if v != 1 {
		t.Fatal(v)
	}
	before, e := h.runtimeQueue.runtimeValues.Size()
	heapMust(t, e)
	e = h.Insert(2)
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("actual strcmp reached closed borrowed bytes", e)
		}
		after, x := h.runtimeQueue.runtimeValues.Size()
		heapMust(t, x)
		if after != before {
			t.Fatal("source entry CASSERT prefix", after, before)
		}
	} else {
		heapMust(t, e)
	}
	// Neither source numeric/member destruction nor tag-table destruction reads
	// the destroyed caller strings. The caller has not transferred ownership.
	heapMust(t, h.Close())
	heapMust(t, r.Close())
	heapMust(t, d.Close())
}

func TestIntHeapDebugFinalizeReinitializeAndCopiedBorrow(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	old := d.Tags
	heapMust(t, d.FinalizeDebug())
	heapMust(t, h.Reset())
	if sourceDebug {
		if d.Tags == nil || d.Tags == old {
			t.Fatal("source CASSERT lazy reinitialization")
		}
	} else if d.Tags != nil {
		t.Fatal("release erased lookup")
	}
	intHeapDebugPhysical(t, d, region, r, b, h)
	// A copied cold DebugContext never owns the original header/table. Its guard
	// is reached only at a source diagnostic read after actual heap construction.
	copy := *d
	second, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, DebugContext: &copy})
	if sourceDebug {
		if !errors.Is(e, ErrDebugContextOwner) {
			t.Fatal("copied borrowed context", e)
		}
	} else {
		heapMust(t, e)
	}
	intHeapDebugPhysical(t, d, region, r, b, h, second)
	if second != nil {
		heapMust(t, second.Close())
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugSwapFourAssertionSitesInSourceOrder(t *testing.T) {
	d, region, b := debugFixture(t)
	heapMust(t, d.EnableDebug(debugTag(t, region, "heap")))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	var events [4]byte
	count := 0
	record := false
	comparator := func(who byte) func(int32, int32) bool {
		return func(a, b int32) bool {
			if record {
				if count >= len(events) {
					t.Fatal("extra source comparator assertion", count)
				}
				events[count] = who
				count++
			}
			return a < b
		}
	}
	a, e := NewIntHeap(5, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d, Less: comparator('A')})
	heapMust(t, e)
	c, e := NewIntHeap(5, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d, Less: comparator('B')})
	heapMust(t, e)
	for _, v := range []int32{1, 4} {
		heapMust(t, a.Insert(v))
	}
	for _, v := range []int32{2, 3} {
		heapMust(t, c.Insert(v))
	}
	oldA, oldB := a.runtimeQueue.runtimeValues.Raw.Block, c.runtimeQueue.runtimeValues.Raw.Block
	record = true
	heapMust(t, a.Swap(c))
	record = false
	if sourceDebug {
		if count != 4 || events != [4]byte{'B', 'A', 'B', 'A'} {
			t.Fatal("other/self before and after physical swap", count, events)
		}
	} else if count != 0 {
		t.Fatal("release evaluated invariant", count)
	}
	if a.runtimeQueue.runtimeValues.Raw.Block != oldB || c.runtimeQueue.runtimeValues.Raw.Block != oldA {
		t.Fatal("source swapped actual tokens")
	}
	intHeapDebugPhysical(t, d, region, r, b, a, c)
	heapMust(t, d.EnableAssertions(false))
	count = 0
	record = true
	heapMust(t, a.Swap(c))
	record = false
	if count != 0 {
		t.Fatal("global disabled evaluated four CASSERT bodies", count)
	}
	heapMust(t, a.Close())
	heapMust(t, c.Close())
	heapMust(t, r.Close())
}

func TestIntHeapDebugBorrowedOwnerSeparateBudgetGraph(t *testing.T) {
	d, region, debugBudget := debugFixture(t)
	// This is an external global-context borrow, not an owned child. Its real
	// lazy table stays on its caller Budget, never on the heap Runtime budget.
	heapBudget := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: heapBudget, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: heapBudget, Runtime: r, DebugContext: d})
	heapMust(t, e)
	if heapBudget.Used() != r.Stats().TotalBytes+intHeapOwnerBytes {
		t.Fatal("borrow charged as heap ownership", heapBudget.Used(), r.Stats())
	}
	debugAudit(t, d, region, debugBudget)
	heapMust(t, h.Close())
	heapMust(t, r.Close())
	if heapBudget.Used() != 0 {
		t.Fatal("heap graph not refunded", heapBudget.Used())
	}
	debugAudit(t, d, region, debugBudget)
}
