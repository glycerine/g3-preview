// Source: src/util/sorting_network.h:97-115; src/util/vector.h:59-100,350-365; src/util/memory_manager.cpp:274-289.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestUnsignedSortingNetworkSourceValidNullGrowthRetryRetainsAcquisitions(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	paramsMust(t, e)
	sn, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	in, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	out, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	values := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1}
	for _, v := range values {
		paramsMust(t, in.Push(v))
	}
	paramsMust(t, out.Resize(16, 0))
	// Real borrowed allocation puts the actual TLS lane just below its source
	// synchronization threshold. The first internal malloc crosses it.
	state := r.State()
	request := int64(99990) - state.LocalSize - 8
	if request <= 0 {
		t.Fatal("bounded public priming recipe", state)
	}
	anchor, e := r.Allocate(uint64(request))
	paramsMust(t, e)
	paramsMust(t, r.SetMaxSize(1))
	cause := sn.Sort(in, out)
	if cause == nil || !r.State().OutOfMemory || sn.currentv.Column.Raw.Block != (Block{}) || sn.currentv.Column.Raw.Pending == (Block{}) {
		t.Fatal("actual NULL source field and returned unjoined allocation", cause, r.State(), sn.currentv.Column.Raw)
	}
	old := sn.currentv.Column.Raw.Pending
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&sn.currentv, &sn.nextv, in, out}, sn.header, anchor)
	_, e = r.Size(old)
	paramsMust(t, e)
	paramsMust(t, r.SetMaxSize(0))
	if e = sn.Sort(in, out); e != nil {
		t.Fatal("source allocate never reads sticky OOM flag; NULL retry must run literal body", e)
	}
	sortingUnsignedOracle(t, out, values)
	if !r.State().OutOfMemory {
		t.Fatal("retry must not clear actual source OOM flag")
	}
	_, e = r.Size(old)
	paramsMust(t, e)
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&sn.currentv, &sn.nextv, in, out}, sn.header, anchor)
	paramsMust(t, sn.SourceFinalize())
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&sn.currentv, &sn.nextv, in, out}, sn.header, anchor)
	_, e = r.Size(old)
	paramsMust(t, e)
	paramsMust(t, sn.Close())
	_, e = r.Size(anchor)
	paramsMust(t, e)
	paramsMust(t, out.Close())
	paramsMust(t, in.Close())
	paramsMust(t, r.Deallocate(&anchor))
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("named acquisitions refund", b.Used())
	}
}

// The graph names only concrete fields and caller-owned allocations; Runtime's
// live directory checks the independently declared roots, never supplies them.
func unsignedGrowthGraph(t *testing.T, b *arena.Budget, r *Runtime, vectors []*RuntimeUnsignedVector, enclosing uint64, anchors ...Block) {
	t.Helper()
	roots := map[Block]bool{}
	cold := enclosing
	add := func(h Block) {
		if h == (Block{}) {
			return
		}
		if roots[h] {
			t.Fatal("duplicate actual acquisition", h)
		}
		roots[h] = true
		_, e := r.Size(h)
		paramsMust(t, e)
	}
	for _, h := range anchors {
		add(h)
	}
	for _, v := range vectors {
		cold += v.header
		add(v.Column.Raw.Block)
		add(v.Column.Raw.Pending)
		for n := v.growthAcquisitions; n != nil; n = n.next {
			if n.owner != v || n.runtime != r || n.invocation == 0 || n.invocation > v.growthSerial {
				t.Fatal("actual acquisition binding", n)
			}
			cold += n.header
			add(n.block)
		}
		for n := v.copyAcquisitions; n != nil; n = n.next {
			cold += n.header
			add(n.block)
		}
	}
	s := r.Stats()
	if uint32(len(roots)) != s.LiveBlocks || b.Used() != s.TotalBytes+cold {
		t.Fatal("named physical/cold graph", len(roots), s, b.Used(), cold)
	}
}

func unsignedGrowthPrime(t *testing.T, r *Runtime) Block {
	t.Helper()
	paramsMust(t, r.SetMaxSize(0))
	request := int64(99990) - r.State().LocalSize - 8
	if request <= 0 {
		t.Fatal("public bounded TLS priming", r.State())
	}
	h, e := r.Allocate(uint64(request))
	paramsMust(t, e)
	paramsMust(t, r.SetMaxSize(1))
	return h
}

