// Source: src/ast/rewriter/bv_rewriter.cpp:1567-1734.
package rewriter

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func bvOrRuntimeFixture(t *testing.T) (*ast.ASTManager, *util.RationalService, ast.FamilyID) {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, q.CloseChecked())
		astTestMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual OR Runtime graph retained", b.Used())
		}
	})
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	return m, q, fid
}

func bvOrRawNumeral(t *testing.T, m *ast.ASTManager, q *util.RationalService, fid ast.FamilyID, width uint32, value *big.Int) ast.ASTHandle {
	t.Helper()
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetString(&n, value.String()))
	v, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	w, e := m.NewParameterUint(width)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ast.ASTParameter{v, w}, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	sort, e := m.BitVectorSort(width)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "bv"), nil, sort, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&w))
	astTestMust(t, m.DelParameter(&v))
	astTestMust(t, q.Del(&n))
	h, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

func TestBVRewriterOREveryActualNumericAndMaskReservation(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		width uint32
	}{{"numeric", 65}, {"numeric", 257}, {"mask", 65}, {"concat", 65}} {
		complete, failures, retainedPieces := false, 0, false
		for ordinal := int64(0); ordinal < 2048 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d/%d", tc.mode, tc.width, ordinal), func(t *testing.T) {
				m, q, fid := bvOrRuntimeFixture(t)
				width := tc.width
				value := new(big.Int).Lsh(big.NewInt(1), uint(width+2))
				value.Add(value, big.NewInt(3))
				if tc.mode == "mask" {
					for bit := uint32(0); bit < width; bit++ {
						value.SetBit(value, int(bit), uint(1-bit%2))
					}
				}
				a := bvOrRawNumeral(t, m, q, fid, width, value)
				other := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(2))
				args := []ast.ASTHandle{a, bvOrRawNumeral(t, m, q, fid, width, other)}
				var x ast.ASTHandle
				if tc.mode == "mask" || tc.mode == "concat" {
					bits := width
					if tc.mode == "concat" {
						bits--
					}
					sort, e := m.BitVectorSort(bits)
					astTestMust(t, e)
					x, e = m.MkVar(0, sort)
					astTestMust(t, e)
					astTestPin(t, m, x)
					args = []ast.ASTHandle{x, a}
					if tc.mode == "concat" {
						left, e := m.MkBitVectorConcat([]ast.ASTHandle{x, bvAddNum(t, m, 0, 1)})
						astTestMust(t, e)
						astTestPin(t, m, left)
						right, e := m.MkBitVectorConcat([]ast.ASTHandle{bvAddNum(t, m, 0, width-1), bvAddNum(t, m, 1, 1)})
						astTestMust(t, e)
						astTestPin(t, m, right)
						args = []ast.ASTHandle{left, right}
					}
				}
				p := bvNotTestOwner(t, m, false, false)
				out, e := m.NewRef(ast.ASTHandle{})
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := p.RewriteOR(ast.SliceArguments(args), out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if p.bvor != nil && len(p.bvor.pieces) >= 16 {
						retainedPieces = true
					}
					_, replay := p.RewriteOR(ast.SliceArguments(args), out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				} else {
					complete = true
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if p.bvor != nil {
						t.Fatal("source scope retained")
					}
					if tc.mode == "numeric" {
						if st != RewriteDone {
							t.Fatal(st)
						}
						n, e := q.NewValue()
						astTestMust(t, e)
						var actualWidth uint32
						_, e = m.BitVectorNumeralValue(got, &n, &actualWidth)
						astTestMust(t, e)
						text, e := q.String(&n)
						astTestMust(t, e)
						want := new(big.Int).Or(value, other)
						want.Mod(want, new(big.Int).Lsh(big.NewInt(1), uint(width)))
						if text != want.String() || actualWidth != width {
							t.Fatal("original numeral folding", text, want, actualWidth)
						}
						astTestMust(t, q.Del(&n))
					} else {
						if st != Rewrite2 {
							t.Fatal(st)
						}
						count, e := m.GetNumArgs(got)
						astTestMust(t, e)
						if tc.mode == "concat" && count != 2 || tc.mode == "mask" && count != width {
							t.Fatal("ordered output pieces", count)
						}
						if tc.mode == "mask" {
							for i := uint32(0); i < count; i++ {
								h, e := m.GetArg(got, i)
								astTestMust(t, e)
								bit := width - 1 - i
								numeric, e := m.IsBitVectorNumeral(h)
								astTestMust(t, e)
								if numeric != (value.Bit(int(bit)) == 1) {
									t.Fatal("independent mask bit", bit, numeric)
								}
								if numeric {
									n, e := q.NewValue()
									astTestMust(t, e)
									var bits uint32
									_, e = m.BitVectorNumeralValue(h, &n, &bits)
									astTestMust(t, e)
									one, e := q.IsOne(&n)
									astTestMust(t, e)
									if !one || bits != 1 {
										t.Fatal("all-one run literal", bits, one)
									}
									astTestMust(t, q.Del(&n))
								} else {
									lo, e := m.BitVectorExtractLow(h)
									astTestMust(t, e)
									hi, e := m.BitVectorExtractHigh(h)
									astTestMust(t, e)
									child, e := m.GetArg(h, 0)
									astTestMust(t, e)
									if lo != bit || hi != bit || child != x {
										t.Fatal("independent extract interval", lo, hi, bit)
									}
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
		if !complete || failures == 0 || tc.mode == "mask" && !retainedPieces {
			t.Fatal("actual OR campaign incomplete", tc, complete, failures, retainedPieces)
		}
	}
}
