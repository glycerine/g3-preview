// Copyright (c) 2006 Microsoft Corporation
// Remaining source rational.h wrappers; tst_rational.cpp::tst8 assertions.
package mpnum

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestRationalServiceExtraSourceTST8(t *testing.T) {
	h := facadeNew(t)
	out := h.keep(h.s.NewValue())
	// Exactly the32integer assertions and2fraction assertions from tst8.
	fixtures := []struct {
		n, root int32
		exact   bool
	}{
		{-4, 0, false}, {-3, 0, false}, {-2, 0, false}, {-1, 0, false}, {0, 0, true}, {1, 1, true},
		{2, 2, false}, {3, 2, false}, {4, 2, true}, {5, 3, false}, {6, 3, false}, {7, 3, false}, {8, 3, false}, {9, 3, true},
		{10, 4, false}, {11, 4, false}, {12, 4, false}, {13, 4, false}, {14, 4, false}, {15, 4, false}, {16, 4, true},
		{17, 5, false}, {18, 5, false}, {19, 5, false}, {20, 5, false}, {21, 5, false}, {22, 5, false}, {23, 5, false}, {24, 5, false}, {25, 5, true}, {26, 6, false}, {36, 6, true},
	}
	for _, f := range fixtures {
		h.assert(func() bool {
			actual, e := h.s.IsIntPerfectSquare(h.integer(f.n), out)
			serviceMust(t, e)
			// Source first six predicates short-circuit onto is_zero/is_one; other
			// assertions construct a rational integer and use structural equality.
			if actual != f.exact {
				return false
			}
			matches := false
			if f.n <= 0 {
				matches = h.is_zero(out)
			} else if f.n == 1 {
				matches = h.is_one(out)
			} else {
				matches = h.eq(out, h.integer(f.root))
			}
			return actual == f.exact && matches
		})
	}
	for _, f := range []struct{ n, d, rn, rd int32 }{{1, 9, 1, 3}, {4, 9, 2, 3}} {
		h.assert(func() bool {
			actual, e := h.s.IsPerfectSquare(h.fraction(f.n, f.d), out)
			serviceMust(t, e)
			return actual && h.eq(out, h.fraction(f.rn, f.rd))
		})
	}
	if h.count != 34 {
		t.Fatal(h.count)
	}
}

func TestRationalServiceExtraRootSourcePublicationAndAliases(t *testing.T) {
	s, _ := serviceNew(t)
	for _, f := range []struct {
		in    string
		n     uint32
		want  string
		exact bool
	}{{"8/27", 3, "2/3", true}, {"-8/27", 3, "-2/3", true}, {"213/32", 5, "3/13", false}, {"8/29", 3, "2/4", false}, {"0", 3, "0", true}} {
		a := serviceValue(t, s, f.in)
		out := serviceValue(t, s, "7/13")
		exact, e := s.Root(&a, f.n, &out)
		serviceMust(t, e)
		if exact != f.exact {
			t.Fatal(f, exact)
		}
		serviceText(t, s, &out, f.want)
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&out))
	}
	for _, text := range []string{"4", "0"} {
		a := serviceValue(t, s, text)
		exact, e := s.IsPerfectSquare(&a, &a)
		serviceMust(t, e)
		if !exact {
			t.Fatal(text)
		}
		serviceText(t, s, &a, "0")
		serviceMust(t, s.Del(&a))
	}
	a := serviceValue(t, s, "4/9")
	exact, e := s.IsPerfectSquare(&a, &a)
	if exact || !errors.Is(e, ErrDivideByZero) {
		t.Fatal(exact, e)
	}
	serviceText(t, s, &a, "4/9")
	serviceMust(t, s.Del(&a))

	for _, f := range []struct {
		in     string
		degree uint32
		want   error
	}{{"1", 0, ErrRootDegreeZero}, {"-4", 2, ErrRootNegativeEven}} {
		in := serviceValue(t, s, f.in)
		out := serviceValue(t, s, "7/13")
		exact, e := s.Root(&in, f.degree, &out)
		if exact || !errors.Is(e, f.want) {
			t.Fatal(f, exact, e)
		}
		serviceText(t, s, &out, "7/13")
		serviceMust(t, s.Del(&in))
		serviceMust(t, s.Del(&out))
	}
}