func TestUnsignedVectorEveryNullRetryReservationPreservesOldAcquisition(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 20; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: bits})
			paramsMust(t, e)
			v, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			anchor := unsignedGrowthPrime(t, r)
			cause := v.Push(17)
			if cause == nil || v.Column.Raw.Pending == (Block{}) || v.Column.Raw.Block != (Block{}) {
				t.Fatal("actual failed malloc", cause)
			}
			old := v.Column.Raw.Pending
			invocation := v.growthInvocation
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, anchor)
			paramsMust(t, r.SetMaxSize(0))
			stateBefore := r.State()
			coldBefore := b.Used()
			b.FailAfter(ordinal)
			e = v.Push(23)
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, anchor)
			if v.growthCause != cause {
				t.Fatal("first body cause sticky")
			}
			if ordinal == 0 && (v.Column.Raw.Pending != old || v.growthInvocation != invocation || v.growthAcquisitions != nil || v.Column.Raw.Block != (Block{})) {
				t.Fatal("failed archive reservation must not transfer authority")
			}
			if ordinal == 0 && (r.State() != stateBefore || b.Used() != coldBefore) {
				t.Fatal("cold archive refusal must not emit a source allocation/accounting event")
			}
			b.FailAfter(-1)
			if e == nil {
				success = ordinal
				x, e := v.At(0)
				paramsMust(t, e)
				if x != 23 {
					t.Fatal(x)
				}
				if v.growthAcquisitions == nil || v.growthAcquisitions.block != old || v.growthAcquisitions.invocation != invocation {
					t.Fatal("old source invocation retained")
				}
			}
			paramsMust(t, v.SourceFinalize())
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, anchor)
			_, e = r.Size(old)
			paramsMust(t, e)
			paramsMust(t, v.Close())
			_, e = r.Size(anchor)
			paramsMust(t, e)
			paramsMust(t, r.Deallocate(&anchor))
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("once-only refund", b.Used())
			}
			if success >= 0 {
				break
			}
		}
		if success < 0 {
			t.Fatal("no first successful retry", bits)
		}
		t.Log("actual retry first-success ordinal", bits, success)
	}
}

