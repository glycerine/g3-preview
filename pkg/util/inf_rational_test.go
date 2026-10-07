package util

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func infValue(t *testing.T, s *RationalService, n, d int64, eps string) OwnedInfRational {
	t.Helper()
	v, e := s.NewInfValue()
	serviceMust(t, e)
	serviceMust(t, s.InfSetFraction(&v, n, uint64(d)))
	if eps != "0" {
		a := serviceValue(t, s, new(big.Rat).SetFrac(big.NewInt(n), big.NewInt(d)).RatString())
		b := serviceValue(t, s, eps)
		serviceMust(t, s.InfSetPair(&v, &a, &b))
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
	}
	return v
}
func infEq(t *testing.T, s *RationalService, a, b *OwnedInfRational) bool {
	t.Helper()
	ok, e := s.InfEqual(a, b)
	serviceMust(t, e)
	return ok
}
func infText(t *testing.T, s *RationalService, v *OwnedInfRational, want string) {
	t.Helper()
	got, e := s.InfString(v)
	if e != nil || got != want {
		t.Fatalf("inf got %q %v want %q", got, e, want)
	}
}

// Every active ENSURE in src/test/inf_rational.cpp, including helper called twice.
func TestInfRationalSource80Assertions(t *testing.T) {
	s, _ := serviceNew(t)
	checks := 0
	ensure := func(ok bool) {
		t.Helper()
		checks++
		if !ok {
			t.Fatalf("source ENSURE execution %d", checks)
		}
	}
	equal := func(a, b *OwnedInfRational) { ensure(infEq(t, s, a, b)) }
	lt := func(a, b *OwnedInfRational) bool { n, e := s.InfLess(a, b); serviceMust(t, e); return n }
	gt := func(a, b *OwnedInfRational) bool { n, e := s.InfGreater(a, b); serviceMust(t, e); return n }
	ge := func(a, b *OwnedInfRational) bool { n, e := s.InfGreaterEqual(a, b); serviceMust(t, e); return n }
	mk := func(n, d int64, eps string) OwnedInfRational { return infValue(t, s, n, d, eps) }
	zero, e := s.InfZero()
	serviceMust(t, e)
	negZero := mk(0, 1, "-1")
	ensure(lt(&negZero, &zero))
	ensure(!(ge(&negZero, &zero)))
	r1 := mk(0, 1, "0")
	r2, e := s.InfClone(&r1)
	serviceMust(t, e)
	equal(&r1, &r2)
	r3 := mk(1, 1, "0")
	r4 := mk(0, 1, "0")
	equal(&r4, &r1)
	ensure(!infEq(t, s, &r3, &r4))
	r5 := mk(0, 1, "0")
	r6 := mk(1, 1, "0")
	r7 := mk(2, 2, "0")
	r8 := mk(7, 5, "0")
	equal(&r1, &r5)
	equal(&r6, &r3)
	equal(&r7, &r3)
	rat := serviceValue(t, s, "7/5")
	r9, e := s.NewInfValue()
	serviceMust(t, e)
	serviceMust(t, s.InfSetRational(&r9, &rat))
	equal(&r8, &r9)
	serviceMust(t, s.InfReset(&r9))
	equal(&r1, &r9)
	ok, e := s.InfIsInt(&r1)
	serviceMust(t, e)
	ensure(ok)
	ok, e = s.InfIsInt(&r8)
	serviceMust(t, e)
	ensure(!ok)
	iv, e := s.InfInt64(&r1)
	serviceMust(t, e)
	ensure(iv == 0)
	serviceMust(t, s.InfCopy(&r9, &r8))
	equal(&r8, &r9)
	n, e := s.InfNumerator(&r7)
	serviceMust(t, e)
	d, e := s.InfDenominator(&r7)
	serviceMust(t, e)
	serviceMust(t, s.InfDel(&n))
	serviceMust(t, s.InfDel(&d)) // source executes constructors without ENSURE
	for _, eps := range []string{"0", "1"} {
		r := mk(7, 5, eps)
		b85 := mk(8, 5, eps)
		b75 := mk(7, 5, eps)
		ehalf := "0"
		if eps == "1" {
			ehalf = "1/2"
		}
		b710 := mk(7, 10, ehalf)
		b1710 := mk(17, 10, ehalf)
		fifth := serviceValue(t, s, "1/5")
		two := serviceValue(t, s, "2")
		half := serviceValue(t, s, "1/2")
		ififth := mk(1, 5, "0")
		ihalf := mk(1, 2, "0")
		serviceMust(t, s.InfAddRationalAssign(&r, &fifth))
		equal(&r, &b85)
		serviceMust(t, s.InfSubRationalAssign(&r, &fifth))
		equal(&r, &b75)
		serviceMust(t, s.InfAddAssign(&r, &ififth))
		equal(&r, &b85)
		serviceMust(t, s.InfSubAssign(&r, &ififth))
		equal(&r, &b75)
		serviceMust(t, s.InfDivRationalAssign(&r, &two))
		equal(&r, &b710)
		pre, e := s.InfPostInc(&r)
		serviceMust(t, e)
		equal(&pre, &b710)
		equal(&r, &b1710)
		serviceMust(t, s.InfDec(&r))
		post, e := s.InfClone(&r)
		serviceMust(t, e)
		equal(&post, &b710)
		equal(&r, &b710)
		serviceMust(t, s.InfInc(&r))
		serviceMust(t, s.InfCopy(&post, &r))
		equal(&post, &b1710)
		equal(&r, &b1710)
		serviceMust(t, s.InfDel(&pre))
		pre, e = s.InfPostDec(&r)
		serviceMust(t, e)
		equal(&pre, &b1710)
		equal(&r, &b710)
		serviceMust(t, s.InfCopy(&pre, &r))
		serviceMust(t, s.InfAddAssign(&pre, &ihalf))
		serviceMust(t, s.InfCopy(&post, &pre))
		serviceMust(t, s.InfSubAssign(&post, &ihalf))
		equal(&r, &post)
		sum, e := s.InfAdd(&r, &ihalf)
		serviceMust(t, e)
		equal(&sum, &pre)
		serviceMust(t, s.InfDel(&sum))
		serviceMust(t, s.InfCopy(&pre, &r))
		serviceMust(t, s.InfDivRationalAssign(&pre, &two))
		serviceMust(t, s.InfCopy(&post, &pre))
		serviceMust(t, s.InfDivRationalAssign(&post, &half))
		equal(&r, &post)
		prod, e := s.InfMulRational(&r, &half)
		serviceMust(t, e)
		equal(&prod, &pre)
		quot, e := s.InfDivRational(&pre, &half)
		serviceMust(t, e)
		equal(&r, &quot)
		for _, v := range []*OwnedInfRational{&r, &b85, &b75, &b710, &b1710, &ififth, &ihalf, &pre, &post, &prod, &quot} {
			serviceMust(t, s.InfDel(v))
		}
		for _, v := range []*OwnedRational{&fifth, &two, &half} {
			serviceMust(t, s.Del(v))
		}
	}
	h := mk(1, 2, "0")
	hp := mk(1, 2, "1")
	hn := mk(1, 2, "-1")
	ensure(gt(&hp, &h))
	ensure(lt(&hn, &h))
	ensure(ge(&hp, &h))
	ensure(ge(&h, &hn))
	ensure(!infEq(t, s, &hn, &h))
	ensure(!infEq(t, s, &hp, &h))
	ensure(!infEq(t, s, &hn, &hp))
	serviceMust(t, s.InfNeg(&hn))
	orig := mk(1, 2, "-1")
	opp, e := s.InfClone(&orig)
	serviceMust(t, e)
	serviceMust(t, s.InfNeg(&opp))
	equal(&hn, &opp)
	serviceMust(t, s.InfNeg(&hn))
	equal(&hn, &orig)
	signCheck := func(v *OwnedInfRational, one bool) {
		z, e := s.InfIsZero(v)
		serviceMust(t, e)
		o, e := s.InfIsOne(v)
		serviceMust(t, e)
		neg, e := s.InfIsNeg(v)
		serviceMust(t, e)
		nn, e := s.InfIsNonNeg(v)
		serviceMust(t, e)
		np, e := s.InfIsNonPos(v)
		serviceMust(t, e)
		pos, e := s.InfIsPos(v)
		serviceMust(t, e)
		ensure(z == !one && o == one && !neg && nn && np == !one && pos == one)
	}
	signCheck(&r1, false)
	signCheck(&r3, true)
	for _, tc := range []struct {
		n, d      int64
		eps, want string
	}{{1, 2, "-1", "0"}, {1, 2, "0", "0"}, {0, 1, "-1", "-1"}, {0, 1, "0", "0"}, {0, 1, "1", "0"}, {1, 1, "-1", "0"}, {1, 1, "0", "1"}, {1, 1, "1", "1"}} {
		v := mk(tc.n, tc.d, tc.eps)
		q, e := s.InfFloor(&v)
		serviceMust(t, e)
		text, e := s.String(&q)
		serviceMust(t, e)
		ensure(text == tc.want)
		serviceMust(t, s.Del(&q))
		serviceMust(t, s.InfDel(&v))
	}
	for _, tc := range []struct {
		n, d      int64
		eps, want string
	}{{1, 2, "-1", "1"}, {1, 2, "0", "1"}, {0, 1, "-1", "0"}, {0, 1, "0", "0"}, {0, 1, "1", "1"}, {1, 1, "-1", "1"}, {1, 1, "0", "1"}, {1, 1, "1", "2"}} {
		v := mk(tc.n, tc.d, tc.eps)
		q, e := s.InfCeil(&v)
		serviceMust(t, e)
		text, e := s.String(&q)
		serviceMust(t, e)
		ensure(text == tc.want)
		serviceMust(t, s.Del(&q))
		serviceMust(t, s.InfDel(&v))
	}
	x := mk(1, 2, "1")
	y := mk(1, 2, "0")
	serviceMust(t, s.InfSwap(&x, &y))
	equal(&x, &h)
	equal(&y, &hp)
	minus, e := s.InfClone(&h)
	serviceMust(t, e)
	serviceMust(t, s.InfNeg(&minus))
	ab, e := s.InfAbs(&minus)
	serviceMust(t, e)
	equal(&h, &ab)
	if checks != 80 {
		t.Fatalf("translated %d source assertion executions, want80", checks)
	}
	for _, v := range []*OwnedInfRational{&zero, &negZero, &r1, &r2, &r3, &r4, &r5, &r6, &r7, &r8, &r9, &h, &hp, &hn, &orig, &opp, &x, &y, &minus, &ab} {
		serviceMust(t, s.InfDel(v))
	}
	serviceMust(t, s.Del(&rat))
}

