// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// mpz.cpp87-140 constructor; mpz.h member order; implicit failed-constructor
// member unwind excludes the completed mpz_manager destructor body.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestNumeralConstructionFailureOwnerMethodSets(t *testing.T) {
	var _ ConstructionFailure = (*IntegerManager)(nil)
	var _ ConstructionFailure = (*RationalManager)(nil)
	var _ ConstructionFailure = (*RationalService)(nil)
}

func TestIntegerFailedConstructorActualMemberUnwind(t *testing.T) {
	for _, direct := range []bool{false, true} {
		observed := false
		for stop := int64(0); stop < 64; stop++ {
			b := arena.NewBudget(1 << 24)
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
			if e != nil {
				t.Fatal(e)
			}
			baseline := b.Used()
			b.FailAfter(stop)
			profile := memory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: !direct}
			m, e := NewIntegerManager(IntegerConfig{Memory: rt, Profile: profile, Cells: arena.Config{Budget: b, PageSize: 2}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b, PageSize: 2}}})
			b.FailAfter(-1)
			if e == nil {
				if e = m.CloseChecked(); e != nil {
					t.Fatal(e)
				}
			} else {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("original source allocation refusal", e)
				}
				var failed ConstructionFailure
				if errors.As(e, &failed) {
					partial, ok := failed.(*IntegerManager)
					if !ok || partial.RuntimeOwner() != rt || partial.OriginalCause() == nil || partial.Unwrap() != partial.OriginalCause() || partial.sourceFinalizing || partial.sourceFinalized {
						t.Fatal("failed constructor falsely invoked completed destructor", e)
					}
					original := partial.OriginalCause()
					hasPrivateRoot := partial.tmp != [2]numericCellRef{} || partial.arg != [2]numericCellRef{} || partial.pendingCell.object != (memory.ObjectRef{})
					candidate := hasPrivateRoot && ((direct && partial.CleanupCause() == nil && rt.Stats().LiveBlocks > 0) || (!direct && errors.Is(partial.CleanupCause(), memory.ErrAllocatorIO)))
					if candidate {
						if partial.natural != nil || partial.headerBytes == 0 || b.Used() <= baseline {
							t.Fatal("member-unwind root/charged owner prefix")
						}
						if direct {
							if rt.Stats().LiveBlocks == 0 || partial.CleanupCause() != nil {
								t.Fatal("successful member unwind lost direct roots or invented cleanup refusal")
							}
						} else {
							if rt.Stats().LiveBlocks != 0 || !errors.Is(partial.CleanupCause(), memory.ErrAllocatorIO) {
								t.Fatal("pool-free then source diagnostic prefix", partial.CleanupCause(), rt.Stats())
							}
						}
						// Actual private scalar roots have not run the destructor del body.
						if partial.tmp == [2]numericCellRef{} && partial.arg == [2]numericCellRef{} && partial.pendingCell.object == (memory.ObjectRef{}) {
							t.Fatal("no real partial constructor allocation observed")
						}
					}
					if e = partial.Cleanup(); e != nil {
						t.Fatal("explicit owning cleanup", e)
					}
					if partial.OriginalCause() != original || partial.CleanupCause() != nil || partial.RuntimeOwner() != rt || partial.sourceFinalized {
						t.Fatal("cleanup changed source/cause/domain history")
					}
					observed = candidate
				}
			}
			if rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes {
				t.Fatal("constructor cleanup lost actual roots", stop, rt.Stats(), b.Used(), baseline)
			}
			if e = rt.Close(); e != nil || b.Used() != 0 {
				t.Fatal(e, b.Used())
			}
			if observed {
				break
			}
		}
		if !observed {
			t.Fatal("no genuine partial member unwind within bounded sweep", direct)
		}
	}
}

// Constructor failures in these factories originate in the actual Integer
// child. This independently sums its retained physical/cold graph: a refunded
// unpublished Rational/Service header cannot hide in the total.
func constructionFailureOwningCleanup(t *testing.T, e error, b *arena.Budget) {
	t.Helper()
	var failure ConstructionFailure
	if !errors.As(e, &failure) {
		return
	}
	m, ok := failure.(*IntegerManager)
	if !ok || m.originalCause == nil || failure.Unwrap() != m.originalCause || !numericPhysicalReservationRefusal(m.originalCause) || !m.constructionFailed || m.sourceFinalizing || m.sourceFinalized {
		t.Fatal("invalid forwarded failed constructor graph", e)
	}
	owned := m.headerBytes
	if m.memory != nil {
		owned += m.memory.Stats().TotalBytes
	}
	if m.allocator != nil {
		owned += m.allocator.Stats().TotalBytes
	}
	if m.cells != nil {
		s := m.cells.Stats()
		owned += s.OwnerBytes + s.RecordBytes + s.MetadataBytes + s.DirectoryBytes
		for i := uint64(1); i <= uint64(s.Slots); i++ {
			c, err := m.cells.Read(arena.Slot(i))
			if err != nil {
				continue
			}
			v, err := m.memory.ReadUint32(c.backing.raw, c.backing.offset+4)
			if c.released {
				if !errors.Is(err, memory.ErrRuntimeOwner) {
					t.Fatal("released source pooled root still physically live", i, err)
				}
			} else if err != nil || v != c.capacity {
				t.Fatal("retained constructor raw cell", i, v, err)
			}
		}
	}
	if m.words != nil {
		s := m.words.Stats()
		owned += s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + s.Descriptors.OwnerBytes + s.Descriptors.RecordBytes + s.Descriptors.MetadataBytes + s.Descriptors.DirectoryBytes
	}
	if m.natural != nil {
		t.Fatal("member unwind did not destroy Natural before allocator refusal")
	}
	if owned != b.Used() || owned == 0 {
		t.Fatal("constructor retained/unpublished outer header accounting", owned, b.Used())
	}
	original := m.originalCause
	b.FailAfter(-1)
	if err := failure.Cleanup(); err != nil {
		t.Fatal("explicit retained constructor cleanup", err)
	}
	if failure.OriginalCause() != original || failure.CleanupCause() != nil || b.Used() != 0 {
		t.Fatal("constructor cause/owning cleanup completion", b.Used())
	}
}
