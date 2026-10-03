// Copyright (c) 2013 Microsoft Corporation. Licensed under the MIT license.
// Complete actual src/util/inf_eps_rational.h<inf_rational> specialization.
package mpnum

// OwnedInfEpsInfRational represents infinity*i + (first + epsilon*second).
// Infinity is a rational coefficient, not a saturating flag. Every component
// is owned; never shallow-copy a live value. Its service owns numeric storage.
type OwnedInfEpsInfRational struct {
	infinity Rational
	numeral  OwnedInfRational
	ownerID  uint64
}

func (s *RationalService) infEpsInfEmpty() OwnedInfEpsInfRational {
	return OwnedInfEpsInfRational{infinity: NewRational(), numeral: s.infEmpty(), ownerID: s.ownerID}
}
func (s *RationalService) infEpsInfCheck(v *OwnedInfEpsInfRational) error {
	if v == nil || v.ownerID == 0 || v.ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	if e := s.manager.check(&v.infinity); e != nil {
		return e
	}
	return s.infCheck(&v.numeral)
}
func (s *RationalService) infEpsInfDestroy(v *OwnedInfEpsInfRational) {
	s.infDestroy(&v.numeral)
	s.manager.Del(&v.infinity)
	*v = OwnedInfEpsInfRational{}
}
func (s *RationalService) infEpsInfCopy(dst, src *OwnedInfEpsInfRational) error {
	if e := s.manager.Copy(&dst.infinity, &src.infinity); e != nil {
		return e
	}
	return s.infCopy(&dst.numeral, &src.numeral)
}
func (s *RationalService) infEpsInfClone(src *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	v := s.infEpsInfEmpty()
	if e := s.infEpsInfCopy(&v, src); e != nil {
		s.infEpsInfDestroy(&v)
		return OwnedInfEpsInfRational{}, e
	}
	return v, nil
}
func (s *RationalService) InfEpsInfNewValue() (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	return s.infEpsInfEmpty(), nil
}
func (s *RationalService) InfEpsInfClone(src *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(src); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	return s.infEpsInfClone(src)
}
func (s *RationalService) InfEpsInfCopy(dst, src *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(dst); e != nil {
		return e
	}
	if e := s.infEpsInfCheck(src); e != nil {
		return e
	}
	return s.infEpsInfCopy(dst, src)
}
func (s *RationalService) InfEpsInfDel(v *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	s.infEpsInfDestroy(v)
	return nil
}
func (s *RationalService) InfEpsInfReset(v *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.manager.Reset(&v.infinity); e != nil {
		return e
	}
	if e := s.manager.Reset(&v.numeral.first); e != nil {
		return e
	}
	return s.manager.Reset(&v.numeral.second)
}
func (s *RationalService) InfEpsInfSwap(a, b *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return e
	}
	if e := s.manager.Swap(&a.infinity, &b.infinity); e != nil {
		return e
	}
	if e := s.manager.Swap(&a.numeral.first, &b.numeral.first); e != nil {
		return e
	}
	return s.manager.Swap(&a.numeral.second, &b.numeral.second)
}
func (s *RationalService) InfEpsInfSetNumeral(v *OwnedInfEpsInfRational, n *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.infCheck(n); e != nil {
		return e
	}
	if e := s.manager.Reset(&v.infinity); e != nil {
		return e
	}
	return s.infCopy(&v.numeral, n)
}
func (s *RationalService) InfEpsInfSetPair(v *OwnedInfEpsInfRational, i *OwnedRational, n *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(i); e != nil {
		return e
	}
	if e := s.infCheck(n); e != nil {
		return e
	}
	if e := s.manager.Copy(&v.infinity, &i.raw); e != nil {
		return e
	}
	return s.infCopy(&v.numeral, n)
}
func (s *RationalService) InfEpsInfSetInt32(v *OwnedInfEpsInfRational, n int32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.manager.Reset(&v.infinity); e != nil {
		return e
	}
	// m_r(n) invokes inf_rational(int): rational(n) initializes its first
	// component through move/elision, not a further copy constructor.
	if e := s.manager.SetInt64(&v.numeral.first, int64(n)); e != nil {
		return e
	}
	return s.manager.Reset(&v.numeral.second)
}
func (s *RationalService) infEpsInfConstant(which uint8) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	v := s.infEpsInfEmpty()
	var e error
	n := &s.infZero
	if which == 1 {
		n = &s.infOne
	} else if which == 2 {
		n = &s.infMinusOne
	} else if which == 3 {
		e = s.manager.Copy(&v.infinity, &s.one)
	}
	if e == nil {
		e = s.infCopy(&v.numeral, n)
	}
	if e != nil {
		s.infEpsInfDestroy(&v)
		return OwnedInfEpsInfRational{}, e
	}
	return v, nil
}
func (s *RationalService) InfEpsInfZero() (OwnedInfEpsInfRational, error) {
	return s.infEpsInfConstant(0)
}
func (s *RationalService) InfEpsInfOne() (OwnedInfEpsInfRational, error) {
	return s.infEpsInfConstant(1)
}
func (s *RationalService) InfEpsInfMinusOne() (OwnedInfEpsInfRational, error) {
	return s.infEpsInfConstant(2)
}
func (s *RationalService) InfEpsInfInfinity() (OwnedInfEpsInfRational, error) {
	return s.infEpsInfConstant(3)
}
func (s *RationalService) InfEpsInfGetNumeral(v *OwnedInfEpsInfRational) (OwnedInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfRational{}, e
	}
	return s.infClone(&v.numeral)
}
func (s *RationalService) InfEpsInfHash(v *OwnedInfEpsInfRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return 0, e
	}
	i, e := s.manager.Hash(&v.infinity)
	if e != nil {
		return 0, e
	}
	a, e := s.manager.Hash(&v.numeral.first)
	if e != nil {
		return 0, e
	}
	b, e := s.manager.Hash(&v.numeral.second)
	return i ^ (a ^ (b + 1)), e
}

