package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:1452-1469.

import (
	"fmt"
	"testing"
)

func TestASTBitVectorConcatEveryActualReservationAndPowerLifetime(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 4096 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			v, e := m.numerals.NewValue()
			astTestMust(t, e)
			astTestMust(t, m.numerals.SetString(&v, "36893488147419103231"))
			high, e := m.BitVectorNumeral(&v, 65)
			astTestMust(t, e)
			astTestPin(t, m, high)
			astTestMust(t, m.numerals.Del(&v))
			m.Budget().FailAfter(ordinal)
			n, body := NewBitVectorConcatNumerals(m)
			var width1, width2 uint32
			if body == nil {
				_, body = n.Read(0, high, &width1)
			}
			if body == nil {
				_, body = n.Read(1, high, &width2)
			}
			if body == nil {
				body = n.Fuse(width1)
			}
			var node ASTHandle
			if body == nil {
				node, body = n.Constant(1, width1+width2)
			}
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if n.placed[2] {
					t.Fatal("power survived multiplication full expression")
				}
				first, e := m.numerals.String(&n.values[0])
				astTestMust(t, e)
				result, e := m.numerals.String(&n.values[1])
				astTestMust(t, e)
				if first != "36893488147419103231" || result != "1361129467683753853853498429727072845823" {
					t.Fatal(first, result)
				}
				out, e := m.NewRef(node)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				astTestMust(t, n.SourceEnd())
				for _, v := range n.placed {
					if v {
						t.Fatal("locals not destroyed")
					}
				}
			} else if n != nil {
				if e = n.Fuse(65); e == nil {
					t.Fatal("stopped replay")
				}
			}
			if n != nil {
				copy := *n
				if e = copy.Close(); e != ErrASTHandle {
					t.Fatal("copied ownership", e)
				}
				b, s, _ := n.scalar.Causes()
				astTestMust(t, n.Close())
				astTestMust(t, n.Close())
				afterB, afterS, _ := n.scalar.Causes()
				if b != afterB || s != afterS {
					t.Fatal("cleanup changed history")
				}
			}
		})
	}
	if !complete {
		t.Fatal("actual allocation campaign incomplete")
	}
}
