package arith

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
)

// Source: propagate_ineqs_tactic.cpp:60-62; tactical.cpp:1107-1128.
func TestCleanPropagateIneqFactoryDefaultParamsAndPerCallRebuild(t *testing.T) {
	m, symbols, q, _ := polynomialASTManager(t)
	numbers, _ := equationManager(t)
	context, value := boundParams(t)
	clean, err := NewCleanPropagateIneqTactic(m, numbers, context, nil)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, clean.Close()) })
	store := clean.core.current.store
	if store.maxRefinements != 16 || store.threshold != 0.05 || store.smallInterval != 128 || store.strict2double != 0.00001 {
		t.Fatal("source default parameter value", store.maxRefinements, store.threshold, store.smallInterval, store.strict2double)
	}
	boolSort, err := m.MkBoolSort()
	equationMust(t, err)
	pName, err := symbols.Intern("clean-p")
	equationMust(t, err)
	p, err := m.MkConst(pName, boolSort)
	equationMust(t, err)
	noBounds, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, noBounds.Close()) })
	equationMust(t, noBounds.Assert(p))
	old := clean.core.current.builder
	got, err := clean.Apply(noBounds)
	equationMust(t, err)
	if got != noBounds || !old.closed || clean.core.current.builder == old {
		t.Fatal("clean factory did not rebuild after no-bound return")
	}
	name := "bound_max_refinements"
	equationMust(t, context.SetUIntName(value, &name, 5))
	equationMust(t, clean.UpdateParams(value))
	if clean.core.current.store.maxRefinements != 5 {
		t.Fatal("factory parameter forwarding")
	}
	translated, err := clean.Translate(m, numbers)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, translated.Close()) })
	if translated.core.current.store == clean.core.current.store || translated.core.current.store.maxRefinements != 5 {
		t.Fatal("clean translation did not copy implementation and parameters")
	}
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	xName, err := symbols.Intern("clean-x")
	equationMust(t, err)
	x, err := m.MkConst(xName, realSort)
	equationMust(t, err)
	v, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&v, "2"))
	two, err := m.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	equationMust(t, q.Del(&v))
	lower, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, input.Close()) })
	equationMust(t, input.Assert(lower))
	old = clean.core.current.builder
	result, err := clean.Apply(input)
	equationMust(t, err)
	if result == nil || result == input || !old.closed || clean.core.current.builder == old || clean.core.current.store.maxRefinements != 5 {
		t.Fatal("clean factory did not rebuild after bound result")
	}
	t.Cleanup(func() { equationMust(t, result.Close()) })
	if size, err := result.Size(); err != nil || size != 1 {
		t.Fatal("clean result", size, err)
	}
	coreGoal, err := goaltactic.NewGoal(m, false, false, true)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, coreGoal.Close()) })
	old = clean.core.current.builder
	if refused, err := clean.Apply(coreGoal); refused != nil || err == nil || clean.core.current.builder != old || old.closed {
		t.Fatal("failed source call unexpectedly ran cleanup", refused, err)
	}
	equationMust(t, clean.Close())
	if _, err := clean.Apply(input); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed clean wrapper accepted goal", err)
	}
	old = translated.core.current.builder
	got, err = translated.Apply(noBounds)
	equationMust(t, err)
	if got != noBounds || !old.closed {
		t.Fatal("translated clean wrapper did not rebuild")
	}
}
