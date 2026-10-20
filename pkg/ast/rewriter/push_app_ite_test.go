package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type pushITEFixture struct {
	m                   *ast.ASTManager
	boolSort, valueSort ast.ASTHandle
	c1, c2              ast.ASTHandle
	x1, y1, x2, y2      ast.ASTHandle
	f                   ast.ASTHandle
}

func newPushITEFixture(t *testing.T, proofs bool) pushITEFixture {
	t.Helper()
	m := astTestManager(t)
	if proofs {
		astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	}
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "push-ite-value"), nil)
	astTestMust(t, e)
	constant := func(name string, sort ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkConst(astTestSymbol(t, m, name), sort)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "push-ite-f"), []ast.ASTHandle{s, s}, s, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	return pushITEFixture{m: m, boolSort: b, valueSort: s,
		c1: constant("push-ite-c1", b), c2: constant("push-ite-c2", b),
		x1: constant("push-ite-x1", s), y1: constant("push-ite-y1", s),
		x2: constant("push-ite-x2", s), y2: constant("push-ite-y2", s), f: f}
}

func pushITEMustApp(t *testing.T, m *ast.ASTManager, d ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkApp(d, args)
	astTestMust(t, e)
	return h
}

func pushITEMustIte(t *testing.T, m *ast.ASTManager, c, a, b ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkIte(c, a, b)
	astTestMust(t, e)
	return h
}

func pushITERewrite(t *testing.T, r *PushAppITE, m *ast.ASTManager, root ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	proof, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, m.DelRef(&proof))
		astTestMust(t, m.DelRef(&out))
	})
	astTestMust(t, r.Rewrite(root, out, proof))
	h, e := m.RefGet(out)
	astTestMust(t, e)
	p, e := m.RefGet(proof)
	astTestMust(t, e)
	return h, p
}

func TestPushAppITELiftsTermArgumentsAndHonorsConservativeMode(t *testing.T) {
	x := newPushITEFixture(t, false)
	first := pushITEMustIte(t, x.m, x.c1, x.x1, x.y1)
	root := pushITEMustApp(t, x.m, x.f, x.x2, first)
	rw, e := NewPushAppITE(x.m, true)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, rw.Close()) })
	got, _ := pushITERewrite(t, rw, x.m, root)
	want := pushITEMustIte(t, x.m, x.c1,
		pushITEMustApp(t, x.m, x.f, x.x2, x.x1),
		pushITEMustApp(t, x.m, x.f, x.x2, x.y1))
	if got != want {
		t.Fatal("single term ITE was not lifted", got, want)
	}

	second := pushITEMustIte(t, x.m, x.c2, x.x2, x.y2)
	two := pushITEMustApp(t, x.m, x.f, first, second)
	got, _ = pushITERewrite(t, rw, x.m, two)
	if got != two {
		t.Fatal("conservative rewrite accepted two ITE arguments", got, two)
	}
	full, e := NewPushAppITE(x.m, false)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, full.Close()) })
	got, _ = pushITERewrite(t, full, x.m, two)
	want = pushITEMustIte(t, x.m, x.c1,
		pushITEMustIte(t, x.m, x.c2,
			pushITEMustApp(t, x.m, x.f, x.x1, x.x2), pushITEMustApp(t, x.m, x.f, x.x1, x.y2)),
		pushITEMustIte(t, x.m, x.c2,
			pushITEMustApp(t, x.m, x.f, x.y1, x.x2), pushITEMustApp(t, x.m, x.f, x.y1, x.y2)))
	if got != want {
		t.Fatal("full rewrite did not lift both ITE arguments", got, want)
	}
}

