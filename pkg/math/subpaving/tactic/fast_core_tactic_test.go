package tactic

import (
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

func TestFastCoreTacticReturnsBorrowedGoalAndCollectsSearchStats(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity = 4
	numbers, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
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
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := sy.Intern("mpff")
	if err != nil {
		t.Fatal(err)
	}
	engineKey, depthKey := "numeral", "max_depth"
	if err := pc.SetSymbolName(&p, &engineKey, selected); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depthKey, 0); err != nil {
		t.Fatal(err)
	}
	family, err := m.RegisterArithmeticPlugin()
	if err != nil {
		t.Fatal(err)
	}
	core, err := NewFastCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	goal, err := goaltactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{goal.Close(), core.Close(), qm.CloseChecked(), pc.Del(&p), pc.Close(), m.Close(), q.Close(), sy.Close(), numbers.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("retained tactic owners", b.Used())
		}
	})
	sort, err := m.MkFamilySort(family, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	name, err := sy.Intern("x")
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.MkConst(name, sort)
	if err != nil {
		t.Fatal(err)
	}
	value, err := q.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := q.Del(&value); e != nil {
			t.Error(e)
		}
	}()
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
	if got, err := core.Apply(goal); err != nil || got != goal {
		t.Fatal("source core returns original goal", got, err)
	}
	if n, err := goal.Size(); err != nil || n != 1 {
		t.Fatal("input goal retained", n, err)
	}
	st := statistics.New()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 2 || st.Key(0) != "new bounds" || st.UintValue(0) != 1 || st.Key(1) != "nodes" || st.UintValue(1) != 1 {
		t.Fatal("source core statistics", err, st.Size())
	}
	oldEngine, oldConverter := core.engine, core.converter
	printKey := "print_nodes"
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	if err := core.Cleanup(); err != nil {
		t.Fatal("fresh implementation from retained parameters", err)
	}
	if _, err := oldEngine.Shell(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("old context retired", err)
	}
	if err := oldConverter.check(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("old converter retired before context", err)
	}
	if options, err := core.engine.Options(); err != nil || options.PrintNodes {
		t.Fatal("caller mutation did not alter tactic snapshot", options, err)
	}
	st.Reset()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 2 || st.UintValue(0) != 1 || st.UintValue(1) != 1 {
		t.Fatal("outer statistics survive cleanup", err, st.Size())
	}
	if got, err := core.Apply(goal); err != nil || got != goal {
		t.Fatal("fresh context searches retained goal", got, err)
	}
	st.Reset()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 4 || st.Key(2) != "new bounds" || st.UintValue(2) != 1 || st.Key(3) != "nodes" || st.UintValue(3) != 1 {
		t.Fatal("source tactic accumulates per-apply contributions", err, st.Size())
	}
	if err := core.ResetStatistics(); err != nil {
		t.Fatal(err)
	}
	st.Reset()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 0 {
		t.Fatal("outer statistics reset", err, st.Size())
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Apply(goal); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed core refusal", err)
	}
	printing, err := NewFastCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := printing.Apply(goal); got != nil || !errors.Is(err, ErrFastCorePrintNodes) {
		t.Fatal("unported variable printer refuses before mutating goal", got, err)
	}
	if err := printing.Close(); err != nil {
		t.Fatal(err)
	}
}
