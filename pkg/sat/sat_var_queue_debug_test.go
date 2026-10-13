//go:build g3debug

package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSolverQueuesBorrowHeapDebugAndStopBeforeReset(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	p, err := ctx.NewValue()
	satOK(t, err)
	t.Cleanup(func() { satOK(t, ctx.Del(&p)) })
	runtime, err := NewRuntime(RuntimeConfig{Budget: ctx.BudgetField, ParamsContext: ctx})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, runtime.Close()) })
	limit, err := control.NewReslimit(ctx.BudgetField)
	satOK(t, err)
	t.Cleanup(limit.Close)
	db := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: db}})
	satOK(t, err)
	debug, err := diagnostic.NewDebugContext(region, db)
	satOK(t, err)
	t.Cleanup(func() {
		satOK(t, debug.Close())
		satOK(t, region.Close())
		if db.Used() != 0 {
			t.Fatal("borrowed SAT debug owner leak", db.Used())
		}
	})
	satOK(t, debug.EnableDebug("heap"))
	s, err := NewSATSolver(SolverConfig{ParamsContext: ctx, Params: p, RLimit: limit,
		Budget: ctx.BudgetField, Runtime: runtime, Debug: debug})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, s.Close()) })
	if s.Variables.queue.debug != debug || s.simplifier.debug != debug {
		t.Fatal("actual SAT owners lost borrowed heap debug context")
	}
	a, q := s.Variables.activity, s.Variables.queue
	for i := BoolVar(0); i < 2; i++ {
		v, e := s.MkVar(false, true)
		satOK(t, e)
		if v != i {
			t.Fatal("SAT variable owner publication", v, i)
		}
	}
	satOK(t, a.Set(0, 5))
	satOK(t, a.Set(1, 4))
	satOK(t, a.Set(1, 6))
	satOK(t, debug.DisableDebug("heap"))
	satOK(t, q.reserve(128))
	satOK(t, debug.EnableDebug("heap"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, q.reserve(128))
	satOK(t, debug.EnableAssertions(true))
	satOK(t, q.DelVar(99)) // wrapper skips source erase for an absent variable
	if err := q.Reset(); !errors.Is(err, diagnostic.ErrDebugNotificationBinding) || q.Empty() {
		t.Fatal("SAT priority queue missed nested source assertion before reset", err, q.Empty())
	}
	satOK(t, a.Set(1, 4))
	blocked, err := newSATSimplifierBlockedQueue(s.simplifier)
	satOK(t, err)
	if blocked.debug != debug {
		t.Fatal("blocked-clause queue lost borrowed debug context")
	}
	satOK(t, blocked.insert(0))
	blocked.indices[0] = 0
	if err := blocked.Reset(); !errors.Is(err, diagnostic.ErrDebugNotificationBinding) || blocked.Empty() {
		t.Fatal("blocked-clause queue missed source inverse-index assertion", err, blocked.Empty())
	}
	blocked.indices[0] = 1
	satOK(t, blocked.Close())
}
