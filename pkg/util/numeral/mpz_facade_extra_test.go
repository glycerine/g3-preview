// Source: mpz.h public methods; test/mpz.cpp and test/mpq.cpp tst1.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerFacadeSourceTst1ComparisonAssertions(t *testing.T) {
	// Two actual active tst1 bodies have the same nine assertions. The second
	// calls inherited integer overloads on synch_mpq_manager; existing Integers
	// composition supplies that owner in Go, without a new manager namespace.
	for context := 0; context < 2; context++ {
		var m *IntegerManager
		var inherited *RationalManager
		if context == 0 {
			m, _ = integerManager(t)
		} else {
			q, _ := rationalManager(t, true)
			inherited = q
			m = q.Integers()
		}
		var v, v2, v3 Int
		const input = "1002034040050606089383838288182"
		must := func(e error) {
			t.Helper()
			if e != nil {
				t.Fatal(e)
			}
		}
		must(m.SetString(&v, input))
		negTwo := NewInt(-2)
		must(m.Mul(&v, &negTwo, &v2))
		must(m.Add(&v, &v2, &v3))
		must(m.Neg(&v3))
		count := 0
		ensure := func(got bool, e error) {
			t.Helper()
			count++
			if e != nil || !got {
				t.Fatal(context, count, got, e)
			}
		}
		ensure(m.Equal(&v, &v3))
		ensure(m.LessEqual(&v, &v3))
		ensure(m.GreaterEqual(&v, &v3))
		ensure(m.Less(&v2, &v))
		ensure(m.LessEqual(&v2, &v))
		ensure(m.Greater(&v, &v2))
		ensure(m.GreaterEqual(&v, &v2))
		ensure(m.NotEqual(&v, &v2))
		neq, e := m.NotEqual(&v, &v3)
		ensure(!neq, e)
		if count != 9 {
			t.Fatal(count)
		}
		// Added permanent output expectation for the source's actual print actions.
		if integerString(t, m, &v) != input || integerString(t, m, &v2) != "-2004068080101212178767676576364" {
			t.Fatal("source print values")
		}
		must(m.Del(&v))
		must(m.Del(&v2))
		must(m.Del(&v3))
		if inherited != nil {
			inherited.Close()
		} else {
			m.Close()
		}
	}
}

func TestIntegerFacadeSignedPredicatesAndSetterShape(t *testing.T) {
	m, b := integerManager(t)
	var a Int
	for _, n := range []int32{math.MinInt32, -99, -1, 0, 1, 99, math.MaxInt32} {
		if e := m.SetInt32(&a, n); e != nil {
			t.Fatal(e)
		}
		if !a.IsSmall() {
			t.Fatal("int32 setter must inline")
		}
		sign, e := m.Sign(&a)
		if e != nil || sign != n {
			t.Fatal("source full inline sign", n, sign, e)
		}
		for _, which := range []uint8{0, 1, 2, 3, 4, 5, 6} {
			var got, want bool
			switch which {
			case 0:
				got, e = m.IsPos(&a)
				want = n > 0
			case 1:
				got, e = m.IsNeg(&a)
				want = n < 0
			case 2:
				got, e = m.IsZero(&a)
				want = n == 0
			case 3:
				got, e = m.IsNonPos(&a)
				want = n <= 0
			case 4:
				got, e = m.IsNonNeg(&a)
				want = n >= 0
			case 5:
				got, e = m.IsOne(&a)
				want = n == 1
			case 6:
				got, e = m.IsMinusOne(&a)
				want = n == -1
			}
			if e != nil || got != want {
				t.Fatal(n, which, got, e)
			}
		}
	}
	for _, n := range []uint32{0, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		if e := m.SetUint32(&a, n); e != nil {
			t.Fatal(e)
		}
		if a.IsSmall() != (n <= math.MaxInt32) {
			t.Fatal("uint32 setter threshold", n)
		}
		u, e := m.Uint64(&a)
		if e != nil || u != uint64(n) {
			t.Fatal(n, u, e)
		}
	}
	if e := m.SetUint64(&a, math.MaxInt32); e != nil {
		t.Fatal(e)
	}
	if a.IsSmall() {
		t.Fatal("source uint64 strict threshold")
	}
	one, e := m.IsOne(&a)
	if e != nil || one {
		t.Fatal(one, e)
	}
	if e := m.SetInt32(&a, 1); e != nil {
		t.Fatal(e)
	}
	baseline := b.Used()
	b.FailAfter(0)
	sign, e := m.Sign(&a)
	b.FailAfter(-1)
	if e != nil || sign != 1 || b.Used() != baseline {
		t.Fatal("read allocated", sign, e)
	}
	m.Del(&a)
}

