// Source: src/sat/sat_solver.cpp:34–58; src/util/memory_manager.cpp; src/util/gparams.cpp.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATFullRuntimeActualOwnersAndColdUnwind(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	before := ctx.BudgetField.Used()
	r, e := NewRuntime(RuntimeConfig{Budget: ctx.BudgetField, ParamsContext: ctx})
	satOK(t, e)
	if r.memory == nil || r.memory.Budget() != ctx.BudgetField || r.globals == nil || r.globals.ContextField != ctx {
		t.Fatal("runtime globals/memory are not actual owners")
	}
	copied := *r
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, r.Close())
	if ctx.BudgetField.Used() != before {
		t.Fatal("owned runtime graph", ctx.BudgetField.Used(), before)
	}
	satOK(t, r.Close())
}
func TestSATFullRuntimeRejectsForeignBudgetBeforeReservation(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	b.FailAfter(0)
	if r, e := NewRuntime(RuntimeConfig{Budget: b, ParamsContext: ctx}); r != nil || !errors.Is(e, ErrSATOwner) {
		t.Fatal(r, e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