func TestUnsignedVectorTwoReturnedNullAcquisitionsAndPrephysicalCause(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	b.FailAfter(0)
	first := v.Push(1)
	if first == nil || v.Column.Raw.Pending != (Block{}) || v.growthCause != first || v.growthAcquisitions != nil {
		t.Fatal("prephysical diagnostic owns no block", first)
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0)
	b.FailAfter(-1)
	a := unsignedGrowthPrime(t, r)
	e = v.Push(2)
	if e == nil {
		t.Fatal("first actual postphysical refusal")
	}
	old := v.Column.Raw.Pending
	oldInvocation := v.growthInvocation
	a2 := unsignedGrowthPrime(t, r)
	e = v.Resize(1, 3)
	if e == nil || v.Column.Raw.Pending == (Block{}) || v.growthAcquisitions == nil || v.growthAcquisitions.block != old || v.growthAcquisitions.invocation != oldInvocation {
		t.Fatal("second real returned block retains first", e)
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, a, a2)
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, v.Resize(1, 7))
	if v.growthAcquisitions.next == nil || v.growthCause != first {
		t.Fatal("two independent acquired tokens and original diagnostic")
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, a, a2)
	cold := b.Used() - r.Stats().TotalBytes
	paramsMust(t, v.Resize(100, 11))
	if b.Used()-r.Stats().TotalBytes != cold || v.growthAcquisitions.next.next != nil {
		t.Fatal("successful warm growth must not retain empty acquisition metadata")
	}
	paramsMust(t, v.SourceFinalize())
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, a, a2)
	paramsMust(t, v.Close())
	paramsMust(t, r.Deallocate(&a))
	paramsMust(t, r.Deallocate(&a2))
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedVectorExecutingGrowthGuardsActualTransportCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var v *RuntimeUnsignedVector
	var closeCause, dtCause error
	calls := 0
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, Transport: func(f Failure) error {
		calls++
		closeCause = v.Close()
		dtCause = v.SourceFinalize()
		return f
	}})
	paramsMust(t, e)
	v, e = NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	anchor := unsignedGrowthPrime(t, r)
	cause := v.Push(42)
	if cause == nil || calls != 1 || closeCause != ErrRuntimeUnsignedGrowthExecuting || dtCause != ErrRuntimeUnsignedGrowthExecuting || v.closed || v.sourceDone || v.destructorCause != nil || v.Column.Raw.Pending == (Block{}) {
		t.Fatal("actual lower callback must not consume executing owner", cause, calls, closeCause, dtCause)
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v}, 0, anchor)
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, v.Push(43))
	paramsMust(t, v.Close())
	paramsMust(t, r.Deallocate(&anchor))
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedVectorGrowthOwnerExactBindingBeforeMutation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	foreign, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	before := b.Used()
	state := r.State()
	copied := *v
	if copied.expandUnsignedSource(nil) != ErrRuntimeVectorOwner || v.Column.Raw.ExpandStrideUnsignedOwner(4, nil, foreign) != ErrRuntimeVectorOwner {
		t.Fatal("copied/foreign column authority")
	}
	if b.Used() != before || r.State() != state || v.growthSerial != 0 || foreign.growthSerial != 0 {
		t.Fatal("binding refusal must be unread/unreserved")
	}
	paramsMust(t, foreign.Close())
	before = b.Used()
	if e = foreign.expandUnsignedSource(nil); e != arena.ErrClosed || before != b.Used() {
		t.Fatal("closed owner guard", e)
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, foreign}, 0)
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedVectorGrowthClosedProviderBeforeArchiveMutation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, r.Close())
	before := b.Used()
	if e = v.expandUnsignedSource(nil); e != arena.ErrClosed || v.growthSerial != 0 || v.growthAcquisitions != nil || b.Used() != before {
		t.Fatal("actual closed borrowed provider before acquisition", e)
	}
	paramsMust(t, v.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedVectorAssignedCurrentGrowthRetainsOlderNullAcquisition(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, src.Push(5))
	paramsMust(t, src.Push(6))
	anchor := unsignedGrowthPrime(t, r)
	first := v.Push(9)
	if first == nil || v.Column.Raw.Pending == (Block{}) || v.Column.Raw.Block != (Block{}) {
		t.Fatal("real NULL unjoined allocation", first)
	}
	old := v.Column.Raw.Pending
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, v.Assign(src))
	if v.Column.Raw.Block == (Block{}) || v.Column.Raw.Pending != old || v.Column.Raw.PendingReplaces {
		t.Fatal("literal copy publication leaves exact old unjoined acquisition")
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
	if e = v.Push(7); e != nil {
		t.Fatal("valid published current growth must not refuse old independently unjoined token", e)
	}
	for i, x := range []uint32{5, 6, 7} {
		y, e := v.At(uint32(i))
		paramsMust(t, e)
		if y != x {
			t.Fatal(i, y, x)
		}
	}
	if v.growthAcquisitions == nil || v.growthAcquisitions.block != old || v.growthCause != first {
		t.Fatal("exact old token and first cause survive valid current realloc")
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
	paramsMust(t, v.SourceFinalize())
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
	paramsMust(t, v.Close())
	paramsMust(t, src.Close())
	paramsMust(t, r.Deallocate(&anchor))
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedVectorEveryAssignedCurrentGrowthReservation(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 12; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: bits})
			paramsMust(t, e)
			v, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			src, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			paramsMust(t, src.Push(5))
			paramsMust(t, src.Push(6))
			anchor := unsignedGrowthPrime(t, r)
			first := v.Push(9)
			if first == nil {
				t.Fatal("actual returned NULL acquisition")
			}
			old := v.Column.Raw.Pending
			paramsMust(t, r.SetMaxSize(0))
			paramsMust(t, v.Assign(src))
			before := v.Column.Raw.Block
			state := r.State()
			invocation := v.growthInvocation
			b.FailAfter(ordinal)
			cause := v.Push(7)
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
			if v.growthCause != first {
				t.Fatal("first source cause preserved")
			}
			if ordinal == 0 && (v.Column.Raw.Block != before || v.Column.Raw.Pending != old || v.growthInvocation != invocation || r.State() != state) {
				t.Fatal("archive refusal before any source realloc action")
			}
			n, e := v.Size()
			paramsMust(t, e)
			if cause != nil && n != 2 || cause == nil && n != 3 {
				t.Fatal("literal element/SIZE publication", cause, n)
			}
			b.FailAfter(-1)
			paramsMust(t, v.SourceFinalize())
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{v, src}, 0, anchor)
			_, e = r.Size(old)
			paramsMust(t, e)
			paramsMust(t, v.Close())
			paramsMust(t, src.Close())
			paramsMust(t, r.Deallocate(&anchor))
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("each actual prefix owning refund", b.Used())
			}
			if cause == nil {
				success = ordinal
				break
			}
		}
		if success != 2 {
			t.Fatal("archive plus actual realloc storage reservations", bits, success)
		}
	}
}
