package numeral

// Source: src/util/mpf.cpp:1047-1170; src/util/mpf.h:133.

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

var mpfIntegralModes = [...]ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatNearestAway, ArbitraryFloatTowardPositive, ArbitraryFloatTowardNegative, ArbitraryFloatTowardZero}

func TestArbitraryFloatRoundToIntegralIEEEBranchesAndAliases(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	values := []float64{0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1), math.NaN(), .25, -.25, .5, -.5, .75, -.75, 1, -1, 1.5, -1.5, 2.5, -2.5, 3.5, -3.5, math.Nextafter(1.5, 0), math.Nextafter(1.5, 2), math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, math.MaxFloat64, -math.MaxFloat64}
	for _, v := range values {
		for _, mode := range mpfIntegralModes {
			for _, alias := range []bool{false, true} {
				var x, separate ArbitraryFloat
				serviceMust(t, m.SetDouble(&x, 11, 53, v))
				out := &separate
				if alias {
					out = &x
				}
				old := x // borrowed identity only
				serviceMust(t, m.RoundToIntegral(mode, &x, out))
				got, e := m.ToDouble(out)
				serviceMust(t, e)
				want := v
				switch mode {
				case ArbitraryFloatNearestEven:
					want = math.RoundToEven(v)
				case ArbitraryFloatNearestAway:
					want = math.Round(v)
				case ArbitraryFloatTowardPositive:
					want = math.Ceil(v)
				case ArbitraryFloatTowardNegative:
					want = math.Floor(v)
				case ArbitraryFloatTowardZero:
					want = math.Trunc(v)
				}
				if math.IsNaN(want) {
					if !math.IsNaN(got) || out.Sign() {
						t.Fatal("canonical source NaN", got, out.Sign())
					}
				} else if math.Float64bits(got) != math.Float64bits(want) {
					t.Fatal("IEEE integral rounding", v, mode, alias, got, want)
				}
				if !alias && x != old {
					t.Fatal("const source changed", v, mode)
				}
				serviceMust(t, m.Del(&separate))
				serviceMust(t, m.Del(&x))
			}
		}
	}
}

func mpfIntegralOracle(value *big.Rat, mode ArbitraryFloatRoundingMode) *big.Int {
	negative := value.Sign() < 0
	n := new(big.Int).Abs(value.Num())
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, value.Denom(), rem)
	inc := false
	switch mode {
	case ArbitraryFloatNearestEven, ArbitraryFloatNearestAway:
		cmp := new(big.Int).Lsh(rem, 1).Cmp(value.Denom())
		inc = cmp > 0 || cmp == 0 && (mode == ArbitraryFloatNearestAway || q.Bit(0) != 0)
	case ArbitraryFloatTowardPositive:
		inc = !negative && rem.Sign() != 0
	case ArbitraryFloatTowardNegative:
		inc = negative && rem.Sign() != 0
	}
	if inc {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return q
}

func TestArbitraryFloatRoundToIntegralIndependentDyadicOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{104, 7, 117}))
	for _, sb := range []uint32{2, 24, 120} {
		for sample := 0; sample < 100; sample++ {
			sig := new(big.Int).SetUint64(rng.Uint64())
			sig.Lsh(sig, uint(rng.Uint32()%120))
			sig.Mod(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exp := int64(rng.Uint32()%(sb+4)) - 4
			negative := sample%2 != 0
			full := new(big.Int).Add(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			den := new(big.Int).Lsh(big.NewInt(1), uint(int64(sb)-1-exp))
			value := new(big.Rat).SetFrac(full, den)
			if negative {
				value.Neg(value)
			}
			for _, mode := range mpfIntegralModes {
				for _, alias := range []bool{false, true} {
					var n Int
					serviceMust(t, m.integer.SetString(&n, sig.String()))
					var x, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sb, negative, exp, &n))
					serviceMust(t, m.integer.Del(&n))
					out := &separate
					if alias {
						out = &x
					}
					serviceMust(t, m.RoundToIntegral(mode, &x, out))
					q := NewRational()
					serviceMust(t, m.ToRational(out, m.rational, &q))
					actualN, e := m.integer.String(&q.Num)
					serviceMust(t, e)
					actualD, e := m.integer.Int64(&q.Den)
					serviceMust(t, e)
					want := mpfIntegralOracle(value, mode)
					if actualD != 1 || actualN != want.String() || out.Sign() != negative {
						t.Fatal("independent integral rounding", sb, exp, sig, mode, alias, actualN, actualD, out.Sign(), want)
					}
					serviceMust(t, m.rational.Del(&q))
					serviceMust(t, m.Del(&separate))
					serviceMust(t, m.Del(&x))
				}
			}
		}
	}
}

func TestArbitraryFloatRoundToIntegralEveryColdGeneralReservation(t *testing.T) {
	for _, sb := range []uint32{24, 120, 4000} {
		for _, alias := range []bool{false, true} {
			for _, mode := range []ArbitraryFloatRoundingMode{ArbitraryFloatNearestEven, ArbitraryFloatTowardZero} {
				complete := false
				for ordinal := int64(0); ordinal < 256; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2k(&n, sb-1, &n))
					serviceMust(t, m.integer.Dec(&n))
					var x, separate ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sb, true, 1, &n))
					serviceMust(t, m.integer.Del(&n))
					old := x // borrowed identity only
					out := &separate
					if alias {
						out = &x
					}
					b.FailAfter(ordinal)
					body := m.RoundToIntegral(mode, &x, out)
					s := &m.integralRound
					integerPhysicalAudit(t, m.integer, &x.significand, &separate.significand, &s.a.significand, &s.div, &s.rem)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate integral scope physical graph", sb, alias, mode, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if !alias && x != old {
						t.Fatal("stopped const input changed", ordinal)
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected cold failure", ordinal, body)
					}
					if s.active {
						if e := m.RoundToIntegral(mode, &x, out); !errors.Is(e, ErrNumericPending) {
							t.Fatal("stopped scope replay", e)
						}
					}
					b.FailAfter(-1)
					if e := m.CleanupRoundToIntegral(); e != nil {
						t.Fatalf("cleanup sb=%d alias=%v mode=%d ordinal=%d body=%v scope=%+v: %v", sb, alias, mode, ordinal, body, *s, e)
					}
					serviceMust(t, m.CleanupValue(&separate))
					serviceMust(t, m.CleanupValue(&x))
					if source := m.SourceFinalize(); source != nil {
						z := m.integer
						if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatal("unexpected source destructor stop", sb, alias, mode, ordinal, source, z.SourceFinalizeStage)
						}
						index := int(z.SourceFinalizeStage-2) / 2
						ref := z.Tmp[index]
						if z.SourceFinalizeStage%2 == 1 {
							ref = z.Arg[index]
						}
						if ref != 0 {
							t.Fatal("unproved actual source scratch hole", ordinal, ref)
						}
					}
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("integral scope refund", ordinal, b.Used())
					}
					if body == nil {
						complete = true
						break
					}
				}
				if !complete {
					t.Fatal("integral rounding reservation horizon")
				}
			}
		}
	}
}
