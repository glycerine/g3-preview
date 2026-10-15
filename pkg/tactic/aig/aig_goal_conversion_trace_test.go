package aig

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGGoalConversionCacheAndAuxTraceSourceOrder(t *testing.T) {
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
	tags.Enable("aig2expr")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	var variables [3]*Ref
	g, err := NewManager(m)
	aigMust(t, err)
	for i, name := range []string{"p", "q", "r"} {
		h, e := m.MkFreshConst(name, sort)
		aigMust(t, e)
		variables[i], e = g.MakeVariable(h)
		aigMust(t, e)
	}
	ab, err := g.InternAnd(variables[0], variables[1])
	aigMust(t, err)
	abc, err := g.InternAnd(ab, variables[2])
	aigMust(t, err)
	auxID := ab.lit.node.id - firstNodeID
	rootID := abc.lit.node.id - firstNodeID
	aigMust(t, ab.Close())
	negated, err := abc.Not()
	aigMust(t, err)
	aigMust(t, abc.Close())
	goal, err := tactic.NewGoal(m, false, false, false)
	aigMust(t, err)
	aigMust(t, g.ToGoal(negated, goal))
	if tags.ShouldTrace("aig2expr") {
		want := "-------- [aig2expr] aig2expr src/tactic/aig/aig.cpp:931 ---------\n" +
			"skipping aux AND @" + strconv.FormatUint(uint64(auxID), 10) + "\n" +
			"------------------------------------------------\n" +
			"-------- [aig2expr] aig2expr src/tactic/aig/aig.cpp:863 ---------\n" +
			"caching AND @" + strconv.FormatUint(uint64(rootID), 10) + "\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("aux/cache trace:\ngot %q\nwant %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace emitted %q", output.String())
	}
	output.Reset()
	ite, err := g.Ite(variables[0], variables[1], variables[2])
	aigMust(t, err)
	iteID := ite.lit.node.id - firstNodeID
	aigMust(t, g.ToGoal(ite, goal))
	if tags.ShouldTrace("aig2expr") {
		want := "-------- [aig2expr] aig2expr src/tactic/aig/aig.cpp:881 ---------\n" +
			"caching ITE/IFF @" + strconv.FormatUint(uint64(iteID), 10) + "\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("ITE cache trace:\ngot %q\nwant %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased ITE trace emitted %q", output.String())
	}
	aigMust(t, ite.Close())
	aigMust(t, negated.Close())
	aigMust(t, goal.Close())
	for _, r := range variables {
		aigMust(t, r.Close())
	}
	aigMust(t, g.Close())
	aigMust(t, m.Close())
	aigMust(t, numbers.Close())
	aigMust(t, symbols.Close())
	if budget.Used() != 0 {
		t.Fatalf("owner graph retained %d bytes", budget.Used())
	}
}
