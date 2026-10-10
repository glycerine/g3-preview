package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverNativeTrailColdHeaderAndRetirement(t *testing.T) {
	header := uint64(reflect.TypeOf(satSolverTrail{}).Size())
	short := arena.NewBudget(header - 1)
	if trail, _, e := newSATSolverTrail(short); trail != nil || !errors.Is(e, arena.ErrBudget) || short.Used() != 0 {
		t.Fatal("cold header refusal leaked", trail, e, short.Used())
	}
	b := arena.NewBudget(header)
	b.FailAfter(0)
	if trail, _, e := newSATSolverTrail(b); trail != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 {
		t.Fatal("injected cold refusal leaked", trail, e, b.Used())
	}
	b.FailAfter(1)
	trail, root, e := newSATSolverTrail(b)
	satOK(t, e)
	for _, l := range []Literal{0, 1, 0xfffffffe, 0xffffffff} {
		satOK(t, trail.Push(root, l))
	}
	if b.Used() != header {
		t.Fatal("literal backing was charged", b.Used())
	}
	for _, bad := range []satSolverLiteralListRef{
		{owner: root.owner + 1, generation: root.generation},
		{owner: root.owner, slot: 1, generation: root.generation},
		{owner: root.owner, generation: root.generation + 1},
	} {
		if trail.IsLive(bad) {
			t.Fatal("invalid root accepted", bad)
		}
	}
	satOK(t, trail.Free(root))
	if trail.IsLive(root) || trail.values != nil {
		t.Fatal("freed native trail retained values")
	}
	if e = trail.Push(root, 42); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("freed trail accepted push", e)
	}
	satOK(t, trail.Close())
	satOK(t, trail.Close())
	if b.Used() != 0 {
		t.Fatal("cold header refund", b.Used())
	}
	if _, e = trail.Size(root); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed native trail", e)
	}
}
