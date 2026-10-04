// Source: src/util/mpz.h:132-164; src/util/mpz.cpp:306-349,1186-1212. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestIntegerSourceFailureTwoChargedSentinels(t *testing.T) {
	m, _ := mpzPublicationDirectOwner(t)
	if m.failedNullRef == 0 || m.failedNullRef == m.failedRef || validNumericRef(m.cells, m.failedNullRef) || validNumericRef(m.cells, m.failedRef) {
		t.Fatal("actual disjoint dead ledger descriptors")
	}
	st := m.cells.Stats()
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
				old := out.cell
				live := m.cells.Stats().Live
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
				want := m.failedNullRef
				if oldBig {
					want = m.failedRef
					live--
				}
				if out.cell != want || out.val != 1 || m.cells.Stats().Live != live || (old != 0 && validNumericRef(m.cells, old)) {
					t.Fatal("source pointer state/sign/free prefix", out, want, m.cells.Stats())
				}
				if e = m.check(&out); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("guarded public value remains invalid", e)
				}
				before := m.memory.Stats()
				guard, e := sourceNumeralDel(m, &out)
				if oldBig {
					if !guard || !errors.Is(e, arena.ErrInvalidSlot) || out.cell != want {
						t.Fatal("source dangling access guard", guard, e, out)
					}
				} else {
					if guard || e != nil || out.cell != 0 || out.val != 1 {
						t.Fatal("defined source NULL leaf", guard, e, out)
					}
				}
				if m.memory.Stats() != before {
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
				if m.cells.Stats().Retired != 2 || m.cells.Stats().Live != 6 {
					t.Fatal("constructor source versus metadata counts", m.cells.Stats())
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
				if z.failedRef != 0 && z.failedNullRef == 0 {
					sawSecond = true
					if z.isFailureRef(0) {
						t.Error("unprepared zero is not a dead descriptor")
					}
					count, cause := z.memory.AllocationCount()
					serviceMust(t, cause)
					if z.cells == nil || z.cells.closed || z.cells.Stats().Retired != 1 || z.cells.Stats().Slots != 1 || count != 0 || b.Used() == 0 {
						t.Fatal("second descriptor refusal retained actual cold owner", z.cells, z.memory.Stats(), b.Used())
					}
				}
				if z.failedNullRef != 0 && z.failedRef == 0 {
					t.Fatal("paired source state preparation order")
				}
				if z.cells != nil && !z.cells.closed && z.cells.Stats().Retired > 2 {
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
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 2, SlotLimit: 7}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b, PageSize: 2}}})
	if m != nil || !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("actual reached ordinary descriptor refusal", m, e)
	}
	var owner ConstructionFailure
	if !errors.As(e, &owner) {
		t.Fatal("physical allocation retained in charged owner", e)
	}
	z, ok := owner.(*IntegerManager)
	if !ok || z.originalCause != arena.ErrSlotLimit || !z.constructorUnwound || z.sourceFinalized || z.sourceFinalizing {
		t.Fatal("actual member unwind only", e)
	}
	cs := z.cells.Stats()
	count, cause := r.AllocationCount()
	serviceMust(t, cause)
	if cs.Slots != 7 || cs.Retired != 2 || cs.Live != 5 || count != 6 || r.Stats().LiveBlocks != 6 || r.Stats().PhysicalBytes != 240 || z.two64.cell != 0 || z.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("two metadata plus five published and sixth physical cell", cs, count, r.Stats(), z.pendingCell)
	}
	actual := map[memory.Block]bool{}
	for _, ref := range []numericCellRef{z.tmp[0], z.arg[0], z.tmp[1], z.arg[1], z.intMin.cell} {
		if !validNumericRef(z.cells, ref) {
			t.Fatal("actual constructor private field", ref)
		}
		c := z.cell(ref)
		if c.backing.raw == (memory.Block{}) || actual[c.backing.raw] {
			t.Fatal("distinct real raw field", c)
		}
		actual[c.backing.raw] = true
		cap, cause := r.ReadUint32(c.backing.raw, 4)
		serviceMust(t, cause)
		if cap != 6 {
			t.Fatal("capacity publication", cap)
		}
	}
	if actual[z.pendingCell.raw] {
		t.Fatal("pending physical root duplicated")
	}
	actual[z.pendingCell.raw] = true
	cap, cause := r.ReadUint32(z.pendingCell.raw, 4)
	serviceMust(t, cause)
	if cap != 6 || len(actual) != 6 {
		t.Fatal("pending capacity/root", cap, len(actual))
	}
	// The borrowed runtime and the real cold adapters are counted once each.
	ws := z.words.Stats()
	owned := z.headerBytes + r.Stats().TotalBytes + z.allocator.Stats().TotalBytes + cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes + ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if z.natural != nil || owned != b.Used() {
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
