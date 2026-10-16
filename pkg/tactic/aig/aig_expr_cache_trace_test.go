package aig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGExprCacheReleaseTracePrecedesLastReference(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	symbols, err := symbol.NewManager(symbol.Config{Budget: budget})
	aigMust(t, err)
	numbers, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	aigMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numbers, Budget: budget, TraceTags: tags})
	aigMust(t, err)
	closeParams := aigTestBindSMT2Params(t, symbols, numbers, budget)
	tags.Enable("expr2aig")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	p, err := m.MkFreshConst("p", sort)
	aigMust(t, err)
	pPin, err := m.NewRef(p)
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	r, err := g.MkAIG(p)
	aigMust(t, err)
	if tags.ShouldTrace("expr2aig") {
		want := "-------- [expr2aig] cache_result src/tactic/aig/aig.cpp:452 ---------\n" +
			"caching:\np!0\n---> #1\n" +
			"------------------------------------------------\n" +
			"-------- [expr2aig] ~expr2aig src/tactic/aig/aig.cpp:439 ---------\n" +
			"dec-ref: #1 ref-count: 2\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("cache release record:\ngot %q\nwant %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace emitted %q", output.String())
	}
	aigMust(t, r.Close())
	output.Reset()
	q, err := m.MkFreshConst("q", sort)
	aigMust(t, err)
	shared, err := m.MkAnd(p, q)
	aigMust(t, err)
	root, err := m.MkOr(shared, shared)
	aigMust(t, err)
	r, err = g.MkAIG(root)
	aigMust(t, err)
	if tags.ShouldTrace("expr2aig") {
		if got := output.String(); !strings.Contains(got, "caching:\n(and p!0 q!1)\n---> @") {
			t.Fatalf("shared compound cache result missing: %q", got)
		}
	}
	aigMust(t, r.Close())
	output.Reset()
	binder, err := symbols.Intern("x")
	aigMust(t, err)
	bound, err := m.MkVar(0, sort)
	aigMust(t, err)
	boundPin, err := m.NewRef(bound)
	aigMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{binder}, Body: bound})
	aigMust(t, err)
	r, err = g.MkAIG(quantifier)
	aigMust(t, err)
	if tags.ShouldTrace("expr2aig") && !strings.Contains(output.String(), "caching:\n(forall ((x Bool)) x)\n---> #") {
		t.Fatalf("quantifier variable cache result missing: %q", output.String())
	}
	aigMust(t, r.Close())
	output.Reset()
	numericQuantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: bound})
	aigMust(t, err)
	r, err = g.MkAIG(numericQuantifier)
	aigMust(t, err)
	if tags.ShouldTrace("expr2aig") && !strings.Contains(output.String(), "caching:\n(forall ((x!1 Bool)) x!1)\n---> #") {
		t.Fatalf("numeric-binder quantifier cache result missing: %q", output.String())
	}
	aigMust(t, r.Close())
	aigMust(t, g.Close())
	aigMust(t, m.DelRef(&boundPin))
	if strings.Contains(output.String(), "aig_lit_count") {
		t.Fatal("unenabled deletion tag emitted")
	}
	aigMust(t, m.DelRef(&pPin))
	aigMust(t, m.Close())
	closeParams()
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
