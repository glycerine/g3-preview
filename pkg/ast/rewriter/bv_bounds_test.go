package rewriter

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func bvBoundsFixture(t *testing.T) (*ast.ASTManager, ast.ASTHandle) {
	t.Helper()
	m, _ := bvAddFixture(t)
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "bv-bounds-x"), sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	return m, x
}

func bvBoundsRewrite(t *testing.T, b *BVBounds, expression ast.ASTHandle) (RewriteStatus, ast.ASTHandle) {
	t.Helper()
	m := b.manager
	decl, err := m.GetDecl(expression)
	astTestMust(t, err)
	n, err := m.GetNumArgs(expression)
	astTestMust(t, err)
	args := make([]ast.ASTHandle, n)
	for i := range args {
		args[i], err = m.GetArg(expression, uint32(i))
		astTestMust(t, err)
	}
	out := astTestPin(t, m, ast.ASTHandle{})
	status, err := b.Rewrite(32, decl, ast.SliceArguments(args), out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	return status, result
}

func TestBVBoundsContradictionAndSingletonRewrite(t *testing.T) {
	for _, contradiction := range []bool{false, true} {
		m, x := bvBoundsFixture(t)
		b, err := NewBVBounds(m)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, b.Close()) })
		three := bvAddNum(t, m, 3, 8)
		lowerValue := uint32(3)
		if contradiction {
			lowerValue = 4
		}
		lower := bvAddNum(t, m, lowerValue, 8)
		upperConstraint, err := m.MkBitVectorULE(x, three)
		astTestMust(t, err)
		lowerConstraint, err := m.MkBitVectorULE(lower, x)
		astTestMust(t, err)
		both, err := m.MkAndArgs([]ast.ASTHandle{upperConstraint, lowerConstraint})
		astTestMust(t, err)
		status, result := bvBoundsRewrite(t, b, both)
		if status != RewriteDone {
			t.Fatal("expected completed bounds rewrite", status)
		}
		if contradiction {
			if !m.IsFalse(result) {
				t.Fatal("contradictory bounds did not reduce to false", result)
			}
			continue
		}
		if kind, e := m.GetDeclKind(result); e != nil || kind != ast.ASTOpEq {
			t.Fatal("singleton did not reduce to equality", kind, e)
		}
		left, err := m.GetArg(result, 0)
		astTestMust(t, err)
		right, err := m.GetArg(result, 1)
		astTestMust(t, err)
		if left != x {
			t.Fatal("singleton equality variable", left, x)
		}
		text, matched, err := m.BitVectorNumeralText(right)
		astTestMust(t, err)
		if !matched || text != "3" {
			t.Fatal("singleton equality value", text, matched)
		}
	}
}

func TestBVBoundsNegativeIntervalsAndSignedConstraint(t *testing.T) {
	m, x := bvBoundsFixture(t)
	b, err := NewBVBounds(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	if okay, err := b.AddBoundUnsigned(x, big.NewInt(0), big.NewInt(254), true); err != nil || !okay {
		t.Fatal("negative interval", okay, err)
	}
	if sat, err := b.IsSatisfiable(); err != nil || !sat {
		t.Fatal("negative interval satisfiability", sat, err)
	}
	singletons, err := b.Singletons()
	astTestMust(t, err)
	if value := singletons[x]; value == nil || value.Int64() != 255 {
		t.Fatal("negative interval singleton", value)
	}

	b.Reset()
	minusOne := bvAddNum(t, m, 255, 8)
	signedUpper, err := m.MkBitVectorSLE(x, minusOne)
	astTestMust(t, err)
	if okay, err := b.AddConstraint(signedUpper); err != nil || !okay {
		t.Fatal("signed bound", okay, err)
	}
	if okay, err := b.AddBoundUnsigned(x, big.NewInt(0), big.NewInt(127), false); err != nil || !okay {
		t.Fatal("unsigned bound addition", okay, err)
	}
	if sat, err := b.IsSatisfiable(); err != nil || sat {
		t.Fatal("signed/unsigned contradiction", sat, err)
	}
}
