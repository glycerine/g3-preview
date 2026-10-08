// Source: src/util/mpf.cpp:1445-1456.
package util

import (
	"math"
	"testing"
)

func TestArbitraryFloatRemSpecialMatrixAndSharedIdentity(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	values := []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, math.Copysign(0, -1), 3, -5}
	for _, a := range values {
		for _, b := range values {
			for alias := 0; alias < 3; alias++ {
				var x, y, separate ArbitraryFloat
				serviceMust(t, m.SetDouble(&x, 11, 53, a))
				serviceMust(t, m.SetDouble(&y, 11, 53, b))
				out := []*ArbitraryFloat{&separate, &x, &y}[alias]
				serviceMust(t, m.Rem(&x, &y, out))
				got, e := m.ToDouble(out)
				serviceMust(t, e)
				want := math.Remainder(a, b)
				if math.IsNaN(want) {
					if !math.IsNaN(got) || out.Sign() {
						t.Fatal("canonical remainder NaN", a, b, alias, got, out)
					}
				} else if math.Float64bits(got) != math.Float64bits(want) {
					t.Fatal("remainder special", a, b, alias, got, want)
				}
				serviceMust(t, m.Del(&separate))
				serviceMust(t, m.Del(&x))
				serviceMust(t, m.Del(&y))
			}
		}
	}
	for _, sign := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			var x, separate ArbitraryFloat
			serviceMust(t, m.MkOne(11, 53, sign, &x))
			x.exponent = 150
			out := &separate
			if alias {
				out = &x
			}
			serviceMust(t, m.Rem(&x, &x, out))
			zero, e := m.IsZero(out)
			serviceMust(t, e)
			if !zero || out.Sign() != sign {
				t.Fatal("shared input exact signed zero", sign, alias, *out)
			}
			serviceMust(t, m.Del(&separate))
			serviceMust(t, m.Del(&x))
		}
	}
}

func TestArbitraryFloatRemEveryColdSpecialReservation(t *testing.T) {
	for _, pair := range [][2]string{{"nan", "finite"}, {"finite", "nan"}, {"inf", "finite"}, {"inf", "inf"}, {"finite", "inf"}, {"finite", "zero"}, {"zero", "finite"}, {"zero", "zero"}} {
		complete := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			var x, y, out ArbitraryFloat
			for i, v := range []*ArbitraryFloat{&x, &y} {
				*v = ArbitraryFloat{format: arbitraryFloatFormat(17, 120, i == 0), significand: NewInt(0)}
				switch pair[i] {
				case "nan":
					v.exponent = 65536
					v.significand = NewInt(1)
				case "inf":
					v.exponent = 65536
				case "zero":
					v.exponent = -65535
				}
			}
			n := NewInt(1)
			serviceMust(t, m.integer.Mul2kInPlace(&n, 4000))
			serviceMust(t, m.SetComponents(&out, 17, 4001, false, 9, &n))
			serviceMust(t, m.integer.Del(&n))
			oldX, oldY := x, y
			b.FailAfter(ordinal)
			body := m.Rem(&x, &y, &out)
			integerPhysicalAudit(t, m.integer, &x.significand, &y.significand, &out.significand)
			if x != oldX || y != oldY || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
				t.Fatal("early branch physical graph", pair, ordinal, body)
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected special refusal", pair, ordinal, body)
			}
			if m.rem.manager != nil || m.RoundScope.Manager != nil || m.partialRemainder.manager != nil {
				t.Fatal("early branch constructed general descendants", pair)
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.Close())
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("special refund", b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("special reservation horizon", pair)
		}
	}
}
