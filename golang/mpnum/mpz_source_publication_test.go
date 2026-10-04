// Licensed under the MIT license.
// Source action-prefix regressions: src/util/mpz.h:156–163 and
// src/util/mpz.cpp:274–279, 315–334, 1183–1207, 1556–1594.
package mpnum

import (
	"testing"

	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// This fixture deliberately uses real manager storage and reservation faults.
// Observe each failing action before Del, String, disabling the fault, or Close.
func mpzPublicationOwner(t *testing.T) (*IntegerManager, *arena.Budget) {
	return mpzPublicationOwnerProfile(t, memory.SmallAllocatorProfile{})
}

// The genuine source Debug NonWindows direct branch makes each new physical
// object a real cold allocation, even when release pool classes are warmed.
// Used only for action-timing fault fixtures; default pooled tests remain.
func mpzPublicationDirectOwner(t *testing.T) (*IntegerManager, *arena.Budget) {
	return mpzPublicationOwnerProfile(t, memory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
}
func mpzPublicationOwnerProfile(t *testing.T, profile memory.SmallAllocatorProfile) (*IntegerManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	m, err := NewIntegerManager(IntegerConfig{
		Cells:   arena.Config{PageSize: 2, Budget: b},
		Profile: profile,
		Words:   arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		m.Close()
		if b.Used() != 0 {
			t.Errorf("publication fixture leaked %d bytes", b.Used())
		}
	})
	return m, b
}

type mpzPublicationRoots struct {
	cells, words uint32
	bytes        uint64
	physical     uint64
	chunks       uint32
}

func mpzPublicationSnapshot(m *IntegerManager, b *arena.Budget) mpzPublicationRoots {
	return mpzPublicationRoots{cells: m.cells.Stats().Live, words: m.words.Stats().Live, bytes: b.Used(), physical: m.memory.Stats().PhysicalBytes, chunks: m.allocator.Stats().Chunks}
}

func mpzPublicationExpectFreed(t *testing.T, m *IntegerManager, b *arena.Budget, old numericCellRef, before mpzPublicationRoots) {
	t.Helper()
	after := mpzPublicationSnapshot(m, b)
	if validNumericRef(m.cells, old) {
		t.Error("source old cell is still live after free-before-allocate failure")
	}
	if after.cells+1 != before.cells || after.words != before.words {
		t.Errorf("source old ownership not released: before=%+v after=%+v", before, after)
	}
	// Release pooling retains the actual chunk. Debug NonWindows allocates
	// these objects directly and refunds their real backing/initialized bits.
	if before.chunks != 0 {
		if after.bytes != before.bytes || after.physical != before.physical {
			t.Errorf("pooled source chunk unexpectedly changed: before=%+v after=%+v", before, after)
		}
	} else {
		extent := uint64(32) + uint64(m.memory.SizeTBits()/8)
		initBytes := ((extent + 63) / 64) * 8
		if after.physical+extent != before.physical || after.bytes+extent+initBytes != before.bytes {
			t.Errorf("debug direct free prefix differs: before=%+v after=%+v", before, after)
		}
	}

}

func TestIntegerSourceTmpFreeBeforeReplacementFailure(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	old := m.tmp[0]
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.ensureTmp(0, 33)
	if before.chunks == 0 {
		failure, ok := err.(memory.Failure)
		if !ok || failure.Kind != memory.FailureMemout || failure.ErrorCode() != 101 {
			t.Fatalf("expected source physical Memout101, got %v", err)
		}
	} else if err != arena.ErrInjectedFailure {
		t.Fatalf("expected cold replacement locator reservation refusal, got %v", err)
	}

	// Source deallocate(m_tmp[0]) precedes the replacement request. Its stale
	// pointer cannot be dereferenced after failure; a safe Go missing-root
	// continuation may clear the field, but must not retain the old live cell.
	mpzPublicationExpectFreed(t, m, b, old, before)
}

func TestIntegerSourceResultPublishedBeforeMissingTmpFailure(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	oldTmp := m.tmp[0]
	before := mpzPublicationSnapshot(m, b)
	out, one := NewInt(17), NewInt(1)
	b.FailAfter(0)
	err := m.Add(&m.two64, &one, &out)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected missing scratch refusal, got %v", err)
	}
	// mpz.cpp publishes sign, swaps the output pointer and publishes size
	// before requesting replacement scratch. The exact result is 2^64+1.
	if out.cell != oldTmp || out.val != 1 || !validNumericRef(m.cells, out.cell) {
		t.Errorf("result transfer did not precede failure: out=%+v oldTmp=%v", out, oldTmp)
	} else {
		c := m.cell(out.cell)
		if c.size != 3 || m.digit(out.cell, 0) != 1 || m.digit(out.cell, 1) != 0 || m.digit(out.cell, 2) != 1 {
			t.Error("published result header/digits differ from source arithmetic prefix")
		}
	}
	if m.tmp[0] != 0 {
		t.Errorf("missing replacement scratch is not empty after source swap: %v", m.tmp[0])
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Errorf("transfer must neither lose nor create an owning cell: before=%+v after=%+v", before, after)
	}
}

func TestIntegerSourceBigSetFreesOldBeforeAllocationFailure(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	var dst, src Int
	if err := m.SetInt64(&dst, 1<<40); err != nil {
		t.Fatal(err)
	}
	var digits [33]uint32
	digits[0], digits[32] = 7, 1
	if err := m.SetLimbs(&src, digits[:]); err != nil {
		t.Fatal(err)
	}
	if err := m.Neg(&src); err != nil {
		t.Fatal(err)
	}
	old := dst.cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.Copy(&dst, &src)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected big_set allocation refusal, got %v", err)
	}
	// Source big_set publishes target.m_val before releasing insufficient
	// old storage and requesting capacity(source), not merely size(source).
	if dst.val != src.val {
		t.Errorf("source sign publication missing: dst=%d src=%d", dst.val, src.val)
	}
	mpzPublicationExpectFreed(t, m, b, old, before)
}