func TestRationalServiceExtraNumberTheoryAndBitwise(t *testing.T) {
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "1329227995784915872903807060280344577")
	b := serviceValue(t, s, "18446744073709551617")
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	defer s.Del(&a)
	defer s.Del(&b)
	defer s.Del(&x)
	defer s.Del(&y)
	aa, _ := new(big.Int).SetString("1329227995784915872903807060280344577", 10)
	bb, _ := new(big.Int).SetString("18446744073709551617", 10)
	for op, f := range []func(*OwnedRational, *OwnedRational) (OwnedRational, error){s.BitwiseOr, s.BitwiseAnd, s.BitwiseXor} {
		v, e := f(&a, &b)
		serviceMust(t, e)
		want := new(big.Int)
		switch op {
		case 0:
			want.Or(aa, bb)
		case 1:
			want.And(aa, bb)
		case 2:
			want.Xor(aa, bb)
		}
		serviceText(t, s, &v, want.String())
		serviceMust(t, s.Del(&v))
	}
	v, e := s.BitwiseNot(129, &a)
	serviceMust(t, e)
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 129), big.NewInt(1))
	want := new(big.Int).Xor(aa, mask)
	serviceText(t, s, &v, want.String())
	serviceMust(t, s.Del(&v))
	serviceMust(t, s.SetInt64(&a, 120))
	serviceMust(t, s.SetInt64(&b, 84))
	v, e = s.LCM(&a, &b)
	serviceMust(t, e)
	serviceText(t, s, &v, "840")
	serviceMust(t, s.Del(&v))
	gcd, e := s.GCDCoefficients(&a, &b, &x, &y)
	serviceMust(t, e)
	serviceText(t, s, &gcd, "12")
	xv, e := s.Int64(&x)
	serviceMust(t, e)
	yv, e := s.Int64(&y)
	serviceMust(t, e)
	if 120*xv+84*yv != 12 {
		t.Fatal(xv, yv)
	}
	serviceMust(t, s.Del(&gcd))
}

func TestRationalServiceExtraDisplayAndProjection(t *testing.T) {
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "-1/3")
	defer s.Del(&a)
	text, e := s.Display(&a)
	serviceMust(t, e)
	if text != "-1/3" {
		t.Fatal(text)
	}
	text, e = s.DecimalString(&a, 4, false)
	serviceMust(t, e)
	if text != "-0.3333?" {
		t.Fatal(text)
	}
	dst := bytes.Repeat([]byte{'!'}, 16)
	n, e := s.FormatDecimalInto(&a, 4, true, dst)
	serviceMust(t, e)
	if string(dst[:n]) != "-0.3333" || dst[n] != '!' {
		t.Fatal(string(dst), n)
	}
	text, e = s.SMT2String(&a, false)
	serviceMust(t, e)
	if text != "(/ (- 1) 3)" {
		t.Fatal(text)
	}
	dst = make([]byte, 32)
	n, e = s.FormatSMT2Into(&a, true, dst)
	serviceMust(t, e)
	if string(dst[:n]) != "(/ (- 1.0) 3.0)" {
		t.Fatal(string(dst[:n]))
	}
	n, e = s.DisplayInto(&a, dst)
	serviceMust(t, e)
	if string(dst[:n]) != "-1/3" {
		t.Fatal(string(dst[:n]))
	}

	for _, f := range []func(*OwnedRational) (OwnedRational, error){s.ToRational, s.GetRational} {
		v, e := f(&a)
		serviceMust(t, e)
		serviceText(t, s, &v, "-1/3")
		serviceMust(t, s.Del(&v))
	}
	v, e := s.GetInfinitesimal(&a)
	serviceMust(t, e)
	serviceText(t, s, &v, "0")
	serviceMust(t, s.Del(&v))
	rational, e := s.IsRational(&a)
	serviceMust(t, e)
	if !rational {
		t.Fatal("source static true")
	}
	for _, f := range []float64{0, 1, -1, math.NaN(), math.Inf(1)} {
		v, e := s.NewDouble(f)
		if !errors.Is(e, ErrRationalDoubleUnsupported) || v.ownerID != 0 {
			t.Fatal(v, e)
		}
		if e = s.AssignDouble(&a, f); !errors.Is(e, ErrRationalDoubleUnsupported) {
			t.Fatal(e)
		}
	}
	serviceText(t, s, &a, "-1/3")
}

