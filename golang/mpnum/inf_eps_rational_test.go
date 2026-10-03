// Copyright (c) 2013 Microsoft Corporation. Licensed under the MIT license.
// No registered source inf_eps_rational tests exist; these are additional tests.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"reflect"
	"testing"
)

func infEpsValue(t *testing.T, s *RationalService, i, a, b string) OwnedInfEpsInfRational {
	t.Helper()
	infty := serviceValue(t, s, i)
	first := serviceValue(t, s, a)
	second := serviceValue(t, s, b)
	n, e := s.NewInfValue()
	serviceMust(t, e)
	serviceMust(t, s.InfSetPair(&n, &first, &second))
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	serviceMust(t, s.InfEpsInfSetPair(&v, &infty, &n))
	for _, q := range []*OwnedRational{&infty, &first, &second} {
		serviceMust(t, s.Del(q))
	}
	serviceMust(t, s.InfDel(&n))
	return v
}
func infEpsText(t *testing.T, s *RationalService, v *OwnedInfEpsInfRational, want string) {
	t.Helper()
	got, e := s.InfEpsInfString(v)
	if e != nil || got != want {
		t.Fatalf("got %q %v want %q", got, e, want)
	}
}
func infEpsComponents(t *testing.T, s *RationalService, v *OwnedInfEpsInfRational) [3]*big.Rat {
	t.Helper()
	var out [3]*big.Rat
	for j, fn := range []func(*OwnedInfEpsInfRational) (OwnedRational, error){s.InfEpsInfGetInfinity, s.InfEpsInfGetRational, s.InfEpsInfGetInfinitesimal} {
		q, e := fn(v)
		serviceMust(t, e)
		text, e := s.String(&q)
		serviceMust(t, e)
		serviceMust(t, s.Del(&q))
		r, ok := new(big.Rat).SetString(text)
		if !ok {
			t.Fatal(text)
		}
		out[j] = r
	}
	return out
}
func infEpsWant(t *testing.T, s *RationalService, v *OwnedInfEpsInfRational, want [3]*big.Rat) {
	t.Helper()
	got := infEpsComponents(t, s, v)
	for j := range got {
		if got[j].Cmp(want[j]) != 0 {
			t.Fatal("triple mismatch", j, got[j], want[j])
		}
	}
}

func TestInfEpsInfFiniteTripleOracles(t *testing.T) {
	s, _ := serviceNew(t)
	for i := int64(-1); i <= 1; i++ {
		for a := int64(-1); a <= 1; a++ {
			for b := int64(-1); b <= 1; b++ {
				for k := int64(-1); k <= 1; k++ {
					for c := int64(-1); c <= 1; c++ {
						for d := int64(-1); d <= 1; d++ {
							x := infEpsValue(t, s, big.NewInt(i).String(), big.NewInt(a).String(), big.NewInt(b).String())
							y := infEpsValue(t, s, big.NewInt(k).String(), big.NewInt(c).String(), big.NewInt(d).String())
							xx, yy := [3]int64{i, a, b}, [3]int64{k, c, d}
							less := false
							equal := true
							for j := range xx {
								if xx[j] != yy[j] {
									equal = false
									less = xx[j] < yy[j]
									break
								}
							}
							eq, e := s.InfEpsInfEqual(&x, &y)
							serviceMust(t, e)
							lt, e := s.InfEpsInfLess(&x, &y)
							serviceMust(t, e)
							gt, e := s.InfEpsInfGreater(&x, &y)
							serviceMust(t, e)
							le, e := s.InfEpsInfLessEqual(&x, &y)
							serviceMust(t, e)
							ge, e := s.InfEpsInfGreaterEqual(&x, &y)
							serviceMust(t, e)
							ne, e := s.InfEpsInfNotEqual(&x, &y)
							serviceMust(t, e)
							if eq != equal || lt != less || gt != (!less && !equal) || le != (less || equal) || ge != !less || ne == equal {
								t.Fatal("lexicographic operators", xx, yy)
							}
							sum, e := s.InfEpsInfAdd(&x, &y)
							serviceMust(t, e)
							diff, e := s.InfEpsInfSub(&x, &y)
							serviceMust(t, e)
							var sw, dw [3]*big.Rat
							for j := range xx {
								sw[j] = big.NewRat(xx[j]+yy[j], 1)
								dw[j] = big.NewRat(xx[j]-yy[j], 1)
							}
							infEpsWant(t, s, &sum, sw)
							infEpsWant(t, s, &diff, dw)
							for _, v := range []*OwnedInfEpsInfRational{&x, &y, &sum, &diff} {
								serviceMust(t, s.InfEpsInfDel(v))
							}
						}
					}
				}
			}
		}
	}
}

