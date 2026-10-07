// Source: src/util/heap.h; src/util/vector.h:102-129,191-202; src/test/heap.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-14.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntHeapRuntimeRealOwnersAndFindLE(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(16, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	if len(h.values) != 0 || len(h.indices) != 0 || len(h.todo) != 0 || len(h.found) != 0 {
		t.Fatal("Runtime profile has shadow arrays")
	}
	if r.Stats().LiveBlocks != 2 {
		t.Fatal("two actual vector allocations", r.Stats())
	}
	for i := int32(1); i <= 7; i++ {
		heapMust(t, h.Insert(i))
	}
	out, e := NewIntHeapVector(r, b)
	heapMust(t, e)
	heapMust(t, out.Push(99))
	heapMust(t, h.FindLEInto(7, out))
	var got []int32
	n, e := out.Size()
	heapMust(t, e)
	for i := uint32(0); i < n; i++ {
		v, e := out.Get(i)
		heapMust(t, e)
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, []int32{99, 1, 3, 7, 6, 2, 5, 4}) {
		t.Fatal(got)
	}
	heapMust(t, h.Close())
	heapMust(t, out.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual graph not refunded", b.Used())
	}
}

// Bijection: only declared columns and pending/lexical outputs may own blocks.
// Runtime's directory is used solely to detect unnamed live blocks.
func intHeapRuntimeAudit(t *testing.T, r *Runtime, heaps []*IntHeap, outs ...*IntHeapVector) {
	t.Helper()
	seen := map[Block]bool{}
	cold := uint64(0)
	physical := uint64(0)
	bitmap := uint64(0)
	add := func(c *RuntimeVectorColumn, dead bool) {
		refs := []Block{c.Block, c.Pending}
		if dead || c.PendingReplaces {
			refs[0] = Block{}
		}
		for _, b := range refs {
			if b == (Block{}) {
				continue
			}
			if seen[b] {
				t.Fatal("duplicate actual owner", b)
			}
			seen[b] = true
			n, e := r.Size(b)
			heapMust(t, e)
			stride := uint64(r.SizeTBits() / 8)
			physical += n + stride
			bitmap += ((n + stride + 63) / 64) * 8
		}
	}
	for _, h := range heaps {
		if h.closed {
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
	for _, o := range outs {
		if !o.closed {
			cold += intHeapVectorOwnerBytes
			add(&o.column.Raw, false)
		}
	}
	st := r.Stats()
	if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
		t.Fatalf("actual named Runtime graph: %+v named=%d physical=%d bitmap=%d", st, len(seen), physical, bitmap)
	}
	if r.Budget().Used() != st.TotalBytes+cold {
		t.Fatalf("once-only charged graph: %d != %d+%d", r.Budget().Used(), st.TotalBytes, cold)
	}
}
func TestIntHeapRuntimeAssignmentEveryPhysicalRefusal(t *testing.T) {
	successes, failures := 0, 0
	for fail := int64(0); fail < 32; fail++ {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		heapMust(t, e)
		src, e := NewIntHeap(16, IntHeapConfig{Budget: b, Runtime: r, Less: func(a, b int32) bool { return a > b }})
		heapMust(t, e)
		dst, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
		heapMust(t, e)
		heapMust(t, src.Insert(3))
		heapMust(t, dst.Insert(1))
		oldValues := dst.runtimeQueue.runtimeValues.Raw.Block
		b.FailAfter(fail)
		e = dst.Assign(src)
		intHeapRuntimeAudit(t, r, []*IntHeap{src, dst})
		if !dst.lt(3, 1) {
			t.Fatal("base assignment did not precede member frontier")
		}
		if e != nil {
			failures++
			intHeapActualRefusal(t, r, e)
			if _, probe := r.Size(oldValues); probe == nil {
				t.Fatal("old first member remains live")
			}
			if dst.sourceStop != e {
				t.Fatal("original source stop lost", e, dst.sourceStop)
			}
			if dst.Close() != e {
				t.Fatal("source Close silently acknowledged dangling field")
			}
		} else {
			successes++
			if !dst.Contains(3) || !dst.CheckInvariant() {
				t.Fatal("exact copied heap")
			}
		}
		b.FailAfter(-1)
		if e != nil {
			heapMust(t, dst.CleanupStopped())
		} else {
			heapMust(t, dst.Close())
		}
		heapMust(t, src.Close())
		heapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("leaked graph", b.Used())
		}
		if successes != 0 {
			break
		}
	}
	if failures < 2 || successes != 1 {
		t.Fatal("all real member refusal endpoints", failures, successes)
	}
}
func TestIntHeapRuntimeFindLEEveryPhysicalRefusal(t *testing.T) {
	successes, failures, prefixes := 0, 0, 0
	for fail := int64(0); fail < 64; fail++ {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		heapMust(t, e)
		h, e := NewIntHeap(16, IntHeapConfig{Budget: b, Runtime: r})
		heapMust(t, e)
		for i := int32(1); i <= 7; i++ {
			heapMust(t, h.Insert(i))
		}
		out, e := NewIntHeapVector(r, b)
		heapMust(t, e)
		heapMust(t, out.Push(99))
		b.FailAfter(fail)
		e = h.FindLEInto(7, out)
		intHeapRuntimeAudit(t, r, []*IntHeap{h}, out)
		n, read := out.Size()
		heapMust(t, read)
		want := []int32{99, 1, 3, 7, 6, 2, 5, 4}
		for i := uint32(0); i < n; i++ {
			x, e := out.Get(i)
			heapMust(t, e)
			if x != want[i] {
				t.Fatal("live output prefix", fail, i, x)
			}
		}
		if e != nil {
			failures++
			intHeapActualRefusal(t, r, e)
			if n > 1 {
				prefixes++
			}
		} else {
			successes++
			if n != uint32(len(want)) {
				t.Fatal("full traversal", n)
			}
		}
		b.FailAfter(-1)
		heapMust(t, h.Close())
		heapMust(t, out.Close())
		heapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("leaked actual roots")
		}
		if successes > 0 {
			break
		}
	}
	if failures < 4 || prefixes == 0 || successes != 1 {
		t.Fatal("failure-prefix campaign", failures, prefixes, successes)
	}
}