func TestRationalServiceExtraScopedIntegerBridge(t *testing.T) {
	s, _ := serviceNew(t)
	m, _ := integerManager(t)
	source, e := NewScopedInt(m)
	serviceMust(t, e)
	defer source.Close()
	serviceMust(t, source.SetInt64(math.MinInt64))
	value, e := s.NewFromScopedInt(source)
	serviceMust(t, e)
	defer s.Del(&value)
	serviceText(t, s, &value, "-9223372036854775808")
	// Handle/cell metadata may coincide across manager namespaces. Mutation
	// below verifies independent ownership rather than comparing raw handle bits.
	serviceMust(t, source.SetInt64(17))
	serviceText(t, s, &value, "-9223372036854775808")
	serviceMust(t, s.SetFromScopedInt(&value, source))
	serviceText(t, s, &value, "17")
	serviceMust(t, s.SetString(&value, "123456789012345678901234567891/998877665544332211"))
	serviceMust(t, s.NumeratorToScopedInt(&value, source))
	if integerString(t, m, &source.num) != "123456789012345678901234567891" {
		t.Fatal("numerator")
	}
	serviceMust(t, s.DenominatorToScopedInt(&value, source))
	if integerString(t, m, &source.num) != "998877665544332211" {
		t.Fatal("denominator")
	}
	shallow := *source
	if v, e := s.NewFromScopedInt(&shallow); v.ownerID != 0 || !errors.Is(e, ErrScopedCopy) {
		t.Fatal(v, e)
	}
	serviceMust(t, source.Close())
	if e := s.SetFromScopedInt(&value, source); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestRationalServiceExtraFailurePrefixesAndOwnership(t *testing.T) {
	for op := 0; op < 12; op++ {
		failures, success := 0, false
		for fault := int64(0); fault < 512; fault++ {
			s, budget := servicePublicationDirectOwner(t)
			a := serviceValue(t, s, "1329227995784915872903807060280344577")
			b := serviceValue(t, s, "18446744073709551617")
			if op >= 4 && op <= 8 {
				serviceMust(t, s.SetString(&a, strings.Repeat("1234567890", 25)+"7"))
				serviceMust(t, s.SetString(&b, strings.Repeat("9876543210", 20)+"9"))
			}
			x := serviceValue(t, s, "123456789012345678901234567890/998877665544332211")
			y := serviceValue(t, s, "3/7")
			oldA, e := s.String(&a)
			serviceMust(t, e)
			oldB, e := s.String(&b)
			serviceMust(t, e)
			oldX, e := s.String(&x)
			serviceMust(t, e)
			// Exhaust prepared cell slots for the pure Copy routes so the
			// next source constructor actually requires a budget reservation.
			var bank []OwnedRational
			if op == 9 || op == 10 {
				for s.manager.z.cells.Stats().Live+s.manager.z.cells.Stats().Retired < s.manager.z.cells.Stats().Slots {
					v, e := s.NewValue()
					serviceMust(t, e)
					serviceMust(t, s.Copy(&v, &a))
					bank = append(bank, v)
					v = OwnedRational{}
				}
			}
			budget.FailAfter(fault)
			var out OwnedRational
			switch op {
			case 0:
				_, e = s.Root(&a, 3, &x)
			case 1:
				_, e = s.IsPerfectSquare(&a, &x)
			case 2:
				_, e = s.IsIntPerfectSquare(&a, &x)
			case 3:
				out, e = s.LCM(&a, &b)
			case 4:
				out, e = s.GCDCoefficients(&a, &b, &x, &y)
			case 5:
				out, e = s.BitwiseOr(&a, &b)
			case 6:
				out, e = s.BitwiseAnd(&a, &b)
			case 7:
				out, e = s.BitwiseXor(&a, &b)
			case 8:
				out, e = s.BitwiseNot(193, &a)
			case 9:
				out, e = s.ToRational(&a)
			case 10:
				out, e = s.GetRational(&a)
			case 11:
				_, e = s.DecimalString(&x, 1024, false)
			}
			roots := []*OwnedRational{&a, &b, &x, &y, &out}
			for i := range bank {
				roots = append(roots, &bank[i])
			}
			facadeAudit(t, s, roots...) // pre-render, pre-Del
			budget.FailAfter(-1)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
					t.Fatal(op, fault, out, e)
				}
				// Manager ExtendedGCD deliberately publishes output denominator/coefficient
				// prefixes. Other operations keep caller output unchanged on failure.
				if op != 4 {
					serviceIndependentText(t, s, &x, oldX)
				} else if !isOneInteger(&x.raw.den) || !isOneInteger(&y.raw.den) {
					t.Fatal("source coefficient denominator prefix missing")
				}
			} else {
				success = true
			}
			serviceIndependentText(t, s, &a, oldA)
			serviceIndependentText(t, s, &b, oldB)
			if out.ownerID != 0 {
				serviceMust(t, s.Del(&out))
			}
			for i := range bank {
				serviceMust(t, s.Del(&bank[i]))
			}
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.Del(&b))
			serviceMust(t, s.Del(&x))
			serviceMust(t, s.Del(&y))
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal(op, fault, budget.Used())
			}
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("sweep lacks failure/completion", op, failures, success)
		}
	}
}

