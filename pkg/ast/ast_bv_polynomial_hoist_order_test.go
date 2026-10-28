package ast

// Source: src/ast/rewriter/poly_rewriter_def.h:428-451.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorPolynomialHoistOrderOriginalBranches(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	product := func(text string, pp ASTHandle) ASTHandle {
		n := bvRawOffsetTestNumeral(t, m, fid, s, text, 8)
		h, e := m.MkBitVectorMul(n, pp)
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	negative, zero, threeX, threeY, large := product("-1", x), product("0", x), product("3", x), product("3", y), product("257", x)
	reversed, e := m.MkBitVectorMul(x, bvRawOffsetTestNumeral(t, m, fid, s, "3", 8))
	astTestMust(t, e)
	astTestPin(t, m, reversed)
	nary, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{bvRawOffsetTestNumeral(t, m, fid, s, "3", 8), x, y})
	astTestMust(t, e)
	astTestPin(t, m, nary)
	// Nonproducts precede products; equal coefficients compare borrowed pp IDs.
	for _, row := range []struct {
		left, right ASTHandle
		want        bool
	}{{x, y, true}, {y, x, false}, {x, negative, true}, {negative, x, false}, {negative, zero, true}, {zero, negative, false}, {threeX, threeY, true}, {threeY, threeX, false}, {threeX, threeX, false}, {threeX, large, true}, {large, threeX, false}, {reversed, zero, true}, {nary, zero, true}} {
		var call ASTBitVectorNumeralInvocation
		before := m.Stats()
		refs, e := m.GetRefCount(x)
		astTestMust(t, e)
		m.Budget().FailAfter(0)
		got, body := m.BitVectorPolynomialHoistLessWithInvocation(row.left, row.right, &call)
		m.Budget().FailAfter(-1)
		astTestMust(t, body)
		after, e := m.GetRefCount(x)
		astTestMust(t, e)
		if got != row.want || before != m.Stats() || after != refs {
			t.Fatal("hoist ordering", row, got)
		}
		f := call.receipt.frame
		if !f.polynomialHoistCompare || !f.consumed[0] || !f.consumed[1] {
			t.Fatal("coefficient locals did not end")
		}
		astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
	}
}

func TestASTBitVectorPolynomialHoistOrderBothReadsBeforeShortcut(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	// The first argument is not a coefficient product, but the second must
	// still execute its reached NUM reader before returning the ordering.
	n := bvRawOffsetTestNumeral(t, m, fid, s, strings.Repeat("9", 1300), 8)
	product, e := m.MkBitVectorMul(n, x)
	astTestMust(t, e)
	astTestPin(t, m, product)
	var call ASTBitVectorNumeralInvocation
	m.Budget().FailAfter(0)
	_, body := m.BitVectorPolynomialHoistLessWithInvocation(x, product, &call)
	m.Budget().FailAfter(-1)
	if body == nil {
		t.Fatal("left nonproduct skipped right coefficient copy")
	}
	f := call.receipt.frame
	child, _, _ := f.children[1].Causes()
	if child == nil || errors.Is(child, arena.ErrInvalidSlot) || !f.placed[0] || !f.complete[0] || !f.placed[1] || !f.complete[1] {
		t.Fatal("lost reached second reader", child)
	}
	if _, e = m.BitVectorPolynomialHoistLessWithInvocation(x, product, &call); !errors.Is(e, ErrASTPending) {
		t.Fatal("failed comparator retried", e)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
}

func TestASTBitVectorPolynomialHoistOrderEveryActualCopyReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 100 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			s, e := m.BitVectorSort(8)
			astTestMust(t, e)
			x, e := m.MkVar(0, s)
			astTestMust(t, e)
			astTestPin(t, m, x)
			makeProduct := func(text string) ASTHandle {
				n := bvRawOffsetTestNumeral(t, m, fid, s, text, 8)
				h, e := m.MkBitVectorMul(n, x)
				astTestMust(t, e)
				astTestPin(t, m, h)
				return h
			}
			left := makeProduct("184467440737095516170000000000000000003")
			right := makeProduct("184467440737095516170000000000000000005")
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			got, body := m.BitVectorPolynomialHoistLessWithInvocation(left, right, &call)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if _, e = m.BitVectorPolynomialHoistLessWithInvocation(left, right, &call); !errors.Is(e, ErrASTPending) {
					t.Fatal("comparator retried", e)
				}
			} else {
				complete = true
				if !got {
					t.Fatal("large coefficient ordering")
				}
			}
			ready := m.BitVectorNumeralInvocationSourceReady(&call)
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			after := m.BitVectorNumeralInvocationSourceReady(&call)
			if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
				t.Fatal("cleanup replaced source history")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("comparison copy horizon", complete, failures)
	}
}
