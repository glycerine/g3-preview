package aig

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGMaxSharingProcessingAndNullTraceSourceFallthrough(t *testing.T) {
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
	tags.Enable("max_sharing")
	sort, err := m.MkBoolSort()
	aigMust(t, err)
	g, err := NewManager(m)
	aigMust(t, err)
	var vars [3]*Ref
	for i, name := range []string{"a", "b", "c"} {
		h, e := m.MkFreshConst(name, sort)
		aigMust(t, e)
		vars[i], e = g.MakeVariable(h)
		aigMust(t, e)
	}
	ab, err := g.InternAnd(vars[0], vars[1])
	aigMust(t, err)
	abNode := ab.lit.node
	abc, err := g.InternAnd(ab, vars[2])
	aigMust(t, err)
	aigMust(t, ab.Close())
	nestedIndex := 0
	if abc.lit.node.children[1].node == abNode {
		nestedIndex = 1
	}
	rootName := "@" + strconv.FormatUint(uint64(abc.lit.node.id-firstNodeID), 10)
	nestedName := "@" + strconv.FormatUint(uint64(abNode.id-firstNodeID), 10)
	check := func(wantNull string) {
		t.Helper()
		var bodies []string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.HasPrefix(line, "processing ") || strings.HasPrefix(line, "r.is_null(): ") {
				bodies = append(bodies, line)
			}
		}
		if !tags.ShouldTrace("max_sharing") {
			if len(bodies) != 0 || output.Len() != 0 {
				t.Fatalf("erased trace emitted %q", output.String())
			}
			return
		}
		want := []string{
			"processing " + rootName + " idx: 0",
			"processing " + nestedName + " idx: 0",
			"processing " + rootName + " idx: " + strconv.Itoa(nestedIndex+1),
			"r.is_null(): " + wantNull,
		}
		if len(bodies) != len(want) {
			t.Fatalf("source records %q, want %q", bodies, want)
		}
		for i := range want {
			if bodies[i] != want[i] {
				t.Fatalf("source record %d = %q, want %q", i, bodies[i], want[i])
			}
		}
	}
	unchanged, err := g.MaxSharing(abc)
	aigMust(t, err)
	check("1")
	aigMust(t, unchanged.Close())
	output.Reset()
	bc, err := g.InternAnd(vars[1], vars[2])
	aigMust(t, err)
	shared, err := g.MaxSharing(abc)
	aigMust(t, err)
	check("0")
	aigMust(t, shared.Close())
	aigMust(t, bc.Close())
	aigMust(t, abc.Close())
	for _, r := range vars {
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
