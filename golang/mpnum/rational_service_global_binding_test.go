// Source: src/util/rational.cpp:64-83; src/util/rational.h:67;
// src/util/inf_rational.cpp:195-198; src/util/inf_int_rational.cpp:56-59.
// Source authors: Leonardo de Moura (2006-09-18); Nikolaj Bjorner (2006-12-05).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
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
	if current == nil || current != s.manager {
		t.Fatal("manager not published at actual constructor success")
	}
	actual := s.manager
	serviceMust(t, s.SourceFinalize())
	mask, e := s.SourceEndedStaticMask()
	serviceMust(t, e)
	if current != nil || s.manager != actual || mask != 0x1ff {
		t.Fatal("source global pointer versus retained concrete owner", mask)
	}
}

// Failed constructors retain only returned concrete child owners. The source
// member unwind may already have consumed their typed descriptor members.
func TestRationalGlobalAcquiredConstructorNamedPhysicalPrefixes(t *testing.T) {
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		b := arena.NewBudget(1 << 24)
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
		serviceMust(t, e)
		cfg := serviceConfig(b)
		cfg.Integers.Memory = r
		cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}

		b.FailAfter(ordinal)
		m, e := NewRationalManager(RationalConfig{Integers: cfg.Integers, Synchronized: true})
		var failed *IntegerManager
		if errors.As(e, &failed) {
			if failed.memory != r || failed.originalCause == nil {
				t.Fatal("foreign returned failed owner")
			}
			// Source destruction has consumed some typed members; name each surviving
			// constructor scalar before comparing actual direct provider objects.
			refs := []numericCellRef{failed.tmp[0], failed.tmp[1], failed.arg[0], failed.arg[1], failed.intMin.cell, failed.two64.cell}
			objects := map[memory.ObjectRef]bool{}
			blocks := map[memory.Block]bool{}
			for _, ref := range refs {
				if ref == 0 || failed.isFailureRef(ref) {
					continue
				}
				if !validNumericRef(failed.cells, ref) {
					t.Fatal("stale retained constructor scalar")
				}
				c := failed.cell(ref)
				if c.released {
					continue
				}
				if c.backing.object == (memory.ObjectRef{}) || objects[c.backing.object] {
					t.Fatal("missing/duplicate constructor payload")
				}
				objects[c.backing.object] = true
				blocks[c.backing.raw] = true
				_, readErr := r.Size(c.backing.raw)
				serviceMust(t, readErr)
			}
			if failed.pendingCell.object != (memory.ObjectRef{}) {
				if objects[failed.pendingCell.object] {
					t.Fatal("duplicate pending cell")
				}
				objects[failed.pendingCell.object] = true
				blocks[failed.pendingCell.raw] = true
				_, readErr := r.Size(failed.pendingCell.raw)
				serviceMust(t, readErr)
			}
			as := failed.allocator.Stats()
			if as.Chunks != 0 || r.Stats().LiveBlocks != uint32(len(blocks))+as.PendingBlocks {
				t.Fatal("unnamed failed constructor physical root", as, r.Stats(), len(objects))
			}
			b.FailAfter(-1)
			serviceMust(t, failed.CloseChecked())
		} else if e == nil {
			integerPhysicalAudit(t, m.z)
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
