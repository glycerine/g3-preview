package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func distributeQuantifier(t *testing.T, m *ast.ASTManager, forall bool, sorts []ast.ASTHandle, body ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	names := make([]usymbol.Symbol, len(sorts))
	for i := range names {
		names[i] = usymbol.Numerical(uint32(i + 1))
	}
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: forall, Sorts: sorts, Names: names, Body: body})
	astTestMust(t, err)
	return q
}

func TestDistributeForallSplitsConjunctionAndEliminatesUnusedVariables(t *testing.T) {
	m, c, g, _ := boolRewriterConfigFixture(t)
	rw, err := NewDistributeForall(m, c, g)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, b)
	astTestMust(t, err)
	v1, err := m.MkVar(1, b)
	astTestMust(t, err)
	disjunction, err := m.MkOrArgs([]ast.ASTHandle{v0, v1})
	astTestMust(t, err)
	negated, err := m.MkNot(disjunction)
	astTestMust(t, err)
	q := distributeQuantifier(t, m, true, []ast.ASTHandle{b, b}, negated)
	astTestPin(t, m, q)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, rw.Apply(q, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if isAnd, err := rw.isBasicOp(got, ast.ASTOpAnd); err != nil || !isAnd {
		t.Fatal("distributed result is not a conjunction", isAnd, err)
	}
	n, err := m.GetNumArgs(got)
	astTestMust(t, err)
	if n != 2 {
		t.Fatal("distributed conjunct count", n)
	}
	for i := uint32(0); i < n; i++ {
		branch, err := m.GetArg(got, i)
		astTestMust(t, err)
		kind, err := m.GetKind(branch)
		astTestMust(t, err)
		if kind != ast.ASTQuantifier {
			t.Fatal("branch kind", i, kind)
		}
		decls, err := m.GetNumDeclarations(branch)
		astTestMust(t, err)
		if decls != 1 {
			t.Fatal("unused declaration was retained", i, decls)
		}
		body, err := m.GetQuantifierBody(branch)
		astTestMust(t, err)
		isNot, err := rw.isBasicOp(body, ast.ASTOpNot)
		astTestMust(t, err)
		if !isNot {
			t.Fatal("branch body was not negated", i)
		}
		variable, err := m.GetArg(body, 0)
		astTestMust(t, err)
		index, err := m.GetVarIndex(variable)
		astTestMust(t, err)
		if index != 0 {
			t.Fatal("remaining variable was not remapped", i, index)
		}
	}
}

func TestDistributeForallRebuildsParentsAndAppliesBooleanSimplification(t *testing.T) {
	m, c, g, _ := boolRewriterConfigFixture(t)
	rw, err := NewDistributeForall(m, c, g)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, b)
	astTestMust(t, err)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	disjunction, err := m.MkOrArgs([]ast.ASTHandle{v0, falsity})
	astTestMust(t, err)
	negated, err := m.MkNot(disjunction)
	astTestMust(t, err)
	q := distributeQuantifier(t, m, true, []ast.ASTHandle{b}, negated)
	decl, err := m.MkFuncDecl(astTestSymbol(t, m, "distribute-parent"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	root, err := m.MkApp(decl, []ast.ASTHandle{q})
	astTestMust(t, err)
	astTestPin(t, m, root)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, rw.Apply(root, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got == root {
		t.Fatal("parent was not rebuilt")
	}
	child, err := m.GetArg(got, 0)
	astTestMust(t, err)
	if kind, err := m.GetKind(child); err != nil || kind != ast.ASTQuantifier {
		t.Fatal("true conjunct was not removed", kind, err)
	}
	body, err := m.GetQuantifierBody(child)
	astTestMust(t, err)
	isNot, err := rw.isBasicOp(body, ast.ASTOpNot)
	astTestMust(t, err)
	if !isNot {
		t.Fatal("surviving branch body", body)
	}

	second := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, rw.Apply(root, second))
	again, err := m.RefGet(second)
	astTestMust(t, err)
	if again != got {
		t.Fatal("repeat application changed canonical result", got, again)
	}
}

func TestDistributeForallNonTargetDebugAndOwnerGuards(t *testing.T) {
	m, c, g, _ := boolRewriterConfigFixture(t)
	rw, err := NewDistributeForall(m, c, g)
	astTestMust(t, err)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	v0, err := m.MkVar(0, b)
	astTestMust(t, err)
	q := distributeQuantifier(t, m, true, []ast.ASTHandle{b}, v0)
	astTestPin(t, m, q)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, rw.Apply(q, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != q {
		t.Fatal("non-target quantifier changed", q, got)
	}
	if ast.ASTSourceDebug {
		exists := distributeQuantifier(t, m, false, []ast.ASTHandle{b}, v0)
		astTestPin(t, m, exists)
		if err = rw.Apply(exists, out); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("debug forall assertion", err)
		}
	}
	copyOwner := *rw
	if err = copyOwner.Apply(q, out); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner accepted", err)
	}
	astTestMust(t, rw.Close())
	if err = rw.Apply(q, out); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed owner accepted", err)
	}
	astTestMust(t, rw.Close())
}
