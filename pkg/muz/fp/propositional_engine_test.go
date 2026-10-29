package fp_test

// Source: src/muz/base/dl_context.cpp query orchestration and the native
// fixedpoint Datalog result for the positive propositional Horn fragment.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/fp"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/statistics"
)

func fpMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fpReference(t *testing.T, script string) []string {
	t.Helper()
	path := os.Getenv("G3_CPP_REFERENCE")
	if path == "" {
		var err error
		path, err = filepath.Abs("../../../build/z3")
		fpMust(t, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("C++ reference executable unavailable:", err)
	}
	command := exec.Command(path, "-in")
	command.Stdin = strings.NewReader(script)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(output))
	}
	return strings.Fields(string(output))
}

func TestPropositionalEngineMatchesCPPFixedpointQueries(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngine(manager)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	declarations := make(map[string]ast.ASTHandle)
	applications := make(map[string]ast.ASTHandle)
	for _, name := range []string{"P", "Q", "R", "U"} {
		symbol, e := manager.Symbols.Intern("fp-" + name)
		fpMust(t, e)
		declarations[name], e = manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, e)
		applications[name], e = manager.MkApp(declarations[name], nil)
		fpMust(t, e)
		fpMust(t, engine.RegisterPredicate(declarations[name]))
	}
	add := func(formula ast.ASTHandle, name string) {
		t.Helper()
		reference, e := manager.NewRef(formula)
		fpMust(t, e)
		t.Cleanup(func() { fpMust(t, manager.DelRef(&reference)) })
		fpMust(t, engine.AddRule(formula, name))
	}
	add(applications["P"], "fact-P")
	for _, edge := range []struct{ from, to string }{{"P", "Q"}, {"Q", "R"}} {
		formula, e := manager.MkImplies(applications[edge.from], applications[edge.to])
		fpMust(t, e)
		add(formula, edge.from+"-"+edge.to)
	}
	got := make([]lbool.LBool, 2)
	got[0], err = engine.Query(applications["R"])
	fpMust(t, err)
	compiled, err := engine.LastQueryUsedCompiledCode()
	fpMust(t, err)
	if !compiled {
		t.Fatal("supported finite Horn program bypassed relational bytecode")
	}
	got[1], err = engine.Query(applications["U"])
	fpMust(t, err)

	cpp := fpReference(t, `(set-option :fixedpoint.engine datalog)
(declare-rel P ())
(declare-rel Q ())
(declare-rel R ())
(declare-rel U ())
(rule P)
(rule (=> P Q))
(rule (=> Q R))
(query R :print-answer false)
(query U :print-answer false)
`)
	if len(cpp) != 2 {
		t.Fatal("C++ fixedpoint output", cpp)
	}
	want := map[lbool.LBool]string{lbool.True: "sat", lbool.False: "unsat"}
	for i := range got {
		if cpp[i] != want[got[i]] {
			t.Fatal("fixedpoint differential", i, got[i], cpp[i])
		}
	}
}

