// Copyright (c) 2006 Microsoft Corporation
// Concrete translation of src/util/inf_int_rational.h/.cpp.
package mpnum

import (
	"errors"
	"math"
	"strconv"
)

var ErrInfIntOverflow = errors.New("mpnum: infinitesimal integer signed overflow")
var ErrInfIntScalar = errors.New("mpnum: multiplication with large rational is not possible")
var ErrInfIntNotRational = errors.New("mpnum: infinitesimal coefficient must be zero")

// OwnedInfIntRational is the source rational/int32 pair plus scalar provenance.
// Its zero value is invalid. Never shallow-copy ownership; use InfIntCopy.
// All operations borrow the service; closing it invalidates its values.
type OwnedInfIntRational struct {
	first   Rational
	second  int32
	ownerID uint64
}

func (s *RationalService) initInfIntRationalConstants() error {
	values := [3]*OwnedInfIntRational{&s.infIntZero, &s.infIntOne, &s.infIntMinusOne}
	bases := [3]*Rational{&s.zero, &s.one, &s.minusOne}
	for i, v := range values {
		*v = OwnedInfIntRational{first: NewRational(), ownerID: s.ownerID}
		if e := s.manager.Copy(&v.first, bases[i]); e != nil {
			return e
		}
	}
	return nil
}
func (s *RationalService) destroyInfIntRationalConstants() {
	for _, v := range [3]*OwnedInfIntRational{&s.infIntZero, &s.infIntOne, &s.infIntMinusOne} {
		if v.ownerID == s.ownerID && v.ownerID != 0 {
			s.manager.Del(&v.first)
			*v = OwnedInfIntRational{}
		}
	}
}
func (s *RationalService) checkInfInt(v *OwnedInfIntRational) error {
	if v == nil || v.ownerID == 0 || v.ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	return s.manager.check(&v.first)
}
func (s *RationalService) newInfInt() OwnedInfIntRational {
	return OwnedInfIntRational{first: NewRational(), ownerID: s.ownerID}
}
func (s *RationalService) copyInfInt(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	out := s.newInfInt()
	if e := s.manager.Copy(&out.first, &v.first); e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	out.second = v.second
	return out, nil
}