// Offset-aware formatting keeps every intermediate byte in the existing
// accounted manager workspace. Only finalString creates host output storage.
func (s *RationalService) infEpsInfPrepareRational(v *Rational, offset uint32) (uint32, error) {
	if e := s.manager.check(v); e != nil {
		return 0, e
	}
	n, e := s.manager.z.prepareInteger(&v.num, offset)
	if e != nil {
		return 0, e
	}
	if isIntegerRational(v) {
		return n, nil
	}
	n, e = s.manager.z.natural.appendDisplay(n, "/")
	if e != nil {
		return 0, e
	}
	return s.manager.z.prepareInteger(&v.den, n)
}
func (s *RationalService) infEpsInfPrepareNumeral(v *OwnedInfRational, offset uint32) (uint32, error) {
	if v.second.num.val == 0 {
		return s.infEpsInfPrepareRational(&v.first, offset)
	}
	n, e := s.manager.z.natural.appendDisplay(offset, "(")
	if e != nil {
		return 0, e
	}
	n, e = s.infEpsInfPrepareRational(&v.first, n)
	if e != nil {
		return 0, e
	}
	sign := " +e*"
	if v.second.num.val < 0 {
		sign = " -e*"
	}
	n, e = s.manager.z.natural.appendDisplay(n, sign)
	if e != nil {
		return 0, e
	}
	abs := NewRational()
	defer s.manager.Del(&abs)
	if e = s.manager.Copy(&abs, &v.second); e != nil {
		return 0, e
	}
	if e = s.manager.Abs(&abs); e != nil {
		return 0, e
	}
	n, e = s.infEpsInfPrepareRational(&abs, n)
	// abs(...).to_string() full-expression ends BEFORE the closing parenthesis.
	s.manager.Del(&abs)
	abs = NewRational()
	if e != nil {
		return 0, e
	}
	return s.manager.z.natural.appendDisplay(n, ")")
}
func (s *RationalService) infEpsInfPrepare(v *OwnedInfEpsInfRational) (uint32, error) {
	if v.infinity.num.val == 0 {
		return s.infEpsInfPrepareNumeral(&v.numeral, 0)
	}
	one, e := s.manager.IsOne(&v.infinity)
	if e != nil {
		return 0, e
	}
	var n uint32
	if one {
		n, e = s.manager.z.natural.appendDisplay(0, "oo")
	} else {
		minus, err := s.manager.IsMinusOne(&v.infinity)
		if err != nil {
			return 0, err
		}
		if minus {
			n, e = s.manager.z.natural.appendDisplay(0, "-oo")
		} else {
			n, e = s.infEpsInfPrepareRational(&v.infinity, 0)
			if e == nil {
				n, e = s.manager.z.natural.appendDisplay(n, "*oo")
			}
		}
	}
	if e != nil {
		return 0, e
	}
	if v.numeral.first.num.val == 0 && v.numeral.second.num.val == 0 {
		return n, nil
	}
	// Source builds the infinity spelling before checking/printing the numeral.
	// Insert '(' while preserving that completed prefix in owner storage.
	end, e := checkedTextEnd(n, 1)
	if e != nil {
		return 0, e
	}
	if e = s.manager.z.natural.reserveText(end); e != nil {
		return 0, e
	}
	for j := n; j > 0; j-- {
		s.manager.z.natural.textWrite(j, s.manager.z.natural.textRead(j-1))
	}
	s.manager.z.natural.textWrite(0, '(')
	n = end
	n, e = s.manager.z.natural.appendDisplay(n, " + ")
	if e != nil {
		return 0, e
	}
	n, e = s.infEpsInfPrepareNumeral(&v.numeral, n)
	if e != nil {
		return 0, e
	}
	return s.manager.z.natural.appendDisplay(n, ")")
}
func (s *RationalService) InfEpsInfString(v *OwnedInfEpsInfRational) (string, error) {
	if e := s.lock(); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return "", e
	}
	n, e := s.infEpsInfPrepare(v)
	if e != nil {
		return "", e
	}
	return s.manager.z.natural.finalString(n)
}
func (s *RationalService) infEpsInfSign(v *OwnedInfEpsInfRational) int32 {
	if v.infinity.num.val != 0 {
		return v.infinity.num.val
	}
	return s.infSign(&v.numeral)
}
func (s *RationalService) infEpsInfEqual(a, b *OwnedInfEpsInfRational) (bool, error) {
	ok, e := s.manager.Equal(&a.infinity, &b.infinity)
	if e != nil || !ok {
		return ok, e
	}
	ok, e = s.manager.Equal(&a.numeral.first, &b.numeral.first)
	if e != nil || !ok {
		return ok, e
	}
	return s.manager.Equal(&a.numeral.second, &b.numeral.second)
}
func (s *RationalService) infEpsInfLess(a, b *OwnedInfEpsInfRational) (bool, error) {
	n, e := s.manager.Compare(&a.infinity, &b.infinity)
	if e != nil || n < 0 {
		return n < 0, e
	}
	eq, e := s.manager.Equal(&a.infinity, &b.infinity)
	if e != nil || !eq {
		return false, e
	}
	return s.infLess(&a.numeral, &b.numeral)
}
func (s *RationalService) infEpsInfLessRational(a *OwnedInfEpsInfRational, b *OwnedRational, reverse bool) (bool, error) {
	i := a.infinity.num.val
	if reverse {
		if i > 0 {
			return true, nil
		}
		if i != 0 {
			return false, nil
		}
	} else {
		if i < 0 {
			return true, nil
		}
		if i != 0 {
			return false, nil
		}
	}
	return s.infLessRational(&a.numeral, b, reverse)
}
func (s *RationalService) infEpsInfEqualRational(a *OwnedInfEpsInfRational, b *OwnedRational, reverse bool) (bool, error) {
	left, right := &a.infinity, &b.raw
	if reverse {
		left, right = right, left
	}
	ok, e := s.manager.Equal(left, right)
	return ok && a.numeral.first.num.val == 0 && a.numeral.second.num.val == 0, e
}
func (s *RationalService) infEpsInfCompound(a, b *OwnedInfEpsInfRational, sub bool) error {
	op := byte('+')
	if sub {
		op = '-'
	}
	if e := s.infRatCompound(&a.infinity, &b.infinity, op); e != nil {
		return e
	}
	return s.infCompound(&a.numeral, &b.numeral, sub)
}
func (s *RationalService) infEpsInfScalar(v *OwnedInfEpsInfRational, r *Rational, op byte) error {
	if op == '*' || op == '/' {
		if e := s.infRatCompound(&v.infinity, r, op); e != nil {
			return e
		}
	}
	return s.infScalar(&v.numeral, r, op)
}
func (s *RationalService) infEpsInfNeg(v *OwnedInfEpsInfRational) error {
	if e := s.manager.Neg(&v.infinity); e != nil {
		return e
	}
	return s.infNeg(&v.numeral)
}
func (s *RationalService) InfEpsInfNeg(v *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	return s.infEpsInfNeg(v)
}
func (s *RationalService) infEpsInfUnary(v *OwnedInfEpsInfRational, abs bool) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	out, e := s.infEpsInfClone(v)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if !abs || s.infEpsInfSign(&out) < 0 {
		if e = s.infEpsInfNeg(&out); e != nil {
			s.infEpsInfDestroy(&out)
			return OwnedInfEpsInfRational{}, e
		}
	}
	return out, nil
}
func (s *RationalService) InfEpsInfUnaryMinus(v *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	return s.infEpsInfUnary(v, false)
}
func (s *RationalService) InfEpsInfAbs(v *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	return s.infEpsInfUnary(v, true)
}
func (s *RationalService) infEpsInfFused(v *OwnedInfEpsInfRational, c *OwnedRational, k *OwnedInfEpsInfRational, sub bool) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(c); e != nil {
		return e
	}
	if e := s.infEpsInfCheck(k); e != nil {
		return e
	}
	if e := s.infRatFused(&v.infinity, &c.raw, &k.infinity, sub); e != nil {
		return e
	}
	if e := s.infRatFused(&v.numeral.first, &c.raw, &k.numeral.first, sub); e != nil {
		return e
	}
	return s.infRatFused(&v.numeral.second, &c.raw, &k.numeral.second, sub)
}
func (s *RationalService) InfEpsInfAddMulAssign(v *OwnedInfEpsInfRational, c *OwnedRational, k *OwnedInfEpsInfRational) error {
	return s.infEpsInfFused(v, c, k, false)
}
func (s *RationalService) InfEpsInfSubMulAssign(v *OwnedInfEpsInfRational, c *OwnedRational, k *OwnedInfEpsInfRational) error {
	return s.infEpsInfFused(v, c, k, true)
}
func (s *RationalService) infEpsInfRound(v *OwnedInfEpsInfRational, ceil bool) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedRational{}, e
	}
	n := &v.numeral
	out := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	var e error
	if isIntegerRational(&n.first) {
		adjust := n.second.num.val < 0
		if ceil {
			adjust = n.second.num.val > 0
		}
		if adjust {
			op := byte('-')
			if ceil {
				op = '+'
			}
			e = s.infRatExpr(&n.first, &s.one, &out.raw, op)
		} else {
			e = s.manager.Copy(&out.raw, &n.first)
		}
	} else if ceil {
		e = s.manager.Ceil(&n.first, &out.raw)
	} else {
		e = s.manager.Floor(&n.first, &out.raw)
	}
	if e != nil {
		s.manager.Del(&out.raw)
		return OwnedRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfEpsInfFloor(v *OwnedInfEpsInfRational) (OwnedRational, error) {
	return s.infEpsInfRound(v, false)
}
func (s *RationalService) InfEpsInfCeil(v *OwnedInfEpsInfRational) (OwnedRational, error) {
	return s.infEpsInfRound(v, true)
}
func (s *RationalService) InfEpsInfGetInfinity(v *OwnedInfEpsInfRational) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedRational{}, e
	}
	return s.copyOutput(&v.infinity)
}
func (s *RationalService) InfEpsInfGetRational(v *OwnedInfEpsInfRational) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedRational{}, e
	}
	return s.copyOutput(&v.numeral.first)
}
func (s *RationalService) InfEpsInfGetInfinitesimal(v *OwnedInfEpsInfRational) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedRational{}, e
	}
	return s.copyOutput(&v.numeral.second)
}
func (s *RationalService) InfEpsInfIsFinite(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return v.infinity.num.val == 0, nil
}
func (s *RationalService) InfEpsInfIsRational(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return v.infinity.num.val == 0 && v.numeral.second.num.val == 0, nil
}
func (s *RationalService) InfEpsInfIsInt(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return v.infinity.num.val == 0 && isIntegerRational(&v.numeral.first) && v.numeral.second.num.val == 0, nil
}
func (s *RationalService) InfEpsInfIsInt64(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	if v.infinity.num.val != 0 {
		return false, nil
	}
	ok, e := s.manager.IsInt64(&v.numeral.first)
	return ok && v.numeral.second.num.val == 0, e
}
func (s *RationalService) InfEpsInfIsUint64(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	if v.infinity.num.val != 0 {
		return false, nil
	}
	ok, e := s.manager.IsUint64(&v.numeral.first)
	return ok && v.numeral.second.num.val == 0, e
}
func (s *RationalService) InfEpsInfIsZero(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return v.infinity.num.val == 0 && v.numeral.first.num.val == 0 && v.numeral.second.num.val == 0, nil
}
func (s *RationalService) InfEpsInfIsOne(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	if v.infinity.num.val != 0 {
		return false, nil
	}
	ok, e := s.manager.IsOne(&v.numeral.first)
	return ok && v.numeral.second.num.val == 0, e
}
func (s *RationalService) InfEpsInfIsMinusOne(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	if v.infinity.num.val != 0 {
		return false, nil
	}
	ok, e := s.manager.IsMinusOne(&v.numeral.first)
	return ok && v.numeral.second.num.val == 0, e
}
func (s *RationalService) InfEpsInfIsPos(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return s.infEpsInfSign(v) > 0, nil
}
func (s *RationalService) InfEpsInfIsNeg(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return s.infEpsInfSign(v) < 0, nil
}
func (s *RationalService) InfEpsInfIsNonPos(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return s.infEpsInfSign(v) <= 0, nil
}
func (s *RationalService) InfEpsInfIsNonNeg(v *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return false, e
	}
	return s.infEpsInfSign(v) >= 0, nil
}
func (s *RationalService) InfEpsInfInt64(v *OwnedInfEpsInfRational) (int64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return 0, e
	}
	if v.infinity.num.val != 0 || v.numeral.second.num.val != 0 {
		return 0, ErrConversion
	}
	return s.manager.Int64(&v.numeral.first)
}
func (s *RationalService) InfEpsInfUint64(v *OwnedInfEpsInfRational) (uint64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return 0, e
	}
	if v.infinity.num.val != 0 || v.numeral.second.num.val != 0 {
		return 0, ErrConversion
	}
	return s.manager.Uint64(&v.numeral.first)
}
func (s *RationalService) InfEpsInfEqual(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqual(a, b)
	return ok, e
}
func (s *RationalService) InfEpsInfNotEqual(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqual(a, b)
	return !ok, e
}
func (s *RationalService) InfEpsInfLess(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLess(a, b)
	return ok, e
}
func (s *RationalService) InfEpsInfGreater(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLess(b, a)
	return ok, e
}
func (s *RationalService) InfEpsInfLessEqual(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLess(b, a)
	return !ok, e
}
func (s *RationalService) InfEpsInfGreaterEqual(a, b *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLess(a, b)
	return !ok, e
}
func (s *RationalService) InfEpsInfEqualRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqualRational(a, b, false)
	return ok, e
}
func (s *RationalService) InfEpsInfNotEqualRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqualRational(a, b, false)
	return !ok, e
}
func (s *RationalService) InfEpsInfLessRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, false)
	return ok, e
}
func (s *RationalService) InfEpsInfGreaterRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, true)
	return ok, e
}
func (s *RationalService) InfEpsInfLessEqualRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, true)
	return !ok, e
}
func (s *RationalService) InfEpsInfGreaterEqualRational(a *OwnedInfEpsInfRational, b *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, false)
	return !ok, e
}
func (s *RationalService) InfEpsInfRationalEqual(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqualRational(a, b, true)
	return ok, e
}
func (s *RationalService) InfEpsInfRationalNotEqual(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfEqualRational(a, b, true)
	return !ok, e
}
func (s *RationalService) InfEpsInfRationalLess(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, true)
	return ok, e
}
func (s *RationalService) InfEpsInfRationalGreater(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, false)
	return ok, e
}
func (s *RationalService) InfEpsInfRationalLessEqual(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, false)
	return !ok, e
}
func (s *RationalService) InfEpsInfRationalGreaterEqual(b *OwnedRational, a *OwnedInfEpsInfRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	ok, e := s.infEpsInfLessRational(a, b, true)
	return !ok, e
}
func (s *RationalService) InfEpsInfAddAssign(a, b *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return e
	}
	return s.infEpsInfCompound(a, b, false)
}
func (s *RationalService) InfEpsInfAdd(a, b *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	work, e := s.infEpsInfClone(a)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.infEpsInfDestroy(&work)
	if e = s.infEpsInfCompound(&work, b, false); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	return s.infEpsInfClone(&work)
}
func (s *RationalService) InfEpsInfAddInfAssign(v *OwnedInfEpsInfRational, n *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.infCheck(n); e != nil {
		return e
	}
	return s.infCompound(&v.numeral, n, false)
}
func (s *RationalService) InfEpsInfSubAssign(a, b *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return e
	}
	return s.infEpsInfCompound(a, b, true)
}
func (s *RationalService) InfEpsInfSub(a, b *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(a); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e := s.infEpsInfCheck(b); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	work, e := s.infEpsInfClone(a)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.infEpsInfDestroy(&work)
	if e = s.infEpsInfCompound(&work, b, true); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	return s.infEpsInfClone(&work)
}
func (s *RationalService) InfEpsInfSubInfAssign(v *OwnedInfEpsInfRational, n *OwnedInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.infCheck(n); e != nil {
		return e
	}
	return s.infCompound(&v.numeral, n, true)
}
func (s *RationalService) InfEpsInfAddRationalAssign(v *OwnedInfEpsInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infEpsInfScalar(v, &r.raw, '+')
}
func (s *RationalService) InfEpsInfSubRationalAssign(v *OwnedInfEpsInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infEpsInfScalar(v, &r.raw, '-')
}
func (s *RationalService) InfEpsInfMulRationalAssign(v *OwnedInfEpsInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infEpsInfScalar(v, &r.raw, '*')
}
func (s *RationalService) InfEpsInfMulRational(v *OwnedInfEpsInfRational, r *OwnedRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e := s.check(r); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	out, e := s.infEpsInfClone(v)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e = s.infEpsInfScalar(&out, &r.raw, '*'); e != nil {
		s.infEpsInfDestroy(&out)
		return OwnedInfEpsInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfEpsInfDivRationalAssign(v *OwnedInfEpsInfRational, r *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	if e := s.check(r); e != nil {
		return e
	}
	return s.infEpsInfScalar(v, &r.raw, '/')
}
func (s *RationalService) InfEpsInfDivRational(v *OwnedInfEpsInfRational, r *OwnedRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e := s.check(r); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	out, e := s.infEpsInfClone(v)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e = s.infEpsInfScalar(&out, &r.raw, '/'); e != nil {
		s.infEpsInfDestroy(&out)
		return OwnedInfEpsInfRational{}, e
	}
	return out, nil
}
func (s *RationalService) InfEpsInfInc(v *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	return s.infRatStep(&v.numeral.first, false)
}
func (s *RationalService) InfEpsInfPostInc(v *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	old, e := s.infEpsInfClone(v)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e = s.infRatStep(&v.numeral.first, false); e != nil {
		s.infEpsInfDestroy(&old)
		return OwnedInfEpsInfRational{}, e
	}
	return old, nil
}
func (s *RationalService) InfEpsInfDec(v *OwnedInfEpsInfRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return e
	}
	return s.infRatStep(&v.numeral.first, true)
}
func (s *RationalService) InfEpsInfPostDec(v *OwnedInfEpsInfRational) (OwnedInfEpsInfRational, error) {
	if e := s.lock(); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.infEpsInfCheck(v); e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	old, e := s.infEpsInfClone(v)
	if e != nil {
		return OwnedInfEpsInfRational{}, e
	}
	if e = s.infRatStep(&v.numeral.first, true); e != nil {
		s.infEpsInfDestroy(&old)
		return OwnedInfEpsInfRational{}, e
	}
	return old, nil
}
