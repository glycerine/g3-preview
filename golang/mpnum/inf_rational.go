// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/inf_rational.h/.cpp; owner-local rational manager adaptation.
package mpnum

import "errors"

var ErrInfRationalDomain = errors.New("mpnum: infinitesimal rational domain")

// OwnedInfRational owns both components. Use InfCopy/Clone/Move/Swap, never a
// shallow copy of a live value. Closing its service invalidates the value.
type OwnedInfRational struct {
	first, second Rational
	ownerID       uint64
}

func (s *RationalService) initInfRationalConstants() error {
	for _, v := range []*OwnedInfRational{&s.infZero, &s.infOne, &s.infMinusOne} {
		*v = OwnedInfRational{first: NewRational(), second: NewRational(), ownerID: s.ownerID}
	}
	if e := s.manager.Copy(&s.infZero.first, &s.zero); e != nil {
		return e
	}
	if e := s.manager.Copy(&s.infOne.first, &s.one); e != nil {
		return e
	}
	return s.manager.Copy(&s.infMinusOne.first, &s.minusOne)
}
func (s *RationalService) destroyInfRationalConstants() {
	for _, v := range []*OwnedInfRational{&s.infZero, &s.infOne, &s.infMinusOne} {
		s.infDestroy(v)
	}
}
func (s *RationalService) infDestroy(v *OwnedInfRational) {
	s.manager.Del(&v.second)
	s.manager.Del(&v.first)
	*v = OwnedInfRational{}
}
func (s *RationalService) infCheck(v *OwnedInfRational) error {
	if v == nil || v.ownerID == 0 || v.ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	if e := s.manager.check(&v.first); e != nil {
		return e
	}
	return s.manager.check(&v.second)
}
func (s *RationalService) infEmpty() OwnedInfRational {
	return OwnedInfRational{first: NewRational(), second: NewRational(), ownerID: s.ownerID}
}
func (s *RationalService) infCopy(dst, src *OwnedInfRational) error {
	if e := s.manager.Copy(&dst.first, &src.first); e != nil {
		return e
	}
	return s.manager.Copy(&dst.second, &src.second)
}
func (s *RationalService) infClone(src *OwnedInfRational) (OwnedInfRational, error) {
	v := s.infEmpty()
	if e := s.infCopy(&v, src); e != nil {
		s.infDestroy(&v)
		return OwnedInfRational{}, e
	}
	return v, nil
}
func (s *RationalService) NewInfValue() (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	return s.infEmpty(), nil
}
func (s *RationalService) InfClone(src *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(src); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infClone(src)
}
func (s *RationalService) InfCopy(dst, src *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(dst); e != nil {
		return e
	}
	if e := s.infCheck(src); e != nil {
		return e
	}
	return s.infCopy(dst, src)
}
func (s *RationalService) InfDel(v *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	s.infDestroy(v)
	return nil
}
func (s *RationalService) InfReset(v *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.manager.Reset(&v.first); e != nil {
		return e
	}
	return s.manager.Reset(&v.second)
}
func (s *RationalService) InfMove(dst, src *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(dst); e != nil {
		return e
	}
	if e := s.infCheck(src); e != nil {
		return e
	}
	if e := s.manager.Move(&dst.first, &src.first); e != nil {
		return e
	}
	return s.manager.Move(&dst.second, &src.second)
}
func (s *RationalService) InfSwap(a, b *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return e
	}
	if e := s.infCheck(b); e != nil {
		return e
	}
	if e := s.manager.Swap(&a.first, &b.first); e != nil {
		return e
	}
	return s.manager.Swap(&a.second, &b.second)
}
func (s *RationalService) InfSetFraction(v *OwnedInfRational, n int64, d uint64) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.manager.SetFraction(&v.first, n, d); e != nil {
		return e
	}
	return s.manager.Reset(&v.second)
}
func (s *RationalService) InfSetRational(v *OwnedInfRational, a *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(a); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.first, &a.raw); e != nil {
		return e
	}
	return s.manager.Reset(&v.second)
}
func (s *RationalService) InfSetPair(v *OwnedInfRational, a, b *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(a); e != nil {
		return e
	}
	if e := s.check(b); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.first, &a.raw); e != nil {
		return e
	}
	return s.manager.Copy(&v.second, &b.raw)
}
func (s *RationalService) InfSetOffset(v *OwnedInfRational, a *OwnedRational, pos bool) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(a); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.first, &a.raw); e != nil {
		return e
	}
	c := &s.minusOne
	if pos {
		c = &s.one
	}
	return s.manager.Copy(&v.second, c)
}
func (s *RationalService) infConstant(v *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	return s.infClone(v)
}
func (s *RationalService) InfZero() (OwnedInfRational, error) {
	if s == nil {
		return OwnedInfRational{}, ErrRationalServiceCopy
	}
	return s.infConstant(&s.infZero)
}
func (s *RationalService) InfOne() (OwnedInfRational, error) {
	if s == nil {
		return OwnedInfRational{}, ErrRationalServiceCopy
	}
	return s.infConstant(&s.infOne)
}
func (s *RationalService) InfMinusOne() (OwnedInfRational, error) {
	if s == nil {
		return OwnedInfRational{}, ErrRationalServiceCopy
	}
	return s.infConstant(&s.infMinusOne)
}
func (s *RationalService) InfGetRational(v *OwnedInfRational) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedRational{}, e
	}
	return s.copyOutput(&v.first)
}
func (s *RationalService) InfGetFirst(v *OwnedInfRational) (OwnedRational, error) {
	return s.InfGetRational(v)
}
func (s *RationalService) InfGetInfinitesimal(v *OwnedInfRational) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedRational{}, e
	}
	return s.copyOutput(&v.second)
}
func (s *RationalService) InfHash(v *OwnedInfRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return 0, e
	}
	a, e := s.manager.Hash(&v.first)
	if e != nil {
		return 0, e
	}
	b, e := s.manager.Hash(&v.second)
	return a ^ (b + 1), e
}
func (s *RationalService) InfString(v *OwnedInfRational) (string, error) {
	if e := s.lock(); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return "", e
	}
	if v.second.num.val == 0 {
		return s.manager.String(&v.first)
	}
	a, e := s.manager.String(&v.first)
	if e != nil {
		return "", e
	}
	abs := NewRational()
	defer s.manager.Del(&abs)
	if e = s.manager.Copy(&abs, &v.second); e != nil {
		return "", e
	}
	if e = s.manager.Abs(&abs); e != nil {
		return "", e
	}
	b, e := s.manager.String(&abs)
	if e != nil {
		return "", e
	}
	sign := " +e*"
	if v.second.num.val < 0 {
		sign = " -e*"
	}
	return "(" + a + sign + b + ")", nil
}
func (s *RationalService) infSign(v *OwnedInfRational) int32 {
	if v.first.num.val != 0 {
		return v.first.num.val
	}
	return v.second.num.val
}
func (s *RationalService) infCompare(a, b *OwnedInfRational) (int, error) {
	n, e := s.manager.Compare(&a.first, &b.first)
	if e != nil || n < 0 {
		return n, e
	}
	eq, e := s.manager.Equal(&a.first, &b.first)
	if e != nil {
		return 0, e
	}
	if eq {
		return s.manager.Compare(&a.second, &b.second)
	}
	return 1, nil
}
func (s *RationalService) InfCompare(a, b *OwnedInfRational) (int, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return 0, e
	}
	if e := s.infCheck(b); e != nil {
		return 0, e
	}
	return s.infCompare(a, b)
}
func (s *RationalService) InfCompareRational(a *OwnedInfRational, b *OwnedRational) (int, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return 0, e
	}
	if e := s.check(b); e != nil {
		return 0, e
	}
	n, e := s.manager.Compare(&a.first, &b.raw)
	if e != nil || n < 0 {
		return n, e
	}
	eq, e := s.manager.Equal(&a.first, &b.raw)
	if e != nil {
		return 0, e
	}
	if !eq {
		return 1, nil
	}
	if a.second.num.val < 0 {
		return -1, nil
	}
	if a.second.num.val > 0 {
		return 1, nil
	}
	return 0, nil
}
func (s *RationalService) InfEqual(a, b *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.infCheck(b); e != nil {
		return false, e
	}
	ok, e := s.manager.Equal(&a.first, &b.first)
	if e != nil || !ok {
		return ok, e
	}
	return s.manager.Equal(&a.second, &b.second)
}
func (s *RationalService) InfEqualRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.manager.Equal(&a.first, &b.raw)
	return ok && a.second.num.val == 0, e
}
func (s *RationalService) infCompound(a, b *OwnedInfRational, sub bool) error {
	if sub {
		if e := s.manager.Sub(&a.first, &b.first, &a.first); e != nil {
			return e
		}
		return s.manager.Sub(&a.second, &b.second, &a.second)
	}
	if e := s.manager.Add(&a.first, &b.first, &a.first); e != nil {
		return e
	}
	return s.manager.Add(&a.second, &b.second, &a.second)
}
func (s *RationalService) infScalar(v *OwnedInfRational, r *Rational, op byte) error {
	if e := s.infRatCompound(&v.first, r, op); e != nil {
		return e
	}
	if op == '+' || op == '-' {
		return nil
	}
	return s.infRatCompound(&v.second, r, op)
}
func (s *RationalService) infRatCompound(a, b *Rational, op byte) error {
	switch op {
	case '+':
		return s.manager.Add(a, b, a)
	case '-':
		return s.manager.Sub(a, b, a)
	case '*':
		return s.manager.Mul(a, b, a)
	default:
		return s.manager.Div(a, b, a)
	}
}

