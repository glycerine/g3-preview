// Source: src/ast/ast_smt2_pp.cpp:277-288; src/util/mpf.h:152-161.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPrinterSourceBiasAndScopedSignificand(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var a ArbitraryFloat
	n := NewInt(13)
	serviceMust(t, m.SetComponents(&a, 8, 24, true, -2, &n))
	bias, e := m.ExponentBias(8)
	serviceMust(t, e)
	if bias != 127 {
		t.Fatal("actual m_powers2.m1", bias)
	}
	var sig Int
	var child ArbitraryFloatValueCopyReceipt
	serviceMust(t, m.CopySignificand(&sig, &a, &child))
	value, e := m.integer.Int64(&sig)
	serviceMust(t, e)
	if value != 13 {
		t.Fatal("source scoped_mpz copy", value)
	}
	serviceMust(t, m.CleanupValueCopy(&child, false))
	serviceMust(t, m.DeleteScopedInteger(&sig))
	serviceMust(t, m.CleanupValueCopy(&child, true))
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("actual printer member owners", b.Used())
	}
}

func TestArbitraryFloatPrinterSignificandEveryReservation(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		b := arena.NewBudget(1 << 26)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		var sig Int
		digits := make([]uint32, 1024)
		for i := range digits {
			digits[i] = 0xffffffff
		}
		digits[len(digits)-1] = 0x7fffffff
		serviceMust(t, m.integer.SetLimbs(&sig, digits))
		var input ArbitraryFloat
		serviceMust(t, m.SetComponents(&input, 11, 40000, false, -42, &sig))
		var out Int
		var child ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.CopySignificand(&out, &input, &child)
		integerPhysicalAudit(t, m.integer, &sig, &input.significand, &out)
		var maps uint64
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal(ordinal, b.Used(), want)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(ordinal, body)
		}
		if child.frame != nil {
			reached = true
			frame, e := m.valueCopyCall(&child)
			serviceMust(t, e)
			if frame.bodyCause != body {
				t.Fatal("exact source child cause", ordinal)
			}
			issued := frame.Issued
			serviceMust(t, m.CleanupValueCopy(&child, false))
			if frame.Issued != issued {
				t.Fatal("false acknowledgment changed tuple", ordinal)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupScopedInteger(&out))
		serviceMust(t, m.CleanupValueCopy(&child, true))
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.integer.Del(&sig))
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if body == nil {
			if !reached {
				t.Fatal("no allocating copy child reached")
			}
			return
		}
	}
	t.Fatal("no successful significand assignment")
}
