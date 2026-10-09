package util

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMemoryTrackedReleaseLatePhysicalPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(100000)
	if e != nil {
		t.Fatal(e)
	}
	r.accounting.count = math.MaxInt64
	r.lane.count = 1
	r.lane.size = 0
	progress, e := a.DeallocateTracked(100000, x)
	if !errors.Is(e, arena.ErrBounds) || !progress.RequestedAdjusted || !progress.RawAccounted || !progress.PhysicalReleased || progress.PooledLinked {
		t.Fatalf("source late-free prefix %+v %v", progress, e)
	}
	if r.Stats().LiveBlocks != 0 || a.Stats().DirectObjects != 0 {
		t.Fatal("completed physical free not reflected")
	}
	requested, e := a.AllocationSize()
	if e != nil || requested != 0 {
		t.Fatal("requested subtraction lost")
	}
	state := r.State()
	if e = a.CompleteRelease(100000, x, &progress); e != nil {
		t.Fatal(e)
	}
	if r.State() != state {
		t.Fatal("cleanup repeated physical/counter action")
	}
	r.accounting.count = 0
	r.lane.count = 0
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("owner leaked")
	}
}

func TestMemoryTrackedReleaseBeforePhysicalRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(300)
	if e != nil {
		t.Fatal(e)
	}
	physical := r.Stats()
	r.accounting.size = math.MinInt64
	progress, e := a.DeallocateTracked(300, x)
	if !errors.Is(e, arena.ErrBounds) || !progress.RequestedAdjusted || progress.RawAccounted || progress.PhysicalReleased || progress.PooledLinked {
		t.Fatalf("before-free prefix %+v %v", progress, e)
	}
	if r.Stats() != physical {
		t.Fatal("before-free refusal discarded real root")
	}
	requested, e := a.AllocationSize()
	if e != nil || requested != 0 {
		t.Fatal("requested adjustment lost")
	}
	r.accounting.size = 304 // restore only the injected undefined counter boundary
	if e = a.CompleteRelease(300, x, &progress); e != nil {
		t.Fatal(e)
	}
	requested, e = a.AllocationSize()
	if e != nil || requested != 0 || r.Stats().LiveBlocks != 0 {
		t.Fatal("continuation subtracted requested twice or leaked")
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("owner leaked")
	}
}

func TestMemoryOrdinaryReleaseResumesRequestedPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(300)
	if e != nil {
		t.Fatal(e)
	}
	r.accounting.size = math.MinInt64
	if e = a.Deallocate(300, x); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("actual beforefree refusal %v", e)
	}
	n, e := a.AllocationSize()
	if e != nil || n != 0 || r.Stats().LiveBlocks != 1 {
		t.Fatal("first source request/root prefix lost")
	}
	r.accounting.size = 304
	if e = a.Deallocate(300, x); e != nil {
		t.Fatal(e)
	}
	n, e = a.AllocationSize()
	if e != nil || n != 0 {
		t.Fatalf("retry re-subtracted completed request, n%d err%v", n, e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("root leak")
	}
}

