package interval

import (
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalExtendedIntervalBranches(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b, out := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		check(m.Del(&a))
		check(m.Del(&b))
		check(m.Del(&out))
		check(m.CloseChecked())
		if budget.Used() != 0 {
			t.Fatalf("extended rational owner retained %d bytes", budget.Used())
		}
	})
	check(m.SetString(&a, "-3/2"))
	check(m.SetString(&b, "2"))
	e := RationalExtended{Numbers: m}
	finite, err := m.String(&a)
	check(err)
	assertExtendedShortWrites(t, finite, func(w io.Writer) error { return e.Display(w, &a, Finite) })
	for _, tc := range []struct {
		ak, bk ExtendedKind
		want   int
	}{
		{MinusInfinity, Finite, -1}, {Finite, PlusInfinity, -1},
		{PlusInfinity, MinusInfinity, 1}, {MinusInfinity, MinusInfinity, 0},
		{Finite, Finite, -1},
	} {
		got, err := e.Compare(&a, tc.ak, &b, tc.bk)
		check(err)
		if got != tc.want {
			t.Fatalf("compare kinds (%d,%d) = %d, want %d", tc.ak, tc.bk, got, tc.want)
		}
	}
	if v, err := e.IsNeg(nil, MinusInfinity); err != nil || !v {
		t.Fatalf("negative infinity sign = %v, %v", v, err)
	}
	if v, err := e.IsZero(nil, PlusInfinity); err != nil || v {
		t.Fatalf("positive infinity zero = %v, %v", v, err)
	}
	assert := func(k ExtendedKind, err error, wantKind ExtendedKind, wantNumber string) {
		t.Helper()
		check(err)
		got, err := m.String(&out)
		check(err)
		if k != wantKind || got != wantNumber {
			t.Fatalf("extended result = (%d,%s), want (%d,%s)", k, got, wantKind, wantNumber)
		}
	}
	// Addition and subtraction preserve exact fractions, while infinity
	// results clear the previously owned output numeral.
	k, err := e.Add(&a, Finite, &b, Finite, &out)
	assert(k, err, Finite, "1/2")
	k, err = e.Sub(&a, Finite, &b, MinusInfinity, &out)
	assert(k, err, PlusInfinity, "0")
	k, err = e.Add(&a, PlusInfinity, &b, MinusInfinity, &out)
	if !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("opposite infinities accepted: kind %d, error %v", k, err)
	}
	k, err = e.Mul(&a, Finite, &b, PlusInfinity, &out)
	assert(k, err, MinusInfinity, "0")
	check(m.SetInt32(&a, 0))
	k, err = e.Mul(&a, Finite, nil, Finite, &out)
	assert(k, err, Finite, "0") // source zero short-circuits the second operand
	k, err = e.Mul(&a, Finite, &b, PlusInfinity, &out)
	assert(k, err, Finite, "0")
	check(m.SetString(&a, "-3/2"))
	k, err = e.Div(&a, Finite, &b, PlusInfinity, &out)
	assert(k, err, Finite, "0")
	k, err = e.Div(&a, MinusInfinity, &b, Finite, &out)
	assert(k, err, MinusInfinity, "0")
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	assert(k, err, Finite, "-3/4")
	check(m.SetInt32(&b, 0))
	_, err = e.Div(&a, Finite, &b, Finite, &out)
	if !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("zero divisor accepted: %v", err)
	}
	k, err = e.Inv(&a, Finite)
	check(err)
	if k != Finite {
		t.Fatal("finite inverse changed kind")
	}
	got, err := m.String(&a)
	check(err)
	if got != "-2/3" {
		t.Fatalf("finite inverse = %s", got)
	}
	k, err = e.Power(&a, MinusInfinity, 2)
	if err != nil || k != PlusInfinity {
		t.Fatalf("even negative-infinity power = %d, %v", k, err)
	}
	k, err = e.Power(&a, Finite, 2)
	check(err)
	got, err = m.String(&a)
	check(err)
	if k != Finite || got != "4/9" {
		t.Fatalf("finite square = (%d,%s)", k, got)
	}
	k, err = e.Inv(&a, PlusInfinity)
	check(err)
	got, err = m.String(&a)
	check(err)
	if k != Finite || got != "0" {
		t.Fatalf("infinite inverse = (%d,%s)", k, got)
	}
}

func TestRationalExtendedStructuralEqualityWithoutCrossProducts(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	value := numeral.NewRational()
	left, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		budget.FailAfter(-1)
		for _, e := range []error{left.Close(), right.Close(), m.Del(&value), m.CloseChecked()} {
			if e != nil {
				t.Fatal(e)
			}
		}
		if budget.Used() != 0 {
			t.Fatal("retained interval ownership", budget.Used())
		}
	})
	if err := m.SetString(&value, "123456789012345678901234567890/7"); err != nil {
		t.Fatal(err)
	}
	for _, v := range []*RationalInterval{left, right} {
		if err := v.SetFinite(&value, &value, false, false); err != nil {
			t.Fatal(err)
		}
	}
	e := RationalExtended{Numbers: m}
	budget.FailAfter(0)
	if equal, err := e.Equal(nil, PlusInfinity, nil, PlusInfinity); err != nil || !equal {
		t.Fatal("same infinity kind", equal, err)
	}
	if equal, err := e.Equal(nil, PlusInfinity, nil, Finite); err != nil || equal {
		t.Fatal("finite branch must be skipped", equal, err)
	}
	if err := left.CheckInvariant(); err != nil {
		t.Fatal("equal finite endpoints must avoid cross-products", err)
	}
	if equal, err := left.Equal(right); err != nil || !equal {
		t.Fatal("equal finite intervals must avoid cross-products", equal, err)
	}
}