func TestPDREngineMatchesCPPIntegerHornQueries(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	globals, err := context.GlobalParameters()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngineWithServices(manager, globals.Context, globals)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	fpMust(t, engine.SelectEngine("pdr"))
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	arithmetic, err := manager.ArithmeticFamilyID()
	fpMust(t, err)
	integer, err := manager.MkFamilySort(arithmetic, 1, nil)
	fpMust(t, err)
	declarations := make(map[string]ast.ASTHandle)
	for _, name := range []string{"P", "Q", "U"} {
		symbol, x := manager.Symbols.Intern("pdr-integer-" + name)
		fpMust(t, x)
		declarations[name], x = manager.MkFuncDecl(symbol, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, x)
		fpMust(t, engine.RegisterPredicate(declarations[name]))
	}
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	fpMust(t, err)
	pZero, err := manager.MkApp(declarations["P"], []ast.ASTHandle{zero})
	fpMust(t, err)
	fpMust(t, engine.AddRule(pZero, "integer-fact"))
	xName, err := manager.Symbols.Intern("pdr-integer-x")
	fpMust(t, err)
	x, err := manager.MkConst(xName, integer)
	fpMust(t, err)
	fpMust(t, engine.RegisterVariable(x))
	pX, err := manager.MkApp(declarations["P"], []ast.ASTHandle{x})
	fpMust(t, err)
	qX, err := manager.MkApp(declarations["Q"], []ast.ASTHandle{x})
	fpMust(t, err)
	step, err := manager.MkImplies(pX, qX)
	fpMust(t, err)
	fpMust(t, engine.AddRule(step, "integer-step"))
	gotReachable, err := engine.QueryPredicate(declarations["Q"])
	fpMust(t, err)
	answer, err := engine.Answer(declarations["Q"])
	fpMust(t, err)
	answerHandle, err := manager.RefGet(answer)
	fpMust(t, err)
	if !manager.IsTrue(answerHandle) {
		t.Fatal("PDR existential declaration answer is not true")
	}
	fpMust(t, manager.DelRef(&answer))
	if unexpected, x := engine.Answer(declarations["U"]); x == nil {
		fpMust(t, manager.DelRef(&unexpected))
		t.Fatal("PDR answer accepted a declaration other than the latest query")
	}
	stats := statistics.New()
	fpMust(t, engine.CollectStatistics(stats))
	if stats.Size() == 0 {
		t.Fatal("PDR engine exposed no statistics after reachable query")
	}
	gotUnreachable, err := engine.QueryPredicate(declarations["U"])
	fpMust(t, err)
	reusedStats := statistics.New()
	fpMust(t, engine.CollectStatistics(reusedStats))
	if reusedStats.Size() == 0 {
		t.Fatal("PDR engine discarded cumulative statistics across an unchanged program")
	}
	fpMust(t, engine.ResetStatistics())
	if gotReachable != lbool.True || gotUnreachable != lbool.False {
		t.Fatal("PDR integer Horn results", gotReachable, gotUnreachable)
	}
	certificate, err := engine.Answer(declarations["U"])
	fpMust(t, err)
	certificateHandle, err := manager.RefGet(certificate)
	fpMust(t, err)
	if manager.IsTrue(certificateHandle) {
		t.Fatal("PDR unsuccessful query exposed an empty certificate")
	}
	fpMust(t, manager.DelRef(&certificate))

	cpp := fpReference(t, `(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(declare-rel Q (Int))
(declare-rel U (Int))
(declare-var x Int)
(rule (P 0))
(rule (=> (P x) (Q x)))
(query Q :print-answer false)
(query U :print-answer false)
`)
	if len(cpp) != 2 || cpp[0] != "sat" || cpp[1] != "unsat" {
		t.Fatal("C++ PDR fixedpoint output", cpp)
	}
}

