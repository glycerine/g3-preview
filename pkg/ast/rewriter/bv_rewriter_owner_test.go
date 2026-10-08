// Source: src/ast/rewriter/bv_rewriter.h:158-164;
// src/ast/rewriter/bv_rewriter.cpp:26-51,2170-2205.
package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOwnerOriginalConstructionAndConfiguredMultiplication(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "mul2concat"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	key = "bv_sort_ac"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	if p.polynomial.config.passes != 2 || p.extract.header != 0 || p.extract.self != &p.extract || p.trailing.extract != &p.extract {
		t.Fatal("derived member construction order/ownership")
	}
	s, e := p.State()
	astTestMust(t, e)
	if !s.MultiplyToConcat || !s.SortAC {
		t.Fatal(s)
	}
	base, e := p.polynomial.State()
	astTestMust(t, e)
	if !base.SortSums {
		t.Fatal("base sort setting", base)
	}
	x := bvMergeVars(t, m, 1)[0]
	four := bvAddNum(t, m, 4, 8)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteMultiplication(ast.SliceArguments([]ast.ASTHandle{four, x}), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	hi, e := p.extract.Apply(5, 0, x)
	astTestMust(t, e)
	want, e := m.MkBitVectorConcatPair(hi, bvAddNum(t, m, 0, 2))
	astTestMust(t, e)
	astTestPin(t, m, want)
	if st != Rewrite2 || got != want {
		t.Fatal(st, got, want)
	}
	// An ordinary derived update executes the base once before its local suffix.
	key = "mul2concat"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	key = "bv_sort_ac"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	astTestMust(t, p.Update(&local))
	if p.polynomial.config.passes != 3 {
		t.Fatal("base update count")
	}
	s, e = p.State()
	astTestMust(t, e)
	if s.MultiplyToConcat || s.SortAC {
		t.Fatal(s)
	}
	st, e = p.RewriteMultiplication(ast.SliceArguments([]ast.ASTHandle{four, x}), out)
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("configured disabled guard", st)
	}
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != want {
		t.Fatal("FAILED output changed")
	}
	// The same physical extraction helper is borrowed by the trailing engine.
	min, max, e := p.trailing.Count(four)
	astTestMust(t, e)
	if min != 2 || max != 2 {
		t.Fatal(min, max)
	}
}

func TestBVRewriterOwnerEveryConstructorReservation(t *testing.T) {
	for _, channel := range []string{"ast", "params"} {
		complete := false
		for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", channel, ordinal), func(t *testing.T) {
				c, g, paramBudget := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				astBudget := m.Budget()
				budget := astBudget
				if channel == "params" {
					budget = paramBudget
				}
				budget.FailAfter(ordinal)
				p, body := NewBVRewriter(m, c, g, nil)
				budget.FailAfter(-1)
				if body == nil {
					complete = true
					s, e := p.State()
					astTestMust(t, e)
					if !s.HiDiv0 || !s.BitToBool || p.defaultPlaced {
						t.Fatal("constructor suffix incomplete", s)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if p != nil {
						if p.cause != body {
							t.Fatal("lost constructor cause")
						}
						if _, e := p.State(); e != body {
							t.Fatal("stopped state", e, body)
						}
						if e := p.Update(nil); e != body {
							t.Fatal("constructor replay", e, body)
						}
					}
				}
				if p != nil {
					copied := *p
					if e := copied.Close(); e == nil {
						t.Fatal("copied owning header accepted")
					}
					cause := p.cause
					astTestMust(t, p.Close())
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("constructor campaign incomplete", channel)
		}
	}
}

func TestBVRewriterOwnerEveryConfiguredMultiplicationReservation(t *testing.T) {
	complete, sawPublished := false, false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			local, e := c.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, c.Del(&local)) }()
			key := "mul2concat"
			astTestMust(t, c.SetBoolName(&local, &key, true))
			p, e := NewBVRewriter(m, c, g, &local)
			astTestMust(t, e)
			base, trailing := p.polynomial, p.trailing
			x := bvMergeVars(t, m, 1)[0]
			four := bvAddNum(t, m, 4, 8)
			_, _, e = p.trailing.Count(four)
			astTestMust(t, e)
			// Pre-existing cache state must survive the call and close on failure.
			_, e = p.extract.Apply(7, 1, x)
			astTestMust(t, e)
			s, e := m.BitVectorSort(8)
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
			st, body := p.RewriteMultiplication(ast.SliceArguments([]ast.ASTHandle{four, x}), out)
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if st != Rewrite2 {
					t.Fatal(st)
				}
			} else {
				if !numericHoistRefusal(body) {
					t.Fatal(body)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != old {
					sawPublished = true
				}
				if _, e = p.State(); e != body {
					t.Fatal("stopped derived state", e, body)
				}
				if e = p.Update(&local); e != body {
					t.Fatal("stopped derived update", e, body)
				}
				if _, e = p.RewriteAddition(ast.AppArguments{}, out); e != body {
					t.Fatal("stopped cross-operation", e, body)
				}
			}
			cause, baseCause := p.cause, base.cause
			astTestMust(t, p.Close())
			astTestMust(t, p.Close())
			if p.cause != cause || base.cause != baseCause {
				t.Fatal("cleanup changed history")
			}
			if !base.closed || !trailing.closed || !p.extract.closed {
				t.Fatal("persistent members not closed")
			}
		})
	}
	if !complete || !sawPublished {
		t.Fatal("configured campaign incomplete", complete, sawPublished)
	}
}
