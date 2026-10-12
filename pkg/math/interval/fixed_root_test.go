package interval

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFixedRootRetainedNewtonAndCertifiedBounds(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	a, err := m.New()
	must(err)
	p, err := m.New()
	must(err)
	lo, err := m.New()
	must(err)
	hi, err := m.New()
	must(err)
	defer func() {
		must(a.Close())
		must(p.Close())
		must(lo.Close())
		must(hi.Close())
		must(m.Close())
	}()
	must(m.SetRatio(p, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := FixedRoot{Numbers: m, Limit: limit}
	for _, tc := range []struct {
		value  int64
		degree uint32
	}{
		{0, 3}, {1, 2}, {-1, 3}, {4, 2}, {8, 3}, {32, 5},
		{2, 2}, {2, 3}, {-2, 3}, {1, 5},
	} {
		limit.Steps = 0
		must(m.SetInt64(a, tc.value))
		must(root.Bounds(a, tc.degree, p, lo, hi))
		lower, err := m.Rational(lo)
		must(err)
		upper, err := m.Rational(hi)
		must(err)
		if lower.Cmp(upper) > 0 {
			t.Fatalf("%d^(1/%d) reversed [%s, %s]", tc.value, tc.degree, lower, upper)
		}
		lowerPower, upperPower := big.NewRat(1, 1), big.NewRat(1, 1)
		for i := uint32(0); i < tc.degree; i++ {
			lowerPower.Mul(lowerPower, lower)
			upperPower.Mul(upperPower, upper)
		}
		target := big.NewRat(tc.value, 1)
		if lowerPower.Cmp(target) > 0 || upperPower.Cmp(target) < 0 {
			t.Fatalf("%d^(1/%d) bounds [%s, %s] do not certify target", tc.value, tc.degree, lower, upper)
		}
	}
	for _, tc := range []struct {
		num, den int64
		degree   uint32
	}{
		{1, 2, 2}, {1, 2, 3}, {3, 2, 3}, {-1, 2, 3},
	} {
		limit.Steps = 0
		must(m.SetRatio(a, tc.num, uint64(tc.den)))
		must(root.Bounds(a, tc.degree, p, lo, hi))
		lower, err := m.Rational(lo)
		must(err)
		upper, err := m.Rational(hi)
		must(err)
		lowerPower, upperPower := big.NewRat(1, 1), big.NewRat(1, 1)
		for i := uint32(0); i < tc.degree; i++ {
			lowerPower.Mul(lowerPower, lower)
			upperPower.Mul(upperPower, upper)
		}
		target := big.NewRat(tc.num, tc.den)
		if lowerPower.Cmp(target) > 0 || upperPower.Cmp(target) < 0 {
			t.Fatalf("%s^(1/%d) bounds [%s, %s] do not certify target", target, tc.degree, lower, upper)
		}
	}
	limit.Steps = 0
	must(m.SetRatio(a, 1, 2))
	must(root.Bounds(a, 3, p, lo, hi))
	fallbackLower, err := m.Rational(lo)
	must(err)
	fallbackUpper, err := m.Rational(hi)
	must(err)
	if fallbackLower.Cmp(big.NewRat(0, 1)) != 0 || fallbackUpper.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("subunit fallback = [%s, %s], want [0, 1]", fallbackLower, fallbackUpper)
	}
	must(m.SetInt64(a, -2))
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("even negative root error = %v", err)
	}
	must(m.SetInt64(a, 2))
	must(m.Reset(p))
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatalf("zero precision error = %v", err)
	}
	must(m.SetRatio(p, 1, 8192))
	limit.Limit, limit.Steps = 1, 0
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrFixedRootCanceled) {
		t.Fatalf("checkpoint cancellation error = %v", err)
	}
}