func TestInfEpsInfSourceQuirksAndHeader(t *testing.T) {
	s, _ := serviceNew(t)
	v := infEpsValue(t, s, "1", "0", "0")
	one := serviceValue(t, s, "1")
	five := serviceValue(t, s, "5")
	// Source mixed EQ compares the infinity coefficient, despite LT comparing the
	// finite numeral only for infinity zero. Keep this inconsistent definition.
	eq, e := s.InfEpsInfEqualRational(&v, &one)
	serviceMust(t, e)
	if !eq {
		t.Fatal("source mixed EQ")
	}
	lt, e := s.InfEpsInfRationalLess(&one, &v)
	serviceMust(t, e)
	if !lt {
		t.Fatal("source infinity LT")
	}
	finite := infEpsValue(t, s, "0", "5", "0")
	eq, e = s.InfEpsInfEqualRational(&finite, &five)
	serviceMust(t, e)
	if eq {
		t.Fatal("finite5 incorrectly equal rational5")
	}
	for _, tc := range []struct{ i, a, b, text string }{{"0", "3/2", "-2/3", "(3/2 -e*2/3)"}, {"1", "0", "0", "oo"}, {"-1", "0", "0", "-oo"}, {"3/2", "0", "0", "3/2*oo"}, {"-1", "5", "0", "(-oo + 5)"}, {"2", "3/2", "-2/3", "(2*oo + (3/2 -e*2/3))"}} {
		x := infEpsValue(t, s, tc.i, tc.a, tc.b)
		infEpsText(t, s, &x, tc.text)
		serviceMust(t, s.InfEpsInfDel(&x))
	}
	epsilon := infValue(t, s, 2, 1, "3")
	serviceMust(t, s.InfEpsInfAddInfAssign(&v, &epsilon))
	infEpsText(t, s, &v, "(oo + (2 +e*3))")
	serviceMust(t, s.InfEpsInfSubInfAssign(&v, &epsilon))
	infEpsText(t, s, &v, "oo")
	serviceMust(t, s.InfEpsInfAddRationalAssign(&v, &five))
	serviceMust(t, s.InfEpsInfSubRationalAssign(&v, &one))
	infEpsText(t, s, &v, "(oo + 4)")
	scalar := serviceValue(t, s, "-3/2")
	product, e := s.InfEpsInfMulRational(&v, &scalar)
	serviceMust(t, e)
	infEpsText(t, s, &product, "(-3/2*oo + -6)")
	quotient, e := s.InfEpsInfDivRational(&product, &scalar)
	serviceMust(t, e)
	infEpsText(t, s, &quotient, "(oo + 4)")
	old, e := s.InfEpsInfPostInc(&v)
	serviceMust(t, e)
	infEpsText(t, s, &old, "(oo + 4)")
	infEpsText(t, s, &v, "(oo + 5)")
	serviceMust(t, s.InfEpsInfDec(&v))
	serviceMust(t, s.InfEpsInfInc(&v))
	serviceMust(t, s.InfEpsInfDel(&old))
	old, e = s.InfEpsInfPostDec(&v)
	serviceMust(t, e)
	infEpsText(t, s, &old, "(oo + 5)")
	serviceMust(t, s.InfEpsInfSetNumeral(&v, &epsilon))
	infEpsText(t, s, &v, "(2 +e*3)")
	serviceMust(t, s.InfEpsInfSetInt32(&v, 7))
	iv, e := s.InfEpsInfInt64(&v)
	serviceMust(t, e)
	uv, e := s.InfEpsInfUint64(&v)
	serviceMust(t, e)
	if iv != 7 || uv != 7 {
		t.Fatal(iv, uv)
	}
	numeral, e := s.InfEpsInfGetNumeral(&v)
	serviceMust(t, e)
	infText(t, s, &numeral, "7")
	serviceMust(t, s.InfDel(&numeral))
	copy, e := s.InfEpsInfClone(&v)
	serviceMust(t, e)
	serviceMust(t, s.InfEpsInfReset(&v))
	serviceMust(t, s.InfEpsInfCopy(&v, &copy))
	serviceMust(t, s.InfEpsInfSwap(&v, &finite))
	infEpsText(t, s, &v, "5")
	infEpsText(t, s, &finite, "7")
	neg, e := s.InfEpsInfUnaryMinus(&product)
	serviceMust(t, e)
	abs, e := s.InfEpsInfAbs(&product)
	serviceMust(t, e)
	infEpsText(t, s, &neg, "(3/2*oo + 6)")
	infEpsText(t, s, &abs, "(3/2*oo + 6)")
	serviceMust(t, s.InfEpsInfNeg(&neg))
	infEpsText(t, s, &neg, "(-3/2*oo + -6)")
	// Source floor/ceil ignore infinity; the finite-domain assertion is commented.
	rounded := infEpsValue(t, s, "-3/2", "2", "-1")
	floor, e := s.InfEpsInfFloor(&rounded)
	serviceMust(t, e)
	ceil, e := s.InfEpsInfCeil(&rounded)
	serviceMust(t, e)
	serviceText(t, s, &floor, "1")
	serviceText(t, s, &ceil, "2")
	hashv := infEpsValue(t, s, "2", "3", "-1")
	hash, e := s.InfEpsInfHash(&hashv)
	serviceMust(t, e)
	if hash != 1 {
		t.Fatal(hash)
	}
	for _, x := range []*OwnedInfEpsInfRational{&v, &finite, &product, &quotient, &old, &copy, &neg, &abs, &rounded, &hashv} {
		serviceMust(t, s.InfEpsInfDel(x))
	}
	for _, q := range []*OwnedRational{&one, &five, &scalar, &floor, &ceil} {
		serviceMust(t, s.Del(q))
	}
	serviceMust(t, s.InfDel(&epsilon))
}