// Free compound expressions return an lvalue: source copies it a second time.
func (s *RationalService) infRatExpr(a, b, out *Rational, op byte) error {
	work := NewRational()
	defer s.manager.Del(&work)
	if e := s.manager.Copy(&work, a); e != nil {
		return e
	}
	if e := s.infRatCompound(&work, b, op); e != nil {
		return e
	}
	return s.manager.Copy(out, &work)
}
func (s *RationalService) infBinary(a, b *OwnedInfRational, sub bool) (OwnedInfRational, error) {
	v, e := s.infClone(a)
	if e != nil {
		return OwnedInfRational{}, e
	}
	defer s.infDestroy(&v)
	if e = s.infCompound(&v, b, sub); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infClone(&v)
}
func (s *RationalService) infScalarResult(v *OwnedInfRational, r *Rational, op byte) (OwnedInfRational, error) {
	out, e := s.infClone(v)
	if e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infScalar(&out, r, op); e != nil {
		s.infDestroy(&out)
		return OwnedInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfAbs(v *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	out, e := s.infClone(v)
	if e != nil {
		return OwnedInfRational{}, e
	}
	if s.infSign(&out) < 0 {
		if e = s.infNeg(&out); e != nil {
			s.infDestroy(&out)
			return OwnedInfRational{}, e
		}
	}
	return out, nil
}
func (s *RationalService) infNeg(v *OwnedInfRational) error {
	if e := s.manager.Neg(&v.first); e != nil {
		return e
	}
	return s.manager.Neg(&v.second)
}
func (s *RationalService) InfNeg(v *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	return s.infNeg(v)
}
func (s *RationalService) infNumDen(v *OwnedInfRational, den bool) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	if v.second.num.val != 0 {
		return OwnedInfRational{}, ErrInfRationalDomain
	}
	tmp := NewRational()
	defer s.manager.Del(&tmp)
	var e error
	if den {
		e = s.manager.DenominatorRational(&v.first, &tmp)
	} else {
		e = s.manager.NumeratorRational(&v.first, &tmp)
	}
	if e != nil {
		return OwnedInfRational{}, e
	}
	out := s.infEmpty()
	if e = s.manager.Copy(&out.first, &tmp); e != nil {
		s.infDestroy(&out)
		return OwnedInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfNumerator(v *OwnedInfRational) (OwnedInfRational, error) {
	return s.infNumDen(v, false)
}
func (s *RationalService) InfDenominator(v *OwnedInfRational) (OwnedInfRational, error) {
	return s.infNumDen(v, true)
}
func (s *RationalService) infRound(v *OwnedInfRational, ceil bool) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedRational{}, e
	}
	out := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	var e error
	if isIntegerRational(&v.first) {
		adjust := v.second.num.val < 0
		if ceil {
			adjust = v.second.num.val > 0
		}
		if adjust {
			op := byte('-')
			if ceil {
				op = '+'
			}
			e = s.infRatExpr(&v.first, &s.one, &out.raw, op)
		} else {
			e = s.manager.Copy(&out.raw, &v.first)
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
func (s *RationalService) InfFloor(v *OwnedInfRational) (OwnedRational, error) {
	return s.infRound(v, false)
}
func (s *RationalService) InfCeil(v *OwnedInfRational) (OwnedRational, error) {
	return s.infRound(v, true)
}

// rational::addmul/submul clone k then multiply before accumulating; these are
// intentionally not replaced by manager fused arithmetic.
func (s *RationalService) infRatFused(v, c, k *Rational, sub bool) error {
	op := byte('+')
	if sub {
		op = '-'
	}
	if isOneInteger(&c.num) && isIntegerRational(c) {
		return s.infRatCompound(v, k, op)
	}
	if c.num.IsSmall() && c.num.val == -1 && isIntegerRational(c) {
		if sub {
			op = '+'
		} else {
			op = '-'
		}
		return s.infRatCompound(v, k, op)
	}
	tmp := NewRational()
	defer s.manager.Del(&tmp)
	if e := s.manager.Copy(&tmp, k); e != nil {
		return e
	}
	if e := s.manager.Mul(&tmp, c, &tmp); e != nil {
		return e
	}
	return s.infRatCompound(v, &tmp, op)
}
func (s *RationalService) infFused(v *OwnedInfRational, c *OwnedRational, k *OwnedInfRational, sub bool) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(c); e != nil {
		return e
	}
	if e := s.infCheck(k); e != nil {
		return e
	}
	if e := s.infRatFused(&v.first, &c.raw, &k.first, sub); e != nil {
		return e
	}
	return s.infRatFused(&v.second, &c.raw, &k.second, sub)
}
func (s *RationalService) InfAddMulAssign(v *OwnedInfRational, c *OwnedRational, k *OwnedInfRational) error {
	return s.infFused(v, c, k, false)
}
func (s *RationalService) InfSubMulAssign(v *OwnedInfRational, c *OwnedRational, k *OwnedInfRational) error {
	return s.infFused(v, c, k, true)
}
func (s *RationalService) InfIsInt(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return isIntegerRational(&v.first) && v.second.num.val == 0, nil
}
func (s *RationalService) InfIsRational(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return v.second.num.val == 0, nil
}
func (s *RationalService) InfIsZero(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return v.first.num.val == 0 && v.second.num.val == 0, nil
}
func (s *RationalService) InfIsOne(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	ok, e := s.manager.IsOne(&v.first)
	return ok && v.second.num.val == 0, e
}
func (s *RationalService) InfIsMinusOne(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	ok, e := s.manager.IsMinusOne(&v.first)
	return ok && v.second.num.val == 0, e
}
func (s *RationalService) InfIsPos(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return s.infSign(v) > 0, nil
}
func (s *RationalService) InfIsNeg(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return s.infSign(v) < 0, nil
}
func (s *RationalService) InfIsNonPos(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return s.infSign(v) <= 0, nil
}
func (s *RationalService) InfIsNonNeg(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	return s.infSign(v) >= 0, nil
}
func (s *RationalService) InfIsInt64(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	ok, e := s.manager.IsInt64(&v.first)
	return ok && v.second.num.val == 0, e
}
func (s *RationalService) InfIsUint64(v *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return false, e
	}
	ok, e := s.manager.IsUint64(&v.first)
	return ok && v.second.num.val == 0, e
}
func (s *RationalService) InfInt64(v *OwnedInfRational) (int64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return 0, e
	}
	if v.second.num.val != 0 {
		return 0, ErrConversion
	}
	return s.manager.Int64(&v.first)
}
func (s *RationalService) InfUint64(v *OwnedInfRational) (uint64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return 0, e
	}
	if v.second.num.val != 0 {
		return 0, ErrConversion
	}
	return s.manager.Uint64(&v.first)
}
func (s *RationalService) InfAddAssign(a, b *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return e
	}
	if e := s.infCheck(b); e != nil {
		return e
	}
	return s.infCompound(a, b, false)
}
func (s *RationalService) InfAdd(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return OwnedInfRational{}, e
	}
	if e := s.infCheck(b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infBinary(a, b, false)
}
func (s *RationalService) InfSubAssign(a, b *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return e
	}
	if e := s.infCheck(b); e != nil {
		return e
	}
	return s.infCompound(a, b, true)
}
func (s *RationalService) InfSub(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return OwnedInfRational{}, e
	}
	if e := s.infCheck(b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infBinary(a, b, true)
}
func (s *RationalService) InfAddRationalAssign(v *OwnedInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infScalar(v, &r.raw, '+')
}
func (s *RationalService) InfSubRationalAssign(v *OwnedInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infScalar(v, &r.raw, '-')
}
func (s *RationalService) InfMulRationalAssign(v *OwnedInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infScalar(v, &r.raw, '*')
}
func (s *RationalService) InfMulRational(v *OwnedInfRational, r *OwnedRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	if e := s.check(r); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infScalarResult(v, &r.raw, '*')
}
func (s *RationalService) InfDivRationalAssign(v *OwnedInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infScalar(v, &r.raw, '/')
}
func (s *RationalService) InfDivRational(v *OwnedInfRational, r *OwnedRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	if e := s.check(r); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infScalarResult(v, &r.raw, '/')
}
func (s *RationalService) InfInc(v *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	return s.infRatStep(&v.first, false)
}
func (s *RationalService) InfPostInc(v *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	out, e := s.infClone(v)
	if e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infRatStep(&v.first, false); e != nil {
		s.infDestroy(&out)
		return OwnedInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfDec(v *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	return s.infRatStep(&v.first, true)
}
func (s *RationalService) InfPostDec(v *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	out, e := s.infClone(v)
	if e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infRatStep(&v.first, true); e != nil {
		s.infDestroy(&out)
		return OwnedInfRational{}, e
	}
	return out, nil
}

// Named return values are transferred to the caller; lvalue compound-expression
// return copies are explicit in infRatExpr and infBinary. No local survives error.
func (s *RationalService) infApproxCheck(a, b *OwnedInfRational) error {
	if e := s.infCheck(a); e != nil {
		return e
	}
	if b != nil {
		return s.infCheck(b)
	}
	return nil
}
func (s *RationalService) infMult(a, b *OwnedInfRational, upper bool) (OwnedInfRational, error) {
	out := s.infEmpty()
	ok := false
	defer func() {
		if !ok {
			s.infDestroy(&out)
		}
	}()
	first := NewRational()
	if e := s.infRatExpr(&a.first, &b.first, &first, '*'); e != nil {
		s.manager.Del(&first)
		return OwnedInfRational{}, e
	}
	e := s.manager.Copy(&out.first, &first)
	s.manager.Del(&first)
	if e != nil {
		return OwnedInfRational{}, e
	}
	// Both products live until the complete sum/assignment expression ends.
	left, right, sum := NewRational(), NewRational(), NewRational()
	defer s.manager.Del(&left)
	defer s.manager.Del(&right)
	defer s.manager.Del(&sum)
	if e = s.infRatExpr(&a.first, &b.second, &left, '*'); e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infRatExpr(&a.second, &b.first, &right, '*'); e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infRatExpr(&left, &right, &sum, '+'); e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.manager.Copy(&out.second, &sum); e != nil {
		return OwnedInfRational{}, e
	}
	// Source full-expression temporaries die before epsilon correction.
	s.manager.Del(&sum)
	sum = NewRational()
	s.manager.Del(&right)
	right = NewRational()
	s.manager.Del(&left)
	left = NewRational()
	ap, bp := a.second.num.val, b.second.num.val
	if upper {
		if ap > 0 && bp > 0 || ap < 0 && bp < 0 {
			e = s.infRatStep(&out.second, false)
		}
	} else {
		if ap > 0 && bp < 0 || ap < 0 && bp > 0 {
			e = s.infRatStep(&out.second, true)
		}
	}
	if e != nil {
		return OwnedInfRational{}, e
	}
	ok = true
	return out, nil
}
func (s *RationalService) InfLowerMult(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infApproxCheck(a, b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infMult(a, b, false)
}
func (s *RationalService) InfUpperMult(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infApproxCheck(a, b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infMult(a, b, true)
}
func (s *RationalService) infDiv(a, b *OwnedInfRational, upper bool) (OwnedInfRational, error) {
	if b.first.num.val == 0 {
		return OwnedInfRational{}, ErrDivideByZero
	}
	minus, plus := false, false
	if upper {
		plus = b.second.num.val > 0 && s.infSign(a) < 0
		minus = b.second.num.val < 0 && s.infSign(a) > 0
	} else {
		minus = b.second.num.val < 0 && s.infSign(a) < 0
		plus = b.second.num.val > 0 && s.infSign(a) > 0
	}
	den := &b.first
	abs, half, adjust, two := NewRational(), NewRational(), NewRational(), NewRational()
	defer s.manager.Del(&abs)
	defer s.manager.Del(&two)
	defer s.manager.Del(&half)
	defer s.manager.Del(&adjust)
	if minus || plus {
		if e := s.manager.Copy(&abs, &b.first); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.Abs(&abs); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.SetInt64(&two, 2); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.infRatExpr(&abs, &two, &half, '/'); e != nil {
			return OwnedInfRational{}, e
		}
		op := byte('+')
		if minus {
			op = '-'
		}
		if e := s.infRatExpr(&b.first, &half, &adjust, op); e != nil {
			return OwnedInfRational{}, e
		}
		den = &adjust
	}
	// r1/den builds a named temporary then result assignment copies both members.
	value, e := s.infScalarResult(a, den, '/')
	if e != nil {
		return OwnedInfRational{}, e
	}
	defer s.infDestroy(&value)
	result, e := s.infClone(&value)
	return result, e
}
func (s *RationalService) InfLowerDiv(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infApproxCheck(a, b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infDiv(a, b, false)
}
func (s *RationalService) InfUpperDiv(a, b *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infApproxCheck(a, b); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infDiv(a, b, true)
}
func (s *RationalService) infPower(v *OwnedInfRational, n uint32, upper bool) (OwnedInfRational, error) {
	if n == 1 {
		return s.infClone(v)
	}
	out := s.infEmpty()
	ok := false
	defer func() {
		if !ok {
			s.infDestroy(&out)
		}
	}()
	a, b := v.first.num.val, v.second.num.val
	even := n&1 == 0
	exact := b == 0
	if upper {
		exact = exact || a > 0 && b < 0 || a < 0 && b > 0 && even
	} else {
		exact = exact || a > 0 && b > 0 || a < 0 && b < 0 && even
	}
	if exact {
		result := NewRational()
		defer s.manager.Del(&result)
		if e := s.manager.Power(&v.first, n, &result); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.Copy(&out.first, &result); e != nil {
			return OwnedInfRational{}, e
		}
	} else if !upper && even {
		// Source default zero, even for exponent zero.
	} else if !upper && a == 0 {
		if e := s.manager.Copy(&out.first, &s.minusOne); e != nil {
			return OwnedInfRational{}, e
		}
	} else if upper && (a == 0 || n == 0) {
		if e := s.manager.Copy(&out.first, &s.one); e != nil {
			return OwnedInfRational{}, e
		}
	} else {
		two, half, expr, base, result := NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
		defer s.manager.Del(&two)
		defer s.manager.Del(&half)
		defer s.manager.Del(&expr)
		defer s.manager.Del(&base)
		defer s.manager.Del(&result)
		if e := s.manager.SetInt64(&two, 2); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.infRatExpr(&v.first, &two, &half, '/'); e != nil {
			return OwnedInfRational{}, e
		}
		op := byte('-')
		if upper {
			if a > 0 || even {
				op = '+'
			}
		} else if a < 0 {
			op = '+'
		}
		if e := s.infRatExpr(&v.first, &half, &expr, op); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.Copy(&base, &expr); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.Power(&base, n, &result); e != nil {
			return OwnedInfRational{}, e
		}
		if e := s.manager.Copy(&out.first, &result); e != nil {
			return OwnedInfRational{}, e
		}
	}
	ok = true
	return out, nil
}
func (s *RationalService) InfLowerPower(v *OwnedInfRational, n uint32) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infPower(v, n, false)
}
func (s *RationalService) InfUpperPower(v *OwnedInfRational, n uint32) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infPower(v, n, true)
}
func (s *RationalService) InfLowerRoot(v *OwnedInfRational, n uint32) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	if s.infSign(v) < 0 {
		return OwnedInfRational{}, ErrInfRationalDomain
	}
	return s.infEmpty(), nil
}
func (s *RationalService) InfUpperRoot(v *OwnedInfRational, n uint32) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	if s.infSign(v) < 0 {
		return OwnedInfRational{}, ErrInfRationalDomain
	}
	return s.infClone(v)
}

func (s *RationalService) infLess(a, b *OwnedInfRational) (bool, error) {
	n, e := s.manager.Compare(&a.first, &b.first)
	if e != nil || n < 0 {
		return n < 0, e
	}
	eq, e := s.manager.Equal(&a.first, &b.first)
	if e != nil || !eq {
		return false, e
	}
	n, e = s.manager.Compare(&a.second, &b.second)
	return n < 0, e
}
func (s *RationalService) InfLess(a, b *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.infCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infLess(a, b)
	return ok, e
}
func (s *RationalService) InfGreater(a, b *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.infCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infLess(b, a)
	return ok, e
}
func (s *RationalService) InfLessEqual(a, b *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.infCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infLess(b, a)
	return !ok, e
}
func (s *RationalService) InfGreaterEqual(a, b *OwnedInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.infCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infLess(a, b)
	return !ok, e
}

func (s *RationalService) InfSetInt32(v *OwnedInfRational, n int32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	tmp := NewRational()
	defer s.manager.Del(&tmp)
	if e := s.manager.SetInt64(&tmp, int64(n)); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.first, &tmp); e != nil {
		return e
	}
	return s.manager.Reset(&v.second)
}
func (s *RationalService) InfSetFractionSigned(v *OwnedInfRational, n, d int32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return e
	}
	if d == 0 {
		return ErrDivideByZero
	}
	nn, dd := int64(n), int64(d)
	if dd < 0 {
		nn = -nn
		dd = -dd
	}
	work := NewRational()
	defer s.manager.Del(&work)
	if e := s.manager.z.SetInt64(&work.num, nn); e != nil {
		return e
	}
	if e := s.manager.z.SetInt64(&work.den, dd); e != nil {
		return e
	}
	if e := s.manager.normalize(&work); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.first, &work); e != nil {
		return e
	}
	return s.manager.Reset(&v.second)
}
func (s *RationalService) InfUnaryMinus(v *OwnedInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	out, e := s.infClone(v)
	if e != nil {
		return OwnedInfRational{}, e
	}
	if e = s.infNeg(&out); e != nil {
		s.infDestroy(&out)
		return OwnedInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfNotEqual(a, b *OwnedInfRational) (bool, error) {
	ok, e := s.InfEqual(a, b)
	return !ok, e
}
func (s *RationalService) InfNotEqualRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	ok, e := s.InfEqualRational(a, b)
	return !ok, e
}
func (s *RationalService) infLessRational(a *OwnedInfRational, b *OwnedRational, reverse bool) (bool, error) {
	left, right := &a.first, &b.raw
	if reverse {
		left, right = right, left
	}
	n, e := s.manager.Compare(left, right)
	if e != nil || n < 0 {
		return n < 0, e
	}
	eq, e := s.manager.Equal(left, right)
	if e != nil || !eq {
		return false, e
	}
	if reverse {
		return a.second.num.val > 0, nil
	}
	return a.second.num.val < 0, nil
}
func (s *RationalService) InfLessRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infLessRational(a, b, false)
	return ok, e
}
func (s *RationalService) InfGreaterRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infLessRational(a, b, true)
	return ok, e
}
func (s *RationalService) InfLessEqualRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infLessRational(a, b, true)
	return !ok, e
}
func (s *RationalService) InfGreaterEqualRational(a *OwnedInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infLessRational(a, b, false)
	return !ok, e
}
func (s *RationalService) InfRationalLess(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfGreaterRational(b, a)
}
func (s *RationalService) InfRationalGreater(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfLessRational(b, a)
}
func (s *RationalService) InfRationalLessEqual(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfGreaterEqualRational(b, a)
}
func (s *RationalService) InfRationalGreaterEqual(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfLessEqualRational(b, a)
}
func (s *RationalService) InfRationalEqual(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfEqualRational(b, a)
}
func (s *RationalService) InfRationalNotEqual(a *OwnedRational, b *OwnedInfRational) (bool, error) {
	return s.InfNotEqualRational(b, a)
}

// rational::++/-- call rational Add/Sub with mk_q(1), not mixed AddInt.
func (s *RationalService) infRatStep(v *Rational, dec bool) error {
	one := NewRational()
	one.num = NewInt(1)
	if dec {
		return s.manager.Sub(v, &one, v)
	}
	return s.manager.Add(v, &one, v)
}