func TestRationalServiceExtraFormattingPublicationAndProvenance(t *testing.T) {
	s, budget := serviceNew(t)
	other, _ := serviceNew(t)
	a := serviceValue(t, s, "1/3")
	foreign := serviceValue(t, other, "4")
	defer s.Del(&a)
	defer other.Del(&foreign)
	dst := bytes.Repeat([]byte{'!'}, 1)
	for _, e := range []error{extraFormatError(s.DisplayInto(&a, dst)), extraFormatError(s.FormatInto(&a, dst)), extraFormatError(s.FormatDecimalInto(&a, 30, false, dst)), extraFormatError(s.FormatSMT2Into(&a, false, dst))} {
		if !errors.Is(e, ErrShortBuffer) || string(dst) != "!" {
			t.Fatal(e, string(dst))
		}
	}
	used := budget.Used()
	budget.FailAfter(0)
	for _, f := range []func(*OwnedRational, *OwnedRational) (OwnedRational, error){s.LCM, s.BitwiseOr, s.BitwiseAnd, s.BitwiseXor} {
		v, e := f(&a, &foreign)
		if !errors.Is(e, ErrRationalServiceOwner) || v.ownerID != 0 {
			t.Fatal(v, e)
		}
	}
	for _, e := range []error{extraBoolError(s.Root(&a, 2, &foreign)), extraBoolError(s.IsPerfectSquare(&foreign, &a)), s.AssignDouble(&foreign, 1), extraFormatError(s.FormatDecimalInto(&foreign, 3, false, dst))} {
		if !errors.Is(e, ErrRationalServiceOwner) {
			t.Fatal(e)
		}
	}
	budget.FailAfter(-1)
	if budget.Used() != used {
		t.Fatal("provenance allocated")
	}
}
func extraFormatError(_ int, e error) error { return e }
func extraBoolError(_ bool, e error) error  { return e }

