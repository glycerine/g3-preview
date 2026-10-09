// Source: src/util/trace.cpp:25-62; src/util/trace.h:38-49.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func traceRegion(t *testing.T, b *arena.Budget) *arena.BytesRegion {
	t.Helper()
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
	smallObjectMust(t, e)
	return r
}
func traceKey(t *testing.T, r *arena.BytesRegion, s string) StrKey {
	t.Helper()
	id := smallObjectID(t, r, s)
	k, e := BorrowCString(r, id.Ref, 0)
	smallObjectMust(t, e)
	return k
}

func TestTraceContextUsesOwnedGoStrings(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	smallObjectMust(t, c.EnableTraceName("parser"))
	ok, e := c.IsTraceNameEnabled("parser\x00ignored")
	smallObjectMust(t, e)
	if !ok || c.Stats().Tags != 1 {
		t.Fatal("enabled trace name was not retained")
	}
	smallObjectMust(t, c.DisableTraceName("parser"))
	ok, e = c.IsTraceNameEnabled("parser")
	smallObjectMust(t, e)
	if ok {
		t.Fatal("disabled trace name remains enabled")
	}
	smallObjectMust(t, c.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("leaked charged storage", b.Used())
	}
}

func TestTraceContextCopiesCompatibilityKey(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	k := traceKey(t, r, "solver")
	smallObjectMust(t, c.EnableTrace(k))
	smallObjectMust(t, r.Reset())
	ok, e := c.IsTraceEnabledCString("solver")
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("trace set retained the arena borrow")
	}
	smallObjectMust(t, c.EnableAllTrace(true))
	smallObjectMust(t, c.FinalizeTrace())
	ok, e = c.IsTraceEnabledCString("anything")
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("finalize changed enable-all")
	}
	smallObjectMust(t, c.Close())
	smallObjectMust(t, r.Close())
}

func TestTraceContextOwnerAndCloseChecks(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	copy := *c
	if e = copy.EnableTraceName("x"); !errors.Is(e, ErrTraceContextOwner) {
		t.Fatal(e)
	}
	smallObjectMust(t, c.Close())
	if _, e = c.IsTraceNameEnabled("x"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	smallObjectMust(t, r.Close())
}
