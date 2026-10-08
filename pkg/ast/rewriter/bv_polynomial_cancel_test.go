// Source: src/ast/rewriter/poly_rewriter_def.h:708-883.
package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVPolynomialCancellationOriginalSidesAndMove(t *testing.T) {
	for _, move := range []bool{false, true} {
		t.Run(map[bool]string{false: "keep", true: "move"}[move], func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			xs := bvMergeVars(t, m, 3)
			two := bvAddNum(t, m, 2, 8)
			three := bvAddNum(t, m, 3, 8)
			lhs := bvMulHoistSum(t, m, xs[0], xs[1], two)
			astTestPin(t, m, lhs)
			rhs := bvMulHoistSum(t, m, xs[0], xs[2], three)
			astTestPin(t, m, rhs)
			lo, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&lo)) }()
			ro, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&ro)) }()
			st, e := p.CancelMonomials(lhs, rhs, move, lo, ro)
			astTestMust(t, e)
			l, e := m.RefGet(lo)
			astTestMust(t, e)
			r, e := m.RefGet(ro)
			astTestMust(t, e)
			wantL, wantR := xs[1], bvMulHoistSum(t, m, bvAddNum(t, m, 1, 8), xs[2])
			if move {
				wantL = bvMulHoistSum(t, m, xs[1], bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[2]))
				wantR = bvAddNum(t, m, 1, 8)
			}
			if st != RewriteDone || l != wantL || r != wantR {
				t.Fatal(st, l, r, wantL, wantR)
			}
		})
	}
}

func TestBVPolynomialCancellationDoesNotNormalizeCombinedCoefficientBeforeZero(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	x := bvMergeVars(t, m, 1)[0]
	zero := bvAddNum(t, m, 0, 8)
	lhs := bvMulHoistSum(t, m, bvMergeMul(t, m, bvAddNum(t, m, 255, 8), x), x)
	astTestPin(t, m, lhs)
	rhs := bvMergeMul(t, m, zero, x)
	astTestPin(t, m, rhs)
	lo, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&lo)) }()
	ro, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&ro)) }()
	st, e := p.CancelMonomials(lhs, rhs, false, lo, ro)
	astTestMust(t, e)
	l, e := m.RefGet(lo)
	astTestMust(t, e)
	r, e := m.RefGet(ro)
	astTestMust(t, e)
	if st != RewriteDone || l != rhs || r != zero {
		t.Fatal("source 0*x was simplified", st, l, r, rhs, zero)
	}
}

func TestBVPolynomialCancellationCheapGuardsConstantPlacementAndAliasedOutputs(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 3)
	two := bvAddNum(t, m, 2, 8)
	three := bvAddNum(t, m, 3, 8)
	zero := bvAddNum(t, m, 0, 8)
	lo, e := m.NewRef(xs[2])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&lo)) }()
	ro, e := m.NewRef(xs[2])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&ro)) }()
	for _, row := range []struct {
		lhs, rhs ast.ASTHandle
		move     bool
	}{
		{xs[0], three, true}, {xs[0], three, false}, {xs[0], xs[1], false},
	} {
		astTestMust(t, m.SetRef(lo, xs[2]))
		astTestMust(t, m.SetRef(ro, xs[2]))
		table := p.positions
		size, capacity := table.size, table.capacity
		st, e := p.CancelMonomials(row.lhs, row.rhs, row.move, lo, ro)
		astTestMust(t, e)
		l, e := m.RefGet(lo)
		astTestMust(t, e)
		r, e := m.RefGet(ro)
		astTestMust(t, e)
		if st != RewriteFailed || l != xs[2] || r != xs[2] || p.positions != table || size != table.size || capacity != table.capacity {
			t.Fatal("cheap guard mutated outputs or map", st, l, r)
		}
	}
	for _, row := range []struct{ lhs, rhs, wantL, wantR ast.ASTHandle }{
		{two, three, zero, bvAddNum(t, m, 1, 8)},
		{two, xs[0], two, xs[0]},
		{xs[0], xs[0], zero, zero},
	} {
		st, e := p.CancelMonomials(row.lhs, row.rhs, false, lo, ro)
		astTestMust(t, e)
		l, e := m.RefGet(lo)
		astTestMust(t, e)
		r, e := m.RefGet(ro)
		astTestMust(t, e)
		if st != RewriteDone || l != row.wantL || r != row.wantR {
			t.Fatal(st, l, r, row)
		}
	}
	lhs := bvMulHoistSum(t, m, xs[0], two)
	astTestPin(t, m, lhs)
	rhs := bvMulHoistSum(t, m, xs[0], three)
	astTestPin(t, m, rhs)
	st, e := p.CancelMonomials(lhs, rhs, false, lo, lo)
	astTestMust(t, e)
	l, e := m.RefGet(lo)
	astTestMust(t, e)
	if st != RewriteDone || l != bvAddNum(t, m, 1, 8) {
		t.Fatal("aliased outputs did not retain later RHS assignment", st, l)
	}
}