func TestInfEpsInfConstantsPredicatesAndMixedRoutes(t *testing.T) {
	s, _ := serviceNew(t)
	zero, e := s.InfEpsInfZero()
	serviceMust(t, e)
	one, e := s.InfEpsInfOne()
	serviceMust(t, e)
	minus, e := s.InfEpsInfMinusOne()
	serviceMust(t, e)
	infinity, e := s.InfEpsInfInfinity()
	serviceMust(t, e)
	infEpsText(t, s, &zero, "0")
	infEpsText(t, s, &one, "1")
	infEpsText(t, s, &minus, "-1")
	infEpsText(t, s, &infinity, "oo")
	for i, v := range []*OwnedInfEpsInfRational{&zero, &one, &minus, &infinity} {
		z, e := s.InfEpsInfIsZero(v)
		serviceMust(t, e)
		o, e := s.InfEpsInfIsOne(v)
		serviceMust(t, e)
		m, e := s.InfEpsInfIsMinusOne(v)
		serviceMust(t, e)
		finite, e := s.InfEpsInfIsFinite(v)
		serviceMust(t, e)
		rat, e := s.InfEpsInfIsRational(v)
		serviceMust(t, e)
		integer, e := s.InfEpsInfIsInt(v)
		serviceMust(t, e)
		i64, e := s.InfEpsInfIsInt64(v)
		serviceMust(t, e)
		u64, e := s.InfEpsInfIsUint64(v)
		serviceMust(t, e)
		pos, e := s.InfEpsInfIsPos(v)
		serviceMust(t, e)
		neg, e := s.InfEpsInfIsNeg(v)
		serviceMust(t, e)
		nn, e := s.InfEpsInfIsNonNeg(v)
		serviceMust(t, e)
		np, e := s.InfEpsInfIsNonPos(v)
		serviceMust(t, e)
		if z != (i == 0) || o != (i == 1) || m != (i == 2) || finite != (i != 3) || rat != finite || integer != finite || i64 != finite || u64 != (i == 0 || i == 1) || pos != (i == 1 || i == 3) || neg != (i == 2) || nn == neg || np == pos {
			t.Fatal("predicates", i)
		}
	}
	for _, tc := range []struct{ i, a, b string }{{"-1", "2", "3"}, {"0", "1", "-1"}, {"0", "1", "0"}, {"0", "1", "1"}, {"1", "-2", "-3"}} {
		v := infEpsValue(t, s, tc.i, tc.a, tc.b)
		r := serviceValue(t, s, "1")
		lt, e := s.InfEpsInfLessRational(&v, &r)
		serviceMust(t, e)
		gt, e := s.InfEpsInfGreaterRational(&v, &r)
		serviceMust(t, e)
		le, e := s.InfEpsInfLessEqualRational(&v, &r)
		serviceMust(t, e)
		ge, e := s.InfEpsInfGreaterEqualRational(&v, &r)
		serviceMust(t, e)
		rlt, e := s.InfEpsInfRationalLess(&r, &v)
		serviceMust(t, e)
		rgt, e := s.InfEpsInfRationalGreater(&r, &v)
		serviceMust(t, e)
		rle, e := s.InfEpsInfRationalLessEqual(&r, &v)
		serviceMust(t, e)
		rge, e := s.InfEpsInfRationalGreaterEqual(&r, &v)
		serviceMust(t, e)
		eq, e := s.InfEpsInfEqualRational(&v, &r)
		serviceMust(t, e)
		neq, e := s.InfEpsInfNotEqualRational(&v, &r)
		serviceMust(t, e)
		req, e := s.InfEpsInfRationalEqual(&r, &v)
		serviceMust(t, e)
		rne, e := s.InfEpsInfRationalNotEqual(&r, &v)
		serviceMust(t, e)
		if gt != rlt || lt != rgt || le == gt || ge == lt || rle == rgt || rge == rlt || eq == neq || req != eq || rne != neq {
			t.Fatal("mixed routes")
		}
		serviceMust(t, s.InfEpsInfDel(&v))
		serviceMust(t, s.Del(&r))
	}
	for _, v := range []*OwnedInfEpsInfRational{&zero, &one, &minus, &infinity} {
		serviceMust(t, s.InfEpsInfDel(v))
	}
}

