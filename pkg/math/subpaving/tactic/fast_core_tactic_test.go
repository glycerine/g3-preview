package tactic

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/diagnostic"
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
	cooperation, err := cooperate.NewCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := cooperation.BeginSection(); err != nil {
		t.Fatal(err)
	}
	var cooperationTrace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&cooperationTrace)
	tags.Enable("cooperate_detail")
	if err := cooperation.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := core.BindCooperation(cooperation); err != nil {
		t.Fatal(err)
	}
	goal, err := goaltactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{goal.Close(), core.Close(), qm.CloseChecked(), pc.Del(&p), pc.Close(), m.Close(), q.Close(), sy.Close(), numbers.CloseChecked(), cooperation.Close()} {
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
	if cooperation.OwnerThread != 0 {
		t.Fatal("source tactic search omitted cooperation checkpoint")
	}
	if tags.ShouldTrace("cooperate_detail") {
		conversion := bytes.Index(cooperationTrace.Bytes(), []byte("\nexpr2subpaving, tid: 0\n"))
		search := bytes.LastIndex(cooperationTrace.Bytes(), []byte("\nsubpaving, tid: 0\n"))
		if conversion < 0 || search <= conversion {
			t.Fatal("source conversion must cooperate before search", cooperationTrace.String())
		}
	}
	if err := cooperation.EndTask(); err != nil {
		t.Fatal(err)
	}
	var converterN, converterD numeral.Int
	if _, err := core.converter.InternalizeTerm(x, &converterN, &converterD); err != nil || cooperation.OwnerThread != 0 {
		t.Fatal("direct converter checkpoint omitted cooperation", err, cooperation.OwnerThread)
	}
	if err := qm.Z.Del(&converterD); err != nil {
		t.Fatal(err)
	}
	if err := qm.Z.Del(&converterN); err != nil {
		t.Fatal(err)
	}
	if err := cooperation.EndTask(); err != nil {
		t.Fatal(err)
	}
	limit := m.ResourceLimit()
	limit.Cancel()
	if _, err := core.converter.InternalizeTerm(x, &converterN, &converterD); !errors.Is(err, ErrExprConverterCanceled) || cooperation.OwnerThread != -1 {
		t.Fatal("cancellation must precede expression cooperation", err, cooperation.OwnerThread)
	}
	limit.ResetCancel()
	if n, err := goal.Size(); err != nil || n != 1 {
		t.Fatal("input goal retained", n, err)
	}
	st := statistics.New()
	if err := core.CollectStatistics(st); err != nil || st.Size() != 2 || st.Key(0) != "new bounds" || st.UintValue(0) != 1 || st.Key(1) != "nodes" || st.UintValue(1) != 1 {
		t.Fatal("source core statistics", err, st.Size())
	}
	display, err := core.converter.VariableDisplay(sy)
	if err != nil {
		t.Fatal(err)
	}
	var shown bytes.Buffer
	if err := display(&shown, 0); err != nil || shown.String() != "x" {
		t.Fatal("inverse expression display", shown.String(), err)
	}
	shown.Reset()
	if err := display(&shown, 99); err != nil || shown.String() != "k!99" {
		t.Fatal("generated variable fallback", shown.String(), err)
	}
	oldEngine, oldConverter := core.engine, core.converter
	if err := cooperation.EndTask(); err != nil {
		t.Fatal(err)
	}
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
	if cooperation.OwnerThread != 0 {
		t.Fatal("cleanup rebuild dropped borrowed cooperation owner")
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
	if err := core.UpdateParams(&p); err != nil {
		t.Fatal("same-engine parameter update", err)
	}
	if options, err := core.engine.Options(); err != nil || !options.PrintNodes {
		t.Fatal("updated live display option", options, err)
	}
	if err := pc.SetBoolName(&p, &printKey, false); err != nil {
		t.Fatal(err)
	}
	if err := core.Cleanup(); err != nil {
		t.Fatal("updated parameter snapshot rebuild", err)
	}
	shown.Reset()
	if got, err := core.ApplyWithOutput(goal, &shown); err != nil || got != goal {
		t.Fatal("caller mutation does not alter updated tactic snapshot", got, err)
	}
	if shown.String() != "x <= 2\nbounds at leaves: \nx <= 2\n" {
		t.Fatal("source print_nodes output", shown.String())
	}
	if err := cooperation.EndSection(); err != nil {
		t.Fatal(err)
	}
	if err := core.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Apply(goal); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed core refusal", err)
	}
	if err := pc.SetBoolName(&p, &printKey, true); err != nil {
		t.Fatal(err)
	}
	printing, err := NewFastCoreTactic(m, numbers, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	quotedName, err := sy.Intern("x y")
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := m.MkConst(quotedName, sort)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := m.DecRef(quoted); e != nil {
			t.Error(e)
		}
	}()
	var n, d numeral.Int
	defer func() {
		if e := qm.Z.Del(&d); e != nil {
			t.Error(e)
		}
		if e := qm.Z.Del(&n); e != nil {
			t.Error(e)
		}
	}()
	quotedVariable, err := printing.converter.InternalizeTerm(quoted, &n, &d)
	if err != nil {
		t.Fatal(err)
	}
	display, err = printing.converter.VariableDisplay(sy)
	if err != nil {
		t.Fatal(err)
	}
	shown.Reset()
	if err := display(&shown, quotedVariable); err != nil || shown.String() != "|x y|" {
		t.Fatal("source SMT2 quoted symbol", shown.String(), err)
	}
	functionName, err := sy.Intern("f g")
	if err != nil {
		t.Fatal(err)
	}
	function, err := m.MkFuncDecl(functionName, []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "-2"); err != nil {
		t.Fatal(err)
	}
	negative, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []struct {
		arg  ast.ASTHandle
		want string
	}{{x, "(|f g| x)"}, {two, "(|f g| 2)"}, {negative, "(|f g| (- 2))"}} {
		application, err := m.MkApp(function, []ast.ASTHandle{term.arg})
		if err != nil {
			t.Fatal(err)
		}
		if err := m.IncRef(application); err != nil {
			t.Fatal(err)
		}
		variable, err := printing.converter.InternalizeTerm(application, &n, &d)
		if err != nil {
			t.Fatal(err)
		}
		display, err := printing.converter.VariableDisplay(sy)
		if err != nil {
			t.Fatal(err)
		}
		shown.Reset()
		if err := display(&shown, variable); err != nil || shown.String() != term.want {
			t.Fatal("uninterpreted arithmetic application", shown.String(), term.want, err)
		}
		if err := m.DecRef(application); err != nil {
			t.Fatal(err)
		}
	}
	realSort, err := m.MkFamilySort(family, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	realFunction, err := m.MkFuncDecl(functionName, []ast.ASTHandle{realSort}, realSort, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "1/2"); err != nil {
		t.Fatal(err)
	}
	fraction, err := m.ArithmeticNumeral(&value, false)
	if err != nil {
		t.Fatal(err)
	}
	fractionApp, err := m.MkApp(realFunction, []ast.ASTHandle{fraction})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(fractionApp); err != nil {
		t.Fatal(err)
	}
	fractionVariable, err := printing.converter.InternalizeTerm(fractionApp, &n, &d)
	if err != nil {
		t.Fatal(err)
	}
	display, err = printing.converter.VariableDisplay(sy)
	if err != nil {
		t.Fatal(err)
	}
	shown.Reset()
	if err := display(&shown, fractionVariable); err != nil || shown.String() != "(|f g| (/ 1.0 2.0))" {
		t.Fatal("source real rational literal in application", shown.String(), err)
	}
	if err := m.DecRef(fractionApp); err != nil {
		t.Fatal(err)
	}
	shown.Reset()
	if got, err := printing.ApplyWithOutput(goal, &shown); err != nil || got != goal || shown.String() != "x <= 2\nbounds at leaves: \nx <= 2\n" {
		t.Fatal("selected print_nodes output", got, err, shown.String())
	}
	if err := printing.Close(); err != nil {
		t.Fatal(err)
	}
}
