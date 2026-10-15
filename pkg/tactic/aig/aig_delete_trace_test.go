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
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGDeletionTraceSourceCascadeOrderAndCompileGate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			budget := arena.NewBudget(1 << 30)
			s, e := symbol.NewManager(symbol.Config{Budget: budget})
			aigMust(t, e)
			q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
				Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
			}})
			aigMust(t, e)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: budget, TraceTags: tags})
			aigMust(t, e)
			if enabled {
				tags.Enable("aig_lit_count")
			}
			sort, e := m.MkBoolSort()
			aigMust(t, e)
			p, e := m.MkFreshConst("p", sort)
			aigMust(t, e)
			qVar, e := m.MkFreshConst("q", sort)
			aigMust(t, e)
			g, e := NewManager(m)
			aigMust(t, e)
			a, e := g.MakeVariable(p)
			aigMust(t, e)
			b, e := g.MakeVariable(qVar)
			aigMust(t, e)
			pair, e := g.InternAnd(a, b)
			aigMust(t, e)
			c, e := g.MakeVariable(p)
			aigMust(t, e)
			aigMust(t, a.Close())
			aigMust(t, b.Close())
			aigMust(t, pair.Close())
			oldRef := c.lit.node.variable
			c.lit.node.variable = ast.ASTRef{}
			if e := c.Close(); !errors.Is(e, ast.ErrASTHandle) {
				t.Fatalf("stopped node release: %v", e)
			}
			c.lit.node.variable = oldRef
			aigMust(t, g.drain())
			aigMust(t, g.Close())
			before := output.String()
			aigMust(t, g.Close())
			if output.String() != before {
				t.Fatal("completed manager replayed deletion TRACE")
			}
			if !tags.ShouldTrace("aig_lit_count") {
				if output.Len() != 0 {
					t.Fatalf("erased/disabled TRACE emitted %q", output.String())
				}
			} else {
				var want strings.Builder
				for _, ref := range []string{"@0", "#2", "#1", "#3", "#0"} {
					want.WriteString("-------- [aig_lit_count] delete_node src/tactic/aig/aig.cpp:149 ---------\n")
					want.WriteString("deleting: " + ref + "\n")
					want.WriteString("------------------------------------------------\n")
				}
				if got := output.String(); got != want.String() {
					t.Fatalf("source deletion order:\ngot %q\nwant %q", got, want.String())
				}
			}
			aigMust(t, m.Close())
			aigMust(t, q.Close())
			aigMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatalf("owner graph retained %d bytes", budget.Used())
			}
		})
	}
}
