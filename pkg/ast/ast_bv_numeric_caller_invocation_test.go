package ast

// Source: src/ast/bv_decl_plugin.cpp:753-765,827-848.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTBitVectorNumericCallerReturnedNormAndInverse(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	value, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&value)) }()
	astTestMust(t, q.SetInt64(&value, -3))
	var norm ASTBitVectorNumeralInvocation
	result, e := m.NormalizeBitVectorValueWithInvocation(&value, 8, &norm)
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&result)) }()
	text, e := q.String(&result)
	astTestMust(t, e)
	if text != "253" {
		t.Fatal(text)
	}
	text, e = q.String(&value)
	astTestMust(t, e)
	if text != "-3" {
		t.Fatal("norm assigned caller", text)
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&norm))
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&norm))
	out, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&out)) }()
	astTestMust(t, q.SetInt64(&value, 17))
	var inverse ASTBitVectorNumeralInvocation
	yes, e := m.BitVectorMultiplicativeInverseWithInvocation(&value, 8, &out, &inverse)
	astTestMust(t, e)
	text, e = q.String(&out)
	astTestMust(t, e)
	if !yes || text != "241" {
		t.Fatal(yes, text)
	}
	astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&inverse))
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&inverse))
}

func TestASTBitVectorNumericCallerExactEveryActualReservation(t *testing.T) {
	for _, operation := range []string{"norm", "inverse"} {
		complete := false
		failures := 0
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", operation, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				q := m.numerals
				value, e := q.NewValue()
				astTestMust(t, e)
				text := "-36893488147419103233"
				want := "36893488147419103231"
				if operation == "inverse" {
					text = "3"
					want = "12297829382473034411"
				}
				astTestMust(t, q.SetString(&value, text))
				output, e := q.NewValue()
				astTestMust(t, e)
				var returned unumeral.OwnedRational
				var call ASTBitVectorNumeralInvocation
				m.Budget().FailAfter(ordinal)
				var body error
				yes := true
				if operation == "norm" {
					returned, body = m.NormalizeBitVectorValueWithInvocation(&value, 65, &call)
				} else {
					yes, body = m.BitVectorMultiplicativeInverseWithInvocation(&value, 65, &output, &call)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if !yes {
						t.Fatal("odd inverse unmatched")
					}
				} else {
					failures++
					var failure umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &failure) || failure.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
				}
				before := m.BitVectorNumeralInvocationSourceReady(&call)
				copied := call
				if operation == "norm" {
					_, e = m.NormalizeBitVectorValueWithInvocation(&value, 65, &copied)
				} else {
					_, e = m.BitVectorMultiplicativeInverseWithInvocation(&value, 65, &output, &copied)
				}
				if !errors.Is(e, ErrASTHandle) {
					t.Fatal("copied context", e)
				}
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				if after := m.BitVectorNumeralInvocationSourceReady(&call); after != before {
					t.Fatal("changed history", before, after)
				}
				if body == nil {
					actual := &output
					if operation == "norm" {
						actual = &returned
					}
					got, e := q.String(actual)
					astTestMust(t, e)
					if got != want {
						t.Fatal(operation, got, want)
					}
				} else {
					if operation == "norm" {
						_, e = m.NormalizeBitVectorValueWithInvocation(&value, 65, &call)
					} else {
						_, e = m.BitVectorMultiplicativeInverseWithInvocation(&value, 65, &output, &call)
					}
					if !errors.Is(e, ErrASTPending) {
						t.Fatal("failed context replay", e)
					}
				}
				original, e := q.String(&value)
				astTestMust(t, e)
				if original != text {
					t.Fatal("input modified", original, text)
				}
				if returned != (unumeral.OwnedRational{}) {
					astTestMust(t, q.CleanupValue(&returned))
				}
				astTestMust(t, q.CleanupValue(&output))
				astTestMust(t, q.Del(&value))
			})
		}
		if !complete || failures == 0 {
			t.Fatal(operation, complete, failures)
		}
	}
}

func TestASTBitVectorNumericCallerOriginalFastPathsAndAliases(t *testing.T) {
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
	astTestMust(t, q.SetInt64(&value, 2))
	yes, e := m.BitVectorMultiplicativeInverseWithInvocation(&value, ^uint32(0), nil, &call)
	astTestMust(t, e)
	if yes {
		t.Fatal("even branch must leave out unread")
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	astTestMust(t, q.SetInt64(&value, 1))
	yes, e = m.BitVectorMultiplicativeInverseWithInvocation(&value, 0, &value, &call)
	astTestMust(t, e)
	if !yes {
		t.Fatal("one zero-width fast path")
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	astTestMust(t, q.SetInt64(&value, 17))
	yes, e = m.BitVectorMultiplicativeInverseWithInvocation(&value, 8, &value, &call)
	astTestMust(t, e)
	text, e := q.String(&value)
	astTestMust(t, e)
	if !yes || text != "241" {
		t.Fatal(yes, text)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	zero, e := m.NormalizeBitVectorValueWithInvocation(&value, 0, &call)
	astTestMust(t, e)
	text, e = q.String(&zero)
	astTestMust(t, e)
	if text != "0" {
		t.Fatal(text)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
	astTestMust(t, q.Del(&zero))
}
