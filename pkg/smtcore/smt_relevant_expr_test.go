package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTCheckAtLabelsSourcePolarityPaths(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "at-a"), boolean)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "at-b"), boolean)
	astTestMust(t, err)
	at := astTestSymbol(t, m, "path@label")
	plain := astTestSymbol(t, m, "plain-label")
	positiveA, err := m.MkLabel(true, []symbol.Symbol{at, plain}, a)
	astTestMust(t, err)
	positiveB, err := m.MkLabel(true, []symbol.Symbol{at}, b)
	astTestMust(t, err)
	conjunction, err := m.MkAnd(positiveA, positiveB)
	astTestMust(t, err)
	disjunction, err := m.MkOr(positiveA, positiveB)
	astTestMust(t, err)
	if good, err := SMTCheckAtLabels(m, conjunction); err != nil || good {
		t.Fatal("positive conjunction must sum @ labels", good, err)
	}
	if good, err := SMTCheckAtLabels(m, disjunction); err != nil || !good {
		t.Fatal("positive disjunction must take the maximum path", good, err)
	}
	negative, err := m.MkLabel(false, []symbol.Symbol{at}, b)
	astTestMust(t, err)
	notNegative, err := m.MkNot(negative)
	astTestMust(t, err)
	twoOnPath, err := m.MkAnd(positiveA, notNegative)
	astTestMust(t, err)
	if good, err := SMTCheckAtLabels(m, twoOnPath); err != nil || good {
		t.Fatal("not must reverse label polarity", good, err)
	}
	literal, err := m.MkLabelLit([]symbol.Symbol{at, at})
	astTestMust(t, err)
	if good, err := SMTCheckAtLabels(m, literal); err != nil || good {
		t.Fatal("each @ label-literal symbol contributes", good, err)
	}
}

func TestSMTRelevantExprTraversalChoosesJustifyingBranchesAndCaches(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "rel-a"), boolean)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "rel-b"), boolean)
	astTestMust(t, err)
	c, err := m.MkConst(astTestSymbol(t, m, "rel-c"), boolean)
	astTestMust(t, err)
	or, err := m.MkOr(a, b)
	astTestMust(t, err)
	ite, err := m.MkIte(c, a, b)
	astTestMust(t, err)
	root, err := m.MkAnd(or, ite)
	astTestMust(t, err)

	context, err := NewSMTRelevantExprContext(m)
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{a, b, c, or, ite, root} {
		astTestMust(t, context.SetRelevant(expression, true))
	}
	for expression, value := range map[ast.ASTHandle]lbool.LBool{
		a: lbool.True, b: lbool.False, c: lbool.True, or: lbool.True, ite: lbool.True, root: lbool.True,
	} {
		astTestMust(t, context.SetAssignment(expression, value))
	}
	var visited []ast.ASTHandle
	astTestMust(t, context.Process(root, func(expression ast.ASTHandle) error {
		visited = append(visited, expression)
		return nil
	}))
	want := []ast.ASTHandle{root, ite, a, c, or}
	if len(visited) != len(want) {
		t.Fatal("visited expression count", visited)
	}
	for i := range want {
		if visited[i] != want[i] {
			t.Fatal("source stack and branch order", i, visited, want)
		}
	}
	astTestMust(t, context.Process(root, func(ast.ASTHandle) error {
		t.Fatal("cached root was revisited")
		return nil
	}))
	context.Reset()
	astTestMust(t, context.SetRelevant(c, false))
	if err := context.Process(root, func(ast.ASTHandle) error { return nil }); err != ast.ErrASTDomain {
		t.Fatal("relevant child invariant", err)
	}
}

func TestSMTRelevantExprLabelCollectors(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	body, err := m.MkConst(astTestSymbol(t, m, "label-body"), boolean)
	astTestMust(t, err)
	positiveName := astTestSymbol(t, m, "positive")
	negativeName := astTestSymbol(t, m, "negative")
	positive, err := m.MkLabel(true, []symbol.Symbol{positiveName}, body)
	astTestMust(t, err)
	negative, err := m.MkLabel(false, []symbol.Symbol{negativeName}, body)
	astTestMust(t, err)
	root, err := m.MkAnd(positive, negative)
	astTestMust(t, err)

	context, err := NewSMTRelevantExprContext(m)
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{body, positive, negative, root} {
		astTestMust(t, context.SetRelevant(expression, true))
	}
	astTestMust(t, context.SetAssignment(root, lbool.True))
	astTestMust(t, context.SetAssignment(positive, lbool.True))
	astTestMust(t, context.SetAssignment(negative, lbool.False))
	var labels []symbol.Symbol
	astTestMust(t, context.CollectLabels(root, &labels))
	if len(labels) != 2 || labels[0] != negativeName || labels[1] != positiveName {
		t.Fatal("polarity-aware labels", labels)
	}

	literalName := astTestSymbol(t, m, "literal")
	literal, err := m.MkLabelLit([]symbol.Symbol{literalName})
	astTestMust(t, err)
	literalContext, err := NewSMTRelevantExprContext(m)
	astTestMust(t, err)
	astTestMust(t, literalContext.SetRelevant(literal, true))
	astTestMust(t, literalContext.SetAssignment(literal, lbool.True))
	var literals []symbol.Symbol
	astTestMust(t, literalContext.CollectLabelLiterals(literal, &literals))
	if len(literals) != 1 || literals[0] != literalName {
		t.Fatal("true relevant label literal", literals)
	}
}