func TestIntHeapRuntimeFullPublicConsumersAndCapturedViews(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(16, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	for _, x := range []int32{3, 1, 2} {
		heapMust(t, h.Insert(x))
	}
	heapMust(t, h.SetBounds(8))
	if h.Bounds() != 8 {
		t.Fatal("source resize shrink", h.Bounds())
	}
	got, e := h.AppendValues(nil)
	heapMust(t, e)
	if !reflect.DeepEqual(got, []int32{1, 3, 2}) {
		t.Fatal("real member read", got)
	}
	display, e := h.Display()
	heapMust(t, e)
	if display != "1\n 3\n 2\n" {
		t.Fatal("source display", display)
	}
	first := h.Begin()
	end := h.End()
	x, e := first.Value()
	heapMust(t, e)
	if x != 1 {
		t.Fatal(x)
	}
	clone, e := h.Clone(b)
	heapMust(t, e)
	if clone.Size() != 3 || !clone.CheckInvariant() {
		t.Fatal("source exact copy")
	}
	other, e := NewIntHeap(16, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	heapMust(t, other.Insert(7))
	heapMust(t, h.Swap(other))
	x, e = first.Value()
	heapMust(t, e)
	if x != 1 {
		t.Fatal("captured payload incorrectly follows current member", x)
	}
	heapMust(t, h.Close())
	x, e = first.Value()
	heapMust(t, e)
	if x != 1 {
		t.Fatal("old owner Close invalidated transferred physical payload", x)
	}
	next := other.Begin()
	heapMust(t, next.Next())
	heapMust(t, next.Next())
	heapMust(t, next.Next())
	if !next.Equal(end) {
		t.Fatal("actual pointer equality after transfer")
	}
	heapMust(t, other.Close())
	if _, e = first.Value(); e == nil {
		t.Fatal("freed payload read accepted")
	}
	heapMust(t, clone.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("owning graph leak")
	}
}

func TestIntHeapRuntimePartialPublishedCopyOwningReceipt(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	src, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	dst, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	old := dst.runtimeQueue.runtimeValues.Raw.Block
	// Explicit malformed raw-source domain: declared size reaches an actually
	// uninitialized slot. This is a reached placement read, never valid-world OOM.
	heapMust(t, r.WriteUint32(src.runtimeQueue.runtimeValues.Raw.Block, 4, 2))
	e = dst.Assign(src)
	if e == nil {
		t.Fatal("guarded source placement read bypassed")
	}
	if dst.runtimeQueue.runtimeValues.Raw.Block == old || dst.destroyed != 0 {
		t.Fatal("new published buffer mistaken for consumed old field")
	}
	if _, e := r.Size(old); e == nil {
		t.Fatal("old buffer not destroyed before new publication")
	}
	x, e := dst.runtimeQueue.runtimeValues.Read(0)
	heapMust(t, e)
	if int32(x) != -1 {
		t.Fatal("completed first placement", x)
	}
	if _, e = dst.runtimeQueue.runtimeValues.Read(1); e == nil {
		t.Fatal("uncopied suffix supplied")
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{src, dst})
	if dst.Close() == nil {
		t.Fatal("source stop silently acknowledged")
	}
	heapMust(t, dst.CleanupStopped())
	heapMust(t, src.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("lost partial published owner")
	}
}
func TestIntHeapRuntimeIndependentMutableComparator(t *testing.T) {
	for code := 0; code < 81; code++ {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		heapMust(t, e)
		var weights [4]int32
		var active [4]bool
		v := code
		for i := 0; i < 4; i++ {
			weights[i] = int32(v % 3)
			v /= 3
			active[i] = true
		}
		h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, Less: func(a, b int32) bool { return weights[a] < weights[b] }})
		heapMust(t, e)
		for i := int32(0); i < 4; i++ {
			heapMust(t, h.Insert(i))
		}
		weights[2] += 4
		heapMust(t, h.Increased(2))
		weights[3] -= 4
		heapMust(t, h.Decreased(3))
		for left := 4; left > 0; left-- {
			x, e := h.Min()
			heapMust(t, e)
			y, e := h.EraseMin()
			heapMust(t, e)
			if x != y || x < 0 || x >= 4 || !active[x] {
				t.Fatal("minimum/extraction", code, x, y)
			}
			for i := 0; i < 4; i++ {
				if active[i] && weights[i] < weights[x] {
					t.Fatal("independent priority minimum", code, x, weights)
				}
			}
			active[x] = false
		}
		intHeapRuntimeAudit(t, r, []*IntHeap{h})
		heapMust(t, h.Close())
		heapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("leak")
		}
	}
}

