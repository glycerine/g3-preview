package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func bvElimFixture(t *testing.T, proofs bool) (*ast.ASTManager, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	m, _ := bvAddFixture(t)
	if proofs {
		astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	}
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	bv, e := m.BitVectorSort(2)
	astTestMust(t, e)
	return m, b, bv
}

func TestBVElimQuantifierBodyPatternsMetadataAndOrder(t *testing.T) {
	m, b, bv := bvElimFixture(t, false)
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	v1, e := m.MkVar(1, bv)
	astTestMust(t, e)
	one := bvAddNum(t, m, 1, 2)
	eq, e := m.MkEq(v1, one)
	astTestMust(t, e)
	body, e := m.MkAnd(v0, eq)
	astTestMust(t, e)
	pred, e := m.MkFuncDecl(astTestSymbol(t, m, "bv-elim-pattern"), []ast.ASTHandle{bv}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	patBody := pushITEMustApp(t, m, pred, v1)
	pattern, e := m.MkPattern([]ast.ASTHandle{patBody})
	astTestMust(t, e)
	qid, skid := astTestSymbol(t, m, "bv-elim-qid"), astTestSymbol(t, m, "bv-elim-skid")
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{bv, b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "x"), astTestSymbol(t, m, "p")}, Body: body, Weight: 9, QID: qid, SkolemID: skid, Patterns: []ast.ASTHandle{pattern}})
	astTestMust(t, e)
	c, e := NewBVElimConfiguration(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	out, proof := maximizeBVOutput(t, m)
	done, e := c.ReduceQuantifier(q, body, ast.SliceArguments([]ast.ASTHandle{pattern}), ast.AppArguments{}, out, proof)
	astTestMust(t, e)
	if !done {
		t.Fatal("BV declaration was not eliminated")
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if forall, x := m.IsForall(got); x != nil || !forall {
		t.Fatal("source always rebuilds a universal quantifier", forall, x)
	}
	if n, x := m.GetNumDeclarations(got); x != nil || n != 3 {
		t.Fatal("expanded declaration count", n, x)
	}
	wantNames := []string{"x_1", "x_0", "p"}
	for i, want := range wantNames {
		name, x := m.GetDeclarationName(got, uint32(i))
		astTestMust(t, x)
		text, x := m.Symbols.Display(name)
		astTestMust(t, x)
		sort, x := m.GetDeclarationSort(got, uint32(i))
		astTestMust(t, x)
		if text != want || sort != b {
			t.Fatal("expanded declaration", i, text, want, sort, b)
		}
	}
	if weight, x := m.GetQuantifierWeight(got); x != nil || weight != 9 {
		t.Fatal("weight", weight, x)
	}
	if a, z, x := m.GetQuantifierIDs(got); x != nil || a != qid || z != skid {
		t.Fatal("quantifier IDs", a, z, x)
	}
	bit0, e := m.MkVar(1, b)
	astTestMust(t, e)
	bit1, e := m.MkVar(2, b)
	astTestMust(t, e)
	rebuilt, e := m.MkBitVectorFromBools([]ast.ASTHandle{bit0, bit1})
	astTestMust(t, e)
	wantEq, e := m.MkEq(rebuilt, one)
	astTestMust(t, e)
	wantBody, e := m.MkAnd(v0, wantEq)
	astTestMust(t, e)
	actualBody, e := m.GetQuantifierBody(got)
	astTestMust(t, e)
	if actualBody != wantBody {
		t.Fatal("substituted body", actualBody, wantBody)
	}
	actualPattern, e := m.GetPattern(got, 0)
	astTestMust(t, e)
	wantPatternBody := pushITEMustApp(t, m, pred, rebuilt)
	wantPattern, e := m.MkPattern([]ast.ASTHandle{wantPatternBody})
	astTestMust(t, e)
	if actualPattern != wantPattern {
		t.Fatal("substituted pattern", actualPattern, wantPattern)
	}
}

func TestBVElimRefusalWrapperProofAndLifecycle(t *testing.T) {
	m, b, bv := bvElimFixture(t, true)
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	plain, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, "plain")}, Body: v0})
	astTestMust(t, e)
	c, e := NewBVElimConfiguration(m)
	astTestMust(t, e)
	out, proof := maximizeBVOutput(t, m)
	done, e := c.ReduceQuantifier(plain, v0, ast.AppArguments{}, ast.AppArguments{}, out, proof)
	astTestMust(t, e)
	if done {
		t.Fatal("quantifier without BV declarations was rewritten")
	}
	astTestMust(t, c.Close())

	bvVar, e := m.MkVar(0, bv)
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{bv}, Names: []usymbol.Symbol{astTestSymbol(t, m, "bits")}, Body: truth, NoPatterns: []ast.ASTHandle{bvVar}})
	astTestMust(t, e)
	rw, e := NewBVElim(m)
	astTestMust(t, e)
	astTestMust(t, rw.Rewrite(q, out, proof))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	p, e := m.RefGet(proof)
	astTestMust(t, e)
	if got == q || p.IsNull() {
		t.Fatal("wrapper result or proof missing", got, q, p)
	}
	fact, e := m.GetProofFact(p)
	astTestMust(t, e)
	wantFact, e := m.MkEq(q, got)
	astTestMust(t, e)
	if fact != wantFact {
		t.Fatal("wrapper proof fact", fact, wantFact)
	}
	copy := *rw
	if e = copy.Reset(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied wrapper guard", e)
	}
	astTestMust(t, rw.Reset())
	astTestMust(t, rw.Close())
	astTestMust(t, rw.Close())
	if e = rw.Rewrite(q, out, proof); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed wrapper guard", e)
	}
}