func TestSpacerEngineRunsReachableAndInductiveHornQueries(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	globals, err := context.GlobalParameters()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngineWithServices(manager, globals.Context, globals)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	fpMust(t, engine.SelectEngine("spacer"))
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	declarations := make(map[string]ast.ASTHandle)
	applications := make(map[string]ast.ASTHandle)
	for _, name := range []string{"P", "Q", "U"} {
		symbol, x := manager.Symbols.Intern("spacer-engine-" + name)
		fpMust(t, x)
		declarations[name], x = manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, x)
		applications[name], x = manager.MkApp(declarations[name], nil)
		fpMust(t, x)
		fpMust(t, engine.RegisterPredicate(declarations[name]))
	}
	fpMust(t, engine.AddRule(applications["P"], "spacer-fact"))
	step, err := manager.MkImplies(applications["P"], applications["Q"])
	fpMust(t, err)
	fpMust(t, engine.AddRule(step, "spacer-step"))
	recursive, err := manager.MkImplies(applications["U"], applications["U"])
	fpMust(t, err)
	fpMust(t, engine.AddRule(recursive, "spacer-unreachable"))
	reachable, err := engine.QueryPredicate(declarations["Q"])
	fpMust(t, err)
	if reachable != lbool.True {
		t.Fatal("Spacer reachable query", reachable)
	}
	levels, err := engine.NumLevels(declarations["Q"])
	fpMust(t, err)
	if levels == 0 {
		t.Fatal("Spacer query exposed no frames")
	}
	answer, err := engine.Answer(declarations["Q"])
	fpMust(t, err)
	answerHandle, err := manager.RefGet(answer)
	fpMust(t, err)
	if !manager.IsTrue(answerHandle) {
		t.Fatal("Spacer fallback answer", answerHandle)
	}
	fpMust(t, manager.DelRef(&answer))
	trace, names, err := engine.RulesAlongTrace()
	fpMust(t, err)
	if len(trace) != 1 || len(names) != 1 || names[0] != "spacer-step" {
		t.Fatal("Spacer preorder trace rules", names, len(trace))
	}
	for index := len(trace) - 1; index >= 0; index-- {
		fpMust(t, manager.DelRef(&trace[index]))
	}
	stats := statistics.New()
	fpMust(t, engine.CollectStatistics(stats))
	if stats.Size() == 0 {
		t.Fatal("Spacer engine exposed no statistics")
	}
	unreachable, err := engine.QueryPredicate(declarations["U"])
	fpMust(t, err)
	if unreachable != lbool.False {
		t.Fatal("Spacer inductive query", unreachable)
	}
	certificate, err := engine.Answer(declarations["U"])
	fpMust(t, err)
	certificateHandle, err := manager.RefGet(certificate)
	fpMust(t, err)
	if manager.IsTrue(certificateHandle) {
		t.Fatal("Spacer unsuccessful query exposed an empty certificate")
	}
	fpMust(t, manager.DelRef(&certificate))
	inductiveModel, err := engine.Model(declarations["U"])
	fpMust(t, err)
	if declarations, err := inductiveModel.Core().NumDeclarations(); err != nil || declarations == 0 {
		t.Fatal("Spacer inductive model", declarations, err)
	}
	fpMust(t, inductiveModel.Close())
}

func TestSpacerEngineExpandsSlicedReachableColumns(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	globals, err := context.GlobalParameters()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngineWithServices(manager, globals.Context, globals)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	fpMust(t, engine.SelectEngine("spacer"))
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	arithmetic, err := manager.ArithmeticFamilyID()
	fpMust(t, err)
	integer, err := manager.MkFamilySort(arithmetic, 1, nil)
	fpMust(t, err)
	pName, err := manager.Symbols.Intern("spacer-sliced-answer-P")
	fpMust(t, err)
	predicate, err := manager.MkFuncDecl(pName, []ast.ASTHandle{integer, integer}, boolean, ast.ASTFuncDeclInfo{})
	fpMust(t, err)
	fpMust(t, engine.RegisterPredicate(predicate))
	xName, err := manager.Symbols.Intern("spacer-sliced-answer-x")
	fpMust(t, err)
	x, err := manager.MkConst(xName, integer)
	fpMust(t, err)
	fpMust(t, engine.RegisterVariable(x))
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	fpMust(t, err)
	pXZero, err := manager.MkApp(predicate, []ast.ASTHandle{x, zero})
	fpMust(t, err)
	pXZeroRef, err := manager.NewRef(pXZero)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, manager.DelRef(&pXZeroRef)) })
	fpMust(t, engine.AddRule(pXZero, "sliced-answer-fact"))
	status, err := engine.QueryPredicate(predicate)
	fpMust(t, err)
	if status != lbool.True {
		t.Fatal("Spacer sliced query", status)
	}
	answer, err := engine.Reachable(predicate)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, manager.DelRef(&answer)) })
	formula, err := manager.RefGet(answer)
	fpMust(t, err)
	used, err := ast.NewASTUsedVars(manager)
	fpMust(t, err)
	fpMust(t, used.Process(formula))
	if used.Contains(0) != (ast.ASTHandle{}) || used.Contains(1) == (ast.ASTHandle{}) {
		t.Fatalf("Spacer reachable formula did not restore the original second-column variable: max=%d v0=%v v1=%v formula=%v", used.MaxFoundVarIndexPlusOne(), used.Contains(0), used.Contains(1), formula)
	}
}

