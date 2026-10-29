package transforms_test

// Source: src/muz/transforms/dl_mk_coalesce.cpp shared-body disjunction.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func TestRuleCoalesceMergesSharedBodyConstraints(t *testing.T) {
	f := newConeFixture(t)
	boolean, err := f.manager.MkBoolSort()
	coneMust(t, err)
	zero, err := f.manager.ArithmeticNumeralRat(big.NewRat(0, 1), true)
	coneMust(t, err)
	one, err := f.manager.ArithmeticNumeralRat(big.NewRat(1, 1), true)
	coneMust(t, err)
	two, err := f.manager.ArithmeticNumeralRat(big.NewRat(2, 1), true)
	coneMust(t, err)
	integer, err := f.manager.GetSort(zero)
	coneMust(t, err)
	declare := func(name string) ast.ASTHandle {
		symbol, makeErr := f.manager.Symbols.Intern("coalesce-" + name)
		coneMust(t, makeErr)
		declaration, makeErr := f.manager.MkFuncDecl(symbol, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
		coneMust(t, makeErr)
		coneMust(t, f.rules.RegisterPredicate(declaration))
		return declaration
	}
	p := declare("P")
	q := declare("Q")
	variable, err := f.manager.MkVar(0, integer)
	coneMust(t, err)
	pVariable, err := f.manager.MkApp(p, []ast.ASTHandle{variable})
	coneMust(t, err)
	qVariable, err := f.manager.MkApp(q, []ast.ASTHandle{variable})
	coneMust(t, err)
	for _, value := range []ast.ASTHandle{zero, one, two} {
		equality, makeErr := f.manager.MkEq(variable, value)
		coneMust(t, makeErr)
		rule, makeErr := f.rules.NewRule(pVariable, []ast.ASTHandle{qVariable, equality}, nil, "branch")
		coneMust(t, makeErr)
		coneMust(t, f.set.Add(rule))
		coneMust(t, rule.Close())
	}
	coneMust(t, f.set.SetOutputPredicate(p))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("coalesce fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(transforms.NewRuleCoalescePlugin(f.manager, f.rules, 0)))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("coalesce plugin returned no replacement")
	}
	count, err := f.set.Len()
	coneMust(t, err)
	if count != 1 {
		t.Fatalf("coalesced rule count %d, want 1", count)
	}
	rule, err := f.set.Rule(0)
	coneMust(t, err)
	defer func() { coneMust(t, rule.Close()) }()
	uninterpreted, err := rule.UninterpretedTailSize()
	coneMust(t, err)
	total, err := rule.TailSize()
	coneMust(t, err)
	formula, _, err := rule.Tail(1)
	coneMust(t, err)
	operation, err := f.manager.GetDeclKind(formula)
	coneMust(t, err)
	if uninterpreted != 1 || total != 2 || operation != ast.ASTOpOr {
		t.Fatal("coalesced body shape", uninterpreted, total, operation)
	}
}
