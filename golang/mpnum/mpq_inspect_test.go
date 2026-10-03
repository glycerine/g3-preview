// Copyright (c) 2006 Microsoft Corporation
// Active source assertions: src/test/rational.cpp tst2 conversion block and
// rational_tester::tst_hash. The global rational wrapper remains unported.
package mpnum

import (
	"errors"

	"github.com/glycerine/g3/golang/arena"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestSourceRationalConversionAssertions(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		max, min := NewRational(), NewRational()
		if e := m.SetString(&max, "9223372036854775807"); e != nil {
			t.Fatal(e)
		}
		if e := m.Copy(&min, &max); e != nil {
			t.Fatal(e)
		}
		m.Neg(&min)
		minusOne := NewRational()
		m.SetInt64(&minusOne, -1)
		m.Add(&min, &minusOne, &min)
		over, under := NewRational(), NewRational()
		one := NewRational()
		m.SetInt64(&one, 1)
		m.Add(&max, &one, &over)
		m.Sub(&min, &one, &under)
		for _, f := range []struct {
			a                *Rational
			signed, unsigned bool
		}{{&max, true, true}, {&min, true, false}, {&over, false, true}, {&under, false, false}} {
			got, e := m.IsInt64(f.a)
			if e != nil || got != f.signed {
				t.Fatal(got, e)
			}
			got, e = m.IsUint64(f.a)
			if e != nil || got != f.unsigned {
				t.Fatal(got, e)
			}
		}
		for _, v := range []int64{0, 1, -1} {
			a := NewRational()
			m.SetInt64(&a, v)
			s, e := m.IsInt64(&a)
			u, ue := m.IsUint64(&a)
			if e != nil || ue != nil || !s || u != (v >= 0) {
				t.Fatal(v, s, u, e, ue)
			}
		}
		twice, umax := NewRational(), NewRational()
		m.SetInt64(&twice, 2)
		m.Mul(&twice, &max, &umax)
		m.Add(&umax, &one, &umax)
		if ok, e := m.IsUint64(&umax); e != nil || !ok {
			t.Fatal(ok, e)
		}
		// Source VERIFY equality, followed by two active signed-access ENSURE checks.
		if v, e := m.Uint64(&umax); e != nil || v != math.MaxUint64 {
			t.Fatal(v, e)
		}
		if v, e := m.Int64(&max); e != nil || v != math.MaxInt64 {
			t.Fatal(v, e)
		}
		if v, e := m.Int64(&min); e != nil || v != math.MinInt64 {
			t.Fatal(v, e)
		}
	}
}
func TestSourceRationalHashAssignmentHistory(t *testing.T) {
	m, _ := rationalManager(t, false)
	rng := rand.New(rand.NewSource(197))
	// Source runs zero, then 10001 random signed inline values. Deterministic
	// bounded generation preserves that case count without C library rand state.
	for i := -1; i <= 10000; i++ {
		v := int32(0)
		if i >= 0 {
			v = rng.Int31()
			if i&1 != 0 {
				v = -v
			}
		}
		a, b, c := NewRational(), NewRational(), NewRational()
		m.SetInt64(&a, int64(v))
		m.SetString(&b, "10203939394995449949494394932929")
		m.SetInt64(&c, int64(v))
		if e := m.Copy(&b, &c); e != nil {
			t.Fatal(e)
		}
		ha, e := m.Hash(&a)
		hb, be := m.Hash(&b)
		if e != nil || be != nil || ha != hb || ha != uint32(v) {
			t.Fatal(v, ha, hb, e, be)
		}
		m.drop(&a)
		m.drop(&b)
		m.drop(&c)
	}
	a := Rational{num: NewInt(1), den: NewInt(2)}
	b := Rational{num: NewInt(1), den: NewInt(3)}
	ha, _ := m.Hash(&a)
	hb, _ := m.Hash(&b)
	if ha != 1 || hb != 1 {
		t.Fatal("source numerator-only hash", ha, hb)
	}
}
func TestRationalInspectionPredicatesAndBoundaries(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		if !m.Precise() || !m.Field() {
			t.Fatal("rational manager capabilities")
		}
		for n := int32(-8); n <= 8; n++ {
			for d := int32(-4); d <= 4; d++ {
				if d == 0 {
					continue
				}
				a := Rational{num: NewInt(n), den: NewInt(d)}
				sign, e := m.Sign(&a)
				if e != nil || sign != n {
					t.Fatal("source nonunit inline sign", n, sign, e)
				}
				for _, f := range []struct {
					call func(*Rational) (bool, error)
					want bool
				}{{m.IsPos, n > 0}, {m.IsNeg, n < 0}, {m.IsZero, n == 0}, {m.IsNonPos, n <= 0}, {m.IsNonNeg, n >= 0}, {m.IsOne, n == 1 && d == 1}, {m.IsMinusOne, n == -1 && d == 1}, {m.IsInt, d == 1}, {m.IsEven, d == 1 && n%2 == 0}} {
					got, e := f.call(&a)
					if e != nil || got != f.want {
						t.Fatal(n, d, got, f.want, e)
					}
				}
				ok, k, e := m.IsPowerOfTwo(&a)
				want := d == 1 && n > 0 && (uint32(n)&(uint32(n)-1)) == 0
				if e != nil || ok != want || ok && uint32(1)<<k != uint32(n) {
					t.Fatal(n, d, ok, k, e)
				}
			}
		}
		for _, f := range []struct {
			text          string
			uns, signed32 bool
		}{{"-2147483648", false, true}, {"2147483647", true, true}, {"4294967295", true, false}, {"4294967296", false, false}, {"-2147483649", false, false}, {"1/2", false, false}, {"-1", false, true}, {"0", true, true}} {
			a := NewRational()
			m.SetString(&a, f.text)
			u, e := m.IsUnsigned(&a)
			s, se := m.IsInt32(&a)
			if e != nil || se != nil || u != f.uns || s != f.signed32 {
				t.Fatal(f, u, s, e, se)
			}
			if u {
				v, e := m.Unsigned(&a)
				oracle, _ := new(big.Int).SetString(f.text, 10)
				if e != nil || uint64(v) != oracle.Uint64() {
					t.Fatal(v, e)
				}
			} else {
				v, e := m.Unsigned(&a)
				if modularSourceDebug {
					if !errors.Is(e, ErrIntegerAssertion) {
						t.Fatal(e)
					}
				} else {
					n := roundPublicationDecode(t, m.z, &a.num)
					want := uint32(projectionLowMagnitude(n))
					if a.num.IsSmall() {
						want = uint32(a.num.val)
					}
					if e != nil || v != want {
						t.Fatal("release numerator unsigned projection", f, v, want, e)
					}
				}
			}
			if s {
				v, e := m.Int32(&a)
				oracle, _ := new(big.Int).SetString(f.text, 10)
				if e != nil || int64(v) != oracle.Int64() {
					t.Fatal(v, e)
				}
			} else {
				v, e := m.Int32(&a)
				if modularSourceDebug {
					if !errors.Is(e, ErrIntegerAssertion) {
						t.Fatal(e)
					}
				} else {
					n := roundPublicationDecode(t, m.z, &a.num)
					want := int32(n.Int64()) // every bounded fixture NUM fits int64
					if e != nil || v != want {
						t.Fatal("release numerator signed narrowing", f, v, want, e)
					}
				}
			}
			m.drop(&a)
		}
		unreduced := Rational{num: NewInt(2), den: NewInt(2)}
		if ok, e := m.IsInt64(&unreduced); e != nil || ok {
			t.Fatal("structural integer test", ok, e)
		}
		if v, e := m.Int64(&unreduced); modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal(e)
			}
		} else if e != nil || v != 2 {
			t.Fatal("release unreduced numerator", v, e)
		}
		negative := Rational{num: NewInt(-1), den: NewInt(1)}
		if v, e := m.Uint64(&negative); modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal("rational accessor debug assertion", e)
			}
		} else if e != nil || v != math.MaxUint64 {
			t.Fatal("release inline negative unsigned cast", v, e)
		}
	}
}
func TestRationalInspectionExtractionAndThinAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, getDen := range []bool{false, true} {
			for alias := 0; alias < 3; alias++ {
				a := NewRational()
				m.SetString(&a, "-18446744073709551617/3")
				var out Int
				dst := &out
				if alias == 1 {
					dst = &a.num
				}
				if alias == 2 {
					dst = &a.den
				}
				want := "-18446744073709551617"
				if getDen {
					want = "3"
				}
				var e error
				if getDen {
					e = m.Denominator(&a, dst)
				} else {
					e = m.Numerator(&a, dst)
				}
				if e != nil || integerString(t, m.z, dst) != want {
					t.Fatal(getDen, alias, e)
				}
				m.drop(&a)
				m.z.Del(&out)
			}
		}
		for _, getDen := range []bool{false, true} {
			for _, alias := range []bool{false, true} {
				a, out := NewRational(), NewRational()
				m.SetString(&a, "-18446744073709551617/3")
				m.SetString(&out, "5/7")
				dst := &out
				if alias {
					dst = &a
				}
				var e error
				if getDen {
					e = m.DenominatorRational(&a, dst)
				} else {
					e = m.NumeratorRational(&a, dst)
				}
				want := "-18446744073709551617"
				if getDen {
					want = "3"
				}
				if e != nil || rationalString(t, m, dst) != want {
					t.Fatal(getDen, alias, e)
				}
				m.drop(&a)
				m.drop(&out)
			}
		}
		a := Rational{num: NewInt(2), den: NewInt(-3)}
		if e := m.SetInteger(&a, &a.den); e != nil || rationalString(t, m, &a) != "-3" {
			t.Fatal("component setter", e)
		}
		if e := m.SetUint64(&a, math.MaxUint64); e != nil || rationalString(t, m, &a) != "18446744073709551615" {
			t.Fatal(e)
		}
		limbs := []uint32{1, 0, 1}
		if e := m.SetLimbs(&a, limbs); e != nil || rationalString(t, m, &a) != "18446744073709551617" {
			t.Fatal(e)
		}
		limbs[0] = 7
		if e := m.Inc(&a); e != nil {
			t.Fatal(e)
		}
		if e := m.Dec(&a); e != nil || rationalString(t, m, &a) != "18446744073709551617" {
			t.Fatal(e)
		}
		m.SetFraction(&a, 2, 3)
		v := NewInt(-5)
		if e := m.SwapNumerator(&v, &a); e != nil || v.val != 2 || rationalString(t, m, &a) != "-5/3" {
			t.Fatal(e)
		}
		if e := m.SwapNumerator(&a.den, &a); e != nil || rationalString(t, m, &a) != "3/-5" {
			t.Fatal("raw denominator swap", e)
		}
		m.drop(&a)
	}
}
func TestRationalInspectionStatisticsFloatAndDomains(t *testing.T) {
	m, _ := rationalManager(t, false)
	for n := int64(-30); n <= 30; n++ {
		for d := int64(1); d <= 9; d++ {
			a := Rational{num: NewInt(int32(n)), den: NewInt(int32(d))}
			bn := uint32(new(big.Int).Abs(big.NewInt(n)).BitLen())
			if bn == 0 {
				bn = 1
			}
			bd := uint32(big.NewInt(d).BitLen())
			want := bn
			if d != 1 {
				want += bd
			}
			got, e := m.Bitsize(&a)
			storage, se := m.StorageSize(&a)
			value, ve := m.Float64(&a)
			if e != nil || se != nil || ve != nil || got != want || storage != 2 || value != float64(n)/float64(d) {
				t.Fatal(n, d, got, storage, value, e, se, ve)
			}
		}
	}
	a := NewRational()
	m.z.SetLimbs(&a.num, []uint32{0, 1})
	m.z.SetInt64(&a.den, 2)
	if v, e := m.Float64(&a); e != nil || v != float64(math.MaxUint32)/2 {
		t.Fatal("source floating radix", v, e)
	}
	if s, e := m.StorageSize(&a); e != nil || s != 4 {
		t.Fatal(s, e)
	}
	if n, e := m.Bitsize(&a); e != nil || n != 35 {
		t.Fatal(n, e)
	}
	a.den = NewInt(-2)
	if v, e := m.Float64(&a); modularSourceDebug {
		if !errors.Is(e, ErrRationalFloatDomain) {
			t.Fatal("source positive floating denominator assertion", e)
		}
	} else if e != nil || v != -float64(math.MaxUint32)/2 {
		t.Fatal("release source floating radix/negative divisor", v, e)
	}
	m.z.Del(&a.num)
	var raw Rational
	if _, e := m.Hash(&raw); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	valid := NewRational()
	m.Close()
	if _, e := m.IsInt(&valid); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestRationalInspectionFloatExceptionalBranches(t *testing.T) {
	m, _ := rationalManager(t, false)
	a := NewRational()
	limbs := make([]uint32, 35)
	limbs[0] = 1
	limbs[34] = 1
	if e := m.z.SetLimbs(&a.num, limbs); e != nil {
		t.Fatal(e)
	}
	// Source Inf*zero inside the digit loop propagates NaN; a valid denominator
	// does not classify a NaN numerator as a domain failure.
	if v, e := m.Float64(&a); e != nil || !math.IsNaN(v) {
		t.Fatal("numerator NaN", v, e)
	}
	if e := m.z.SetLimbs(&a.den, limbs); e != nil {
		t.Fatal(e)
	}
	if v, e := m.Float64(&a); modularSourceDebug {
		if !errors.Is(e, ErrRationalFloatDomain) {
			t.Fatal("converted denominator NaN assertion", e)
		}
	} else if e != nil || !math.IsNaN(v) {
		t.Fatal("release NaN denominator quotient", v, e)
	}
	for i := range limbs {
		limbs[i] = 1
	}
	if e := m.z.SetLimbs(&a.num, limbs); e != nil {
		t.Fatal(e)
	}
	m.z.SetInt64(&a.den, 1)
	if v, e := m.Float64(&a); e != nil || !math.IsInf(v, 1) {
		t.Fatal("valid Inf numerator", v, e)
	}
	if e := m.z.SetLimbs(&a.den, limbs); e != nil {
		t.Fatal(e)
	}
	if v, e := m.Float64(&a); e != nil || !math.IsNaN(v) {
		t.Fatal("valid Inf/Inf", v, e)
	}
	m.z.SetInt64(&a.num, 1)
	if v, e := m.Float64(&a); e != nil || v != 0 {
		t.Fatal("finite/Inf", v, e)
	}
}
func TestRationalInspectionGuardResultAndBorrowLifetime(t *testing.T) {
	m, _ := rationalManager(t, false)
	var raw Rational
	for _, call := range []func(*Rational) (bool, error){m.IsPos, m.IsNeg, m.IsZero, m.IsNonPos, m.IsNonNeg, m.IsOne, m.IsMinusOne, m.IsEven} {
		if got, e := call(&raw); !errors.Is(e, ErrUninitializedRational) || got {
			t.Fatal("invalid bool result", got, e)
		}
	}
	for _, call := range []func(*Rational) (bool, error){m.IsInt, m.IsInt64, m.IsUint64, m.IsUnsigned, m.IsInt32} {
		if got, e := call(&raw); e != nil || got {
			t.Fatal("source scalar fit predicate short-circuits DEN0", got, e)
		}
	}
	a := NewRational()
	m.SetString(&a, "18446744073709551617/3")
	var out Int
	if e := m.Numerator(&a, &out); e != nil {
		t.Fatal(e)
	}
	m.Reset(&a)
	if got := integerString(t, m.z, &out); got != "18446744073709551617" {
		t.Fatal("extraction borrowed ownership", got)
	}
	if e := m.SwapNumerator(&out, &a); e != nil {
		t.Fatal(e)
	}
	if out.val != 0 || rationalString(t, m, &a) != "18446744073709551617" {
		t.Fatal("swap transfer")
	}
	m.z.Del(&out)
	stale := NewRational()
	m.SetString(&stale, "18446744073709551617/5")
	snapshot := stale
	m.Del(&stale)
	if _, e := m.Hash(&snapshot); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale numeral", e)
	}
}
func TestRationalInspectionEveryFailurePrefix(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for operation := 0; operation < 9; operation++ {
			completed, failures := false, 0
			for fault := int64(0); fault < 192; fault++ {
				b := arena.NewBudget(1 << 25)
				m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}}}, Synchronized: synch})
				if e != nil {
					t.Fatal(e)
				}
				limbs := make([]uint32, 130)
				for i := range limbs {
					limbs[i] = 0x12345678
				}
				limbs[129] = 1
				a, out := NewRational(), NewRational()
				var z, integerOut Int
				m.z.SetLimbs(&a.num, limbs)
				m.z.SetLimbs(&a.den, limbs[:129])
				m.z.SetLimbs(&z, limbs)
				m.SetFraction(&out, 3, 5)
				// Fill the constructor's spare capacity-eight Words slots so
				// UINT64 publication actually needs a new physical reservation.
				var bank [2]Int
				for i := range bank {
					if e = m.z.SetUint64(&bank[i], 1<<32); e != nil {
						t.Fatal(e)
					}
				}
				beforeA, beforeOut, beforeZ, beforeInt := a, out, z, integerOut
				b.FailAfter(fault)
				switch operation {
				case 0:
					e = m.Numerator(&a, &integerOut)
				case 1:
					e = m.Denominator(&a, &integerOut)
				case 2:
					e = m.NumeratorRational(&a, &out)
				case 3:
					e = m.DenominatorRational(&a, &out)
				case 4:
					e = m.SetUint64(&out, math.MaxUint64)
				case 5:
					e = m.SetInteger(&out, &z)
				case 6:
					e = m.SetLimbs(&out, limbs)
				case 7:
					e = m.Inc(&a)
				case 8:
					e = m.Dec(&a)
				}
				// Independently enumerate caller/base/source-cache roots immediately; no
				// formatter or Close is allowed to hide a lost temporary. This existing
				// test audit lists only known concrete ownership fields, never all live slots.
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den, &z, &integerOut, &bank[0], &bank[1]})
				b.FailAfter(-1)
				if e != nil {
					failures++
					if !numericPhysicalReservationRefusal(e) || (operation < 7 && a != beforeA) || z != beforeZ {
						t.Fatal("failure changed source", synch, operation, fault, e)
					}
					// big_set publishes sign and a failed owning descriptor before
					// its allocation returns; reset_denominator follows only success.
					wantOut, wantInteger := beforeOut, beforeInt
					switch operation {
					case 0, 1:
						wantInteger = Int{val: 1, cell: m.z.failedRef}
					case 2, 3, 5, 6:
						wantOut.num = Int{val: 1, cell: m.z.failedRef}
					}
					if out != wantOut || integerOut != wantInteger {
						t.Fatal("source allocation prefix", synch, operation, fault, out, integerOut)
					}
					if operation == 0 || operation == 1 {
						if _, readError := m.z.Sign(&integerOut); !errors.Is(readError, arena.ErrInvalidSlot) {
							t.Fatal("failed descriptor readable", readError)
						}
					}
					if operation == 7 || operation == 8 {
						an, ad := new(big.Int), new(big.Int)
						for i := len(limbs) - 1; i >= 0; i-- {
							an.Lsh(an, 32)
							an.Add(an, new(big.Int).SetUint64(uint64(limbs[i])))
						}
						for i := 128; i >= 0; i-- {
							ad.Lsh(ad, 32)
							ad.Add(ad, new(big.Int).SetUint64(uint64(limbs[i])))
						}
						pn := new(big.Int).Set(an)
						if operation == 7 {
							pn.Add(pn, ad)
						} else {
							pn.Sub(pn, ad)
						}
						addPublicationExpectRaw(t, m, &a, an.String(), ad.String(), pn, ad)
					} else {
						inspectionAssertMagnitude(t, m.z, &a.num, limbs)
						inspectionAssertMagnitude(t, m.z, &a.den, limbs[:129])
					}
					inspectionAssertMagnitude(t, m.z, &z, limbs)
				} else {
					completed = true
					switch operation {
					case 0:
						inspectionAssertMagnitude(t, m.z, &integerOut, limbs)
					case 1:
						inspectionAssertMagnitude(t, m.z, &integerOut, limbs[:129])
					case 2, 5, 6:
						inspectionAssertMagnitude(t, m.z, &out.num, limbs)
					case 3:
						inspectionAssertMagnitude(t, m.z, &out.num, limbs[:129])
					case 4:
						value, readError := m.Uint64(&out)
						if readError != nil || value != math.MaxUint64 {
							t.Fatal("successful unsigned setter", value, readError)
						}
					case 7, 8:
						n, d := new(big.Int), new(big.Int)
						for i := len(limbs) - 1; i >= 0; i-- {
							n.Lsh(n, 32).Add(n, new(big.Int).SetUint64(uint64(limbs[i])))
							if i < len(limbs)-1 {
								d.Lsh(d, 32).Add(d, new(big.Int).SetUint64(uint64(limbs[i])))
							}
						}
						want := new(big.Rat).SetFrac(n, d)
						if operation == 7 {
							want.Add(want, big.NewRat(1, 1))
						} else {
							want.Sub(want, big.NewRat(1, 1))
						}
						if rationalString(t, m, &a) != want.RatString() {
							t.Fatal("successful normalized increment/decrement", operation)
						}
					}
					if operation >= 2 && operation <= 6 && out.den != NewInt(1) {
						t.Fatal("successful setter denominator", operation, out.den)
					}
				}
				m.Close()
				if b.Used() != 0 {
					t.Fatal("failure graph Close", b.Used())
				}
				if completed {
					break
				}
			}
			if !completed || failures == 0 {
				t.Fatal("incomplete inspection failure sweep", synch, operation, failures)
			}
		}
	}
}
func TestRationalInspectionWarmAllocationAndIndependentValues(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		var n, d Int
		m.SetString(&a, "-18446744073709551617/3")
		m.Numerator(&a, &n)
		m.Denominator(&a, &d)
		// Cross-field reads do not reserve storage or allocate Go objects.
		used := b.Used()
		b.FailAfter(0)
		var e error
		allocs := testing.AllocsPerRun(30, func() {
			m.Sign(&a)
			m.IsInt(&a)
			m.IsOne(&a)
			m.IsMinusOne(&a)
			m.IsPos(&a)
			m.IsNeg(&a)
			m.IsZero(&a)
			m.IsNonPos(&a)
			m.IsNonNeg(&a)
			m.IsInt64(&a)
			m.IsUint64(&a)
			m.IsUnsigned(&a)
			m.IsInt32(&a)
			m.IsEven(&a)
			m.IsPowerOfTwo(&a)
			m.Bitsize(&a)
			m.StorageSize(&a)
			m.Float64(&a)
			_, e = m.Hash(&a)
		})
		b.FailAfter(-1)
		if e != nil || allocs != 0 || b.Used() != used {
			t.Fatal("pure inspection", synch, allocs, e)
		}
		calls := []func() error{func() error { return m.Numerator(&a, &n) }, func() error { return m.Denominator(&a, &d) }, func() error { return m.NumeratorRational(&a, &out) }, func() error { return m.DenominatorRational(&a, &out) }, func() error { return m.SetInteger(&out, &n) }, func() error { return m.SetUint64(&out, math.MaxUint64) }, func() error { return m.SetLimbs(&out, []uint32{1, 0, 1}) }, func() error { return m.Inc(&a) }, func() error { return m.Dec(&a) }}
		for index, call := range calls {
			for i := 0; i < 4; i++ {
				if e = call(); e != nil {
					t.Fatal(e)
				}
			}
			beforeWarm := numericWarmProfileSnapshot(t, m.z, b)
			allocs = testing.AllocsPerRun(20, func() { e = call() })
			b.FailAfter(-1)
			if e != nil {
				t.Fatal("warm inspection mutation", synch, allocs, e)
			}
			numericWarmProfileAudit(t, m.z, b, beforeWarm, allocs, m.z.allocator.Stats().DirectObjects > 0 && index >= 7)
		}
		for _, v := range []int64{math.MinInt64, -4294967296, -2147483648, -1, 0, 1, 2147483647, 4294967295, math.MaxInt64} {
			x := NewRational()
			m.SetInt64(&x, v)
			signed, se := m.Int64(&x)
			fits, fe := m.IsInt64(&x)
			if se != nil || fe != nil || signed != v || !fits {
				t.Fatal(v, signed, se, fe)
			}
			unsigned, ue := m.Uint64(&x)
			if v >= 0 {
				if ue != nil || unsigned != uint64(v) {
					t.Fatal(v, unsigned, ue)
				}
			} else if modularSourceDebug {
				if !errors.Is(ue, ErrIntegerAssertion) {
					t.Fatal(v, ue)
				}
			} else {
				want := uint64(v)
				if !x.num.IsSmall() {
					want = uint64(-v)
				}
				if ue != nil || unsigned != want {
					t.Fatal("release numerator unsigned shape", v, unsigned, want, ue)
				}
			}
			m.drop(&x)
		}
		m.z.Del(&n)
		m.z.Del(&d)
	}
}

