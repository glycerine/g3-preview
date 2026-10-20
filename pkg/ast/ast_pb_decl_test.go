package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTPseudoBooleanOriginalDeclarationsAndNames(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	k, e := m.NewParameterInt(2)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&k)) }()
	for kind, name := range []string{"at-most", "at-least"} {
		d, e := m.MkFamilyFuncDecl(fid, int32(kind), []ASTParameter{k}, []ASTHandle{b, b, b}, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		if got, e := m.GetRange(d); e != nil || got != b {
			t.Fatal("PB Boolean range", kind, got, e)
		}
		s, e := m.GetASTName(d)
		astTestMust(t, e)
		text, e := m.Symbols.String(s)
		astTestMust(t, e)
		if text != name {
			t.Fatal("PB name", kind, text)
		}
	}
	for _, logicText := range []string{"", "QF_FD", "ALL", "QF_LIA"} {
		logic := usymbol.Symbol{}
		if logicText != "" {
			logic = astTestSymbol(t, m, logicText)
		}
		var names ASTBuiltinNames
		astTestMust(t, m.GetPseudoBooleanOpNames(&names, logic))
		want := 0
		if logicText == "" || logicText == "QF_FD" || logicText == "ALL" {
			want = 5
		}
		if len(names.Items()) != want {
			t.Fatal("PB visible names", logicText, len(names.Items()), want)
		}
	}
}

func TestASTPseudoBooleanOriginalParameterValidationAndConversion(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	integer, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&integer)) }()
	astTestMust(t, m.numerals.SetInt64(&integer, 7))
	rational, e := m.NewParameterRational(&integer)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&rational)) }()
	bound, e := m.NewParameterInt(3)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&bound)) }()
	d, e := m.MkFamilyFuncDecl(fid, ASTPBOpLE, []ASTParameter{bound, rational}, []ASTHandle{b}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	r, e := m.requireKind(d, ASTFuncDecl)
	astTestMust(t, e)
	info, e := m.funcInfo(m.declAt(r.slot).info)
	astTestMust(t, e)
	if info.count != 2 {
		t.Fatal("PB parameter count", info.count)
	}
	converted, e := m.infoParameter(info, 1)
	astTestMust(t, e)
	value, e := m.parameter(converted)
	astTestMust(t, e)
	if value.Kind != ASTParameterInt || value.integer != 7 {
		t.Fatal("PB rational integer normalization", value.Kind, value.integer)
	}
	negative, e := m.NewParameterInt(-1)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&negative)) }()
	if _, e = m.MkFamilyFuncDecl(fid, ASTPBOpAtMost, []ASTParameter{negative}, []ASTHandle{b}, ASTHandle{}); e != ErrASTDomain {
		t.Fatal("negative cardinality", e)
	}
	if _, e = m.MkFamilyFuncDecl(fid, ASTPBOpLE, []ASTParameter{bound}, []ASTHandle{b}, ASTHandle{}); e != ErrASTDomain {
		t.Fatal("coefficient arity", e)
	}
}