func TestRationalServiceExtraWarmReuse(t *testing.T) {
	s, budget := serviceNew(t)
	a := serviceValue(t, s, "18446744073709551617")
	b := serviceValue(t, s, "4294967295")
	out, _ := s.NewValue()
	defer s.Del(&a)
	defer s.Del(&b)
	defer s.Del(&out)
	dst := make([]byte, 128)
	run := func() {
		v, e := s.BitwiseOr(&a, &b)
		if e != nil {
			panic(e)
		}
		if e = s.Del(&v); e != nil {
			panic(e)
		}
		v, e = s.ToRational(&a)
		if e != nil {
			panic(e)
		}
		if e = s.Del(&v); e != nil {
			panic(e)
		}
		if _, e = s.FormatInto(&a, dst); e != nil {
			panic(e)
		}
	}
	for i := 0; i < 4; i++ {
		run()
	}
	beforeWarm := numericWarmProfileSnapshot(t, s.manager.z, budget)
	n := testing.AllocsPerRun(30, run)
	numericWarmProfileAudit(t, s.manager.z, budget, beforeWarm, n, s.manager.z.allocator.Stats().DirectObjects > 0)
}

func TestRationalServiceExtraIndependentRootsAndCoefficients(t *testing.T) {
	s, _ := serviceNew(t)
	a, _ := s.NewValue()
	b, _ := s.NewValue()
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	out, _ := s.NewValue()
	defer s.Del(&a)
	defer s.Del(&b)
	defer s.Del(&x)
	defer s.Del(&y)
	defer s.Del(&out)
	for degree := uint32(1); degree <= 5; degree++ {
		for n := int64(0); n <= 8; n++ {
			for d := int64(1); d <= 6; d++ {
				nn := new(big.Int).Exp(big.NewInt(n), big.NewInt(int64(degree)), nil)
				dd := new(big.Int).Exp(big.NewInt(d), big.NewInt(int64(degree)), nil)
				input := new(big.Rat).SetFrac(nn, dd)
				serviceMust(t, s.SetString(&a, input.RatString()))
				exact, e := s.Root(&a, degree, &out)
				serviceMust(t, e)
				if !exact {
					t.Fatal(degree, n, d)
				}
				want := new(big.Rat).SetFrac(big.NewInt(n), big.NewInt(d))
				serviceText(t, s, &out, want.RatString())
				exact, e = s.Root(&a, degree, &a)
				serviceMust(t, e)
				if !exact {
					t.Fatal("root input/output alias", degree, n, d)
				}
				serviceText(t, s, &a, want.RatString())
			}
		}
	}
	for ai := int64(-15); ai <= 15; ai++ {
		for bi := int64(-11); bi <= 11; bi++ {
			serviceMust(t, s.SetInt64(&a, ai))
			serviceMust(t, s.SetInt64(&b, bi))
			gcd, e := s.GCDCoefficients(&a, &b, &x, &y)
			serviceMust(t, e)
			g, e := s.Int64(&gcd)
			serviceMust(t, e)
			xx, e := s.Int64(&x)
			serviceMust(t, e)
			yy, e := s.Int64(&y)
			serviceMust(t, e)
			want := new(big.Int).GCD(nil, nil, big.NewInt(ai), big.NewInt(bi)).Int64()
			if g != want || ai*xx+bi*yy != g {
				t.Fatal(ai, bi, g, xx, yy, want)
			}
			serviceMust(t, s.Del(&gcd))
		}
	}
	// Source manager coefficients alias one another, so their publication order
	// is deliberately not replaced by staged independent output records.
	serviceMust(t, s.SetInt64(&a, 120))
	serviceMust(t, s.SetInt64(&b, 84))
	gcd, e := s.GCDCoefficients(&a, &b, &x, &x)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, errExtendedGCDInvariant) || gcd.ownerID != 0 {
			t.Fatal("source debug assertion before gcd value return", e)
		}
	} else {
		serviceMust(t, e)
		serviceText(t, s, &gcd, "12")
		serviceMust(t, s.Del(&gcd))
	}
	// Source actions independently imply shared coefficient2 after the three
	// quotient steps1,2,3. This is not a C++ execution or mathematical oracle.
	serviceText(t, s, &x, "2")
	for _, f := range []struct {
		av, bv       int64
		alias        int
		wantX, wantY string
	}{{120, 84, 0, "2", "3"}, {120, -84, 1, "-2", "3"}} {
		serviceMust(t, s.SetInt64(&a, f.av))
		serviceMust(t, s.SetInt64(&b, f.bv))
		xp, yp := &x, &y
		if f.alias == 0 {
			xp = &a
		} else {
			yp = &b
		}
		gcd, e := s.GCDCoefficients(&a, &b, xp, yp)
		if sourceExtendedGCDDebug {
			if !errors.Is(e, errExtendedGCDInvariant) || gcd.ownerID != 0 {
				t.Fatal("source debug assertion before gcd value return", f, e)
			}
		} else {
			serviceMust(t, e)
			serviceText(t, s, &gcd, "12")
		}
		serviceText(t, s, xp, f.wantX)
		serviceText(t, s, yp, f.wantY)
		if !sourceExtendedGCDDebug {
			serviceMust(t, s.Del(&gcd))
		}
	}
	for _, f := range []struct {
		a, b int64
		want string
	}{{-6, 1, "-6"}, {1, -6, "-6"}, {-6, -6, "-6"}, {-6, 4, "-12"}, {6, -4, "-12"}, {-6, -4, "12"}, {0, 0, "0"}, {0, 6, "0"}} {
		serviceMust(t, s.SetInt64(&a, f.a))
		serviceMust(t, s.SetInt64(&b, f.b))
		v, e := s.LCM(&a, &b)
		serviceMust(t, e)
		serviceText(t, s, &v, f.want)
		serviceMust(t, s.Del(&v))
	}

}

