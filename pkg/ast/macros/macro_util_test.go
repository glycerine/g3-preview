package macros_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(t, err)
	n, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	must(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b})
	must(t, err)
	t.Cleanup(func() {
		must(t, m.Close())
		must(t, n.Close())
		must(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("retained %d bytes", b.Used())
		}
	})
	return m
}

func declaration(t *testing.T, m *ast.ASTManager, name string, domain []ast.ASTHandle, result ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	s, err := m.Symbols.Intern(name)
	must(t, err)
	d, err := m.MkFuncDecl(s, domain, result, ast.ASTFuncDeclInfo{})
	must(t, err)
	return d
}

func TestMacroUtilHeadsSimpleMacrosAndForbiddenDeclarations(t *testing.T) {
	m := fixture(t)
	u, err := macros.NewUtil(m)
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	v1, err := m.MkVar(1, b)
	must(t, err)
	f := declaration(t, m, "f", []ast.ASTHandle{b, b}, b)
	head, err := m.MkApp(f, []ast.ASTHandle{v1, v0})
	must(t, err)
	if yes, e := u.IsMacroHead(head, 2); e != nil || !yes {
		t.Fatal("permuted head", yes, e)
	}
	duplicate, err := m.MkApp(f, []ast.ASTHandle{v0, v0})
	must(t, err)
	if yes, e := u.IsMacroHead(duplicate, 2); e != nil || yes {
		t.Fatal("duplicate head", yes, e)
	}
	eq, err := m.MkEq(head, v0)
	must(t, err)
	gotHead, def, yes, err := u.IsSimpleMacro(eq, 2)
	must(t, err)
	if !yes || gotHead != head || def != v0 {
		t.Fatal("simple macro", yes, gotHead, def)
	}
	u.SetForbidden([]ast.ASTHandle{f})
	if _, _, matched, e := u.IsSimpleMacro(eq, 2); e != nil || matched {
		t.Fatal("forbidden macro", matched, e)
	}
	u.SetForbidden(nil)
	self, err := m.MkEq(head, duplicate)
	must(t, err)
	if _, _, matched, e := u.IsSimpleMacro(self, 2); e != nil || matched {
		t.Fatal("recursive definition", matched, e)
	}
}

func TestMacroUtilQuasiHeadNormalizationAndCandidates(t *testing.T) {
	m := fixture(t)
	u, err := macros.NewUtil(m)
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	v1, err := m.MkVar(1, b)
	must(t, err)
	f := declaration(t, m, "f", []ast.ASTHandle{b, b, b}, b)
	g := declaration(t, m, "g", []ast.ASTHandle{b, b}, b)
	groundName, err := m.Symbols.Intern("ground")
	must(t, err)
	ground, err := m.MkConst(groundName, b)
	must(t, err)
	quasi, err := m.MkApp(f, []ast.ASTHandle{v0, ground, v1})
	must(t, err)
	if yes, e := u.IsQuasiMacroHead(quasi, 2); e != nil || !yes {
		t.Fatal("quasi head", yes, e)
	}

	head, err := m.MkApp(g, []ast.ASTHandle{v1, v0})
	must(t, err)
	definition, err := m.MkApp(f, []ast.ASTHandle{v1, v0, ground})
	must(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	must(t, u.NormalizeExpr(head, 2, definition, out))
	normalized, err := m.RefGet(out)
	must(t, err)
	want, err := m.MkApp(f, []ast.ASTHandle{v0, v1, ground})
	must(t, err)
	if normalized != want {
		t.Fatal("variable permutation was not normalized", normalized, want)
	}
	t.Cleanup(func() { must(t, m.DelRef(&out)) })

	eq, err := m.MkEq(head, definition)
	must(t, err)
	candidates, err := macros.NewCandidates(m)
	must(t, err)
	t.Cleanup(func() { must(t, candidates.Close()) })
	must(t, u.CollectMacroCandidates(eq, 2, candidates))
	if candidates.Len() != 1 {
		t.Fatalf("candidates = %d", candidates.Len())
	}
	row, err := candidates.At(0)
	must(t, err)
	if row.Declaration != g || row.Definition != want || !m.IsTrue(row.Condition) || !row.SatisfyAtom {
		t.Fatal("candidate row", row)
	}
}
