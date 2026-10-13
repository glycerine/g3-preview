package numeral

import (
	"errors"
	"math"
	"testing"
)

func TestHardwareFloatRationalStringMatchesSourceConversion(t *testing.T) {
	qm, _ := rationalManager(t, false)
	core := NewHardwareFloatManager()
	nm, e := NewHardwareFloatNumeralManager(core, 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	for _, bits := range []uint64{
		0, 0x8000000000000000,
		0x0000000000000001, 0x8000000000000001,
		0x000fffffffffffff, 0x0010000000000000,
		math.Float64bits(1), math.Float64bits(-1.5),
		math.Float64bits(0.1), math.Float64bits(math.MaxFloat64),
	} {
		x := HardwareFloat{math.Float64frombits(bits)}
		q := NewRational()
		if e := core.ToRational(&x, qm, &q); e != nil {
			t.Fatal(bits, e)
		}
		want, e := qm.String(&q)
		if e != nil {
			t.Fatal(bits, e)
		}
		got, e := nm.ToRationalString(&x)
		if e != nil || got != want {
			t.Fatalf("bits %#x: %q want %q: %v", bits, got, want, e)
		}
		qm.Del(&q)
	}
}

func TestHardwareFloatRationalStringSourceFiniteGuard(t *testing.T) {
	core := NewHardwareFloatManager()
	x := HardwareFloat{math.Inf(1)}
	got, e := core.ToRationalString(&x)
	if modularSourceDebug {
		if !errors.Is(e, ErrHardwareFloatNonRegular) || got != "" {
			t.Fatal("debug finite assertion", got, e)
		}
	} else if e != nil || got != "0" {
		t.Fatal("release infinity bit path", got, e)
	}
	if _, e := core.ToRationalString(nil); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("nil input", e)
	}
}
