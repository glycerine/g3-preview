package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTInternalizationOrderBooleanGateContextsAndNotElision(t *testing.T) {
	m := astTestManager(t)
	a := smtRelevancyBool(t, m, "order-a")
	b := smtRelevancyBool(t, m, "order-b")
	c := smtRelevancyBool(t, m, "order-c")
	and, err := m.MkAnd(a, b)
	astTestMust(t, err)
	not, err := m.MkNot(c)
	astTestMust(t, err)
	root, err := m.MkOr(and, not)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	order, err := NewSMTInternalizationOrder(m, context)
	astTestMust(t, err)
	got, err := order.Sort(root)
	astTestMust(t, err)
	want := []ast.ASTHandle{a, b, and, c}
	if len(got) != len(want) {
		t.Fatal("order length", got)
	}
	for index := range want {
		if got[index].Expression != want[index] || !got[index].GateContext {
			t.Fatal("Boolean topological order", index, got[index], want[index])
		}
	}
}

func TestSMTInternalizationOrderTermITEContextsAndInternalizedCutoff(t *testing.T) {
	m := astTestManager(t)
	condition := smtRelevancyBool(t, m, "order-condition")
	sort, err := m.MkSort(astTestSymbol(t, m, "OrderSort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	thenTerm, thenNode := smtEnodeConstant(t, m, 2220, sort, 0)
	elseTerm, _ := smtEnodeConstant(t, m, 2221, sort, 0)
	root, err := m.MkIte(condition, thenTerm, elseTerm)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	astTestMust(t, context.RegisterEnode(thenNode, false))
	order, err := NewSMTInternalizationOrder(m, context)
	astTestMust(t, err)
	got, err := order.Sort(root)
	astTestMust(t, err)
	if len(got) != 3 || got[0].Expression != elseTerm || got[0].GateContext || got[1].Expression != thenTerm || got[1].GateContext || got[2].Expression != condition || !got[2].GateContext {
		t.Fatal("term ITE source order and contexts", got)
	}
}
