// Source: src/ast/rewriter/bv_rewriter.cpp:1155-1170.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterNegatableOriginalBranches(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(xs[1])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, n := range []uint32{0, 1, 127, 255} {
		yes, e := p.IsNegatable(bvAddNum(t, m, n, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if !yes || got != bvAddNum(t, m, 255-n, 8) {
			t.Fatal("numeral complement", n, yes, got)
		}
	}
	not, e := m.MkBitVectorNot(xs[0])
	astTestMust(t, e)
	astTestPin(t, m, not)
	yes, e := p.IsNegatable(not, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if !yes || got != xs[0] {
		t.Fatal("remove existing not", yes, got)
	}
	yes, e = p.IsNegatable(xs[1], out)
	astTestMust(t, e)
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if yes || got != xs[0] {
		t.Fatal("false changed output")
	}
	yes, e = p.IsNegatable(xs[1], ast.ASTRef{})
	astTestMust(t, e)
	if yes {
		t.Fatal("unread nil output")
	}
}

func TestBVRewriterNegatableEveryColdReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "not", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				c, g, _ := bvPolynomialParamsFixture(t)
				p, e := NewBVRewriter(m, c, g, nil)
				astTestMust(t, e)
				xs := bvMergeVars(t, m, 2)
				arg := xs[0]
				if mode == "numeric" {
					arg = bvAddNum(t, m, 127, 8)
				}
				if mode == "not" {
					arg, e = m.MkBitVectorNot(arg)
					astTestMust(t, e)
					astTestPin(t, m, arg)
				}
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				var children [32]ast.ASTHandle
				for i := range children {
					children[i], e = m.MkVar(uint32(i+100), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(children[:])
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				yes, body := p.IsNegatable(arg, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if yes != (mode != "failed") {
						t.Fatal("match", mode, yes)
					}
					if !yes && got != old {
						t.Fatal("false published")
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
					}
					if _, e = p.IsNegatable(arg, out); e != body {
						t.Fatal("stopped replay", e, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if cause != p.cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete {
			t.Fatal("incomplete campaign", mode)
		}
		if mode != "failed" && !published {
			t.Fatal("old DEC frontier missing", mode)
		}
	}
}

func TestBVRewriterNegatableOriginalNotArityAssertion(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	c, g, _ := bvPolynomialParamsFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 2)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 32, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "raw-nonunary-not"), []ast.ASTHandle{sort, sort}, sort, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	h, e := m.MkApp(d, xs)
	astTestMust(t, e)
	astTestPin(t, m, h)
	out, e := m.NewRef(xs[1])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	yes, body := p.IsNegatable(h, out)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if ast.ASTSourceDebug {
		if body != ast.ErrASTDomain || yes || got != xs[1] {
			t.Fatal("reached DEBUG arity guard", body, yes, got)
		}
	} else {
		astTestMust(t, body)
		if !yes || got != xs[0] {
			t.Fatal("release reads original first child", yes, got)
		}
	}
}
