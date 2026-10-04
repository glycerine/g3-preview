// Source: src/sat/sat_solver.cpp:34–58; src/util/memory_manager.cpp; src/util/gparams.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestSATFullRuntimeActualOwnersAndColdUnwind(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	before := ctx.budget.Used()
	r, e := newSATRuntime(satRuntimeConfig{Budget: ctx.budget, ParamsContext: ctx})
	satOK(t, e)
	if r.memory == nil || r.memory.Budget() != ctx.budget || r.globals == nil || r.globals.context != ctx {
		t.Fatal("runtime globals/memory are not actual owners")
	}
	copied := *r
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, r.Close())
	if ctx.budget.Used() != before {
		t.Fatal("owned runtime graph", ctx.budget.Used(), before)
	}
	satOK(t, r.Close())
}
func TestSATFullRuntimeRejectsForeignBudgetBeforeReservation(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	b.FailAfter(0)
	if r, e := newSATRuntime(satRuntimeConfig{Budget: b, ParamsContext: ctx}); r != nil || !errors.Is(e, ErrSATOwner) {
		t.Fatal(r, e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
