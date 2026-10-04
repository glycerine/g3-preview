// Licensed under the MIT license.
// Source mpff.cpp display and memory_manager.cpp TLS paths.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func fastRuntimeConfig(b *arena.Budget, r *memory.Runtime, p uint32) FastFloatConfig {
	c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
	c.PrecisionWords, c.InitialCapacity, c.Memory = p, 2, r
	return c
}

func TestFastFloatBorrowedRuntimeIdentityAndGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	before := b.Used()
	c := fastRuntimeConfig(b, r, 2)
	wrong := arena.NewBudget(1 << 20)
	c.Storage.Descriptors.Budget = wrong
	wrong.FailAfter(0)
	if m, e := NewFastFloatManager(c); m != nil || e != arena.ErrConfig || wrong.Used() != 0 {
		t.Fatal("budget guard before header", m, e, wrong.Used())
	}
	copied := *r
	c = fastRuntimeConfig(b, &copied, 2)
	b.FailAfter(0)
	if m, e := NewFastFloatManager(c); m != nil || e != memory.ErrRuntimeOwner || b.Used() != before {
		t.Fatal("copied runtime before header", m, e, b.Used())
	}
	b.FailAfter(-1)
	c = fastRuntimeConfig(b, r, 2)
	m, e := NewFastFloatManager(c)
	serviceMust(t, e)
	if m.RuntimeOwner() != r || m.natural.ownsRuntime {
		t.Fatal("actual borrowed identity")
	}

	var absent *FastFloatManager
	clone := *m
	if absent.RuntimeOwner() != nil || clone.RuntimeOwner() != nil || clone.CloseChecked() != ErrFastFloatOwner {
		t.Fatal("nil/copied identity guard")
	}
	var zero FastFloat
	if _, e := clone.String(&zero); e != ErrFastFloatOwner {
		t.Fatal("copied ordinary read", e)
	}
	caller, e := r.Allocate(17)
	serviceMust(t, e)
	serviceMust(t, r.Write(caller, 0, 73))
	serviceMust(t, m.CloseChecked())
	if _, e := m.String(&zero); e != arena.ErrClosed {
		t.Fatal("closed ordinary read", e)
	}
	serviceMust(t, r.CheckOpen())
	v, e := r.Read(caller, 0)
	if e != nil || v != 73 {
		t.Fatal("borrowed caller destroyed", v, e)
	}
	serviceMust(t, r.Deallocate(&caller))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	b.FailAfter(0)
	if m, e := NewFastFloatManager(c); m != nil || e != arena.ErrClosed {
		t.Fatal("closed before header", m, e)
	}

	invalid := c
	invalid.PrecisionWords = 1
	if m, e := NewFastFloatManager(invalid); m != nil || e != ErrFastFloatDomain {
		t.Fatal("original domain check before borrowed runtime", m, e)
	}
	b.FailAfter(-1)
	m, e = NewFastFloatManager(fastRuntimeConfig(b, nil, 2))
	serviceMust(t, e)
	if m.RuntimeOwner() == nil || !m.natural.ownsRuntime {
		t.Fatal("default owns runtime")
	}
	owned := m.RuntimeOwner()
	serviceMust(t, m.CloseChecked())
	if owned.CheckOpen() != arena.ErrClosed || b.Used() != 0 {
		t.Fatal("owned runtime not closed", b.Used())
	}
}

func TestFastFloatBorrowedRuntimeConstructorRefusalPrefixes(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100; fault++ {
		b := arena.NewBudget(1 << 28)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32, ThreadLocal: true})
		serviceMust(t, e)
		baseline := b.Used()
		state := r.State()
		b.FailAfter(fault)
		m, e := NewFastFloatManager(fastRuntimeConfig(b, r, 2))
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(fault, e)
			}
		} else {
			success = true
			serviceMust(t, m.CloseChecked())
		}
		if b.Used() != baseline || r.State() != state || r.Stats().LiveBlocks != 0 {
			t.Fatal("constructor borrowed graph", fault, b.Used(), baseline, r.State(), state)
		}
		serviceMust(t, r.CheckOpen())
		b.FailAfter(-1)
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(fault, b.Used())
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("no actual successful constructor")
	}
}

