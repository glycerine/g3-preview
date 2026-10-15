package aig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGDisplaySourceBreadthFirstAndInvertedRoot(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	symbols, err := symbol.NewManager(symbol.Config{Budget: budget})
	aigMust(t, err)
	numbers, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	aigMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numbers, Budget: budget})
	aigMust(t, err)
	closeParams := aigTestBindSMT2Params(t, symbols, numbers, budget)
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	a, err := m.MkFreshConst("a", sort)
	aigMust(t, err)
	b, err := m.MkFreshConst("b", sort)
	aigMust(t, err)
	ra, err := g.MakeVariable(a)
	aigMust(t, err)
	rb, err := g.MakeVariable(b)
	aigMust(t, err)
	gate, err := g.InternAnd(ra, rb)
	aigMust(t, err)
	var output bytes.Buffer
	aigMust(t, g.Display(&output, gate))
	want := "@0\n@0: #1 #2\n#1: a!0\n#2: b!1\n"
	if got := output.String(); got != want {
		t.Fatalf("source display %q, want %q", got, want)
	}
	inverted, err := gate.Not()
	aigMust(t, err)
	output.Reset()
	aigMust(t, g.Display(&output, inverted))
	if got := output.String(); got != "-"+want {
		t.Fatalf("inverted source display %q", got)
	}
	c, err := m.MkFreshConst("c", sort)
	aigMust(t, err)
	rc, err := g.MakeVariable(c)
	aigMust(t, err)
	ac, err := g.InternAnd(ra, rc)
	aigMust(t, err)
	shared, err := g.InternAnd(gate, ac)
	aigMust(t, err)
	output.Reset()
	aigMust(t, g.Display(&output, shared))
	got := output.String()
	if strings.Count(got, "#1: a!0\n") != 1 || !strings.Contains(got, "#3: c!2\n") ||
		strings.Index(got, "@0:") < 0 || strings.Index(got, "@1:") < 0 ||
		strings.Index(got, "@0:") > strings.Index(got, "#1:") ||
		strings.Index(got, "@1:") > strings.Index(got, "#1:") {
		t.Fatalf("shared-child breadth-first display %q", got)
	}
	aigMust(t, shared.Close())
	aigMust(t, ac.Close())
	aigMust(t, rc.Close())
	aigMust(t, inverted.Close())
	aigMust(t, gate.Close())
	aigMust(t, rb.Close())
	aigMust(t, ra.Close())
	aigMust(t, g.Close())
	aigMust(t, m.Close())
	closeParams()
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
