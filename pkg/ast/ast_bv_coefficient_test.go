// Source: src/ast/rewriter/bv_rewriter.h:39; src/ast/bv_decl_plugin.cpp:753-765.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorCoefficientNormalizationIndependentModulo(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	value, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&value)) }()
	var call ASTBitVectorNumeralInvocation
	for _, width := range []uint32{1, 8, 32, 65, 129} {
		s, e := m.BitVectorSort(width)
		astTestMust(t, e)
		for _, literal := range []string{"0", "1", "-1", "257", "-257", "340282366920938463463374607431768211457", "-340282366920938463463374607431768211459"} {
			astTestMust(t, q.SetString(&value, literal))
			before := m.Stats()
			astTestMust(t, m.NormalizeBitVectorCoefficientWithInvocation(&value, s, &call))
			astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
			got, e := q.String(&value)
			astTestMust(t, e)
			input, ok := new(big.Int).SetString(literal, 10)
			if !ok {
				t.Fatal("oracle integer")
			}
			modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
			want := new(big.Int).Mod(input, modulus)
			if got != want.String() || m.Stats() != before {
				t.Fatal("unsigned coefficient normalization", width, literal, got, want, m.Stats(), before)
			}
		}
	}
}

func TestASTBitVectorCoefficientSortGuardBeforeNumericalAcquisition(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	before := m.Stats()
	m.Budget().FailAfter(0)
	body := m.NormalizeBitVectorCoefficient(nil, ASTHandle{})
	m.Budget().FailAfter(-1)
	if !errors.Is(body, ErrASTHandle) || m.Stats() != before {
		t.Fatal("sort guard reached numerical work", body)
	}
	if f := &m.bitVectorPlugin.numeralPrimary; f.invocation != 0 {
		t.Fatal("invalid sort placed a numerical frame")
	}
}

func TestASTBitVectorCoefficientEveryActualNormalizationAndAssignmentReservation(t *testing.T) {
	for _, mode := range []struct {
		width     uint32
		warmPower bool
	}{{65, false}, {4097, true}} {
		success, failures, returnedTemporary, assignmentRefusal := false, 0, false, false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			t.Run(fmt.Sprintf("%d/%d", mode.width, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				q := m.numerals
				s, e := m.BitVectorSort(mode.width)
				astTestMust(t, e)
				if mode.warmPower {
					// Populate only the large power cache to reach final copy refusal
					// without sweeping thousands of cache-fill allocations again.
					power, e := q.PowerOfTwo(mode.width)
					astTestMust(t, e)
					astTestMust(t, q.Del(&power))
				}
				value, e := q.NewValue()
				astTestMust(t, e)
				if mode.warmPower {
					// A small receiver cannot reuse storage for this large return;
					// direct payload allocation makes final copy refusal observable.
					astTestMust(t, q.SetInt64(&value, -3))
				} else {
					// Preserve a cold power cache and large negative arithmetic.
					var words [33]uint32
					words[0], words[32] = 3, 3
					astTestMust(t, q.Manager.Z.SetLimbs(&value.Raw.Num, words[:]))
					astTestMust(t, q.Manager.Z.Neg(&value.Raw.Num))
				}
				var call ASTBitVectorNumeralInvocation
				m.Budget().FailAfter(ordinal)
				body := m.NormalizeBitVectorCoefficientWithInvocation(&value, s, &call)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f umemory.Failure
						if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
							t.Fatal("unexpected Runtime refusal", body)
						}
					}
					if f := call.receipt.frame; f != nil {
						if !f.coefficientNormalization || f.invocation == 0 || f.retired {
							t.Fatal("failed assignment frame lost")
						}
						if f.placed[0] {
							returnedTemporary = true
							if !f.normalizationChild.retired {
								t.Fatal("assignment began before norm source exit")
							}
							childBody, _, _ := f.children[0].Causes()
							assignmentRefusal = assignmentRefusal || childBody != nil && !errors.Is(childBody, arena.ErrInvalidSlot)
						}
					}
					if e := m.NormalizeBitVectorCoefficientWithInvocation(&value, s, &call); !errors.Is(e, ErrASTPending) {
						t.Fatal("stopped coefficient call retried", e)
					}
				} else {
					success = true
					got, e := q.String(&value)
					astTestMust(t, e)
					want := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(mode.width)), big.NewInt(3))
					if got != want.String() {
						t.Fatal("large negative unsigned residue", got)
					}
					if f := call.receipt.frame; f == nil || !f.coefficientNormalization || !f.retired || !f.consumed[0] {
						t.Fatal("return temporary survived successful assignment")
					}
				}
				ready := m.BitVectorNumeralInvocationSourceReady(&call)
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				after := m.BitVectorNumeralInvocationSourceReady(&call)
				if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
					t.Fatal("cleanup changed source readiness", ready, after)
				}
				astTestMust(t, q.CleanupValue(&value))
			})
			if success {
				break
			}
		}
		if !success || failures == 0 || mode.warmPower && (!returnedTemporary || !assignmentRefusal) {
			t.Fatal("normalization/assignment horizon", mode.width, success, failures, returnedTemporary, assignmentRefusal)
		}
	}
}

func TestASTBitVectorCoefficientZeroWidthDoesNotUseNumeralFactoryGuard(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	bits, e := m.NewParameterUint(0)
	astTestMust(t, e)
	s := bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{bits})
	astTestMust(t, m.DelParameter(&bits))
	q := m.numerals
	value, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetInt64(&value, 17))
	astTestMust(t, m.NormalizeBitVectorCoefficient(&value, s))
	zero, e := q.IsZero(&value)
	astTestMust(t, e)
	if !zero {
		t.Fatal("norm modulo power-of-two zero should return zero")
	}
	astTestMust(t, q.Del(&value))
}
