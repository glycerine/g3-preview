package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func smtModelTraversalFixture(t *testing.T, count int) (*ast.ASTManager, *SMTModelGenerator, []*SMTEnode, ast.ASTHandle) {
	t.Helper()
	m := astTestManager(t)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "traversal-sort"), nil)
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, m, nil)
	nodes := make([]*SMTEnode, count)
	for index := range nodes {
		expression, err := m.MkConst(astTestSymbol(t, m, fmt.Sprint("traversal-node-", index)), sort)
		astTestMust(t, err)
		nodes[index], err = state.MkEnode(expression, false, false, true)
		astTestMust(t, err)
	}
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return m, g, nodes, sort
}

func TestSMTModelTraversalUsesNativeDependencyStackOrder(t *testing.T) {
	_, g, nodes, _ := smtModelTraversalFixture(t, 3)
	first, err := NewSMTEnodeValueDependency(nodes[1])
	astTestMust(t, err)
	last, err := NewSMTEnodeValueDependency(nodes[2])
	astTestMust(t, err)
	procedures := map[*SMTEnode]SMTModelValueProcedure{
		nodes[0]: &smtModelDependencyProcedure{dependencies: []SMTModelValueDependency{first, last}},
		nodes[1]: &smtModelDependencyProcedure{},
		nodes[2]: &smtModelDependencyProcedure{},
	}
	ordered, err := g.topSortSources(nodes, procedures)
	astTestMust(t, err)
	if len(ordered) != 3 {
		t.Fatal("source count", len(ordered))
	}
	for index, expected := range []*SMTEnode{nodes[2], nodes[1], nodes[0]} {
		actual, err := ordered[index].Enode()
		astTestMust(t, err)
		if actual != expected {
			t.Fatal("native last-pushed dependency order", index, actual.Owner(), expected.Owner())
		}
	}
}

func TestSMTModelTraversalSharesImplicitSortDependencies(t *testing.T) {
	_, g, nodes, sort := smtModelTraversalFixture(t, 1)
	outer, err := g.MkExtraFreshValue(sort)
	astTestMust(t, err)
	inner, err := g.MkExtraFreshValue(sort)
	astTestMust(t, err)
	dependency, err := NewSMTFreshValueDependency(inner)
	astTestMust(t, err)
	procedures := map[*SMTEnode]SMTModelValueProcedure{
		nodes[0]: &smtModelDependencyProcedure{dependencies: []SMTModelValueDependency{dependency}},
	}
	// The first fresh source traverses nonfresh roots of this sort. The
	// second must reuse that traversal, so root -> inner is not expanded
	// into a spurious second implicit inner -> root dependency.
	ordered, err := g.topSortSources(nodes, procedures)
	astTestMust(t, err)
	if len(ordered) != 3 {
		t.Fatal("source count", len(ordered))
	}
	first, err := ordered[0].FreshValue()
	astTestMust(t, err)
	second, err := ordered[1].Enode()
	astTestMust(t, err)
	third, err := ordered[2].FreshValue()
	astTestMust(t, err)
	if first != inner || second != nodes[0] || third != outer {
		t.Fatal("shared implicit sort traversal order", first, second, third)
	}
}

type smtModelTraversalCountingProcedure struct {
	smtModelDependencyProcedure
	calls int
}

func (p *smtModelTraversalCountingProcedure) Dependencies() []SMTModelValueDependency {
	p.calls++
	return p.smtModelDependencyProcedure.Dependencies()
}

func TestSMTModelTraversalRechecksDependenciesOnlyInSourceDebug(t *testing.T) {
	_, g, nodes, _ := smtModelTraversalFixture(t, 1)
	procedure := &smtModelTraversalCountingProcedure{}
	_, err := g.topSortSources(nodes, map[*SMTEnode]SMTModelValueProcedure{nodes[0]: procedure})
	astTestMust(t, err)
	want := 1
	if ast.ASTSourceDebug {
		// Native process_source calls visit_children again inside SASSERT.
		want = 2
	}
	if procedure.calls != want {
		t.Fatal("dependency queries", procedure.calls, "want", want)
	}
}
