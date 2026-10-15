package tactic

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestFixedCoreTacticBorrowedGoalPrintStatisticsAndCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	family, err := m.RegisterArithmeticPlugin()
	if err != nil {
		t.Fatal(err)
	}
	sort, err := m.MkFamilySort(family, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	engineName, err := sy.Intern("mpfx")
	if err != nil {
		t.Fatal(err)
	}
	engineKey, depthKey, printKey := "numeral", "max_depth", "print_nodes"
	if err := pc.SetSymbolName(&p, &engineKey, engineName); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depthKey, 0); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	core, err := NewFixedCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	goal, err := goaltactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	value, err := q.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{goal.Close(), core.Close(), qm.CloseChecked(), numbers.Close(), pc.Del(&p), pc.Close(), q.Del(&value), m.Close(), q.Close(), sy.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("MPFX core owner bytes", b.Used())
		}
	})
	name, err := sy.Intern("x")
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.MkConst(name, sort)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "2"); err != nil {
		t.Fatal(err)
	}
	two, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	ineq, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{x, two})
	if err != nil {
		t.Fatal(err)
	}
	if err := goal.Assert(ineq); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if got, err := core.ApplyWithOutput(goal, &out); err != nil || got != goal {
		t.Fatal("borrowed source result", got, err)
	}
	if want := "x <= 2\nbounds at leaves: \nx <= 2\n"; out.String() != want {
		t.Fatal("selected print_nodes", out.String())
	}
	st := statistics.New()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 2 || st.Key(0) != "new bounds" || st.UintValue(0) != 1 || st.Key(1) != "nodes" || st.UintValue(1) != 1 {
		t.Fatal("source core statistics", err)
	}
	m.ResourceLimit().Cancel()
	if got, err := core.ApplyWithOutput(goal, &out); got != nil {
		t.Fatal("canceled result", got, err)
	} else {
		var tacticErr goaltactic.TacticException
		if !errors.As(err, &tacticErr) || tacticErr.Message() != "canceled" {
			t.Fatal("source cancellation exception translation", err)
		}
	}
	m.ResourceLimit().ResetCancel()
	if err := pc.SetBoolName(&p, &printKey, false); err != nil {
		t.Fatal(err)
	}
	oldEngine, oldConverter := core.engine, core.converter
	if err := core.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := oldEngine.Shell(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("old context retired", err)
	}
	if err := oldConverter.check(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("old converter retired", err)
	}
	if options, err := core.engine.Options(); err != nil || !options.PrintNodes {
		t.Fatal("retained parameter snapshot", options, err)
	}
	if err := core.UpdateParams(&p); err != nil {
		t.Fatal(err)
	}
	if options, err := core.engine.Options(); err != nil || options.PrintNodes {
		t.Fatal("selected option update", options, err)
	}
	if got, err := core.Apply(goal); err != nil || got != goal {
		t.Fatal("fresh context and borrowed goal", got, err)
	}
	st.Reset()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 4 {
		t.Fatal("accumulated source statistics", err, st.Size())
	}
	if err := core.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 0 {
		t.Fatal("outer reset", err)
	}
	if err := q.SetString(&value, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	large, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	product, err := m.MkFamilyApp(family, 9, nil, []ast.ASTHandle{large, x})
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "0"); err != nil {
		t.Fatal(err)
	}
	zero, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{product, zero})
	if err != nil {
		t.Fatal(err)
	}
	inexact, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{sum, two})
	if err != nil {
		t.Fatal(err)
	}
	badGoal, err := goaltactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := badGoal.Assert(inexact); err != nil {
		t.Fatal(err)
	}
	badCore, err := NewFixedCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := badCore.Apply(badGoal); got != nil {
		t.Fatal("inexact coefficient returned a goal", got, err)
	} else {
		var tacticErr goaltactic.TacticException
		if !errors.As(err, &tacticErr) || tacticErr.Message() != "failed to internalize goal into subpaving module" {
			t.Fatal("source inexact-coefficient translation", err)
		}
	}
	if err := badGoal.Close(); err != nil {
		t.Fatal(err)
	}
	if err := badCore.Close(); err != nil {
		t.Fatal(err)
	}
	splitParams, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.Assign(&splitParams, &p); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&splitParams, &depthKey, 1); err != nil {
		t.Fatal(err)
	}
	realSort, err := m.MkFamilySort(family, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	realName, err := sy.Intern("xr")
	if err != nil {
		t.Fatal(err)
	}
	xr, err := m.MkConst(realName, realSort)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "0"); err != nil {
		t.Fatal(err)
	}
	zeroBound, err := m.ArithmeticNumeral(&value, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "1/4294967296"); err != nil {
		t.Fatal(err)
	}
	tinyBound, err := m.ArithmeticNumeral(&value, false)
	if err != nil {
		t.Fatal(err)
	}
	if text, _, matched, err := m.ArithmeticNumeralText(tinyBound); err != nil || !matched || text != "1/4294967296" {
		t.Fatal("tiny rational AST numeral", text, matched, err)
	}
	lower, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{xr, zeroBound})
	if err != nil {
		t.Fatal(err)
	}
	upper, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{xr, tinyBound})
	if err != nil {
		t.Fatal(err)
	}
	splitGoal, err := goaltactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := splitGoal.Assert(lower); err != nil {
		t.Fatal(err)
	}
	if err := splitGoal.Assert(upper); err != nil {
		t.Fatal(err)
	}
	splitCore, err := NewFixedCoreTactic(m, numbers, qm, pc, &splitParams)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := splitCore.Apply(splitGoal); got != nil {
		var observed bytes.Buffer
		if shell, shellErr := splitCore.engine.Shell(); shellErr == nil {
			_ = shell.DisplayBounds(&observed)
		}
		t.Fatal("failed split returned a goal", got, err, observed.String())
	} else {
		var tacticErr goaltactic.TacticException
		if !errors.As(err, &tacticErr) || tacticErr.Message() != "failed building subpaving tree..." {
			t.Fatal("source split exception translation", err)
		}
	}
	if err := splitGoal.Close(); err != nil {
		t.Fatal(err)
	}
	if err := splitCore.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pc.Del(&splitParams); err != nil {
		t.Fatal(err)
	}
	owned, err := NewFixedCoreTacticOwned(m, pc, &p)
	if err != nil {
		t.Fatal("owning source default managers", err)
	}
	if got, err := owned.Apply(goal); err != nil || got != goal {
		t.Fatal("owning MPFX tactic", got, err)
	}
	if err := owned.Close(); err != nil {
		t.Fatal("owned numeral retirement", err)
	}
}