func TestRationalInspectionRawActionsAndIndependentIncrement(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, out := Rational{num: NewInt(2), den: NewInt(4)}, NewRational()
		if e := m.NumeratorRational(&a, &out); e != nil || rationalString(t, m, &out) != "2" {
			t.Fatal("unreduced numerator", e)
		}
		if e := m.DenominatorRational(&a, &out); e != nil || rationalString(t, m, &out) != "4" {
			t.Fatal("unreduced denominator", e)
		}
		a = Rational{num: NewInt(2), den: NewInt(-3)}
		if e := m.DenominatorRational(&a, &out); e != nil || rationalString(t, m, &out) != "-3" {
			t.Fatal("signed denominator copy", e)
		}
		if e := m.Inc(&a); e != nil || rationalString(t, m, &a) != "-1/-3" {
			t.Fatal("source raw increment", e)
		}
		if e := m.Dec(&a); e != nil || rationalString(t, m, &a) != "2/-3" {
			t.Fatal("source raw decrement", e)
		}
		// The source low-level component alias can create denominator zero; do not
		// normalize or retroactively reject the completed Copy action.
		a = Rational{num: NewInt(0), den: NewInt(7)}
		if e := m.Numerator(&a, &a.den); e != nil || a.den.val != 0 {
			t.Fatal("source zero-denominator component alias", e)
		}
		if got, e := m.IsInt(&a); e != nil || got {
			t.Fatal("source structural denominator-zero predicate", got, e)
		}
		a = NewRational()
		for n := int64(-9); n <= 9; n++ {
			for d := uint64(1); d <= 7; d++ {
				m.SetFraction(&a, n, d)
				oracle := new(big.Rat).SetFrac(big.NewInt(n), new(big.Int).SetUint64(d))
				oracle.Add(oracle, new(big.Rat).SetInt64(1))
				if e := m.Inc(&a); e != nil || rationalString(t, m, &a) != oracle.RatString() {
					t.Fatal("independent increment", n, d, e)
				}
				oracle.Sub(oracle, new(big.Rat).SetInt64(1))
				if e := m.Dec(&a); e != nil || rationalString(t, m, &a) != oracle.RatString() {
					t.Fatal("independent decrement", n, d, e)
				}
			}
		}
	}
}
func TestRationalInspectionAliasedCopyFailureCleanup(t *testing.T) {
	for operation := 0; operation < 3; operation++ {
		completed, failures := false, 0
		for fault := int64(0); fault < 64; fault++ {
			m, b := rationalManager(t, false)
			a := NewRational()
			digits := make([]uint32, 130)
			digits[129] = 1
			digits[128] = 3
			m.z.SetLimbs(&a.num, digits)
			m.z.SetLimbs(&a.den, digits[:129])
			before := a
			b.FailAfter(fault)
			var e error
			switch operation {
			case 0:
				e = m.Numerator(&a, &a.den)
			case 1:
				e = m.SetInteger(&a, &a.den)
			case 2:
				e = m.DenominatorRational(&a, &a)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
			b.FailAfter(-1)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("alias failure category", operation, fault, e)
				}
				if operation != 0 || a.num != before.num || a.den.val != before.num.val || a.den.cell != m.z.failedRef {
					t.Fatal("source failed component Copy prefix", operation, fault, a, e)
				}
				inspectionAssertMagnitude(t, m.z, &a.num, digits)
			} else {
				completed = true
				if operation == 0 {
					inspectionAssertMagnitude(t, m.z, &a.num, digits)
					inspectionAssertMagnitude(t, m.z, &a.den, digits)
				} else {
					inspectionAssertMagnitude(t, m.z, &a.num, digits[:129])
					if a.den != NewInt(1) {
						t.Fatal("successful alias denominator", operation, a.den)
					}
				}
			}
			m.Close()
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if completed {
				break
			}
		}
		// Source self/component copies can reuse all existing storage; an
		// operation without a new physical reservation need not fail. The separate
		// fresh-output sweep verifies allocation failures at real growth boundaries.
		if !completed {
			t.Fatal(operation, completed, failures)
		}
	}
}

// Borrow indexed operand digits only after the live-root audit. This verifies
// unchanged payload values independently of scalar handle identity and avoids
// preparing any owner formatting workspace before cleanup evidence.
func inspectionAssertMagnitude(t *testing.T, m *IntegerManager, a *Int, want []uint32) {
	t.Helper()
	negative, digits, e := m.Decompose(a, nil)
	if e != nil || negative || len(digits) != len(want) {
		t.Fatal("changed operand payload", e)
	}
	for i := range digits {
		if digits[i] != want[i] {
			t.Fatal("changed operand digit", i)
		}
	}
}
