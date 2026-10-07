// Source: src/ast/rewriter/poly_rewriter_def.h:403-408,490-523;
// src/ast/rewriter/bv_rewriter.h:35,41-44.
package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func bvPolynomialRaw(t *testing.T, m *ast.ASTManager, fid ast.FamilyID, k int32, s ast.ASTHandle, name string, args []ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	domain := make([]ast.ASTHandle, len(args))
	for i := range domain {
		domain[i] = s
	}
	info, e := m.NewFuncDeclInfo(fid, k, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, name), domain, s, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	h, e := m.MkApp(d, args)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

func TestBVPolynomialPowerProductBorrowingAndUnreadNumeralMetadata(t *testing.T) {
	m, fid := bvAddFixture(t)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	// This raw NUM has no value/width parameters; identity checks must not read them.
	raw := bvPolynomialRaw(t, m, fid, 0, s, "power-product-unread-num", nil)
	binary, e := m.MkBitVectorMul(raw, x)
	astTestMust(t, e)
	astTestPin(t, m, binary)
	reversed, e := m.MkBitVectorMul(x, raw)
	astTestMust(t, e)
	astTestPin(t, m, reversed)
	nary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{raw, x, y})
	astTestMust(t, e)
	astTestPin(t, m, nary)
	unary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{raw})
	astTestMust(t, e)
	astTestPin(t, m, unary)
	empty := bvPolynomialRaw(t, m, fid, 6, s, "power-product-empty-mul", nil)
	for _, row := range []struct{ input, want ast.ASTHandle }{{x, x}, {raw, raw}, {binary, x}, {reversed, reversed}, {nary, nary}, {unary, unary}, {empty, empty}} {
		before := m.Stats()
		refs, e := m.GetRefCount(row.want)
		astTestMust(t, e)
		m.Budget().FailAfter(0)
		got, body := bvPowerProduct(m, row.input)
		m.Budget().FailAfter(-1)
		astTestMust(t, body)
		after, e := m.GetRefCount(row.want)
		astTestMust(t, e)
		if got != row.want || m.Stats() != before || after != refs {
			t.Fatal("borrowed power product", got, row.want, after, refs)
		}
	}
}

func TestBVPolynomialMonomialOrdinalOriginalBranchesAndTies(t *testing.T) {
	m, fid := bvAddFixture(t)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	raw := bvPolynomialRaw(t, m, fid, 0, s, "ordinal-unread-num", nil)
	mul, e := m.MkBitVectorMul(raw, x)
	astTestMust(t, e)
	astTestPin(t, m, mul)
	nary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{raw, x, y})
	astTestMust(t, e)
	astTestPin(t, m, nary)
	symbolic, e := m.MkBitVectorMul(x, y)
	astTestMust(t, e)
	astTestPin(t, m, symbolic)
	unary, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, unary)
	add, e := m.MkBitVectorAdd(x, y)
	astTestMust(t, e)
	astTestPin(t, m, add)
	// A foreign family's MUL-shaped declaration stays essentially variable.
	foreign := bvPolynomialRaw(t, m, 0, 6, s, "foreign-mul", []ast.ASTHandle{raw, y})
	rows := []struct {
		h, idFrom ast.ASTHandle
		numeral   bool
	}{{x, x, false}, {y, y, false}, {raw, raw, true}, {mul, x, false}, {nary, x, false}, {symbolic, symbolic, false}, {unary, unary, false}, {add, add, false}, {foreign, foreign, false}}
	ordinals := make([]int32, len(rows))
	for i, row := range rows {
		id, e := m.GetASTID(row.idFrom)
		astTestMust(t, e)
		want := int32(id)
		if row.numeral {
			want = -1
		}
		before := m.Stats()
		m.Budget().FailAfter(0)
		got, body := bvMonomialOrdinal(m, fid, row.h)
		m.Budget().FailAfter(-1)
		astTestMust(t, body)
		if got != want || m.Stats() != before {
			t.Fatal("ordinal branch", i, got, want)
		}
		ordinals[i] = want
	}
	for i, left := range rows {
		for j, right := range rows {
			got, e := bvMonomialLess(m, fid, false, left.h, right.h)
			astTestMust(t, e)
			if got != (ordinals[i] < ordinals[j]) {
				t.Fatal("ordinal pair/tie", i, j, got)
			}
		}
	}
	// Unlike get_power_product, ordinal does not test multiplication arity.
	// A one-operand coefficient MUL reaches its absent second operand.
	bad, e := m.MkFamilyApp(fid, 6, nil, []ast.ASTHandle{raw})
	astTestMust(t, e)
	astTestPin(t, m, bad)
	if _, e = bvMonomialOrdinal(m, fid, bad); e == nil {
		t.Fatal("missing ordinal power product accepted")
	}
}

func TestBVPolynomialMonomialStructuralOrderIgnoresAllocationIDs(t *testing.T) {
	m, fid := bvAddFixture(t)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	// Deliberately construct variables in the reverse of structural index order.
	earlier, e := m.MkVar(10, s)
	astTestMust(t, e)
	astTestPin(t, m, earlier)
	later, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, later)
	ordinal, e := bvMonomialLess(m, fid, false, earlier, later)
	astTestMust(t, e)
	structural, e := bvMonomialLess(m, fid, true, earlier, later)
	astTestMust(t, e)
	if !ordinal || structural {
		t.Fatal("structural mode substituted IDs", ordinal, structural)
	}
	reverse, e := bvMonomialLess(m, fid, true, later, earlier)
	astTestMust(t, e)
	equal, e := bvMonomialLess(m, fid, true, later, later)
	astTestMust(t, e)
	if !reverse || equal {
		t.Fatal("structural reverse/identity", reverse, equal)
	}
}

func TestBVPolynomialMonomialOrdinalSignedIDConversion(t *testing.T) {
	m, fid := bvAddFixture(t)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	// Original ordinal returns int from an unsigned ID on the 32-bit-int source profile.
	m.ExprIDs.Next = uint32(1) << 31
	h, e := m.MkVar(23, s)
	astTestMust(t, e)
	astTestPin(t, m, h)
	got, e := bvMonomialOrdinal(m, fid, h)
	astTestMust(t, e)
	if got != -2147483648 {
		t.Fatal("signed source ordinal", got)
	}
	if _, e = bvMonomialOrdinal(nil, fid, h); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("nil manager", e)
	}
}