func TestIntHeapRuntimeAssignmentIndependentDestinationDomain(t *testing.T) {
	a, b := arena.NewBudget(1<<24), arena.NewBudget(1<<24)
	ra, e := NewRuntime(RuntimeConfig{Budget: a, SizeTBits: 64})
	heapMust(t, e)
	rb, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	heapMust(t, e)
	src, e := NewIntHeap(8, IntHeapConfig{Budget: a, Runtime: ra})
	heapMust(t, e)
	dst, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: rb})
	heapMust(t, e)
	heapMust(t, src.Insert(2))
	heapMust(t, dst.Assign(src))
	if dst.runtimeQueue.runtimeValues.Raw.Block.OwnerID() != rb.OwnerID() {
		t.Fatal("destination copy allocated in source Runtime")
	}
	intHeapRuntimeAudit(t, ra, []*IntHeap{src})
	intHeapRuntimeAudit(t, rb, []*IntHeap{dst})
	heapMust(t, src.Close())
	heapMust(t, ra.Close())
	x, e := dst.Min()
	heapMust(t, e)
	if x != 2 {
		t.Fatal("destination depends on destroyed source", x)
	}
	heapMust(t, dst.Close())
	heapMust(t, rb.Close())
	if a.Used() != 0 || b.Used() != 0 {
		t.Fatal("independent domain roots not refunded")
	}
}
func TestIntHeapRuntimeFreshTodoEveryCallAndCallerGuards(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	o, e := NewIntHeapVector(r, b)
	heapMust(t, e)
	heapMust(t, h.FindLEInto(2, o))
	before := r.State().GlobalCount
	heapMust(t, h.FindLEInto(2, o))
	after := r.State().GlobalCount
	if after != before+1 {
		t.Fatal("source lexical todo allocation was cached", before, after)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h}, o)
	b.FailAfter(0)
	e = h.FindLEInto(2, o)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual cold lexical frame reservation", e)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h}, o)
	b.FailAfter(-1)
	copyOwner := *o
	if e = copyOwner.Push(3); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("copied output owning domain", e)
	}
	foreign := arena.NewBudget(1 << 24)
	foreign.FailAfter(0)
	if _, e = NewIntHeapVector(r, foreign); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("foreign cohort not rejected before reservation", e)
	}
	if _, e = NewIntHeap(4, IntHeapConfig{Budget: foreign, Runtime: r}); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("foreign heap cohort", e)
	}
	heapMust(t, o.Close())
	if e = h.FindLEInto(2, o); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed output", e)
	}
	heapMust(t, h.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 || foreign.Used() != 0 {
		t.Fatal("owner guard charged storage")
	}
}