func TestIntegerFacadeDupAndScalarComparison(t *testing.T) {
	m, _ := integerManager(t)
	var a Int
	if e := m.SetString(&a, "42949672961234567890123456789"); e != nil {
		t.Fatal(e)
	}
	copy, e := m.Dup(&a)
	if e != nil {
		t.Fatal(e)
	}
	if copy.Cell == a.Cell || m.Cell(copy.Cell).Capacity != m.Cell(a.Cell).Capacity {
		t.Fatal("source Dup independent cell/requested capacity")
	}
	if e = m.SetInt32(&a, -3); e != nil {
		t.Fatal(e)
	}
	if integerString(t, m, &copy) != "42949672961234567890123456789" {
		t.Fatal("Dup aliased source")
	}
	for _, n := range []int32{math.MinInt32, -4, -3, -2, 0, math.MaxInt32} {
		got, e := m.LessInt(&a, n)
		if e != nil || got != (-3 < n) {
			t.Fatal(n, got, e)
		}
		got, e = m.LessInt(&copy, n)
		if e != nil || got {
			t.Fatal("big/scalar", n, got, e)
		}
	}
	m.Del(&a)
	m.Del(&copy)
}

func TestIntegerFacadeFiniteMathAndIncDecAliases(t *testing.T) {
	m, _ := integerManager(t)
	for n := int64(-64); n <= 64; n++ {
		var a Int
		m.SetInt64(&a, n)
		if e := m.Inc(&a); e != nil {
			t.Fatal(e)
		}
		got, e := m.Int64(&a)
		if e != nil || got != n+1 {
			t.Fatal(n, got, e)
		}
		if e = m.Dec(&a); e != nil {
			t.Fatal(e)
		}
		got, e = m.Int64(&a)
		if e != nil || got != n {
			t.Fatal(n, got, e)
		}
		m.Del(&a)
	}
	for _, negative := range []bool{false, true} {
		z := new(big.Int).Lsh(big.NewInt(1), 500)
		if negative {
			z.Neg(z)
		}
		var a Int
		if e := m.SetString(&a, z.String()); e != nil {
			t.Fatal(e)
		}
		if e := m.Inc(&a); e != nil {
			t.Fatal(e)
		}
		want := new(big.Int).Add(z, big.NewInt(1))
		if integerString(t, m, &a) != want.String() {
			t.Fatal("large increment")
		}
		if e := m.Dec(&a); e != nil {
			t.Fatal(e)
		}
		if integerString(t, m, &a) != z.String() {
			t.Fatal("large decrement")
		}
		sign, e := m.Sign(&a)
		if e != nil || sign != int32(z.Sign()) {
			t.Fatal(sign, e)
		}
		m.Del(&a)
	}
}

