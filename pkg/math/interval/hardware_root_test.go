package interval

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareRootRetainedNewtonCertifiedBoundsAndCancellation(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var a, precision, lo, hi numeral.HardwareFloat
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(m.SetRatioInt32(&precision, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := HardwareRoot{Numbers: m, Limit: limit}
	for _, tc := range []struct {
		num, den int32
		degree   uint32
	}{
		{0, 1, 3}, {1, 1, 2}, {-1, 1, 3}, {4, 1, 2}, {8, 1, 3},
		{32, 1, 5}, {2, 1, 2}, {2, 1, 3}, {-2, 1, 3},
		{1, 2, 2}, {1, 2, 3}, {-1, 2, 3},
	} {
		limit.Steps = 0
		must(m.SetRatioInt32(&a, tc.num, tc.den))
		must(root.Bounds(&a, tc.degree, &precision, &lo, &hi))
		if m.Greater(&lo, &hi) {
			t.Fatalf("reversed bounds for %d/%d degree %d", tc.num, tc.den, tc.degree)
		}
		exact := big.NewRat(int64(tc.num), int64(tc.den))
		lower := new(big.Rat).SetFloat64(m.ToFloat64(&lo))
		upper := new(big.Rat).SetFloat64(m.ToFloat64(&hi))
		lowerPower := new(big.Rat).SetInt64(1)
		upperPower := new(big.Rat).SetInt64(1)
		for i := uint32(0); i < tc.degree; i++ {
			lowerPower.Mul(lowerPower, lower)
			upperPower.Mul(upperPower, upper)
		}
		if lowerPower.Cmp(exact) > 0 || upperPower.Cmp(exact) < 0 {
			t.Fatalf("uncertified root %d/%d degree %d: [%s,%s]", tc.num, tc.den, tc.degree, lower, upper)
		}
	}
	must(m.SetInt32(&a, -2))
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("negative even root", err)
	}
	must(m.SetInt32(&a, 2))
	must(m.Reset(&precision))
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("zero precision", err)
	}
	must(m.SetRatioInt32(&precision, 1, 8192))
	limit.Limit, limit.Steps = 1, 0
	if err := root.Bounds(&a, 2, &precision, &lo, &hi); !errors.Is(err, ErrHardwareRootCanceled) {
		t.Fatal("root checkpoint", err)
	}
}
