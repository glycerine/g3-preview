// Source: src/util/prime_generator.cpp:99-107.
// This Go binding accessor exposes the exact pointer selected by the literal
// null-generator iterator constructor; it is not a named C++ getter port.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestDefaultPrimeBorrowExactBindingAndFinalizedIdentity(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewDefaultPrimeService(b)
	smallObjectMust(t, e)
	old, used := s.generator, b.Used()
	g, e := s.BorrowGenerator()
	smallObjectMust(t, e)
	if g != old || b.Used() != used {
		t.Fatal("borrow must not create or own a second generator")
	}
	smallObjectMust(t, s.Finalize())
	after := b.Used()
	again, e := s.BorrowGenerator()
	smallObjectMust(t, e)
	if again != old || b.Used() != after {
		t.Fatal("Finalize must preserve actual generator identity")
	}
	it, e := NewPrimeServiceIterator(s, again)
	smallObjectMust(t, e)
	_, e = it.Next()
	if !errors.Is(e, arena.ErrBounds) || it.index != 1 || b.Used() != after {
		t.Fatal("reached empty-vector source prefix changed", e, it.index)
	}
	copied := *s
	if _, e = copied.BorrowGenerator(); e != ErrDefaultPrimeServiceOwner {
		t.Fatal("copied binding accepted", e)
	}
	smallObjectMust(t, s.Close())
	if _, e = s.BorrowGenerator(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed binding", e)
	}
	if b.Used() != 0 {
		t.Fatal("borrow took ownership", b.Used())
	}
}
