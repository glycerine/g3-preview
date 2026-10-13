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

func TestSexprRefVectorStoppedRangeAndDetachedRelease(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(*SexprRefVector) error
		keep bool
	}{
		{"finalize", (*SexprRefVector).Finalize, true},
		{"reset", (*SexprRefVector).Reset, true},
		{"resize", func(v *SexprRefVector) error { return v.Resize(0) }, true},
		{"pop", (*SexprRefVector).PopBack, false},
		{"erase", func(v *SexprRefVector) error { return v.EraseIndex(0) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, _, q := sxOwner(t, arena.NewBudget(1<<24))
			n := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579")
			h, e := m.MkNumeral(&n, 0, 0)
			sxMust(t, e)
			sxMust(t, q.Del(&n))
			v, e := m.NewRefVector()
			sxMust(t, e)
			sxMust(t, v.PushBack(h))
			q.Manager.Z.SourceFinalizing = true
			e = tc.act(v)
			q.Manager.Z.SourceFinalizing = false
			if !errors.Is(e, arena.ErrClosed) {
				t.Fatalf("stopped numeral deletion: %v", e)
			}
			if tc.keep {
				got, e := v.Get(0)
				sxMust(t, e)
				if v.Size() != 1 || got != h || v.owned[0] {
					t.Fatalf("source range publication after stopped DEC: %+v", v)
				}
			} else if v.Size() != 0 || len(v.pending) != 0 {
				t.Fatalf("source detached entry after consumed DEC: %+v", v)
			}
			sxMust(t, v.Close())
		})
	}
}

func TestSexprRefVectorMoveAndSwapCarryStoppedSetReceipt(t *testing.T) {
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
	moved, e := v.Move()
	sxMust(t, e)
	if v.Size() != 0 || len(v.pending) != 0 || moved.Size() != 1 || len(moved.pending) != 1 {
		t.Fatalf("move ownership: source %+v, moved %+v", v, moved)
	}
	other, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, moved.Swap(other))
	if moved.Size() != 0 || len(moved.pending) != 0 || other.nodes[0] != old || other.owned[0] || other.pending[0] != incoming {
		t.Fatalf("swap ownership: moved %+v, other %+v", moved, other)
	}
	if _, e := other.Clone(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("clone of consumed raw entry: %v", e)
	}
	sxMust(t, other.Close())
	sxMust(t, moved.Close())
	sxMust(t, v.Close())
	if refs, e := m.RefCount(incoming); e != nil || refs != 1 {
		t.Fatalf("incoming refs after transfer cleanup = %d, %v", refs, e)
	}
	sxMust(t, m.DecRef(incoming))
}
