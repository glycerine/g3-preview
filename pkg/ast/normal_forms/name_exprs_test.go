package normal_forms

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func nameExprOutputs(t *testing.T, m *ast.ASTManager) (ast.ASTRef, ast.ASTRef) {
	t.Helper()
	r, e := m.NewRef(ast.ASTHandle{})
	definedNamesMust(t, e)
	p, e := m.NewRef(ast.ASTHandle{})
	definedNamesMust(t, e)
	t.Cleanup(func() {
		definedNamesMust(t, m.DelRef(&p))
		definedNamesMust(t, m.DelRef(&r))
	})
	return r, p
}

func releaseNameExprRefs(t *testing.T, m *ast.ASTManager, refs []ast.ASTRef) {
	t.Helper()
	for i := range refs {
		definedNamesMust(t, m.DelRef(&refs[i]))
	}
}

func nameExprBoolConst(t *testing.T, m *ast.ASTManager, name string, sort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkFreshConst(name, sort)
	definedNamesMust(t, e)
	definedNamesPin(t, m, h)
	return h
}

func TestNestedFormulaNamerNamesImmediateBasicChildrenWithProof(t *testing.T) {
	m := definedNamesManager(t, true)
	b, e := m.MkBoolSort()
	definedNamesMust(t, e)
	a := nameExprBoolConst(t, m, "nested-a", b)
	bb := nameExprBoolConst(t, m, "nested-b", b)
	c := nameExprBoolConst(t, m, "nested-c", b)
	left, e := m.MkNot(a)
	definedNamesMust(t, e)
	definedNamesPin(t, m, left)
	right, e := m.MkOr(bb, c)
	definedNamesMust(t, e)
	definedNamesPin(t, m, right)
	root, e := m.MkAnd(left, right)
	definedNamesMust(t, e)
	definedNamesPin(t, m, root)
	names, e := NewDefinedNames(m)
	definedNamesMust(t, e)
	t.Cleanup(func() { definedNamesMust(t, names.Close()) })
	namer, e := NewNestedFormulaNamer(m, names)
	definedNamesMust(t, e)
	t.Cleanup(func() { definedNamesMust(t, namer.Close()) })
	out, proof := nameExprOutputs(t, m)
	var definitions, definitionProofs []ast.ASTRef
	definedNamesMust(t, namer.Rewrite(root, &definitions, &definitionProofs, out, proof))
	t.Cleanup(func() {
		releaseNameExprRefs(t, m, definitionProofs)
		releaseNameExprRefs(t, m, definitions)
	})
	if len(definitions) != 2 || len(definitionProofs) != 2 {
		t.Fatal("nested definitions and proofs", len(definitions), len(definitionProofs))
	}
	result, e := m.RefGet(out)
	definedNamesMust(t, e)
	if op, x := m.GetDeclKind(result); x != nil || op != ast.ASTOpAnd {
		t.Fatal("root operator changed", op, x)
	}
	gotLeft, e := m.GetArg(result, 0)
	definedNamesMust(t, e)
	gotRight, e := m.GetArg(result, 1)
	definedNamesMust(t, e)
	if gotLeft == left || gotRight == right {
		t.Fatal("immediate nested formulas were not named", gotLeft, gotRight)
	}
	p, e := m.RefGet(proof)
	definedNamesMust(t, e)
	if yes, x := m.IsProofKind(p, ast.ASTPrMonotonicity); x != nil || !yes {
		t.Fatal("root congruence proof", yes, x)
	}
	fact, e := m.GetProofFact(p)
	definedNamesMust(t, e)
	if factLeft, x := m.GetArg(fact, 0); x != nil || factLeft != root {
		t.Fatal("proof source", factLeft, x)
	}
	if factRight, x := m.GetArg(fact, 1); x != nil || factRight != result {
		t.Fatal("proof result", factRight, x)
	}
}

func TestQuantifierLabelNamerAndReset(t *testing.T) {
	m := definedNamesManager(t, false)
	b, e := m.MkBoolSort()
	definedNamesMust(t, e)
	v, e := m.MkVar(0, b)
	definedNamesMust(t, e)
	definedNamesPin(t, m, v)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: v})
	definedNamesMust(t, e)
	definedNamesPin(t, m, q)
	base := nameExprBoolConst(t, m, "label-base", b)
	labelName, e := m.Symbols.Intern("named-label")
	definedNamesMust(t, e)
	label, e := m.MkLabel(true, []usymbol.Symbol{labelName}, base)
	definedNamesMust(t, e)
	definedNamesPin(t, m, label)
	root, e := m.MkAnd(q, label)
	definedNamesMust(t, e)
	definedNamesPin(t, m, root)
	names, e := NewDefinedNames(m)
	definedNamesMust(t, e)
	t.Cleanup(func() { definedNamesMust(t, names.Close()) })
	namer, e := NewQuantifierLabelNamer(m, names)
	definedNamesMust(t, e)
	t.Cleanup(func() { definedNamesMust(t, namer.Close()) })
	out, proof := nameExprOutputs(t, m)
	var definitions, definitionProofs []ast.ASTRef
	definedNamesMust(t, namer.Rewrite(root, &definitions, &definitionProofs, out, proof))
	if len(definitions) != 2 || len(definitionProofs) != 0 {
		t.Fatal("quantifier/label definitions", len(definitions), len(definitionProofs))
	}
	releaseNameExprRefs(t, m, definitions)
	definitions = nil
	definedNamesMust(t, namer.Reset())
	definedNamesMust(t, namer.Rewrite(root, &definitions, &definitionProofs, out, proof))
	if len(definitions) != 0 {
		t.Fatal("defined-name cache should suppress duplicate definitions", len(definitions))
	}
}
