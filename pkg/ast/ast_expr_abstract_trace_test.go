package ast

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestExprAbstractSourceTraceBothCallers(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.Close()) }()
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	constant, err := m.MkConst(astTestSymbol(t, m, "abstract-trace-x"), sort)
	astTestMust(t, err)
	pinned, err := m.NewRef(constant)
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&pinned)) }()
	out, err := m.NewRef(ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	a, err := NewExprAbstractor(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, a.Close()) }()
	if err = a.Abstract(0, []ASTHandle{constant}, constant, out); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatal("disabled source trace evaluated", output.String())
	}
	tags.Enable("expr_abstract")
	if err = a.MkForall([]ASTHandle{constant}, constant, out); err != nil {
		t.Fatal(err)
	}
	if !tags.ShouldTrace("expr_abstract") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	first := "-------- [expr_abstract] expr_abstract src/ast/expr_abstract.cpp:113 ---------\n"
	second := "-------- [expr_abstract] mk_quantifier src/ast/expr_abstract.cpp:130 ---------\n"
	got := output.String()
	if strings.Count(got, first) != 1 || strings.Count(got, second) != 1 || strings.Index(got, first) >= strings.Index(got, second) || strings.Count(got, "------------------------------------------------\n") != 2 {
		t.Fatalf("source trace ordering: %q", got)
	}
	if !strings.Contains(got, "abstract-trace-x\n") || !strings.Contains(got, "abstract-trace-x \n") {
		t.Fatalf("source input and bound rendering: %q", got)
	}
	output.Reset()
	if err = a.MkExists(nil, constant, out); err != nil {
		t.Fatal(err)
	}
	got = output.String()
	if strings.Count(got, first) != 1 || strings.Count(got, second) != 1 || !strings.Contains(got, second+"abstract-trace-x\n\nabstract-trace-x\n") {
		t.Fatalf("zero-bound source trace: %q", got)
	}
}
