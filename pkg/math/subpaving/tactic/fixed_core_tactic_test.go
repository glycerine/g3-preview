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
