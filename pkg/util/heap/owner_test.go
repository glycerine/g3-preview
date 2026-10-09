// Source: src/util/heap.h:27-28; src/util/vector.h:102-129,191-202.
package heap

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHeapLegacyCopiedCloseRejectsBeforeRefund(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	h, e := New(4, Config{Budget: b})
	heapMust(t, e)
	copied := *h
	before := b.Used()
	if e = copied.Close(); !errors.Is(e, ErrOwner) {
		t.Fatal("copied cold owner Close admitted", e)
	}
	if b.Used() != before {
		t.Fatal("copy guard refunded actual original owner")
	}
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("original owner not refunded")
	}
}
