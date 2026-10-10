// Source: src/ast/rewriter/bv_rewriter.cpp:1362-1406.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorBV2IntEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"numeric", "power65", "power130"} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				q := m.numerals
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				input, e := m.MkVar(0, sort)
				astTestMust(t, e)
				if mode == "numeric" {
					v, e := q.NewValue()
					astTestMust(t, e)
					astTestMust(t, q.SetString(&v, "36893488147419103231"))
					input, e = m.BitVectorNumeral(&v, 65)
					astTestMust(t, e)
					astTestMust(t, q.Del(&v))
				}
				astTestPin(t, m, input)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorBV2IntNumerals(m)
				var size uint32
				if body == nil {
					_, body = n.Read(input, &size)
				}
				var h ASTHandle
				if body == nil {
					if mode == "numeric" {
						h, body = n.Constant(0)
					} else {
						width := uint32(65)
						if mode == "power130" {
							width = 130
						}
						h, body = n.Coefficient(width)
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if mode == "numeric" {
						text, e := q.String(&n.values[0])
						astTestMust(t, e)
						if text != "36893488147419103231" || size != 65 {
							t.Fatal("numeric read", text, size)
						}
					} else {
						text, e := q.String(&n.values[2])
						astTestMust(t, e)
						want := "36893488147419103232"
						if mode == "power130" {
							want = "1361129467683753853853498429727072845824"
						}
						base, e := q.String(&n.values[1])
						astTestMust(t, e)
						if text != want || base != "2" || !n.placed[1] || !n.placed[2] {
							t.Fatal("full expression", text, base, n.placed)
						}
					}
					ref, e := m.NewRef(h)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&ref)) }()
					if mode != "numeric" {
						astTestMust(t, n.End(2, 1))
						if n.placed[1] || n.placed[2] || !n.placed[0] {
							t.Fatal("expression retirement", n.placed)
						}
					}
					astTestMust(t, n.SourceEnd())
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
					sb, ss, _ := n.scalar.Causes()
					ab, as := n.call.bodyCause, n.call.sourceCause
					astTestMust(t, n.CloseExpression())
					if body != nil && !n.placed[0] {
						t.Fatal("expression cleanup ended older v")
					}
					if n.call.frame != nil && !n.call.retired {
						t.Fatal("arithmetic child not joined")
					}
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					xb, xs, _ := n.scalar.Causes()
					if sb != xb || ss != xs || ab != n.call.bodyCause || as != n.call.sourceCause {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual numeric campaign incomplete", mode)
		}
	}
}
