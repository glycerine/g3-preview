// Source: src/ast/rewriter/bv_rewriter.cpp:1343-1361.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTBitVectorInt2BVEveryActualReservation(t *testing.T) {
	for _, tc := range []struct{ text, want string }{{"-36893488147419103233", "36893488147419103231"}, {"73786976294838206467", "3"}} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				fid, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				astTestMust(t, m.numerals.SetString(&v, tc.text))
				h, e := m.ArithmeticNumeral(&v, true)
				astTestMust(t, e)
				astTestPin(t, m, h)
				astTestMust(t, m.numerals.Del(&v))
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorInt2BVNumerals(m)
				integer := false
				if body == nil {
					_, body = n.Read(fid, h, &integer)
				}
				if body == nil {
					body = n.Normalize(65)
				}
				var node ASTHandle
				if body == nil {
					node, body = n.Constant(0, 65)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					got, e := m.numerals.String(&n.values[0])
					astTestMust(t, e)
					if !integer || got != tc.want || n.placed[1] {
						t.Fatal("physical normalization", integer, got, n.placed)
					}
					out, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					astTestMust(t, n.SourceEnd())
					if n.placed[0] {
						t.Fatal("val survived source end")
					}
				} else {
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
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
			t.Fatal("actual conversion campaign incomplete")
		}
	}
}
