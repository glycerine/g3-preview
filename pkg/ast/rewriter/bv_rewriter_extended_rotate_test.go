// Source: src/ast/rewriter/bv_rewriter.cpp:2020-2038.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func bvExtendedRotateCall(p *BVRewriter, right bool, a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if right {
		return p.RewriteExtendedRotateRight(a, b, out)
	}
	return p.RewriteExtendedRotateLeft(a, b, out)
}
func TestBVRewriterExtendedRotationOriginalWidthsAndGuards(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 2)
	out, e := m.NewRef(xs[1])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, right := range []bool{false, true} {
		for _, tc := range []struct{ number, width uint32 }{{0, 8}, {1, 8}, {7, 8}, {8, 8}, {255, 8}, {70, 65}, {97, 65}, {255, 257}} {
			shift := tc.number % tc.width
			expectedOut, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			wantStatus, e := bvDerivedRotateCall(p, right, shift, xs[0], expectedOut)
			astTestMust(t, e)
			want, e := m.RefGet(expectedOut)
			astTestMust(t, e)
			st, e := bvExtendedRotateCall(p, right, xs[0], bvAddNum(t, m, tc.number, tc.width), out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != wantStatus || got != want || p.extendedRotate != nil {
				t.Fatal("two source widths/default retirement", right, tc, st, wantStatus)
			}
			astTestMust(t, m.DelRef(&expectedOut))
		}
		before, e := m.RefGet(out)
		astTestMust(t, e)
		st, e := bvExtendedRotateCall(p, right, ast.ASTHandle{}, xs[1], out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteFailed || got != before {
			t.Fatal("symbolic shift read first operand/output")
		}
		st, e = bvExtendedRotateCall(p, right, ast.ASTHandle{}, xs[1], ast.ASTRef{})
		astTestMust(t, e)
		if st != RewriteFailed {
			t.Fatal("failed read nil output")
		}
	}
}
func TestBVRewriterExtendedRotationEveryColdReservation(t *testing.T) {
	for _, right := range []bool{false, true} {
		for _, mode := range []string{"identity", "rotate", "failed"} {
			complete, published := false, false
			for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", right, mode, ordinal), func(t *testing.T) {
					m, _, _, _ := boolRewriterConfigFixture(t)
					_, e := m.RegisterArithmeticPlugin()
					astTestMust(t, e)
					_, e = m.RegisterBitVectorPlugin()
					astTestMust(t, e)
					p := bvNotTestOwner(t, m, false, false)
					xs := bvMergeVars(t, m, 2)
					b := xs[1]
					if mode == "identity" {
						b = bvAddNum(t, m, 97, 65)
					}
					if mode == "rotate" {
						b = bvAddNum(t, m, 70, 65)
					}
					sort, e := m.BitVectorSort(8)
					astTestMust(t, e)
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
					st, body := bvExtendedRotateCall(p, right, xs[0], b, out)
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
						if mode == "identity" {
							want = RewriteDone
						}
						if mode == "failed" {
							want = RewriteFailed
						}
						if st != want || p.extendedRotate != nil {
							t.Fatal("source status/owner", st, want)
						}
						if mode == "failed" && got != old {
							t.Fatal("false publication")
						}
					} else {
						if !numericHoistRefusal(body) {
							t.Fatal(body)
						}
						if got != old {
							published = true
							if st == RewriteFailed {
								t.Fatal("publication status")
							}
						}
						_, replay := bvExtendedRotateCall(p, right, xs[0], b, out)
						if replay != body {
							t.Fatal("stopped replay", replay, body)
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
			if !complete || mode != "failed" && !published {
				t.Fatal("campaign incomplete", right, mode, complete, published)
			}
		}
	}
}
