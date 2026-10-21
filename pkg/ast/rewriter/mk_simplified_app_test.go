package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func simplifiedAppFixture(t *testing.T) (*ast.ASTManager, *SimplifiedApp) {
	t.Helper()
	m, c, g, local := boolRewriterConfigFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterArrayPlugin()
	astTestMust(t, err)
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	r, err := NewSimplifiedApp(m, c, g, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	return m, r
}

func TestSimplifiedAppTheoryApplicationAndEqualityDispatch(t *testing.T) {
	m, r := simplifiedAppFixture(t)
	two := arithRewriterNumeral(t, m, "2", true)
	three := arithRewriterNumeral(t, m, "3", true)
	sum, err := m.MkFamilyApp(r.arithmetic.family, arithOpAdd, nil, []ast.ASTHandle{two, three})
	astTestMust(t, err)
	sumDecl, err := m.GetDecl(sum)
	astTestMust(t, err)
	out := astTestPin(t, m, sum)
	status, err := r.MkCore(sumDecl, ast.SliceArguments([]ast.ASTHandle{two, three}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone {
		t.Fatal("arithmetic dispatch status", status)
	}
	arithRewriterWantNumeral(t, m, got, "5", true)

	left := bvAddNum(t, m, 7, 8)
	right := bvAddNum(t, m, 7, 8)
	equality, err := m.MkEq(left, right)
	astTestMust(t, err)
	equalityDecl, err := m.GetDecl(equality)
	astTestMust(t, err)
	status, err = r.MkCore(equalityDecl, ast.SliceArguments([]ast.ASTHandle{left, right}), out)
	astTestMust(t, err)
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || !m.IsTrue(got) {
		t.Fatal("bit-vector equality dispatch", status, got)
	}
}

func TestSimplifiedAppMakeClearsOutputAndFallsBack(t *testing.T) {
	m, r := simplifiedAppFixture(t)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	name := astTestSymbol(t, m, "simplified-app-uninterpreted")
	info, err := m.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	decl, err := m.MkFuncDecl(name, nil, boolSort, info)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	want, err := m.MkApp(decl, nil)
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, truth)

	status, err := r.MkCore(decl, ast.AppArguments{}, out)
	astTestMust(t, err)
	unchanged, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteFailed || unchanged != truth {
		t.Fatal("core fallback changed output", status, unchanged)
	}
	astTestMust(t, r.Make(decl, ast.AppArguments{}, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != want {
		t.Fatal("raw application fallback", got, want)
	}
}

func TestSimplifiedAppBooleanFallbackAndCloseGuards(t *testing.T) {
	m, r := simplifiedAppFixture(t)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	negation, err := m.MkNot(truth)
	astTestMust(t, err)
	decl, err := m.GetDecl(negation)
	astTestMust(t, err)
	out := astTestPin(t, m, truth)
	status, err := r.MkCore(decl, ast.SliceArguments([]ast.ASTHandle{truth}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || !m.IsFalse(got) {
		t.Fatal("Boolean dispatch", status, got)
	}
	astTestMust(t, r.Close())
	if status, err = r.MkCore(decl, ast.SliceArguments([]ast.ASTHandle{truth}), out); status != RewriteFailed || err != ast.ErrASTHandle {
		t.Fatal("closed core accepted", status, err)
	}
}
