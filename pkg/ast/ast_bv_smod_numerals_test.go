// Source: src/ast/rewriter/bv_rewriter.cpp:1282-1341.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTBitVectorSignedModuloEveryActualReservation(t *testing.T) {
	for _, tc := range []struct{ name, a, b, want string }{
		{"positive", "18446744073709551614", "3", "2"},
		{"negative-positive", "18446744073709551618", "3", "1"},
		{"positive-negative", "18446744073709551614", "36893488147419103229", "36893488147419103231"},
		{"negative-negative", "18446744073709551618", "36893488147419103229", "36893488147419103230"},
		{"minimum", "18446744073709551616", "36893488147419103231", "0"},
	} {
		complete := false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.name, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				numeral := func(text string) ASTHandle {
					v, e := m.numerals.NewValue()
					astTestMust(t, e)
					astTestMust(t, m.numerals.SetString(&v, text))
					h, e := m.BitVectorNumeral(&v, 65)
					astTestMust(t, e)
					astTestPin(t, m, h)
					astTestMust(t, m.numerals.Del(&v))
					return h
				}
				lhs, rhs := numeral(tc.a), numeral(tc.b)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorSignedModuloNumerals(m)
				var width uint32
				if body == nil {
					_, body = n.Read(0, lhs, &width)
				}
				if body == nil {
					body = n.Normalize(0, width)
				}
				if body == nil {
					_, body = n.Read(1, rhs, &width)
				}
				if body == nil {
					body = n.Normalize(1, width)
				}
				if body == nil {
					body = n.Modulo(width)
				}
				var node ASTHandle
				if body == nil {
					node, body = n.Constant(6, width)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if e = n.Modulo(width); e != ErrASTPending {
						t.Fatal("live block overwritten", e)
					}
					got, e := m.numerals.String(&n.values[6])
					astTestMust(t, e)
					if got != tc.want {
						t.Fatal("numeric modulo", got, tc.want)
					}
					for _, i := range []int{2, 7, 8} {
						if n.placed[i] {
							t.Fatal("expression temporary survived", i)
						}
					}
					for _, i := range []int{0, 1, 3, 4, 5, 6} {
						if !n.placed[i] {
							t.Fatal("local ended before publication", i)
						}
					}
					absA, _ := m.numerals.String(&n.values[3])
					absB, _ := m.numerals.String(&n.values[4])
					if absA == "0" || absB == "0" {
						t.Fatal("absolute operands", absA, absB)
					}
					out, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					astTestMust(t, n.SourceEnd())
					for _, placed := range n.placed {
						if placed {
							t.Fatal("source local survived")
						}
					}
				} else {
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil && n.stopped {
						if _, e = n.IsZero(0); e != ErrASTPending {
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
					nb, ns := n.norm.receipt.bodyCause, n.norm.receipt.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					ab, as, _ := n.scalar.Causes()
					if b != ab || s != as || nb != n.norm.receipt.bodyCause || ns != n.norm.receipt.sourceCause {
						t.Fatal("cleanup changed history")
					}
					for _, call := range []*ASTBitVectorNumeralInvocation{&n.call, &n.norm} {
						r := &call.receipt
						if r.frame != nil && !r.retired && !r.frame.ownDone {
							t.Fatal("captured child not joined")
						}
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual modulo campaign incomplete", tc.name)
		}
	}
}
