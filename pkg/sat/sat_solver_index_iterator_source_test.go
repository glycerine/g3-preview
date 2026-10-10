package sat

import (
	"errors"
	"testing"
)

func TestSATConsequenceNativeIndexIteratorBorrowLifetime(t *testing.T) {
	var set satSolverIndexSet
	for _, key := range []uint32{17, 1, 9} {
		satOK(t, set.insert(key))
	}
	it, e := set.capture()
	satOK(t, e)
	copied := it
	satOK(t, set.insert(17))
	for _, iter := range []*satSolverIndexIterator{&it, &copied} {
		v, e := iter.next()
		satOK(t, e)
		if !v.present || v.value != 17 {
			t.Fatal("copy/duplicate changed iterator", v)
		}
	}
	satOK(t, set.insert(33))
	for _, iter := range []*satSolverIndexIterator{&it, &copied} {
		if _, e = iter.next(); !errors.Is(e, ErrSATOwner) {
			t.Fatal("key change retained borrowed view", e)
		}
	}
	it, e = set.capture()
	satOK(t, e)
	satOK(t, set.reset())
	if _, e = it.next(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("reset retained borrowed view", e)
	}
	it, e = set.capture()
	satOK(t, e)
	v, e := it.next()
	satOK(t, e)
	if v.present {
		t.Fatal("empty iterator", v)
	}
	satOK(t, set.close())
	if _, e = it.next(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("close retained empty borrowed view", e)
	}
	var zero satSolverIndexIterator
	if _, e = zero.next(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("zero iterator accepted", e)
	}
}
