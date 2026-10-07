// Source: src/util/mpz.h:132-164; src/util/mpz.cpp:306-349,1186-1212.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerSourceFailureTwoChargedSentinels(t *testing.T) {
	m, _ := mpzPublicationDirectOwner(t)
	if m.FailedNullRef == 0 || m.FailedNullRef == m.FailedRef || ValidNumericRef(m.Cells, m.FailedNullRef) || ValidNumericRef(m.Cells, m.FailedRef) {
		t.Fatal("actual disjoint dead ledger descriptors")
	}
	st := m.Cells.Stats()
	if st.Retired != 2 || st.Slots != 8 || st.Live != 6 {
		t.Fatal("two Go descriptors versus six source cells", st)
	}
	if e := arena.ValidateRecordType(reflect.TypeOf(Int{})); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerSourceFreshNullAndDestructiveDangling(t *testing.T) {
	for _, operation := range []string{"copy", "limbs", "capacity"} {
		for _, oldBig := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/fresh", true: "/old"}[oldBig], func(t *testing.T) {
				m, b := mpzPublicationDirectOwner(t)
				src := NewInt(0)
				out := NewInt(-19)
				digits := make([]uint32, 33)
				digits[0] = 7
				digits[32] = 1
				serviceMust(t, m.SetLimbs(&src, digits))
				defer func() { b.FailAfter(-1); serviceMust(t, m.Del(&src)); serviceMust(t, m.Del(&out)) }()
				if oldBig {
					serviceMust(t, m.SetUint64(&out, 1<<40))
				}
				old := out.Cell
				live := m.Cells.Stats().Live
				b.FailAfter(0)
				var e error
				switch operation {
				case "copy":
					e = m.Copy(&out, &src)
				case "limbs":
					e = m.SetLimbs(&out, digits)
				case "capacity":
					e = m.allocateIfNeeded(&out, 33)
				}
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual source physical refusal", e)
				}
				want := m.FailedNullRef
				if oldBig {
					want = m.FailedRef
					live--
				}
				if out.Cell != want || out.Val != 1 || m.Cells.Stats().Live != live || (old != 0 && ValidNumericRef(m.Cells, old)) {
					t.Fatal("source pointer state/sign/free prefix", out, want, m.Cells.Stats())
				}
				if e = m.check(&out); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("guarded public value remains invalid", e)
				}
				before := m.Memory.Stats()
				guard, e := sourceNumeralDel(m, &out)
				if oldBig {
					if !guard || !errors.Is(e, arena.ErrInvalidSlot) || out.Cell != want {
						t.Fatal("source dangling access guard", guard, e, out)
					}
				} else {
					if guard || e != nil || out.Cell != 0 || out.Val != 1 {
						t.Fatal("defined source NULL leaf", guard, e, out)
					}
				}
				if m.Memory.Stats() != before {
					t.Fatal("source NULL/guard invented memory action")
				}
			})
		}
	}
}

func TestIntegerSourceFailureSentinelPreparationPrefixes(t *testing.T) {
	for _, page := range []uint32{1, 2} {
		success := false
		failures := 0
		sawSecond := false
		for ordinal := int64(0); ordinal < 80; ordinal++ {
			b := arena.NewBudget(1 << 20)
			b.FailAfter(ordinal)
			m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: page, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
			if e == nil {
				if m.Cells.Stats().Retired != 2 || m.Cells.Stats().Live != 6 {
					t.Fatal("constructor source versus metadata counts", m.Cells.Stats())
				}
				b.FailAfter(-1)
				serviceMust(t, m.CloseChecked())
				if b.Used() != 0 {
					t.Fatal("success cleanup", b.Used())
				}
				success = true
				break
			}
			failures++
			var owner ConstructionFailure
			if errors.As(e, &owner) {
				z, ok := owner.(*IntegerManager)
				if !ok || z.originalCause == nil {
					t.Fatal("retained original owner/cause", e)
				}
				if z.FailedRef != 0 && z.FailedNullRef == 0 {
					sawSecond = true
					if z.IsFailureRef(0) {
						t.Error("unprepared zero is not a dead descriptor")
					}
					count, cause := z.Memory.AllocationCount()
					serviceMust(t, cause)
					if z.Cells == nil || z.Cells.closed || z.Cells.Stats().Retired != 1 || z.Cells.Stats().Slots != 1 || count != 0 || b.Used() == 0 {
						t.Fatal("second descriptor refusal retained actual cold owner", z.Cells, z.Memory.Stats(), b.Used())
					}
				}
				if z.FailedNullRef != 0 && z.FailedRef == 0 {
					t.Fatal("paired source state preparation order")
				}
				if z.Cells != nil && !z.Cells.closed && z.Cells.Stats().Retired > 2 {
					t.Fatal("synthetic failure descriptors")
				}
				b.FailAfter(-1)
				serviceMust(t, owner.Cleanup())
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("exact real preparation refusal", e)
			}
			if b.Used() != 0 {
				t.Fatal("all retained constructor roots cleaned", page, ordinal, b.Used())
			}
		}
		if !success || failures == 0 || (page == 1 && !sawSecond) {
			t.Fatal("bounded complete preparation campaign", page, failures)
		}
	}
}

