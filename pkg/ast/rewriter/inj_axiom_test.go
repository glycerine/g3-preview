package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func injectivityFixture(t *testing.T, m *ast.ASTManager, duplicate bool) (ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	u, err := m.MkUninterpretedSort(astTestSymbol(t, m, "inj-U"), nil)
	astTestMust(t, err)
	v0, err := m.MkVar(0, u)
	astTestMust(t, err)
	v1, err := m.MkVar(1, u)
	astTestMust(t, err)
	c, err := m.MkConst(astTestSymbol(t, m, "inj-c"), u)
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "inj-f"), []ast.ASTHandle{u, u}, u, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	leftArgs := []ast.ASTHandle{v0, c}
	rightArgs := []ast.ASTHandle{v1, c}
	if duplicate {
		leftArgs[1], rightArgs[1] = v0, v1
	}
	left, err := m.MkApp(f, leftArgs)
	astTestMust(t, err)
	right, err := m.MkApp(f, rightArgs)
	astTestMust(t, err)
	appsEqual, err := m.MkEq(left, right)
	astTestMust(t, err)
	notAppsEqual, err := m.MkNot(appsEqual)
	astTestMust(t, err)
	varsEqual, err := m.MkEq(v0, v1)
	astTestMust(t, err)
	body, err := m.MkOr(notAppsEqual, varsEqual)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true,
		Sorts:  []ast.ASTHandle{u, u},
		Names:  []usymbol.Symbol{usymbol.Numerical(10), usymbol.Numerical(11)},
		Body:   body,
	})
	astTestMust(t, err)
	astTestPin(t, m, q)
	return q, u
}

func TestSimplifyInjectivityAxiomBuildsInverseAndPattern(t *testing.T) {
	m := astTestManager(t)
	q, _ := injectivityFixture(t, m, false)
	out := astTestPin(t, m, ast.ASTHandle{})
	changed, err := SimplifyInjectivityAxiom(m, q, out)
	astTestMust(t, err)
	if !changed {
		t.Fatal("source injectivity shape was refused")
	}
	got, err := m.RefGet(out)
	astTestMust(t, err)
	forall, err := m.IsForall(got)
	astTestMust(t, err)
	if !forall {
		t.Fatal("optimized axiom is not universal")
	}
	n, err := m.GetNumDeclarations(got)
	astTestMust(t, err)
	if n != 1 {
		t.Fatal("optimized declaration count", n)
	}
	patterns, err := m.GetNumPatterns(got)
	astTestMust(t, err)
	if patterns != 1 {
		t.Fatal("optimized pattern count", patterns)
	}
	pattern, err := m.GetPattern(got, 0)
	astTestMust(t, err)
	isPattern, err := m.IsPattern(pattern)
	astTestMust(t, err)
	if !isPattern {
		t.Fatal("optimized trigger is not a pattern")
	}
	body, err := m.GetQuantifierBody(got)
	astTestMust(t, err)
	projection, variable, ok, err := injBasicBinary(m, body, ast.ASTOpEq)
	astTestMust(t, err)
	if !ok {
		t.Fatal("optimized body is not equality")
	}
	if kind, err := m.GetKind(variable); err != nil || kind != ast.ASTVar {
		t.Fatal("optimized distinguished variable", kind, err)
	}
	decl, err := m.GetDecl(projection)
	astTestMust(t, err)
	skolem, err := m.IsSkolemDecl(decl)
	astTestMust(t, err)
	if !skolem {
		t.Fatal("inverse declaration is not fresh/skolem")
	}
	good, err := m.IsWellSorted(got, nil)
	astTestMust(t, err)
	if !good {
		t.Fatal("optimized axiom is not well sorted")
	}
}

func TestSimplifyInjectivityAxiomRefusesNonshapeWithoutChangingOutput(t *testing.T) {
	m := astTestManager(t)
	duplicate, _ := injectivityFixture(t, m, true)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, truth)
	changed, err := SimplifyInjectivityAxiom(m, duplicate, out)
	astTestMust(t, err)
	if changed {
		t.Fatal("duplicate varying argument accepted")
	}
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != truth {
		t.Fatal("refusal changed output", got, truth)
	}
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	v, err := m.MkVar(0, b)
	astTestMust(t, err)
	exists, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{usymbol.Numerical(1)}, Body: v})
	astTestMust(t, err)
	astTestPin(t, m, exists)
	changed, err = SimplifyInjectivityAxiom(m, exists, out)
	astTestMust(t, err)
	if changed {
		t.Fatal("existential accepted")
	}
}
