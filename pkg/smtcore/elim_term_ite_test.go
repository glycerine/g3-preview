package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/normal_forms"
)

func TestSMTTermITEEliminatorNamesOnlyTermITEAndOwnsDefinitions(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	condition, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	two, err := m.ArithmeticIntegerNumeralText("2")
	astTestMust(t, err)
	termITE, err := m.MkIte(condition, one, two)
	astTestMust(t, err)
	formula, err := m.MkEq(termITE, one)
	astTestMust(t, err)
	names, err := normal_forms.NewDefinedNames(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, names.Close()) })
	eliminator, err := NewSMTTermITEEliminator(m, names)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, eliminator.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	proof, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&proof)); astTestMust(t, m.DelRef(&out)) })
	astTestMust(t, eliminator.Rewrite(formula, out, proof))
	rewritten, err := m.RefGet(out)
	astTestMust(t, err)
	left, err := m.GetArg(rewritten, 0)
	astTestMust(t, err)
	if left == termITE {
		t.Fatal("term ITE was not replaced by its defined name")
	}
	definitions, err := eliminator.NewDefinitions()
	astTestMust(t, err)
	if len(definitions) != 1 {
		t.Fatal("new definition count", len(definitions))
	}
	definition, err := definitions[0].Formula()
	astTestMust(t, err)
	if op, err := m.GetDeclKind(definition); err != nil || op != ast.ASTOpAnd {
		t.Fatal("term ITE defining formula", op, err)
	}
	astTestMust(t, eliminator.Reset())
	definitions, err = eliminator.NewDefinitions()
	astTestMust(t, err)
	if len(definitions) != 0 {
		t.Fatal("reset retained new definitions")
	}
}

func TestSMTTermITEEliminatorLeavesBooleanITE(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	condition, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	left, err := m.MkBoolVal(false)
	astTestMust(t, err)
	booleanITE, err := m.MkIte(condition, left, condition)
	astTestMust(t, err)
	names, err := normal_forms.NewDefinedNames(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, names.Close()) })
	eliminator, err := NewSMTTermITEEliminator(m, names)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, eliminator.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	proof, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&proof)); astTestMust(t, m.DelRef(&out)) })
	astTestMust(t, eliminator.Rewrite(booleanITE, out, proof))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if result != booleanITE {
		t.Fatal("Boolean ITE was eliminated")
	}
	definitions, err := eliminator.NewDefinitions()
	astTestMust(t, err)
	if len(definitions) != 0 {
		t.Fatal("Boolean ITE created a definition")
	}
}