func TestInfEpsInfLifetimeAndDomain(t *testing.T) {
	s, _ := serviceNew(t)
	other, _ := serviceNew(t)
	v := infEpsValue(t, s, "1", "0", "0")
	foreign := infEpsValue(t, other, "0", "2", "0")
	if e := s.InfEpsInfCopy(&v, &foreign); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	if _, e := s.InfEpsInfInt64(&v); !errors.Is(e, ErrConversion) {
		t.Fatal(e)
	}
	if _, e := s.InfEpsInfUint64(&v); !errors.Is(e, ErrConversion) {
		t.Fatal(e)
	}
	serviceMust(t, s.Close())
	if _, e := s.InfEpsInfHash(&v); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	serviceMust(t, other.InfEpsInfDel(&foreign))
}

func infEpsAudit(t *testing.T, s *RationalService, values []*OwnedInfEpsInfRational, inf []*OwnedInfRational, rat []*OwnedRational) {
	t.Helper()
	roots := []*Int{}
	add := func(q *Rational) { roots = append(roots, &q.num, &q.den) }
	for _, q := range []*Rational{&s.zero, &s.one, &s.minusOne, &s.infZero.first, &s.infZero.second, &s.infOne.first, &s.infOne.second, &s.infMinusOne.first, &s.infMinusOne.second, &s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first} {
		add(q)
	}
	for i := range s.cache {
		add(&s.cache[i])
	}
	for _, v := range values {
		add(&v.infinity)
		add(&v.numeral.first)
		add(&v.numeral.second)
	}
	for _, v := range inf {
		add(&v.first)
		add(&v.second)
	}
	for _, v := range rat {
		add(&v.raw)
	}
	qopsAuditLiveOwnership(t, s.manager, roots)
}
func TestInfEpsInfFailurePrefixesImmediateRoots(t *testing.T) {
	for op := 0; op < 18; op++ {
		success := false
		failures := 0
		for prefix := int64(0); prefix < 512; prefix++ {
			budget := arena.NewBudget(1 << 25)
			s, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3)).String()
			d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 175), big.NewInt(7)).String()
			a := infEpsValue(t, s, n+"/11", n+"/13", "-"+d+"/17")
			b := infEpsValue(t, s, "7/5", "11/3", "-13/7")
			c := serviceValue(t, s, "3/2")
			numeral := infValue(t, s, 2, 3, d+"/7")
			var out OwnedInfEpsInfRational
			var q OwnedRational
			filler := infEpsFillCellPage(t, s)
			budget.FailAfter(prefix)
			switch op {
			case 0:
				out, e = s.InfEpsInfAdd(&a, &b)
			case 1:
				out, e = s.InfEpsInfSub(&a, &b)
			case 2:
				out, e = s.InfEpsInfMulRational(&a, &c)
			case 3:
				out, e = s.InfEpsInfDivRational(&a, &c)
			case 4:
				out, e = s.InfEpsInfClone(&a)
			case 5:
				e = s.InfEpsInfCopy(&b, &a)
			case 6:
				e = s.InfEpsInfMulRationalAssign(&a, &c)
			case 7:
				e = s.InfEpsInfDivRationalAssign(&a, &c)
			case 8:
				e = s.InfEpsInfAddMulAssign(&a, &c, &b)
			case 9:
				e = s.InfEpsInfSubMulAssign(&a, &c, &b)
			case 10:
				out, e = s.InfEpsInfPostInc(&a)
			case 11:
				out, e = s.InfEpsInfPostDec(&a)
			case 12:
				q, e = s.InfEpsInfFloor(&a)
			case 13:
				_, e = s.InfEpsInfString(&a)
			case 14:
				e = s.InfEpsInfSetNumeral(&a, &numeral)
			case 15:
				_, e = s.InfEpsInfLess(&a, &b)
			case 16:
				e = s.InfEpsInfAddAssign(&a, &a)
			case 17:
				e = s.InfEpsInfSubAssign(&a, &b)
			}
			infEpsAudit(t, s, []*OwnedInfEpsInfRational{&a, &b, &out}, []*OwnedInfRational{&numeral}, infEpsRootsWithFiller([]*OwnedRational{&c, &q}, filler)) // before restore, format, Del, Close
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(op, prefix, e)
				}
				failures++
			} else {
				success = true
			}
			budget.FailAfter(-1)
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal("retained graph", op, prefix, budget.Used())
			}
			if success {
				break
			}
		}
		if !success || failures == 0 && op != 14 {
			t.Fatal("no failure/success boundary", op, failures)
		}
	}
}

