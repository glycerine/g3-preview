package rel_test

// Source: src/muz/rel/dl_compiler.cpp retained finite ground and identity-rule
// compilation path.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/rel"
)

func compilerDeclaration(t *testing.T, manager *ast.ASTManager, name string, domain []ast.ASTHandle, boolean ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	symbol, err := manager.Symbols.Intern(name)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := manager.MkFuncDecl(symbol, domain, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	return declaration
}

func compilerRule(t *testing.T, rules *base.RuleManager, set *base.RuleSet, head ast.ASTHandle, tail []ast.ASTHandle, name string) {
	t.Helper()
	rule, err := rules.NewRule(head, tail, nil, name)
	if err != nil {
		t.Fatal(err)
	}
	if err = set.Add(rule); err != nil {
		t.Fatal(err)
	}
	if err = rule.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFiniteCompilerOrdersFactsAndIdentityRules(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Close() })
	manager, err := command.Manager()
	if err != nil {
		t.Fatal(err)
	}
	relations, err := rel.NewRelationManager(manager)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = relations.Close() })
	rules, err := base.NewRuleManager(manager)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rules.Close() })
	set, err := base.NewRuleSet(rules)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = set.Close() })
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	p := compilerDeclaration(t, manager, "compiled-P", []ast.ASTHandle{boolean, boolean}, boolean)
	q := compilerDeclaration(t, manager, "compiled-Q", []ast.ASTHandle{boolean, boolean}, boolean)
	z := compilerDeclaration(t, manager, "compiled-Z", nil, boolean)
	for _, predicate := range []ast.ASTHandle{p, q, z} {
		if err = rules.RegisterPredicate(predicate); err != nil {
			t.Fatal(err)
		}
	}
	variable0, err := manager.MkVar(0, boolean)
	if err != nil {
		t.Fatal(err)
	}
	variable1, err := manager.MkVar(1, boolean)
	if err != nil {
		t.Fatal(err)
	}
	pVariables, err := manager.MkApp(p, []ast.ASTHandle{variable0, variable1})
	if err != nil {
		t.Fatal(err)
	}
	qVariables, err := manager.MkApp(q, []ast.ASTHandle{variable0, variable1})
	if err != nil {
		t.Fatal(err)
	}
	compilerRule(t, rules, set, qVariables, []ast.ASTHandle{pVariables}, "identity-first")
	truth, err := manager.MkTrue()
	if err != nil {
		t.Fatal(err)
	}
	falsity, err := manager.MkFalse()
	if err != nil {
		t.Fatal(err)
	}
	pFact, err := manager.MkApp(p, []ast.ASTHandle{truth, falsity})
	if err != nil {
		t.Fatal(err)
	}
	compilerRule(t, rules, set, pFact, nil, "fact-after-dependent")
	zFact, err := manager.MkApp(z, nil)
	if err != nil {
		t.Fatal(err)
	}
	compilerRule(t, rules, set, zFact, nil, "zero-arity-fact")
	valid, err := set.Finalize()
	if err != nil || !valid {
		t.Fatal("finalize", valid, err)
	}
	execution, termination, err := rel.CompileFiniteRules(relations, set)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = execution.Close(); _ = termination.Close() })
	context, err := rel.NewExecutionContextWithRelations(relations, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = context.Close() })
	complete, err := execution.Perform(context)
	if err != nil || !complete {
		t.Fatal("compiled execution", complete, err)
	}
	complete, err = termination.Perform(context)
	if err != nil || !complete {
		t.Fatal("compiled termination", complete, err)
	}
	for _, predicate := range []ast.ASTHandle{p, q} {
		table, found, x := relations.TryGetRelation(predicate)
		if x != nil || !found || table == nil {
			t.Fatal("compiled predicate relation", predicate, table, found, x)
		}
		facts, x := table.Facts()
		if x != nil || len(facts) != 1 || len(facts[0]) != 2 || facts[0][0] != 1 || facts[0][1] != 0 {
			t.Fatal("compiled facts", predicate, facts, x)
		}
	}
	zTable, found, err := relations.TryGetRelation(z)
	if err != nil || !found || zTable == nil {
		t.Fatal("zero-arity compiled relation", zTable, found, err)
	}
	zFacts, err := zTable.Facts()
	if err != nil || len(zFacts) != 1 || len(zFacts[0]) != 0 {
		t.Fatal("zero-arity compiled fact", zFacts, err)
	}
}

func TestFiniteCompilerRejectsRicherRulesWithoutPartialExecution(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Close() })
	manager, err := command.Manager()
	if err != nil {
		t.Fatal(err)
	}
	relations, err := rel.NewRelationManager(manager)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = relations.Close() })
	rules, err := base.NewRuleManager(manager)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rules.Close() })
	set, err := base.NewRuleSet(rules)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = set.Close() })
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	p := compilerDeclaration(t, manager, "unsupported-P", []ast.ASTHandle{boolean}, boolean)
	if err = rules.RegisterPredicate(p); err != nil {
		t.Fatal(err)
	}
	variable, err := manager.MkVar(0, boolean)
	if err != nil {
		t.Fatal(err)
	}
	head, err := manager.MkApp(p, []ast.ASTHandle{variable})
	if err != nil {
		t.Fatal(err)
	}
	compilerRule(t, rules, set, head, nil, "unbound-head")
	if _, _, err = rel.CompileFiniteRules(relations, set); !errors.Is(err, rel.ErrCompilerUnsupported) {
		t.Fatal("unsupported rule was not rejected", err)
	}
}
