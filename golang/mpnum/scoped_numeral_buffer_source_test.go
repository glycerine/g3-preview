// Source: src/util/scoped_numeral_buffer.h; src/util/buffer.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012-01-23.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func scopedBufferFixture(t *testing.T) (*IntegerModularManager, *scopedIntegerModularBuffer128) {
	t.Helper()
	z, _ := mpzPublicationDirectOwner(t)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	b := new(scopedIntegerModularBuffer128)
	b.construct(m)
	t.Cleanup(func() { serviceMust(t, b.cleanup()); serviceMust(t, m.Close()) })
	return m, b
}
func scopedBufferAudit(t *testing.T, b *scopedIntegerModularBuffer128, extra ...*Int) {
	t.Helper()
	values := make([]Int, b.count)
	roots := append([]*Int{}, extra...)
	for i := range values {
		v, e := b.get(uint32(i))
		serviceMust(t, e)
		values[i] = v
		roots = append(roots, &values[i])
	}
	raw := []memory.Block{}
	for _, r := range []memory.Block{b.raw, b.pending} {
		if r != (memory.Block{}) {
			raw = append(raw, r)
		}
	}
	integerPhysicalAuditWithRaw(t, b.manager.manager, raw, roots...)
}
func TestScopedNumeralBufferInlineGrowthResetAndResize(t *testing.T) {
	m, b := scopedBufferFixture(t)
	for i := int32(0); i < 130; i++ {
		v := NewInt(i - 60)
		serviceMust(t, b.push(&v))
	}
	if b.count != 130 || b.capacity != 256 || b.raw == (memory.Block{}) {
		t.Fatal("source doubling", b.count, b.capacity, b.raw)
	}
	for i := uint32(0); i < 130; i++ {
		v, e := b.get(i)
		serviceMust(t, e)
		s, e := m.manager.String(&v)
		serviceMust(t, e)
		want := NewInt(int32(i) - 60)
		ws, e := m.manager.String(&want)
		serviceMust(t, e)
		if s != ws {
			t.Fatal(i, s, ws)
		}
	}
	serviceMust(t, b.shrink(2))
	serviceMust(t, b.resize(5))
	for i := uint32(2); i < 5; i++ {
		v, e := b.get(i)
		serviceMust(t, e)
		if v != NewInt(0) {
			t.Fatal("resize zero construction", i, v)
		}
	}
	raw := b.raw
	serviceMust(t, b.reset())
	if b.count != 0 || b.raw != raw || b.capacity != 256 {
		t.Fatal("reset must retain backing")
	}
	serviceMust(t, b.resize(3))
	scopedBufferAudit(t, b)
	serviceMust(t, b.sourceClose())
	if b.raw != (memory.Block{}) {
		t.Fatal("base destructor retains allocation")
	}
}
func TestScopedNumeralBufferBigCopiesAndFieldNormalization(t *testing.T) {
	m, b := scopedBufferFixture(t)
	z := m.manager
	var a Int
	serviceMust(t, z.Copy(&a, &z.two64))
	serviceMust(t, b.push(&a))
	serviceMust(t, z.SetInt64(&a, -3))
	v, e := b.get(0)
	serviceMust(t, e)
	s, e := z.String(&v)
	serviceMust(t, e)
	if s != "18446744073709551616" {
		t.Fatal("source deep set", s)
	}
	serviceMust(t, b.resize(0))
	serviceMust(t, m.SetZPUint64(7))
	serviceMust(t, z.SetInt64(&a, -11))
	serviceMust(t, b.push(&a))
	v, e = b.get(0)
	serviceMust(t, e)
	s, e = z.String(&v)
	serviceMust(t, e)
	if s != "3" {
		t.Fatal("source manager normalization", s)
	}
	scopedBufferAudit(t, b, &a)
	serviceMust(t, z.Del(&a))
}
func TestScopedNumeralBufferEveryActualReservationPrefix(t *testing.T) {
	succeeded := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := NewIntegerModularManager(z)
		serviceMust(t, e)
		var b scopedIntegerModularBuffer128
		b.construct(m)
		serviceMust(t, b.resize(128))
		budget.FailAfter(ordinal)
		e = b.push(&z.two64)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatalf("prefix%d: %v", ordinal, e)
		}
		if b.count < 128 || b.count > 129 || b.capacity != 128 && b.capacity != 256 {
			t.Fatal("actual published range", ordinal, b.count, b.capacity)
		}
		scopedBufferAudit(t, &b)
		if e == nil {
			succeeded = true
		}
		budget.FailAfter(-1)
		serviceMust(t, b.cleanup())
		integerPhysicalAudit(t, z)
		serviceMust(t, m.Close())
		if succeeded {
			t.Log("first success", ordinal)
			break
		}
	}
	if !succeeded {
		t.Fatal("actual prefix horizon exhausted")
	}
}