func TestPDREngineReadsContextAndCacheParameters(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	globals, err := context.GlobalParameters()
	fpMust(t, err)
	fpMust(t, globals.SourceSetName("fixedpoint.pdr.max_num_contexts", "1"))
	fpMust(t, globals.SourceSetName("fixedpoint.pdr.cache_mode", "3"))
	engine, err := fp.NewPropositionalEngineWithServices(manager, globals.Context, globals)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	fpMust(t, engine.SelectEngine("pdr"))
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	name, err := manager.Symbols.Intern("pdr-invalid-cache")
	fpMust(t, err)
	declaration, err := manager.MkFuncDecl(name, nil, boolean, ast.ASTFuncDeclInfo{})
	fpMust(t, err)
	fpMust(t, engine.RegisterPredicate(declaration))
	fact, err := manager.MkApp(declaration, nil)
	fpMust(t, err)
	fpMust(t, engine.AddRule(fact, "invalid-cache-fact"))
	if _, err = engine.QueryPredicate(declaration); err == nil || !strings.Contains(err.Error(), "invalid pdr.cache_mode 3") {
		t.Fatal("PDR cache parameter was not validated", err)
	}
}

func TestPropositionalEngineCompilesSimpleJoinNormalization(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngine(manager)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	applications := make([]ast.ASTHandle, 4)
	for index, name := range []string{"A", "B", "C", "Goal"} {
		symbol, x := manager.Symbols.Intern("simple-join-" + name)
		fpMust(t, x)
		declaration, x := manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, x)
		applications[index], x = manager.MkApp(declaration, nil)
		fpMust(t, x)
		fpMust(t, engine.RegisterPredicate(declaration))
	}
	for index := 0; index < 3; index++ {
		fpMust(t, engine.AddRule(applications[index], "simple-join-fact"))
	}
	first, err := manager.MkAnd(applications[0], applications[1])
	fpMust(t, err)
	body, err := manager.MkAnd(first, applications[2])
	fpMust(t, err)
	rule, err := manager.MkImplies(body, applications[3])
	fpMust(t, err)
	fpMust(t, engine.AddRule(rule, "simple-join-goal"))
	status, err := engine.Query(applications[3])
	fpMust(t, err)
	if status != lbool.True {
		t.Fatal("three-tail rule was not derived", status)
	}
	compiled, err := engine.LastQueryUsedCompiledCode()
	fpMust(t, err)
	if !compiled {
		t.Fatal("simple-join normalization bypassed relational bytecode")
	}
}

