// Source: src/ast/rewriter/bv_rewriter.cpp:2205-2277.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterBit2BoolOriginalORAndXOR(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	s, e := m.BitVectorSort(1)
	astTestMust(t, e)
	xs := make([]ast.ASTHandle, 3)
	for i := range xs {
		xs[i], e = m.MkVar(uint32(i), s)
		astTestMust(t, e)
		astTestPin(t, m, xs[i])
	}
	one := bvAddNum(t, m, 1, 1)
	zero := bvAddNum(t, m, 0, 1)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, kind := range []int32{31, 33} {
		lhs, e := m.MkFamilyApp(fid, kind, nil, xs)
		astTestMust(t, e)
		astTestPin(t, m, lhs)
		eqs := make([]ast.ASTHandle, len(xs))
		for i, x := range xs {
			eqs[i], e = m.MkEq(x, one)
			astTestMust(t, e)
			astTestPin(t, m, eqs[i])
		}
		var base ast.ASTHandle
		if kind == 31 {
			base, e = m.MkOrArgs(eqs)
		} else {
			base, e = m.MkXorArgs(eqs)
		}
		astTestMust(t, e)
		astTestPin(t, m, base)
		for _, rhs := range []ast.ASTHandle{one, zero} {
			st, e := p.RewriteBit2Bool(lhs, rhs, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			want := base
			status := Rewrite2
			if rhs == zero {
				want, e = m.MkNot(base)
				astTestMust(t, e)
				astTestPin(t, m, want)
				status = Rewrite3
			}
			if st != status || got != want {
				t.Fatal(kind, st, got, want)
			}
		}
	}
}

func TestBVRewriterBit2BoolGuardsNumeralsNOTAndITE(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	s, e := m.BitVectorSort(1)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	zero := bvAddNum(t, m, 0, 1)
	one := bvAddNum(t, m, 1, 1)
	old, e := m.MkTrue()
	astTestMust(t, e)
	astTestPin(t, m, old)
	out, e := m.NewRef(old)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	// The width guard precedes swap and all rhs reads/numerical placement.
	st, e := p.RewriteBit2Bool(bvAddNum(t, m, 1, 8), ast.ASTHandle{}, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || got != old || p.bitBool != nil {
		t.Fatal(st, got)
	}
	for _, pair := range [][2]ast.ASTHandle{{x, y}, {x, one}, {one, x}} {
		st, e = p.RewriteBit2Bool(pair[0], pair[1], out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteFailed || got != old || p.bitBool != nil {
			t.Fatal(st, got)
		}
	}
	for _, pair := range [][2]ast.ASTHandle{{one, one}, {one, zero}, {zero, one}} {
		st, e = p.RewriteBit2Bool(pair[0], pair[1], out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		want, e := m.MkBoolVal(pair[0] == pair[1])
		astTestMust(t, e)
		if st != RewriteDone || got != want {
			t.Fatal(st, got, want)
		}
	}
	not, e := m.MkBitVectorNot(x)
	astTestMust(t, e)
	astTestPin(t, m, not)
	for _, rhs := range []ast.ASTHandle{one, zero} {
		st, e = p.RewriteBit2Bool(not, rhs, out)
		astTestMust(t, e)
		opposite := one
		if rhs == one {
			opposite = zero
		}
		want, e := m.MkEq(x, opposite)
		astTestMust(t, e)
		astTestPin(t, m, want)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != Rewrite1 || got != want {
			t.Fatal(st, got, want)
		}
	}
	cond, e := m.MkEq(x, y)
	astTestMust(t, e)
	astTestPin(t, m, cond)
	ite, e := m.MkIte(cond, x, y)
	astTestMust(t, e)
	astTestPin(t, m, ite)
	st, e = p.RewriteBit2Bool(ite, zero, out)
	astTestMust(t, e)
	left, e := m.MkEq(x, zero)
	astTestMust(t, e)
	right, e := m.MkEq(y, zero)
	astTestMust(t, e)
	want, e := m.MkIte(cond, left, right)
	astTestMust(t, e)
	astTestPin(t, m, want)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if st != Rewrite2 || got != want {
		t.Fatal(st, got, want)
	}
}

func TestBVRewriterBit2BoolEveryColdReservation(t *testing.T) {
	complete := false
	sawSpill := false
	sawIntermediate := false
	for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			s, e := m.BitVectorSort(1)
			astTestMust(t, e)
			xs := make([]ast.ASTHandle, 40)
			for i := range xs {
				xs[i], e = m.MkVar(uint32(i), s)
				astTestMust(t, e)
				astTestPin(t, m, xs[i])
			}
			lhs, e := m.MkBitVectorOr(xs)
			astTestMust(t, e)
			astTestPin(t, m, lhs)
			rhs := bvAddNum(t, m, 0, 1)
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
			st, body := p.RewriteBit2Bool(lhs, rhs, out)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				nativeResult, nativeErr := m.RefGet(out)
				astTestMust(t, nativeErr)
				if nativeResult != old {
					sawIntermediate = true
				}
				if st != Rewrite3 || p.bitBool != nil {
					t.Fatal(st)
				}
			} else {
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				if p.bitBool != nil && len(p.bitBool.args.Slice()) > 16 {
					sawSpill = true
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != old {
					sawIntermediate = true
					if st != Rewrite2 {
						t.Fatal("intermediate status", st)
					}
				}
				cause := p.cause
				if _, e = p.RewriteBit2Bool(lhs, rhs, out); e != cause {
					t.Fatal("replay", e, cause)
				}
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history")
				}
			}
			astTestMust(t, p.Close())
		})
	}
	if !complete || !sawSpill || !sawIntermediate {
		t.Fatal(complete, sawSpill, sawIntermediate)
	}
}

func TestBVRewriterBit2BoolNOTAndITEEveryColdReservation(t *testing.T) {
	for _, branch := range []string{"not", "ite"} {
		complete := false
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", branch, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVPolynomialRewriter(m, c, g, nil)
				astTestMust(t, e)
				s, e := m.BitVectorSort(1)
				astTestMust(t, e)
				x, e := m.MkVar(0, s)
				astTestMust(t, e)
				astTestPin(t, m, x)
				y, e := m.MkVar(1, s)
				astTestMust(t, e)
				astTestPin(t, m, y)
				var lhs ast.ASTHandle
				if branch == "not" {
					lhs, e = m.MkBitVectorNot(x)
				} else {
					condition, err := m.MkEq(x, y)
					astTestMust(t, err)
					lhs, e = m.MkIte(condition, x, y)
				}
				astTestMust(t, e)
				astTestPin(t, m, lhs)
				rhs := bvAddNum(t, m, 0, 1)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteBit2Bool(lhs, rhs, out)
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					want := Rewrite1
					if branch == "ite" {
						want = Rewrite2
					}
					if st != want {
						t.Fatal(st, want)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					cause := p.cause
					if _, e = p.RewriteBit2Bool(lhs, rhs, out); e != cause {
						t.Fatal("stopped replay", e, cause)
					}
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("changed history")
					}
				}
				astTestMust(t, p.Close())
			})
		}
		if !complete {
			t.Fatal(branch, "missing success")
		}
	}
}
