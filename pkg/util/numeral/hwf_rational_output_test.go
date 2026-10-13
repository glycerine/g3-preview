package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestHardwareFloatSourceToRationalBitDecomposition(t *testing.T) {
	qm, _ := rationalManager(t, false)
	m := NewHardwareFloatManager()
	minDen := new(big.Int).Lsh(big.NewInt(1), 1075).String()
	rows := []struct {
		bits uint64
		want string
	}{
		{math.Float64bits(1), "1"},
		{math.Float64bits(1.5), "3/2"},
		{math.Float64bits(-1.5), "-3/2"},
		{math.Float64bits(0), "0"},
		{0x8000000000000000, "0"},
		{0x0000000000000001, "1/" + minDen},
		{0x8000000000000001, "-1/" + minDen},
	}
	for _, row := range rows {
		x := HardwareFloat{math.Float64frombits(row.bits)}
		out := NewRational()
		if e := m.ToRational(&x, qm, &out); e != nil {
			t.Fatal(row.bits, e)
		}
		got, e := qm.String(&out)
		if e != nil || got != row.want {
			t.Fatalf("bits %#x: %q, want %q: %v", row.bits, got, row.want, e)
		}
		qm.Del(&out)
	}
}

func TestHardwareFloatSourceToRationalOwnerGuards(t *testing.T) {
	qm, _ := rationalManager(t, false)
	m := NewHardwareFloatManager()
	x := HardwareFloat{1}
	out := NewRational()
	defer qm.Del(&out)
	if e := m.ToRational(&x, nil, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("missing owner", e)
	}
	if e := m.ToRational(nil, qm, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("missing input", e)
	}
	if e := m.ToRational(&x, qm, &out); e != nil {
		t.Fatal(e)
	}
	got, e := qm.String(&out)
	if e != nil || got != "1" {
		t.Fatal("guard changed output", got, e)
	}
	x.value = math.Inf(1)
	e = m.ToRational(&x, qm, &out)
	if modularSourceDebug {
		if !errors.Is(e, ErrHardwareFloatNonRegular) {
			t.Fatal("source debug finite assertion", e)
		}
		got, e = qm.String(&out)
		if e != nil || got != "1" {
			t.Fatal("debug assertion changed output", got, e)
		}
	} else {
		if e != nil {
			t.Fatal("source release nonfinite bit path", e)
		}
		got, e = qm.String(&out)
		if e != nil || got != "0" {
			t.Fatal("source release infinity projection", got, e)
		}
	}
}
