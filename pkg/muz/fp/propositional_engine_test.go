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
	if compiled {
		t.Fatal("unsupported negated rule was partially compiled")
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
