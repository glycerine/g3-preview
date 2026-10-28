package memory

// Source: src/util/small_object_allocator.cpp:72-96,98-145; src/util/fixed_bit_vector.cpp:78-80.
// Source revision: Leonardo de Moura (leonardo) 2011-04-27; rewrote/simplified allocator.
// Fixed-vector

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func rawAllocatorPair(t *testing.T, p SmallAllocatorProfile) (*SmallAllocator, *SmallAllocator, *Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: p.PointerBits})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: p})
	smallObjectMust(t, e)
	d, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: p})
	smallObjectMust(t, e)
	return a, d, r, b
}
func rawAllocatorPointer(t *testing.T, a *SmallAllocator, v ObjectRef) RawObjectPointer {
	t.Helper()
	h, o, e := a.RawLocation(v)
	smallObjectMust(t, e)
	return RawObjectPointer{Runtime: a.config.Runtime, Block: h, Offset: o}
}
func rawAllocatorGraph(t *testing.T, a, d *SmallAllocator, r *Runtime, b *arena.Budget, roots ...Block) {
	t.Helper()
	named := map[Block]bool{}
	for _, h := range roots {
		if h != (Block{}) {
			named[h] = true
			if _, e := r.block(h); e != nil {
				t.Fatal("named physical root", e)
			}
		}
	}
	for i, x := range r.blocks {
		if x.live {
			h := Block{r.origin, uint32(i + 1), x.generation}
			if !named[h] {
				t.Fatal("unnamed raw", h)
			}
		}
	}
	if len(named) != int(r.live) {
		t.Fatal("physical bijection", len(named), r.live)
	}
	if want := r.Stats().TotalBytes + a.Stats().TotalBytes + d.Stats().TotalBytes; b.Used() != want {
		t.Fatal("once-only graph", b.Used(), want)
	}
}
func rawAllocatorFinish(t *testing.T, a, d *SmallAllocator, r *Runtime, b *arena.Budget) {
	t.Helper()
	smallObjectMust(t, a.Close())
	smallObjectMust(t, d.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("cold refund", b.Used())
	}
}
func TestSmallAllocatorRawForeignDirectSourceCounters(t *testing.T) {
	for _, p := range []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 64, Debug: true}, {PointerBits: 64, Debug: true, Windows: true}, {PointerBits: 32}, {PointerBits: 32, Debug: true, Windows: true}} {
		t.Run(profileName(p), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, p)
			av := smallObjectAlloc(t, a, 300)
			dv := smallObjectAlloc(t, d, 300)
			ap := rawAllocatorPointer(t, a, av)
			dp := rawAllocatorPointer(t, d, dv)
			rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
			prog, e := a.DeallocateRawTracked(300, dp)
			smallObjectMust(t, e)
			if !prog.PhysicalReleased {
				t.Fatal("real raw release receipt", prog)
			}
			if a.allocSize != 0 || d.allocSize != func() uint64 {
				if p.Debug && !p.Windows {
					return 0
				}
				return 300
			}() {
				t.Fatal("receiver/donor source counters", a.allocSize, d.allocSize)
			}
			if _, e = r.Read(dp.Block, 0); e == nil {
				t.Fatal("foreign block still live")
			}
			rawAllocatorGraph(t, a, d, r, b, ap.Block)
			_, e = d.DeallocateRawTracked(300, ap)
			smallObjectMust(t, e)
			rawAllocatorGraph(t, a, d, r, b)
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}
func profileName(p SmallAllocatorProfile) string {
	if p.Debug && !p.Windows {
		return "debug-direct"
	}
	if p.Debug {
		return "debug-windows"
	}
	if p.PointerBits == 32 {
		return "release32"
	}
	return "release64"
}
func TestSmallAllocatorRawReceiverAssertionAndReleaseWrap(t *testing.T) {
	for _, p := range []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 32}, {PointerBits: 64, Debug: true, Windows: true}} {
		t.Run(profileName(p), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, p)
			dv := smallObjectAlloc(t, d, 300)
			dp := rawAllocatorPointer(t, d, dv)
			prog, e := a.DeallocateRawTracked(300, dp)
			if p.Debug {
				if !errors.Is(e, ErrAllocatorAssertion) || prog.PhysicalReleased || a.allocSize != 0 {
					t.Fatal("assert-before-subtract/free", e, prog, a.allocSize)
				}
				rawAllocatorGraph(t, a, d, r, b, dp.Block)
				smallObjectMust(t, d.Deallocate(300, dv))
			} else {
				smallObjectMust(t, e)
				zero := uint64(0)
				if a.allocSize != a.counter(zero-300) {
					t.Fatal("source size_t wrap", a.allocSize)
				}
				rawAllocatorGraph(t, a, d, r, b)
				smallObjectMust(t, a.Reset())
				smallObjectMust(t, d.Reset())
			}
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}
func TestSmallAllocatorRawForeignPooledReuseAndResetLifetime(t *testing.T) {
	for _, p := range []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 32}, {PointerBits: 64, Debug: true, Windows: true}} {
		t.Run(profileName(p), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, p)
			av := smallObjectAlloc(t, a, 17)
			dv := smallObjectAlloc(t, d, 17)
			ap := rawAllocatorPointer(t, a, av)
			dp := rawAllocatorPointer(t, d, dv)
			smallObjectMust(t, d.Write(dv, 8, 0x5a))
			prog, e := a.DeallocateRawTracked(17, dp)
			smallObjectMust(t, e)
			if !prog.PooledLinked || prog.PhysicalReleased || prog.RawAccounted {
				t.Fatal("source pooled link", prog)
			}
			rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
			reused := smallObjectAlloc(t, a, 17)
			rp := rawAllocatorPointer(t, a, reused)
			if rp.Block != dp.Block || rp.Offset != dp.Offset {
				t.Fatal("foreign actual pointer reuse")
			}
			v, e := a.Read(reused, 8)
			smallObjectMust(t, e)
			if v != 0x5a {
				t.Fatal("borrowed payload changed", v)
			}
			smallObjectMust(t, a.Reset())
			v, e = a.Read(reused, 8)
			smallObjectMust(t, e)
			if v != 0x5a {
				t.Fatal("receiver reset freed donor")
			}
			rawAllocatorGraph(t, a, d, r, b, dp.Block)
			smallObjectMust(t, d.Reset())
			if _, e = a.Read(reused, 8); e == nil {
				t.Fatal("donor reset lifetime guard")
			}
			rawAllocatorGraph(t, a, d, r, b)
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}