func TestRationalServiceExtraIntegerBridgeFailureCleanup(t *testing.T) {
	sourceManager, _ := integerManager(t)
	source, e := NewScopedInt(sourceManager)
	serviceMust(t, e)
	defer source.Close()
	serviceMust(t, sourceManager.SetString(&source.num, strings.Repeat("1234567890", 25)+"7"))
	for op := 0; op < 2; op++ {
		failures, success := 0, false
		for fault := int64(0); fault < 100; fault++ {
			s, budget := serviceNew(t)
			a := serviceValue(t, s, "3/7")
			budget.FailAfter(fault)
			var out OwnedRational
			if op == 0 {
				out, e = s.NewFromScopedInt(source)
			} else {
				e = s.SetFromScopedInt(&a, source)
			}
			budget.FailAfter(-1)
			facadeAudit(t, s, &a, &out)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
					t.Fatal(op, fault, e)
				}
				serviceText(t, s, &a, "3/7")
			} else {
				success = true
			}
			if out.ownerID != 0 {
				serviceMust(t, s.Del(&out))
			}
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal(op, fault, budget.Used())
			}
			if success {
				break
			}
		}
		if failures == 0 || !success {
			t.Fatal(op, failures, success)
		}
	}
	// Extraction allocates in the explicitly supplied scoped integer owner.
	for denominator := 0; denominator < 2; denominator++ {
		failures, success := 0, false
		for fault := int64(0); fault < 100; fault++ {
			s, _ := serviceNew(t)
			a := serviceValue(t, s, strings.Repeat("1234567890", 25)+"7/"+strings.Repeat("9876543210", 20)+"9")
			m, budget := integerManager(t)
			dst, e := NewScopedInt(m)
			serviceMust(t, e)
			serviceMust(t, dst.SetInt64(17))
			cells, words := m.cells.Stats().Live, m.words.Stats().Live
			budget.FailAfter(fault)
			if denominator == 0 {
				e = s.NumeratorToScopedInt(&a, dst)
			} else {
				e = s.DenominatorToScopedInt(&a, dst)
			}
			budget.FailAfter(-1)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) || m.cells.Stats().Live != cells || m.words.Stats().Live != words {
					t.Fatal(denominator, fault, e, "unreachable destination-owner temp")
				}
				if dst.num.val != 17 || !dst.num.IsSmall() {
					t.Fatal("failed extraction mutated destination")
				}
			} else {
				success = true
			}
			facadeAudit(t, s, &a)
			serviceMust(t, dst.Close())
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.Close())
			if success {
				break
			}
		}
		if failures == 0 || !success {
			t.Fatal("extraction sweep", denominator, failures, success)
		}
	}
}

