package normal_forms

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func pullQuantFixture(t *testing.T, m *ast.ASTManager) (ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	b, err := m.MkBoolSort()
	definedNamesMust(t, err)
	x, err := m.MkVar(0, b)
	definedNamesMust(t, err)
	y, err := m.MkVar(1, b)
	definedNamesMust(t, err)
	p, err := m.MkFuncDecl(symbol.Numerical(9910), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	definedNamesMust(t, err)
	r, err := m.MkFuncDecl(symbol.Numerical(9911), []ast.ASTHandle{b, b}, b, ast.ASTFuncDeclInfo{})
	definedNamesMust(t, err)
	px, err := m.MkApp(p, []ast.ASTHandle{x})
	definedNamesMust(t, err)
	ryx, err := m.MkApp(r, []ast.ASTHandle{y, x})
	definedNamesMust(t, err)
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(2)}, Body: ryx, Weight: 2, QID: symbol.Numerical(101)})
	definedNamesMust(t, err)
	body, err := m.MkOr(px, inner)
	definedNamesMust(t, err)
	outer, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(1)}, Body: body, Weight: 9, QID: symbol.Numerical(100)})
	definedNamesMust(t, err)
	definedNamesPin(t, m, outer)
	py, err := m.MkApp(p, []ast.ASTHandle{y})
	definedNamesMust(t, err)
	expectedBody, err := m.MkOr(py, ryx)
	definedNamesMust(t, err)
	expected, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b, b}, Names: []symbol.Symbol{symbol.Numerical(1), symbol.Numerical(2)}, Body: expectedBody, Weight: 2, QID: symbol.Numerical(100)})
	definedNamesMust(t, err)
	definedNamesPin(t, m, expected)
	return outer, expected
}

func TestPullQuantProofTraversalPreservesBindersAndMetadata(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(fmt.Sprint(proofs), func(t *testing.T) {
			m := definedNamesManager(t, proofs)
			root, want := pullQuantFixture(t, m)
			proc, err := NewPullQuant(m)
			definedNamesMust(t, err)
			t.Cleanup(func() { definedNamesMust(t, proc.Close()) })
			out, err := m.NewRef(ast.ASTHandle{})
			definedNamesMust(t, err)
			defer func() { definedNamesMust(t, m.DelRef(&out)) }()
			proof, err := m.NewRef(ast.ASTHandle{})
			definedNamesMust(t, err)
			defer func() { definedNamesMust(t, m.DelRef(&proof)) }()
			definedNamesMust(t, proc.RewriteWithProof(root, out, proof))
			got, err := m.RefGet(out)
			definedNamesMust(t, err)
			if got != want {
				t.Fatal("pulling changed binder indices, order, weight or qid", got, want)
			}
			p, err := m.RefGet(proof)
			definedNamesMust(t, err)
			if proofs {
				fact, err := m.GetProofFact(p)
				definedNamesMust(t, err)
				left, err := m.GetArg(fact, 0)
				definedNamesMust(t, err)
				right, err := m.GetArg(fact, 1)
				definedNamesMust(t, err)
				if left != root || right != want {
					t.Fatal("pull proof does not relate the input to the flattened result")
				}
			} else if !p.IsNull() {
				t.Fatal("proof-disabled traversal produced a proof")
			}
			definedNamesMust(t, proc.Reset())
			definedNamesMust(t, proc.Close())
			got, err = m.RefGet(out)
			definedNamesMust(t, err)
			_, err = m.GetQuantifierBody(got)
			definedNamesMust(t, err)
		})
	}
}

func TestPullNestedQuantifiersRewritesBelowApplications(t *testing.T) {
	m := definedNamesManager(t, true)
	root, flat := pullQuantFixture(t, m)
	b, err := m.MkBoolSort()
	definedNamesMust(t, err)
	f, err := m.MkFuncDecl(symbol.Numerical(9912), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	definedNamesMust(t, err)
	application, err := m.MkApp(f, []ast.ASTHandle{root})
	definedNamesMust(t, err)
	definedNamesPin(t, m, application)
	want, err := m.MkApp(f, []ast.ASTHandle{flat})
	definedNamesMust(t, err)
	definedNamesPin(t, m, want)
	proc, err := NewPullNestedQuantifiers(m)
	definedNamesMust(t, err)
	t.Cleanup(func() { definedNamesMust(t, proc.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	definedNamesMust(t, err)
	defer func() { definedNamesMust(t, m.DelRef(&out)) }()
	proof, err := m.NewRef(ast.ASTHandle{})
	definedNamesMust(t, err)
	defer func() { definedNamesMust(t, m.DelRef(&proof)) }()
	for i := 0; i < 2; i++ {
		definedNamesMust(t, proc.RewriteWithProof(application, out, proof))
		got, err := m.RefGet(out)
		definedNamesMust(t, err)
		if got != want {
			t.Fatal("nested pull did not preserve the surrounding application")
		}
		p, err := m.RefGet(proof)
		definedNamesMust(t, err)
		fact, err := m.GetProofFact(p)
		definedNamesMust(t, err)
		left, err := m.GetArg(fact, 0)
		definedNamesMust(t, err)
		right, err := m.GetArg(fact, 1)
		definedNamesMust(t, err)
		if left != application || right != want {
			t.Fatal("surrounding application lost its congruence proof")
		}
		definedNamesMust(t, proc.Reset())
	}
}
