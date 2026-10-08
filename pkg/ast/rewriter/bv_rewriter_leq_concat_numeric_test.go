// Source: src/ast/rewriter/bv_rewriter.cpp:358-393.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"math/big"
	"testing"
)

func TestBVRewriterLEConcatEveryActualNumericReservation(t *testing.T) {
	for _, mode := range []string{"signed-negative-high", "partial-wide"} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				p := bvNotTestOwner(t, m, false, false)
				var a, b ast.ASTHandle
				var x ast.ASTHandle
				if mode == "signed-negative-high" {
					a = bvPropagationConcat(t, m, bvOrRawNumeral(t, m, q, fid, 257, big.NewInt(-3)), bvPropagationVar(t, m, 0, 8))
					b = bvPropagationConcat(t, m, bvOrRawNumeral(t, m, q, fid, 65, big.NewInt(-2)), bvPropagationVar(t, m, 1, 200))
				} else {
					av := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 70), big.NewInt(3))
					bv := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 68), big.NewInt(1))
					aa := []ast.ASTHandle{bvOrRawNumeral(t, m, q, fid, 65, av)}
					bb := []ast.ASTHandle{bvOrRawNumeral(t, m, q, fid, 64, bv)}
					for i := uint32(0); i < 40; i++ {
						h := bvPropagationVar(t, m, i, 8)
						aa = append(aa, h)
						if i == 0 {
							x = h
							bb = append(bb, bvPropagationVar(t, m, 100, 9))
						} else {
							bb = append(bb, h)
						}
					}
					a = bvPropagationConcat(t, m, aa...)
					b = bvPropagationConcat(t, m, bb...)
				}
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteLEConcat(true, a, b, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					_, replay := p.RewriteLEConcat(true, a, b, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if p.leConcat != nil {
						t.Fatal("source scope retained")
					}
					if mode == "signed-negative-high" {
						no, e := m.MkFalse()
						astTestMust(t, e)
						// floor(-3 / 2^192) == -1, and signed norm65(-2) == -2.
						if st != RewriteDone || got != no {
							t.Fatal("independent negative high comparison", st, got)
						}
					} else {
						kind, e := m.GetDeclKind(got)
						astTestMust(t, e)
						left, e := m.GetArg(got, 0)
						astTestMust(t, e)
						right, e := m.GetArg(got, 1)
						astTestMust(t, e)
						count, e := m.GetNumArgs(left)
						astTestMust(t, e)
						rc, e := m.GetNumArgs(right)
						astTestMust(t, e)
						head, e := m.GetArg(left, 0)
						astTestMust(t, e)
						tail, e := m.GetArg(left, 1)
						astTestMust(t, e)
						n, e := q.NewValue()
						astTestMust(t, e)
						var width uint32
						_, e = m.BitVectorNumeralValue(head, &n, &width)
						astTestMust(t, e)
						text, e := q.String(&n)
						astTestMust(t, e)
						// Both normalized64 high prefixes are one; a retains its raw low bit.
						if st != Rewrite2 || kind != 22 || count != 41 || rc != 40 || width != 1 || text != "1" || tail != x {
							t.Fatal("source partial-prefix shape", st, kind, count, rc, width, text)
						}
						astTestMust(t, q.Del(&n))
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
		if !complete || failures == 0 {
			t.Fatal("actual concat comparison incomplete", mode, complete, failures)
		}
	}
}
