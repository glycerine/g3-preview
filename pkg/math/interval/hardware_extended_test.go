package interval

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareExtendedRetainedBranches(t *testing.T) {
	core := numeral.NewHardwareFloatManager()
	m, err := numeral.NewHardwareFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	e := HardwareExtended{Numbers: m}
	var a, b, out numeral.HardwareFloat
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(m.SetInt32(&a, -3))
	must(m.SetInt32(&b, 2))
	check := func(k, wantKind ExtendedKind, want float64) {
		t.Helper()
		if k != wantKind || m.ToFloat64(&out) != want {
			t.Fatalf("extended result kind %v value %v, want %v %v", k, m.ToFloat64(&out), wantKind, want)
		}
	}
	k, err := e.Add(&a, Finite, &b, Finite, &out)
	must(err)
	check(k, Finite, -1)
	k, err = e.Sub(&a, Finite, &b, Finite, &out)
	must(err)
	check(k, Finite, -5)
	k, err = e.Mul(&a, Finite, &b, Finite, &out)
	must(err)
	check(k, Finite, -6)
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	must(err)
	check(k, Finite, -1.5)
	if !e.IsNeg(&a, Finite) || !e.IsPos(&b, Finite) || !e.IsNeg(nil, MinusInfinity) || !e.IsPos(nil, PlusInfinity) {
		t.Fatal("extended signs")
	}
	if e.Compare(&a, Finite, &b, Finite) >= 0 || e.Compare(nil, MinusInfinity, nil, PlusInfinity) >= 0 {
		t.Fatal("extended order")
	}
	if !e.Equal(nil, PlusInfinity, nil, PlusInfinity) || e.Equal(nil, MinusInfinity, nil, PlusInfinity) {
		t.Fatal("infinity equality")
	}
	k, err = e.Add(&a, MinusInfinity, &b, Finite, &out)
	must(err)
	check(k, MinusInfinity, 0)
	if _, err = e.Add(nil, MinusInfinity, nil, PlusInfinity, &out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("opposite infinities", err)
	}
	k, err = e.Mul(&a, MinusInfinity, &b, Finite, &out)
	must(err)
	check(k, MinusInfinity, 0)
	must(m.Reset(&a))
	k, err = e.Mul(&a, Finite, nil, PlusInfinity, &out)
	must(err)
	check(k, Finite, 0)
	if _, err = e.Div(&b, Finite, &a, Finite, &out); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("zero divisor", err)
	}
	k, err = e.Inv(&out, PlusInfinity)
	must(err)
	check(k, Finite, 0)
	k, err = e.Power(nil, MinusInfinity, 2)
	must(err)
	if k != PlusInfinity {
		t.Fatal("even negative infinity power", k)
	}
	must(m.SetInt32(&a, -2))
	k, err = e.Power(&a, Finite, 3)
	must(err)
	if k != Finite || m.ToFloat64(&a) != -8 {
		t.Fatal("finite power", k, m.ToFloat64(&a))
	}
	must(m.SetInt32(&a, 1))
	must(m.SetInt32(&b, 3))
	m.RoundToMinusInf()
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	must(err)
	down := m.ToFloat64(&out)
	m.RoundToPlusInf()
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	must(err)
	up := m.ToFloat64(&out)
	target := big.NewRat(1, 3)
	if k != Finite || new(big.Rat).SetFloat64(down).Cmp(target) >= 0 || new(big.Rat).SetFloat64(up).Cmp(target) <= 0 || up != math.Nextafter(down, math.Inf(1)) {
		t.Fatal("directed third", down, up)
	}
	for _, tc := range []struct {
		kind ExtendedKind
		want string
	}{{MinusInfinity, "-oo"}, {PlusInfinity, "+oo"}} {
		var text bytes.Buffer
		must(e.Display(&text, nil, tc.kind))
		if text.String() != tc.want {
			t.Fatal("infinity display", text.String())
		}
	}
	var finiteText bytes.Buffer
	must(e.Display(&finiteText, &b, Finite))
	wantText, err := m.ToString(&b)
	must(err)
	if finiteText.String() != wantText {
		t.Fatal("finite display", finiteText.String(), wantText)
	}
}
