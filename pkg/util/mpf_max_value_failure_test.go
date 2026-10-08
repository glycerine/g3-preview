// Source: src/util/mpf.cpp:1898-1904.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatMkMaxValueEveryActualReservationPrefix(t *testing.T) {
	for _, sb := range []uint32{2, 24, 120} {
		for _, bigOutput := range []bool{false, true} {
			testArbitraryFloatMkMaxValuePrefix(t, sb, bigOutput)
		}
	}
}
func testArbitraryFloatMkMaxValuePrefix(t *testing.T, sb uint32, bigOutput bool) {
	completed := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 27)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		serviceMust(t, e)
		m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		value := ArbitraryFloat{format: 1 << 31, significand: NewInt(9), exponent: 7}
		if bigOutput {
			serviceMust(t, m.integer.Mul2kInPlace(&value.significand, 200))
		}
		var receipt ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		call := m.MkMaxValue(8, sb, false, &value, &receipt)
		integerPhysicalAudit(t, m.integer, &value.significand)
		if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
			t.Fatal("before-lift actual special owner graph", ordinal, call, b.Used(), m.OwnedStorageBytes(), r.Stats())
		}
		if value.EBits() != 8 || value.SBits() != sb || value.Sign() || value.exponent != 7 && value.exponent != 127 {
			t.Fatal("literal bitfield prefix", ordinal, value)
		}
		if call != nil && !numericPhysicalReservationRefusal(call) {
			t.Fatal("unexpected constructor refusal", ordinal, call)
		}
		if receipt.frame != nil && value.exponent != 127 {
			t.Fatal("exponent must precede COPY", ordinal, value)
		}
		if call == nil {
			completed = true
			if value.Sign() || value.exponent != 127 {
				t.Fatal("literal final special publication", value)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupValueCopy(&receipt, false))
		serviceMust(t, m.CleanupValue(&value))
		serviceMust(t, m.CleanupValueCopy(&receipt, true))
		serviceMust(t, m.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("actual special context end", ordinal, b.Used())
		}
		if completed {
			t.Log("first actual completed special prefix", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("special constructor prefix horizon")
	}
}