func TestIntegerSourceDigitSetterFreesOldBeforeAllocationFailure(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var dst Int
	if err := m.SetInt64(&dst, -(1 << 40)); err != nil {
		t.Fatal(err)
	}
	old := dst.cell
	before := mpzPublicationSnapshot(m, b)
	var digits [33]uint32
	digits[0], digits[32] = 9, 1
	b.FailAfter(0)
	err := m.SetLimbs(&dst, digits[:])
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected digit setter allocation refusal, got %v", err)
	}
	if dst.val != 1 {
		t.Errorf("source positive-sign publication missing: %d", dst.val)
	}
	mpzPublicationExpectFreed(t, m, b, old, before)
}

func TestIntegerSourceEnsureCapacityRetainsOldUntilReplacement(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var a Int
	if err := m.SetInt64(&a, -(1 << 40)); err != nil {
		t.Fatal(err)
	}
	old := a.cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.ensureOwnedCapacity(&a, 33)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected replacement allocation refusal, got %v", err)
	}
	if a.cell != old || a.val != -1 || !validNumericRef(m.cells, old) {
		t.Fatalf("ensure_capacity changed old ownership before allocation: %+v", a)
	}
	if c := m.cell(old); c.size != 2 || c.capacity != 6 || m.digit(old, 0) != 0 || m.digit(old, 1) != 256 {
		t.Error("retained old header/digits changed")
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Errorf("ensure_capacity changed roots on failed preparation: before=%+v after=%+v", before, after)
	}
}