func (s *RationalService) lockInfInt(v *OwnedInfIntRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	if e := s.checkInfInt(v); e != nil {
		s.mu.Unlock()
		return e
	}
	return nil
}
func (s *RationalService) lockInfInt2(a, b *OwnedInfIntRational) error {
	if e := s.lockInfInt(a); e != nil {
		return e
	}
	if e := s.checkInfInt(b); e != nil {
		s.mu.Unlock()
		return e
	}
	return nil
}
func (s *RationalService) lockInfIntRat(a *OwnedInfIntRational, b *OwnedRational) error {
	if e := s.lockInfInt(a); e != nil {
		return e
	}
	if e := s.check(b); e != nil {
		s.mu.Unlock()
		return e
	}
	return nil
}
func (s *RationalService) InfIntNewValue() (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.newInfInt(), nil
}
func (s *RationalService) InfIntFromInt(n int32) (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	out := s.newInfInt()
	if e := s.manager.SetInt64(&out.first, int64(n)); e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfIntFromFraction(n, d int32) (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	if d == 0 {
		return OwnedInfIntRational{}, ErrDivideByZero
	}
	out := s.newInfInt()
	nn, dd := int64(n), int64(d)
	if dd < 0 {
		nn = -nn
		dd = -dd
	}
	e := s.manager.z.SetInt64(&out.first.num, nn)
	if e == nil {
		e = s.manager.z.SetInt64(&out.first.den, dd)
	}
	if e == nil {
		e = s.manager.normalize(&out.first)
	}
	if e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfIntFromRational(r *OwnedRational, epsilon int32) (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.check(r); e != nil {
		return OwnedInfIntRational{}, e
	}
	out := s.newInfInt()
	if e := s.manager.Copy(&out.first, &r.raw); e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	out.second = epsilon
	return out, nil
}
func (s *RationalService) InfIntFromSignedInf(r *OwnedRational, positive bool) (OwnedInfIntRational, error) {
	epsilon := int32(-1)
	if positive {
		epsilon = 1
	}
	return s.InfIntFromRational(r, epsilon)
}
func (s *RationalService) InfIntZero() (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyInfInt(&s.infIntZero)
}
func (s *RationalService) InfIntOne() (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyInfInt(&s.infIntOne)
}
func (s *RationalService) InfIntMinusOne() (OwnedInfIntRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyInfInt(&s.infIntMinusOne)
}
func (s *RationalService) InfIntDel(v *OwnedInfIntRational) error {
	if e := s.lockInfInt(v); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.Del(&v.first); e != nil {
		return e
	}
	*v = OwnedInfIntRational{}
	return nil
}
func (s *RationalService) InfIntCopy(dst, src *OwnedInfIntRational) error {
	if e := s.lockInfInt2(dst, src); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.Copy(&dst.first, &src.first); e != nil {
		return e
	}
	dst.second = src.second
	return nil
}
func (s *RationalService) InfIntSetParts(dst *OwnedInfIntRational, src *OwnedRational, epsilon int32) error {
	if e := s.lockInfIntRat(dst, src); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.Copy(&dst.first, &src.raw); e != nil {
		return e
	}
	dst.second = epsilon
	return nil
}
func (s *RationalService) InfIntSetRational(dst *OwnedInfIntRational, src *OwnedRational) error {
	return s.InfIntSetParts(dst, src, 0)
}
func (s *RationalService) InfIntSetInt(dst *OwnedInfIntRational, n int32) error {
	if e := s.lockInfInt(dst); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.SetInt64(&dst.first, int64(n)); e != nil {
		return e
	}
	dst.second = 0
	return nil
}
func (s *RationalService) InfIntReset(v *OwnedInfIntRational) error {
	if e := s.lockInfInt(v); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.Reset(&v.first); e != nil {
		return e
	}
	v.second = 0
	return nil
}
func (s *RationalService) InfIntSwap(a, b *OwnedInfIntRational) error {
	if e := s.lockInfInt2(a, b); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.manager.Swap(&a.first, &b.first); e != nil {
		return e
	}
	a.second, b.second = b.second, a.second
	return nil
}
func infIntPublishSecond(v *OwnedInfIntRational, n int64) error {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return ErrInfIntOverflow
	}
	v.second = int32(n)
	return nil
}
func (s *RationalService) addSubInfInt(a, b *OwnedInfIntRational, sub bool) error {
	if sub {
		if e := s.manager.Sub(&a.first, &b.first, &a.first); e != nil {
			return e
		}
		return infIntPublishSecond(a, int64(a.second)-int64(b.second))
	}
	if e := s.manager.Add(&a.first, &b.first, &a.first); e != nil {
		return e
	}
	return infIntPublishSecond(a, int64(a.second)+int64(b.second))
}
func (s *RationalService) InfIntAddAssign(a, b *OwnedInfIntRational) error {
	if e := s.lockInfInt2(a, b); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.addSubInfInt(a, b, false)
}
func (s *RationalService) InfIntSubAssign(a, b *OwnedInfIntRational) error {
	if e := s.lockInfInt2(a, b); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.addSubInfInt(a, b, true)
}
func (s *RationalService) mulInfInt(a *OwnedInfIntRational, r *Rational) error {
	valid, e := s.manager.IsInt32(r)
	if e != nil {
		return e
	}
	if !valid {
		return ErrInfIntScalar
	}
	if e = s.manager.Mul(&a.first, r, &a.first); e != nil {
		return e
	}
	// Source gets coefficient AFTER rational multiplication. Keep this ordering,
	// including scalar alias with first in private/internal use.
	coefficient, e := s.manager.Int32(r)
	if e != nil {
		return e
	}
	return infIntPublishSecond(a, int64(a.second)*int64(coefficient))
}
func (s *RationalService) InfIntMulAssign(a *OwnedInfIntRational, r *OwnedRational) error {
	if e := s.lockInfIntRat(a, r); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.mulInfInt(a, &r.raw)
}
func (s *RationalService) InfIntAddRationalAssign(a *OwnedInfIntRational, r *OwnedRational) error {
	if e := s.lockInfIntRat(a, r); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.manager.Add(&a.first, &r.raw, &a.first)
}
func (s *RationalService) InfIntSubRationalAssign(a *OwnedInfIntRational, r *OwnedRational) error {
	if e := s.lockInfIntRat(a, r); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.manager.Sub(&a.first, &r.raw, &a.first)
}

// Source ++/-- resolves the rational facade Add/Sub with mk_q(1), not the
// manager's distinct mixed-int Inc/Dec route (which has different fault actions).
func (s *RationalService) stepInfInt(v *Rational, dec bool) error {
	one := NewRational()
	one.num = NewInt(1)
	if dec {
		return s.manager.Sub(v, &one, v)
	}
	return s.manager.Add(v, &one, v)
}
func (s *RationalService) InfIntInc(v *OwnedInfIntRational) error {
	if e := s.lockInfInt(v); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.stepInfInt(&v.first, false)
}
func (s *RationalService) InfIntDec(v *OwnedInfIntRational) error {
	if e := s.lockInfInt(v); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.stepInfInt(&v.first, true)
}
func (s *RationalService) postInfInt(v *OwnedInfIntRational, dec bool) (OwnedInfIntRational, error) {
	old, e := s.copyInfInt(v)
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	e = s.stepInfInt(&v.first, dec)
	if e != nil {
		s.manager.Del(&old.first)
		return OwnedInfIntRational{}, e
	}
	return old, nil
}
func (s *RationalService) InfIntPostInc(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.postInfInt(v, false)
}
func (s *RationalService) InfIntPostDec(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.postInfInt(v, true)
}
func (s *RationalService) negInfInt(v *OwnedInfIntRational) error {
	if e := s.manager.Neg(&v.first); e != nil {
		return e
	}
	return infIntPublishSecond(v, -int64(v.second))
}
func (s *RationalService) InfIntNeg(v *OwnedInfIntRational) error {
	if e := s.lockInfInt(v); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.negInfInt(v)
}
func (s *RationalService) resultInfInt(a, b *OwnedInfIntRational, sub bool) (OwnedInfIntRational, error) {
	// Source free binary returns a compound-operator lvalue: copy operand,
	// mutate temporary, copy return, destroy temporary (no elision of that copy).
	tmp, e := s.copyInfInt(a)
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.manager.Del(&tmp.first)
	if e = s.addSubInfInt(&tmp, b, sub); e != nil {
		return OwnedInfIntRational{}, e
	}
	return s.copyInfInt(&tmp)
}
func (s *RationalService) InfIntAdd(a, b *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.resultInfInt(a, b, false)
}
func (s *RationalService) InfIntSub(a, b *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.resultInfInt(a, b, true)
}
func (s *RationalService) InfIntMul(r *OwnedRational, a *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfIntRat(a, r); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	tmp, e := s.copyInfInt(a)
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.manager.Del(&tmp.first)
	if e = s.mulInfInt(&tmp, &r.raw); e != nil {
		return OwnedInfIntRational{}, e
	}
	return s.copyInfInt(&tmp)
}
func (s *RationalService) InfIntNegCopy(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	out, e := s.copyInfInt(v)
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	if e = s.negInfInt(&out); e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfIntAbs(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	out, e := s.copyInfInt(v)
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	if out.first.num.val < 0 || (out.first.num.val == 0 && out.second < 0) {
		if e = s.negInfInt(&out); e != nil {
			s.manager.Del(&out.first)
			return OwnedInfIntRational{}, e
		}
	}
	return out, nil
}
func (s *RationalService) InfIntHash(v *OwnedInfIntRational) (uint32, error) {
	if e := s.lockInfInt(v); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	h, e := s.manager.Hash(&v.first)
	return h ^ (uint32(v.second) + 1), e
}
func (s *RationalService) InfIntString(v *OwnedInfIntRational) (string, error) {
	if e := s.lockInfInt(v); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	first, e := s.manager.String(&v.first)
	if e != nil {
		return "", e
	}
	if v.second == 0 {
		return first, nil
	}
	if v.second == math.MinInt32 {
		return "", ErrInfIntOverflow
	}
	if v.second < 0 {
		return "(" + first + " -e*" + strconv.FormatInt(-int64(v.second), 10) + ")", nil
	}
	return "(" + first + " +e*" + strconv.FormatInt(int64(v.second), 10) + ")", nil
}
func (s *RationalService) InfIntFirst(v *OwnedInfIntRational) (OwnedRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyOutput(&v.first)
}
func (s *RationalService) InfIntRational(v *OwnedInfIntRational) (OwnedRational, error) {
	return s.InfIntFirst(v)
}
func (s *RationalService) InfIntInfinitesimal(v *OwnedInfIntRational) (OwnedRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	out := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	if e := s.manager.SetInt64(&out.raw, int64(v.second)); e != nil {
		s.manager.Del(&out.raw)
		return OwnedRational{}, e
	}
	return out, nil
}
func (s *RationalService) partInfInt(v *OwnedInfIntRational, den bool) (OwnedInfIntRational, error) {
	if v.second != 0 {
		return OwnedInfIntRational{}, ErrInfIntNotRational
	}
	part := NewRational()
	defer s.manager.Del(&part)
	var e error
	if den {
		e = s.manager.DenominatorRational(&v.first, &part)
	} else {
		e = s.manager.NumeratorRational(&v.first, &part)
	}
	if e != nil {
		return OwnedInfIntRational{}, e
	}
	out := s.newInfInt()
	if e = s.manager.Copy(&out.first, &part); e != nil {
		s.manager.Del(&out.first)
		return OwnedInfIntRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfIntNumerator(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.partInfInt(v, false)
}
func (s *RationalService) InfIntDenominator(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.partInfInt(v, true)
}
func (s *RationalService) roundInfInt(v *OwnedInfIntRational, ceil bool) (OwnedRational, error) {
	out := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	var e error
	if isIntegerRational(&v.first) {
		if (ceil && v.second <= 0) || (!ceil && v.second >= 0) {
			e = s.manager.Copy(&out.raw, &v.first)
		} else {
			// Source rational free +/- also uses clone→compound→return copy.
			tmp := NewRational()
			defer s.manager.Del(&tmp)
			e = s.manager.Copy(&tmp, &v.first)
			if e == nil {
				if ceil {
					e = s.manager.Add(&tmp, &s.one, &tmp)
				} else {
					e = s.manager.Sub(&tmp, &s.one, &tmp)
				}
			}
			if e == nil {
				e = s.manager.Copy(&out.raw, &tmp)
			}
		}
	} else if ceil {
		e = s.manager.Ceil(&v.first, &out.raw)
	} else {
		e = s.manager.Floor(&v.first, &out.raw)
	}
	if e != nil {
		s.manager.Del(&out.raw)
		return OwnedRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfIntFloor(v *OwnedInfIntRational) (OwnedRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.roundInfInt(v, false)
}
func (s *RationalService) InfIntCeil(v *OwnedInfIntRational) (OwnedRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.roundInfInt(v, true)
}

func (s *RationalService) lessInfInt(a *Rational, ae int32, b *Rational, be int32) (bool, error) {
	cmp, e := s.manager.Compare(a, b)
	if e != nil {
		return false, e
	}
	if cmp < 0 {
		return true, nil
	}
	equal, e := s.manager.Equal(a, b)
	if e != nil || !equal {
		return false, e
	}
	return ae < be, nil
}
func (s *RationalService) equalInfInt(a *Rational, ae int32, b *Rational, be int32) (bool, error) {
	equal, e := s.manager.Equal(a, b)
	if e != nil || !equal {
		return false, e
	}
	return ae == be, nil
}
func (s *RationalService) InfIntInt64(v *OwnedInfIntRational) (int64, error) {
	if e := s.lockInfInt(v); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	ok, e := s.manager.IsInt64(&v.first)
	if e != nil {
		return 0, e
	}
	if !ok || v.second != 0 {
		return 0, ErrConversion
	}
	return s.manager.Int64(&v.first)
}
func (s *RationalService) InfIntUint64(v *OwnedInfIntRational) (uint64, error) {
	if e := s.lockInfInt(v); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	ok, e := s.manager.IsUint64(&v.first)
	if e != nil {
		return 0, e
	}
	if !ok || v.second != 0 {
		return 0, ErrConversion
	}
	return s.manager.Uint64(&v.first)
}
func (s *RationalService) InfIntIsZero(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.val == 0 && v.second == 0, nil
}
func (s *RationalService) InfIntIsOne(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return isOneInteger(&v.first.num) && isIntegerRational(&v.first) && v.second == 0, nil
}
func (s *RationalService) InfIntIsMinusOne(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.IsSmall() && v.first.num.val == -1 && isIntegerRational(&v.first) && v.second == 0, nil
}
func (s *RationalService) InfIntIsNeg(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.val < 0 || (v.first.num.val == 0 && v.second < 0), nil
}
func (s *RationalService) InfIntIsPos(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.val > 0 || (v.first.num.val == 0 && v.second > 0), nil
}
func (s *RationalService) InfIntIsNonNeg(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.val > 0 || (v.first.num.val == 0 && v.second >= 0), nil
}
func (s *RationalService) InfIntIsNonPos(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.first.num.val < 0 || (v.first.num.val == 0 && v.second <= 0), nil
}
func (s *RationalService) InfIntIsRational(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return v.second == 0, nil
}
func (s *RationalService) InfIntIsInt(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	ok, e := s.manager.IsInt(&v.first)
	return ok && v.second == 0, e
}
func (s *RationalService) InfIntIsInt64(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	ok, e := s.manager.IsInt64(&v.first)
	return ok && v.second == 0, e
}
func (s *RationalService) InfIntIsUint64(v *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt(v); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	ok, e := s.manager.IsUint64(&v.first)
	return ok && v.second == 0, e
}
func (s *RationalService) InfIntEqual(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.equalInfInt(&a.first, a.second, &b.first, b.second)
}
func (s *RationalService) InfIntNotEqual(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.equalInfInt(&a.first, a.second, &b.first, b.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntLess(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&a.first, a.second, &b.first, b.second)
}
func (s *RationalService) InfIntGreater(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&b.first, b.second, &a.first, a.second)
}
func (s *RationalService) InfIntLessEqual(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&b.first, b.second, &a.first, a.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntGreaterEqual(a, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfInt2(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&a.first, a.second, &b.first, b.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntEqualRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.equalInfInt(&a.first, a.second, &b.raw, 0)
}
func (s *RationalService) InfIntNotEqualRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.equalInfInt(&a.first, a.second, &b.raw, 0)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntLessRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&a.first, a.second, &b.raw, 0)
}
func (s *RationalService) InfIntGreaterRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&b.raw, 0, &a.first, a.second)
}
func (s *RationalService) InfIntLessEqualRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&b.raw, 0, &a.first, a.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntGreaterEqualRational(a *OwnedInfIntRational, b *OwnedRational) (bool, error) {
	if e := s.lockInfIntRat(a, b); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&a.first, a.second, &b.raw, 0)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntRationalEqual(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.equalInfInt(&a.raw, 0, &b.first, b.second)
}
func (s *RationalService) InfIntRationalNotEqual(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.equalInfInt(&a.raw, 0, &b.first, b.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntRationalLess(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&a.raw, 0, &b.first, b.second)
}
func (s *RationalService) InfIntRationalGreater(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.lessInfInt(&b.first, b.second, &a.raw, 0)
}
func (s *RationalService) InfIntRationalLessEqual(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&b.first, b.second, &a.raw, 0)
	if e != nil {
		return false, e
	}
	return !v, nil
}
func (s *RationalService) InfIntRationalGreaterEqual(a *OwnedRational, b *OwnedInfIntRational) (bool, error) {
	if e := s.lockInfIntRat(b, a); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	v, e := s.lessInfInt(&a.raw, 0, &b.first, b.second)
	if e != nil {
		return false, e
	}
	return !v, nil
}

func (s *RationalService) InfIntClone(v *OwnedInfIntRational) (OwnedInfIntRational, error) {
	if e := s.lockInfInt(v); e != nil {
		return OwnedInfIntRational{}, e
	}
	defer s.mu.Unlock()
	return s.copyInfInt(v)
}
