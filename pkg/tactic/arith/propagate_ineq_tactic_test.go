package arith

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
)

// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:39-103,515-549.
func TestPropagateIneqTacticParameterSnapshotCleanupAndGoalReuse(t *testing.T) {
	m, symbols, q, _ := polynomialASTManager(t)
	numbers, _ := equationManager(t)
	context, value := boundParams(t)
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	boolSort, err := m.MkBoolSort()
	equationMust(t, err)
	mkConst := func(name string, sort ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		symbol, e := symbols.Intern(name)
		equationMust(t, e)
		h, e := m.MkConst(symbol, sort)
		equationMust(t, e)
		return h
	}
	x := mkConst("tactic-x", realSort)
	p := mkConst("tactic-p", boolSort)
	v, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&v, "2"))
	two, err := m.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	equationMust(t, q.Del(&v))
	lower, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	name := "bound_max_refinements"
	equationMust(t, context.SetUIntName(value, &name, 3))
	tactic, err := NewPropagateIneqTactic(m, numbers, context, value)
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	var results []*goaltactic.Goal
	t.Cleanup(func() {
		for _, goal := range results {
			equationMust(t, goal.Close())
		}
		equationMust(t, input.Close())
		equationMust(t, tactic.Close())
	})
	if tactic.current.store.maxRefinements != 3 {
		t.Fatal("constructor parameter snapshot")
	}
	equationMust(t, context.SetUIntName(value, &name, 5))
	equationMust(t, tactic.UpdateParams(value))
	if tactic.current.store.maxRefinements != 5 {
		t.Fatal("source parameter forwarding")
	}
	equationMust(t, input.Assert(lower))
	equationMust(t, input.Assert(p))
	equationMust(t, input.SetDepth(4))
	checkApply := func() {
		t.Helper()
		result, e := tactic.Apply(input)
		equationMust(t, e)
		if result == nil || result == input {
			t.Fatal("tactic did not own a result goal")
		}
		results = append(results, result)
		if depth, e := result.Depth(); e != nil || depth != 5 {
			t.Fatal("result depth", depth, e)
		}
		if size, e := result.Size(); e != nil || size != 2 {
			t.Fatal("result formula count", size, e)
		}
		for i, want := range []ast.ASTHandle{p, lower} {
			got, e := result.Form(uint32(i))
			equationMust(t, e)
			if got != want {
				t.Fatal("source non-bound/bound order", i, got, want)
			}
		}
	}
	checkApply()
	oldStore, oldBuilder := tactic.current.store, tactic.current.builder
	equationMust(t, context.SetUIntName(value, &name, 7))
	equationMust(t, tactic.Cleanup())
	if tactic.current.store == oldStore || tactic.current.builder == oldBuilder || !oldStore.closed || !oldBuilder.closed || tactic.current.store.maxRefinements != 5 {
		t.Fatal("cleanup did not rebuild from the retained parameter snapshot")
	}
	checkApply()
	equationMust(t, tactic.Close())
	if _, err := tactic.Apply(input); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed tactic accepted goal", err)
	}
}

// Source: propagate_ineqs_tactic.cpp:47-49. A translated tactic gets a new
// implementation and its own params_ref copy for the destination manager.
func TestPropagateIneqTacticTranslateOwnsIndependentSnapshot(t *testing.T) {
	sourceManager, _, _, _ := polynomialASTManager(t)
	sourceNumbers, _ := equationManager(t)
	destinationManager, symbols, q, _ := polynomialASTManager(t)
	destinationNumbers, _ := equationManager(t)
	context, value := boundParams(t)
	name := "bound_max_refinements"
	equationMust(t, context.SetUIntName(value, &name, 5))
	source, err := NewPropagateIneqTactic(sourceManager, sourceNumbers, context, value)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, source.Close()) })
	translated, err := source.Translate(destinationManager, destinationNumbers)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, translated.Close()) })
	if translated == source || translated.current.store == source.current.store || translated.current.builder == source.current.builder {
		t.Fatal("translation reused the source implementation")
	}
	equationMust(t, context.SetUIntName(value, &name, 7))
	equationMust(t, source.UpdateParams(value))
	if translated.current.store.maxRefinements != 5 || source.current.store.maxRefinements != 7 {
		t.Fatal("translation did not retain its own parameter snapshot")
	}
	equationMust(t, source.Close())
	equationMust(t, translated.Cleanup())
	if translated.current.store.maxRefinements != 5 {
		t.Fatal("translation cleanup read the source's updated parameters")
	}
	family, err := destinationManager.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := destinationManager.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	xName, err := symbols.Intern("translated-x")
	equationMust(t, err)
	x, err := destinationManager.MkConst(xName, realSort)
	equationMust(t, err)
	v, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&v, "2"))
	two, err := destinationManager.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	equationMust(t, q.Del(&v))
	lower, err := destinationManager.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	input, err := goaltactic.NewGoal(destinationManager, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, input.Close()) })
	equationMust(t, input.Assert(lower))
	result, err := translated.Apply(input)
	equationMust(t, err)
	if result == nil || result == input {
		t.Fatal("translated tactic did not process destination goal")
	}
	t.Cleanup(func() { equationMust(t, result.Close()) })
	if size, err := result.Size(); err != nil || size != 1 {
		t.Fatal("translated result bounds", size, err)
	}
	got, err := result.Form(0)
	equationMust(t, err)
	if got != lower {
		t.Fatal("translated result uses wrong manager", got, lower)
	}
	if _, err := source.Translate(destinationManager, destinationNumbers); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed source translated", err)
	}
}