// Independent whole shared graph, including caller raw blocks and every actual
// borrowed owner. No budget-to-malloc substitution and no inferred orphan root.
func fastSharedRuntimeAudit(t *testing.T, r *memory.Runtime, owners []*FastFloatManager, caller ...memory.Block) {
	t.Helper()
	roots := map[memory.Block]bool{}
	var physical, initialization uint64
	add := func(v memory.Block) {
		if v == (memory.Block{}) {
			return
		}
		if roots[v] {
			t.Fatal("shared root counted twice", v)
		}
		roots[v] = true
		n, e := r.Size(v)
		serviceMust(t, e)
		extent := n + uint64(r.SizeTBits()/8)
		physical += extent
		initialization += ((extent + 63) / 64) * 8
	}
	owned := r.Stats().TotalBytes
	for _, m := range owners {
		if m.RuntimeOwner() != r || m.natural.ownsRuntime {
			t.Fatal("wrong runtime identity")
		}
		fastFormattingNamedRoots(m, add)
		owned += fastOwnedBytes(m) - r.Stats().TotalBytes
	}
	for _, v := range caller {
		add(v)
	}
	stats := r.Stats()
	if stats.LiveBlocks != uint32(len(roots)) || stats.PhysicalBytes != physical || stats.InitializationBytes != initialization {
		t.Fatal("shared actual physical graph", stats, len(roots), physical, initialization)
	}
	if r.Budget().Used() != owned {
		t.Fatal("shared owner and backing charges", r.Budget().Used(), owned)
	}
}

func TestFastFloatSharedRuntimeProfileAndClose(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, tls := range []bool{false, true} {
			b := arena.NewBudget(1 << 28)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: bits, ThreadLocal: tls})
			serviceMust(t, e)
			first, e := NewFastFloatManager(fastRuntimeConfig(b, r, 96))
			serviceMust(t, e)
			second, e := NewFastFloatManager(fastRuntimeConfig(b, r, 2))
			serviceMust(t, e)
			caller, e := r.Allocate(17)
			serviceMust(t, e)
			serviceMust(t, r.Write(caller, 0, 91))
			var a, c FastFloat
			serviceMust(t, first.SetInt32(&a, 1))
			serviceMust(t, second.SetFraction32(&c, 3, 2))
			var dst [128]byte
			n, e := first.FormatInto(&a, dst[:])
			serviceMust(t, e)
			if string(dst[:n]) != "1" {
				t.Fatal(n, dst)
			}
			n, e = second.FormatDecimalInto(&c, 8, 128, dst[:])
			serviceMust(t, e)
			if string(dst[:n]) != "1.5" {
				t.Fatal(n, dst)
			}
			fastSharedRuntimeAudit(t, r, []*FastFloatManager{first, second}, caller)
			state := r.State()
			extent := int64(17 + bits/8)
			if tls {
				if state.GlobalSize != 0 || state.GlobalCount != 0 || state.LocalSize != extent || state.LocalCount != 8 {
					t.Fatal("TLS getters do not flush", bits, state)
				}
			} else {
				if state.GlobalSize != extent || state.GlobalCount != 8 || state.LocalSize != 0 || state.LocalCount != 0 {
					t.Fatal("nonTLS accounting", bits, state)
				}
			}

			// Two independently completed source defaults now own simultaneous actual
			// overflow arrays in the same Runtime. Closing one owner must not free the
			// other's backing or the caller's block.
			serviceMust(t, first.beginFastFormatting())
			first.formatting.completed = 1
			serviceMust(t, first.formatting.large[0].resize(r, 1025))
			serviceMust(t, second.beginFastFormatting())
			second.formatting.completed = 4
			serviceMust(t, second.formatting.small.resize(r, 32))
			serviceMust(t, second.formatting.small.write(r, 0, 219))
			secondRaw := second.formatting.small.raw
			fastSharedRuntimeAudit(t, r, []*FastFloatManager{first, second}, caller)
			serviceMust(t, first.CloseChecked())
			serviceMust(t, r.CheckOpen())
			fastSharedRuntimeAudit(t, r, []*FastFloatManager{second}, caller)
			if second.formatting.small.raw != secondRaw {
				t.Fatal("other owner's raw token moved")
			}
			word, e := second.formatting.small.read(r, 0)
			if e != nil || word != 219 {
				t.Fatal("other owner's live bytes", word, e)
			}
			serviceMust(t, second.endFastFormatting(nil))
			n, e = second.FormatInto(&c, dst[:])
			serviceMust(t, e)
			if string(dst[:n]) != "3/2" {
				t.Fatal(n, dst)
			}
			serviceMust(t, second.CloseChecked())
			serviceMust(t, r.CheckOpen())
			fastSharedRuntimeAudit(t, r, nil, caller)
			v, e := r.Read(caller, 0)
			if e != nil || v != 91 {
				t.Fatal(v, e)
			}
			serviceMust(t, r.Deallocate(&caller))
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(bits, tls, b.Used())
			}
		}
	}
}

