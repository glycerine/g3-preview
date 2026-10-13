package sexpr

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSexprRefVectorSetKeepsSourcePointerAndIncomingOnStoppedDeletion(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	n := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579")
	old, e := m.MkNumeral(&n, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&n))
	incoming, e := m.MkString("incoming", 0, 0)
	sxMust(t, e)
	v, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, v.PushBack(old))
	sxMust(t, m.IncRef(incoming))
	q.Manager.Z.SourceFinalizing = true
	e = v.Set(0, incoming)
	q.Manager.Z.SourceFinalizing = false
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("stopped numeral deletion: %v", e)
	}
	got, e := v.Get(0)
	sxMust(t, e)
	if got != old || v.owned[0] || len(v.pending) != 1 || v.pending[0] != incoming {
		t.Fatalf("source publication and incoming ownership: got %v, vector %+v", got, v)
	}
	if refs, e := m.RefCount(incoming); e != nil || refs != 2 {
		t.Fatalf("incoming refs = %d, %v", refs, e)
	}
	sxMust(t, v.Close())
	if refs, e := m.RefCount(incoming); e != nil || refs != 1 {
		t.Fatalf("incoming refs after owning cleanup = %d, %v", refs, e)
	}
	sxMust(t, m.DecRef(incoming))
}

func TestSexprRefVectorShrinkKeepsRawPrefixAfterStoppedDeletion(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	first, e := m.MkString("first", 0, 0)
	sxMust(t, e)
	n := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579")
	second, e := m.MkNumeral(&n, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&n))
	v, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, v.PushBack(first))
	sxMust(t, v.PushBack(second))
	q.Manager.Z.SourceFinalizing = true
	e = v.Shrink(0)
	q.Manager.Z.SourceFinalizing = false
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("stopped numeral deletion: %v", e)
	}
	if v.Size() != 2 || v.nodes[0] != first || v.nodes[1] != second || v.owned[0] || v.owned[1] {
		t.Fatalf("source raw vector after stopped range: %+v", v)
	}
	sxMust(t, v.Close())
}
