package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:1599-1734,1772-1903.

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorBitwiseNumeralsEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"or", "xor", "toggle", "compare", "constant", "mask-constant", "or-mask", "xor-mask"} {
		for _, width := range []uint32{65, 257} {
			complete, failures := false, 0
			for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%s/%d/%d", mode, width, ordinal), func(t *testing.T) {
					m := bvOffsetRuntimeTestManager(t)
					_, e := m.RegisterArithmeticPlugin()
					astTestMust(t, e)
					fid, e := m.RegisterBitVectorPlugin()
					astTestMust(t, e)
					sort, e := m.BitVectorSort(width)
					astTestMust(t, e)
					const raw = "73786976294838206467"
					arg := bvRawOffsetTestNumeral(t, m, fid, sort, raw, width)
					other := bvRawOffsetTestNumeral(t, m, fid, sort, "18446744073709551618", width)
					m.Budget().FailAfter(ordinal)
					n, body := NewBitVectorBitwiseNumerals(m)
					var readWidth uint32
					if body == nil {
						_, body = n.Read(arg, &readWidth)
					}
					if body == nil {
						body = n.Combine(false)
					}
					value, _ := new(big.Int).SetString(raw, 10)
					allone := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
					want := new(big.Int).Set(value)
					var node ASTHandle
					var equal bool
					var bits []bool
					if body == nil {
						switch mode {
						case "or", "xor":
							_, body = n.Read(other, &readWidth)
							if body == nil {
								body = n.Combine(mode == "xor")
							}
							b, _ := new(big.Int).SetString("18446744073709551618", 10)
							if mode == "xor" {
								want.Xor(value, b)
							} else {
								want.Or(value, b)
							}
						case "toggle":
							body = n.ToggleAllOne(width)
							want.Xor(value, allone)
						case "compare":
							equal, body = n.IsAllOne(width)
						case "constant", "mask-constant":
							if mode == "mask-constant" {
								body = n.BeginMask()
							}
							if body == nil {
								body = n.AllOne(width)
							}
							if body == nil {
								node, body = n.Constant(5, width)
							}
						case "or-mask", "xor-mask":
							if mode == "or-mask" {
								body = n.Normalize(width)
								want.And(value, allone)
							}
							if body == nil {
								body = n.BeginMask()
							}
							for bit := uint32(0); bit < width && body == nil; bit++ {
								var one, zero bool
								one, body = n.LowBit(true)
								if body == nil {
									zero, body = n.LowBit(false)
								}
								if body == nil && one == zero {
									t.Fatal("source modulo predicates disagree", bit)
								}
								bits = append(bits, one)
								if body == nil {
									body = n.ShiftMask()
								}
							}
							want.Rsh(want, uint(width))
						}
					}
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						if readWidth != width || !n.placed[0] || !n.placed[1] || n.placed[6] || n.placed[7] || n.placed[8] {
							t.Fatal("default/return lifetimes", readWidth, n.placed)
						}
						got, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if got != want.String() || mode == "compare" && equal != (value.Cmp(allone) == 0) {
							t.Fatal("physical accumulator", got, want, equal)
						}
						for i, one := range bits {
							if one != (value.Bit(i) != 0) {
								t.Fatal("source mask division", i, one)
							}
						}
						if mode == "constant" || mode == "mask-constant" {
							if !n.placed[3] || !n.placed[4] || !n.placed[5] {
								t.Fatal("all-one expression ended before caller publication")
							}
							ref, e := m.NewRef(node)
							astTestMust(t, e)
							defer func() { astTestMust(t, m.DelRef(&ref)) }()
							v, e := m.numerals.NewValue()
							astTestMust(t, e)
							_, e = m.BitVectorNumeralValue(node, &v, &readWidth)
							astTestMust(t, e)
							text, e := m.numerals.String(&v)
							astTestMust(t, e)
							if text != allone.String() {
								t.Fatal("factory all-one", text, allone)
							}
							astTestMust(t, m.numerals.Del(&v))
							astTestMust(t, n.End(5, 4, 3))
							if n.placed[2] != (mode == "mask-constant") {
								t.Fatal("all-one expression ended older mask literal")
							}
							// The repeated run expression must reuse the same older two.
							if mode == "mask-constant" {
								astTestMust(t, n.AllOne(3))
								astTestMust(t, n.End(5, 4, 3))
								if !n.placed[2] {
									t.Fatal("second run ended mask literal")
								}
							}
						}
						astTestMust(t, n.SourceEnd())
					} else {
						failures++
						var f umemory.Failure
						if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
							t.Fatal(body)
						}
						if n != nil {
							if _, e = n.IsZero(); e != ErrASTPending {
								t.Fatal("stopped numerical replay", e)
							}
						}
					}
					if n != nil {
						copy := *n
						if e = copy.Close(); e != ErrASTHandle {
							t.Fatal("copied numerical frame", e)
						}
						b, s, _ := n.scalar.Causes()
						astTestMust(t, n.Close())
						astTestMust(t, n.Close())
						ab, as, _ := n.scalar.Causes()
						if b != ab || s != as {
							t.Fatal("owning cleanup changed history")
						}
					}
				})
			}
			if !complete || failures == 0 {
				t.Fatal("actual reservation campaign incomplete", mode, width, complete, failures)
			}
		}
	}
}

func TestASTBitVectorBitwiseNumeralsStoredWidthAndSymbolicRead(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	// Recognition reports the stored width parameter, not the expression sort.
	arg := bvRawOffsetTestNumeral(t, m, fid, sort, "36893488147419103231", 65)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	ref, e := m.NewRef(x)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&ref)) }()
	n, e := NewBitVectorBitwiseNumerals(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, n.Close()) }()
	width := uint32(123)
	yes, e := n.Read(arg, &width)
	astTestMust(t, e)
	if !yes || width != 65 {
		t.Fatal("stored width", yes, width)
	}
	astTestMust(t, n.Combine(false))
	yes, e = n.IsAllOne(width)
	astTestMust(t, e)
	if !yes || n.placed[3] || n.placed[4] || n.placed[5] {
		t.Fatal("comparison full-expression lifetime", yes, n.placed)
	}
	width = 123
	yes, e = n.Read(x, &width)
	astTestMust(t, e)
	text, e := m.numerals.String(&n.values[1])
	astTestMust(t, e)
	if yes || width != 123 || text != "36893488147419103231" {
		t.Fatal("symbolic recognition modified outputs", yes, width, text)
	}
	astTestMust(t, n.SourceEnd())
}

func TestASTBitVectorBitwiseCloseExpressionPreservesOlderMaskLocals(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	n, e := NewBitVectorBitwiseNumerals(m)
	astTestMust(t, e)
	astTestMust(t, n.BeginMask())
	astTestMust(t, n.AllOne(65))
	astTestMust(t, n.CloseExpression())
	for i, placed := range n.placed {
		if placed != (i < 3) {
			t.Fatal("expression cleanup crossed caller buffer/two boundary", i, placed)
		}
	}
	if n.ended {
		t.Fatal("owning cleanup credited source return")
	}
	if e = n.SourceEnd(); e != ErrASTPending {
		t.Fatal("source return after owning cleanup", e)
	}
	astTestMust(t, n.CloseExpression())
	astTestMust(t, n.Close())
	astTestMust(t, n.Close())
}
