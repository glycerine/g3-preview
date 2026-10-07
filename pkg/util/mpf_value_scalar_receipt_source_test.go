// Source: src/util/mpf.cpp:316-333; src/ast/fpa_decl_plugin.cpp:98-139.
// Source authors: Christoph Wintersteiger; Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatScalarCopyReceiptEveryPrefix(t *testing.T) {
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
		digits[1023] = 0x7fffffff
		serviceMust(t, m.integer.SetLimbs(&sig, digits))
		var input, out ArbitraryFloat
		serviceMust(t, m.SetComponents(&input, 11, 40000, true, -42, &sig))
		var child ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		body := m.CopyWithReceipt(&out, &input, &child)
		integerPhysicalAudit(t, m.integer, &sig, &input.significand, &out.significand)
		var maps uint64
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal(ordinal, b.Used(), want)
		}
		if out.Exponent() != -42 || out.EBits() != 11 || out.SBits() != 40000 || !out.Sign() {
			t.Fatal("source fields before SET", ordinal)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal(body)
		}
		if child.frame != nil {
			q, err := m.valueCopyCall(&child)
			serviceMust(t, err)
			if q.bodyCause != body {
				t.Fatal("original body cause", ordinal)
			}
			reached = true
			before := q.Issued
			serviceMust(t, m.CleanupValueCopy(&child, false))
			if q.Issued != before {
				t.Fatal("false ACK changed issued child", ordinal)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.CleanupValueCopy(&child, true))
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.integer.Del(&sig))
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if body == nil {
			if !reached {
				t.Fatal("no actual allocating child")
			}
			return
		}
	}
	t.Fatal("no successful Copy")
}
