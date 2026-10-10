// Source: src/util/mpf.h:75,181; src/util/mpf.cpp:1890-1896,2133-2141.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func mpfNumeralHelpersFixture(t *testing.T) (*ArbitraryFloatManager, *umemory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		serviceMust(t, m.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("numeral helper owner end", b.Used())
		}
	})
	return m, r, b
}

func TestArbitraryFloatRegularSourceClassesAndDefaultErasure(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	// sbits=0 returns true before any exponent-cache or significand reads.
	defaultValue := ArbitraryFloat{exponent: math.MaxInt64, significand: Int{Val: 1, Cell: NumericCellRef(math.MaxUint32)}}
	before := b.Used()
	b.FailAfter(0)
	ok, e := m.IsRegular(&defaultValue)
	serviceMust(t, e)
	if !ok || b.Used() != before {
		t.Fatal("default numeral did not erase numeric reads", ok, b.Used(), before)
	}
	b.FailAfter(-1)
	for _, sign := range []bool{false, true} {
		for _, c := range []struct {
			exp  int64
			sig  int32
			want bool
		}{
			{128, 0, false}, {128, 1, false}, // infinities and NaNs
			{-127, 0, false}, {-127, 1, true}, // formatted zeros are not regular
			{0, 0, true}, {0, 1, true}, {127, 7, true},
		} {
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, sign), exponent: c.exp, significand: NewInt(c.sig)}
			ok, e := m.IsRegular(&x)
			serviceMust(t, e)
			if ok != c.want {
				t.Fatal("original is_normal || is_denormal", sign, c, ok)
			}
			before := b.Used()
			b.FailAfter(0)
			again, e := m.IsRegular(&x)
			serviceMust(t, e)
			if again != ok || b.Used() != before {
				t.Fatal("warm classification allocated")
			}
			b.FailAfter(-1)
		}
	}
}

func TestArbitraryFloatPrevPowerSourceFormulaAndDebugDomains(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, sb := range []uint32{1, 2, 24, 53, 120, 4000, 65535} {
		for _, exp := range []int64{math.MinInt64, -65536, -4000, -120, -53, -24, -2, -1, 0, 1, 127, 129, math.MaxInt32, math.MaxUint32, math.MaxInt64 - 65535} {
			for _, sign := range []bool{false, true} {
				x := ArbitraryFloat{format: arbitraryFloatFormat(8, sb, sign), exponent: exp, significand: NewInt(1)}
				got, e := m.PrevPowerOfTwo(&x)
				serviceMust(t, e)
				var want uint32
				if !sign && exp > -int64(sb) {
					// Independent arbitrary-width arithmetic and explicit unsigned cast.
					n := new(big.Int).Add(big.NewInt(exp), new(big.Int).SetUint64(uint64(sb)))
					n.Sub(n, big.NewInt(1))
					n.Mod(n, new(big.Int).Lsh(big.NewInt(1), 32))
					want = uint32(n.Uint64())
				}
				if got != want {
					t.Fatal("source sbits + exponent - 1", sb, exp, sign, got, want)
				}
			}
		}
	}
	for _, sign := range []bool{false, true} {
		for _, sig := range []int32{0, 1} {
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, sign), exponent: 128, significand: NewInt(sig)}
			got, e := m.PrevPowerOfTwo(&x)
			if modularSourceDebug {
				if !errors.Is(e, ErrIntegerAssertion) {
					t.Fatal("debug special-value assertion", sign, sig, got, e)
				}
			} else {
				serviceMust(t, e)
				want := uint32(0)
				if !sign && sig == 0 {
					want = 151 // release +inf follows the ordinary source formula
				}
				if got != want {
					t.Fatal("release special-value erasure", sign, sig, got, want)
				}
			}
		}
	}
	for _, sb := range []uint32{24, 4000} {
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, sb, false), exponent: -127}
		got, e := m.PrevPowerOfTwo(&x)
		serviceMust(t, e)
		want := uint32(0)
		if sb == 4000 {
			want = 3872 // source does not separately exclude positive zero
		}
		if got != want {
			t.Fatal("source positive-zero boundary", sb, got, want)
		}
	}
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: math.MaxInt64, significand: NewInt(1)}
	if _, e := m.PrevPowerOfTwo(&x); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached undefined signed addition", e)
	}
	x.assignSign(true)
	if got, e := m.PrevPowerOfTwo(&x); e != nil || got != 0 {
		t.Fatal("negative branch must erase overflowing expression", got, e)
	}
}

