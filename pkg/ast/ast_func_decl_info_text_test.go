package ast

import (
	"strconv"
	"testing"
)

// Source: src/ast/ast.cpp:115-130,219-226,283-294. A declaration's attached
// info prints its source parameter types and all nine Boolean flags.
func TestASTFuncDeclInfoSourceText(t *testing.T) {
	m := astTestManager(t)
	bs, err := m.MkBoolSort()
	astTestMust(t, err)
	plain, err := m.MkFuncDecl(astTestSymbol(t, m, "plain-info"), nil, bs, ASTFuncDeclInfo{})
	astTestMust(t, err)
	if body, present, err := m.FuncDeclInfoText(plain); err != nil || present || body != "" {
		t.Fatalf("plain declaration metadata: %q %t %v", body, present, err)
	}
	integer, err := m.NewParameterInt(-7)
	astTestMust(t, err)
	symbol, err := m.NewParameterSymbol(astTestSymbol(t, m, "meta-symbol"))
	astTestMust(t, err)
	rationalValue, err := m.numerals.NewValue()
	astTestMust(t, err)
	astTestMust(t, m.numerals.SetString(&rationalValue, "7/3"))
	rational, err := m.NewParameterRational(&rationalValue)
	astTestMust(t, err)
	astTestMust(t, m.numerals.Del(&rationalValue))
	node, err := m.NewParameterAST(bs)
	astTestMust(t, err)
	double, err := m.NewParameterDouble(1.25)
	astTestMust(t, err)
	external, err := m.NewParameterExternal(9)
	astTestMust(t, err)
	parameters := []ASTParameter{integer, symbol, rational, node, double, external}
	defer func() {
		for i := range parameters {
			astTestMust(t, m.DelParameter(&parameters[i]))
		}
	}()
	flags := ASTFuncFlags{LeftAssociative: true, Commutative: true, Skolem: true}
	info, err := m.NewFuncDeclInfo(NullFamilyID, 17, parameters, flags, false)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelFuncDeclInfo(&info)) }()
	d, err := m.MkFuncDecl(astTestSymbol(t, m, "metadata-info"), []ASTHandle{bs, bs}, bs, info)
	astTestMust(t, err)
	body, present, err := m.FuncDeclInfoText(d)
	astTestMust(t, err)
	if !present {
		t.Fatal("attached declaration info missing")
	}
	id, err := m.GetASTID(bs)
	astTestMust(t, err)
	want := ":fid -1 :decl-kind 17 :parameters (-7 meta-symbol 7/3 #" + strconv.FormatUint(uint64(id), 10) + " 1.25 @9)" +
		" :left-assoc 1 :right-assoc 0 :flat-associative 0 :commutative 1" +
		" :chainable 0 :pairwise 0 :injective 0 :idempotent 0 :skolem 1"
	if body != want {
		t.Fatalf("source declaration info: got %q, want %q", body, want)
	}
}
