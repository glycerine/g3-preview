package ast

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestASTFloatingPointBooleanNumeralScopedValue(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, sigValue := range []int32{0, 1, 19} {
		sig := unumeral.NewInt(sigValue)
		var value unumeral.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&value, 8, 24, false, 3, &sig))
		n, e := p.mkNumeral(&value, nil)
		astTestMust(t, e)
		before := m.BudgetField.Used()
		found, e := p.isNumeralOnly(n)
		astTestMust(t, e)
		if !found || p.numeralProbeFrames != nil || m.BudgetField.Used() != before {
			t.Fatal("source boolean numeral and scoped value", found, p.numeralProbeFrames, m.BudgetField.Used(), before)
		}
		astTestMust(t, p.numeric.Del(&value))
	}
	h, e := m.MkTrue()
	astTestMust(t, e)
	n, e := m.internal(h)
	astTestMust(t, e)
	before := m.BudgetField.Used()
	found, e := p.isNumeralOnly(n)
	astTestMust(t, e)
	if found || p.numeralProbeFrames != nil || m.BudgetField.Used() != before {
		t.Fatal("unmatched numeral scoped value", found, p.numeralProbeFrames, m.BudgetField.Used(), before)
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointBooleanNumeralRetainsFailedScalarCopy(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	var sig unumeral.Int
	z, e := p.numeric.BorrowScopedInteger(&sig)
	astTestMust(t, e)
	digits := make([]uint32, 128)
	for i := range digits {
		digits[i] = 0xffffffff
	}
	digits[len(digits)-1] = 0x7fffffff
	astTestMust(t, z.SetLimbs(&sig, digits))
	var source unumeral.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&source, 11, 5000, false, 3, &sig))
	n, e := p.mkNumeral(&source, nil)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	found, body := p.isNumeralOnly(n)
	if body == nil || found || p.numeralProbeFrames == nil || p.numeralProbeFrames.retired {
		t.Fatal("failed source scalar copy did not retain exact owner", found, body, p.numeralProbeFrames)
	}
	m.BudgetField.FailAfter(-1)
	older := p.numeralProbeFrames
	h, e := m.MkTrue()
	astTestMust(t, e)
	other, e := m.internal(h)
	astTestMust(t, e)
	found, e = p.isNumeralOnly(other)
	astTestMust(t, e)
	if found || p.numeralProbeFrames != older || older.retired || older.header != 0 {
		t.Fatal("later query displaced stopped scalar owner", found, p.numeralProbeFrames, older)
	}
	astTestMust(t, p.numeric.Del(&source))
	astTestMust(t, z.Del(&sig))
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, m.Close())
}