func TestIntegerSourceFailedDescriptorConstructorAndUnwind(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	if m.failedRef == 0 || validNumericRef(m.cells, m.failedRef) {
		t.Fatal("missing permanently dead source-failure descriptor")
	}
	if m.cells.Stats().Retired != 2 || m.cells.Stats().Live != 6 {
		t.Fatal("failed descriptor was counted as a source live cell", m.cells.Stats())
	}
	before := mpzPublicationSnapshot(m, b)
	v := Int{val: -1, cell: m.failedRef}
	if _, e := m.Int64(&v); e != arena.ErrInvalidSlot {
		t.Fatal("failed source value read", e)
	}
	if e := m.Del(&v); e != nil || v.cell != 0 || v.val != -1 {
		t.Fatal("dead-token cleanup", e, v)
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Fatal("dead-token Del released real storage", before, after)
	}
	// Cleanup must not accept an ordinary invalid scalar as this token.
	bad := Int{val: 1, cell: numericCellRef(^uint32(0))}
	if e := m.Del(&bad); e != arena.ErrInvalidSlot || bad.cell == 0 {
		t.Fatal("ordinary invalid token was mistaken for failedRef", e, bad)
	}
	completed := false
	for fault := int64(0); fault < 64; fault++ {
		budget := arena.NewBudget(1 << 20)
		budget.FailAfter(fault)
		owner, e := NewIntegerManager(IntegerConfig{
			Cells: arena.Config{PageSize: 2, Budget: budget},
			Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}},
		})
		if e != nil {
			if owner != nil {
				t.Fatal("failed constructor published usable owner", fault)
			}
			var partial ConstructionFailure
			if errors.As(e, &partial) {
				if partial.OriginalCause() == nil || partial.Unwrap() != partial.OriginalCause() || !numericPhysicalReservationRefusal(partial.OriginalCause()) {
					t.Fatal("retained constructor cause", fault, e)
				}
				retained := partial.(*IntegerManager)
				if budget.Used() == 0 || retained.RuntimeOwner() == nil || (retained.RuntimeOwner().Stats().LiveBlocks == 0 && retained.CleanupCause() == nil) {
					t.Fatal("retained owner has no actual cleanup roots", fault, e)
				}
				budget.FailAfter(-1)
				if cleanup := partial.Cleanup(); cleanup != nil {
					t.Fatal("explicit failed constructor cleanup", fault, cleanup)
				}
			}
			if budget.Used() != 0 {
				t.Fatal("constructor dead-ledger unwind", fault, e, budget.Used())
			}
			continue
		}
		if owner.failedRef == 0 || validNumericRef(owner.cells, owner.failedRef) {
			t.Fatal("constructor published live failed token", fault)
		}
		owner.Close()
		if budget.Used() != 0 {
			t.Fatal("constructor ledger cleanup", fault, budget.Used())
		}
		completed = true
		break
	}
	if !completed {
		t.Fatal("constructor failure sweep never reached success")
	}
}

func TestIntegerSourceSmallDestinationFailureKeepsSignAndInvalidToken(t *testing.T) {
	for _, copyRoute := range []bool{false, true} {
		// Actual supported direct profile pins each source allocation action.
		m, b := mpzPublicationDirectOwner(t)
		var digits [33]uint32
		digits[0], digits[32] = 3, 1
		var src Int
		if copyRoute {
			if e := m.SetLimbs(&src, digits[:]); e != nil {
				t.Fatal(e)
			}
			if e := m.Neg(&src); e != nil {
				t.Fatal(e)
			}
		}
		before := mpzPublicationSnapshot(m, b)
		dst := NewInt(17)
		b.FailAfter(0)
		var e error
		if copyRoute {
			e = m.Copy(&dst, &src)
		} else {
			e = m.SetLimbs(&dst, digits[:])
		}
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal(copyRoute, e)
		}
		// Observe the entire named physical graph while the refusal is active.
		integerPhysicalAudit(t, m, &src)
		wantSign := int32(1)
		if copyRoute {
			wantSign = -1
		}
		if dst.val != wantSign || dst.cell != m.failedNullRef {
			t.Fatal("small destination source sign/token prefix", copyRoute, dst)
		}
		if _, e = m.Int64(&dst); e != arena.ErrInvalidSlot {
			t.Fatal("failed source publication read as an inline numeral", copyRoute, e)
		}
		if after := mpzPublicationSnapshot(m, b); after != before {
			t.Fatal("failed small publication roots", copyRoute, before, after)
		}
		if e = m.Reset(&dst); e != nil || dst != (Int{}) {
			t.Fatal("failed-token Reset", copyRoute, e, dst)
		}
		if after := mpzPublicationSnapshot(m, b); after != before {
			t.Fatal("failed-token Reset released unrelated storage", before, after)
		}
	}
}

