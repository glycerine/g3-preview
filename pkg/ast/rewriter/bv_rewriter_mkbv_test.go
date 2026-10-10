// Source: src/ast/rewriter/bv_rewriter.cpp:2636-2655.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterMkBVOriginalBitOrderAndGuards(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	out, e := m.NewRef(bvAddNum(t, m, 9, 8))
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	args := ast.SliceArguments([]ast.ASTHandle{yes, no, yes, no})
	st, e := p.RewriteMkBV(args, out)
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("default option", st)
	}
	astTestMust(t, p.SetMkBVToNumeral(true))
	st, e = p.RewriteMkBV(args, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 5, 4) {
		t.Fatal("LSB-first bits", st, got)
	}
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	variable, e := m.MkVar(0, boolSort)
	astTestMust(t, e)
	astTestPin(t, m, variable)
	m.Budget().FailAfter(0)
	st, e = p.RewriteMkBV(ast.SliceArguments([]ast.ASTHandle{yes, variable, no}), out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	unchanged, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteFailed || unchanged != got {
		t.Fatal("nonliteral guard", st)
	}
}

func TestBVRewriterMkBVEveryColdReservationAndPublishedResult(t *testing.T) {
	complete, published := false, false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVRewriter(m, c, g, nil)
			astTestMust(t, e)
			astTestMust(t, p.SetMkBVToNumeral(true))
			yes, e := m.MkTrue()
			astTestMust(t, e)
			no, e := m.MkFalse()
			astTestMust(t, e)
			sort, e := m.BitVectorSort(8)
			astTestMust(t, e)
			older := make([]ast.ASTHandle, 32)
			for i := range older {
				older[i], e = m.MkVar(uint32(i+100), sort)
				astTestMust(t, e)
			}
			old, e := m.MkBitVectorOr(older)
			astTestMust(t, e)
			out, e := m.NewRef(old)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.RewriteMkBV(ast.SliceArguments([]ast.ASTHandle{yes, no, yes}), out)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				nativeResult, nativeErr := m.RefGet(out)
				astTestMust(t, nativeErr)
				if nativeResult != old {
					published = true
				}
				if st != RewriteDone {
					t.Fatal(st)
				}
			} else {
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != old {
					published = true
					if st != RewriteDone {
						t.Fatal("published status", st)
					}
				}
				if _, e = p.RewriteMkBV(ast.AppArguments{}, out); e != body {
					t.Fatal("stopped replay", e, body)
				}
			}
			cause := p.cause
			astTestMust(t, p.Close())
			astTestMust(t, p.Close())
			if p.cause != cause {
				t.Fatal("cleanup changed history")
			}
		})
	}
	if !complete || !published {
		t.Fatal("MkBV cold campaign incomplete", complete, published)
	}
}
