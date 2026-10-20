package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type maximizeBVFixture struct {
	m      *ast.ASTManager
	family ast.FamilyID
	sort   ast.ASTHandle
	vars   []ast.ASTHandle
}

func newMaximizeBVFixture(t *testing.T, proofs bool) maximizeBVFixture {
	t.Helper()
	m, family := bvAddFixture(t)
	if proofs {
		astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	}
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	vars := make([]ast.ASTHandle, 6)
	for i := range vars {
		vars[i], e = m.MkVar(uint32(i), sort)
		astTestMust(t, e)
		astTestPin(t, m, vars[i])
	}
	return maximizeBVFixture{m: m, family: family, sort: sort, vars: vars}
}

func maximizeBVApp(t *testing.T, f maximizeBVFixture, kind int32, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := f.m.MkFamilyApp(f.family, kind, nil, args)
	astTestMust(t, e)
	return h
}

func maximizeBVOutput(t *testing.T, m *ast.ASTManager) (ast.ASTRef, ast.ASTRef) {
	t.Helper()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	proof, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, m.DelRef(&proof))
		astTestMust(t, m.DelRef(&out))
	})
	return out, proof
}

func TestMaximizeBVSharingBalancedTreesReuseAndNumeralPosition(t *testing.T) {
	f := newMaximizeBVFixture(t, false)
	c, e := NewMaximizeBVSharingConfiguration(f.m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	out, proof := maximizeBVOutput(t, f.m)
	decl, e := f.m.GetDecl(maximizeBVApp(t, f, 4, f.vars[0], f.vars[1]))
	astTestMust(t, e)
	args := ast.SliceArguments(f.vars[:4])
	status, e := c.ReduceApp(decl, args, out, proof)
	astTestMust(t, e)
	if status != RewriteFailed {
		t.Fatal("kinds initialized before first scope", status)
	}
	astTestMust(t, c.PushScope())
	status, e = c.ReduceApp(decl, args, out, proof)
	astTestMust(t, e)
	if status != RewriteDone {
		t.Fatal("balanced rewrite status", status)
	}
	ab := maximizeBVApp(t, f, 4, f.vars[0], f.vars[1])
	cd := maximizeBVApp(t, f, 4, f.vars[2], f.vars[3])
	want := maximizeBVApp(t, f, 4, ab, cd)
	got, e := f.m.RefGet(out)
	astTestMust(t, e)
	if got != want || len(c.entries) != 3 {
		t.Fatal("balanced tree or registered pairs", got, want, len(c.entries))
	}
	status, e = c.ReduceApp(decl, ast.SliceArguments([]ast.ASTHandle{f.vars[0], f.vars[1], f.vars[4]}), out, proof)
	astTestMust(t, e)
	got, e = f.m.RefGet(out)
	astTestMust(t, e)
	want = maximizeBVApp(t, f, 4, ab, f.vars[4])
	if status != RewriteDone || got != want || len(c.entries) != 4 {
		t.Fatal("cached pair reuse", status, got, want, len(c.entries))
	}
	numeral := bvAddNum(t, f.m, 3, 8)
	status, e = c.ReduceApp(decl, ast.SliceArguments([]ast.ASTHandle{numeral, f.vars[0], f.vars[1], f.vars[2]}), out, proof)
	astTestMust(t, e)
	got, e = f.m.RefGet(out)
	astTestMust(t, e)
	want = maximizeBVApp(t, f, 4, numeral, maximizeBVApp(t, f, 4, ab, f.vars[2]))
	if status != RewriteDone || got != want {
		t.Fatal("leading numeral position", status, got, want)
	}
}

func TestMaximizeBVSharingScopesResetAndGuards(t *testing.T) {
	f := newMaximizeBVFixture(t, false)
	c, e := NewMaximizeBVSharingConfiguration(f.m)
	astTestMust(t, e)
	out, proof := maximizeBVOutput(t, f.m)
	decl, e := f.m.GetDecl(maximizeBVApp(t, f, 6, f.vars[0], f.vars[1]))
	astTestMust(t, e)
	astTestMust(t, c.PushScope())
	_, e = c.ReduceApp(decl, ast.SliceArguments(f.vars[:4]), out, proof)
	astTestMust(t, e)
	outer := len(c.entries)
	astTestMust(t, c.PushScope())
	_, e = c.ReduceApp(decl, ast.SliceArguments([]ast.ASTHandle{f.vars[0], f.vars[1], f.vars[4]}), out, proof)
	astTestMust(t, e)
	if len(c.entries) <= outer {
		t.Fatal("inner scope added no entries")
	}
	astTestMust(t, c.PopScope(1))
	if len(c.entries) != outer {
		t.Fatal("inner scope restoration", len(c.entries), outer)
	}
	c.Reset()
	if len(c.cache) != 0 || len(c.entries) != outer {
		t.Fatal("reset must retain owned entries but clear lookup", len(c.cache), len(c.entries))
	}
	if e = c.PopScope(0); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("zero pop guard", e)
	}
	astTestMust(t, c.PopScope(1))
	if len(c.entries) != 0 || len(c.scopes) != 0 {
		t.Fatal("outer scope restoration", len(c.entries), len(c.scopes))
	}
	astTestMust(t, c.Close())
	astTestMust(t, c.Close())
	if e = c.PushScope(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed configuration guard", e)
	}
}

func TestMaximizeBVSharingWrapperProofAndLifecycle(t *testing.T) {
	f := newMaximizeBVFixture(t, true)
	rw, e := NewMaximizeBVSharing(f.m)
	astTestMust(t, e)
	astTestMust(t, rw.PushScope())
	root := maximizeBVApp(t, f, 30, f.vars[0], f.vars[1], f.vars[2], f.vars[3])
	out, proof := maximizeBVOutput(t, f.m)
	astTestMust(t, rw.Rewrite(root, out, proof))
	got, e := f.m.RefGet(out)
	astTestMust(t, e)
	p, e := f.m.RefGet(proof)
	astTestMust(t, e)
	if got == root || p.IsNull() {
		t.Fatal("wrapper result or proof", got, root, p)
	}
	fact, e := f.m.GetProofFact(p)
	astTestMust(t, e)
	wantFact, e := f.m.MkEq(root, got)
	astTestMust(t, e)
	if fact != wantFact {
		t.Fatal("wrapper proof fact", fact, wantFact)
	}
	copy := *rw
	if e = copy.Reset(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied wrapper guard", e)
	}
	astTestMust(t, rw.PopScope(1))
	astTestMust(t, rw.Close())
	astTestMust(t, rw.Close())
	if e = rw.Rewrite(root, out, proof); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed wrapper guard", e)
	}
}
