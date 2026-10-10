// Source: src/util/mpf.cpp:756-801 (early branches and actual callees).
package numeral

import (
	"math"
	"testing"
)

func TestArbitraryFloatFMAEveryColdSpecialReservation(t *testing.T) {
	for _, pair := range [][2]string{{"nan", "finite"}, {"finite", "nan"}, {"inf", "finite"}, {"finite", "inf"}, {"inf", "inf"}, {"zero", "zero"}, {"zero", "finite"}, {"finite", "zero"}} {
		complete := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			var x, y, z, out ArbitraryFloat
			var values [2]float64
			for i, v := range []*ArbitraryFloat{&x, &y} {
				*v = ArbitraryFloat{format: arbitraryFloatFormat(17, 120, i == 0), significand: NewInt(0)}
				values[i] = 1
				switch pair[i] {
				case "nan":
					v.exponent, v.significand = 65536, NewInt(1)
					values[i] = math.NaN()
				case "inf":
					v.exponent = 65536
					values[i] = math.Inf(1)
				case "zero":
					v.exponent = -65535
					values[i] = 0
				}
				if i == 0 {
					values[i] = math.Copysign(values[i], -1)
				}
			}
			n := NewInt(1)
			serviceMust(t, m.integer.Mul2k(&n, 4000, &n))
			serviceMust(t, m.SetComponents(&out, 17, 4001, false, 9, &n))
			serviceMust(t, m.integer.Del(&n))
			serviceMust(t, m.MkOne(17, 120, false, &z))
			oldX, oldY := x, y
			operation := m.FMA
			b.FailAfter(ordinal)
			body := operation(ArbitraryFloatNearestEven, &x, &y, &z, &out)
			integerPhysicalAudit(t, m.integer, &x.significand, &y.significand, &out.significand)
			if x != oldX || y != oldY || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
				t.Fatal("early branch physical graph", pair, ordinal, body)
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected special refusal", pair, ordinal, body)
			}
			if m.fma.manager != nil || m.RoundScope.Manager != nil {
				t.Fatal("early branch constructed general or Round locals")
			}
			b.FailAfter(-1)
			if body == nil {
				want := math.FMA(values[0], values[1], 1)
				if math.IsNaN(want) {
					ok, e := m.IsNaN(&out)
					serviceMust(t, e)
					if !ok || out.Sign() {
						t.Fatal("canonical NaN", pair, out)
					}
				} else {
					var ok bool
					var e error
					switch {
					case math.IsInf(want, 0):
						ok, e = m.IsInf(&out)
					case want == 0:
						ok, e = m.IsZero(&out)
					default:
						ok, e = m.IsOne(&out)
					}
					serviceMust(t, e)
					if !ok || out.Sign() != math.Signbit(want) {
						t.Fatal("early branch result", pair, out, want)
					}
				}
			}
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.Close())
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("special branch refund", pair, ordinal, b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("special branch reservation horizon", pair)
		}
	}
}
