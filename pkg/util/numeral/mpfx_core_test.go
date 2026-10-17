package numeral

import (
	"errors"
	"math/big"
	"testing"
)

func TestFixedPointRetainedArithmeticAndDirectedRounding(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		must(a.Close())
		must(b.Close())
		must(c.Close())
		must(m.Close())
	}()
	want := func(v *FixedPoint, expected string) {
		t.Helper()
		r, err := m.Rational(v)
		must(err)
		if got := r.RatString(); got != expected {
			t.Fatalf("fixed value = %s, want %s", got, expected)
		}
	}
	must(m.SetInt64(a, -3))
	must(m.SetUint64(b, 2))
	must(m.Add(a, b, c))
	want(c, "-1")
	must(m.Sub(b, a, c))
	want(c, "5")
	must(m.Add(a, b, a)) // source permits owned output/input aliases
	want(a, "-1")
	must(m.Mul(a, b, b))
	want(b, "-2")
	must(m.Div(b, a, c))
	want(c, "2")
	must(m.Neg(c))
	want(c, "-2")
	must(m.Abs(c))
	want(c, "2")
	integer, err := m.IsInt(c)
	must(err)
	if !integer {
		t.Fatal("exact integer classified as fractional")
	}
	must(m.SetInt64(a, 1))
	must(m.SetInt64(b, 3))
	must(m.SetRounding(false))
	must(m.Div(a, b, c))
	down, err := m.Rational(c)
	must(err)
	must(m.SetRounding(true))
	must(m.Div(a, b, c))
	up, err := m.Rational(c)
	must(err)
	exact := big.NewRat(1, 3)
	if down.Cmp(exact) >= 0 || up.Cmp(exact) <= 0 {
		t.Fatalf("positive directed division = [%s,%s], exact %s", down, up, exact)
	}
	grid := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 32))
	if new(big.Rat).Sub(up, down).Cmp(grid) != 0 {
		t.Fatalf("directed quotient spans %s, want one grid step", new(big.Rat).Sub(up, down))
	}
	must(m.Neg(a))
	must(m.SetRounding(false))
	must(m.Div(a, b, c))
	down, err = m.Rational(c)
	must(err)
	must(m.SetRounding(true))
	must(m.Div(a, b, c))
	up, err = m.Rational(c)
	must(err)
	exact.Neg(exact)
	if down.Cmp(exact) >= 0 || up.Cmp(exact) <= 0 {
		t.Fatalf("negative directed division = [%s,%s], exact %s", down, up, exact)
	}
	must(m.SetRounding(false))
	must(m.SetRatio(a, 1, 3))
	must(m.SetRatio(b, 1, 3))
	left, err := m.Rational(a)
	must(err)
	right, err := m.Rational(b)
	must(err)
	exact.Mul(left, right)
	must(m.Mul(a, b, c))
	down, err = m.Rational(c)
	must(err)
	must(m.SetRounding(true))
	must(m.Mul(a, b, c))
	up, err = m.Rational(c)
	must(err)
	if down.Cmp(exact) > 0 || up.Cmp(exact) < 0 || new(big.Rat).Sub(up, down).Cmp(grid) > 0 {
		t.Fatalf("directed product = [%s,%s], exact %s", down, up, exact)
	}
	must(m.Neg(a))
	exact.Neg(exact)
	must(m.SetRounding(false))
	must(m.Mul(a, b, c))
	down, err = m.Rational(c)
	must(err)
	must(m.SetRounding(true))
	must(m.Mul(a, b, c))
	up, err = m.Rational(c)
	must(err)
	if down.Cmp(exact) > 0 || up.Cmp(exact) < 0 || new(big.Rat).Sub(up, down).Cmp(grid) > 0 {
		t.Fatalf("negative directed product = [%s,%s], exact %s", down, up, exact)
	}
	must(m.SetRatio(a, 1, 1<<32))
	must(m.SetRatio(b, 1, 1<<32))
	must(m.SetRounding(false))
	must(m.Mul(a, b, c))
	want(c, "0")
	must(m.SetRounding(true))
	must(m.Mul(a, b, c))
	want(c, "1/4294967296")
	integer, err = m.IsInt(a)
	must(err)
	if integer {
		t.Fatal("1/3 approximation classified as integer")
	}
	must(m.Reset(b))
	want(b, "0")
	if err := m.Div(a, b, c); !errors.Is(err, ErrFixedPointDivZero) {
		t.Fatalf("zero divisor error = %v", err)
	}
}

func TestFixedPointRangeOwnershipAndPublication(t *testing.T) {
	m, err := NewFixedPointManager(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	b, _ := m.New()
	c, _ := m.New()
	defer func() {
		if err := a.Close(); err != nil {
			t.Fatal(err)
		}
		if err := b.Close(); err != nil {
			t.Fatal(err)
		}
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		if err := m.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := m.SetUint64(a, 1<<32); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("one-word integer overflow = %v", err)
	}
	if err := m.SetRatio(c, 1, 1<<32); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("source denominator construction overflow = %v", err)
	}
	if err := m.SetUint64(a, (1<<32)-1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(b, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(c, 7); err != nil {
		t.Fatal(err)
	}
	if err := m.Add(a, b, c); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("addition overflow = %v", err)
	}
	r, err := m.Rational(c)
	if err != nil || r.Sign() != 0 {
		t.Fatalf("overflow did not publish wrapped destination words: %v, %v", r, err)
	}
	if zero, err := m.IsZero(c); err != nil || zero {
		t.Fatalf("wrapped allocated zero slot became canonical: %v, %v", zero, err)
	}
	if err := m.Close(); !errors.Is(err, ErrFixedPointOwner) {
		t.Fatalf("manager close with live values = %v", err)
	}
	other, err := NewFixedPointManager(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Copy(c, foreign); !errors.Is(err, ErrFixedPointOwner) {
		t.Fatalf("foreign numeral error = %v", err)
	}
	if err := foreign.Close(); err != nil {
		t.Fatal(err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
}