func TestInfEpsInfAliasesCancellationAndAssignmentPrefix(t *testing.T) {
	s, _ := serviceNew(t)
	v := infEpsValue(t, s, "2", "3", "5")
	k := infEpsValue(t, s, "7", "11", "13")
	c := serviceValue(t, s, "3/2")
	serviceMust(t, s.InfEpsInfAddMulAssign(&v, &c, &v))
	infEpsText(t, s, &v, "(5*oo + (15/2 +e*25/2))")
	serviceMust(t, s.InfEpsInfSubMulAssign(&v, &c, &v))
	infEpsText(t, s, &v, "(-5/2*oo + (-15/4 -e*25/4))")
	serviceMust(t, s.InfEpsInfSubAssign(&v, &v))
	infEpsText(t, s, &v, "0") // no infinity saturation
	two := serviceValue(t, s, "2")
	three := serviceValue(t, s, "3")
	five := serviceValue(t, s, "5")
	n, e := s.NewInfValue()
	serviceMust(t, e)
	serviceMust(t, s.InfSetPair(&n, &three, &five))
	serviceMust(t, s.InfEpsInfSetPair(&v, &two, &n))
	serviceMust(t, s.lock())
	e = s.infEpsInfScalar(&v, &v.infinity, '*')
	s.mu.Unlock()
	serviceMust(t, e)
	infEpsText(t, s, &v, "(4*oo + (12 +e*20))")
	serviceMust(t, s.InfEpsInfSetPair(&v, &two, &n))
	serviceMust(t, s.lock())
	e = s.infEpsInfScalar(&v, &v.infinity, '/')
	s.mu.Unlock()
	serviceMust(t, e)
	infEpsText(t, s, &v, "(oo + (3 +e*5))")
	serviceMust(t, s.InfEpsInfSetPair(&v, &two, &n))
	serviceMust(t, s.lock())
	e = s.infRatFused(&v.infinity, &v.infinity, &k.infinity, false)
	if e == nil {
		e = s.infRatFused(&v.numeral.first, &v.infinity, &k.numeral.first, false)
	}
	if e == nil {
		e = s.infRatFused(&v.numeral.second, &v.infinity, &k.numeral.second, false)
	}
	s.mu.Unlock()
	serviceMust(t, e)
	infEpsText(t, s, &v, "(16*oo + (179 +e*213))")
	for _, x := range []*OwnedInfEpsInfRational{&v, &k} {
		serviceMust(t, s.InfEpsInfDel(x))
	}
	for _, q := range []*OwnedRational{&c, &two, &three, &five} {
		serviceMust(t, s.Del(q))
	}
	serviceMust(t, s.InfDel(&n))
	// operator=(Numeral) resets infinity before a fallible numeral copy. Its
	// published reset survives failure; this must not become whole-value staging.
	found := false
	for prefix := int64(0); prefix < 100; prefix++ {
		budget := arena.NewBudget(1 << 25)
		owner, e := NewRationalService(serviceConfig(budget))
		serviceMust(t, e)
		bigtext := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3)).String()
		out := infEpsValue(t, owner, "2", "3", "5")
		num := infValue(t, owner, 2, 3, bigtext+"/7")
		filler := infEpsFillCellPage(t, owner)
		budget.FailAfter(prefix)
		e = owner.InfEpsInfSetNumeral(&out, &num)
		infEpsAudit(t, owner, []*OwnedInfEpsInfRational{&out}, []*OwnedInfRational{&num}, infEpsRootsWithFiller(nil, filler))
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if out.infinity.num.val != 0 {
				t.Fatal("source reset prefix rolled back")
			}
			found = true
		}
		budget.FailAfter(-1)
		serviceMust(t, owner.Close())
		if e == nil {
			break
		}
	}
	if !found {
		t.Fatal("no copy failure prefix")
	}
}