// Actual owner-primitive dangling SET continuation, not a mathematical caller
// reachability claim. The older second numeral must survive source unwinding.
func TestScopedNumeralBufferSourceStopBeforeOlderNumeral(t *testing.T) {
	for _, route := range []string{"reset", "shrink", "destructor"} {
		t.Run(route, func(t *testing.T) {
			m, b := scopedBufferFixture(t)
			z := m.manager
			serviceMust(t, b.push(&z.two64))
			serviceMust(t, b.push(&z.two64))
			older, e := b.get(1)
			serviceMust(t, e)
			var large Int
			limbs := make([]uint32, 8192)
			limbs[0] = 17
			limbs[len(limbs)-1] = 1
			serviceMust(t, z.SetLimbs(&large, limbs))
			t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
			z.budget.FailAfter(0)
			body := b.set(0, &large)
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual source replacement refusal", body)
			}
			first, e := b.get(0)
			serviceMust(t, e)
			if first.cell != z.failedRef {
				t.Fatal("actual dead token", first)
			}
			source := b.reset
			if route == "shrink" {
				source = func() error { return b.shrink(0) }
			}
			if route == "destructor" {
				source = b.sourceClose
			}
			e = source()
			now, x := b.get(1)
			serviceMust(t, x)
			if !errors.Is(e, arena.ErrInvalidSlot) || b.count != 2 || now != older {
				t.Fatal("source consumed dead token or older numeral", e, b.count, now, older)
			}
			if again := source(); again != e {
				t.Fatal("source first cause changed", e, again)
			}
			scopedBufferAudit(t, b, &large)
			z.budget.FailAfter(-1)
			serviceMust(t, b.cleanup())
			integerPhysicalAudit(t, z, &large)
		})
	}
}

