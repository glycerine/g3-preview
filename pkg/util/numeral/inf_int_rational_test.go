// Source: src/util/inf_int_rational.h, src/util/inf_int_rational.cpp.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func infIntValue(t *testing.T, s *RationalService, text string, e int32) OwnedInfIntRational {
	t.Helper()
	r := serviceValue(t, s, text)
	v, err := s.InfIntFromRational(&r, e)
	serviceMust(t, err)
	serviceMust(t, s.Del(&r))
	return v
}
func infIntCheck(t *testing.T, s *RationalService, v *OwnedInfIntRational, text string, e int32) {
	t.Helper()
	first, err := s.InfIntFirst(v)
	serviceMust(t, err)
	serviceText(t, s, &first, text)
	serviceMust(t, s.Del(&first))
	if v.second != e {
		t.Fatal(v.second, e)
	}
}

// No source test registration or active ENSURE count exists for this unit.
func TestInfIntRationalSourceConstructorsConstantsLifetime(t *testing.T) {
	s, b := serviceNew(t)
	serviceMust(t, arena.ValidateRecordType(reflect.TypeOf(OwnedInfIntRational{})))
	v, e := s.InfIntNewValue()
	serviceMust(t, e)
	infIntCheck(t, s, &v, "0", 0)
	one, e := s.InfIntOne()
	serviceMust(t, e)
	minus, e := s.InfIntMinusOne()
	serviceMust(t, e)
	zero, e := s.InfIntZero()
	serviceMust(t, e)
	infIntCheck(t, s, &one, "1", 0)
	infIntCheck(t, s, &minus, "-1", 0)
	infIntCheck(t, s, &zero, "0", 0)
	serviceMust(t, s.InfIntSetInt(&one, 99))
	again, e := s.InfIntOne()
	serviceMust(t, e)
	infIntCheck(t, s, &again, "1", 0)
	fraction, e := s.InfIntFromFraction(1, math.MaxInt32)
	serviceMust(t, e)
	if !fraction.First.IsSmall() {
		t.Fatal("signed constructor representation")
	}
	r := serviceValue(t, s, "123456789012345678901/19")
	pos, e := s.InfIntFromSignedInf(&r, true)
	serviceMust(t, e)
	neg, e := s.InfIntFromSignedInf(&r, false)
	serviceMust(t, e)
	infIntCheck(t, s, &pos, "123456789012345678901/19", 1)
	infIntCheck(t, s, &neg, "123456789012345678901/19", -1)
	clone, err := s.InfIntClone(&pos)
	serviceMust(t, err)
	if clone.First.Num.Cell == pos.First.Num.Cell {
		t.Fatal("copy constructor shallow ownership")
	}
	serviceMust(t, s.InfIntDel(&clone))
	serviceMust(t, s.InfIntCopy(&v, &pos))
	if v.First.Num.Cell == pos.First.Num.Cell {
		t.Fatal("shallow copy")
	}
	serviceMust(t, s.InfIntSetRational(&pos, &r))
	if pos.second != 0 {
		t.Fatal("assignment did not reset epsilon")
	}
	serviceMust(t, s.InfIntSetParts(&pos, &r, 7))
	serviceMust(t, s.InfIntSwap(&pos, &v))
	if v.second != 7 || pos.second != 1 {
		t.Fatal("swap")
	}
	serviceMust(t, s.InfIntReset(&v))
	infIntCheck(t, s, &v, "0", 0)
	for _, p := range []*OwnedInfIntRational{&v, &one, &minus, &zero, &again, &fraction, &pos, &neg} {
		serviceMust(t, s.InfIntDel(p))
	}
	serviceMust(t, s.Del(&r))
	serviceMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestInfIntRationalIndependentArithmeticAndAliases(t *testing.T) {
	s, _ := serviceNew(t)
	rng := rand.New(rand.NewSource(17))
	for i := 0; i < 150; i++ {
		n, d := int64(rng.Intn(2001)-1000), int64(rng.Intn(31)+1)
		m, e := int64(rng.Intn(2001)-1000), int64(rng.Intn(31)+1)
		ep, eq := int32(rng.Intn(101)-50), int32(rng.Intn(101)-50)
		ar, br := big.NewRat(n, d), big.NewRat(m, e)
		a, b := infIntValue(t, s, ar.RatString(), ep), infIntValue(t, s, br.RatString(), eq)
		plus, err := s.InfIntAdd(&a, &b)
		serviceMust(t, err)
		infIntCheck(t, s, &plus, new(big.Rat).Add(ar, br).RatString(), ep+eq)
		sub, err := s.InfIntSub(&a, &b)
		serviceMust(t, err)
		infIntCheck(t, s, &sub, new(big.Rat).Sub(ar, br).RatString(), ep-eq)
		serviceMust(t, s.InfIntAddAssign(&a, &a))
		infIntCheck(t, s, &a, new(big.Rat).Mul(ar, big.NewRat(2, 1)).RatString(), 2*ep)
		serviceMust(t, s.InfIntSubAssign(&b, &b))
		infIntCheck(t, s, &b, "0", 0)
		scalar := serviceValue(t, s, "-3")
		product, err := s.InfIntMul(&scalar, &plus)
		serviceMust(t, err)
		infIntCheck(t, s, &product, new(big.Rat).Mul(new(big.Rat).Add(ar, br), big.NewRat(-3, 1)).RatString(), -3*(ep+eq))
		serviceMust(t, s.InfIntAddRationalAssign(&a, &scalar))
		serviceMust(t, s.InfIntSubRationalAssign(&a, &scalar))
		infIntCheck(t, s, &a, new(big.Rat).Mul(ar, big.NewRat(2, 1)).RatString(), 2*ep)
		neg, err := s.InfIntNegCopy(&plus)
		serviceMust(t, err)
		infIntCheck(t, s, &neg, new(big.Rat).Neg(new(big.Rat).Add(ar, br)).RatString(), -(ep + eq))
		ab, err := s.InfIntAbs(&neg)
		serviceMust(t, err)
		isneg := new(big.Rat).Add(ar, br).Sign() > 0 || (new(big.Rat).Add(ar, br).Sign() == 0 && ep+eq > 0)
		wantFirst := new(big.Rat).Neg(new(big.Rat).Add(ar, br))
		wantE := -(ep + eq)
		if isneg {
			wantFirst.Neg(wantFirst)
			wantE = -wantE
		}
		infIntCheck(t, s, &ab, wantFirst.RatString(), wantE)
		for _, p := range []*OwnedInfIntRational{&a, &b, &plus, &sub, &product, &neg, &ab} {
			serviceMust(t, s.InfIntDel(p))
		}
		serviceMust(t, s.Del(&scalar))
	}
}

func TestInfIntRationalOverflowSourcePrefixesAndScalarDomain(t *testing.T) {
	s, _ := serviceNew(t)
	a := infIntValue(t, s, "2", math.MaxInt32)
	b := infIntValue(t, s, "3", 1)
	if e := s.InfIntAddAssign(&a, &b); !errors.Is(e, ErrInfIntOverflow) {
		t.Fatal(e)
	}
	infIntCheck(t, s, &a, "5", math.MaxInt32)
	r := serviceValue(t, s, "2")
	serviceMust(t, s.InfIntSetParts(&a, &r, math.MinInt32))
	if err := s.InfIntNeg(&a); !errors.Is(err, ErrInfIntOverflow) {
		t.Fatal(err)
	}
	infIntCheck(t, s, &a, "-2", math.MinInt32)
	if _, err := s.InfIntString(&a); !errors.Is(err, ErrInfIntOverflow) {
		t.Fatal("INTMIN textual negation is source UB", err)
	}
	serviceMust(t, s.InfIntSetParts(&a, &r, math.MinInt32))
	if err := s.InfIntSubAssign(&a, &b); !errors.Is(err, ErrInfIntOverflow) {
		t.Fatal(err)
	}
	infIntCheck(t, s, &a, "-1", math.MinInt32)
	serviceMust(t, s.InfIntSetParts(&a, &r, math.MaxInt32))
	if err := s.InfIntMulAssign(&a, &r); !errors.Is(err, ErrInfIntOverflow) {
		t.Fatal(err)
	}
	infIntCheck(t, s, &a, "4", math.MaxInt32)
	for _, text := range []string{"1/2", "2147483648", "-2147483649"} {
		serviceMust(t, s.SetString(&r, text))
		before := a.First.Num.Val
		err := s.InfIntMulAssign(&a, &r)
		var source z3exception.DefaultException
		if !errors.Is(err, ErrInfIntScalar) || !errors.As(err, &source) || source.Message() != "multiplication with large rational is not possible" {
			t.Fatal(text, err)
		}
		if a.First.Num.Val != before || a.second != math.MaxInt32 {
			t.Fatal("domain validation followed mutation")
		}
	}
	for _, text := range []string{"2147483647", "-2147483648"} {
		serviceMust(t, s.SetString(&r, text))
		zero, _ := s.InfIntFromInt(1)
		serviceMust(t, s.InfIntMulAssign(&zero, &r))
		infIntCheck(t, s, &zero, text, 0)
		serviceMust(t, s.InfIntDel(&zero))
	}
	serviceMust(t, s.InfIntDel(&a))
	serviceMust(t, s.InfIntDel(&b))
	serviceMust(t, s.Del(&r))
}

func TestInfIntRationalComparisonsSignsAndRounding(t *testing.T) {
	s, _ := serviceNew(t)
	for _, text := range []string{"-3/2", "-1", "0", "1", "3/2"} {
		for _, ep := range []int32{-2, 0, 2} {
			a := infIntValue(t, s, text, ep)
			base, _ := new(big.Rat).SetString(text)
			r := serviceValue(t, s, text)
			for _, other := range []string{"-2", "-1", "0", "1", "2"} {
				for _, eq := range []int32{-1, 0, 1} {
					b := infIntValue(t, s, other, eq)
					br, _ := new(big.Rat).SetString(other)
					cmp := base.Cmp(br)
					if cmp == 0 {
						if ep < eq {
							cmp = -1
						} else if ep > eq {
							cmp = 1
						}
					}
					results := []struct {
						got  bool
						want bool
						err  error
					}{}
					v, e := s.InfIntEqual(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp == 0, e})
					v, e = s.InfIntNotEqual(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp != 0, e})
					v, e = s.InfIntLess(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp < 0, e})
					v, e = s.InfIntGreater(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp > 0, e})
					v, e = s.InfIntLessEqual(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp <= 0, e})
					v, e = s.InfIntGreaterEqual(&a, &b)
					results = append(results, struct {
						got  bool
						want bool
						err  error
					}{v, cmp >= 0, e})
					for _, x := range results {
						if x.err != nil || x.got != x.want {
							t.Fatal(text, ep, other, eq, x)
						}
					}
					serviceMust(t, s.InfIntDel(&b))
				}
			}
			got, e := s.InfIntLessRational(&a, &r)
			if e != nil || got != (ep < 0) {
				t.Fatal(got, e)
			}
			got, e = s.InfIntRationalLess(&r, &a)
			if e != nil || got != (ep > 0) {
				t.Fatal(got, e)
			}
			got, e = s.InfIntEqualRational(&a, &r)
			if e != nil || got != (ep == 0) {
				t.Fatal(got, e)
			}
			floor, e := s.InfIntFloor(&a)
			serviceMust(t, e)
			ceil, e := s.InfIntCeil(&a)
			serviceMust(t, e)
			q := new(big.Int).Quo(base.Num(), base.Denom())
			f, c := new(big.Int).Set(q), new(big.Int).Set(q)
			if base.IsInt() {
				if ep < 0 {
					f.Sub(f, big.NewInt(1))
				}
				if ep > 0 {
					c.Add(c, big.NewInt(1))
				}
			} else {
				if base.Sign() < 0 {
					f.Sub(f, big.NewInt(1))
				}
				if base.Sign() > 0 {
					c.Add(c, big.NewInt(1))
				}
			}
			serviceText(t, s, &floor, f.String())
			serviceText(t, s, &ceil, c.String())
			serviceMust(t, s.Del(&floor))
			serviceMust(t, s.Del(&ceil))
			neg := base.Sign() < 0 || (base.Sign() == 0 && ep < 0)
			pos := base.Sign() > 0 || (base.Sign() == 0 && ep > 0)
			got, e = s.InfIntIsNeg(&a)
			if e != nil || got != neg {
				t.Fatal(got, e)
			}
			got, e = s.InfIntIsPos(&a)
			if e != nil || got != pos {
				t.Fatal(got, e)
			}
			got, e = s.InfIntIsNonNeg(&a)
			if e != nil || got == neg {
				t.Fatal(got, e)
			}
			got, e = s.InfIntIsNonPos(&a)
			if e != nil || got == pos {
				t.Fatal(got, e)
			}
			got, e = s.InfIntIsInt(&a)
			if e != nil || got != (base.IsInt() && ep == 0) {
				t.Fatal(got, e)
			}
			got, e = s.InfIntIsRational(&a)
			if e != nil || got != (ep == 0) {
				t.Fatal(got, e)
			}
			serviceMust(t, s.InfIntDel(&a))
			serviceMust(t, s.Del(&r))
		}
	}
}

