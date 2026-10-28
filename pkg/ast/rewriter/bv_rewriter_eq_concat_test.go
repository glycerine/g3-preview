package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:2306-2370.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func bvConcatVar(t *testing.T, m *ast.ASTManager, index, width uint32) ast.ASTHandle {
	t.Helper()
	s, e := m.BitVectorSort(width)
	astTestMust(t, e)
	h, e := m.MkVar(index, s)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
func TestBVRewriterConcatEqualityOriginalAlignment(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	ex, e := NewMkExtractProc(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, ex.Close()) }()
	a := bvConcatVar(t, m, 0, 3)
	b := bvConcatVar(t, m, 1, 5)
	d := bvConcatVar(t, m, 2, 6)
	f := bvConcatVar(t, m, 3, 2)
	lhs, e := m.MkBitVectorConcatPair(a, b)
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	rhs, e := m.MkBitVectorConcatPair(d, f)
	astTestMust(t, e)
	astTestPin(t, m, rhs)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteConcatenationEquality(lhs, rhs, out, ex)
	astTestMust(t, e)
	left0, e := ex.Apply(1, 0, b)
	astTestMust(t, e)
	right0, e := ex.Apply(1, 0, f)
	astTestMust(t, e)
	left1, e := ex.Apply(4, 2, b)
	astTestMust(t, e)
	right1, e := ex.Apply(2, 0, d)
	astTestMust(t, e)
	left2, e := ex.Apply(2, 0, a)
	astTestMust(t, e)
	right2, e := ex.Apply(5, 3, d)
	astTestMust(t, e)
	terms := make([]ast.ASTHandle, 3)
	for i, pair := range [][2]ast.ASTHandle{{left0, right0}, {left1, right1}, {left2, right2}} {
		terms[i], e = m.MkEq(pair[0], pair[1])
		astTestMust(t, e)
		astTestPin(t, m, terms[i])
	}
	want, e := m.MkAndArgs(terms)
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite3 || got != want {
		t.Fatal(st, got, want)
	}
}

func TestBVRewriterConcatEqualitySingletonSidesAndTargetGuards(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	ex, e := NewMkExtractProc(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, ex.Close()) }()
	a := bvConcatVar(t, m, 0, 3)
	b := bvConcatVar(t, m, 1, 5)
	whole := bvConcatVar(t, m, 2, 8)
	concat, e := m.MkBitVectorConcatPair(a, b)
	astTestMust(t, e)
	astTestPin(t, m, concat)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, reverse := range []bool{false, true} {
		lhs, rhs := concat, whole
		if reverse {
			lhs, rhs = rhs, lhs
		}
		st, e := p.RewriteConcatenationEquality(lhs, rhs, out, ex)
		astTestMust(t, e)
		lo, e := ex.Apply(4, 0, whole)
		astTestMust(t, e)
		hi, e := ex.Apply(7, 5, whole)
		astTestMust(t, e)
		lb, e := ex.Apply(4, 0, b)
		astTestMust(t, e)
		la, e := ex.Apply(2, 0, a)
		astTestMust(t, e)
		pairs := [][2]ast.ASTHandle{{lb, lo}, {la, hi}}
		if reverse {
			pairs = [][2]ast.ASTHandle{{lo, lb}, {hi, la}}
		}
		terms := make([]ast.ASTHandle, 2)
		for i, pair := range pairs {
			terms[i], e = m.MkEq(pair[0], pair[1])
			astTestMust(t, e)
			astTestPin(t, m, terms[i])
		}
		want, e := m.MkAndArgs(terms)
		astTestMust(t, e)
		astTestPin(t, m, want)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite3 || got != want {
			t.Fatal(reverse, st, got, want)
		}
	}
	yes, e := IsBitVectorConcatSplitTarget(nil, ast.ASTHandle{}, true)
	astTestMust(t, e)
	if !yes {
		t.Fatal("option must erase argument read")
	}
	yes, e = IsBitVectorConcatEqualityTarget(m, concat, ast.ASTHandle{}, true)
	astTestMust(t, e)
	if !yes {
		t.Fatal("concat first option must erase rhs read")
	}
	or, e := m.MkBitVectorOr([]ast.ASTHandle{whole, whole})
	astTestMust(t, e)
	astTestPin(t, m, or)
	xor, e := m.MkBitVectorXor([]ast.ASTHandle{whole, whole})
	astTestMust(t, e)
	astTestPin(t, m, xor)
	num := bvAddNum(t, m, 2, 8)
	for _, tc := range []struct {
		other         ast.ASTHandle
		enabled, want bool
	}{{concat, false, true}, {num, false, true}, {or, false, true}, {xor, false, false}, {whole, false, false}, {whole, true, true}} {
		for _, reverse := range []bool{false, true} {
			lhs, rhs := concat, tc.other
			if reverse {
				lhs, rhs = rhs, lhs
			}
			yes, e := IsBitVectorConcatEqualityTarget(m, lhs, rhs, tc.enabled)
			astTestMust(t, e)
			if yes != tc.want {
				t.Fatal(tc, reverse, yes)
			}
		}
	}
	yes, e = IsBitVectorConcatEqualityTarget(m, whole, num, true)
	astTestMust(t, e)
	if yes {
		t.Fatal("option cannot replace concat requirement")
	}
}

