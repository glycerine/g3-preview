package ast

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTExpandedDistinctSourceTraceAfterConjunction(t *testing.T) {
	c, g, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(g)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	var args []ASTHandle
	for _, name := range []string{"distinct-a", "distinct-b", "distinct-c"} {
		symbol, e := c.SymbolsField.Intern(name)
		astTestMust(t, e)
		h, e := m.MkConst(symbol, sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		args = append(args, h)
	}
	_, err = m.MkDistinctExpanded(args[:2])
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("two arguments reached expansion TRACE: %q", output.String())
	}
	tags.Enable("distinct")
	result, err := m.MkDistinctExpanded(args)
	astTestMust(t, err)
	astTestPin(t, m, result)
	if !tags.ShouldTrace("distinct") {
		if output.Len() != 0 {
			t.Fatalf("erased distinct TRACE: %q", output.String())
		}
		return
	}
	want := "-------- [distinct] mk_distinct_expanded src/ast/ast.cpp:2521 ---------\n" +
		"expanded distinct:\n" +
		"(and (not (= distinct-a distinct-b))\n" +
		"     (not (= distinct-a distinct-c))\n" +
		"     (not (= distinct-b distinct-c)))\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source expanded distinct TRACE: got %q, want %q", got, want)
	}
}
