package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTModelValueDependenciesAndProcedures(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	value, err := m.MkTrue()
	astTestMust(t, err)
	fresh, err := NewSMTExtraFreshValue(sort, 7)
	astTestMust(t, err)
	if fresh.Sort() != sort || fresh.Index() != 7 {
		t.Fatal("fresh metadata", fresh.Sort(), fresh.Index())
	}
	if _, err = fresh.Value(); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("unset fresh value", err)
	}
	astTestMust(t, fresh.SetValue(value))
	if got, getErr := fresh.Value(); getErr != nil || got != value {
		t.Fatal("fresh value", got, getErr)
	}
	if err = fresh.SetValue(value); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("fresh value reset", err)
	}

	wrapper, err := NewSMTExpressionValueProcedure(value)
	astTestMust(t, err)
	generator, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	if wrapper.IsFresh() || len(wrapper.Dependencies()) != 0 {
		t.Fatal("expression procedure metadata")
	}
	if got, makeErr := wrapper.MakeValue(generator, nil); makeErr != nil || got != value {
		t.Fatal("expression procedure", got, makeErr)
	}

	freshProcedure, err := NewSMTFreshValueProcedure(fresh)
	astTestMust(t, err)
	dependencies := freshProcedure.Dependencies()
	if !freshProcedure.IsFresh() || len(dependencies) != 1 || !dependencies[0].IsFreshValue() {
		t.Fatal("fresh procedure metadata", dependencies)
	}
	if got, makeErr := freshProcedure.MakeValue(generator, []ast.ASTHandle{value}); makeErr != nil || got != value {
		t.Fatal("fresh procedure", got, makeErr)
	}
	if _, makeErr := freshProcedure.MakeValue(generator, nil); !errors.Is(makeErr, ast.ErrASTDomain) {
		t.Fatal("fresh procedure arity", makeErr)
	}
}

func TestSMTModelValueEnodeDependencyCanonicalizesRoot(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	left, err := m.MkConst(astTestSymbol(t, m, "model-dependency-left"), sort)
	astTestMust(t, err)
	right, err := m.MkConst(astTestSymbol(t, m, "model-dependency-right"), sort)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	leftNode, err := state.MkEnode(left, false, false, true)
	astTestMust(t, err)
	rightNode, err := state.MkEnode(right, false, false, true)
	astTestMust(t, err)
	rightNode.root = leftNode
	dependency, err := NewSMTEnodeValueDependency(rightNode)
	astTestMust(t, err)
	if dependency.IsFreshValue() {
		t.Fatal("enode dependency tagged fresh")
	}
	if got, getErr := dependency.Enode(); getErr != nil || got != leftNode {
		t.Fatal("canonical enode dependency", got, getErr)
	}
	rightNode.root = rightNode
}
