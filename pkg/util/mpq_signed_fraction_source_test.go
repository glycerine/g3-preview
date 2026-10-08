// Source: src/util/mpq.h:678-687; src/util/mpz.cpp:560-817.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"math/big"
	"testing"
)

func TestRationalSignedFractionSourceIndependentPairs(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	q := m.rational
	for n := int32(-20); n <= 20; n++ {
		for d := int32(-20); d <= 20; d++ {
			if d == 0 {
				continue
			}
			a := NewRational()
			serviceMust(t, q.SetFractionSignedSource(&a, n, d))
			expected := new(big.Rat).SetFrac(big.NewInt(int64(n)), big.NewInt(int64(d)))
			if actual := rationalString(t, q, &a); actual != expected.RatString() {
				t.Fatal("source signed pair", n, d, actual, expected)
			}
			serviceMust(t, q.Del(&a))
		}
	}
	for _, tc := range [][2]int32{{math.MinInt32, 1}, {math.MinInt32, 3}, {math.MinInt32, math.MaxInt32}, {math.MaxInt32, -1}, {math.MaxInt32, -math.MaxInt32}, {0, -math.MaxInt32}} {
		a := NewRational()
		serviceMust(t, q.SetFractionSignedSource(&a, tc[0], tc[1]))
		expected := new(big.Rat).SetFrac(big.NewInt(int64(tc[0])), big.NewInt(int64(tc[1])))
		if actual := rationalString(t, q, &a); actual != expected.RatString() {
			t.Fatal("boundary signed pair", tc, actual, expected)
		}
		serviceMust(t, q.Del(&a))
		serviceMust(t, q.Z.CleanupGCDRaw())
		serviceMust(t, q.Z.CleanupDivision())
	}
}

func TestRationalSignedFractionSourceReachedUndefinedNegation(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	q := m.rational
	for _, tc := range [][2]int32{{math.MinInt32, -1}, {1, math.MinInt32}, {math.MinInt32, math.MinInt32}, {0, math.MinInt32}} {
		a := Rational{Num: NewInt(7), Den: NewInt(9)}
		before := a
		used := b.Used()
		if e := q.SetFractionSignedSource(&a, tc[0], tc[1]); !errors.Is(e, arena.ErrBounds) || a != before || b.Used() != used {
			t.Fatal("reached signed32 negation", tc, e, a)
		}
	}
	// Existing widened service behavior remains an explicitly separate profile.
	a := NewRational()
	serviceMust(t, q.SetFractionSigned(&a, math.MinInt32, -1))
	if actual := rationalString(t, q, &a); actual != "2147483648" {
		t.Fatal("widened service changed", actual)
	}
	serviceMust(t, q.Del(&a))
}

func TestRationalSignedFractionSourceZeroDenominator(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, n := range []int32{0, 1, -1} {
		a := Rational{Num: NewInt(7), Den: NewInt(9)}
		e := m.rational.SetFractionSignedSource(&a, n, 0)
		if modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) || a.Num.Val != 7 || a.Den.Val != 9 {
				t.Fatal("source denominator assertion", n, e, a)
			}
		} else if n == 0 {
			if !errors.Is(e, ErrDivideByZero) || a.Num.Val != 0 || a.Den.Val != 0 {
				t.Fatal("reached normalize divide-by-zero", e, a)
			}
		} else if e != nil || a.Num.Val != n || a.Den.Val != 0 {
			t.Fatal("literal nonzero/zero normalize", n, e, a)
		}
		serviceMust(t, m.rational.Z.CleanupRawNumeral(&a.Num))
		serviceMust(t, m.rational.Z.CleanupRawNumeral(&a.Den))
	}
}