func TestPushAppITERefusesBooleanITEAndITEDeclaration(t *testing.T) {
	x := newPushITEFixture(t, false)
	p := pushITEMustIte(t, x.m, x.c1, x.c1, x.c2)
	g, e := x.m.MkFuncDecl(astTestSymbol(t, x.m, "push-ite-bool-f"), []ast.ASTHandle{x.boolSort}, x.boolSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	root := pushITEMustApp(t, x.m, g, p)
	rw, e := NewPushAppITE(x.m, false)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, rw.Close()) })
	got, _ := pushITERewrite(t, rw, x.m, root)
	if got != root {
		t.Fatal("Boolean ITE argument was lifted")
	}
	term := pushITEMustIte(t, x.m, x.c1, x.x1, x.y1)
	got, _ = pushITERewrite(t, rw, x.m, term)
	if got != term {
		t.Fatal("ITE declaration itself was rewritten")
	}
}

func TestNGPushAppITERequiresNongroundArgument(t *testing.T) {
	x := newPushITEFixture(t, false)
	groundITE := pushITEMustIte(t, x.m, x.c1, x.x1, x.y1)
	ground := pushITEMustApp(t, x.m, x.f, groundITE, x.x2)
	rw, e := NewNGPushAppITE(x.m, false)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, rw.Close()) })
	got, _ := pushITERewrite(t, rw, x.m, ground)
	if got != ground {
		t.Fatal("nonground variation rewrote ground application")
	}
	v, e := x.m.MkVar(0, x.boolSort)
	astTestMust(t, e)
	q, e := x.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{x.boolSort}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: v})
	astTestMust(t, e)
	h, e := x.m.MkFuncDecl(astTestSymbol(t, x.m, "push-ite-quantified"), []ast.ASTHandle{x.boolSort}, x.valueSort, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	quantified := pushITEMustApp(t, x.m, h, q)
	nonground := pushITEMustApp(t, x.m, x.f, groundITE, quantified)
	got, _ = pushITERewrite(t, rw, x.m, nonground)
	if got == nonground {
		t.Fatal("nonground variation refused free-variable application")
	}
}

func TestPushAppITEConfigurationProofAndOwnerGuards(t *testing.T) {
	x := newPushITEFixture(t, true)
	term := pushITEMustIte(t, x.m, x.c1, x.x1, x.y1)
	root := pushITEMustApp(t, x.m, x.f, term, x.x2)
	cfg, e := NewPushAppITEConfiguration(x.m, true)
	astTestMust(t, e)
	if cfg.RewritePatterns() {
		t.Fatal("push-app-ite must leave quantifier patterns untouched")
	}
	out, e := x.m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	proof, e := x.m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() {
		astTestMust(t, x.m.DelRef(&proof))
		astTestMust(t, x.m.DelRef(&out))
	}()
	status, e := cfg.ReduceApp(x.f, ast.SliceArguments([]ast.ASTHandle{term, x.x2}), out, proof)
	astTestMust(t, e)
	if status != Rewrite2 {
		t.Fatal("configuration status", status)
	}
	p, e := x.m.RefGet(proof)
	astTestMust(t, e)
	if yes, xerr := x.m.IsProofKind(p, ast.ASTPrRewrite); xerr != nil || !yes {
		t.Fatal("configuration rewrite proof", yes, xerr)
	}
	rw, e := NewPushAppITE(x.m, true)
	astTestMust(t, e)
	got, p := pushITERewrite(t, rw, x.m, root)
	if got == root || p.IsNull() {
		t.Fatal("owner result or proof missing", got, p)
	}
	fact, e := x.m.GetProofFact(p)
	astTestMust(t, e)
	wantFact, e := x.m.MkEq(root, got)
	astTestMust(t, e)
	if fact != wantFact {
		t.Fatal("owner proof fact", fact, wantFact)
	}
	copy := *rw
	if e = copy.Rewrite(root, out, proof); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner guard", e)
	}
	astTestMust(t, rw.Reset())
	astTestMust(t, rw.Close())
	astTestMust(t, rw.Close())
	if e = rw.Rewrite(root, out, proof); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed rewrite guard", e)
	}
}
