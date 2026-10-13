package sexpr

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSexprRefAssignmentKeepsOldOnStoppedNumeralDeletion(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, _, q := sxOwner(t, b)
	v := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579/47890485652059026823698344598447161988085597568237573")
	old, e := m.MkNumeral(&v, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&v))
	incoming, e := m.MkString("incoming", 0, 0)
	sxMust(t, e)
	r, e := m.NewRef(old)
	sxMust(t, e)
	sxMust(t, m.IncRef(incoming))
	q.Manager.Z.SourceFinalizing = true
	e = m.AssignRef(&r, incoming)
	q.Manager.Z.SourceFinalizing = false
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("old numeral deletion refusal: %v", e)
	}
	got, e := m.GetRef(&r)
	sxMust(t, e)
	if got != old || r.owned || len(r.pending) != 1 || r.pending[0] != incoming {
		t.Fatalf("source pointer and owning incoming after stopped DEC: %+v, got %v", r, got)
	}
	if refs, e := m.RefCount(incoming); e != nil || refs != 2 {
		t.Fatalf("incoming reference count = %d, %v", refs, e)
	}
	moved, e := m.MoveRef(&r)
	sxMust(t, e)
	if !r.IsNull() || moved.h != old || len(moved.pending) != 1 {
		t.Fatal("move lost stopped source pointer or incoming ownership", r, moved)
	}
	other, e := m.NewRef(incoming)
	sxMust(t, e)
	sxMust(t, m.SwapRef(&moved, &other))
	got, e = m.GetRef(&other)
	sxMust(t, e)
	if got != old || other.owned || len(other.pending) != 1 {
		t.Fatal("swap lost stopped source pointer or incoming ownership", other, got)
	}
	sxMust(t, m.CloseRef(&other))
	sxMust(t, m.CloseRef(&moved))
	sxMust(t, m.CloseRef(&r))
	if refs, e := m.RefCount(incoming); e != nil || refs != 1 {
		t.Fatalf("incoming owning cleanup = %d, %v", refs, e)
	}
	sxMust(t, m.DecRef(incoming))
}

func TestSexprRefResetKeepsOldOnStoppedNumeralDeletion(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	v := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579")
	old, e := m.MkNumeral(&v, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&v))
	r, e := m.NewRef(old)
	sxMust(t, e)
	q.Manager.Z.SourceFinalizing = true
	e = m.ResetRef(&r)
	q.Manager.Z.SourceFinalizing = false
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("old numeral reset refusal: %v", e)
	}
	got, e := m.GetRef(&r)
	sxMust(t, e)
	if got != old || r.owned || len(r.pending) != 0 {
		t.Fatal("source reset pointer after stopped DEC", r, got)
	}
	sxMust(t, m.CloseRef(&r))
	if !r.IsNull() {
		t.Fatal("owning reset cleanup retained raw pointer")
	}
}