func TestRationalServiceExtraScopedRationalConstructor(t *testing.T) {
	s, _ := serviceNew(t)
	m, _ := rationalManager(t, true)
	source, e := NewScopedRational(m)
	serviceMust(t, e)
	defer source.Close()
	n, d := NewInt(6), NewInt(-9)
	serviceMust(t, m.RatDiv(&n, &d, &source.num))
	value, e := s.NewFromScopedRational(source)
	serviceMust(t, e)
	defer s.Del(&value)
	serviceText(t, s, &value, "2/-3")
	serviceMust(t, source.SetInt64(17))
	serviceText(t, s, &value, "2/-3")
	serviceMust(t, s.SetFromScopedRational(&value, source))
	serviceText(t, s, &value, "17")
	shallow := *source
	if v, e := s.NewFromScopedRational(&shallow); v.ownerID != 0 || !errors.Is(e, ErrScopedCopy) {
		t.Fatal(v, e)
	}
	serviceMust(t, m.SetString(&source.num, "123456789012345678901234567891/998877665544332211"))
	for op := 0; op < 2; op++ {
		failures, success := 0, false
		for fault := int64(0); fault < 100; fault++ {
			dest, budget := serviceNew(t)
			a := serviceValue(t, dest, "3/7")
			var out OwnedRational
			budget.FailAfter(fault)
			if op == 0 {
				out, e = dest.NewFromScopedRational(source)
			} else {
				e = dest.SetFromScopedRational(&a, source)
			}
			budget.FailAfter(-1)
			facadeAudit(t, dest, &a, &out)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
					t.Fatal(op, fault, e)
				}
				serviceText(t, dest, &a, "3/7")
			} else {
				success = true
			}
			if out.ownerID != 0 {
				serviceMust(t, dest.Del(&out))
			}
			serviceMust(t, dest.Del(&a))
			serviceMust(t, dest.Close())
			if budget.Used() != 0 {
				t.Fatal(op, fault, budget.Used())
			}
			if success {
				break
			}
		}
		if failures == 0 || !success {
			t.Fatal(op, failures, success)
		}
	}
}

func TestRationalServiceExtraToScopedRationalOwnershipAndFailures(t *testing.T) {
	for fault := int64(0); fault < 100; fault++ {
		s, _ := serviceNew(t)
		a := serviceValue(t, s, "123456789012345678901234567891/998877665544332211")
		m, budget := rationalManager(t, true)
		dst, e := NewScopedRational(m)
		serviceMust(t, e)
		serviceMust(t, m.SetFraction(&dst.num, 3, 7))
		budget.FailAfter(fault)
		e = s.ToScopedRational(&a, dst)
		budget.FailAfter(-1)
		qopsAuditLiveOwnership(t, m, []*Int{&dst.num.num, &dst.num.den})
		facadeAudit(t, s, &a)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fault, e)
			}
			if rationalString(t, m, &dst.num) != "3/7" {
				t.Fatal("failed copy mutated scoped destination")
			}
		}
		elseSuccess := e == nil
		serviceMust(t, dst.Close())
		m.Close()
		if budget.Used() != 0 {
			t.Fatal(fault, budget.Used())
		}
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Close())
		if elseSuccess {
			if fault == 0 {
				t.Fatal("no actual failure reservation")
			}
			break
		}
		if fault == 99 {
			t.Fatal("no copy completion")
		}
	}
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "1/3")
	defer s.Del(&a)
	m, _ := rationalManager(t, true)
	dst, e := NewScopedRational(m)
	serviceMust(t, e)
	defer dst.Close()
	serviceMust(t, s.manager.z.SetInt64(&a.raw.den, -3))
	serviceMust(t, s.ToScopedRational(&a, dst))
	if rationalString(t, m, &dst.num) != "1/-3" {
		t.Fatal("normalized source accessor copy")
	}
	shallow := *dst
	if e := s.ToScopedRational(&a, &shallow); !errors.Is(e, ErrScopedCopy) {
		t.Fatal(e)
	}
}