func TestIntegerSourcePairedDivisionPublishesQuotientBeforeMissingScratch(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	q, r, one := NewInt(19), NewInt(23), NewInt(1)
	oldTmp := m.tmp[0]
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	e := m.MachineDivRem(&m.two64, &one, &q, &r)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	if q.cell != oldTmp || q.val != 1 || !validNumericRef(m.cells, q.cell) {
		t.Fatal("quotient was not transferred before its scratch allocation", q, oldTmp)
	}
	if c := m.cell(q.cell); c.size != 3 || m.digit(q.cell, 0) != 0 || m.digit(q.cell, 1) != 0 || m.digit(q.cell, 2) != 1 {
		t.Fatal("source quotient prefix differs")
	}
	if r != NewInt(23) || m.tmp[0] != 0 {
		t.Fatal("remainder setter ran before failing quotient setter completed", r, m.tmp[0])
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Fatal("paired publication roots", before, after)
	}
	if e = m.ensureTmp(0, 3); e != arena.ErrInvalidSlot {
		t.Fatal("missing scratch access was not guarded", e)
	}
	// A missing manager scratch must not poison the already-published result.
	if e = m.Del(&q); e != nil {
		t.Fatal("published quotient cleanup", e)
	}
}

func TestIntegerSourceTmpUnsignedCapacityGuardAfterOldFree(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	old := m.tmp[0]
	oldObject := m.cell(old).backing.object
	oldGeneration := m.cells.state(numericSlot(old)).generation
	before := mpzPublicationSnapshot(m, b)
	oldRawExtent := uint64(32) + uint64(m.memory.SizeTBits()/8)
	// mpz.h:159–162 frees the old object, computes unsigned new capacity,
	// then allocate writes capacity before its caller publishes the pointer.
	// UINT_MAX -> INT_MAX capacity -> unsigned cell_size 4 (mpz.h:122).
	e := m.ensureTmp(0, ^uint32(0))
	if m.cells.IsLive(numericSlot(old), oldGeneration) {
		t.Fatal("old descriptor generation survived destructive replacement")
	}
	if _, _, err := m.allocator.RawLocation(oldObject); err != memory.ErrAllocatorOwner {
		t.Fatal("old actual object remained accessible", err)
	}
	if m.pendingCell.object != (memory.ObjectRef{}) {
		// Debug NonWindows uses exact-size direct backing; ILP32 pooling also
		// rounds four bytes to four. The capacity word is outside this object.
		pending := m.pendingCell
		extent, err := m.allocator.ObjectExtent(pending.object)
		if err != nil || extent != 4 || pending.requested != 4 || e != arena.ErrBounds {
			t.Fatal("actual endangered capacity write", extent, e, err)
		}
		if m.tmp[0] != 0 || m.cells.Stats().Live+1 != before.cells {
			t.Fatal("undefined capacity access published descriptor/tmp")
		}
		if _, err = pending.readWord(m.allocator, 0); !errors.Is(err, memory.ErrRuntimeUninitialized) {
			t.Fatal("pending source m_size initialized", err)
		}
		if _, err = pending.readWord(m.allocator, 4); err != arena.ErrBounds {
			t.Fatal("capacity word escaped rounded object", err)
		}
		if before.chunks == 0 {
			physical := m.memory.Stats().PhysicalBytes
			newRawExtent := uint64(4) + uint64(m.memory.SizeTBits()/8)
			if physical+oldRawExtent != before.physical+newRawExtent {
				t.Fatal("actual free/allocation physical prefix", before.physical, physical)
			}
		}
	} else {
		// LP64 pooling rounds request4 to8; capacity succeeds and the new tmp
		// pointer is published before the source debug capacity assertion.
		if modularSourceDebug {
			if e != ErrIntegerAssertion {
				t.Fatal("source post-publication capacity assertion", e)
			}
		} else if e != nil {
			t.Fatal("defined wrapped LP64 allocation", e)
		}
		ref := m.tmp[0]
		if ref == 0 || !validNumericRef(m.cells, ref) {
			t.Fatal("new source pointer not live")
		}
		c := m.cell(ref)
		extent, err := m.allocator.ObjectExtent(c.backing.object)
		if err != nil || c.capacity != 0x7fffffff || c.backing.requested != 4 || extent != 8 {
			t.Fatal("source unsigned capacity/actual rounding", c, extent, err)
		}
		if v, err := c.backing.readWord(m.allocator, 4); err != nil || v != 0x7fffffff {
			t.Fatal("physical capacity publication", v, err)
		}
		if _, err := c.backing.readWord(m.allocator, 0); !errors.Is(err, memory.ErrRuntimeUninitialized) {
			t.Fatal("replacement source size synthesized", err)
		}
		after := mpzPublicationSnapshot(m, b)
		if after.cells != before.cells || after.words != before.words || after.physical != before.physical+8192+uint64(m.memory.SizeTBits()/8) {
			t.Fatal("actual new class chunk prefix", before, after)
		}
		if e = m.prepareTmpSpan(0, ^uint32(0)); e != arena.ErrBounds {
			t.Fatal("guard at actual oversized span access", e)
		}
		if m.tmp[0] != ref || mpzPublicationSnapshot(m, b) != after || m.cell(ref).span != 0 {
			t.Fatal("span refusal changed completed roots")
		}
	}
	if m.words.Stats().Live != before.words {
		t.Fatal("numeric source allocation fabricated Words root")
	}
	if e = m.CloseChecked(); e != nil {
		t.Fatal("owning physical-prefix cleanup", e)
	}
	if b.Used() != 0 {
		t.Fatal("retained source root leaked", b.Used())
	}
}