func TestScopedNumeralBufferReturnedGrowthAllocationSourceAndAbort(t *testing.T) {
	budget := arena.NewBudget(1 << 22)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	var b scopedIntegerModularBuffer128
	b.construct(m)
	serviceMust(t, b.resize(128))
	local := r.State().LocalSize
	if local < 0 || local >= 99980 {
		t.Fatal("actual constructor lane", r.State())
	}
	prep, e := r.Allocate(uint64(99980 - local - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	zero := NewInt(0)
	body := b.push(&zero)
	var f memory.Failure
	if !errors.As(body, &f) || f.Kind != memory.FailureMemout || b.pending == (memory.Block{}) || b.count != 128 || b.raw != (memory.Block{}) {
		t.Fatal("returned allocation before source publication", body, b.count, b.raw, b.pending)
	}
	integerPhysicalAuditWithRaw(t, z, []memory.Block{prep, b.pending})
	pending := b.pending
	serviceMust(t, b.sourceClose())
	if b.pending != pending {
		t.Fatal("source destructor consumed unpublished storage")
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, b.cleanup())
	integerPhysicalAuditWithRaw(t, z, []memory.Block{prep})
	serviceMust(t, m.Close())
	serviceMust(t, z.CloseChecked())
	serviceMust(t, r.Deallocate(&prep))
	serviceMust(t, r.Close())
	if budget.Used() != 0 {
		t.Fatal("actual full refund", budget.Used())
	}
}
func TestScopedNumeralBufferDescriptorRefusalAndForeignReplacement(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		budget := arena.NewBudget(1 << 22)
		z, e := NewIntegerManager(IntegerConfig{Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
		serviceMust(t, e)
		m, e := NewIntegerModularManager(z)
		serviceMust(t, e)
		var b scopedIntegerModularBuffer128
		b.construct(m)
		e = b.push(&z.two64)
		if !errors.Is(e, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) || b.count != 1 {
			t.Fatal("real pending numeral before descriptor refusal", e, b.count, z.pendingCell)
		}
		first := z.pendingCell
		scopedBufferAudit(t, &b)
		serviceMust(t, b.sourceClose())
		if z.pendingCell != first {
			t.Fatal("source acquired invisible leaf pending")
		}
		foreign := NewInt(19)
		if replacement {
			serviceMust(t, z.CleanupPendingAllocation())
			e = z.Copy(&foreign, &z.two64)
			if !errors.Is(e, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) || z.pendingCell.object == first.object {
				t.Fatal("distinct actual later ObjectRef generation", e, z.pendingCell)
			}
			newer := z.pendingCell
			serviceMust(t, b.cleanup())
			if z.pendingCell != newer {
				t.Fatal("buffer acquired foreign continuation")
			}
			integerPhysicalAudit(t, z, &foreign)
			serviceMust(t, z.CleanupPendingAllocation())
			serviceMust(t, z.Del(&foreign))
		} else {
			serviceMust(t, b.cleanup())
			if z.pendingCell.object != (memory.ObjectRef{}) {
				t.Fatal("matched leaf receipt not consumed")
			}
		}
		serviceMust(t, m.Close())
		serviceMust(t, z.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("full actual refund", budget.Used())
		}
	}
}

// Actual public-reallocation owner-primitive signed-domain fixture. This is
// not an ordinary mathematical OOM route: free synchronizes without limit
// transport, but signed overflow guards only after consuming the first block.
func TestScopedNumeralBufferPhysicalDestructorSignedGuardAndOwningRetry(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	var b scopedIntegerModularBuffer128
	b.construct(m)
	var input Int
	limbs := make([]uint32, 40000)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, z.SetLimbs(&input, limbs))
	serviceMust(t, b.push(&input))
	serviceMust(t, b.push(&input))
	first, e := b.get(0)
	serviceMust(t, e)
	older, e := b.get(1)
	serviceMust(t, e)
	before := r.Stats()
	contextTLSNearMinimum(t, r)
	cause := b.sourceClose()
	if !errors.Is(cause, arena.ErrBounds) || b.count != 2 || !validNumericRef(z.cells, first.cell) || !z.cell(first.cell).released {
		t.Fatal("actual first payload physically consumed before source stop", cause, b.count, r.State())
	}
	now, e := b.get(1)
	serviceMust(t, e)
	if now != older || !validNumericRef(z.cells, older.cell) || z.cell(older.cell).released || r.Stats().LiveBlocks+1 != before.LiveBlocks {
		t.Fatal("older actual completed numeral changed", now, older, r.Stats(), before)
	}
	state := r.State()
	stats := r.Stats()
	if again := b.sourceClose(); again != cause || r.State() != state || r.Stats() != stats {
		t.Fatal("source stop replayed physical accounting", again, cause)
	}
	scopedBufferAudit(t, &b, &input)
	contextTLSFinish(t, r, b.cleanup)
	integerPhysicalAudit(t, z, &input)
	contextTLSFinish(t, r, func() error { return z.Del(&input) })
	contextTLSFinish(t, r, m.Close)
	contextTLSFinish(t, r, z.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if budget.Used() != 0 {
		t.Fatal("full physical refund", budget.Used())
	}
}

func TestScopedNumeralBufferSecondGrowthEveryActualPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := NewIntegerModularManager(z)
		serviceMust(t, e)
		var b scopedIntegerModularBuffer128
		b.construct(m)
		serviceMust(t, b.resize(256))
		serviceMust(t, b.set(127, &z.two64))
		serviceMust(t, b.set(255, &z.two64))
		oldRaw := b.raw
		originals := [2]Int{}
		for j, i := range []uint32{127, 255} {
			originals[j], e = b.get(i)
			serviceMust(t, e)
		}
		scalar := NewInt(-7)
		budget.FailAfter(ordinal)
		e = b.push(&scalar)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual second expansion", ordinal, e)
		}
		if e != nil && b.count != 256 {
			t.Fatal("growth refused before source slot publication", ordinal, b.count)
		}
		if e == nil && (b.count != 257 || b.capacity != 512 || b.raw == oldRaw) {
			t.Fatal("source relocation commit", ordinal, b.count, b.capacity)
		}
		scopedBufferAudit(t, &b)
		for j, i := range []uint32{127, 255} {
			v, x := b.get(i)
			serviceMust(t, x)
			if v != originals[j] || z.cell(v.cell).size != 3 || z.digit(v.cell, 2) != 1 {
				t.Fatal("relocation changed actual source numerator", i, v, originals[j])
			}
		}
		budget.FailAfter(-1)
		serviceMust(t, b.cleanup())
		integerPhysicalAudit(t, z)
		serviceMust(t, m.Close())
		if e == nil {
			success = true
			t.Log("first second-growth success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("second growth prefix horizon")
	}
}
func TestScopedNumeralBufferBorrowedActualInlineAndRuntimeViews(t *testing.T) {
	for _, n := range []uint32{3, 130} {
		m, b := scopedBufferFixture(t)
		serviceMust(t, b.resize(n))
		for i, v := range []int32{42, 14, 28} {
			value := NewInt(v)
			serviceMust(t, b.set(uint32(i), &value))
		}
		view, e := b.view()
		serviceMust(t, e)
		if view.count != n {
			t.Fatal("source c_ptr plus logical count", view.count, n)
		}
		serviceMust(t, view.store(2, NewInt(56)))
		actual, e := b.get(2)
		serviceMust(t, e)
		if actual != NewInt(56) {
			t.Fatal("borrowed view is shadow storage", actual)
		}
		var g Int
		serviceMust(t, m.GCDManyIndexed(view, &g))
		str, e := m.manager.String(&g)
		serviceMust(t, e)
		if str != "14" {
			t.Fatal("real indexed consumer", str)
		}
		scopedBufferAudit(t, b, &g)
		serviceMust(t, m.Del(&g))
	}
}
