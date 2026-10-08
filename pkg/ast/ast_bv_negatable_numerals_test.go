// Source: src/ast/rewriter/bv_rewriter.cpp:1155-1170.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorNegatableEveryActualReservation(t *testing.T) {
	for _, tc := range []struct{ text, want string }{{"36893488147419103231", "0"}, {"18446744073709551616", "18446744073709551615"}, {"73786976294838206467", "36893488147419103228"}} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				// Retain source raw stored numeral, including the noncanonical value.
				h := bvRawOffsetTestNumeral(t, m, fid, sort, tc.text, 65)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorNegatableNumerals(m)
				var width uint32
				if body == nil {
					_, body = n.Read(h, &width)
				}

				var node ASTHandle
				if body == nil {
					body = n.Complement(width)
				}
				if body == nil {
					node, body = n.Constant(0, width)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					got, e := m.numerals.String(&n.values[0])
					astTestMust(t, e)
					if got != tc.want || n.placed[1] {
						t.Fatal("physical complement/return retirement", got, n.placed)
					}
					ref, e := m.NewRef(node)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&ref)) }()
					astTestMust(t, n.SourceEnd())
				} else {
					var f util.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != util.FailureMemout) {
						t.Fatal(body)
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					sb, ss, _ := n.scalar.Causes()
					nb, ns := n.call.receipt.bodyCause, n.call.receipt.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					xb, xs, _ := n.scalar.Causes()
					if sb != xb || ss != xs || nb != n.call.receipt.bodyCause || ns != n.call.receipt.sourceCause {
						t.Fatal("history changed")
					}
					if n.call.receipt.frame != nil && !n.call.receipt.retired && !n.call.receipt.frame.ownDone {
						t.Fatal("factory child not joined")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual campaign incomplete", tc.text)
		}
	}
}

func TestASTBitVectorNegatableStoredReadAndReturnLifetime(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(65)
	astTestMust(t, e)
	const stored = "73786976294838206467"
	h := bvRawOffsetTestNumeral(t, m, fid, sort, stored, 65)
	n, e := NewBitVectorNegatableNumerals(m)
	astTestMust(t, e)
	var width uint32
	yes, e := n.Read(h, &width)
	astTestMust(t, e)
	before, e := m.numerals.String(&n.values[0])
	astTestMust(t, e)
	if !yes || width != 65 || before != stored {
		t.Fatal("stored value normalized early", before, width)
	}
	astTestMust(t, n.Complement(width))
	node, e := n.Constant(0, width)
	astTestMust(t, e)
	ref, e := m.NewRef(node)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&ref)) }()
	astTestMust(t, n.SourceEnd())
	astTestMust(t, n.Close())
	value, e := m.numerals.NewValue()
	astTestMust(t, e)
	yes, e = m.BitVectorNumeralValue(node, &value, &width)
	astTestMust(t, e)
	got, e := m.numerals.String(&value)
	astTestMust(t, e)
	if !yes || width != 65 || got != "36893488147419103228" {
		t.Fatal("returned AST lost", got, width)
	}
	astTestMust(t, m.numerals.Del(&value))
}
