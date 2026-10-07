// Source: src/util/mpf.cpp:1857-1919.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatSpecialNumeralLiteralConstructors(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	var value ArbitraryFloat
	for _, tc := range []struct {
		kind int
		sign bool
		exp  int64
		sig  int32
	}{
		{0, false, -127, 0}, {0, true, -127, 0}, {1, false, 128, 0}, {1, true, 128, 0}, {2, false, 128, (1 << 23) - 1},
	} {
		switch tc.kind {
		case 0:
			e = m.MkZero(8, 24, tc.sign, &value)
		case 1:
			e = m.MkInf(8, 24, tc.sign, &value)
		case 2:
			e = m.MkNaN(8, 24, &value, nil)
		}
		serviceMust(t, e)
		got, e := m.integer.Int64(&value.significand)
		serviceMust(t, e)
		if value.EBits() != 8 || value.SBits() != 24 || value.Sign() != tc.sign || value.exponent != tc.exp || got != int64(tc.sig) {
			t.Fatal("literal special constructor", tc, value, got)
		}
	}
	serviceMust(t, m.Del(&value))
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual owner end", b.Used())
	}
}

func TestArbitraryFloatSpecialNaNEveryActualReservationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 27)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		serviceMust(t, e)
		m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		value := ArbitraryFloat{format: 1 << 31, significand: NewInt(9), exponent: 7}
		var receipt ArbitraryFloatValueCopyReceipt
		b.FailAfter(ordinal)
		call := m.MkNaN(8, 120, &value, &receipt)
		integerPhysicalAudit(t, m.integer, &value.significand)
		if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
			t.Fatal("before-lift actual special owner graph", ordinal, call, b.Used(), m.OwnedStorageBytes(), r.Stats())
		}
		if value.EBits() != 8 || value.SBits() != 120 {
			t.Fatal("literal bitfield prefix", ordinal, value)
		}
		if call == nil {
			completed = true
			if value.Sign() || value.exponent != 128 {
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
