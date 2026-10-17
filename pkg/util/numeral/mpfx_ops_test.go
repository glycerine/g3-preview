package numeral

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func assertFixedPointException(t *testing.T, err, specific, sibling error, message string) {
	t.Helper()
	if !errors.Is(err, specific) || !errors.Is(err, ErrFixedPointDomain) || errors.Is(err, sibling) {
		t.Fatalf("fixed-point exception inheritance: %v", err)
	}
	var base z3exception.Z3Exception
	if !errors.As(err, &base) || base.Message() != message || base.ErrorCode() != z3exception.ErrorOK || base.HasErrorCode() {
		t.Fatalf("fixed-point z3_exception surface: %v", err)
	}
}

func TestFixedPointRetainedFloorCeilInverseAndPower(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	out, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() { must(a.Close()); must(out.Close()); must(m.Close()) }()
	want := func(v *FixedPoint, expected string) {
		t.Helper()
		r, err := m.Rational(v)
		must(err)
		if got := r.RatString(); got != expected {
			t.Fatalf("fixed value = %s, want %s", got, expected)
		}
	}
	for _, tc := range []struct {
		num         int64
		floor, ceil string
	}{
		{5, "2", "3"}, {-5, "-3", "-2"},
		{1, "0", "1"}, {-1, "-1", "0"},
	} {
		must(m.SetRatio(a, tc.num, 2))
		must(m.Copy(out, a))
		must(m.Floor(out))
		want(out, tc.floor)
		must(m.Copy(out, a))
		must(m.Ceil(out))
		want(out, tc.ceil)
	}
	must(m.SetInt64(a, 3))
	must(m.SetRounding(false))
	must(m.Copy(out, a))
	must(m.Inv(out))
	down, err := m.Rational(out)
	must(err)
	must(m.SetRounding(true))
	must(m.Copy(out, a))
	must(m.Inv(out))
	up, err := m.Rational(out)
	must(err)
	exact := big.NewRat(1, 3)
	if down.Cmp(exact) >= 0 || up.Cmp(exact) <= 0 {
		t.Fatalf("inverse of 3 = [%s,%s]", down, up)
	}
	must(m.SetRatio(a, 3, 2))
	must(m.Power(a, 0, out))
	want(out, "1")
	must(m.Power(a, 3, out))
	want(out, "27/8")
	must(m.Power(a, 9, out))
	want(out, "19683/512")
	must(m.Copy(out, a))
	must(m.Power(out, 3, out))
	want(out, "27/8")
	must(m.Reset(a))
	assertFixedPointException(t, m.Power(a, 0, out), ErrFixedPointDomain, ErrFixedPointOverflow, "multi-precision fixed point (mpfx) exception")
	want(out, "27/8")
	must(m.Power(a, 3, out))
	want(out, "0")
	must(m.SetInt64(a, 2))
	if err := m.Power(a, 1<<31, out); !errors.Is(err, ErrFixedPointDomain) {
		t.Fatalf("nonterminating source mask refused with %v", err)
	}
}

func TestFixedPointPowerAliasExtraSquareAndCeilOverflow(t *testing.T) {
	m, err := NewFixedPointManager(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	out, _ := m.New()
	defer func() {
		if err := a.Close(); err != nil {
			t.Fatal(err)
		}
		if err := out.Close(); err != nil {
			t.Fatal(err)
		}
		if err := m.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := m.SetInt64(a, 256); err != nil {
		t.Fatal(err)
	}
	if err := m.Power(a, 3, out); err != nil {
		t.Fatal(err)
	}
	r, err := m.Rational(out)
	if err != nil || r.RatString() != "16777216" {
		t.Fatalf("distinct-operand cube = %v, %v", r, err)
	}
	assertFixedPointException(t, m.Power(a, 3, a), ErrFixedPointOverflow, ErrFixedPointDivZero, "multi-precision fixed point (mpfx) overflow")
	r, err = m.Rational(a)
	if err != nil || r.RatString() != "16777216" {
		t.Fatalf("source completed cube before extra-square failure: %v, %v", r, err)
	}
	if err := m.SetUint64(a, (1<<32)-1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetRatio(out, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.Add(a, out, out); err != nil {
		t.Fatal(err)
	}
	if err := m.Ceil(out); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("positive ceiling overflow = %v", err)
	}
	r, err = m.Rational(out)
	if err != nil || r.Sign() != 0 {
		t.Fatalf("ceiling overflow did not wrap source words to zero: %v, %v", r, err)
	}
	if err := m.SetUint64(a, (1<<32)-1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetRatio(out, -1, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.Sub(out, a, out); err != nil {
		t.Fatal(err)
	}
	if err := m.Floor(out); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("negative floor overflow = %v", err)
	}
	r, err = m.Rational(out)
	if err != nil || r.Sign() != 0 {
		t.Fatalf("floor overflow did not wrap source words to zero: %v, %v", r, err)
	}
}
