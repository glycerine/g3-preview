package spacer

// Source: src/muz/spacer/spacer_util.cpp expand_literals arithmetic branch.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
)

func TestExpandInductiveArithmeticEquality(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	owner, err := NewManager(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := owner.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	if err != nil {
		t.Fatal(err)
	}
	integer, err := manager.GetSort(zero)
	if err != nil {
		t.Fatal(err)
	}
	name, err := manager.Symbols.Intern("expand-inductive-equality")
	if err != nil {
		t.Fatal(err)
	}
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := manager.MkFuncDecl(name, []ast.ASTHandle{integer}, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	transformer, err := NewRulePredicateTransformer(owner, predicate, PropSolverConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := transformer.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	xName, err := manager.Symbols.Intern("expand-x")
	if err != nil {
		t.Fatal(err)
	}
	x, err := manager.MkConst(xName, integer)
	if err != nil {
		t.Fatal(err)
	}
	yName, err := manager.Symbols.Intern("expand-y")
	if err != nil {
		t.Fatal(err)
	}
	y, err := manager.MkConst(yName, integer)
	if err != nil {
		t.Fatal(err)
	}
	equality, err := manager.MkEq(x, y)
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := transformer.expandInductiveLiteral(equality)
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 2 {
		t.Fatal("expanded equality size", len(expanded))
	}
	for index, expected := range []int32{2, 3} {
		kind, kindErr := manager.GetDeclKind(expanded[index])
		if kindErr != nil || kind != expected {
			t.Fatal("expanded equality operator", index, kind, kindErr)
		}
	}
}