// SlotLimit is not preflighted as an invented minimum. The sixth actual
// source allocation completes before its ordinary descriptor is refused.
func TestIntegerSourceSentinelSlotLimitReachedAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 2, SlotLimit: 7}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b, PageSize: 2}}})
	if m != nil || !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("actual reached ordinary descriptor refusal", m, e)
	}
	var owner ConstructionFailure
	if !errors.As(e, &owner) {
		t.Fatal("physical allocation retained in charged owner", e)
	}
	z, ok := owner.(*IntegerManager)
	if !ok || z.originalCause != arena.ErrSlotLimit || !z.ConstructorUnwound || z.SourceFinalized || z.SourceFinalizing {
		t.Fatal("actual member unwind only", e)
	}
	cs := z.Cells.Stats()
	count, cause := r.AllocationCount()
	serviceMust(t, cause)
	if cs.Slots != 7 || cs.Retired != 2 || cs.Live != 5 || count != 6 || r.Stats().LiveBlocks != 6 || r.Stats().PhysicalBytes != 240 || z.Two64.Cell != 0 || z.PendingCell.Object == (ObjectRef{}) {
		t.Fatal("two metadata plus five published and sixth physical cell", cs, count, r.Stats(), z.PendingCell)
	}
	actual := map[Block]bool{}
	for _, ref := range []NumericCellRef{z.Tmp[0], z.Arg[0], z.Tmp[1], z.Arg[1], z.IntMin.Cell} {
		if !ValidNumericRef(z.Cells, ref) {
			t.Fatal("actual constructor private field", ref)
		}
		c := z.Cell(ref)
		if c.Backing.Raw == (Block{}) || actual[c.Backing.Raw] {
			t.Fatal("distinct real raw field", c)
		}
		actual[c.Backing.Raw] = true
		cap, cause := r.ReadUint32(c.Backing.Raw, 4)
		serviceMust(t, cause)
		if cap != 6 {
			t.Fatal("capacity publication", cap)
		}
	}
	if actual[z.PendingCell.Raw] {
		t.Fatal("pending physical root duplicated")
	}
	actual[z.PendingCell.Raw] = true
	cap, cause := r.ReadUint32(z.PendingCell.Raw, 4)
	serviceMust(t, cause)
	if cap != 6 || len(actual) != 6 {
		t.Fatal("pending capacity/root", cap, len(actual))
	}
	// The borrowed runtime and the real cold adapters are counted once each.
	ws := z.Words.Stats()
	owned := z.HeaderBytes + r.Stats().TotalBytes + z.Allocator.Stats().TotalBytes + cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes + ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if z.Natural != nil || owned != b.Used() {
		t.Fatal("actual immediate retained graph", owned, b.Used())
	}
	serviceMust(t, owner.Cleanup())
	if r.Stats().LiveBlocks != 0 || b.Used() != r.Stats().TotalBytes || owner.OriginalCause() != arena.ErrSlotLimit {
		t.Fatal("owning cleanup/cause", r.Stats(), b.Used())
	}
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("borrowed runtime cleanup", b.Used())
	}
}
