// Concrete storage adapter checks for source nested literal vectors.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satNestedTestStore(t *testing.T) (*satSolverLiteralStore, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverLiteralStore(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("nested store retained %d", b.Used())
		}
	})
	return s, b
}

func TestSATSolverNestedLiteralCapacityAndRetainedTail(t *testing.T) {
	s, _ := satNestedTestStore(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for i, wantCap := range []uint32{2, 2, 3, 5, 5} {
		paramsMust(t, s.Push(r, Literal(i+10)))
		got, e := s.Capacity(r)
		paramsMust(t, e)
		if got != wantCap {
			t.Fatalf("source capacity %d: %d != %d", i, got, wantCap)
		}
	}
	paramsMust(t, s.Shrink(r, 2))
	paramsMust(t, s.Reserve(r, 8))
	got, e := s.readPhysical(r, 4)
	paramsMust(t, e)
	if got != 14 {
		t.Fatalf("source trivial growth lost retained tail: %d", got)
	}
	paramsMust(t, s.Resize(r, 5))
	for i := uint32(2); i < 5; i++ {
		got, e := s.Read(r, i)
		paramsMust(t, e)
		if got != DefaultLiteral() {
			t.Fatalf("source default constructor %d: %d", i, got)
		}
	}
}

func TestSATSolverNestedLiteralDeepCopyAndLedger(t *testing.T) {
	s, _ := satNestedTestStore(t)
	a, e := s.NewList()
	paramsMust(t, e)
	for i := uint32(0); i < 4; i++ {
		paramsMust(t, s.Push(a, Literal(i+20)))
	}
	paramsMust(t, s.Shrink(a, 2))
	b, e := s.CloneList(a)
	paramsMust(t, e)
	ca, e := s.Capacity(a)
	paramsMust(t, e)
	cb, e := s.Capacity(b)
	paramsMust(t, e)
	if ca != 5 || cb != ca {
		t.Fatalf("source clone capacity %d/%d", ca, cb)
	}
	paramsMust(t, s.Write(a, 0, 91))
	v, e := s.Read(b, 0)
	paramsMust(t, e)
	if v != 20 {
		t.Fatal("clone aliases")
	}
	paramsMust(t, s.Free(a))
	c, e := s.NewList()
	paramsMust(t, e)
	if c.slot != a.slot || c.generation == a.generation {
		t.Fatal("recycle source identity ledger")
	}
	if _, e = s.Read(a, 0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("stale %v", e)
	}
	paramsMust(t, s.Reset())
	if s.IsLive(b) || s.IsLive(c) {
		t.Fatal("reset lifetime")
	}
	d, e := s.NewList()
	paramsMust(t, e)
	if d == a || d == b || d == c {
		t.Fatal("reset reused generation")
	}
}

func TestSATSolverNestedLiteralFaultPrefixAndRoots(t *testing.T) {
	s, b := satNestedTestStore(t)
	r, e := s.NewList()
	paramsMust(t, e)
	paramsMust(t, s.Push(r, 11))
	paramsMust(t, s.Push(r, 12))
	before := b.Used()
	b.FailAfter(0)
	e = s.Reserve(r, 9)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("growth fail %v", e)
	}
	n, _ := s.Size(r)
	c, _ := s.Capacity(r)
	if n != 2 || c != 2 || b.Used() != before {
		t.Fatalf("first refusal changed actual state n=%d cap=%d used=%d/%d", n, c, b.Used(), before)
	}
	b.FailAfter(-1)
	// Exact capacity cloning and repeated growth may retain arena free pages,
	// but all such physical storage remains charged to this owner until Close.
	for k := 0; k < 50; k++ {
		x, e := s.CloneList(r)
		paramsMust(t, e)
		paramsMust(t, s.Free(x))
	}
	used := b.Used()
	for k := 0; k < 50; k++ {
		x, e := s.CloneList(r)
		paramsMust(t, e)
		paramsMust(t, s.Free(x))
	}
	if b.Used() != used {
		t.Fatalf("warmed plateau %d/%d", b.Used(), used)
	}
}

func TestSATSolverNestedLiteralOwnersPointerFreedomWarmAllocation(t *testing.T) {
	s, b := satNestedTestStore(t)
	other, _ := satNestedTestStore(t)
	r, e := s.NewList()
	paramsMust(t, e)
	if _, e = other.Size(r); !errors.Is(e, ErrSATOwner) {
		t.Fatalf("foreign %v", e)
	}
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf(satSolverLiteralListRef{})))
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf(satSolverLiteralListRecord{})))
	for i := 0; i < 10; i++ {
		paramsMust(t, s.Push(r, 17))
	}
	paramsMust(t, s.Shrink(r, 0))
	before := b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.Push(r, 17); e != nil {
			panic(e)
		}
		if e := s.Shrink(r, 0); e != nil {
			panic(e)
		}
	})
	if allocs != 0 || b.Used() != before {
		t.Fatalf("warm allocations=%v budget=%d/%d", allocs, b.Used(), before)
	}
	b.FailAfter(-1)
}