func TestBVRewriterConcatEqualityEveryColdReservation(t *testing.T) {
	complete := false
	sawLongArguments := false
	sawAssignment := false
	for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			ex, e := NewMkExtractProc(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, ex.Close()) }()
			left, right := make([]ast.ASTHandle, 40), make([]ast.ASTHandle, 40)
			for i := range left {
				left[i] = bvConcatVar(t, m, uint32(i), 1)
				right[i] = bvConcatVar(t, m, uint32(i+40), 1)
			}
			lhs, e := m.MkBitVectorConcat(left)
			astTestMust(t, e)
			astTestPin(t, m, lhs)
			rhs, e := m.MkBitVectorConcat(right)
			astTestMust(t, e)
			astTestPin(t, m, rhs)
			s, e := m.BitVectorSort(40)
			astTestMust(t, e)
			older := make([]ast.ASTHandle, 32)
			for i := range older {
				older[i], e = m.MkVar(uint32(i+100), s)
				astTestMust(t, e)
			}
			old, e := m.MkBitVectorOr(older)
			astTestMust(t, e)
			out, e := m.NewRef(old)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteConcatenationEquality(lhs, rhs, out, ex)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				nativeResult, nativeErr := m.RefGet(out)
				astTestMust(t, nativeErr)
				if nativeResult != old {
					sawAssignment = true
				}
				if st != Rewrite3 || p.concatEquality != nil {
					t.Fatal(st)
				}
			} else {
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if b := p.concatEquality; b != nil && len(b.equalities) > 0 {
					count := uint32(len(b.equalities))
					if b.i1 != 40-count || b.i2 != 40-count || b.low1 != 0 || b.low2 != 0 {
						t.Fatal("cursor advanced across refused push", count, b.i1, b.i2, b.low1, b.low2)
					}
				}
				if p.concatEquality != nil && len(p.concatEquality.equalities) > 16 {
					sawLongArguments = true
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != old {
					sawAssignment = true
					if st != Rewrite3 {
						t.Fatal(st)
					}
				}
				cause := p.cause
				if _, e = p.RewriteConcatenationEquality(lhs, rhs, out, ex); e != cause {
					t.Fatal("stopped replay", e, cause)
				}
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history")
				}
			}
			astTestMust(t, p.Close())
		})
	}
	if !complete || !sawLongArguments || !sawAssignment {
		t.Fatal(complete, sawLongArguments, sawAssignment)
	}
}

func TestBVRewriterConcatEqualityOriginalDebugCompletionGuard(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	ex, e := NewMkExtractProc(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, ex.Close()) }()
	a := bvConcatVar(t, m, 0, 8)
	b := bvConcatVar(t, m, 1, 4)
	rhs := bvConcatVar(t, m, 2, 8)
	lhs, e := m.MkBitVectorConcatPair(a, b)
	astTestMust(t, e)
	astTestPin(t, m, lhs)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, body := p.RewriteConcatenationEquality(lhs, rhs, out, ex)
	if ast.ASTSourceDebug {
		if !errors.Is(body, ast.ErrASTDomain) || p.concatEquality == nil || p.concatEquality.i1 != 1 || p.concatEquality.i2 != 0 || p.concatEquality.low1 != 4 {
			t.Fatal(st, body)
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if !got.IsNull() {
			t.Fatal("DEBUG assertion published output")
		}
	} else {
		astTestMust(t, body)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		n, e := m.GetNumArgs(got)
		astTestMust(t, e)
		if st != Rewrite3 || n != 2 {
			t.Fatal(st, n)
		}
	}
}
