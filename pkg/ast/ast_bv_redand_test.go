package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:2048-2056.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorReductionANDEveryNumericalReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			q := m.numerals
			n, e := q.NewValue()
			astTestMust(t, e)
			astTestMust(t, q.SetString(&n, "36893488147419103231"))
			input, e := m.BitVectorNumeral(&n, 65)
			astTestMust(t, e)
			astTestMust(t, q.Del(&n))
			astTestPin(t, m, input)
			out := astTestPin(t, m, input)
			m.Budget().FailAfter(ordinal)
			matched, body := m.ReduceBitVectorAND(input, out)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f umemory.Failure
					if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
						t.Fatal("unexpected refusal", body)
					}
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != input {
					t.Fatal("factory failure published output")
				}
				for f := m.bitVectorPlugin.numeralFrames; f != nil; f = f.next {
					if f.allOne && f.normalizationChild.frame != nil {
						// A stopped nested numerical constructor is reached before ending
						// the ternary full-expression and the older function-local rational.
						if f.normalizationChild.sourceCause != nil {
							for _, i := range []int{0, 1, 2, 4} {
								if f.consumed[i] {
									t.Fatal("older source value ended before nested source stop", i)
								}
							}
						}
					}
				}
			} else {
				success = true
				if !matched || m.bitVectorPlugin.numeralFrames != nil {
					t.Fatal("successful reduction not retired")
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				n, e := q.NewValue()
				astTestMust(t, e)
				var width uint32
				yes, e := m.BitVectorNumeralValue(got, &n, &width)
				astTestMust(t, e)
				text, e := q.String(&n)
				astTestMust(t, e)
				astTestMust(t, q.Del(&n))
				if !yes || width != 1 || text != "1" {
					t.Fatal("all-one reduction result", yes, width, text)
				}
			}
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("numerical reservation horizon", success, failures)
	}
}

func TestASTBitVectorReductionANDPublicationAfterNumericalCondition(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(255, 8)
	astTestMust(t, e)
	astTestPin(t, m, input)
	// Invalid output is a checked Go boundary, reached after the source condition
	// and nested numeral constructor, rather than checked at method entry.
	matched, body := m.ReduceBitVectorAND(input, ASTRef{})
	if !matched || !errors.Is(body, ErrASTHandle) {
		t.Fatal("publication boundary", matched, body)
	}
	f := m.bitVectorPlugin.numeralFrames
	if f == nil || !f.allOne || !f.normalizationChild.retired || !f.sourceDone || f.retired {
		t.Fatal("missing reached source numerical/full-expression owner")
	}
	for _, i := range []int{0, 1, 2, 3, 4} {
		if !f.placed[i] || !f.complete[i] || !f.consumed[i] {
			t.Fatal("full-expression/source local DT", i)
		}
	}
}
