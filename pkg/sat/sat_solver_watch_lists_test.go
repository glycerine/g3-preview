package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satVariableBinaryWatch(t *testing.T, l Literal, learned bool) satWatched {
	t.Helper()
	w, e := newSATBinaryWatch(l, learned)
	satOK(t, e)
	return w
}

func satVariableWatchFixture(t *testing.T) (*satSolverWatchLists, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	w, e := newSATSolverWatchLists(b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}})
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, w.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return w, b
}
func TestSATSolverWatchListsRealRelocationAndPayloadLifetime(t *testing.T) {
	w, b := satVariableWatchFixture(t)
	satOK(t, w.PushEmpty())
	satOK(t, w.PushEmpty())
	v, e := w.Get(0)
	satOK(t, e)
	satOK(t, v.Push(satVariableBinaryWatch(t, 2, false)))
	ref, e := v.ref()
	satOK(t, e)
	epoch := w.backingID
	before := w.store.Stats()
	satOK(t, w.PushEmpty())
	if w.backingID == epoch {
		t.Fatal("expected outer source relocation")
	}
	if _, e = v.Size(); e != ErrSATOwner {
		t.Fatal("old inline object address", e)
	}
	if !w.store.IsLive(ref) || w.store.Stats().Payloads.Live != before.Payloads.Live {
		t.Fatal("move must not clone/free payload", w.store.Stats(), before)
	}
	moved, e := w.Get(0)
	satOK(t, e)
	got, e := moved.Read(0)
	satOK(t, e)
	if got != satVariableBinaryWatch(t, 2, false) {
		t.Fatal(got)
	}
	for i := 0; i < 8; i++ {
		satOK(t, moved.Push(satVariableBinaryWatch(t, Literal(4+i*2), false)))
	}
	n, e := moved.Size()
	satOK(t, e)
	if n != 9 {
		t.Fatal(n)
	}
	used := b.Used()
	if w.Stats().TotalBytes != used {
		t.Fatal(w.Stats(), used)
	}
	satOK(t, w.Finalize())
	if w.store.IsLive(ref) {
		t.Fatal("source ascending member destruction")
	}
	if _, e = moved.Size(); e == nil {
		t.Fatal(e)
	}
	// The real descriptor free list receives ascending destruction; the last
	// source child is reused first. This is Go physical metadata, not C++ malloc.
	satOK(t, w.PushEmpty())
	last, e := w.Get(0)
	satOK(t, e)
	lastRef, e := last.ref()
	satOK(t, e)
	if lastRef.slot != 2 {
		t.Fatal("ascending child destruction", lastRef)
	}
}
func TestSATSolverWatchListsPlacementRefusalAndConstructorUnwind(t *testing.T) {
	w, b := satVariableWatchFixture(t)
	base := b.Used()
	b.FailAfter(1)
	e := w.PushEmpty()
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if len(w.lists) != 0 || cap(w.lists) != 2 || b.Used() != base+32 {
		t.Fatal("outer growth precedes child placement", w.Stats(), b.Used(), base)
	}
	b.FailAfter(-1)
	for fail := int64(0); fail < 20; fail++ {
		bb := arena.NewBudget(^uint64(0))
		bb.FailAfter(fail)
		x, e := newSATSolverWatchLists(bb, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: bb}})
		bb.FailAfter(-1)
		if e == nil {
			satOK(t, x.Close())
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		if bb.Used() != 0 {
			t.Fatal(fail, bb.Used())
		}
		if e == nil {
			break
		}
	}
}
