// Source: src/ast/rewriter/bv_rewriter.cpp:210-242.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestBVRewriterLEOriginalWrappers(t *testing.T) {
	m, _ := bvAddFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 2)
	a, b := xs[0], xs[1]
	ule, e := m.MkBitVectorULE(b, a)
	astTestMust(t, e)
	astTestPin(t, m, ule)
	sle, e := m.MkBitVectorSLE(b, a)
	astTestMust(t, e)
	astTestPin(t, m, sle)
	notU, e := m.MkNot(ule)
	astTestMust(t, e)
	astTestPin(t, m, notU)
	notS, e := m.MkNot(sle)
	astTestMust(t, e)
	astTestPin(t, m, notS)
	for _, tc := range []struct {
		name string
		f    func(ast.ASTHandle, ast.ASTHandle, ast.ASTRef) (RewriteStatus, error)
		want ast.ASTHandle
		st   RewriteStatus
	}{
		{"ule", p.RewriteULE, a, RewriteFailed}, {"sle", p.RewriteSLE, a, RewriteFailed},
		{"uge-fallback", p.RewriteUGE, ule, RewriteDone}, {"sge-fallback", p.RewriteSGE, sle, RewriteDone},
		{"ult", p.RewriteULT, notU, Rewrite2}, {"slt", p.RewriteSLT, notS, Rewrite2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, e := m.NewRef(a)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := tc.f(a, b, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != tc.st || got != tc.want {
				t.Fatal("wrapper reversal/status", st, got)
			}
		})
	}
	// GE must preserve a successful reversed core rewrite.
	zero := bvAddNum(t, m, 0, 8)
	out, e := m.NewRef(a)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteUGE(a, zero, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	if st != RewriteDone || got != yes {
		t.Fatal("reversed core success", st)
	}
}

func TestBVRewriterLEWrappersEveryColdReservationAndPublication(t *testing.T) {
	for _, mode := range []string{"uge-fallback", "sge-fallback", "uge-core", "ult", "slt"} {
		complete, published := false, false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _ := bvAddFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 2)
				a, b := xs[0], xs[1]
				want := RewriteDone
				call := p.RewriteUGE
				switch mode {
				case "sge-fallback":
					call = p.RewriteSGE
				case "uge-core":
					b = bvAddNum(t, m, 0, 8)
				case "ult":
					call = p.RewriteULT
					want = Rewrite2
				case "slt":
					call = p.RewriteSLT
					want = Rewrite2
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
				st, body := call(a, b, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					if st != want || p.leq != nil {
						t.Fatal("source wrapper return", st, want)
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
					_, replay := call(a, b, out)
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
		if !complete || !published {
			t.Fatal("wrapper campaign incomplete", mode, complete, published)
		}
	}
}