func TestSmallAllocatorRawForeignPooledEverySidecarRefusal(t *testing.T) {
	failures, success := 0, 0
	for fail := int64(0); fail < 16; fail++ {
		a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 64})
		av := smallObjectAlloc(t, a, 17)
		dv := smallObjectAlloc(t, d, 17)
		ap := rawAllocatorPointer(t, a, av)
		dp := rawAllocatorPointer(t, d, dv)
		b.FailAfter(fail)
		prog, e := a.DeallocateRawTracked(17, dp)
		rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
		if a.allocSize != 0 || d.allocSize != 17 || !prog.RequestedAdjusted {
			t.Fatal("source subtract prefix", a.allocSize, d.allocSize, prog)
		}
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("real sidecar refusal", e)
			}
			p, n, active := a.RawReleasePending()
			if !active || n != 17 || p.Block != dp.Block || p.Offset != dp.Offset {
				t.Fatal("actual retained borrow", p, n, active)
			}
			zero, zerr := a.DeallocateRawTracked(0, RawObjectPointer{})
			if zerr != nil || zero.RequestedAdjusted || zero.PhysicalReleased {
				t.Fatal("source zero must leave stopped invocation unread", zero, zerr)
			}
			again, againN, againActive := a.RawReleasePending()
			if again != p || againN != n || !againActive {
				t.Fatal("zero changed actual pending frame")
			}
			if _, e = a.Allocate(17); !errors.Is(e, ErrAllocatorPending) {
				t.Fatal("ordinary mutation during source stop", e)
			}
			if e = a.Close(); !errors.Is(e, ErrAllocatorPending) {
				t.Fatal("owning root dropped", e)
			}
		} else {
			success++
			if !prog.PooledLinked {
				t.Fatal("actual link")
			}
		}
		b.FailAfter(-1)
		if e != nil {
			prog, e = a.DeallocateRawTracked(17, dp)
			smallObjectMust(t, e)
			if !prog.PooledLinked || a.allocSize != 0 {
				t.Fatal("repeat subtraction", a.allocSize, prog)
			}
		}
		v := smallObjectAlloc(t, a, 17)
		vp := rawAllocatorPointer(t, a, v)
		if vp.Block != dp.Block || vp.Offset != dp.Offset {
			t.Fatal("retry actual pointer")
		}
		smallObjectMust(t, a.Reset())
		smallObjectMust(t, d.Reset())
		rawAllocatorGraph(t, a, d, r, b)
		rawAllocatorFinish(t, a, d, r, b)
		if success != 0 {
			t.Log("firstsuccess", fail)
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal("complete refusal campaign", failures, success)
	}
}

func TestSmallAllocatorRawForeignPooledWarmPointerCodec(t *testing.T) {
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 64})
	av := smallObjectAlloc(t, a, 17)
	dv := smallObjectAlloc(t, d, 17)
	ap := rawAllocatorPointer(t, a, av)
	dp := rawAllocatorPointer(t, d, dv)
	_, e := a.DeallocateRawTracked(17, dp)
	smallObjectMust(t, e)
	v := smallObjectAlloc(t, a, 17)
	before := a.Stats()
	b.FailAfter(0)
	_, e = a.DeallocateRawTracked(17, rawAllocatorPointer(t, a, v))
	rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
	if e != nil {
		t.Fatal("source pointer already has concrete codec; no sidecar growth", e)
	}
	for i := 0; i < 32; i++ {
		v = smallObjectAlloc(t, a, 17)
		p := rawAllocatorPointer(t, a, v)
		if p.Block != dp.Block || p.Offset != dp.Offset {
			t.Fatal("pointer identity")
		}
		_, e = a.DeallocateRawTracked(17, p)
		smallObjectMust(t, e)
	}
	if a.Stats() != before {
		t.Fatal("cold plateau", a.Stats(), before)
	}
	rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
	b.FailAfter(-1)
	smallObjectMust(t, a.Reset())
	smallObjectMust(t, d.Reset())
	rawAllocatorFinish(t, a, d, r, b)
}