func TestInfEpsInfDestructorReverseMemberOrder(t *testing.T) {
	s, _ := serviceNew(t)
	bigtext := new(big.Int).Lsh(big.NewInt(1), 220).String()
	v := infEpsValue(t, s, bigtext, bigtext, bigtext)
	// C++ ~inf_eps is empty: nested Numeral is destroyed before infinity;
	// ~inf_rational implicitly destroys second before first. Observe production
	// InfEpsInfDel using allocator LIFO reuse, not a mirrored deletion helper.
	infty, first, second := v.infinity.num.cell, v.numeral.first.num.cell, v.numeral.second.num.cell
	serviceMust(t, s.InfEpsInfDel(&v))
	var nums [3]Int
	for j := range nums {
		serviceMust(t, s.manager.z.SetUint64(&nums[j], 1<<40))
	}
	if numericSlot(nums[0].cell) != numericSlot(infty) || numericSlot(nums[1].cell) != numericSlot(first) || numericSlot(nums[2].cell) != numericSlot(second) {
		t.Fatal("destructor reuse order", nums[0].cell, nums[1].cell, nums[2].cell, infty, first, second)
	}
	for j := range nums {
		serviceMust(t, s.manager.z.Del(&nums[j]))
	}
}

func TestInfEpsInfStructuralEqualityAndReturnCopy(t *testing.T) {
	s, _ := serviceNew(t)
	a := infEpsValue(t, s, "1/2", "-5", "0")
	b := infEpsValue(t, s, "1/2", "5", "0")
	serviceMust(t, s.manager.z.SetInt64(&b.infinity.num, 2))
	serviceMust(t, s.manager.z.SetInt64(&b.infinity.den, 4))
	lt, e := s.InfEpsInfLess(&a, &b)
	serviceMust(t, e)
	gt, e := s.InfEpsInfGreater(&a, &b)
	serviceMust(t, e)
	eq, e := s.InfEpsInfEqual(&a, &b)
	serviceMust(t, e)
	if lt || gt || eq {
		t.Fatal("raw infinity EQ route replaced by numeric order")
	}
	serviceMust(t, s.InfEpsInfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&b))
	seen := false
	for prefix := int64(0); prefix < 60; prefix++ {
		run := func(copyReturn bool) bool {
			budget := arena.NewBudget(1 << 25)
			owner, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			defer owner.Close()
			text := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 220), big.NewInt(3)).String()
			x := infEpsValue(t, owner, text, text, text)
			zero, e := owner.InfEpsInfNewValue()
			serviceMust(t, e)
			var out OwnedInfEpsInfRational
			budget.FailAfter(prefix)
			if copyReturn {
				out, e = owner.InfEpsInfAdd(&x, &zero)
			} else {
				out, e = owner.infEpsInfClone(&x)
				if e == nil {
					e = owner.infEpsInfCompound(&out, &zero, false)
				}
			}
			infEpsAudit(t, owner, []*OwnedInfEpsInfRational{&x, &zero, &out}, nil, nil)
			budget.FailAfter(-1)
			return e == nil
		}
		if run(false) && !run(true) {
			seen = true
		}
	}
	if !seen {
		t.Fatal("missing returned lvalue copy failure boundary")
	}
}