func TestIntegerSourceDivModWrapperDestinationBeforeFailure(t *testing.T) {
	for _, modRoute := range []bool{false, true} {
		m, b := mpzPublicationDirectOwner(t)
		var a, divisor Int
		if e := m.Copy(&a, &m.two64); e != nil {
			t.Fatal(e)
		}
		if e := m.Neg(&a); e != nil {
			t.Fatal(e)
		}
		divisor = NewInt(1)
		idx := 0
		if modRoute {
			if e := m.Copy(&divisor, &m.two64); e != nil {
				t.Fatal(e)
			}
			if e := m.Inc(&divisor); e != nil {
				t.Fatal(e)
			}
			idx = 1
		}
		out := NewInt(19)
		oldTmp := m.tmp[idx]
		before := mpzPublicationSnapshot(m, b)
		b.FailAfter(0)
		var e error
		if modRoute {
			e = m.Mod(&a, &divisor, &out)
		} else {
			e = m.Div(&a, &divisor, &out)
		}
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal(modRoute, e)
		}
		if out.cell != oldTmp || out.val != -1 || !validNumericRef(m.cells, out.cell) {
			t.Fatal("wrapper hid source destination prefix in a local work value", modRoute, out, oldTmp)
		}
		if m.tmp[idx] != 0 {
			t.Fatal("failed wrapper replacement did not leave missing scratch", modRoute)
		}
		if after := mpzPublicationSnapshot(m, b); after != before {
			t.Fatal("wrapper cleanup destroyed transferred caller root", modRoute, before, after)
		}
	}
}