func TestInfRationalApproximationFrozenBranches(t *testing.T) {
	s, _ := serviceNew(t)
	x := infValue(t, s, 2, 1, "3")
	y := infValue(t, s, 5, 1, "-7")
	lo, e := s.InfLowerMult(&x, &y)
	serviceMust(t, e)
	hi, e := s.InfUpperMult(&x, &y)
	serviceMust(t, e)
	infText(t, s, &lo, "10")
	infText(t, s, &hi, "(10 +e*1)")
	serviceMust(t, s.InfDel(&lo))
	serviceMust(t, s.InfDel(&hi))
	for _, tc := range []struct {
		a      int64
		b      string
		n      uint32
		lo, hi string
	}{{2, "-1", 2, "0", "4"}, {2, "1", 2, "4", "9"}, {-2, "-1", 2, "4", "9"}, {-2, "1", 3, "-27", "-1"}, {0, "-1", 3, "-1", "1"}, {0, "1", 0, "0", "1"}, {2, "-1", 1, "(2 -e*1)", "(2 -e*1)"}} {
		v := infValue(t, s, tc.a, 1, tc.b)
		l, e := s.InfLowerPower(&v, tc.n)
		serviceMust(t, e)
		u, e := s.InfUpperPower(&v, tc.n)
		serviceMust(t, e)
		infText(t, s, &l, tc.lo)
		infText(t, s, &u, tc.hi)
		serviceMust(t, s.InfDel(&l))
		serviceMust(t, s.InfDel(&u))
		serviceMust(t, s.InfDel(&v))
	}
	root, e := s.InfLowerRoot(&x, 0)
	serviceMust(t, e)
	infText(t, s, &root, "0")
	serviceMust(t, s.InfDel(&root))
	root, e = s.InfUpperRoot(&x, 0)
	serviceMust(t, e)
	infText(t, s, &root, "(2 +e*3)")
	serviceMust(t, s.InfDel(&root))
	// b=5,d=-7,positive numerator: upper denominator 5-5/2; lower5.
	l, e := s.InfLowerDiv(&x, &y)
	serviceMust(t, e)
	u, e := s.InfUpperDiv(&x, &y)
	serviceMust(t, e)
	infText(t, s, &l, "(2/5 +e*3/5)")
	infText(t, s, &u, "(4/5 +e*6/5)")
	for _, v := range []*OwnedInfRational{&x, &y, &l, &u} {
		serviceMust(t, s.InfDel(v))
	}
}

