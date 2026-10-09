// Source: src/util/prime_generator.cpp:99-107.
// This Go binding accessor exposes the exact pointer selected by the literal
// null-generator iterator constructor; it is not a named C++ getter port.
package prime

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestDefaultPrimeBorrowExactBindingAndFinalizedIdentity(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewDefaultService(b)
	primeMust(t, e)
	old, used := s.generator, b.Used()
	g, e := s.BorrowGenerator()
	primeMust(t, e)
	if g != old || b.Used() != used {
		t.Fatal("borrow must not create or own a second generator")
	}
	primeMust(t, s.Finalize())
	after := b.Used()
	again, e := s.BorrowGenerator()
	primeMust(t, e)
	if again != old || b.Used() != after {
		t.Fatal("Finalize must preserve actual generator identity")
	}
	it, e := NewServiceIterator(s, again)
	primeMust(t, e)
	_, e = it.Next()
	if !errors.Is(e, arena.ErrBounds) || it.index != 1 || b.Used() != after {
		t.Fatal("reached empty-vector source prefix changed", e, it.index)
	}
	copied := *s
	if _, e = copied.BorrowGenerator(); e != ErrServiceOwner {
		t.Fatal("copied binding accepted", e)
	}
	primeMust(t, s.Close())
	if _, e = s.BorrowGenerator(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed binding", e)
	}
	if b.Used() != 0 {
		t.Fatal("borrow took ownership", b.Used())
	}
}