func TestMemoryReleaseAuthorityIgnoresCallerFlagsAndRejectsReuse(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(300)
	if e != nil {
		t.Fatal(e)
	}
	r.accounting.size = math.MinInt64
	p, e := a.DeallocateTracked(300, x)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	// Public snapshots cannot authorize skipping the retained real raw root.
	p.RequestedAdjusted = false
	p.RawAccounted = true
	p.PhysicalReleased = true
	p.PooledLinked = true
	r.accounting.size = 304
	if e = a.CompleteRelease(300, x, &p); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 || !p.PhysicalReleased || !p.RawAccounted || p.PooledLinked {
		t.Fatal("caller flags authorized skipped free or forged pool state")
	}
	state, used := r.State(), b.Used()
	if e = a.CompleteRelease(300, x, &p); e != nil {
		t.Fatal(e)
	}
	if r.State() != state || b.Used() != used {
		t.Fatal("completed authority repeated actions")
	}
	y, e := a.Allocate(300)
	if e != nil {
		t.Fatal(e)
	}
	before := r.State()
	if e = a.CompleteRelease(300, y, &p); e == nil {
		t.Fatal("stale release snapshot authorized new generation")
	}
	if r.State() != before || r.Stats().LiveBlocks != 1 {
		t.Fatal("stale authority damaged reused actual root")
	}
	var empty ReleaseProgress
	if e = a.CompleteRelease(300, y, &empty); e == nil {
		t.Fatal("unbound progress accepted")
	}
	if e = a.CompleteRelease(299, y, &p); e == nil {
		t.Fatal("wrong request accepted")
	}
	if e = a.Deallocate(300, y); e != nil {
		t.Fatal(e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("release metadata leaked")
	}
}

func TestMemoryReleaseAuthorityPreservesDuplicatePoolAndCharges(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(17)
	if e != nil {
		t.Fatal(e)
	}
	physical := r.Stats()
	p, e := a.DeallocateTracked(17, x)
	if e != nil || !p.PooledLinked || !p.RequestedAdjusted || p.PhysicalReleased || p.RawAccounted {
		t.Fatalf("pool phase %+v %v", p, e)
	}
	p2, e := a.DeallocateTracked(17, x)
	if e != nil || !p2.PooledLinked {
		t.Fatalf("source defined duplicate pooled free %v", e)
	}
	n, e := a.AllocationSize()
	if e != nil || n != math.MaxUint64-16 {
		t.Fatalf("source size_t counter wrap %d %v", n, e)
	}
	if r.Stats() != physical {
		t.Fatal("intrusive free generated raw event")
	}
	if _, e = a.NumFreeObjects(); !errors.Is(e, ErrAllocatorNonterminating) {
		t.Fatalf("self-link not preserved %v", e)
	}
	if e = a.CompleteRelease(17, x, &p); e == nil {
		t.Fatal("old serial still authorizes a newer release")
	}
	if e = a.CompleteRelease(17, x, &p2); e != nil {
		t.Fatal(e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("charged release directory metadata leaked")
	}
}

func TestMemoryReleaseAuthorityRejectsForeignAndTamperedBinding(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	other, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(300)
	if e != nil {
		t.Fatal(e)
	}
	r.accounting.size = math.MinInt64
	p, e := a.DeallocateTracked(300, x)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	before, used := r.State(), b.Used()
	bad := p
	bad.serial++
	if e = a.CompleteRelease(300, x, &bad); e == nil {
		t.Fatal("forged serial accepted")
	}
	bad = p
	bad.generation++
	if e = a.CompleteRelease(300, x, &bad); e == nil {
		t.Fatal("forged generation accepted")
	}
	copied := *a
	if e = copied.CompleteRelease(300, x, &p); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatalf("copied authority owner %v", e)
	}
	if e = other.CompleteRelease(300, x, &p); e == nil {
		t.Fatal("foreign allocator accepted authority")
	}
	if e = a.CompleteRelease(300, x, nil); e == nil {
		t.Fatal("nil authority accepted")
	}
	if r.State() != before || b.Used() != used || r.Stats().LiveBlocks != 1 {
		t.Fatal("rejected authority changed root or counter prefix")
	}
	r.accounting.size = 304
	if e = a.CompleteRelease(300, x, &p); e != nil {
		t.Fatal(e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = other.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("owner leak")
	}
}

func TestMemoryOwningCleanupResumesBeforePhysicalFree(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.Allocate(17); e != nil {
		t.Fatal(e)
	}
	r.accounting.size = math.MinInt64
	err := a.Close()
	if !errors.Is(err, arena.ErrBounds) || r.Stats().LiveBlocks != 1 || a.Stats().OwnerBytes == 0 {
		t.Fatalf("beforefree owning cleanup err%v", err)
	}
	r.accounting.size = 8196
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 || a.Stats().TotalBytes != 0 {
		t.Fatal("resumed owning cleanup lost or duplicated real root")
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("charged owner leak")
	}
}

func TestMemoryReleaseAuthorityPhysicalChargesWarmAndNamespace(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(17)
	if e != nil {
		t.Fatal(e)
	}
	if e = arena.ValidateRecordType(reflect.TypeOf(ReleaseProgress{})); e != nil {
		t.Fatal(e)
	}
	if b.Used() != r.Stats().TotalBytes+a.Stats().TotalBytes {
		t.Fatal("real release authority headers/directory charges disagree")
	}
	if a.Stats().LocationBytes != uint64(cap(a.locations))*uint64(reflect.TypeOf(smallObjectLocation{}).Size()) {
		t.Fatal("actual location authority stride uncharged")
	}
	saved := a.nextReleaseSerial
	a.nextReleaseSerial = math.MaxUint64
	state, used := r.State(), b.Used()
	if _, e = a.DeallocateTracked(17, x); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatalf("namespace wrapped/reused %v", e)
	}
	n, e := a.AllocationSize()
	if e != nil || n != 17 || r.State() != state || b.Used() != used {
		t.Fatal("identity refusal changed source prefix")
	}
	a.nextReleaseSerial = saved
	if _, e = a.DeallocateTracked(17, x); e != nil {
		t.Fatal(e)
	}
	state, used = r.State(), b.Used()
	physical := r.Stats()
	b.FailAfter(0)
	var problem error
	var wrong bool
	allocs := testing.AllocsPerRun(1000, func() {
		if problem != nil || wrong {
			return
		}
		y, e := a.Allocate(17)
		if e != nil {
			problem = e
			return
		}
		p, e := a.DeallocateTracked(17, y)
		if e != nil {
			problem = e
			return
		}
		if !p.PooledLinked || !p.RequestedAdjusted || p.PhysicalReleased {
			wrong = true
			return
		}
		if e = a.CompleteRelease(17, y, &p); e != nil {
			problem = e
		}
	})
	if problem != nil || wrong || allocs != 0 || r.State() != state || r.Stats() != physical || b.Used() != used {
		t.Fatalf("warmed actual authority allocations%g err%v wrong%v", allocs, problem, wrong)
	}
	b.FailAfter(-1)
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("authority graph leak")
	}
}
