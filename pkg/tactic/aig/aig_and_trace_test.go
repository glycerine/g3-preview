package aig

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGMkAndSourceTraceOriginalOperandsAndResult(t *testing.T) {
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
	quiet, err := g.And(ra, rb)
	aigMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("disabled TRACE emitted %q", output.String())
	}
	tags.Enable("mk_and_bug")
	gate, err := g.And(ra, rb)
	aigMust(t, err)
	if tags.ShouldTrace("mk_and_bug") {
		want := "-------- [mk_and_bug] mk_and src/tactic/aig/aig.cpp:1343 ---------\n" +
			"#1\n#1: a!0\nAND\n#2\n#2: b!1\n-->\n@0\n@0: #1 #2\n#1: a!0\n#2: b!1\n\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("source mk_and trace:\ngot %q\nwant %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased TRACE emitted %q", output.String())
	}
	output.Reset()
	same, err := g.And(ra, ra)
	aigMust(t, err)
	if tags.ShouldTrace("mk_and_bug") && !strings.Contains(output.String(), "#1\n#1: a!0\n\n") {
		t.Fatalf("simplified result display missing: %q", output.String())
	}
	aigMust(t, same.Close())
	aigMust(t, gate.Close())
	aigMust(t, quiet.Close())
	aigMust(t, ra.Close())
	c, err := m.MkFreshConst("c", sort)
	aigMust(t, err)
	rc, err := g.MakeVariable(c)
	aigMust(t, err)
	closeParams()
	before := g.nodes
	stopped, err := g.And(rb, rc)
	if tags.ShouldTrace("mk_and_bug") {
		if !errors.Is(err, uparams.ErrGlobalParamsOwner) || stopped != nil || g.nodes != before {
			t.Fatalf("stopped trace retained gate: result %v, error %v, nodes %d want %d", stopped, err, g.nodes, before)
		}
	} else {
		aigMust(t, err)
		aigMust(t, stopped.Close())
	}
	aigMust(t, rc.Close())
	aigMust(t, rb.Close())
	aigMust(t, g.Close())
	aigMust(t, m.Close())
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