func TestSmallAllocatorRawActualSizeTEntryBeforeZeroAndBypass(t *testing.T) {
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 32, Debug: true})
	dv := smallObjectAlloc(t, d, 300)
	dp := rawAllocatorPointer(t, d, dv)
	progress, e := a.DeallocateRawTracked(uint64(1)<<32, RawObjectPointer{})
	smallObjectMust(t, e)
	if progress.PhysicalReleased || progress.RequestedAdjusted {
		t.Fatal("actual size_t zero", progress)
	}
	rawAllocatorGraph(t, a, d, r, b, dp.Block)
	progress, e = a.DeallocateRawTracked((uint64(1)<<32)+1, dp)
	smallObjectMust(t, e)
	if !progress.PhysicalReleased || progress.RequestedAdjusted || a.allocSize != 0 || d.allocSize != 0 {
		t.Fatal("bypass ignores nonzero size", progress)
	}
	rawAllocatorGraph(t, a, d, r, b)
	rawAllocatorFinish(t, a, d, r, b)
}

func TestSmallAllocatorRawUnrelatedReleasePreservesPendingAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	a, e := NewSmallAllocator(cfg)
	smallObjectMust(t, e)
	d, e := NewSmallAllocator(cfg)
	smallObjectMust(t, e)
	dv := smallObjectAlloc(t, d, 300)
	dp := rawAllocatorPointer(t, d, dv)
	external, e := r.Allocate(99672)
	smallObjectMust(t, e)
	smallObjectMust(t, r.SetMaxSize(1))
	v, e := a.Allocate(12)
	var f Failure
	if !errors.As(e, &f) || f.ErrorCode() != 101 || !r.State().OutOfMemory || v != (ObjectRef{}) {
		t.Fatal("real late TLS allocation refusal", v, e, r.State())
	}
	pending, e := a.PendingRoots()
	smallObjectMust(t, e)
	if pending[0] == (Block{}) {
		t.Fatal("no actual pending root")
	}
	rawAllocatorGraph(t, a, d, r, b, external, dp.Block, pending[0])
	progress, e := a.DeallocateRawTracked(300, dp)
	smallObjectMust(t, e)
	if !progress.PhysicalReleased {
		t.Fatal(progress)
	}
	after, e := a.PendingRoots()
	smallObjectMust(t, e)
	if after != pending {
		t.Fatal("unrelated raw release changed pending malloc", pending, after)
	}
	rawAllocatorGraph(t, a, d, r, b, external, pending[0])
	smallObjectMust(t, r.SetMaxSize(0))
	smallObjectMust(t, a.CleanupPending())
	smallObjectMust(t, r.Deallocate(&external))
	rawAllocatorGraph(t, a, d, r, b)
	rawAllocatorFinish(t, a, d, r, b)
}
func TestSmallAllocatorRawSamePendingReleaseReceiptBeforeSubtraction(t *testing.T) {
	// Defined source operations build a borrowed pool pointer, then the caller
	// destroys its donor page. The subsequent dangling-pointer guard is a typed
	// source-invalid-lifetime continuation fixture, not an ordinary allocation fault.
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 64})
	av := smallObjectAlloc(t, a, 17)
	dv := smallObjectAlloc(t, d, 17)
	ap := rawAllocatorPointer(t, a, av)
	dp := rawAllocatorPointer(t, d, dv)
	_, e := a.DeallocateRawTracked(17, dp)
	smallObjectMust(t, e)
	v := smallObjectAlloc(t, a, 17)
	smallObjectMust(t, d.Reset())
	receipt, e := a.DeallocateTracked(17, v)
	if !errors.Is(e, ErrRuntimeOwner) || !receipt.RequestedAdjusted || receipt.PooledLinked {
		t.Fatal("actual stopped reached payload write", e, receipt)
	}
	before := a.allocSize
	state := r.State()
	used := b.Used()
	rawAllocatorGraph(t, a, d, r, b, ap.Block)
	progress, e := a.DeallocateRawTracked(17, dp)
	if !errors.Is(e, ErrAllocatorPending) || progress.RequestedAdjusted || a.allocSize != before || r.State() != state || b.Used() != used {
		t.Fatal("same actual receipt must guard before second source subtraction", e, progress, a.allocSize, before)
	}
	rawAllocatorGraph(t, a, d, r, b, ap.Block)
	// The dead donor borrow has no remaining raw ownership. Source Reset and
	// destructor still drain the receiver's actual owned chunk, with no repair.
	smallObjectMust(t, a.Reset())
	rawAllocatorGraph(t, a, d, r, b)
	rawAllocatorFinish(t, a, d, r, b)
}