func TestInfIntRationalAccessorsHashDisplayPostOperations(t *testing.T) {
	s, _ := serviceNew(t)
	a := infIntValue(t, s, "-3/2", 2)
	hash, e := s.InfIntHash(&a)
	serviceMust(t, e)
	base, e := s.Manager.Hash(&a.First)
	serviceMust(t, e)
	if hash != base^(uint32(2)+1) {
		t.Fatal(hash, base)
	}
	text, e := s.InfIntString(&a)
	if e != nil || text != "(-3/2 +e*2)" {
		t.Fatal(text, e)
	}
	epsilon, e := s.InfIntInfinitesimal(&a)
	serviceMust(t, e)
	serviceText(t, s, &epsilon, "2")
	serviceMust(t, s.Del(&epsilon))
	old, e := s.InfIntPostInc(&a)
	serviceMust(t, e)
	infIntCheck(t, s, &old, "-3/2", 2)
	infIntCheck(t, s, &a, "-1/2", 2)
	serviceMust(t, s.InfIntDel(&old))
	old, e = s.InfIntPostDec(&a)
	serviceMust(t, e)
	infIntCheck(t, s, &old, "-1/2", 2)
	infIntCheck(t, s, &a, "-3/2", 2)
	serviceMust(t, s.InfIntDel(&old))
	extracted, err := s.InfIntNumerator(&a)
	if modularSourceDebug {
		if !errors.Is(err, ErrInfIntNotRational) || extracted.ownerID != 0 {
			t.Fatal(err, extracted)
		}
	} else {
		serviceMust(t, err)
		infIntCheck(t, s, &extracted, "-3", 0)
		serviceMust(t, s.InfIntDel(&extracted))
	}
	serviceMust(t, s.InfIntReset(&a))
	r := serviceValue(t, s, "-3/2")
	serviceMust(t, s.InfIntSetRational(&a, &r))
	num, e := s.InfIntNumerator(&a)
	serviceMust(t, e)
	den, e := s.InfIntDenominator(&a)
	serviceMust(t, e)
	infIntCheck(t, s, &num, "-3", 0)
	infIntCheck(t, s, &den, "2", 0)
	serviceMust(t, s.InfIntSetInt(&a, -1))
	ok, e := s.InfIntIsMinusOne(&a)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	serviceMust(t, s.InfIntInc(&a))
	ok, e = s.InfIntIsZero(&a)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	serviceMust(t, s.InfIntInc(&a))
	ok, e = s.InfIntIsOne(&a)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	serviceMust(t, s.InfIntDec(&a))
	for _, p := range []*OwnedInfIntRational{&a, &num, &den} {
		serviceMust(t, s.InfIntDel(p))
	}
	serviceMust(t, s.Del(&r))
}

