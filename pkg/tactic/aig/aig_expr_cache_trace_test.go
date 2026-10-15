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
	tags.Enable("expr2aig")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	p, err := m.MkFreshConst("p", sort)
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	r, err := g.MkAIG(p)
	aigMust(t, err)
	if tags.ShouldTrace("expr2aig") {
		want := "-------- [expr2aig] ~expr2aig src/tactic/aig/aig.cpp:439 ---------\n" +
			"dec-ref: #1 ref-count: 2\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("cache release record:\ngot %q\nwant %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace emitted %q", output.String())
	}
	aigMust(t, r.Close())
	aigMust(t, g.Close())
	if strings.Contains(output.String(), "aig_lit_count") {
		t.Fatal("unenabled deletion tag emitted")
	}
	aigMust(t, m.Close())
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
