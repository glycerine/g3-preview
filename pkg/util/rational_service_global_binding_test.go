// Source: src/util/rational.cpp:64-83; src/util/rational.h:67;
// src/util/inf_rational.cpp:195-198; src/util/inf_int_rational.cpp:56-59.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceSourceGlobalSlotExactPublicationAndStaticEnd(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := PrepareRationalService(serviceConfig(b))
	serviceMust(t, e)
	defer func() {
		serviceMust(t, s.CloseChecked())
		if b.Used() != 0 {
			t.Error("actual static context")
		}
	}()
	var current *RationalManager
	serviceMust(t, s.BindSourceGlobalSlot(&current))
	var foreign *RationalManager
	if e := s.ValidateSourceGlobalSlot(&foreign); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal("foreign scalar slot accepted", e)
	}
	serviceMust(t, s.InitializeSource())
	if current == nil || current != s.Manager {
		t.Fatal("manager not published at actual constructor success")
	}
	actual := s.Manager
	serviceMust(t, s.SourceFinalize())
	mask, e := s.SourceEndedStaticMask()
	serviceMust(t, e)
	if current != nil || s.Manager != actual || mask != 0x1ff {
		t.Fatal("source global pointer versus retained concrete owner", mask)
	}
}

// Failed constructors retain only returned concrete child owners. The source
// member unwind may already have consumed their typed descriptor members.
func TestRationalGlobalAcquiredConstructorNamedPhysicalPrefixes(t *testing.T) {
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		b := arena.NewBudget(1 << 24)
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
		serviceMust(t, e)
		cfg := serviceConfig(b)
		cfg.Integers.Memory = r
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}

		b.FailAfter(ordinal)
		m, e := NewRationalManager(RationalConfig{Integers: cfg.Integers, Synchronized: true})
		var failed *IntegerManager
		if errors.As(e, &failed) {
			if failed.Memory != r || failed.originalCause == nil {
				t.Fatal("foreign returned failed owner")
			}
			// Source destruction has consumed some typed members; name each surviving
			// constructor scalar before comparing actual direct provider objects.
			refs := []NumericCellRef{failed.Tmp[0], failed.Tmp[1], failed.Arg[0], failed.Arg[1], failed.IntMin.Cell, failed.Two64.Cell}
			objects := map[umemory.ObjectRef]bool{}
			blocks := map[umemory.Block]bool{}
			for _, ref := range refs {
				if ref == 0 || failed.IsFailureRef(ref) {
					continue
				}
				if !ValidNumericRef(failed.Cells, ref) {
					t.Fatal("stale retained constructor scalar")
				}
				c := failed.Cell(ref)
				if c.Released {
					continue
				}
				if c.Backing.Object == (umemory.ObjectRef{}) || objects[c.Backing.Object] {
					t.Fatal("missing/duplicate constructor payload")
				}
				objects[c.Backing.Object] = true
				blocks[c.Backing.Raw] = true
				_, readErr := r.Size(c.Backing.Raw)
				serviceMust(t, readErr)
			}
			if failed.PendingCell.Object != (umemory.ObjectRef{}) {
				if objects[failed.PendingCell.Object] {
					t.Fatal("duplicate pending cell")
				}
				objects[failed.PendingCell.Object] = true
				blocks[failed.PendingCell.Raw] = true
				_, readErr := r.Size(failed.PendingCell.Raw)
				serviceMust(t, readErr)
			}
			as := failed.Allocator.Stats()
			if as.Chunks != 0 || r.Stats().LiveBlocks != uint32(len(blocks))+as.PendingBlocks {
				t.Fatal("unnamed failed constructor physical root", as, r.Stats(), len(objects))
			}
			b.FailAfter(-1)
			serviceMust(t, failed.CloseChecked())
		} else if e == nil {
			integerPhysicalAudit(t, m.Z)
			b.FailAfter(-1)
			serviceMust(t, m.CloseChecked())
		} else if !numericPhysicalReservationRefusal(e) {
			t.Fatal(e)
		}
		if b.Used() != r.Stats().TotalBytes || r.Stats().LiveBlocks != 0 {
			t.Fatal("failed acquisition cleanup lost actual root", ordinal, b.Used(), r.Stats().TotalBytes, r.Stats())
		}
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("retained cold provider")
		}
		if e == nil {
			t.Log("first complete constructor", ordinal)
			return
		}
	}
	t.Fatal("constructor prefix horizon never reaches completion")
}