func TestBVPolynomialCancellationCrossSideMarksAndMoveOnlySorting(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 4)
	two := bvAddNum(t, m, 2, 8)
	lo, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&lo)) }()
	ro, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&ro)) }()
	astTestMust(t, p.SetSortSums(true))
	lhs := bvMulHoistSum(t, m, xs[2], xs[2], xs[1], two)
	astTestPin(t, m, lhs)
	rhs := xs[0]
	st, e := p.CancelMonomials(lhs, rhs, false, lo, ro)
	astTestMust(t, e)
	l, e := m.RefGet(lo)
	astTestMust(t, e)
	want := bvMulHoistSum(t, m, xs[2], xs[2], xs[1])
	if st != RewriteDone || l != want {
		t.Fatal("lhs-only duplicates combined or no-move sorted", st, l, want)
	}
	st, e = p.CancelMonomials(lhs, rhs, true, lo, ro)
	astTestMust(t, e)
	l, e = m.RefGet(lo)
	astTestMust(t, e)
	count, e := m.GetNumArgs(l)
	astTestMust(t, e)
	if st != RewriteDone || count != 4 {
		t.Fatal(st, count)
	}
	fid, e := m.GetFamilyID(l)
	astTestMust(t, e)
	for i := uint32(1); i < count; i++ {
		a, e := m.GetArg(l, i-1)
		astTestMust(t, e)
		b, e := m.GetArg(l, i)
		astTestMust(t, e)
		less, e := bvMonomialLess(m, fid, true, b, a)
		astTestMust(t, e)
		if less {
			t.Fatal("move tail not sorted", i)
		}
	}
}

