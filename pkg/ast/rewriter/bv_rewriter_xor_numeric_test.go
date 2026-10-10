// Source: src/ast/rewriter/bv_rewriter.cpp:1736-1903.
package rewriter

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterXOREveryActualNumericAndMaskReservation(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		width uint32
	}{{"numeric", 65}, {"numeric", 257}, {"opposite", 65}, {"mask", 65}, {"single-piece", 65}} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.mode, tc.width, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				width := tc.width
				value := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(width+2)), big.NewInt(3))
				if tc.mode == "mask" {
					for bit := uint32(0); bit < width; bit++ {
						value.SetBit(value, int(bit), uint(1-bit%2))
					}
				}
				if tc.mode == "single-piece" {
					value = new(big.Int).Lsh(big.NewInt(1), uint(width+2))
				}
				a := bvOrRawNumeral(t, m, q, fid, width, value)
				other := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(2))
				args := []ast.ASTHandle{a, bvOrRawNumeral(t, m, q, fid, width, other)}
				var x ast.ASTHandle
				if tc.mode != "numeric" {
					sort, e := m.BitVectorSort(width)
					astTestMust(t, e)
					x, e = m.MkVar(0, sort)
					astTestMust(t, e)
					astTestPin(t, m, x)
					args = []ast.ASTHandle{x, a}
					if tc.mode == "opposite" {
						not, e := m.MkBitVectorNot(x)
						astTestMust(t, e)
						astTestPin(t, m, not)
						args = []ast.ASTHandle{x, not, x}
					}
				}
				p := bvNotTestOwner(t, m, false, false)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteXOR(ast.SliceArguments(args), out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if p.bvxor != nil {
						for _, h := range p.bvxor.pieces {
							refs, e := m.GetRefCount(h)
							astTestMust(t, e)
							if refs == 0 {
								t.Fatal("native mask root lost ownership")
							}
						}
					}
					_, replay := p.RewriteXOR(ast.SliceArguments(args), out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if p.bvxor != nil {
						t.Fatal("source scope retained")
					}
					if tc.mode == "numeric" {
						if st != RewriteDone {
							t.Fatal(st)
						}
						n, e := q.NewValue()
						astTestMust(t, e)
						var bits uint32
						_, e = m.BitVectorNumeralValue(got, &n, &bits)
						astTestMust(t, e)
						text, e := q.String(&n)
						astTestMust(t, e)
						want := new(big.Int).Xor(value, other)
						want.Mod(want, new(big.Int).Lsh(big.NewInt(1), uint(width)))
						if text != want.String() || bits != width {
							t.Fatal("independent XOR value", text, want, bits)
						}
						astTestMust(t, q.Del(&n))
					} else if tc.mode == "opposite" {
						if st != RewriteDone {
							t.Fatal(st)
						}
						child, e := m.GetArg(got, 0)
						astTestMust(t, e)
						kind, e := m.GetDeclKind(got)
						astTestMust(t, e)
						if kind != 32 || child != x {
							t.Fatal("opposite-polarity survivor")
						}
					} else {
						if st != Rewrite3 {
							t.Fatal(st)
						}
						if tc.mode == "single-piece" {
							// Raw high bits make v1 nonzero even when every width bit
							// is zero. Original mask returns one full-width extract.
							lo, e := m.BitVectorExtractLow(got)
							astTestMust(t, e)
							hi, e := m.BitVectorExtractHigh(got)
							astTestMust(t, e)
							child, e := m.GetArg(got, 0)
							astTestMust(t, e)
							if lo != 0 || hi != width-1 || child != x {
								t.Fatal("source unnormalized single extract", lo, hi)
							}
						} else {
							count, e := m.GetNumArgs(got)
							astTestMust(t, e)
							if count != width {
								t.Fatal("mask run count", count)
							}
							for i := uint32(0); i < count; i++ {
								h, e := m.GetArg(got, i)
								astTestMust(t, e)
								bit := width - 1 - i
								lo, e := m.BitVectorExtractLow(h)
								astTestMust(t, e)
								hi, e := m.BitVectorExtractHigh(h)
								astTestMust(t, e)
								child, e := m.GetArg(h, 0)
								astTestMust(t, e)
								if lo != bit || hi != bit {
									t.Fatal("independent run interval", lo, hi, bit)
								}
								if value.Bit(int(bit)) != 0 {
									kind, e := m.GetDeclKind(child)
									astTestMust(t, e)
									term, e := m.GetArg(child, 0)
									astTestMust(t, e)
									if kind != 32 || term != x {
										t.Fatal("mask-one did not extract from owning NOT")
									}
								} else if child != x {
									t.Fatal("mask-zero did not extract original operand")
								}
							}
						}
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
		if !complete || failures == 0 {
			t.Fatal("actual XOR campaign incomplete", tc, complete, failures)
		}
	}
}
