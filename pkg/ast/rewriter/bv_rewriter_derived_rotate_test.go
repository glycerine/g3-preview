// Source: src/ast/rewriter/bv_rewriter.cpp:2000-2021.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func bvDerivedRotateCall(p *BVRewriter, right bool, n uint32, arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if right {
		return p.RewriteRotateRight(n, arg, out)
	}
	return p.RewriteRotateLeft(n, arg, out)
}
func TestBVRewriterDerivedFixedRotations(t *testing.T) {
	for _, width := range []uint32{1, 8, 65, 257} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m, _, _, _ := boolRewriterConfigFixture(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			p := bvNotTestOwner(t, m, false, false)
			sort, e := m.BitVectorSort(width)
			astTestMust(t, e)
			x, e := m.MkVar(0, sort)
			astTestMust(t, e)
			astTestPin(t, m, x)
			out, e := m.NewRef(x)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			for _, n := range []uint32{0, 1, width - 1, width, width + 1, ^uint32(0)} {
				for _, right := range []bool{false, true} {
					st, e := bvDerivedRotateCall(p, right, n, x, out)
					astTestMust(t, e)
					got, e := m.RefGet(out)
					astTestMust(t, e)
					shift := n % width
					if shift == 0 || width == 1 {
						if st != RewriteDone || got != x {
							t.Fatal("identity", n, right, st)
						}
						continue
					}
					if right {
						shift = width - shift
					}
					kind, e := m.GetDeclKind(got)
					astTestMust(t, e)
					count, e := m.GetNumArgs(got)
					astTestMust(t, e)
					if st != Rewrite2 || kind != 37 || count != 2 {
						t.Fatal("ordered concat", n, right, st, kind, count)
					}
					for i, bounds := range [][2]uint32{{0, width - shift - 1}, {width - shift, width - 1}} {
						h, e := m.GetArg(got, uint32(i))
						astTestMust(t, e)
						var low, high uint32
						var base ast.ASTHandle
						yes, e := m.BitVectorExtractValue(h, &low, &high, &base)
						astTestMust(t, e)
						if !yes || low != bounds[0] || high != bounds[1] || base != x {
							t.Fatal("original extract order", n, right, i, low, high, bounds)
						}
						if i == 1 {
							decl, e := m.GetDecl(h)
							astTestMust(t, e)
							if p.extract.cached != decl || !p.extract.acquired || p.extract.domain != sort {
								t.Fatal("embedded cache not retained")
							}
						}
					}
					if p.rotationArgs != ([2]ast.ASTHandle{}) {
						t.Fatal("source pair survived return")
					}
				}
			}
			astTestMust(t, m.SetRef(out, x))
			before := m.Stats()
			cached, low, high := p.extract.cached, p.extract.low, p.extract.high
			m.Budget().FailAfter(0)
			_, leftErr := p.RewriteRotateLeft(width, x, out)
			_, rightErr := p.RewriteRotateRight(width, x, out)
			m.Budget().FailAfter(-1)
			astTestMust(t, leftErr)
			astTestMust(t, rightErr)
			if m.Stats() != before || p.extract.cached != cached || p.extract.low != low || p.extract.high != high {
				t.Fatal("identity touched extract cache")
			}
		})
	}
}

func TestBVRewriterDerivedRotationEveryColdReservation(t *testing.T) {
	for _, right := range []bool{false, true} {
		for _, n := range []uint32{0, 3} {
			complete, published, retained := false, false, false
			for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%t/%d/%d", right, n, ordinal), func(t *testing.T) {
					m, _, _, _ := boolRewriterConfigFixture(t)
					_, e := m.RegisterArithmeticPlugin()
					astTestMust(t, e)
					_, e = m.RegisterBitVectorPlugin()
					astTestMust(t, e)
					p := bvNotTestOwner(t, m, false, false)
					sort, e := m.BitVectorSort(65)
					astTestMust(t, e)
					x, e := m.MkVar(0, sort)
					astTestMust(t, e)
					astTestPin(t, m, x)
					var args [32]ast.ASTHandle
					for i := range args {
						args[i], e = m.MkVar(uint32(i+100), sort)
						astTestMust(t, e)
					}
					old, e := m.MkBitVectorOr(args[:])
					astTestMust(t, e)
					out, e := m.NewRef(old)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					m.Budget().FailAfter(ordinal)
					st, body := bvDerivedRotateCall(p, right, n, x, out)
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
						want := Rewrite2
						if n == 0 {
							want = RewriteDone
						}
						if st != want || p.rotationArgs != ([2]ast.ASTHandle{}) {
							t.Fatal("source status/pair", st)
						}
					} else {
						if !numericHoistRefusal(body) {
							t.Fatal(body)
						}
						if !p.rotationArgs[0].IsNull() && p.rotationArgs[1].IsNull() {
							retained = true
						}
						if got != old {
							published = true
							if st == RewriteFailed {
								t.Fatal("publication status")
							}
						}
						_, replay := bvDerivedRotateCall(p, right, n, x, out)
						if replay != body {
							t.Fatal("stopped replay", replay, body)
						}
					}
					cause, extractCause := p.cause, p.extract.failure
					astTestMust(t, p.Close())
					astTestMust(t, p.Close())
					if p.cause != cause || p.extract.failure != extractCause || p.rotationArgs != ([2]ast.ASTHandle{}) {
						t.Fatal("cleanup changed source history")
					}
				})
			}
			// Native argument storage need not fail between the two rotation arguments.
			if !complete || !published {
				t.Fatal("rotation campaign incomplete", right, n, complete, published, retained)
			}
		}
	}
}