func TestRationalSignedFractionSourceEveryColdGCDReservation(t *testing.T) {
	for _, den := range []int32{3, 8, math.MaxInt32} {
		for _, reuse := range []bool{false, true} {
			complete := false
			for ordinal := int64(0); ordinal < 256; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				q, z := m.rational, m.integer
				a := NewRational()
				if reuse {
					serviceMust(t, z.SetString(&a.Num, "123456789012345678901234567890"))
					serviceMust(t, z.SetString(&a.Den, "98765432109876543210987654321"))
				}
				b.FailAfter(ordinal)
				body := q.SetFractionSignedSource(&a, math.MinInt32, den)
				integerPhysicalAudit(t, z, &a.Num, &a.Den, &q.NTmp)
				if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
					t.Fatal("source signed pair immediate graph", den, reuse, ordinal, body, b.Used())
				}
				if body != nil {
					if !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected source signed pair refusal", den, reuse, ordinal, body)
					}
					if z.GcdRaw.Active && !z.GcdRaw.BodySucceeded && (a.Num.Val != math.MinInt32 || a.Den.Val != den || a.Num.Cell != 0 || a.Den.Cell != 0) {
						t.Fatal("GCD failure lost completed scalar SETs", a)
					}
				} else {
					b.FailAfter(-1)
					expected := new(big.Rat).SetFrac(big.NewInt(math.MinInt32), big.NewInt(int64(den)))
					if actual := rationalString(t, q, &a); actual != expected.RatString() {
						t.Fatal("cold signed pair result", actual, expected)
					}
				}
				b.FailAfter(-1)
				serviceMust(t, z.CleanupGCDRaw())
				serviceMust(t, z.CleanupDivision())
				serviceMust(t, z.CleanupRawNumeral(&a.Num))
				serviceMust(t, z.CleanupRawNumeral(&a.Den))
				if e := m.Close(); e != nil {
					if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
						t.Fatal("signed pair source stop", e)
					}
					i := int(z.SourceFinalizeStage-2) / 2
					ref := z.Tmp[i]
					if z.SourceFinalizeStage%2 == 1 {
						ref = z.Arg[i]
					}
					if ref != 0 {
						t.Fatal("unproved signed pair scratch hole", ref)
					}
					serviceMust(t, m.Close())
				}
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("signed pair refunds", den, reuse, ordinal, b.Used())
				}
				if body == nil {
					complete = true
					break
				}
			}
			if !complete {
				t.Fatal("signed pair reservation horizon", den, reuse)
			}
		}
	}
}

func TestRationalSignedFractionSourceWarmReinitialization(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	a := NewRational()
	serviceMust(t, m.rational.SetFractionSignedSource(&a, 12, 18))
	serviceMust(t, m.rational.SetFractionSignedSource(&a, math.MinInt32, 3))
	serviceMust(t, m.integer.CleanupGCDRaw())
	serviceMust(t, m.integer.CleanupDivision())
	serviceMust(t, m.rational.SetFractionSignedSource(&a, 12, 18))
	used, stats := b.Used(), r.Stats()
	b.FailAfter(0)
	serviceMust(t, m.rational.SetFractionSignedSource(&a, 14, -21))
	if a.Num.Val != -2 || a.Den.Val != 3 || b.Used() != used || r.Stats() != stats {
		t.Fatal("warm signed pair allocated")
	}
	b.FailAfter(-1)
	serviceMust(t, m.rational.Del(&a))
}

func TestRationalSignedFractionSourceNumeratorReleaseStopsPublication(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	a := NewRational()
	digits := make([]uint32, 125)
	digits[len(digits)-1] = 1
	serviceMust(t, m.integer.SetLimbs(&a.Num, digits))
	serviceMust(t, m.integer.SetLimbs(&a.Den, []uint32{3, 1}))
	old := a
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	delta := int64(-99950) - r.State().LocalSize
	if e := r.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("public failed TLS realloc", e)
	}
	body := m.rational.SetFractionSignedSource(&a, -3, 7)
	if !errors.Is(body, arena.ErrBounds) || a != old || a.Num.Cell == 0 || !m.integer.Cell(a.Num.Cell).Released {
		t.Fatal("numerator DEL must stop before scalar SET", body, a, old)
	}
	state, used := r.State(), budget.Used()
	if e := m.rational.SetFractionSignedSource(&a, -3, 7); !errors.Is(e, arena.ErrInvalidSlot) || r.State() != state || budget.Used() != used || a != old {
		t.Fatal("released source replay", e)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &a.Num, &a.Den, &m.rational.NTmp)
	if budget.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("signed setter source stop graph", budget.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	contextTLSFinish(t, r, func() error { return m.integer.CleanupRawNumeral(&a.Num) })
	contextTLSFinish(t, r, func() error { return m.integer.CleanupRawNumeral(&a.Den) })
	contextTLSFinish(t, r, m.Close)
	for _, block := range []*Block{&lane, &recovery} {
		contextTLSFinish(t, r, func() error {
			if *block == (Block{}) {
				return nil
			}
			return r.Deallocate(block)
		})
	}
	contextTLSFinish(t, r, r.Close)
	if budget.Used() != 0 {
		t.Fatal("signed setter source stop refunds", budget.Used())
	}
}

// Source: src/test/mpq.cpp:24-31, original tst0 including denominator alias.
func TestRationalSignedFractionSourceDriverTst0(t *testing.T) {
	for _, synch := range []bool{false, true} {
		q, _ := rationalManager(t, synch)
		a, b, expected := NewRational(), NewRational(), NewRational()
		serviceMust(t, q.SetFractionSignedSource(&a, 2, 3))
		serviceMust(t, q.SetFractionSignedSource(&b, 4, 3))
		serviceMust(t, q.Div(&a, &b, &b))
		serviceMust(t, q.SetFractionSignedSource(&expected, 1, 2))
		equal, e := q.Equal(&b, &expected)
		serviceMust(t, e)
		if !equal {
			t.Fatal("original tst0 alias quotient", synch)
		}
		for _, v := range []*Rational{&a, &b, &expected} {
			serviceMust(t, q.Del(v))
		}
	}
}
