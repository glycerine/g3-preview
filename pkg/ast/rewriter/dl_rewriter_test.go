package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestDLRewriterFiniteNumeralsIdentityZeroAndRetainedZeroLeftRefusal(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "DL4"), 4)
	astTestMust(t, err)
	astTestPin(t, m, sort)
	values := make([]ast.ASTHandle, 3)
	for i := range values {
		values[i], err = m.MkDatalogNumeral(uint64(i), sort)
		astTestMust(t, err)
		astTestPin(t, m, values[i])
	}
	if _, err = m.MkDatalogNumeral(4, sort); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("out-of-domain numeral", err)
	}
	lt, err := m.MkDatalogLT(values[0], values[1])
	astTestMust(t, err)
	astTestPin(t, m, lt)
	decl, err := m.GetDecl(lt)
	astTestMust(t, err)
	rw, err := NewDLRewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	out := astTestPin(t, m, ast.ASTHandle{})
	check := func(left, right ast.ASTHandle, want bool) {
		t.Helper()
		status, err := rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{left, right}), out)
		astTestMust(t, err)
		if status != RewriteDone {
			t.Fatal("expected rewrite", status)
		}
		got, err := m.RefGet(out)
		astTestMust(t, err)
		if got != func() ast.ASTHandle { h, e := m.MkBoolVal(want); astTestMust(t, e); return h }() {
			t.Fatal("comparison result", want, got)
		}
	}
	check(values[1], values[2], true)
	check(values[2], values[1], false)

	x, err := m.MkConst(astTestSymbol(t, m, "dl-x"), sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	check(x, x, false)
	check(x, values[0], false)
	sentinel, err := m.MkTrue()
	astTestMust(t, err)
	astTestMust(t, m.SetRef(out, sentinel))
	status, err := rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{values[0], x}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("source's unreachable 0 < x branch fired", status)
	}
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != sentinel {
		t.Fatal("refusal changed output", got, sentinel)
	}
}

func TestDLRewriterExtendedBVBooleanNumeralsAndWidthBoundary(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "DL2"), 2)
	astTestMust(t, err)
	zero, err := m.MkDatalogNumeral(0, sort)
	astTestMust(t, err)
	one, err := m.MkDatalogNumeral(1, sort)
	astTestMust(t, err)
	lt, err := m.MkDatalogLT(zero, one)
	astTestMust(t, err)
	decl, err := m.GetDecl(lt)
	astTestMust(t, err)
	rw, err := NewDLRewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	out := astTestPin(t, m, ast.ASTHandle{})
	bv1, err := m.BitVectorNumeralUint64(1, 8)
	astTestMust(t, err)
	bv2, err := m.BitVectorNumeralUint64(2, 8)
	astTestMust(t, err)
	status, err := rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{bv1, bv2}), out)
	astTestMust(t, err)
	if status != RewriteDone {
		t.Fatal("narrow BV numeral refusal", status)
	}
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if !m.IsTrue(got) {
		t.Fatal("narrow BV comparison", got)
	}
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	status, err = rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{truth, falsity}), out)
	astTestMust(t, err)
	if status != RewriteDone {
		t.Fatal("Boolean numeral refusal", status)
	}
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if !m.IsFalse(got) {
		t.Fatal("Boolean comparison", got)
	}
	wide0, err := m.BitVectorNumeralUint64(0, 64)
	astTestMust(t, err)
	wide1, err := m.BitVectorNumeralUint64(1, 64)
	astTestMust(t, err)
	status, err = rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{wide0, wide1}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("source width<64 boundary", status)
	}
}

func TestDLRewriterFamilyArityAndOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	sort, err := m.MkDatalogFiniteSort(astTestSymbol(t, m, "DLGuard"), 2)
	astTestMust(t, err)
	zero, err := m.MkDatalogNumeral(0, sort)
	astTestMust(t, err)
	one, err := m.MkDatalogNumeral(1, sort)
	astTestMust(t, err)
	lt, err := m.MkDatalogLT(zero, one)
	astTestMust(t, err)
	decl, err := m.GetDecl(lt)
	astTestMust(t, err)
	rw, err := NewDLRewriter(m)
	astTestMust(t, err)
	out := astTestPin(t, m, ast.ASTHandle{})
	if status, err := rw.RewriteApp(decl, ast.SliceArguments([]ast.ASTHandle{zero}), out); status != RewriteFailed || !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("arity guard", status, err)
	}
	copyOwner := *rw
	if _, err = copyOwner.FamilyID(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner accepted", err)
	}
	astTestMust(t, rw.Close())
	if _, err = rw.FamilyID(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed owner accepted", err)
	}
	astTestMust(t, rw.Close())
}
