// Source: src/ast/rewriter/bv_trailing.cpp:51-85,392-394.
package rewriter

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBVTrailingOriginalEqualityReduction(t *testing.T) {
	m, p, fid := bvTrailingFixture(t)
	num := func(v uint32) ast.ASTHandle {
		h, e := m.BitVectorNumeralUnsigned(v, 8)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	app := func(k int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, k, nil, args)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	zero, two, twelve, twenty := num(0), num(2), num(12), num(20)
	low, e := m.MkBitVectorExtract(6, 0, x)
	satOK(t, e)
	astTestPin(t, m, low)
	zero1, e := m.BitVectorNumeralUnsigned(0, 1)
	satOK(t, e)
	astTestPin(t, m, zero1)
	rows := []struct {
		left, right ast.ASTHandle
		status      RewriteStatus
		kind        int32
		width       uint32
	}{
		{twelve, two, RewriteDone, ast.ASTOpFalse, 0},
		{x, twelve, RewriteFailed, ast.ASTOpEq, 8},
		{x, x, RewriteFailed, ast.ASTOpEq, 8},
		{zero, app(4, zero, zero), RewriteDone, ast.ASTOpTrue, 0},
		{app(6, two, x), app(6, two, x), RewriteDone, ast.ASTOpTrue, 0},
		{app(6, two, x), app(37, low, zero1), RewriteDone, ast.ASTOpTrue, 0},
		{twelve, twenty, Rewrite2, ast.ASTOpEq, 6},
	}
	for i, row := range rows {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := astTestPin(t, m, x)
			status, e := p.EqRemoveTrailing(row.left, row.right, out)
			satOK(t, e)
			h, e := m.RefGet(out)
			satOK(t, e)
			kind, e := m.GetDeclKind(h)
			satOK(t, e)
			if status != row.status || kind != row.kind {
				t.Fatal("source equality branch", status, kind, row.status, row.kind)
			}
			if row.kind == ast.ASTOpEq {
				for j := uint32(0); j < 2; j++ {
					a, e := m.GetArg(h, j)
					satOK(t, e)
					width, e := m.BitVectorSize(a)
					satOK(t, e)
					if width != row.width {
						t.Fatal("equality operand width", width)
					}
				}
				if row.status == RewriteFailed {
					left, e := m.GetArg(h, 0)
					satOK(t, e)
					right, e := m.GetArg(h, 1)
					satOK(t, e)
					if left != row.left || right != row.right {
						t.Fatal("zero lower bound must publish original equality")
					}
				}
			}
		})
	}
}

func TestBVTrailingEqualityReachedVerifyOverRemovalDomain(t *testing.T) {
	m, p, fid := bvTrailingFixture(t)
	num := func(v, w uint32) ast.ASTHandle {
		h, e := m.BitVectorNumeralUnsigned(v, w)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	variable := func(w uint32) ast.ASTHandle {
		s, e := m.BitVectorSort(w)
		satOK(t, e)
		h, e := m.MkVar(0, s)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	left, e := m.MkFamilyApp(fid, 37, nil, []ast.ASTHandle{variable(1), num(0, 4), num(0, 1)})
	satOK(t, e)
	astTestPin(t, m, left)
	right, e := m.MkBitVectorMul(num(4, 6), variable(6))
	satOK(t, e)
	astTestPin(t, m, right)
	out := astTestPin(t, m, left)
	_, body := p.EqRemoveTrailing(left, right, out)
	if !errors.Is(body, ast.ErrASTDomain) {
		t.Fatal("source VERIFY must reject concat removing 3 for request 2", body)
	}
	h, e := m.RefGet(out)
	satOK(t, e)
	if h != left {
		t.Fatal("failed VERIFY published equality")
	}
	if _, _, e := p.Count(left); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("failed VERIFY source owner resumed", e)
	}
}

func TestBVTrailingEqualityEveryColdReservation(t *testing.T) {
	success, failures := false, 0
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m, p, _ := bvTrailingFixture(t)
			sort, e := m.BitVectorSort(8)
			satOK(t, e)
			x, e := m.MkVar(0, sort)
			satOK(t, e)
			astTestPin(t, m, x)
			two, e := m.BitVectorNumeralUnsigned(2, 8)
			satOK(t, e)
			astTestPin(t, m, two)
			four, e := m.BitVectorNumeralUnsigned(4, 8)
			satOK(t, e)
			astTestPin(t, m, four)
			left, e := m.MkBitVectorMul(two, x)
			satOK(t, e)
			astTestPin(t, m, left)
			right, e := m.MkBitVectorMul(four, x)
			satOK(t, e)
			astTestPin(t, m, right)
			out := astTestPin(t, m, x)
			m.Budget().FailAfter(ordinal)
			status, body := p.EqRemoveTrailing(left, right, out)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f umemory.Failure
					if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
						t.Fatal("unexpected refusal", body)
					}
				}
				h, e := m.RefGet(out)
				satOK(t, e)
				if h != x {
					t.Fatal("failed equality body published")
				}
				if _, e := p.EqRemoveTrailing(left, right, out); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("failed equality retried", e)
				}
			} else {
				success = true
				if status != Rewrite2 {
					t.Fatal("completed equality status", status)
				}
			}
			satOK(t, p.Close())
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("cold equality horizon", success, failures)
	}
}
