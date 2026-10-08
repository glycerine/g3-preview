// Source: src/ast/rewriter/bv_rewriter.cpp:590-666,668-690.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorExtractNumeralsEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		text             string
		width, high, low uint32
	}{
		{"255", 8, 6, 2},
		{"18446744073709551615", 65, 63, 0},
		{"1", 257, 129, 65},
		{"73786976294838206467", 257, 65, 1},
		{"-128", 8, 7, 4},
		{"-3", 257, 256, 64},
	} {
		complete, failures := false, 0
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(tc.width)
				astTestMust(t, e)
				arg := bvRawOffsetTestNumeral(t, m, fid, sort, tc.text, tc.width)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorExtractNumerals(m)
				var width uint32
				if body == nil {
					_, body = n.Read(arg, &width)
				}
				var node ASTHandle
				if body == nil {
					node, body = n.Evaluate(tc.high, tc.low)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if width != tc.width || !n.placed[0] || n.placed[1] {
						t.Fatal("val/power full-expression lifetimes", width, n.placed)
					}
					bits := tc.high - tc.low + 1
					modulus := new(big.Int).Lsh(big.NewInt(1), uint(bits))
					want, _ := new(big.Int).SetString(tc.text, 10)
					if want.Sign() < 0 {
						want.Mod(want, modulus)
					}
					// Source reduces negative input at result width BEFORE shift.
					want.Rsh(want, uint(tc.low))
					want.Mod(want, modulus)
					ref, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&ref)) }()
					astTestMust(t, n.SourceEnd())
					v, e := m.numerals.NewValue()
					astTestMust(t, e)
					_, e = m.BitVectorNumeralValue(node, &v, &width)
					astTestMust(t, e)
					text, e := m.numerals.String(&v)
					astTestMust(t, e)
					if text != want.String() || width != bits {
						t.Fatal("original extraction oracle", text, want, width, bits)
					}
					astTestMust(t, m.numerals.Del(&v))
				} else {
					failures++
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil {
						if _, e = n.Read(arg, &width); e != ErrASTPending {
							t.Fatal("stopped replay", e)
						}
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					b, s, _ := n.scalar.Causes()
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					ab, as, _ := n.scalar.Causes()
					if b != ab || s != as {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete || failures == 0 {
			t.Fatal("actual extraction campaign incomplete", tc, complete, failures)
		}
	}
}

func TestASTBitVectorExtractNumeralsExpressionCleanupRetainsOlderDefault(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	arg := bvRawOffsetTestNumeral(t, m, fid, sort, "255", 8)
	n, e := NewBitVectorExtractNumerals(m)
	astTestMust(t, e)
	var width uint32
	_, e = n.Read(arg, &width)
	astTestMust(t, e)
	_, e = n.Evaluate(6, 2)
	astTestMust(t, e)
	if !n.placed[0] || !n.placed[2] {
		t.Fatal("source local/literal not retained")
	}
	astTestMust(t, n.CloseExpression())
	if !n.placed[0] || n.placed[1] || n.placed[2] {
		t.Fatal("expression cleanup consumed older v", n.placed)
	}
	if e = n.SourceEnd(); e != ErrASTPending {
		t.Fatal("cleanup credited unreached source return", e)
	}
	astTestMust(t, n.Close())
	astTestMust(t, n.Close())
}