func TestInfRationalOwnershipErrors(t *testing.T) {
	s, _ := serviceNew(t)
	other, _ := serviceNew(t)
	v := infValue(t, s, 1, 2, "-1")
	q := serviceValue(t, other, "2")
	if e := s.InfSetRational(&v, &q); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	z := infValue(t, s, 0, 1, "1")
	div, e := s.InfLowerDiv(&v, &z)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrInfRationalDomain) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
		if div.First.Num.Val != 1 || div.First.Den.Val != 0 || div.Second.Num.Val != -1 || div.Second.Den.Val != 0 {
			t.Fatal(div)
		}
		infAuditRoots(t, s, []*OwnedInfRational{&v, &z, &div}, nil)
		serviceMust(t, s.InfDel(&div))
	}
	root, e := s.InfLowerRoot(&z, 2)
	serviceMust(t, e)
	serviceMust(t, s.InfDel(&root))
	serviceMust(t, s.InfNeg(&z))
	root, e = s.InfUpperRoot(&z, 2)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrInfRationalDomain) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
		infText(t, s, &root, "(0 -e*1)")
		infAuditRoots(t, s, []*OwnedInfRational{&v, &z, &root}, nil)
		serviceMust(t, s.InfDel(&root))
	}
	serviceMust(t, s.Close())
	if _, e := s.InfHash(&v); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestInfRationalIndependentFiniteBounds(t *testing.T) {
	s, _ := serviceNew(t)
	read := func(v *OwnedInfRational, first bool) *big.Rat {
		t.Helper()
		var q OwnedRational
		var e error
		if first {
			q, e = s.InfGetRational(v)
		} else {
			q, e = s.InfGetInfinitesimal(v)
		}
		serviceMust(t, e)
		text, e := s.String(&q)
		serviceMust(t, e)
		serviceMust(t, s.Del(&q))
		r, ok := new(big.Rat).SetString(text)
		if !ok {
			t.Fatal(text)
		}
		return r
	}
	eval := func(v *OwnedInfRational) *big.Rat {
		return new(big.Rat).Add(read(v, true), new(big.Rat).Mul(read(v, false), big.NewRat(1, 10000)))
	}
	for a := int64(-2); a <= 2; a++ {
		for b := int64(-1); b <= 1; b++ {
			for c := int64(-2); c <= 2; c++ {
				for d := int64(-1); d <= 1; d++ {
					x := infValue(t, s, a, 1, big.NewInt(b).String())
					y := infValue(t, s, c, 1, big.NewInt(d).String())
					actual := new(big.Rat).Mul(eval(&x), eval(&y))
					lo, e := s.InfLowerMult(&x, &y)
					serviceMust(t, e)
					hi, e := s.InfUpperMult(&x, &y)
					serviceMust(t, e)
					if eval(&lo).Cmp(actual) > 0 || eval(&hi).Cmp(actual) < 0 {
						t.Fatalf("mult bounds %d %d %d %d", a, b, c, d)
					}
					serviceMust(t, s.InfDel(&lo))
					serviceMust(t, s.InfDel(&hi))
					if c != 0 {
						actual.Quo(eval(&x), eval(&y))
						lo, e = s.InfLowerDiv(&x, &y)
						serviceMust(t, e)
						hi, e = s.InfUpperDiv(&x, &y)
						serviceMust(t, e)
						if eval(&lo).Cmp(actual) > 0 || eval(&hi).Cmp(actual) < 0 {
							t.Fatalf("division bounds %d %d %d %d", a, b, c, d)
						}
						serviceMust(t, s.InfDel(&lo))
						serviceMust(t, s.InfDel(&hi))
					}
					serviceMust(t, s.InfDel(&x))
					serviceMust(t, s.InfDel(&y))
				}
			}
		}
	}
	for a := int64(-2); a <= 2; a++ {
		for b := int64(-1); b <= 1; b++ {
			for n := uint32(0); n <= 5; n++ {
				x := infValue(t, s, a, 1, big.NewInt(b).String())
				base := eval(&x)
				actual := big.NewRat(1, 1)
				for i := uint32(0); i < n; i++ {
					actual.Mul(actual, base)
				}
				lo, e := s.InfLowerPower(&x, n)
				serviceMust(t, e)
				hi, e := s.InfUpperPower(&x, n)
				serviceMust(t, e)
				if eval(&lo).Cmp(actual) > 0 || eval(&hi).Cmp(actual) < 0 {
					t.Fatalf("power bounds %d %d %d", a, b, n)
				}
				for _, v := range []*OwnedInfRational{&x, &lo, &hi} {
					serviceMust(t, s.InfDel(v))
				}
			}
		}
	}
}