func TestFastFloatBorrowedTLSCharConstructorPostphysicalPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewFastFloatManager(fastRuntimeConfig(b, r, 6364))
	serviceMust(t, e)
	caller, e := r.Allocate(17)
	serviceMust(t, e)
	var a FastFloat
	serviceMust(t, m.SetInt32(&a, -1))
	serviceMust(t, m.natural.reserveText(256))
	serviceMust(t, r.SetMaxCount(1))
	var dst [32]byte
	for i := range dst {
		dst[i] = '!'
	}
	_, e = m.FormatInto(&a, dst[:])
	var refusal memory.Failure
	if !errors.As(e, &refusal) || refusal.Kind != memory.FailureAllocExceeded {
		t.Fatal("actual postphysical TLS count refusal", e)
	}
	char := &m.natural.charFormatting
	if char.completed || !char.pending || char.buffer.pos != 65536 || char.buffer.raw == (memory.Block{}) || char.buffer.pending == (memory.Block{}) {
		t.Fatal("failed constructor retains both physical roots", char.buffer.pos, char.completed, char.pending)
	}
	old, e := r.Size(char.buffer.raw)
	serviceMust(t, e)
	pending, e := r.Size(char.buffer.pending)
	serviceMust(t, e)
	if old != 65536 || pending != 131072 || m.natural.textRead(0) != '-' || m.formatting.originalCause != refusal || m.formatting.cleanupCause != nil {
		t.Fatal("real publication and cause", old, pending, m.formatting.originalCause, m.formatting.cleanupCause)
	}
	for _, v := range dst {
		if v != '!' {
			t.Fatal("caller bytes published")
		}
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, caller)
	before := r.State()
	// Threshold sync follows the successful new physical allocation; caller + old
	// + pending are charged, local size reset, count stays cumulative (seven char
	// expansions plus the caller). No failed-ctor destructor freed either root.
	if before.GlobalSize != 25+65544+131080 || before.LocalSize != 0 || before.GlobalCount != 8 || before.LocalCount != 8 {
		t.Fatal("source TLS postphysical prefix", before)
	}
	serviceMust(t, m.CleanupFormatting()) // free ignores allocation-only maxcount
	after := r.State()
	if after.GlobalSize != 25+65544 || after.LocalSize != -65544 || after.GlobalCount != 16 || after.LocalCount != 8 {
		t.Fatal("negative threshold free sync and cumulative count", after)
	}
	if m.formatting.pending || char.pending || char.buffer.raw != (memory.Block{}) || char.buffer.pending != (memory.Block{}) {
		t.Fatal("owning cleanup not complete")
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, caller)
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.CheckOpen())
	serviceMust(t, r.Deallocate(&caller))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFastFloatBorrowedTLSCompletedDefaultsAfterLateAllocationFailure(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewFastFloatManager(fastRuntimeConfig(b, r, 9000))
	serviceMust(t, e)
	caller, e := r.Allocate(17)
	serviceMust(t, e)
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 3, 2))
	serviceMust(t, m.natural.reserveText(256))
	serviceMust(t, r.SetMaxCount(1))
	var dst [32]byte
	for i := range dst {
		dst[i] = '!'
	}
	_, e = m.FormatDecimalInto(&a, 8, 128, dst[:])
	var refusal memory.Failure
	if !errors.As(e, &refusal) || refusal.Kind != memory.FailureAllocExceeded {
		t.Fatal(e)
	}
	// Real default b1,b2,b3 were completed before interleaved pushes. Their
	// destructors release published buffers, but the failed expansion's new
	// completed allocation remains an explicit unpublished owning root.
	if m.formatting.completed != 0 || m.formatting.active || !m.formatting.pending || m.formatting.originalCause != refusal || m.formatting.cleanupCause != nil {
		t.Fatal("completed destructor cursor", m.formatting.completed, m.formatting.pending, m.formatting.originalCause, m.formatting.cleanupCause)
	}
	var pending memory.Block
	for i := range m.formatting.large {
		v := &m.formatting.large[i]
		if v.raw != (memory.Block{}) || v.pos != 0 || v.capacity() != 1024 {
			t.Fatal("source completed buffer destroyed", i)
		}
		if v.pending != (memory.Block{}) {
			if pending != (memory.Block{}) {
				t.Fatal("two failed allocations")
			}
			pending = v.pending
		}
	}
	if m.formatting.small.raw != (memory.Block{}) || m.formatting.small.pos != 0 || m.formatting.small.capacity() != 16 {
		t.Fatal("small completed destructor")
	}
	if m.formatting.small.pending != (memory.Block{}) {
		if pending != (memory.Block{}) {
			t.Fatal("two failed allocations")
		}
		pending = m.formatting.small.pending
	}
	if pending == (memory.Block{}) || r.Stats().LiveBlocks != 2 {
		t.Fatal("retained allocation and caller", r.Stats())
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, caller)
	for _, v := range dst {
		if v != '!' {
			t.Fatal("caller bytes")
		}
	}
	// Owning close drains that real root and closes FF words/headers only. Runtime
	// remains callable, allocation-limit count is not reset or used as a free guard.
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.CheckOpen())
	fastSharedRuntimeAudit(t, r, nil, caller)
	serviceMust(t, r.Deallocate(&caller))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFastFloatBorrowedRuntimeInlineWarmScope(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewFastFloatManager(fastRuntimeConfig(b, r, 2))
	serviceMust(t, e)
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 3, 2))
	var dst [32]byte
	n, e := m.FormatInto(&a, dst[:])
	serviceMust(t, e)
	if string(dst[:n]) != "3/2" {
		t.Fatal(n, dst)
	}
	before, used := r.State(), b.Used()
	b.FailAfter(0)
	var problem error
	wrong := false
	allocations := testing.AllocsPerRun(100, func() {
		n, e := m.FormatInto(&a, dst[:])
		problem = e
		if n != 3 || dst[0] != '3' || dst[1] != '/' || dst[2] != '2' {
			wrong = true
		}
	})
	if allocations != 0 || problem != nil || wrong || r.State() != before || b.Used() != used {
		t.Fatal("actual inline warm profile", allocations, problem, wrong, r.State(), before, b.Used(), used)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m})
	b.FailAfter(-1)
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Two actual oversized reallocations preserve source counter-before-physical
// prefixes while the host-size guard retains the real old caller blocks. This
// deliberately reaches guarded signed-overflow at a later TLS synchronization;
// it is not valid allocation exhaustion or an allocation-limit free refusal.
func TestFastFloatBorrowedTLSGuardedLateFreeCursor(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewFastFloatManager(fastRuntimeConfig(b, r, 2))
	serviceMust(t, e)
	first, e := r.Allocate(8)
	serviceMust(t, e)
	second, e := r.Allocate(8)
	serviceMust(t, e)
	serviceMust(t, r.Write(first, 0, 13))
	serviceMust(t, r.Write(second, 0, 17))
	serviceMust(t, m.beginFastFormatting())
	m.formatting.completed = 1
	serviceMust(t, m.formatting.large[0].resize(r, 32768))
	sourceRaw := m.formatting.large[0].raw
	n, e := r.Size(sourceRaw)
	serviceMust(t, e)
	if n != 131072 {
		t.Fatal(n)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, first, second)
	before := r.State()
	old := first
	delta := int64(-1000000)
	// old physical extent16, prospective n+8 =16+delta (source uint64 wrap).
	request := uint64(16) + uint64(delta) - 8
	if request <= uint64(^uint(0)>>1) {
		t.Fatal("fixture not host guarded")
	}
	if e = r.Reallocate(&first, request); e != arena.ErrBounds || first != old {
		t.Fatal("real old caller retained", e, first, old)
	}
	prefix := r.State()
	if prefix.GlobalSize != before.GlobalSize || prefix.LocalSize != before.LocalSize+delta || prefix.LocalCount != before.LocalCount+1 || prefix.OutOfMemory {
		t.Fatal("first source realloc counter prefix", before, prefix)
	}
	v, e := r.Read(first, 0)
	if e != nil || v != 13 {
		t.Fatal("old bytes", v, e)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, first, second)
	serviceMust(t, r.Deallocate(&first))
	negative := r.State()
	if negative.GlobalSize != prefix.GlobalSize+prefix.LocalSize-16 || negative.LocalSize != 0 || negative.GlobalSize >= 0 || negative.GlobalCount != prefix.GlobalCount+prefix.LocalCount {
		t.Fatal("first actual physical free/sync", prefix, negative)
	}
	// The first completed free made global negative. A second real failed realloc
	// can place the lane near MinInt64 without overflowing the source lane write.
	target := int64(math.MinInt64) + 1000 - negative.GlobalSize + 16
	request = uint64(target) + 16 - 8
	old = second
	if request <= uint64(^uint(0)>>1) {
		t.Fatal("second fixture not host guarded")
	}
	if e = r.Reallocate(&second, request); e != arena.ErrBounds || second != old {
		t.Fatal("second real old caller retained", e)
	}
	prefix = r.State()
	if prefix.LocalSize != target || prefix.LocalCount != negative.LocalCount+1 || prefix.GlobalSize != negative.GlobalSize {
		t.Fatal("second source realloc counter prefix", prefix, target)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m}, second)
	serviceMust(t, r.Deallocate(&second))
	near := r.State()
	if near.GlobalSize != math.MinInt64+1000 || near.LocalSize != 0 || near.GlobalCount != prefix.GlobalCount+prefix.LocalCount {
		t.Fatal("bounded actual second free", near)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m})
	if e = m.endFastFormatting(nil); e != arena.ErrBounds {
		t.Fatal("signed overflow after physical source free", e)
	}
	stopped := r.State()
	if stopped.GlobalSize != near.GlobalSize || stopped.LocalSize != -131080 || stopped.GlobalCount != near.GlobalCount || stopped.LocalCount != near.LocalCount {
		t.Fatal("actual overflow stopped at sync addition", near, stopped)
	}
	if m.formatting.large[0].raw != (memory.Block{}) || m.formatting.completed != 1 || m.formatting.active || !m.formatting.pending || m.formatting.cleanupCause != arena.ErrBounds || m.formatting.originalCause != nil {
		t.Fatal("real completed physical free cursor", m.formatting.completed, m.formatting.pending, m.formatting.cleanupCause)
	}
	if _, e = r.Size(sourceRaw); e != memory.ErrRuntimeOwner {
		t.Fatal("old freed physical token", e)
	}
	fastSharedRuntimeAudit(t, r, []*FastFloatManager{m})
	// Cursor acknowledgement does not retry the failed TLS synchronization, replay
	// physical free, or repair the guarded source counter. No root is lost.
	serviceMust(t, m.CleanupFormatting())
	if r.State() != stopped || m.formatting.completed != 0 || m.formatting.pending {
		t.Fatal("cleanup acknowledgement replayed source actions", r.State(), stopped)
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
