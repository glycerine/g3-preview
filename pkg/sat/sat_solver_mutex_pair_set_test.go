// Source: src/util/hashtable.h; actual literal pair specialization growth/copies.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATMutexPairSetGrowBeforeDuplicateAndEveryPhysicalRefusal(t *testing.T) {
	fixture := func(t *testing.T) (*Solver, *satMutexPairSet) {
		s, _, _ := satSolverCheckFixture(t)
		f, e := s.mutexFrame()
		satOK(t, e)
		f.seenUsed = true
		satOK(t, f.seen.init(s.runtime.memory))
		f.seenComplete = true
		t.Cleanup(func() { satOK(t, s.mutexDrop(f, nil)) })
		for i := uint32(0); i < 7; i++ {
			satOK(t, f.seen.insert(i, i^0x80000000))
		}
		return s, &f.seen
	}
	h := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		s, p := fixture(t)
		old := p.raw
		const start = int64(1 << 50)
		s.budget.FailAfter(start)
		satOK(t, p.insert(0, 0x80000000))
		h = start - testFieldBySourceName(reflect.ValueOf(s.budget).Elem(), "remaining").Int()
		if p.capacity != 16 || p.size != 7 {
			t.Fatal(p.capacity, p.size)
		}
		if _, e := s.runtime.memory.Read(old, 0); e == nil {
			t.Fatal("old array not released")
		}
		for i := uint32(0); i < 7; i++ {
			yes, e := p.contains(i, i^0x80000000)
			satOK(t, e)
			if !yes {
				t.Fatal(i)
			}
		}
		satFullSolverRuntimeAudit(t, s)
		s.budget.FailAfter(-1)
	})
	for fault := int64(0); fault <= h; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, p := fixture(t)
			old := p.raw
			s.budget.FailAfter(fault)
			e := p.insert(0, 0x80000000)
			satFullSolverRuntimeAudit(t, s)
			if fault < h {
				satProbingExpectedRefusal(t, s.propagation, e)
				if p.raw != old || p.size != 7 || p.capacity != 8 {
					t.Fatal("source old table changed on allocation refusal")
				}
			} else {
				satOK(t, e)
				if p.pending != (util.Block{}) {
					t.Fatal("successful pending prefix retained")
				}
			}
			s.budget.FailAfter(-1)
		})
	}
}

func TestSATMutexPairSetDefaultConstructsBothLiteralMembers(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.mutexFrame()
	satOK(t, e)
	f.seenUsed = true
	satOK(t, f.seen.init(s.runtime.memory))
	f.seenComplete = true
	t.Cleanup(func() { satOK(t, s.mutexDrop(f, nil)) })
	for i := uint32(0); i < 8; i++ {
		for _, off := range []uint64{8, 12} {
			v, e := s.runtime.memory.ReadUint32(f.seen.raw, uint64(i)*16+off)
			if e != nil || v != 0xfffffffe {
				t.Errorf("source default literal member i%d offset%d value%x err%v", i, off, v, e)
			}
		}
	}
	if _, e = s.runtime.memory.ReadUint32(f.seen.raw, 0); !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("cached hash is genuinely uninitialized on FREE entry", e)
	}
}

// Malformed source capacity: real small old payload with only its first entry
// initialized; wrapped target extent must be guarded before reading old entry1.
func TestSATMutexPairSetWrappedCopyGuardAtActualDestination(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.mutexFrame()
	satOK(t, e)
	f.seenUsed = true
	satOK(t, f.seen.init(s.runtime.memory))
	f.seenComplete = true
	t.Cleanup(func() { satOK(t, s.mutexDrop(f, nil)) })
	p := &f.seen
	satOK(t, s.runtime.memory.Deallocate(&p.raw))
	p.raw, e = s.runtime.memory.Allocate(128)
	satOK(t, e)
	for i, v := range []uint32{1, 2, 10, 20} {
		satOK(t, s.runtime.memory.WriteUint32(p.raw, uint64(i)*4, v))
	}
	p.capacity = 1 << 31
	p.size = 1
	old := p.raw
	e = p.grow()
	want := error(arena.ErrBounds)
	if satStorageSourceDebug {
		want = ErrSATAssertion
	}
	if !errors.Is(e, want) {
		t.Fatal("source target copy assertion/dereference precedes old uninitialized suffix", e, want)
	}
	if p.raw != old || p.capacity != 1<<31 || p.pending == (util.Block{}) {
		t.Fatal("source completed wrapped allocation prefix lost", p.raw, p.capacity, p.pending)
	}
	satFullSolverRuntimeAudit(t, s)
}
