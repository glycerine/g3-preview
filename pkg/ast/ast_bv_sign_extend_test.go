// Source: src/ast/rewriter/bv_rewriter.cpp:1532-1540.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorSignExtensionEveryNumericalReservation(t *testing.T) {
	success := false
	failures := 0
	nested := false
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
			matched, body := m.ExtendBitVectorNumeralSign(17, input, out)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f util.Failure
					if !errors.As(body, &f) || f.Kind != util.FailureMemout {
						t.Fatal("unexpected refusal", body)
					}
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if got != input {
					t.Fatal("factory failure published output")
				}
				for f := m.bitVectorPlugin.numeralFrames; f != nil; f = f.next {
					if f.signExtension && f.normalizationChild.frame != nil {
						nested = true
						// A stopped nested numerical constructor is reached before ending
						// the numeral publication and the older function-local rational.
						if f.normalizationChild.sourceCause != nil {
							for _, i := range []int{0} {
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
				if !yes || width != 82 || text != "4835703278458516698824703" {
					t.Fatal("sign-extension result", yes, width, text)
				}
			}
		})
		if success {
			break
		}
	}
	if !success || failures == 0 || !nested {
		t.Fatal("numerical reservation horizon", success, failures, nested)
	}
}

func TestASTBitVectorSignExtensionWidthWrapAfterArithmetic(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(1, 1)
	astTestMust(t, e)
	astTestPin(t, m, input)
	out := astTestPin(t, m, input)
	matched, body := m.ExtendBitVectorNumeralSign(^uint32(0), input, out)
	var exception ASTException
	if !matched || !errors.As(body, &exception) || exception.Error() != "bit-vector size must be greater than zero" {
		t.Fatal("wrapped width", matched, body)
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != input {
		t.Fatal("invalid width published output")
	}
	f := m.bitVectorPlugin.numeralFrames
	for f != nil && !f.signExtension {
		f = f.next
	}
	if f == nil || !f.signExtension || f.normalizationChild.frame == nil || !f.normalizationChild.sourceDone {
		t.Fatal("width rejected before reaching numeral constructor")
	}
	for _, i := range []int{0, 1, 2} {
		if !f.placed[i] || !f.complete[i] || !f.consumed[i] {
			t.Fatal("source arithmetic/full-expression not completed", i)
		}
	}
}

func TestASTBitVectorSignExtensionPublicationAfterNumericalConstructor(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(255, 8)
	astTestMust(t, e)
	astTestPin(t, m, input)
	matched, body := m.ExtendBitVectorNumeralSign(17, input, ASTRef{})
	if !matched || !errors.Is(body, ErrASTHandle) {
		t.Fatal("publication boundary", matched, body)
	}
	f := m.bitVectorPlugin.numeralFrames
	if f == nil || !f.signExtension || !f.normalizationChild.retired || !f.sourceDone || f.retired {
		t.Fatal("missing parent publication owner")
	}
	for _, i := range []int{0, 1, 2} {
		if !f.placed[i] || !f.complete[i] || !f.consumed[i] {
			t.Fatal("local/full-expression DT", i)
		}
	}
}
