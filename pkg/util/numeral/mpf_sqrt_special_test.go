package numeral

// Source: src/util/mpf.cpp:1010-1017 (early branches and their real callees).

import (
	"testing"
)

func TestArbitraryFloatSqrtEveryColdSpecialReservation(t *testing.T) {
	for _, input := range []struct {
		name string
		exp  int64
		sig  int32
		sign bool
	}{
		{"nan", 65536, 1, true}, {"inf", 65536, 0, false}, {"zero", -65535, 0, true},
		{"negative", -1, 0, true}, {"negative-infinity", 65536, 0, true},
	} {
		t.Run(input.name, func(t *testing.T) {
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				x := ArbitraryFloat{format: arbitraryFloatFormat(17, 120, input.sign), exponent: input.exp, significand: NewInt(input.sig)}
				var out ArbitraryFloat
				n := NewInt(1)
				serviceMust(t, m.integer.Mul2k(&n, 4000, &n))
				serviceMust(t, m.SetComponents(&out, 17, 4000, false, 9, &n))
				serviceMust(t, m.integer.Del(&n))
				before := x
				b.FailAfter(ordinal)
				body := m.Sqrt(ArbitraryFloatNearestEven, &x, &out)
				integerPhysicalAudit(t, m.integer, &x.significand, &out.significand)
				if x != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
					t.Fatal("stopped early-branch graph", input.name, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected early failure", ordinal, body)
				}
				if m.sqrt.active || m.sqrt.manager != nil {
					t.Fatal("early branch constructed unrelated general locals")
				}
				b.FailAfter(-1)
				if body == nil {
					if out.EBits() != 17 || out.SBits() != 120 {
						t.Fatal("early format", out)
					}
					switch input.name {
					case "nan", "negative", "negative-infinity":
						ok, e := m.IsNaN(&out)
						serviceMust(t, e)
						if !ok || out.Sign() {
							t.Fatal("canonical NaN", out)
						}
					case "inf":
						ok, e := m.EqCore(&x, &out)
						serviceMust(t, e)
						if !ok {
							t.Fatal("literal special/integer copy", x, out)
						}
					default:
						ok, e := m.IsZero(&out)
						serviceMust(t, e)
						if !ok || !out.Sign() {
							t.Fatal("signed zero", out)
						}
					}
				}
				serviceMust(t, m.CleanupValue(&out))
				serviceMust(t, m.Close())
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("early branch refund", ordinal, b.Used())
				}
				if body == nil {
					return
				}
			}
			t.Fatal("early branch reservation horizon")
		})
	}
}