func TestIntegerSourceTmpFiniteWrappedCapacities(t *testing.T) {
	for _, f := range []struct{ request, capacity uint32 }{{1431655767, 3}, {1431655773, 12}} {
		m, b := mpzPublicationOwner(t)
		before := mpzPublicationSnapshot(m, b)
		e := m.ensureTmp(0, f.request)
		if modularSourceDebug {
			if e == nil {
				t.Fatal("missing source capacity assertion", f)
			}
			if f.capacity < 6 {
				if m.tmp[0] != 0 || m.cells.Stats().Live+1 != before.cells {
					t.Fatal("allocate capacity assertion did not follow old-free and precede replacement", f)
				}
				continue
			}
		} else if e != nil {
			t.Fatal("defined release finite wrapped allocation rejected", f, e)
		}
		if m.tmp[0] == 0 || m.cell(m.tmp[0]).capacity != f.capacity || m.cells.Stats().Live != before.cells {
			t.Fatal("unsigned32 wrapped capacity/publication", f, m.tmp[0], m.cells.Stats())
		}
	}
}

// Cleanup is a Go continuation for a failed source allocation, not a read.
func mpzPublicationFailLeaf(t *testing.T, z *IntegerManager, b *arena.Budget, leaf *Int) {
	t.Helper()
	if e := z.SetInt64(leaf, 1<<40); e != nil {
		t.Fatal(e)
	}
	var digits [33]uint32
	digits[0], digits[32] = 9, 1
	b.FailAfter(0)
	if e := z.SetLimbs(leaf, digits[:]); !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	if leaf.cell != z.failedRef {
		t.Fatal("missing failed leaf", *leaf)
	}
	if e := z.check(leaf); e != arena.ErrInvalidSlot {
		t.Fatal("read accepted failed leaf", e)
	}
}
func TestIntegerSourcePartialRationalCleanup(t *testing.T) {
	for _, reset := range []bool{false, true} {
		b := arena.NewBudget(1 << 20)
		q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
		if e != nil {
			t.Fatal(e)
		}
		a := NewRational()
		if e = q.z.SetInt64(&a.den, 1<<40); e != nil {
			t.Fatal(e)
		}
		mpzPublicationFailLeaf(t, q.z, b, &a.num)
		companionRef := a.den.cell
		before := mpzPublicationSnapshot(q.z, b)
		if q.check(&a) != arena.ErrInvalidSlot {
			t.Fatal("rational read guard changed")
		}
		if reset {
			e = q.Reset(&a)
		} else {
			e = q.Del(&a)
		}
		if e != nil {
			t.Fatal("partial cleanup rejected", reset, e)
		}
		integerPhysicalAudit(t, q.z)
		mpzPublicationExpectFreed(t, q.z, b, companionRef, before)
		after := mpzPublicationSnapshot(q.z, b)
		if after.cells+1 != before.cells || after.words != before.words || a.num.cell != 0 || a.den.cell != 0 {
			t.Fatal("live companion not released exactly once", before, after, a)
		}
		if reset && (a.num.val != 0 || a.den.val != 1) {
			t.Fatal("reset shape", a)
		}
		stale := Rational{num: Int{cell: numericCellRef(^uint32(0)), val: 1}, den: NewInt(1)}
		if q.Del(&stale) != arena.ErrInvalidSlot {
			t.Fatal("ordinary stale accepted")
		}
		b.FailAfter(-1)
		q.Close()
		if b.Used() != 0 {
			t.Fatal("owner leak", b.Used())
		}
	}
}
func TestIntegerSourcePartialBinaryCleanup(t *testing.T) {
	for _, reset := range []bool{false, true} {
		z, b := mpzPublicationDirectOwner(t)
		diag, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
		if e != nil {
			t.Fatal(e)
		}
		bm, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diag})
		if e != nil {
			t.Fatal(e)
		}
		im, e := NewBinaryIntervalManager(bm)
		if e != nil {
			t.Fatal(e)
		}
		a := NewBinaryInterval()
		a.lower.k = 7
		a.upper.k = 9
		if e = z.SetInt64(&a.upper.num, 1<<40); e != nil {
			t.Fatal(e)
		}
		mpzPublicationFailLeaf(t, z, b, &a.lower.num)
		companionRef := a.upper.num.cell
		before := mpzPublicationSnapshot(z, b)
		if im.check(&a) != arena.ErrInvalidSlot {
			t.Fatal("interval read guard changed")
		}
		if reset {
			e = im.Reset(&a)
		} else {
			e = im.Del(&a)
		}
		if e != nil {
			t.Fatal("partial interval cleanup rejected", reset, e)
		}
		integerPhysicalAudit(t, z)
		mpzPublicationExpectFreed(t, z, b, companionRef, before)
		after := mpzPublicationSnapshot(z, b)
		if after.cells+1 != before.cells || after.words != before.words || a.lower.num.cell != 0 || a.upper.num.cell != 0 {
			t.Fatal("interval companion not released", before, after, a)
		}
		if reset && (a.lower.k != 0 || a.upper.k != 0) {
			t.Fatal("reset powers", a)
		}
		foreign := BinaryRational{num: Int{cell: numericCellRef(^uint32(0)), val: 1}}
		if bm.Del(&foreign) != arena.ErrInvalidSlot {
			t.Fatal("ordinary stale binary token accepted")
		}
		im.Close()
		bm.Close()
		diag.Close()
	}
}
func TestIntegerSourcePartialOwnedRationalCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.NewValue()
	if e != nil {
		t.Fatal(e)
	}
	if e = s.manager.z.SetInt64(&v.raw.den, 1<<40); e != nil {
		t.Fatal(e)
	}
	mpzPublicationFailLeaf(t, s.manager.z, b, &v.raw.num)
	companionRef := v.raw.den.cell
	before := mpzPublicationSnapshot(s.manager.z, b)
	foreign := OwnedRational{raw: NewRational(), ownerID: s.ownerID + 1}
	if s.Del(&foreign) != ErrRationalServiceOwner {
		t.Fatal("foreign cleanup accepted")
	}
	if s.check(&v) != arena.ErrInvalidSlot {
		t.Fatal("owned read guard changed")
	}
	if e = s.Del(&v); e != nil {
		t.Fatal("owned partial cleanup rejected", e)
	}
	integerPhysicalAudit(t, s.manager.z)
	mpzPublicationExpectFreed(t, s.manager.z, b, companionRef, before)
	after := mpzPublicationSnapshot(s.manager.z, b)
	if after.cells+1 != before.cells || after.words != before.words || v.ownerID != 0 {
		t.Fatal("owned companion cleanup", before, after, v)
	}
	b.FailAfter(-1)
	s.Close()
	if b.Used() != 0 {
		t.Fatal("service leaked", b.Used())
	}
}

