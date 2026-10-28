package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:2687-2698.

import (
	"fmt"
	"testing"
)

func TestASTBitVectorIteEveryActualCopyReservation(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			rhs, e := m.BitVectorNumeralUnsigned(1, 1)
			astTestMust(t, e)
			astTestPin(t, m, rhs)
			number := func(text string) ASTHandle {
				v, e := m.numerals.NewValue()
				astTestMust(t, e)
				astTestMust(t, m.numerals.SetString(&v, text))
				h, e := m.BitVectorNumeral(&v, 65)
				astTestMust(t, e)
				astTestPin(t, m, h)
				astTestMust(t, m.numerals.Del(&v))
				return h
			}
			then := number("36893488147419103231")
			otherwise := number("18446744073709551616")
			m.Budget().FailAfter(ordinal)
			n, body := NewBitVectorIteNumerals(m)
			rw, tw, ew := uint32(0), uint32(0), uint32(0)
			var yes bool
			if body == nil {
				yes, body = n.Read(0, rhs, &rw)
				if body == nil && !yes {
					t.Fatal("rhs")
				}
			}
			if body == nil {
				yes, body = n.Read(2, then, &tw)
				if body == nil && !yes {
					t.Fatal("then")
				}
			}
			if body == nil {
				yes, body = n.Read(1, otherwise, &ew)
				if body == nil && !yes {
					t.Fatal("else")
				}
			}
			m.Budget().FailAfter(-1)
			if body == nil {
				complete = true
				if rw != 1 || tw != 65 || ew != 65 {
					t.Fatal("independent stored widths", rw, tw, ew)
				}
				for i, text := range []string{"1", "18446744073709551616", "36893488147419103231"} {
					got, e := m.numerals.String(&n.values[i])
					astTestMust(t, e)
					if got != text {
						t.Fatal("physical declaration slot", i, got, text)
					}
				}
				astTestMust(t, n.SourceEnd())
				for _, placed := range n.placed {
					if placed {
						t.Fatal("source declaration DT incomplete")
					}
				}
			} else if n != nil {
				if _, e = n.Read(0, rhs, &rw); e == nil {
					t.Fatal("failed copy replayed")
				}
			}
			if n != nil {
				copy := *n
				if e := copy.Close(); e == nil {
					t.Fatal("copied numeric owner accepted")
				}
				b, s, _ := n.scalar.Causes()
				astTestMust(t, n.Close())
				afterB, afterS, _ := n.scalar.Causes()
				if b != afterB || s != afterS {
					t.Fatal("cleanup changed recorded source history")
				}
			}
		})
	}
	if !complete {
		t.Fatal("actual ITE numerical campaign incomplete")
	}
}