func TestPDREngineCertificateHidesSimpleJoinAuxiliaries(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	globals, err := context.GlobalParameters()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngineWithServices(manager, globals.Context, globals)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	fpMust(t, engine.SelectEngine("pdr"))
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	applications := make([]ast.ASTHandle, 4)
	declarations := make([]ast.ASTHandle, 4)
	for index, name := range []string{"A", "B", "C", "Goal"} {
		symbol, x := manager.Symbols.Intern("certificate-join-" + name)
		fpMust(t, x)
		declarations[index], x = manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, x)
		applications[index], x = manager.MkApp(declarations[index], nil)
		fpMust(t, x)
		fpMust(t, engine.RegisterPredicate(declarations[index]))
	}
	fpMust(t, engine.AddRule(applications[0], "certificate-a"))
	fpMust(t, engine.AddRule(applications[1], "certificate-b"))
	left, err := manager.MkAnd(applications[0], applications[1])
	fpMust(t, err)
	body, err := manager.MkAnd(left, applications[2])
	fpMust(t, err)
	rule, err := manager.MkImplies(body, applications[3])
	fpMust(t, err)
	fpMust(t, engine.AddRule(rule, "certificate-goal"))
	status, err := engine.QueryPredicate(declarations[3])
	fpMust(t, err)
	if status != lbool.False {
		t.Fatal("three-tail unreachable query", status)
	}
	certificate, err := engine.Answer(declarations[3])
	fpMust(t, err)
	handle, err := manager.RefGet(certificate)
	fpMust(t, err)
	printer, err := ast.NewASTPPUtil(manager)
	fpMust(t, err)
	var text strings.Builder
	fpMust(t, printer.CollectAll([]ast.ASTHandle{handle}))
	fpMust(t, printer.DisplayAssertions(&text, []ast.ASTHandle{handle}, true))
	if strings.Contains(text.String(), "simple-join") || !strings.Contains(text.String(), "certificate-join-A") {
		t.Fatalf("converted PDR certificate:\n%s", text.String())
	}
	fpMust(t, manager.DelRef(&certificate))
}

func TestPropositionalEngineExecutesStratifiedNegation(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngine(manager)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	applications := make(map[string]ast.ASTHandle)
	for _, name := range []string{"P", "Q"} {
		symbol, e := manager.Symbols.Intern("fp-refuse-" + name)
		fpMust(t, e)
		declaration, e := manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		fpMust(t, e)
		applications[name], e = manager.MkApp(declaration, nil)
		fpMust(t, e)
		fpMust(t, engine.RegisterPredicate(declaration))
	}
	negated, err := manager.MkNot(applications["P"])
	fpMust(t, err)
	formula, err := manager.MkImplies(negated, applications["Q"])
	fpMust(t, err)
	formulaRef, err := manager.NewRef(formula)
	fpMust(t, err)
	defer func() { fpMust(t, manager.DelRef(&formulaRef)) }()
	fpMust(t, engine.AddRule(formula, "negative"))
	status, err := engine.Query(applications["Q"])
	fpMust(t, err)
	if status != lbool.True {
		t.Fatal("stratified negative Horn rule was not derived", status)
	}
	compiled, err := engine.LastQueryUsedCompiledCode()
	fpMust(t, err)
	if !compiled {
		t.Fatal("stratified finite negation bypassed relational bytecode")
	}
}

func TestPropositionalEngineScopeGuards(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngine(manager)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	if err = engine.Pop(); err == nil || !strings.Contains(err.Error(), "no backtracking points") {
		t.Fatal("scope underflow accepted", err)
	}
	boolean, err := manager.MkBoolSort()
	fpMust(t, err)
	symbol, err := manager.Symbols.Intern("fp-scope-query")
	fpMust(t, err)
	declaration, err := manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
	fpMust(t, err)
	query, err := manager.MkApp(declaration, nil)
	fpMust(t, err)
	fpMust(t, engine.RegisterPredicate(declaration))
	fpMust(t, engine.Push())
	status, err := engine.Query(query)
	fpMust(t, err)
	if status != lbool.False {
		t.Fatal("unexpected empty-program query", status)
	}
	if err = engine.Pop(); err == nil || !strings.Contains(err.Error(), "before a query") {
		t.Fatal("post-query pop accepted", err)
	}
}

func TestPropositionalEngineSelection(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, context.Close()) })
	manager, err := context.Manager()
	fpMust(t, err)
	engine, err := fp.NewPropositionalEngine(manager)
	fpMust(t, err)
	t.Cleanup(func() { fpMust(t, engine.Close()) })
	for _, name := range []string{"auto-config", "datalog"} {
		if err = engine.SelectEngine(name); err != nil {
			t.Fatal(name, err)
		}
	}
	for _, name := range []string{"spacer", "unknown"} {
		if err = engine.SelectEngine(name); err == nil {
			t.Fatal("unavailable engine accepted", name)
		}
	}
}
