// Source: src/ast/rewriter/bv_rewriter.cpp:1409-1446.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterOriginalLeadingZerosAndOverflowHelpers(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	for _, tc := range []struct{ v, want uint32 }{{0, 8}, {1, 7}, {2, 6}, {127, 1}, {128, 0}, {255, 0}} {
		got, e := p.LeadingZeroBits(bvAddNum(t, m, tc.v, 8))
		astTestMust(t, e)
		if got != tc.want {
			t.Fatal(tc, got)
		}
	}
	got, e := p.LeadingZeroBits(xs[0])
	astTestMust(t, e)
	if got != 0 {
		t.Fatal("symbolic", got)
	}
	for _, tc := range []struct {
		args []ast.ASTHandle
		want uint32
	}{
		{[]ast.ASTHandle{bvAddNum(t, m, 0, 8), bvAddNum(t, m, 1, 8)}, 15},
		{[]ast.ASTHandle{bvAddNum(t, m, 1, 8), bvAddNum(t, m, 0, 8)}, 7},
		{[]ast.ASTHandle{bvAddNum(t, m, 0, 8), bvAddNum(t, m, 0, 8), bvAddNum(t, m, 0, 8)}, 16},
		{[]ast.ASTHandle{xs[0], bvAddNum(t, m, 0, 8)}, 0},
	} {
		h, e := m.MkBitVectorConcat(tc.args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		got, e := p.LeadingZeroBits(h)
		astTestMust(t, e)
		if got != tc.want {
			t.Fatal("source first-two concat", got, tc.want)
		}
	}
	for _, tc := range []struct {
		kind int32
		a, b uint32
		want bool
	}{{6, 3, 3, true}, {6, 15, 15, false}, {6, 0, 255, false}, {4, 0, 127, true}, {4, 1, 128, false}} {
		h, e := m.MkFamilyApp(p.bvFamily, tc.kind, nil, []ast.ASTHandle{bvAddNum(t, m, tc.a, 8), bvAddNum(t, m, tc.b, 8)})
		astTestMust(t, e)
		astTestPin(t, m, h)
		var got bool
		if tc.kind == 6 {
			got, e = p.IsMultiplyWithoutOverflow(h)
		} else {
			got, e = p.IsAddWithoutOverflow(h)
		}
		astTestMust(t, e)
		if got != tc.want {
			t.Fatal("overflow helper", tc, got)
		}
	}
	// The source add criterion checks each operand independently. It remains
	// true for four 64s even though their mathematical sum wraps at width 8.
	four := bvAddNum(t, m, 64, 8)
	sum, e := m.MkFamilyApp(p.bvFamily, 4, nil, []ast.ASTHandle{four, four, four, four})
	astTestMust(t, e)
	astTestPin(t, m, sum)
	if got, e := p.IsAddWithoutOverflow(sum); e != nil || !got {
		t.Fatal("original additive criterion", got, e)
	}

	// Both helper kind guards reject a plain numeral before descending into it.
	if got, e := p.IsMultiplyWithoutOverflow(bvAddNum(t, m, 1, 8)); e != nil || got {
		t.Fatal(got, e)
	}
	if got, e := p.IsAddWithoutOverflow(xs[0]); e != nil || got {
		t.Fatal(got, e)
	}
}

func TestBVRewriterLeadingEveryColdReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "recursive", "short", "multiply", "add"} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				c, g, _ := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				x := bvMergeVars(t, m, 1)[0]
				zero, one := bvAddNum(t, m, 0, 8), bvAddNum(t, m, 1, 8)
				h := one
				if mode == "recursive" || mode == "short" {
					tail := one
					if mode == "short" {
						tail = x
					}
					h, e = m.MkBitVectorConcat([]ast.ASTHandle{zero, tail})
					astTestMust(t, e)
					astTestPin(t, m, h)
				}
				if mode == "multiply" || mode == "add" {
					kind := int32(6)
					if mode == "add" {
						kind = 4
					}
					h, e = m.MkFamilyApp(p.bvFamily, kind, nil, []ast.ASTHandle{one, one})
					astTestMust(t, e)
					astTestPin(t, m, h)
				}
				invoke := func() error {
					if mode == "multiply" {
						_, e := p.IsMultiplyWithoutOverflow(h)
						return e
					}
					if mode == "add" {
						_, e := p.IsAddWithoutOverflow(h)
						return e
					}
					_, e := p.LeadingZeroBits(h)
					return e
				}
				m.Budget().FailAfter(ordinal)
				body := invoke()
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if e := invoke(); e != body {
						t.Fatal("stopped replay", e, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if cause != p.cause {
					t.Fatal("cleanup changed history")
				}
			})
		}
		if !complete {
			t.Fatal("cold campaign incomplete", mode)
		}
	}
}