func TestIntegerSourceScratchSpanGuardAfterFiniteWrappedAllocation(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	// This unsigned request creates a real finite 12-word replacement before
	// the source's invalid oversized memory span would be accessed.
	e := m.ensureTmp(0, 1431655773)
	if modularSourceDebug {
		if e != ErrIntegerAssertion {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	ref := m.tmp[0]
	before := mpzPublicationSnapshot(m, b)
	size := m.cell(ref).size
	if e = m.prepareTmpSpan(0, 1431655773); e != arena.ErrBounds {
		t.Fatal("undersized actual scratch span", e)
	}
	if m.tmp[0] != ref || m.cell(ref).size != size || mpzPublicationSnapshot(m, b) != before {
		t.Fatal("span guard undid completed replacement")
	}
	if e = m.prepareTmpSpan(0, 12); e != nil {
		t.Fatal("valid replacement span", e)
	}
	c := m.cell(ref)
	if c.size != size || c.span != 12 || mpzPublicationSnapshot(m, b) != before {
		t.Fatal("span borrowed without source size publication", c)
	}
	if _, err := c.backing.readWord(m.allocator, 0); !errors.Is(err, memory.ErrRuntimeUninitialized) {
		t.Fatal("scratch borrow invented initialized m_size", err)
	}
}

func TestIntegerSourcePartialInfCleanup(t *testing.T) {
	for kind := 0; kind < 3; kind++ {
		t.Run([]string{"InfInt", "Inf", "InfEps"}[kind], func(t *testing.T) {
			b := arena.NewBudget(1 << 20)
			s, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
			if e != nil {
				t.Fatal(e)
			}
			var ii OwnedInfIntRational
			var ir OwnedInfRational
			var ie OwnedInfEpsInfRational
			var leaf, companion *Int
			switch kind {
			case 0:
				ii = s.newInfInt()
				leaf, companion = &ii.first.num, &ii.first.den
			case 1:
				ir = s.infEmpty()
				leaf, companion = &ir.first.num, &ir.second.num
			case 2:
				ie = s.infEpsInfEmpty()
				leaf, companion = &ie.infinity.num, &ie.numeral.second.num
			}
			if e = s.manager.z.SetInt64(companion, 1<<40); e != nil {
				t.Fatal(e)
			}
			mpzPublicationFailLeaf(t, s.manager.z, b, leaf)
			companionRef := companion.cell
			before := mpzPublicationSnapshot(s.manager.z, b)
			switch kind {
			case 0:
				if s.checkInfInt(&ii) != arena.ErrInvalidSlot {
					t.Fatal("InfInt read changed")
				}
				e = s.InfIntDel(&ii)
			case 1:
				if s.infCheck(&ir) != arena.ErrInvalidSlot {
					t.Fatal("Inf read changed")
				}
				e = s.InfDel(&ir)
			case 2:
				if s.infEpsInfCheck(&ie) != arena.ErrInvalidSlot {
					t.Fatal("InfEps read changed")
				}
				e = s.InfEpsInfDel(&ie)
			}
			if e != nil {
				t.Fatal("partial Inf cleanup rejected", kind, e)
			}
			integerPhysicalAudit(t, s.manager.z)
			mpzPublicationExpectFreed(t, s.manager.z, b, companionRef, before)
			after := mpzPublicationSnapshot(s.manager.z, b)
			if after.cells+1 != before.cells || after.words != before.words {
				t.Fatal("Inf companion cleanup", kind, before, after)
			}
			switch kind {
			case 0:
				if ii.ownerID != 0 {
					t.Fatal(ii)
				}
				ii = s.newInfInt()
				ii.ownerID++
				if s.InfIntDel(&ii) != ErrRationalServiceOwner {
					t.Fatal("foreign InfInt accepted")
				}
			case 1:
				if ir.ownerID != 0 {
					t.Fatal(ir)
				}
				ir = s.infEmpty()
				ir.ownerID++
				if s.InfDel(&ir) != ErrRationalServiceOwner {
					t.Fatal("foreign Inf accepted")
				}
			case 2:
				if ie.ownerID != 0 {
					t.Fatal(ie)
				}
				ie = s.infEpsInfEmpty()
				ie.numeral.ownerID++
				if s.InfEpsInfDel(&ie) != ErrRationalServiceOwner {
					t.Fatal("foreign nested Inf accepted")
				}
			}
			b.FailAfter(-1)
			s.Close()
			if b.Used() != 0 {
				t.Fatal("Inf owner leak", kind, b.Used())
			}
		})
	}
}

// Audit real named roots before disabling faults, formatting, or teardown.
// A missing scratch or the exact owned failed descriptor is not a live root.
func mpzPublicationAudit(t *testing.T, m *IntegerManager, public ...*Int) {
	t.Helper()
	integerPhysicalAudit(t, m, public...)
}
