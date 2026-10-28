package sat

// Source: src/sat/sat_solver.cpp:34–58; src/util/memory_manager.cpp; src/util/gparams.cpp.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSATRuntimeSharedProcessAndNumeralCohort(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	p := umemory.NewProcessAccounting()
	numericRuntime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	satOK(t, e)
	integers := unumeral.IntegerConfig{Memory: numericRuntime, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: integers})
	satOK(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	satOK(t, e)
	ctx, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b})
	satOK(t, e)
	before := b.Used()
	if rejected, err := NewRuntime(RuntimeConfig{Budget: b, ParamsContext: ctx}); rejected != nil || !errors.Is(err, ErrSATOwner) || b.Used() != before {
		t.Fatal("unbound SAT process accepted", rejected, err, b.Used(), before)
	}
	if rejected, err := NewRuntime(RuntimeConfig{Budget: b, ParamsContext: ctx, Process: umemory.NewProcessAccounting()}); rejected != nil || !errors.Is(err, ErrSATOwner) || b.Used() != before {
		t.Fatal("foreign SAT process accepted", rejected, err, b.Used(), before)
	}
	r, e := NewRuntime(RuntimeConfig{Budget: b, ParamsContext: ctx, Process: p})
	satOK(t, e)
	if r.memory.ProcessOwner() != p {
		t.Fatal("SAT raw allocator lost shared process")
	}
	params, e := ctx.NewValue()
	satOK(t, e)
	limit, e := control.NewReslimit(b)
	satOK(t, e)
	solver, e := NewSATSolver(SolverConfig{ParamsContext: ctx, Params: params, RLimit: limit, Budget: b, Runtime: r})
	satOK(t, e)
	variable, e := solver.MkVar(true, true)
	satOK(t, e)
	literal := MakeLiteral(variable, false)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	satOK(t, e)
	beforeBytes := p.AllocationSize()
	satOK(t, solver.displayMemoryStat(out))
	beforeText, e := out.String()
	satOK(t, e)
	if want := fmt.Sprintf(" :memory %.2f", float64(beforeBytes)/(1024*1024)); beforeText != want {
		t.Fatal("SAT report before cross-runtime allocation", beforeText, want)
	}
	satOK(t, out.ResetText())
	other, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	satOK(t, e)
	block, e := other.Allocate(1 << 20)
	satOK(t, e)
	actual, e := r.memory.AllocationSize()
	satOK(t, e)
	if actual != p.AllocationSize() || actual < beforeBytes+(1<<20) {
		t.Fatal("SAT owner cannot see second runtime allocation", actual, p.AllocationSize())
	}
	// sat_solver.h checks the process total only at every tenth checkpoint.
	// A foreign Runtime must therefore trigger the same strict source cutoff.
	propagation := solver.propagation
	propagation.checkpointEnabled = true
	propagation.config.state.MaxMemory = beforeBytes
	for i := 0; i < 9; i++ {
		satOK(t, propagation.checkpoint())
	}
	if propagation.numCheckpoints != 9 {
		t.Fatal("solver memory checkpoint ran before source cadence", propagation.numCheckpoints)
	}
	if e := propagation.checkpoint(); !errors.Is(e, ErrSATSolverMaxMemory) || propagation.numCheckpoints != 0 {
		t.Fatal("solver ignored foreign process allocation", e, propagation.numCheckpoints)
	}
	if !solver.probing.cacheEnabled {
		t.Fatal("source probing cache unexpectedly disabled")
	}
	solver.probing.cacheLimit = beforeBytes
	satOK(t, solver.probing.cacheBins(literal, 0))
	if _, cached, e := solver.probing.cachedImpliedLits(literal); e != nil || cached {
		t.Fatal("probing cache ignored another runtime's process allocation", cached, e)
	}
	satOK(t, solver.displayMemoryStat(out))
	afterText, e := out.String()
	satOK(t, e)
	if want := fmt.Sprintf(" :memory %.2f", float64(actual)/(1024*1024)); afterText != want || afterText == beforeText {
		t.Fatal("SAT report omitted cross-runtime allocation", afterText, want, beforeText)
	}
	satOK(t, other.Deallocate(&block))
	for i := 0; i < 10; i++ {
		satOK(t, propagation.checkpoint())
	}
	if propagation.numCheckpoints != 0 {
		t.Fatal("solver memory checkpoint did not reset at equality", propagation.numCheckpoints)
	}
	propagation.config.state.MaxMemory = ^uint64(0)
	satOK(t, solver.probing.cacheBins(literal, 0))
	if _, cached, e := solver.probing.cachedImpliedLits(literal); e != nil || !cached {
		t.Fatal("probing cache did not resume at the process limit", cached, e)
	}
	block, e = other.Allocate(1 << 20)
	satOK(t, e)
	if done, e := solver.probing.run(true); e != nil || !done {
		t.Fatal("source probing run after process limit", done, e)
	}
	if solver.probing.cache.blockLive() {
		t.Fatal("probing run retained cache above process limit")
	}
	satOK(t, other.Deallocate(&block))
	satOK(t, other.Close())
	satOK(t, out.Close())
	satOK(t, solver.Close())
	limit.Close()
	satOK(t, ctx.Del(&params))
	satOK(t, r.Close())
	satOK(t, ctx.Close())
	satOK(t, q.Close())
	satOK(t, sy.Close())
	satOK(t, numericRuntime.Close())
	if p.AllocationSize() != 0 || b.Used() != 0 {
		t.Fatal("retained SAT process or budget", p.AllocationSize(), b.Used())
	}
}

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
