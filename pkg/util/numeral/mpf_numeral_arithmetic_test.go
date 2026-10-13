package numeral

import "testing"

func TestArbitraryFloatNumeralSourceReciprocalIncrementAndPower(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	}()
	var x, out ArbitraryFloat
	defer core.Del(&x)
	defer core.Del(&out)
	check := func(label string, x *ArbitraryFloat, want float64) {
		t.Helper()
		got, err := core.ToDouble(x)
		if err != nil || got != want {
			t.Fatalf("%s: got %v, %v; want %v", label, got, err, want)
		}
	}
	if err := m.SetInt32(&x, 4); err != nil {
		t.Fatal(err)
	}
	if err := m.Inv(&x); err != nil {
		t.Fatal(err)
	}
	check("inverse", &x, 0.25)
	if err := m.Increment(&x); err != nil {
		t.Fatal(err)
	}
	check("increment", &x, 1.25)
	if err := m.Decrement(&x); err != nil {
		t.Fatal(err)
	}
	check("decrement", &x, 0.25)
	if err := m.SetInt32(&x, 2); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		exp  uint32
		want float64
	}{{0, 1}, {1, 2}, {3, 8}, {10, 1024}} {
		if err := m.Power(&x, c.exp, &out); err != nil {
			t.Fatal(c.exp, err)
		}
		check("power", &out, c.want)
	}
	if err := m.Power(&x, 3, &x); err != nil {
		t.Fatal(err)
	}
	check("aliased power", &x, 8)
}
