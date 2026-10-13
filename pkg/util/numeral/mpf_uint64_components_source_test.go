// Source: src/util/mpf.cpp:303-316; src/api/api_fpa.cpp:364-377.
package numeral

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatUint64ComponentsSourcePublication(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var out ArbitraryFloat
	for _, tc := range []struct {
		sig  uint64
		sign bool
		exp  int64
	}{
		{0, false, 0},
		{math.MaxInt32, true, -2},
		{math.MaxUint64, false, 17},
	} {
		serviceMust(t, m.SetComponentsUint64(&out, 11, 65, tc.sign, tc.exp, tc.sig))
		got, e := m.integer.Uint64(&out.significand)
		serviceMust(t, e)
		if out.EBits() != 11 || out.SBits() != 65 || out.Sign() != tc.sign || out.Exponent() != tc.exp || got != tc.sig {
			t.Fatal("source scalar component SET", tc, out, got)
		}
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("component owner refund", b.Used())
	}
}

func TestArbitraryFloatUint64ComponentsDebugAssertionAfterPublication(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source SASSERT is disabled")
	}
	b := arena.NewBudget(1 << 25)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var out ArbitraryFloat
	if e := m.SetComponentsUint64(&out, 8, 4, true, 3, 8); e != ErrIntegerAssertion {
		t.Fatal("reached significand assertion", e)
	}
	if out.EBits() != 8 || out.SBits() != 4 || !out.Sign() || out.Exponent() != 3 {
		t.Fatal("source fields before assertion", out)
	}
	got, e := m.integer.Uint64(&out.significand)
	serviceMust(t, e)
	if got != 8 {
		t.Fatal("source integer SET before assertion", got)
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("assertion owner refund", b.Used())
	}
}

func TestArbitraryFloatUint64ComponentsAllocationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		b := arena.NewBudget(1 << 25)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		var out ArbitraryFloat
		var receipt ArbitraryFloatValueInputReceipt
		b.FailAfter(ordinal)
		body := m.SetComponentsUint64WithReceipt(&out, 11, 65, true, 17, math.MaxUint64, &receipt)
		if out.EBits() != 11 || out.SBits() != 65 || !out.Sign() {
			t.Fatal("format/sign must precede integer SET", ordinal, out)
		}
		if body != nil && out.Exponent() != 0 || body == nil && out.Exponent() != 17 {
			t.Fatal("exponent publishes only after successful integer SET", ordinal, body, out)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupValueInput(&receipt, false))
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.CleanupValueInput(&receipt, true))
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("failed scalar SET retained ownership", ordinal, b.Used())
		}
		if body == nil {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatal("no successful scalar SET prefix")
	}
}