func TestBVPolynomialCancellationEveryColdReservationAndLeftPublication(t *testing.T) {
	complete := false
	sawLeft := false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			xs := bvMergeVars(t, m, 88)
			two := bvAddNum(t, m, 2, 8)
			three := bvAddNum(t, m, 3, 8)
			la := make([]ast.ASTHandle, 0, 100)
			ra := make([]ast.ASTHandle, 0, 80)
			for i := 0; i < 48; i++ {
				la = append(la, xs[i], xs[i])
				ra = append(ra, xs[i])
			}
			la = append(la, xs[48:68]...)
			la = append(la, two)
			ra = append(ra, xs[68:]...)
			ra = append(ra, three)
			lhs := bvMulHoistSum(t, m, la...)
			astTestPin(t, m, lhs)
			rhs := bvMulHoistSum(t, m, ra...)
			astTestPin(t, m, rhs)
			lo, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&lo)) }()
			ro, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&ro)) }()
			table := p.positions
			m.Budget().FailAfter(ordinal)
			st, body := p.CancelMonomials(lhs, rhs, false, lo, ro)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if st != RewriteDone || p.cancellation != nil || p.positions != table || table.size != 48 {
					t.Fatal("source completion or persistent map", st)
				}
				l, e := m.RefGet(lo)
				astTestMust(t, e)
				r, e := m.RefGet(ro)
				astTestMust(t, e)
				count, e := m.GetNumArgs(l)
				astTestMust(t, e)
				if count != 68 {
					t.Fatal(count)
				}
				for i := uint32(0); i < count; i++ {
					h, e := m.GetArg(l, i)
					astTestMust(t, e)
					if h != xs[i] {
						t.Fatal("first encounter order", i)
					}
				}
				count, e = m.GetNumArgs(r)
				astTestMust(t, e)
				if count != 21 {
					t.Fatal(count)
				}
			} else {
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal(body)
				}
				if n := p.cancellation; n != nil {
					if n.positions != table {
						t.Fatal("fresh map substituted")
					}
					if n.lhsAssigned {
						sawLeft = true
						l, e := m.RefGet(lo)
						astTestMust(t, e)
						r, e := m.RefGet(ro)
						astTestMust(t, e)
						if l.IsNull() || !r.IsNull() {
							t.Fatal("wrong output publication prefix", l, r)
						}
					}
				}
				cause := p.cause
				if _, e = p.CancelMonomials(lhs, rhs, true, lo, ro); e != cause {
					t.Fatal("stopped retry", e, cause)
				}
			}
			cause := p.cause
			astTestMust(t, p.Close())
			if p.cause != cause {
				t.Fatal("history changed")
			}
			one, e := ast.NewASTFastMark1(m)
			astTestMust(t, e)
			markTwo, e := ast.NewASTFastMark2(m)
			astTestMust(t, e)
			for _, x := range xs {
				a, e := one.IsMarked(x)
				astTestMust(t, e)
				b, e := markTwo.IsMarked(x)
				astTestMust(t, e)
				if a || b {
					t.Fatal("unlisted bit survived cleanup")
				}
			}
			astTestMust(t, markTwo.Close())
			astTestMust(t, one.Close())
		})
	}
	if !complete || !sawLeft {
		t.Fatal("missing success or left-before-right failure", complete, sawLeft)
	}
}

func TestBVPolynomialCancellationSparseResetRefusalPreservesActualPrefix(t *testing.T) {
	complete := false
	sawFreed := false
	for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			xs := bvMergeVars(t, m, 48)
			la := make([]ast.ASTHandle, 0, 96)
			for _, x := range xs {
				la = append(la, x, x)
			}
			lo, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&lo)) }()
			ro, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&ro)) }()
			astTestMust(t, p.MakeAddition(ast.SliceArguments(la), lo))
			two := bvAddNum(t, m, 2, 8)
			three := bvAddNum(t, m, 3, 8)
			lhs := bvMulHoistSum(t, m, xs[0], two)
			astTestPin(t, m, lhs)
			rhs := bvMulHoistSum(t, m, xs[0], three)
			astTestPin(t, m, rhs)
			_, e = p.CancelMonomials(lhs, rhs, false, lo, ro)
			astTestMust(t, e)
			table := p.positions
			capacity := table.capacity
			if table.size != 1 || capacity <= 16 {
				t.Fatal("not sparse warm map", table.size, capacity)
			}
			m.Budget().FailAfter(ordinal)
			_, body := p.CancelMonomials(lhs, rhs, false, lo, ro)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if table.capacity != capacity/2 {
					t.Fatal("source sparse reset did not halve", table.capacity, capacity)
				}
			} else {
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal(body)
				}
				if table.stopped {
					sawFreed = true
					if table.entries != nil || table.bytes != 0 || table.capacity != capacity/2 {
						t.Fatal("old table not freed before refused replacement")
					}
				}
			}
			astTestMust(t, p.Close())
		})
	}
	if !complete || !sawFreed {
		t.Fatal("missing sparse reset refusal", complete, sawFreed)
	}
}