func TestArbitraryFloatMkOneAndResetReplaceBIGNumerals(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	sig := NewInt(1)
	serviceMust(t, m.integer.Mul2k(&sig, 4000, &sig))
	defer func() { serviceMust(t, m.CleanupValue(&x)); serviceMust(t, m.integer.CleanupRawNumeral(&sig)) }()
	for _, sign := range []bool{false, true} {
		serviceMust(t, m.SetComponents(&x, 8, 4000, !sign, 19, &sig))
		serviceMust(t, m.MkOne(11, 53, sign, &x))
		v, e := m.integer.Int64(&x.significand)
		serviceMust(t, e)
		if x.EBits() != 11 || x.SBits() != 53 || x.Sign() != sign || x.exponent != 0 || v != 0 {
			t.Fatal("literal mk_one publication", x, v)
		}
		got, e := m.ToDouble(&x)
		serviceMust(t, e)
		want := float64(1)
		if sign {
			want = -1
		}
		if got != want {
			t.Fatal("one value", got, want)
		}
		serviceMust(t, m.SetComponents(&x, 8, 4000, sign, 19, &sig))
		serviceMust(t, m.Reset(&x, 8, 24))
		zero, e := m.IsZero(&x)
		serviceMust(t, e)
		if !zero || x.Sign() || x.EBits() != 8 || x.SBits() != 24 || x.exponent != -127 {
			t.Fatal("reset delegates signed machine-int zero set", x)
		}
		integerPhysicalAudit(t, m.integer, &sig, &x.significand)
		if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
			t.Fatal("actual retained owner bytes", b.Used(), m.OwnedStorageBytes(), r.Stats())
		}
	}
}

func TestArbitraryFloatNumeralHelpersEveryColdReservation(t *testing.T) {
	for _, operation := range []string{"regular", "prev", "reset", "one"} {
		t.Run(operation, func(t *testing.T) {
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				x := ArbitraryFloat{format: arbitraryFloatFormat(8, 4000, true), exponent: 7, significand: NewInt(9)}
				if operation == "reset" || operation == "one" {
					serviceMust(t, m.integer.Mul2k(&x.significand, 4000, &x.significand))
				}
				oldSignificand := x.significand // borrowed identity for prefix inspection only
				b.FailAfter(ordinal)
				var body error
				switch operation {
				case "regular":
					_, body = m.IsRegular(&x)
				case "prev":
					_, body = m.PrevPowerOfTwo(&x)
				case "reset":
					body = m.Reset(&x, 11, 120)
				case "one":
					body = m.MkOne(11, 120, false, &x)
				}
				integerPhysicalAudit(t, m.integer, &x.significand)
				if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
					t.Fatal("immediate cold physical graph", ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
				}
				if operation == "regular" || operation == "prev" {
					if x.format != arbitraryFloatFormat(8, 4000, true) || x.exponent != 7 || x.significand != NewInt(9) {
						t.Fatal("const input mutated", ordinal, x)
					}
				}
				if operation == "reset" && (x.EBits() != 11 || x.SBits() != 120 || x.Sign()) {
					t.Fatal("initial zero-set field prefix", ordinal, x)
				}
				if operation == "reset" && body != nil && (x.exponent != 7 || x.significand != oldSignificand) {
					t.Fatal("refused exponent cache changed later fields", ordinal, x)
				}
				if body == nil && (operation == "reset" || operation == "one") {
					wantExponent := int64(0)
					if operation == "reset" {
						wantExponent = -1023
					}
					if x.EBits() != 11 || x.SBits() != 120 || x.Sign() || x.exponent != wantExponent || x.significand != NewInt(0) {
						t.Fatal("completed replacement value", ordinal, operation, x)
					}
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected cold failure", ordinal, body)
				}
				b.FailAfter(-1)
				serviceMust(t, m.CleanupValue(&x))
				serviceMust(t, m.Close())
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("cold owner refund", ordinal, b.Used())
				}
				if body == nil {
					return
				}
			}
			t.Fatal("helper reservation horizon")
		})
	}
}