func intHeapActualRefusal(t *testing.T, r *Runtime, e error) {
	t.Helper()
	if errors.Is(e, arena.ErrInjectedFailure) {
		return
	}
	var failure Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal("unexpected actual refusal category", e, r.State())
	}
}

func TestIntHeapLexicalContextActualPostPhysicalStopAcknowledgment(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	heapMust(t, h.startFindFrame())
	heapMust(t, h.findFrame.todo.Push(1))
	// Ownership-protocol fixture, not a reachable valid FindLE stack depth or
	// algorithm RED. Real reserve expansions enlarge the named lexical owner's
	// block sufficiently to reach actual TLS sync on source destruction.
	for {
		n, e := h.findFrame.todo.Capacity()
		heapMust(t, e)
		if n >= 30000 {
			break
		}
		heapMust(t, h.findFrame.todo.Raw.ExpandStride(4))
	}
	contextTLSNearMinimum(t, r)
	cause := h.finishFindFrame()
	if !errors.Is(cause, arena.ErrBounds) || h.findFrame == nil || h.findFrame.todo.Raw.Block != (Block{}) {
		t.Fatal("actual consumed source destruction stop", cause)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h})
	if e = h.Close(); e != cause {
		t.Fatal("source destructor stop hidden", e, cause)
	}
	for attempt := 0; attempt < 3; attempt++ {
		e = h.CleanupStopped()
		intHeapRuntimeAudit(t, r, []*IntHeap{h})
		if e == nil {
			break
		}
		if !errors.Is(e, arena.ErrBounds) || h.findFrame != nil || h.runtimeQueue.runtimeSourceStop == nil {
			t.Fatal("unexpected later actual source stop", attempt, e)
		}
	}
	heapMust(t, e)
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("named lexical context lost or leaked")
	}
}

func TestIntHeapOutputOwnerStopsMutationAfterActualConsumedFree(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	heapMust(t, e)
	out, e := NewIntHeapVector(r, b)
	heapMust(t, e)
	for i := int32(0); i < 30000; i++ {
		heapMust(t, out.Push(i))
	}
	contextTLSNearMinimum(t, r)
	e = out.Close()
	if !errors.Is(e, arena.ErrBounds) || out.closed || out.column.Raw.Block != (Block{}) {
		t.Fatal("actual source free/owning header prefix", e, out.closed)
	}
	intHeapRuntimeAudit(t, r, nil, out)
	if e = out.Push(42); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("stopped owning cleanup accepted mutation", e)
	}
	heapMust(t, out.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("output owner lost")
	}
}

func TestIntHeapFindLEReachedValueRereadAfterComparator(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	var h *IntHeap
	changed := false
	h, e = NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r, Less: func(a, b int32) bool {
		// Explicit raw member-mutation domain: preserve the one-element heap's
		// inverse identity while the source LT call is executing. No heap reentry,
		// fake failure or alternative comparison algorithm is used.
		if !changed {
			changed = true
			heapMust(t, r.WriteUint32(h.runtimeQueue.runtimeValues.Raw.Block, 12, 2))
			heapMust(t, r.WriteUint32(h.runtimeQueue.runtimeIndices.Raw.Block, 12, 0))
			heapMust(t, r.WriteUint32(h.runtimeQueue.runtimeIndices.Raw.Block, 16, 1))
		}
		return a < b
	}})
	heapMust(t, e)
	heapMust(t, h.Insert(1))
	out, e := NewIntHeapVector(r, b)
	heapMust(t, e)
	heapMust(t, h.FindLEInto(3, out))
	x, e := out.Get(0)
	heapMust(t, e)
	if x != 2 {
		t.Fatal("result expression did not perform its actual post-comparator source read", x)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h}, out)
	heapMust(t, out.Close())
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}

func TestIntHeapInvariantReachedSizeReadAllProfiles(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	// Explicit dangling raw member fixture, with the actual consumed-ownership
	// receipt. Source check_invariant_core's size conditional exists in release.
	token := h.runtimeQueue.runtimeValues.Raw.Block
	heapMust(t, r.Deallocate(&token))
	h.destroyed = 1
	var cause any
	func() { defer func() { cause = recover() }(); h.CheckInvariant() }()
	if cause == nil {
		t.Fatal("release check_invariant skipped its actual size dereference")
	}
	if e = h.Close(); e == nil {
		t.Fatal("ordinary source destruction silently acknowledged dangling field")
	}
	heapMust(t, h.CleanupStopped())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("lost reached owner receipt")
	}
}
