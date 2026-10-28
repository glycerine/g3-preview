package base_test

// Source: src/muz/base/dl_rule_set.cpp grouped ownership, dependency
// population, reopen/finalize, and stratified-negation rejection.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
)

type ruleSetFixture struct {
	t       *testing.T
	context *cmd_context.Context
	manager *ast.ASTManager
	rules   *base.RuleManager
	decl    map[string]ast.ASTHandle
	app     map[string]ast.ASTHandle
}

func newRuleSetFixture(t *testing.T, names ...string) *ruleSetFixture {
	t.Helper()
	context, err := cmd_context.NewStandaloneContext()
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, context.Close()) })
	manager, err := context.Manager()
	ruleMust(t, err)
	rules, err := base.NewRuleManager(manager)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, rules.Close()) })
	boolean, err := manager.MkBoolSort()
	ruleMust(t, err)
	f := &ruleSetFixture{t: t, context: context, manager: manager, rules: rules, decl: make(map[string]ast.ASTHandle), app: make(map[string]ast.ASTHandle)}
	for _, name := range names {
		symbol, e := manager.Symbols.Intern("ruleset-" + name)
		ruleMust(t, e)
		f.decl[name], e = manager.MkFuncDecl(symbol, nil, boolean, ast.ASTFuncDeclInfo{})
		ruleMust(t, e)
		f.app[name], e = manager.MkApp(f.decl[name], nil)
		ruleMust(t, e)
		ruleMust(t, rules.RegisterPredicate(f.decl[name]))
	}
	return f
}

func (f *ruleSetFixture) rule(head string, tail []string, negated []bool) *base.Rule {
	f.t.Helper()
	expressions := make([]ast.ASTHandle, len(tail))
	for i, name := range tail {
		expressions[i] = f.app[name]
	}
	rule, err := f.rules.NewRule(f.app[head], expressions, negated, head)
	ruleMust(f.t, err)
	return rule
}

func TestRuleSetFinalizesDependenciesAndStratifiedNegation(t *testing.T) {
	f := newRuleSetFixture(t, "P", "Q", "R")
	set, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, set.Close()) })
	inputs := []*base.Rule{
		f.rule("R", nil, nil),
		f.rule("Q", []string{"R"}, nil),
		f.rule("P", []string{"Q"}, []bool{true}),
	}
	for _, rule := range inputs {
		ruleMust(t, set.Add(rule))
		ruleMust(t, rule.Close())
	}
	if count, err := set.Len(); err != nil || count != 3 {
		t.Fatal("rule count", count, err)
	}
	group, err := set.PredicateRules(f.decl["Q"])
	ruleMust(t, err)
	if len(group) != 1 {
		t.Fatal("grouped rule count", len(group))
	}
	for _, rule := range group {
		ruleMust(t, rule.Close())
	}
	ok, err := set.Finalize()
	ruleMust(t, err)
	if !ok {
		t.Fatal("stratified program rejected")
	}
	dependencies, err := set.Dependencies(f.decl["P"])
	ruleMust(t, err)
	if len(dependencies) != 1 || dependencies[0] != f.decl["Q"] {
		t.Fatal("P dependencies", dependencies)
	}
	for name, want := range map[string]uint32{"R": 0, "Q": 1, "P": 2} {
		got, e := set.PredicateStratum(f.decl[name])
		ruleMust(t, e)
		if got != want {
			t.Fatal("predicate stratum", name, got, want)
		}
	}
	candidate := f.rule("R", nil, nil)
	if err = set.Add(candidate); err == nil {
		t.Fatal("finalized rule set accepted mutation")
	}
	ruleMust(t, candidate.Close())
	ruleMust(t, set.Reopen())
	if finalized, err := set.IsFinalized(); err != nil || finalized {
		t.Fatal("reopen state", finalized, err)
	}
}

func TestRuleSetRejectsNegativeDependencyCycle(t *testing.T) {
	f := newRuleSetFixture(t, "P", "Q")
	set, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, set.Close()) })
	first := f.rule("P", []string{"Q"}, []bool{true})
	second := f.rule("Q", []string{"P"}, nil)
	for _, rule := range []*base.Rule{first, second} {
		ruleMust(t, set.Add(rule))
		ruleMust(t, rule.Close())
	}
	ok, err := set.Finalize()
	ruleMust(t, err)
	if ok {
		t.Fatal("negative dependency cycle accepted")
	}
	if finalized, err := set.IsFinalized(); err != nil || finalized {
		t.Fatal("failed finalization retained derived state", finalized, err)
	}
}

func TestRuleSetPredicateMappingsCopiesAndFoundedPartition(t *testing.T) {
	f := newRuleSetFixture(t, "O", "D", "R", "Q", "L")
	source, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, source.Close()) })
	ruleMust(t, source.SetOutputPredicate(f.decl["O"]))

	mapped, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, mapped.Close()) })
	ruleMust(t, mapped.InheritPredicate(source, f.decl["O"], f.decl["D"]))
	output, err := mapped.OutputPredicate()
	ruleMust(t, err)
	if output != f.decl["D"] {
		t.Fatal("derived output predicate", output, f.decl["D"])
	}
	original, err := mapped.OriginalPredicate(f.decl["D"])
	ruleMust(t, err)
	derived, derivedErr := mapped.DerivedPredicate(f.decl["O"])
	ruleMust(t, derivedErr)
	if original != f.decl["O"] || derived != f.decl["D"] {
		t.Fatal("predicate mapping", original, derived)
	}

	program, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, program.Close()) })
	for _, rule := range []*base.Rule{
		f.rule("R", nil, nil),
		f.rule("Q", []string{"R"}, nil),
		f.rule("L", []string{"L"}, nil),
	} {
		ruleMust(t, program.Add(rule))
		ruleMust(t, rule.Close())
	}
	ruleMust(t, program.InheritPredicates(mapped))
	founded, nonFounded, err := program.SplitFounded()
	ruleMust(t, err)
	if len(founded) != 2 || founded[0] != f.decl["R"] || founded[1] != f.decl["Q"] {
		t.Fatal("founded predicates", founded)
	}
	if len(nonFounded) != 1 || nonFounded[0] != f.decl["L"] {
		t.Fatal("non-founded predicates", nonFounded)
	}

	copySet, err := base.NewRuleSet(f.rules)
	ruleMust(t, err)
	t.Cleanup(func() { ruleMust(t, copySet.Close()) })
	ruleMust(t, copySet.AddRules(program))
	if count, err := copySet.Len(); err != nil || count != 3 {
		t.Fatal("copied rule count", count, err)
	}
	if output, err := copySet.OutputPredicate(); err != nil || output != f.decl["D"] {
		t.Fatal("copied output", output, err)
	}
	ruleMust(t, copySet.Reset())
	if count, err := copySet.Len(); err != nil || count != 0 {
		t.Fatal("reset rule count", count, err)
	}
	if outputs, err := copySet.OutputPredicates(); err != nil || len(outputs) != 0 {
		t.Fatal("reset outputs", outputs, err)
	}
}
