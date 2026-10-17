// Source action-prefix regressions: src/util/mpz.h:156–163 and
// src/util/mpz.cpp:274–279, 315–334, 1183–1207, 1556–1594.
package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// This fixture deliberately uses real manager storage and reservation faults.
// Observe each failing action before Del, String, disabling the fault, or Close.
func mpzPublicationOwner(t *testing.T) (*IntegerManager, *arena.Budget) {
	return mpzPublicationOwnerProfile(t, umemory.SmallAllocatorProfile{})
}

// The genuine source Debug NonWindows direct branch makes each new physical
// object a real cold allocation, even when release pool classes are warmed.
// Used only for action-timing fault fixtures; default pooled tests remain.
func mpzPublicationDirectOwner(t *testing.T) (*IntegerManager, *arena.Budget) {
	return mpzPublicationOwnerProfile(t, umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
}
func mpzPublicationOwnerProfile(t *testing.T, profile umemory.SmallAllocatorProfile) (*IntegerManager, *arena.Budget) {
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
	cells    uint32
	bytes    uint64
	physical uint64
	chunks   uint32
}

func mpzPublicationSnapshot(m *IntegerManager, b *arena.Budget) mpzPublicationRoots {
	return mpzPublicationRoots{cells: m.Cells.Stats().Live, bytes: b.Used(), physical: m.Memory.Stats().PhysicalBytes, chunks: m.Allocator.Stats().Chunks}
}

func mpzPublicationExpectFreed(t *testing.T, m *IntegerManager, b *arena.Budget, old NumericCellRef, before mpzPublicationRoots) {
	t.Helper()
	after := mpzPublicationSnapshot(m, b)
	if ValidNumericRef(m.Cells, old) {
		t.Error("source old cell is still live after free-before-allocate failure")
	}
	if after.cells+1 != before.cells {
		t.Errorf("source old ownership not released: before=%+v after=%+v", before, after)
	}
	// Release pooling retains the actual chunk. Debug NonWindows allocates
	// these objects directly and refunds their real backing/initialized bits.
	if before.chunks != 0 {
		if after.bytes != before.bytes || after.physical != before.physical {
			t.Errorf("pooled source chunk unexpectedly changed: before=%+v after=%+v", before, after)
		}
	} else {
		extent := uint64(32) + uint64(m.Memory.SizeTBits()/8)
		initBytes := ((extent + 63) / 64) * 8
		if after.physical+extent != before.physical || after.bytes+extent+initBytes != before.bytes {
			t.Errorf("debug direct free prefix differs: before=%+v after=%+v", before, after)
		}
	}

}

func TestIntegerSourceTmpFreeBeforeReplacementFailure(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	old := m.Tmp[0]
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.EnsureTmp(0, 33)
	if before.chunks == 0 {
		failure, ok := err.(umemory.Failure)
		if !ok || failure.Kind != umemory.FailureMemout || failure.ErrorCode() != 101 {
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
	oldTmp := m.Tmp[0]
	before := mpzPublicationSnapshot(m, b)
	out, one := NewInt(17), NewInt(1)
	b.FailAfter(0)
	err := m.Add(&m.Two64, &one, &out)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected missing scratch refusal, got %v", err)
	}
	// mpz.cpp publishes sign, swaps the output pointer and publishes size
	// before requesting replacement scratch. The exact result is 2^64+1.
	if out.Cell != oldTmp || out.Val != 1 || !ValidNumericRef(m.Cells, out.Cell) {
		t.Errorf("result transfer did not precede failure: out=%+v oldTmp=%v", out, oldTmp)
	} else {
		c := m.Cell(out.Cell)
		if c.Size != 3 || m.Digit(out.Cell, 0) != 1 || m.Digit(out.Cell, 1) != 0 || m.Digit(out.Cell, 2) != 1 {
			t.Error("published result header/digits differ from source arithmetic prefix")
		}
	}
	if m.Tmp[0] != 0 {
		t.Errorf("missing replacement scratch is not empty after source swap: %v", m.Tmp[0])
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
	old := dst.Cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.Copy(&dst, &src)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected big_set allocation refusal, got %v", err)
	}
	// Source big_set publishes target.m_val before releasing insufficient
	// old storage and requesting capacity(source), not merely size(source).
	if dst.Val != src.Val {
		t.Errorf("source sign publication missing: dst=%d src=%d", dst.Val, src.Val)
	}
	mpzPublicationExpectFreed(t, m, b, old, before)
}

func TestIntegerSourceDigitSetterFreesOldBeforeAllocationFailure(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var dst Int
	if err := m.SetInt64(&dst, -(1 << 40)); err != nil {
		t.Fatal(err)
	}
	old := dst.Cell
	before := mpzPublicationSnapshot(m, b)
	var digits [33]uint32
	digits[0], digits[32] = 9, 1
	b.FailAfter(0)
	err := m.SetLimbs(&dst, digits[:])
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected digit setter allocation refusal, got %v", err)
	}
	if dst.Val != 1 {
		t.Errorf("source positive-sign publication missing: %d", dst.Val)
	}
	mpzPublicationExpectFreed(t, m, b, old, before)
}

func TestIntegerSourceEnsureCapacityRetainsOldUntilReplacement(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var a Int
	if err := m.SetInt64(&a, -(1 << 40)); err != nil {
		t.Fatal(err)
	}
	old := a.Cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	err := m.EnsureOwnedCapacity(&a, 33)
	if !numericPhysicalReservationRefusal(err) {
		t.Fatalf("expected replacement allocation refusal, got %v", err)
	}
	if a.Cell != old || a.Val != -1 || !ValidNumericRef(m.Cells, old) {
		t.Fatalf("ensure_capacity changed old ownership before allocation: %+v", a)
	}
	if c := m.Cell(old); c.Size != 2 || c.Capacity != 6 || m.Digit(old, 0) != 0 || m.Digit(old, 1) != 256 {
		t.Error("retained old header/digits changed")
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Errorf("ensure_capacity changed roots on failed preparation: before=%+v after=%+v", before, after)
	}
}

func TestIntegerSourceFailedDescriptorConstructorAndUnwind(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	if m.FailedRef == 0 || ValidNumericRef(m.Cells, m.FailedRef) {
		t.Fatal("missing permanently dead source-failure descriptor")
	}
	if m.Cells.Stats().Retired != 2 || m.Cells.Stats().Live != 6 {
		t.Fatal("failed descriptor was counted as a source live cell", m.Cells.Stats())
	}
	before := mpzPublicationSnapshot(m, b)
	v := Int{Val: -1, Cell: m.FailedRef}
	if _, e := m.Int64(&v); e != arena.ErrInvalidSlot {
		t.Fatal("failed source value read", e)
	}
	if e := m.Del(&v); e != nil || v.Cell != 0 || v.Val != -1 {
		t.Fatal("dead-token cleanup", e, v)
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Fatal("dead-token Del released real storage", before, after)
	}
	// Cleanup must not accept an ordinary invalid scalar as this token.
	bad := Int{Val: 1, Cell: NumericCellRef(^uint32(0))}
	if e := m.Del(&bad); e != arena.ErrInvalidSlot || bad.Cell == 0 {
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
			var partial umemory.ConstructionFailure
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
		if owner.FailedRef == 0 || ValidNumericRef(owner.Cells, owner.FailedRef) {
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
		if dst.Val != wantSign || dst.Cell != m.FailedNullRef {
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
	oldTmp := m.Tmp[0]
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	e := m.MachineDivRem(&m.Two64, &one, &q, &r)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	if q.Cell != oldTmp || q.Val != 1 || !ValidNumericRef(m.Cells, q.Cell) {
		t.Fatal("quotient was not transferred before its scratch allocation", q, oldTmp)
	}
	if c := m.Cell(q.Cell); c.Size != 3 || m.Digit(q.Cell, 0) != 0 || m.Digit(q.Cell, 1) != 0 || m.Digit(q.Cell, 2) != 1 {
		t.Fatal("source quotient prefix differs")
	}
	if r != NewInt(23) || m.Tmp[0] != 0 {
		t.Fatal("remainder setter ran before failing quotient setter completed", r, m.Tmp[0])
	}
	if after := mpzPublicationSnapshot(m, b); after != before {
		t.Fatal("paired publication roots", before, after)
	}
	if e = m.EnsureTmp(0, 3); e != arena.ErrInvalidSlot {
		t.Fatal("missing scratch access was not guarded", e)
	}
	// A missing manager scratch must not poison the already-published result.
	if e = m.Del(&q); e != nil {
		t.Fatal("published quotient cleanup", e)
	}
}

func TestIntegerSourceTmpUnsignedCapacityGuardAfterOldFree(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "T")
	}
	m, b := mpzPublicationOwner(t)
	old := m.Tmp[0]
	oldObject := m.Cell(old).Backing.Object
	oldGeneration := m.Cells.State(NumericSlot(old)).Generation
	before := mpzPublicationSnapshot(m, b)
	oldRawExtent := uint64(32) + uint64(m.Memory.SizeTBits()/8)
	// mpz.h:159–162 frees the old object, computes unsigned new capacity,
	// then allocate writes capacity before its caller publishes the pointer.
	// UINT_MAX -> INT_MAX capacity -> unsigned cell_size 4 (mpz.h:122).
	e := m.EnsureTmp(0, ^uint32(0))
	if m.Cells.IsLive(NumericSlot(old), oldGeneration) {
		t.Fatal("old descriptor generation survived destructive replacement")
	}
	if _, _, err := m.Allocator.RawLocation(oldObject); err != umemory.ErrAllocatorOwner {
		t.Fatal("old actual object remained accessible", err)
	}
	if m.PendingCell.Object != (umemory.ObjectRef{}) {
		// Debug NonWindows uses exact-size direct backing; ILP32 pooling also
		// rounds four bytes to four. The capacity word is outside this object.
		pending := m.PendingCell
		extent, err := m.Allocator.ObjectExtent(pending.Object)
		if err != nil || extent != 4 || pending.Requested != 4 || e != arena.ErrBounds {
			t.Fatal("actual endangered capacity write", extent, e, err)
		}
		if m.Tmp[0] != 0 || m.Cells.Stats().Live+1 != before.cells {
			t.Fatal("undefined capacity access published descriptor/tmp")
		}
		if _, err = pending.ReadWord(m.Allocator, 0); !errors.Is(err, umemory.ErrRuntimeUninitialized) {
			t.Fatal("pending source m_size initialized", err)
		}
		if _, err = pending.ReadWord(m.Allocator, 4); err != arena.ErrBounds {
			t.Fatal("capacity word escaped rounded object", err)
		}
		if before.chunks == 0 {
			physical := m.Memory.Stats().PhysicalBytes
			newRawExtent := uint64(4) + uint64(m.Memory.SizeTBits()/8)
			if physical+oldRawExtent != before.physical+newRawExtent {
				t.Fatal("actual free/allocation physical prefix", before.physical, physical)
			}
		}
	} else {
		// LP64 pooling rounds request4 to8; capacity succeeds and the new tmp
		// pointer is published before the source debug capacity assertion.
		if modularSourceDebug {
			if !isMPNVerificationThrow(e) {
				t.Fatal("source post-publication capacity assertion", e)
			}
		} else if e != nil {
			t.Fatal("defined wrapped LP64 allocation", e)
		}
		ref := m.Tmp[0]
		if ref == 0 || !ValidNumericRef(m.Cells, ref) {
			t.Fatal("new source pointer not live")
		}
		c := m.Cell(ref)
		extent, err := m.Allocator.ObjectExtent(c.Backing.Object)
		if err != nil || c.Capacity != 0x7fffffff || c.Backing.Requested != 4 || extent != 8 {
			t.Fatal("source unsigned capacity/actual rounding", c, extent, err)
		}
		if v, err := c.Backing.ReadWord(m.Allocator, 4); err != nil || v != 0x7fffffff {
			t.Fatal("physical capacity publication", v, err)
		}
		if _, err := c.Backing.ReadWord(m.Allocator, 0); !errors.Is(err, umemory.ErrRuntimeUninitialized) {
			t.Fatal("replacement source size synthesized", err)
		}
		after := mpzPublicationSnapshot(m, b)
		if after.cells != before.cells || after.physical != before.physical+8192+uint64(m.Memory.SizeTBits()/8) {
			t.Fatal("actual new class chunk prefix", before, after)
		}
		if e = m.prepareTmpSpan(0, ^uint32(0)); e != arena.ErrBounds {
			t.Fatal("guard at actual oversized span access", e)
		}
		if m.Tmp[0] != ref || mpzPublicationSnapshot(m, b) != after || m.Cell(ref).span != 0 {
			t.Fatal("span refusal changed completed roots")
		}
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
		if e := m.Copy(&a, &m.Two64); e != nil {
			t.Fatal(e)
		}
		if e := m.Neg(&a); e != nil {
			t.Fatal(e)
		}
		divisor = NewInt(1)
		idx := 0
		if modRoute {
			if e := m.Copy(&divisor, &m.Two64); e != nil {
				t.Fatal(e)
			}
			if e := m.Inc(&divisor); e != nil {
				t.Fatal(e)
			}
			idx = 1
		}
		out := NewInt(19)
		oldTmp := m.Tmp[idx]
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
		if out.Cell != oldTmp || out.Val != -1 || !ValidNumericRef(m.Cells, out.Cell) {
			t.Fatal("wrapper hid source destination prefix in a local work value", modRoute, out, oldTmp)
		}
		if m.Tmp[idx] != 0 {
			t.Fatal("failed wrapper replacement did not leave missing scratch", modRoute)
		}
		if after := mpzPublicationSnapshot(m, b); after != before {
			t.Fatal("wrapper cleanup destroyed transferred caller root", modRoute, before, after)
		}
	}
}

func TestIntegerSourceTmpFiniteWrappedCapacities(t *testing.T) {
	for _, f := range []struct{ request, capacity uint32 }{{1431655767, 3}, {1431655773, 12}} {
		if modularSourceDebug {
			bindMPNDebugActions(t, "T")
		}
		m, b := mpzPublicationOwner(t)
		before := mpzPublicationSnapshot(m, b)
		e := m.EnsureTmp(0, f.request)
		if modularSourceDebug {
			if e == nil {
				t.Fatal("missing source capacity assertion", f)
			}
			if f.capacity < 6 {
				if !isMPNVerificationThrow(e) {
					t.Fatal("source allocate assertion action", f, e)
				}
				if m.Tmp[0] != 0 || m.Cells.Stats().Live+1 != before.cells {
					t.Fatal("allocate capacity assertion did not follow old-free and precede replacement", f)
				}
				continue
			}
			if !isMPNVerificationThrow(e) {
				t.Fatal("source post-publication capacity assertion action", f, e)
			}
		} else if e != nil {
			t.Fatal("defined release finite wrapped allocation rejected", f, e)
		}
		if m.Tmp[0] == 0 || m.Cell(m.Tmp[0]).Capacity != f.capacity || m.Cells.Stats().Live != before.cells {
			t.Fatal("unsigned32 wrapped capacity/publication", f, m.Tmp[0], m.Cells.Stats())
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
	if leaf.Cell != z.FailedRef {
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
		if e = q.Z.SetInt64(&a.Den, 1<<40); e != nil {
			t.Fatal(e)
		}
		mpzPublicationFailLeaf(t, q.Z, b, &a.Num)
		companionRef := a.Den.Cell
		before := mpzPublicationSnapshot(q.Z, b)
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
		integerPhysicalAudit(t, q.Z)
		mpzPublicationExpectFreed(t, q.Z, b, companionRef, before)
		after := mpzPublicationSnapshot(q.Z, b)
		if after.cells+1 != before.cells || a.Num.Cell != 0 || a.Den.Cell != 0 {
			t.Fatal("live companion not released exactly once", before, after, a)
		}
		if reset && (a.Num.Val != 0 || a.Den.Val != 1) {
			t.Fatal("reset shape", a)
		}
		stale := Rational{Num: Int{Cell: NumericCellRef(^uint32(0)), Val: 1}, Den: NewInt(1)}
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
		a.lower.KField = 7
		a.upper.KField = 9
		if e = z.SetInt64(&a.upper.Num, 1<<40); e != nil {
			t.Fatal(e)
		}
		mpzPublicationFailLeaf(t, z, b, &a.lower.Num)
		companionRef := a.upper.Num.Cell
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
		if after.cells+1 != before.cells || a.lower.Num.Cell != 0 || a.upper.Num.Cell != 0 {
			t.Fatal("interval companion not released", before, after, a)
		}
		if reset && (a.lower.KField != 0 || a.upper.KField != 0) {
			t.Fatal("reset powers", a)
		}
		foreign := BinaryRational{Num: Int{Cell: NumericCellRef(^uint32(0)), Val: 1}}
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
	if e = s.Manager.Z.SetInt64(&v.Raw.Den, 1<<40); e != nil {
		t.Fatal(e)
	}
	mpzPublicationFailLeaf(t, s.Manager.Z, b, &v.Raw.Num)
	companionRef := v.Raw.Den.Cell
	before := mpzPublicationSnapshot(s.Manager.Z, b)
	foreign := OwnedRational{Raw: NewRational(), ownerID: s.ownerID + 1}
	if s.Del(&foreign) != ErrRationalServiceOwner {
		t.Fatal("foreign cleanup accepted")
	}
	if s.check(&v) != arena.ErrInvalidSlot {
		t.Fatal("owned read guard changed")
	}
	if e = s.Del(&v); e != nil {
		t.Fatal("owned partial cleanup rejected", e)
	}
	integerPhysicalAudit(t, s.Manager.Z)
	mpzPublicationExpectFreed(t, s.Manager.Z, b, companionRef, before)
	after := mpzPublicationSnapshot(s.Manager.Z, b)
	if after.cells+1 != before.cells || v.ownerID != 0 {
		t.Fatal("owned companion cleanup", before, after, v)
	}
	b.FailAfter(-1)
	s.Close()
	if b.Used() != 0 {
		t.Fatal("service leaked", b.Used())
	}
}

func TestIntegerSourceScratchSpanGuardAfterFiniteWrappedAllocation(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "T")
	}
	m, b := mpzPublicationOwner(t)
	// This unsigned request creates a real finite 12-word replacement before
	// the source's invalid oversized memory span would be accessed.
	e := m.EnsureTmp(0, 1431655773)
	if modularSourceDebug {
		if !isMPNVerificationThrow(e) {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	ref := m.Tmp[0]
	before := mpzPublicationSnapshot(m, b)
	size := m.Cell(ref).Size
	if e = m.prepareTmpSpan(0, 1431655773); e != arena.ErrBounds {
		t.Fatal("undersized actual scratch span", e)
	}
	if m.Tmp[0] != ref || m.Cell(ref).Size != size || mpzPublicationSnapshot(m, b) != before {
		t.Fatal("span guard undid completed replacement")
	}
	if e = m.prepareTmpSpan(0, 12); e != nil {
		t.Fatal("valid replacement span", e)
	}
	c := m.Cell(ref)
	if c.Size != size || c.span != 12 || mpzPublicationSnapshot(m, b) != before {
		t.Fatal("span borrowed without source size publication", c)
	}
	if _, err := c.Backing.ReadWord(m.Allocator, 0); !errors.Is(err, umemory.ErrRuntimeUninitialized) {
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
				leaf, companion = &ii.First.Num, &ii.First.Den
			case 1:
				ir = s.infEmpty()
				leaf, companion = &ir.First.Num, &ir.Second.Num
			case 2:
				ie = s.infEpsInfEmpty()
				leaf, companion = &ie.infinity.Num, &ie.numeral.Second.Num
			}
			if e = s.Manager.Z.SetInt64(companion, 1<<40); e != nil {
				t.Fatal(e)
			}
			mpzPublicationFailLeaf(t, s.Manager.Z, b, leaf)
			companionRef := companion.Cell
			before := mpzPublicationSnapshot(s.Manager.Z, b)
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
			integerPhysicalAudit(t, s.Manager.Z)
			mpzPublicationExpectFreed(t, s.Manager.Z, b, companionRef, before)
			after := mpzPublicationSnapshot(s.Manager.Z, b)
			if after.cells+1 != before.cells {
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