// Force a real backing reservation rather than assuming pooled copies allocate.
// Every filler is an explicit test-owned root, independent of operation temps.
func infEpsFillCellPage(t *testing.T, s *RationalService) []OwnedRational {
	t.Helper()
	var values []OwnedRational
	for {
		stats := s.manager.z.cells.Stats()
		if stats.Live == stats.Slots && stats.Slots%2 == 0 {
			return values
		}
		v, e := s.NewValue()
		serviceMust(t, e)
		serviceMust(t, s.SetUint64(&v, 1<<40))
		values = append(values, v)
		if len(values) > 100 {
			t.Fatal("unable to occupy cell page")
		}
	}
}
func infEpsRootsWithFiller(roots []*OwnedRational, values []OwnedRational) []*OwnedRational {
	for j := range values {
		roots = append(roots, &values[j])
	}
	return roots
}

func TestInfEpsInfPointerFreeWarmPaths(t *testing.T) {
	var check func(reflect.Type)
	check = func(typ reflect.Type) {
		switch typ.Kind() {
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				check(typ.Field(i).Type)
			}
		case reflect.Array:
			check(typ.Elem())
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.String, reflect.Func, reflect.Chan, reflect.UnsafePointer:
			t.Fatal("non-scalar hot field", typ)
		}
	}
	check(reflect.TypeOf(OwnedInfEpsInfRational{}))
	s, _ := serviceNew(t)
	a := infEpsValue(t, s, "123456789012345678901234567890/7", "314159265358979323846/11", "-271828182845904523536/13")
	b := infEpsValue(t, s, "5/3", "7/5", "11/7")
	c := serviceValue(t, s, "3/2")
	run := func() {
		v, e := s.InfEpsInfAdd(&a, &b)
		if e != nil {
			panic(e)
		}
		if e = s.InfEpsInfDel(&v); e != nil {
			panic(e)
		}
		v, e = s.InfEpsInfMulRational(&a, &c)
		if e != nil {
			panic(e)
		}
		if e = s.InfEpsInfDel(&v); e != nil {
			panic(e)
		}
		_, e = s.InfEpsInfLess(&a, &b)
		if e != nil {
			panic(e)
		}
	}
	for i := 0; i < 10; i++ {
		run()
	}
	if n := testing.AllocsPerRun(50, run); n != 0 {
		t.Fatal("warm boxed allocation", n)
	}
	serviceMust(t, s.InfEpsInfDel(&a))
	serviceMust(t, s.InfEpsInfDel(&b))
	serviceMust(t, s.Del(&c))
}

func TestInfEpsInfFormattingOneFinalAllocation(t *testing.T) {
	s, _ := serviceNew(t)
	v := infEpsValue(t, s, "123456789012345678901/7", "314159265358979323846/11", "-271828182845904523536/13")
	want := "(123456789012345678901/7*oo + (314159265358979323846/11 -e*271828182845904523536/13))"
	for i := 0; i < 10; i++ {
		infEpsText(t, s, &v, want)
	}
	var got string
	var err error
	if n := testing.AllocsPerRun(50, func() { got, err = s.InfEpsInfString(&v) }); n != 1 || err != nil || got != want {
		t.Fatal("final output allocations", n, got, err)
	}
	serviceMust(t, s.InfEpsInfDel(&v))
}
func TestInfEpsInfConstructorSignedScalarShapes(t *testing.T) {
	s, budget := serviceNew(t)
	v := infEpsValue(t, s, "123456789012345678901", "314159265358979323846", "271828182845904523536")
	for _, n := range []int32{-1 << 31, -1, 0, 1, 1<<31 - 1} {
		budget.FailAfter(0)
		e := s.InfEpsInfSetInt32(&v, n)
		infEpsAudit(t, s, []*OwnedInfEpsInfRational{&v}, nil, nil)
		budget.FailAfter(-1)
		serviceMust(t, e)
		if !v.numeral.first.num.IsSmall() || v.numeral.first.num.val != n || !isIntegerRational(&v.numeral.first) || v.infinity.num.val != 0 || v.numeral.second.num.val != 0 {
			t.Fatal("source scalar constructor shape", n)
		}
	}
	serviceMust(t, s.InfEpsInfDel(&v))
}