func TestInfRationalFullHeaderQueriesAndAliases(t *testing.T) {
	s, _ := serviceNew(t)
	a := infValue(t, s, 7, 5, "-2/3")
	b := infValue(t, s, 7, 5, "1/2")
	r := serviceValue(t, s, "7/5")
	lesser, e := s.InfLess(&a, &b)
	serviceMust(t, e)
	if !lesser {
		t.Fatal("epsilon lt")
	}
	g, e := s.InfGreater(&b, &a)
	serviceMust(t, e)
	if !g {
		t.Fatal("reversed lt")
	}
	le, e := s.InfLessEqual(&a, &b)
	serviceMust(t, e)
	ge, e := s.InfGreaterEqual(&b, &a)
	serviceMust(t, e)
	if !le || !ge {
		t.Fatal("negated reverse lt")
	}
	lt, e := s.InfLessRational(&a, &r)
	serviceMust(t, e)
	rt, e := s.InfRationalLess(&r, &b)
	serviceMust(t, e)
	if !lt || !rt {
		t.Fatal("mixed LT")
	}
	neq, e := s.InfNotEqual(&a, &b)
	serviceMust(t, e)
	if !neq {
		t.Fatal("neq")
	}
	hash, e := s.InfHash(&a)
	serviceMust(t, e)
	if hash != uint32(7)^(uint32(0xfffffffe)+1) {
		t.Fatal(hash)
	} // source numerator-only hash (-2 -> uint32(-2))
	rational, e := s.InfIsRational(&a)
	serviceMust(t, e)
	i64, e := s.InfIsInt64(&a)
	serviceMust(t, e)
	u64, e := s.InfIsUint64(&a)
	serviceMust(t, e)
	if rational || i64 || u64 {
		t.Fatal("epsilon predicates")
	}
	if got, e := s.InfUint64(&a); modularSourceDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal(e)
		}
	} else if e != nil || got != 7 {
		t.Fatal("release FIRST projection ignores nonzero epsilon", got, e)
	}
	minus, e := s.InfMinusOne()
	serviceMust(t, e)
	ok, e := s.InfIsMinusOne(&minus)
	serviceMust(t, e)
	if !ok {
		t.Fatal("minusone constant")
	}
	serviceMust(t, s.InfSetFractionSigned(&minus, 1, -2))
	infText(t, s, &minus, "-1/2")
	second, e := s.InfGetInfinitesimal(&a)
	serviceMust(t, e)
	serviceText(t, s, &second, "-2/3")
	first, e := s.InfGetFirst(&a)
	serviceMust(t, e)
	serviceText(t, s, &first, "7/5")
	serviceMust(t, s.InfSetOffset(&minus, &r, false))
	infText(t, s, &minus, "(7/5 -e*1)")
	serviceMust(t, s.InfSetInt32(&minus, 3))
	infText(t, s, &minus, "3")
	n, e := s.InfNumerator(&minus)
	serviceMust(t, e)
	d, e := s.InfDenominator(&minus)
	serviceMust(t, e)
	infText(t, s, &n, "3")
	infText(t, s, &d, "1")
	extracted, err := s.InfNumerator(&a)
	if modularSourceDebug {
		if !errors.Is(err, ErrInfRationalDomain) || extracted.ownerID != 0 {
			t.Fatal(err, extracted)
		}
	} else {
		serviceMust(t, err)
		infText(t, s, &extracted, "7")
		serviceMust(t, s.InfDel(&extracted))
	}
	serviceMust(t, s.InfAddAssign(&a, &a))
	infText(t, s, &a, "(14/5 -e*4/3)")
	serviceMust(t, s.InfSubAssign(&a, &a))
	infText(t, s, &a, "0")
	serviceMust(t, s.InfMove(&a, &b))
	infText(t, s, &a, "(7/5 +e*1/2)")
	infText(t, s, &b, "(7/5 +e*1/2)")
	unary, e := s.InfUnaryMinus(&a)
	serviceMust(t, e)
	infText(t, s, &unary, "(-7/5 -e*1/2)")
	c := serviceValue(t, s, "3/2")
	serviceMust(t, s.InfAddMulAssign(&a, &c, &a))
	infText(t, s, &a, "(7/2 +e*5/4)")
	serviceMust(t, s.InfSubMulAssign(&a, &c, &a))
	infText(t, s, &a, "(-7/4 -e*5/8)")
	for _, v := range []*OwnedInfRational{&a, &b, &minus, &n, &d, &unary} {
		serviceMust(t, s.InfDel(v))
	}
	for _, v := range []*OwnedRational{&r, &second, &first, &c} {
		serviceMust(t, s.Del(v))
	}
}

