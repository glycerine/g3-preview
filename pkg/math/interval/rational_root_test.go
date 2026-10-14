package interval

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalRootNewtonCertifiedBoundsAndCancellation(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	a, precision, lo, hi := numeral.NewRational(), numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{m.Del(&a), m.Del(&precision), m.Del(&lo), m.Del(&hi), m.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational root retained budget", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	toRat := func(v *numeral.Rational) *big.Rat {
		t.Helper()
		s, err := m.String(v)
		must(err)
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			t.Fatal("invalid exact rational", s)
		}
		return r
	}
	must(m.SetFractionSigned(&precision, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := RationalRoot{Numbers: m, Limit: limit}
	for _, tc := range []struct {
		num, den int32
		degree   uint32
	}{{0, 1, 2}, {1, 1, 5}, {-1, 1, 3}, {4, 1, 2}, {8, 1, 3}, {2, 1, 2}, {2, 1, 3}, {-2, 1, 3}, {1, 2, 2}, {3, 2, 5}} {
		limit.Steps = 0
		must(m.SetFractionSigned(&a, tc.num, tc.den))
		if err := root.Bounds(&a, tc.degree, &precision, &lo, &hi); err != nil {
			t.Fatalf("root %d/%d degree %d: %v", tc.num, tc.den, tc.degree, err)
		}
		lower, upper := toRat(&lo), toRat(&hi)
		if lower.Cmp(upper) > 0 {
			t.Fatal("reversed root enclosure", lower, upper)
		}
		lowerPower, upperPower := big.NewRat(1, 1), big.NewRat(1, 1)
		for i := uint32(0); i < tc.degree; i++ {
			lowerPower.Mul(lowerPower, lower)
			upperPower.Mul(upperPower, upper)
		}
		target := new(big.Rat).SetFrac64(int64(tc.num), int64(tc.den))
		if lowerPower.Cmp(target) > 0 || upperPower.Cmp(target) < 0 {
			t.Fatalf("uncertified root %s^(1/%d): [%s,%s]", target, tc.degree, lower, upper)
		}
	}
	must(m.SetInt32(&a, -2))
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("negative even root", err)
	}
	must(m.SetInt32(&a, 2))
	limit.Limit = 1
	limit.Steps = 1
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrRationalRootCanceled) {
		t.Fatal("root checkpoint", err)
	}
	limit.Limit = 0
	must(m.Reset(&precision))
	must(m.SetInt32(&a, 1))
	must(root.Bounds(&a, 5, &precision, &lo, &hi))
	must(m.SetInt32(&a, 2))
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("nontrivial root requires positive precision", err)
	}
}