func TestIntegerFacadeDupIncDecFailureRoots(t *testing.T) {
	for op := 0; op < 3; op++ {
		complete := false
		failures := 0
		for prefix := int64(0); prefix < 32; prefix++ {
			m, b := mpzPublicationDirectOwner(t)
			limbs := make([]uint32, 40)
			for i := range limbs {
				limbs[i] = math.MaxUint32
			}
			var a Int
			if e := m.SetLimbs(&a, limbs); e != nil {
				t.Fatal(e)
			}
			baseline := m.Cells.Stats().Live
			oldCell, oldVal, oldSize := a.Cell, a.Val, m.Cell(a.Cell).Size
			var copy Int
			b.FailAfter(prefix)
			var e error
			switch op {
			case 0:
				e = m.Inc(&a)
			case 1:
				e = m.Dec(&a)
			case 2:
				copy, e = m.Dup(&a)
			}
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				if a.Cell != oldCell || a.Val != oldVal || m.Cell(a.Cell).Size != oldSize {
					t.Fatal("failed publication")
				}
				for i, v := range limbs {
					if m.Digit(a.Cell, uint32(i)) != v {
						t.Fatal("failed mutation")
					}
				}
				mpzPublicationAudit(t, m, &a, &copy)
			} else {
				complete = true
				if op == 2 {
					if m.Cells.Stats().Live != baseline+1 || copy.Cell == a.Cell {
						t.Fatal("Dup roots")
					}
				} else if m.Cells.Stats().Live != baseline {
					t.Fatal("Inc/Dec roots")
				}
			}
			b.FailAfter(-1)
			m.Del(&a)
			m.Del(&copy)
			m.Close()
			if b.Used() != 0 {
				t.Fatal("failure graph retained")
			}
			if complete {
				break
			}
		}
		if !complete || failures == 0 {
			t.Fatal("incomplete fault sweep", op, complete, failures)
		}
	}
}

func TestIntegerFacadeLifetimeAndDup(t *testing.T) {
	m, b := integerManager(t)
	a := NewInt(17)
	baseline := b.Used()
	b.FailAfter(0)
	copy, ce := m.Dup(&a)
	b.FailAfter(-1)
	if ce != nil || a.Val != 17 || copy.Val != 17 || !copy.IsSmall() || b.Used() != baseline {
		t.Fatal("small Dup", ce)
	}
	for op := 0; op < 3; op++ {
		var e error
		switch op {
		case 0:
			_, e = m.Sign(nil)
		case 1:
			e = m.Inc(nil)
		case 2:
			_, e = m.Dup(nil)
		}
		if !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("nil guard", op, e)
		}
	}
	m.Close()
	if _, e := m.IsZero(&a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestIntegerFacadeRepresentationEqualityAndWarmReads(t *testing.T) {
	m, b := integerManager(t)
	var inline, bigValue Int
	if e := m.SetUint32(&inline, math.MaxInt32); e != nil {
		t.Fatal(e)
	}
	if e := m.SetUint64(&bigValue, math.MaxInt32); e != nil {
		t.Fatal(e)
	}
	if inline.Cell != 0 || bigValue.Cell == 0 {
		t.Fatal("source overload representation")
	}
	equal, e := m.Equal(&inline, &bigValue)
	if e != nil || !equal {
		t.Fatal("small/big equality", equal, e)
	}
	if e = m.Neg(&bigValue); e != nil {
		t.Fatal(e)
	}
	minus, e := m.IsMinusOne(&bigValue)
	if e != nil || minus {
		t.Fatal("big m_val=-1 is not minus one")
	}
	positive, e := m.IsPos(&bigValue)
	if e != nil || positive {
		t.Fatal(positive, e)
	}
	a := NewInt(41)
	baseline := b.Used()
	var failure error
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(50, func() {
		if failure != nil {
			return
		}
		_, failure = m.Sign(&a)
		if failure == nil {
			_, failure = m.IsPos(&a)
		}
		if failure == nil {
			_, failure = m.IsNeg(&a)
		}
		if failure == nil {
			_, failure = m.IsZero(&a)
		}
		if failure == nil {
			_, failure = m.IsNonPos(&a)
		}
		if failure == nil {
			_, failure = m.IsNonNeg(&a)
		}
		if failure == nil {
			_, failure = m.IsOne(&a)
		}
		if failure == nil {
			_, failure = m.IsMinusOne(&a)
		}
		if failure == nil {
			_, failure = m.Equal(&a, &a)
		}
		if failure == nil {
			_, failure = m.NotEqual(&a, &a)
		}
		if failure == nil {
			_, failure = m.LessInt(&a, 42)
		}
		if failure == nil {
			failure = m.Inc(&a)
		}
		if failure == nil {
			failure = m.Dec(&a)
		}
		if failure == nil {
			_, failure = m.Dup(&a)
		}
	})
	b.FailAfter(-1)
	if failure != nil || allocs != 0 || b.Used() != baseline {
		t.Fatal("warm noallocation", failure, allocs, b.Used(), baseline)
	}
	m.Del(&inline)
	m.Del(&bigValue)
}