func TestInfRationalRawStateComparisonRoutes(t *testing.T) {
	s, _ := serviceNew(t)
	a := infValue(t, s, 1, 2, "-1")
	b := infValue(t, s, 1, 2, "1")
	// Source EQ is structural and does not normalize. Numeric LT equal alone does
	// not authorize epsilon comparison when first-component source EQ is false.
	serviceMust(t, s.Manager.Z.SetInt64(&b.First.Num, 2))
	serviceMust(t, s.Manager.Z.SetInt64(&b.First.Den, 4))
	for _, fn := range []func(*OwnedInfRational, *OwnedInfRational) (bool, error){s.InfLess, s.InfGreater} {
		ok, e := fn(&a, &b)
		serviceMust(t, e)
		if ok {
			t.Fatal("structural eq route")
		}
	}
	for _, fn := range []func(*OwnedInfRational, *OwnedInfRational) (bool, error){s.InfLessEqual, s.InfGreaterEqual} {
		ok, e := fn(&a, &b)
		serviceMust(t, e)
		if !ok {
			t.Fatal("negated LT route")
		}
	}
	eq, e := s.InfEqual(&a, &b)
	serviceMust(t, e)
	if eq {
		t.Fatal("raw EQ")
	}
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
}

func infAuditRoots(t *testing.T, s *RationalService, inf []*OwnedInfRational, rat []*OwnedRational) {
	t.Helper()
	roots := []*Int{}
	add := func(q *Rational) { roots = append(roots, &q.Num, &q.Den) }
	for _, q := range []*Rational{&s.ZeroField, &s.OneField, &s.MinusOneField, &s.InfZeroField.First, &s.InfZeroField.Second, &s.InfOneField.First, &s.InfOneField.Second, &s.InfMinusOneField.First, &s.InfMinusOneField.Second, &s.InfIntZeroField.First, &s.InfIntOneField.First, &s.InfIntMinusOneField.First} {
		add(q)
	}
	for i := range s.Cache {
		add(&s.Cache[i])
	}
	for _, v := range inf {
		add(&v.First)
		add(&v.Second)
	}
	for _, v := range rat {
		add(&v.Raw)
	}
	qopsAuditLiveOwnership(t, s.Manager, roots)
}
func TestInfRationalFailurePrefixesImmediateRoots(t *testing.T) {
	for op := 0; op < 15; op++ {
		success := false
		failures := 0
		for prefix := int64(0); prefix < 4096; prefix++ {
			budget := arena.NewBudget(1 << 25)
			config := serviceConfig(budget)
			// Scalar arithmetic and integer floor copies reuse pooled storage; select
			// the supported direct profile to exercise real source malloc refusals.
			if op == 9 || op == 10 || op == 12 {
				config.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
			}
			s, e := NewRationalService(config)
			serviceMust(t, e)
			huge := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3)).String()
			eps := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 175), big.NewInt(5)).String()
			qa := serviceValue(t, s, huge)
			qe := serviceValue(t, s, eps)
			qb := serviceValue(t, s, "7/5")
			qm := serviceValue(t, s, "-3/2")
			c := serviceValue(t, s, "3/2")
			a, e := s.NewInfValue()
			serviceMust(t, e)
			b, e := s.NewInfValue()
			serviceMust(t, e)
			serviceMust(t, s.InfSetPair(&a, &qa, &qe))
			serviceMust(t, s.InfSetPair(&b, &qb, &qm))
			var value OwnedInfRational
			var rational OwnedRational
			budget.FailAfter(prefix)
			switch op {
			case 0:
				value, e = s.InfAdd(&a, &b)
			case 1:
				value, e = s.InfLowerMult(&a, &b)
			case 2:
				value, e = s.InfUpperMult(&a, &b)
			case 3:
				value, e = s.InfLowerDiv(&a, &b)
			case 4:
				value, e = s.InfUpperDiv(&a, &b)
			case 5:
				value, e = s.InfLowerPower(&a, 3)
			case 6:
				value, e = s.InfUpperPower(&a, 3)
			case 7:
				e = s.InfAddMulAssign(&a, &c, &b)
			case 8:
				e = s.InfSubMulAssign(&a, &c, &b)
			case 9:
				e = s.InfMulRationalAssign(&a, &c)
			case 10:
				e = s.InfDivRationalAssign(&a, &c)
			case 11:
				value, e = s.InfClone(&a)
			case 12:
				rational, e = s.InfFloor(&a)
			case 13:
				_, e = s.InfString(&a)
			case 14:
				e = s.InfCopy(&b, &a)
			}
			// Audit BEFORE String, Del, Close, or restoring the injected budget. Only
			// actual public roots plus service-owned constants/cache can justify cells.
			infAuditRoots(t, s, []*OwnedInfRational{&a, &b, &value}, []*OwnedRational{&qa, &qe, &qb, &qm, &c, &rational})
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(op, prefix, e)
				}
				failures++
			} else {
				success = true
			}
			budget.FailAfter(-1)
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal("failed prefix retained", op, prefix, budget.Used())
			}
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("missing failure/success prefix", op, failures)
		}
	}
}

