// Source: src/ast/rewriter/bv_rewriter.cpp:1525-1553.
package rewriter

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterSignExtensionOriginalBranches(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	x := bvMergeVars(t, m, 1)[0]
	sign, e := m.MkBitVectorExtract(7, 7, x)
	astTestMust(t, e)
	astTestPin(t, m, sign)
	want, e := m.MkBitVectorConcat([]ast.ASTHandle{sign, sign, sign, x})
	astTestMust(t, e)
	astTestPin(t, m, want)
	for _, tc := range []struct {
		amount    uint32
		arg, want ast.ASTHandle
		st        RewriteStatus
		eliminate bool
	}{
		{0, x, x, RewriteDone, false},
		{3, bvAddNum(t, m, 128, 8), bvAddNum(t, m, 1920, 11), RewriteDone, false},
		{3, x, want, Rewrite2, true},
		{3, x, x, RewriteFailed, false},
	} {
		p.polynomial.config.bitVector.EliminateSignExtension = tc.eliminate
		out, e := m.NewRef(x)
		astTestMust(t, e)
		st, e := p.RewriteSignExtend(tc.amount, tc.arg, out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != tc.st || got != tc.want || p.signExtension != nil {
			t.Fatal("source extension", st, tc.st, got, tc.want)
		}
		astTestMust(t, m.DelRef(&out))
	}
	st, e := p.RewriteSignExtend(3, x, ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("FAILED read output")
	}
	out, e := m.NewRef(x)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	m.Budget().FailAfter(0)
	st, e = p.RewriteSignExtend(0, x, out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteDone {
		t.Fatal("zero identity allocated")
	}
}

func TestBVRewriterSignExtensionEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"numeric", "symbolic-wide", "failed"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				x := bvMergeVars(t, m, 1)[0]
				arg := x
				amount := uint32(40)
				want := Rewrite2
				if mode == "numeric" {
					arg = bvAddNum(t, m, 128, 8)
					want = RewriteDone
				}
				p.polynomial.config.bitVector.EliminateSignExtension = mode != "failed"
				if mode == "failed" {
					want = RewriteFailed
				}
				s, e := m.BitVectorSort(8)
				astTestMust(t, e)
				var oldArgs [32]ast.ASTHandle
				for i := range oldArgs {
					oldArgs[i], e = m.MkVar(uint32(i+100), s)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(oldArgs[:])
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteSignExtend(amount, arg, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					if st != want || p.signExtension != nil || want == RewriteFailed && got != old {
						t.Fatal("source return", st, want)
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if got != old {
						published = true
						if st != want {
							t.Fatal("publication status", st, want)
						}
					}
					_, replay := p.RewriteSignExtend(amount, arg, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history changed")
				}
			})
		}
		if !complete || mode != "failed" && !published {
			t.Fatal("extension campaign incomplete", mode, complete, published)
		}
	}
}

func TestBVRewriterSignExtensionWidthWrapRetainsFailure(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	arg := bvAddNum(t, m, 1, 1)
	out, e := m.NewRef(arg)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	_, body := p.RewriteSignExtend(^uint32(0), arg, out)
	var exception ast.ASTException
	if !errors.As(body, &exception) || exception.Error() != "bit-vector size must be greater than zero" || p.signExtension == nil {
		t.Fatal("source width wrap", body)
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != arg {
		t.Fatal("wrapped factory published output")
	}
	_, replay := p.RewriteSignExtend(1, arg, out)
	if replay != body {
		t.Fatal("stopped width-wrap replay", replay, body)
	}
	astTestMust(t, p.Close())
	astTestMust(t, p.Close())
}