func TestInfIntRationalConversionsAndMixedComparisons(t *testing.T) {
	s, _ := serviceNew(t)
	for _, text := range []string{"-9223372036854775808", "9223372036854775807", "18446744073709551615", "1/2"} {
		a := infIntValue(t, s, text, 0)
		r := serviceValue(t, s, text)
		q, _ := new(big.Rat).SetString(text)
		wantI := q.IsInt() && q.Num().IsInt64()
		wantU := q.IsInt() && q.Sign() >= 0 && q.Num().BitLen() <= 64
		ok, e := s.InfIntIsInt64(&a)
		if e != nil || ok != wantI {
			t.Fatal(text, ok, e)
		}
		ok, e = s.InfIntIsUint64(&a)
		if e != nil || ok != wantU {
			t.Fatal(text, ok, e)
		}
		iv, e := s.InfIntInt64(&a)
		if wantI {
			if e != nil || iv != q.Num().Int64() {
				t.Fatal(iv, e)
			}
		} else if modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal(e)
			}
		} else {
			want := int64(projectionLowMagnitude(q.Num()))
			if q.Sign() < 0 {
				want = -want
			}
			if e != nil || iv != want {
				t.Fatal("release FIRST signed lowword", text, iv, want, e)
			}
		}
		uv, e := s.InfIntUint64(&a)
		if wantU {
			if e != nil || uv != q.Num().Uint64() {
				t.Fatal(uv, e)
			}
		} else if modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal(e)
			}
		} else {
			want := projectionLowMagnitude(q.Num())
			if a.First.Num.IsSmall() {
				want = uint64(int64(a.First.Num.Val))
			}
			if e != nil || uv != want {
				t.Fatal("release FIRST unsigned scalar/magnitude", text, uv, want, e)
			}
		}
		serviceAuditRationalRoots(t, s, &a.First, &r.Raw)
		a.second = 1
		if got, e := s.InfIntInt64(&a); modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal(e)
			}
		} else {
			want := int64(projectionLowMagnitude(q.Num()))
			if q.Sign() < 0 {
				want = -want
			}
			if e != nil || got != want {
				t.Fatal("release ignores epsilon signed", text, got, want, e)
			}
		}
		if got, e := s.InfIntUint64(&a); modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) {
				t.Fatal(e)
			}
		} else {
			want := projectionLowMagnitude(q.Num())
			if a.First.Num.IsSmall() {
				want = uint64(int64(a.First.Num.Val))
			}
			if e != nil || got != want {
				t.Fatal("release ignores epsilon unsigned", text, got, want, e)
			}
		}
		b := infIntValue(t, s, text, 1)
		ok, e = s.InfIntEqual(&a, &b)
		if e != nil || !ok {
			t.Fatal(ok, e)
		}
		got := []bool{}
		want := []bool{false, true, false, true, false, true, false, true, true, false, true, false}
		appendResult := func(v bool, e error) { serviceMust(t, e); got = append(got, v) }
		appendResult(s.InfIntEqualRational(&a, &r))
		appendResult(s.InfIntNotEqualRational(&a, &r))
		appendResult(s.InfIntLessRational(&a, &r))
		appendResult(s.InfIntGreaterRational(&a, &r))
		appendResult(s.InfIntLessEqualRational(&a, &r))
		appendResult(s.InfIntGreaterEqualRational(&a, &r))
		appendResult(s.InfIntRationalEqual(&r, &a))
		appendResult(s.InfIntRationalNotEqual(&r, &a))
		appendResult(s.InfIntRationalLess(&r, &a))
		appendResult(s.InfIntRationalGreater(&r, &a))
		appendResult(s.InfIntRationalLessEqual(&r, &a))
		appendResult(s.InfIntRationalGreaterEqual(&r, &a))
		if !reflect.DeepEqual(got, want) {
			t.Fatal(got, want)
		}
		serviceMust(t, s.InfIntDel(&a))
		serviceMust(t, s.InfIntDel(&b))
		serviceMust(t, s.Del(&r))
	}
}