func TestInfRationalComponentAliasSourceRereads(t *testing.T) {
	s, _ := serviceNew(t)
	v := infValue(t, s, 2, 1, "3")
	// get_rational() is a const reference in C++; the Go public accessor copies.
	// Exercise the same finite component alias through the locked raw helper.
	serviceMust(t, s.lock())
	e := s.infScalar(&v, &v.First, '*')
	s.mu.Unlock()
	serviceMust(t, e)
	infText(t, s, &v, "(4 +e*12)")
	serviceMust(t, s.InfSetInt32(&v, 2))
	three := serviceValue(t, s, "3")
	two := serviceValue(t, s, "2")
	serviceMust(t, s.InfSetPair(&v, &two, &three))
	serviceMust(t, s.lock())
	e = s.infScalar(&v, &v.First, '/')
	s.mu.Unlock()
	serviceMust(t, e)
	infText(t, s, &v, "(1 +e*3)")
	serviceMust(t, s.InfDel(&v))
	serviceMust(t, s.Del(&two))
	serviceMust(t, s.Del(&three))
}

func TestInfRationalIncrementSourceRationalRoute(t *testing.T) {
	// Source rational::++/-- dispatch Add/Sub with mk_q(1), unlike mixed AddInt.
	// Compare failure decisions with that independently exercised primitive route
	// on fresh owners, so hidden cross-product copy/reservation actions matter.
	for operation := 0; operation < 4; operation++ {
		for prefix := int64(0); prefix < 35; prefix++ {
			run := func(reference bool) (bool, error) {
				budget := arena.NewBudget(1 << 25)
				s, e := NewRationalService(serviceConfig(budget))
				if e != nil {
					return false, e
				}
				defer s.Close()
				num := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3))
				den := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 180), big.NewInt(7))
				q := serviceValue(t, s, num.String()+"/"+den.String())
				v, e := s.NewInfValue()
				serviceMust(t, e)
				serviceMust(t, s.InfSetRational(&v, &q))
				var old OwnedInfRational
				budget.FailAfter(prefix)
				if reference {
					if operation >= 2 {
						old, e = s.infClone(&v)
					}
					if e == nil {
						one := NewRational()
						one.Num = NewInt(1)
						if operation%2 == 0 {
							e = s.Manager.Add(&v.First, &one, &v.First)
						} else {
							e = s.Manager.Sub(&v.First, &one, &v.First)
						}
					}
				} else {
					switch operation {
					case 0:
						e = s.InfInc(&v)
					case 1:
						e = s.InfDec(&v)
					case 2:
						old, e = s.InfPostInc(&v)
					case 3:
						old, e = s.InfPostDec(&v)
					}
				}
				if reference && e != nil && old.ownerID != 0 {
					s.infDestroy(&old)
				}
				infAuditRoots(t, s, []*OwnedInfRational{&v, &old}, []*OwnedRational{&q})
				budget.FailAfter(-1)
				return e == nil, e
			}
			want, we := run(true)
			got, ge := run(false)
			if got != want || errors.Is(ge, arena.ErrInjectedFailure) != errors.Is(we, arena.ErrInjectedFailure) {
				t.Fatalf("source rational ++/-- route operation%d prefix%d: got%v %v want%v %v", operation, prefix, got, ge, want, we)
			}
		}
	}
}

func TestInfRationalCompoundReturnCopyFailure(t *testing.T) {
	seen := false
	for prefix := int64(0); prefix < 40; prefix++ {
		run := func(withReturnCopy bool) bool {
			budget := arena.NewBudget(1 << 25)
			s, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			defer s.Close()
			text := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3)).String()
			q := serviceValue(t, s, text)
			a, e := s.NewInfValue()
			serviceMust(t, e)
			b, e := s.NewInfValue()
			serviceMust(t, e)
			serviceMust(t, s.InfSetPair(&a, &q, &q))
			var result OwnedInfRational
			budget.FailAfter(prefix)
			if withReturnCopy {
				result, e = s.InfAdd(&a, &b)
			} else {
				result, e = s.infClone(&a)
				if e == nil {
					e = s.infCompound(&result, &b, false)
				}
			}
			infAuditRoots(t, s, []*OwnedInfRational{&a, &b, &result}, []*OwnedRational{&q})
			budget.FailAfter(-1)
			return e == nil
		}
		partial := run(false)
		full := run(true)
		if partial && !full {
			seen = true
		}
	}
	if !seen {
		t.Fatal("source return-copy failure boundary disappeared")
	}
}
func TestInfRationalCoarseRootAndFusedComponentAlias(t *testing.T) {
	s, _ := serviceNew(t)
	v := infValue(t, s, 1, 2, "0")
	upper, e := s.InfUpperRoot(&v, 2)
	serviceMust(t, e)
	infText(t, s, &upper, "1/2")
	lower, e := s.InfLowerRoot(&v, 2)
	serviceMust(t, e)
	infText(t, s, &lower, "0") // deliberately source coarse, upper is not sqrt(1/2)
	serviceMust(t, s.InfSetInt32(&v, 2))
	two := serviceValue(t, s, "2")
	three := serviceValue(t, s, "3")
	serviceMust(t, s.InfSetPair(&v, &two, &three))
	k := infValue(t, s, 4, 1, "5")
	serviceMust(t, s.lock())
	e = s.infRatFused(&v.First, &v.First, &k.First, false)
	if e == nil {
		e = s.infRatFused(&v.Second, &v.First, &k.Second, false)
	}
	s.mu.Unlock()
	serviceMust(t, e)
	infText(t, s, &v, "(10 +e*53)")
	for _, x := range []*OwnedInfRational{&v, &upper, &lower, &k} {
		serviceMust(t, s.InfDel(x))
	}
	serviceMust(t, s.Del(&two))
	serviceMust(t, s.Del(&three))
}

func TestInfRationalIntegerConstructorCopyRoute(t *testing.T) {
	for prefix := int64(0); prefix < 24; prefix++ {
		run := func(reference bool) bool {
			budget := arena.NewBudget(1 << 25)
			s, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			defer s.Close()
			v, e := s.NewInfValue()
			serviceMust(t, e)
			budget.FailAfter(prefix)
			if reference {
				tmp := NewRational()
				e = s.Manager.SetInt64(&tmp, -1<<31)
				if e == nil {
					e = s.Manager.Copy(&v.First, &tmp)
				}
				s.Manager.Del(&tmp)
			} else {
				e = s.InfSetInt32(&v, -1<<31)
			}
			infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
			budget.FailAfter(-1)
			return e == nil
		}
		want := run(true)
		got := run(false)
		if got != want {
			t.Fatalf("constructor rational(int) temporary copy route prefix%d got%v want%v", prefix, got, want)
		}
	}
}