func TestInfIntRationalFaultPrefixesTemporaryCleanup(t *testing.T) {
	failures, successes := 0, 0
	for n := int64(0); n < 80; n++ {
		s, b := serviceNew(t)
		a := infIntValue(t, s, "123456789012345678901234567890/19", 7)
		r := infIntValue(t, s, "998877665544332211009988776655/23", -2)
		b.FailAfter(n)
		out, e := s.InfIntAdd(&a, &r)
		serviceAuditRationalRoots(t, s, &a.First, &r.First, &out.First) // immediate before format/Del
		b.FailAfter(-1)
		serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
		if e != nil {
			failures++
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(n, e)
			}
		} else {
			successes++
			infIntCheck(t, s, &out, new(big.Rat).Add(infIntBigRat("123456789012345678901234567890/19"), infIntBigRat("998877665544332211009988776655/23")).RatString(), 5)
			serviceMust(t, s.InfIntDel(&out))
		}
		infIntCheck(t, s, &a, "123456789012345678901234567890/19", 7)
		infIntCheck(t, s, &r, "998877665544332211009988776655/23", -2)
		serviceMust(t, s.InfIntDel(&a))
		serviceMust(t, s.InfIntDel(&r))
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(n, b.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	s, b := servicePublicationDirectOwner(t)
	src := infIntValue(t, s, "123456789012345678901234567890/19", 99)
	dst := infIntValue(t, s, "3", 7)
	var fillers []OwnedRational
	for s.Manager.Z.Cells.Free != 0 {
		x, _ := s.NewValue()
		serviceMust(t, s.SetUint64(&x, math.MaxUint64))
		fillers = append(fillers, x)
	}
	b.FailAfter(0)
	e := s.InfIntCopy(&dst, &src)
	roots := []*Rational{&src.First, &dst.First}
	for i := range fillers {
		roots = append(roots, &fillers[i].Raw)
	}
	serviceAuditRationalRoots(t, s, roots...)
	if !numericPhysicalReservationRefusal(e) || dst.First.Num.Cell != s.Manager.Z.FailedNullRef || dst.First.Den != NewInt(1) || dst.second != 7 || !errors.Is(s.Manager.Z.check(&dst.First.Num), arena.ErrInvalidSlot) {
		t.Fatal("first component Copy refusal before epsilon publication", dst, e)
	}
	b.FailAfter(-1)
	serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
	infIntCheck(t, s, &src, "123456789012345678901234567890/19", 99)
	for i := range fillers {
		serviceMust(t, s.Del(&fillers[i]))
	}
	serviceMust(t, s.InfIntDel(&src))
	serviceMust(t, s.InfIntDel(&dst))
}
func infIntBigRat(text string) *big.Rat {
	v, ok := new(big.Rat).SetString(text)
	if !ok {
		panic("invalid test rational")
	}
	return v
}

func TestInfIntRationalAliasReReadWarmReuseAndGuards(t *testing.T) {
	s, b := serviceNew(t)
	a := infIntValue(t, s, "2", 3)
	// Exercise genuine source scalar alias internally; public getter deep-copies.
	s.mu.Lock()
	e := s.mulInfInt(&a, &a.First)
	s.mu.Unlock()
	serviceMust(t, e)
	infIntCheck(t, s, &a, "4", 12)
	r := serviceValue(t, s, "123456789012345678901/19")
	serviceMust(t, s.InfIntSetParts(&a, &r, 2))
	copy, _ := s.InfIntNewValue()
	serviceMust(t, s.InfIntCopy(&copy, &a))
	serviceMust(t, s.InfIntCopy(&copy, &a)) // prepare actual transient copy peak
	b.FailAfter(0)
	var err error
	allocs := testing.AllocsPerRun(100, func() {
		err = s.InfIntCopy(&copy, &a)
		if err == nil {
			err = s.InfIntSwap(&a, &copy)
		}
	})
	b.FailAfter(-1)
	if err != nil || allocs != 0 {
		t.Fatal(err, allocs)
	}
	other, _ := serviceNew(t)
	foreign, _ := other.InfIntNewValue()
	used := b.Used()
	if e = s.InfIntCopy(&a, &foreign); !errors.Is(e, ErrRationalServiceOwner) || b.Used() != used {
		t.Fatal(e)
	}
	if e = s.InfIntMulAssign(&foreign, &r); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, s.InfIntDel(&copy))
	if e = s.InfIntDel(&copy); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, s.InfIntDel(&a))
	serviceMust(t, s.Del(&r))
	serviceMust(t, other.InfIntDel(&foreign))
	serviceMust(t, s.Close())
	if _, e = s.InfIntNewValue(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestInfIntRationalIncrementSourceRationalRoute(t *testing.T) {
	for _, dec := range []bool{false, true} {
		for n := int64(0); n < 32; n++ {
			actual, ab := serviceNew(t)
			reference, rb := serviceNew(t)
			a := infIntValue(t, actual, "123456789012345678901234567890/99887766554433221101", 4)
			r := infIntValue(t, reference, "123456789012345678901234567890/99887766554433221101", 4)
			if ab.Used() != rb.Used() {
				t.Fatal("fixture owners differ")
			}
			ab.FailAfter(n)
			rb.FailAfter(n)
			var got, want error
			one := NewRational()
			one.Num = NewInt(1)
			if dec {
				got = actual.InfIntDec(&a)
				want = reference.Manager.Sub(&r.First, &one, &r.First)
			} else {
				got = actual.InfIntInc(&a)
				want = reference.Manager.Add(&r.First, &one, &r.First)
			}
			ab.FailAfter(-1)
			rb.FailAfter(-1)
			if (got == nil) != (want == nil) || !errors.Is(got, want) || ab.Used() != rb.Used() || actual.Manager.Z.Cells.Stats().Live != reference.Manager.Z.Cells.Stats().Live {
				t.Fatalf("source rational route differs dec=%v prefix=%d errors=%v/%v used=%d/%d", dec, n, got, want, ab.Used(), rb.Used())
			}
			x, e := actual.Manager.String(&a.First)
			serviceMust(t, e)
			y, e := reference.Manager.String(&r.First)
			serviceMust(t, e)
			if x != y || a.second != 4 {
				t.Fatal(x, y, a.second)
			}
			serviceMust(t, actual.InfIntDel(&a))
			serviceMust(t, reference.InfIntDel(&r))
			serviceMust(t, actual.Close())
			serviceMust(t, reference.Close())
		}
	}
}

func TestInfIntRationalPostFailureAndGetterIndependence(t *testing.T) {
	failures, successes := 0, 0
	for _, dec := range []bool{false, true} {
		for n := int64(0); n < 40; n++ {
			s, b := serviceNew(t)
			a := infIntValue(t, s, "123456789012345678901234567890/99887766554433221101", -4)
			b.FailAfter(n)
			var old OwnedInfIntRational
			var e error
			if dec {
				old, e = s.InfIntPostDec(&a)
			} else {
				old, e = s.InfIntPostInc(&a)
			}
			serviceAuditRationalRoots(t, s, &a.First, &old.First) // actual remaining roots before format/Del
			b.FailAfter(-1)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) || old.ownerID != 0 {
					t.Fatal(n, e, old.ownerID)
				}
			} else {
				successes++
				serviceMust(t, s.InfIntDel(&old))
			}
			want := infIntBigRat("123456789012345678901234567890/99887766554433221101")
			if e == nil {
				if dec {
					want.Sub(want, big.NewRat(1, 1))
				} else {
					want.Add(want, big.NewRat(1, 1))
				}
			}
			if e != nil {
				pn := new(big.Int).Set(want.Num())
				if dec {
					pn.Sub(pn, want.Denom())
				} else {
					pn.Add(pn, want.Denom())
				}
				addPublicationExpectRaw(t, s.Manager, &a.First, want.Num().String(), want.Denom().String(), pn, want.Denom())
				if a.second != -4 {
					t.Fatal("integer infinitesimal changed before first-component action completed", n, a.second)
				}
			} else {
				infIntCheck(t, s, &a, want.RatString(), -4)
			}
			serviceMust(t, s.InfIntDel(&a))
			serviceMust(t, s.Close())
			if b.Used() != 0 {
				t.Fatal(n, b.Used())
			}
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	s, _ := serviceNew(t)
	a := infIntValue(t, s, "123456789012345678901/19", 3)
	r, e := s.InfIntRational(&a)
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&r, 99))
	infIntCheck(t, s, &a, "123456789012345678901/19", 3)
	serviceMust(t, s.Del(&r))
	// Unary overflow disposes its already-copied first, never leaking the named
	// result's numerical storage or publishing a returned owning token.
	a.second = math.MinInt32
	a.First.Num.Val = -1
	cells := s.Manager.Z.Cells.Stats().Live
	neg, e := s.InfIntNegCopy(&a)
	if !errors.Is(e, ErrInfIntOverflow) || neg.ownerID != 0 || s.Manager.Z.Cells.Stats().Live != cells {
		t.Fatal(e)
	}
	ab, e := s.InfIntAbs(&a)
	if !errors.Is(e, ErrInfIntOverflow) || ab.ownerID != 0 || s.Manager.Z.Cells.Stats().Live != cells {
		t.Fatal(e)
	}
	serviceMust(t, s.InfIntDel(&a))
}
