// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/rational.h owner-local operator facade.
package mpnum

import "errors"

var ErrRationalFacadeDomain = errors.New("mpnum: rational facade source assertion domain")

type rationalFacadeBinaryOp uint8

const (
	facadeInvalid rationalFacadeBinaryOp = iota
	rationalFacadeAdd
	rationalFacadeSub
	rationalFacadeMul
	rationalFacadeDiv
	rationalFacadeRem
	rationalFacadeIDiv
	rationalFacadeMachineDiv
	rationalFacadeMod
	rationalFacadeGCD
)

// These helpers require the caller to hold s.mu. All input provenance is
// checked before numeric actions; no foreign raw numeral is admitted.
func (s *RationalService) facadeAssignLocked(a, b *OwnedRational, op rationalFacadeBinaryOp) error {
	if e := s.check(a); e != nil {
		return e
	}
	if e := s.check(b); e != nil {
		return e
	}
	return s.facadeRawBinaryLocked(&a.raw, &b.raw, &a.raw, op)
}
func (s *RationalService) facadeRawBinaryLocked(a, b, out *Rational, op rationalFacadeBinaryOp) error {
	switch op {
	case rationalFacadeAdd:
		return s.manager.Add(a, b, out)
	case rationalFacadeSub:
		return s.manager.Sub(a, b, out)
	case rationalFacadeMul:
		return s.manager.Mul(a, b, out)
	case rationalFacadeDiv:
		return s.manager.Div(a, b, out)
	case rationalFacadeRem:
		return s.manager.Rem(a, b, out)
	case rationalFacadeIDiv:
		return s.manager.IDiv(a, b, out)
	case rationalFacadeMachineDiv:
		return s.manager.MachineIDiv(a, b, out)
	case rationalFacadeMod:
		return s.manager.Mod(a, b, out)
	case rationalFacadeGCD:
		return s.manager.GCD(a, b, out)
	default:
		return ErrRationalFacadeDomain
	}
}
func (s *RationalService) facadeValueLocked(a, b *OwnedRational, op rationalFacadeBinaryOp) (_lexOut OwnedRational, _lexErr error) {
	if e := s.beginLexical(); e != nil {
		return OwnedRational{}, e
	}
	defer s.endLexicalRational(&_lexOut, &_lexErr)
	if e := s.check(a); e != nil {
		return OwnedRational{}, e
	}
	if e := s.check(b); e != nil {
		return OwnedRational{}, e
	}
	work := NewRational()
	workComplete := !(op >= rationalFacadeAdd && op <= rationalFacadeDiv)
	defer s.endLexicalLocal(&work, &workComplete)
	if op >= rationalFacadeAdd && op <= rationalFacadeDiv {
		// return rational(a) op= b: copy before compound, then copy the returned
		// lvalue into the value result. The second copy is intentionally observable.
		if e := s.manager.Copy(&work, &a.raw); e != nil {
			return OwnedRational{}, e
		}
		workComplete = true
		if e := s.facadeRawBinaryLocked(&work, &b.raw, &work, op); e != nil {
			return OwnedRational{}, e
		}
		return s.copyOutput(&work)
	}
	if e := s.facadeRawBinaryLocked(&a.raw, &b.raw, &work, op); e != nil {
		return OwnedRational{}, e
	}
	return s.facadeTakeLocked(&work)
}

// Named local return transfers ownership; it does not introduce a second copy.
func (s *RationalService) facadeTakeLocked(work *Rational) (OwnedRational, error) {
	if e := s.lexicalReady(); e != nil {
		return OwnedRational{}, e
	}
	out := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	if e := s.manager.Move(&out.raw, work); e != nil {
		return OwnedRational{}, e
	}
	return out, nil
}
func (s *RationalService) facadeValue(a, b *OwnedRational, op rationalFacadeBinaryOp) (OwnedRational, error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	return s.facadeValueLocked(a, b, op)
}
func (s *RationalService) facadeAssign(a, b *OwnedRational, op rationalFacadeBinaryOp) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.facadeAssignLocked(a, b, op)
}
func (s *RationalService) Add(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeAdd)
}
func (s *RationalService) Sub(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeSub)
}
func (s *RationalService) Mul(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeMul)
}
func (s *RationalService) Div(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeDiv)
}
func (s *RationalService) Rem(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeRem)
}
func (s *RationalService) IDiv(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeIDiv)
}
func (s *RationalService) MachineDiv(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeMachineDiv)
}
func (s *RationalService) Mod(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeMod)
}
func (s *RationalService) GCD(a, b *OwnedRational) (OwnedRational, error) {
	return s.facadeValue(a, b, rationalFacadeGCD)
}
func (s *RationalService) AddAssign(a, b *OwnedRational) error {
	return s.facadeAssign(a, b, rationalFacadeAdd)
}
func (s *RationalService) SubAssign(a, b *OwnedRational) error {
	return s.facadeAssign(a, b, rationalFacadeSub)
}
func (s *RationalService) MulAssign(a, b *OwnedRational) error {
	return s.facadeAssign(a, b, rationalFacadeMul)
}
func (s *RationalService) DivAssign(a, b *OwnedRational) error {
	return s.facadeAssign(a, b, rationalFacadeDiv)
}
func (s *RationalService) RemAssign(a, b *OwnedRational) error {
	return s.facadeAssign(a, b, rationalFacadeRem)
}
func (s *RationalService) IDivInto(a, b, out *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	for _, v := range []*OwnedRational{a, b, out} {
		if e := s.check(v); e != nil {
			return e
		}
	}
	return s.facadeRawBinaryLocked(&a.raw, &b.raw, &out.raw, rationalFacadeIDiv)
}
func (s *RationalService) ModInto(a, b, out *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	for _, v := range []*OwnedRational{a, b, out} {
		if e := s.check(v); e != nil {
			return e
		}
	}
	return s.facadeRawBinaryLocked(&a.raw, &b.raw, &out.raw, rationalFacadeMod)
}

// Scalar overloads construct an actual rational(int), rather than taking the
// raw manager's mixed-integer arithmetic shortcut.
func (s *RationalService) facadeScalar(a *OwnedRational, n int32, op rationalFacadeBinaryOp, left bool) (_lexOut OwnedRational, _lexErr error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return OwnedRational{}, e
	}
	defer s.endLexicalRational(&_lexOut, &_lexErr)
	if e := s.check(a); e != nil {
		return OwnedRational{}, e
	}
	v := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	vComplete := false
	defer s.endLexicalLocal(&v.raw, &vComplete)
	if e := s.manager.SetInt64(&v.raw, int64(n)); e != nil {
		return OwnedRational{}, e
	}
	vComplete = true
	if left {
		return s.facadeValueLocked(&v, a, op)
	}
	return s.facadeValueLocked(a, &v, op)
}
func (s *RationalService) AddInt(a *OwnedRational, n int32) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeAdd, false)
}
func (s *RationalService) IntAdd(n int32, a *OwnedRational) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeAdd, true)
}
func (s *RationalService) SubInt(a *OwnedRational, n int32) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeSub, false)
}
func (s *RationalService) IntSub(n int32, a *OwnedRational) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeSub, true)
}
func (s *RationalService) MulInt(a *OwnedRational, n int32) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeMul, false)
}
func (s *RationalService) IntMul(n int32, a *OwnedRational) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeMul, true)
}
func (s *RationalService) DivInt(a *OwnedRational, n int32) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeDiv, false)
}
func (s *RationalService) IntDiv(n int32, a *OwnedRational) (OwnedRational, error) {
	return s.facadeScalar(a, n, rationalFacadeDiv, true)
}
func (s *RationalService) AddAssignInt(a *OwnedRational, n int32) (_lexErr error) {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	if e := s.check(a); e != nil {
		return e
	}
	v := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	vComplete := false
	defer s.endLexicalLocal(&v.raw, &vComplete)
	if e := s.manager.SetInt64(&v.raw, int64(n)); e != nil {
		return e
	}
	vComplete = true
	return s.facadeAssignLocked(a, &v, rationalFacadeAdd)
}
func (s *RationalService) AssignInt(a *OwnedRational, n int32) (_lexErr error) {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	if e := s.check(a); e != nil {
		return e
	}
	v := NewRational()
	vComplete := false
	defer s.endLexicalLocal(&v, &vComplete)
	if e := s.manager.SetInt64(&v, int64(n)); e != nil {
		return e
	}
	vComplete = true
	return s.manager.Copy(&a.raw, &v)
}
func (s *RationalService) facadePrefixLocked(a *OwnedRational, dec bool) error {
	if e := s.check(a); e != nil {
		return e
	}
	// Source mk_q(1) creates a local inline rational, not the cached constant.
	one := NewRational()
	one.num = NewInt(1)
	if dec {
		return s.manager.Sub(&a.raw, &one, &a.raw)
	}
	return s.manager.Add(&a.raw, &one, &a.raw)
}
func (s *RationalService) facadePrefix(a *OwnedRational, dec bool) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.facadePrefixLocked(a, dec)
}
func (s *RationalService) PrefixInc(a *OwnedRational) error { return s.facadePrefix(a, false) }
func (s *RationalService) PrefixDec(a *OwnedRational) error { return s.facadePrefix(a, true) }
func (s *RationalService) facadePostfix(a *OwnedRational, dec bool) (_lexOut OwnedRational, _lexErr error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return OwnedRational{}, e
	}
	defer s.endLexicalRational(&_lexOut, &_lexErr)
	if e := s.check(a); e != nil {
		return OwnedRational{}, e
	}
	old := NewRational()
	oldComplete := false
	defer s.endLexicalLocal(&old, &oldComplete)
	if e := s.manager.Copy(&old, &a.raw); e != nil {
		return OwnedRational{}, e
	}
	oldComplete = true
	if e := s.facadePrefixLocked(a, dec); e != nil {
		return OwnedRational{}, e
	}
	return s.facadeTakeLocked(&old)
}
func (s *RationalService) PostInc(a *OwnedRational) (OwnedRational, error) {
	return s.facadePostfix(a, false)
}
func (s *RationalService) PostDec(a *OwnedRational) (OwnedRational, error) {
	return s.facadePostfix(a, true)
}

type rationalFacadeUnaryOp uint8

const (
	facadeNeg rationalFacadeUnaryOp = iota
	facadeAbs
	facadeFloor
	facadeCeil
	facadeNumerator
	facadeDenominator
	facadePower
)

func (s *RationalService) facadeUnary(a *OwnedRational, op rationalFacadeUnaryOp, p uint32) (_lexOut OwnedRational, _lexErr error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return OwnedRational{}, e
	}
	defer s.endLexicalRational(&_lexOut, &_lexErr)
	if op == facadeNumerator || op == facadeDenominator {
		if e := s.rationalScalarOwner(a); e != nil {
			return OwnedRational{}, e
		}
	} else {
		if e := s.check(a); e != nil {
			return OwnedRational{}, e
		}
	}
	work := NewRational()
	workComplete := op != facadeNeg && op != facadeAbs
	defer s.endLexicalLocal(&work, &workComplete)
	var e error
	switch op {
	case facadeNeg, facadeAbs:
		e = s.manager.Copy(&work, &a.raw)
		if e == nil {
			workComplete = true
			if op == facadeNeg {
				e = s.manager.Neg(&work)
			} else {
				e = s.manager.Abs(&work)
			}
		}
	case facadeFloor:
		e = s.manager.Floor(&a.raw, &work)
	case facadeCeil:
		e = s.manager.Ceil(&a.raw, &work)
	case facadeNumerator:
		e = s.manager.NumeratorRational(&a.raw, &work)
	case facadeDenominator:
		e = s.manager.DenominatorRational(&a.raw, &work)
	case facadePower:
		e = s.manager.Power(&a.raw, p, &work)
	default:
		e = ErrRationalFacadeDomain
	}
	if e != nil {
		return OwnedRational{}, e
	}
	return s.facadeTakeLocked(&work)
}
func (s *RationalService) Neg(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeNeg, 0)
}
func (s *RationalService) Abs(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeAbs, 0)
}
func (s *RationalService) Floor(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeFloor, 0)
}
func (s *RationalService) Ceil(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeCeil, 0)
}
func (s *RationalService) Numerator(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeNumerator, 0)
}
func (s *RationalService) Denominator(a *OwnedRational) (OwnedRational, error) {
	return s.facadeUnary(a, facadeDenominator, 0)
}
func (s *RationalService) Power(a *OwnedRational, p uint32) (OwnedRational, error) {
	return s.facadeUnary(a, facadePower, p)
}

// Expt preserves source signed-to-unsigned forwarding, never reciprocal power.
func (s *RationalService) Expt(a *OwnedRational, p int32) (OwnedRational, error) {
	return s.Power(a, uint32(p))
}
func (s *RationalService) NegAssign(a *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return e
	}
	return s.manager.Neg(&a.raw)
}
func (s *RationalService) Reset(a *OwnedRational) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return e
	}
	return s.manager.Reset(&a.raw)
}

// Comparison routes retain operand reversal, structural equality, and source
// sign shortcuts even for noncanonical raw denominators.
func (s *RationalService) facadeCompareLocked(a, b *OwnedRational, op uint8) (bool, error) {
	if e := s.check(a); e != nil {
		return false, e
	}
	if e := s.check(b); e != nil {
		return false, e
	}
	if op == 0 {
		v, e := s.manager.Equal(&a.raw, &b.raw)
		return !v, e
	}
	x, y := a, b
	if op == 2 || op == 3 {
		x, y = b, a
	}
	c, e := s.manager.Compare(&x.raw, &y.raw)
	if op == 3 || op == 4 {
		return c >= 0, e
	}
	return c < 0, e
}
func (s *RationalService) facadeCompare(a, b *OwnedRational, op uint8) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.facadeCompareLocked(a, b, op)
}
func (s *RationalService) NotEqual(a, b *OwnedRational) (bool, error) {
	return s.facadeCompare(a, b, 0)
}
func (s *RationalService) Less(a, b *OwnedRational) (bool, error)    { return s.facadeCompare(a, b, 1) }
func (s *RationalService) Greater(a, b *OwnedRational) (bool, error) { return s.facadeCompare(a, b, 2) }
func (s *RationalService) LessEqual(a, b *OwnedRational) (bool, error) {
	return s.facadeCompare(a, b, 3)
}
func (s *RationalService) GreaterEqual(a, b *OwnedRational) (bool, error) {
	return s.facadeCompare(a, b, 4)
}
func (s *RationalService) facadeCompareInt(a *OwnedRational, n int32, op uint8) (_lexValue bool, _lexErr error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return false, e
	}
	defer s.endLexicalError(&_lexErr)
	if e := s.check(a); e != nil {
		return false, e
	}
	v := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	vComplete := false
	defer s.endLexicalLocal(&v.raw, &vComplete)
	if e := s.manager.SetInt64(&v.raw, int64(n)); e != nil {
		return false, e
	}
	vComplete = true
	if op == 5 {
		return s.manager.Equal(&a.raw, &v.raw)
	}
	return s.facadeCompareLocked(a, &v, op)
}
func (s *RationalService) NotEqualInt(a *OwnedRational, n int32) (bool, error) {
	return s.facadeCompareInt(a, n, 0)
}
func (s *RationalService) LessInt(a *OwnedRational, n int32) (bool, error) {
	return s.facadeCompareInt(a, n, 1)
}
func (s *RationalService) GreaterInt(a *OwnedRational, n int32) (bool, error) {
	return s.facadeCompareInt(a, n, 2)
}
func (s *RationalService) EqualInt(a *OwnedRational, n int32) (bool, error) {
	return s.facadeCompareInt(a, n, 5)
}
func (s *RationalService) IsPos(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsPos(&a.raw)
}
func (s *RationalService) IsNeg(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsNeg(&a.raw)
}
func (s *RationalService) IsZero(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsZero(&a.raw)
}
func (s *RationalService) IsOne(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsOne(&a.raw)
}
func (s *RationalService) IsMinusOne(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsMinusOne(&a.raw)
}
func (s *RationalService) IsNonNeg(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsNonNeg(&a.raw)
}
func (s *RationalService) IsNonPos(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsNonPos(&a.raw)
}

// Projection provenance guards do not inspect unread numeral components.
func (s *RationalService) scalarProjectionOwner(ownerID uint64) error {
	if ownerID == 0 || ownerID != s.ownerID {
		return ErrRationalServiceOwner
	}
	return s.manager.checkOwnerReady()
}
func (s *RationalService) rationalScalarOwner(a *OwnedRational) error {
	if a == nil {
		return ErrRationalServiceOwner
	}
	return s.scalarProjectionOwner(a.ownerID)
}
func (s *RationalService) IsInt(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsInt(&a.raw)
}
func (s *RationalService) IsUint64(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsUint64(&a.raw)
}
func (s *RationalService) IsInt64(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsInt64(&a.raw)
}
func (s *RationalService) IsUnsigned(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsUnsigned(&a.raw)
}
func (s *RationalService) IsInt32(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return false, e
	}
	return s.manager.IsInt32(&a.raw)
}
func (s *RationalService) IsEven(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return false, e
	}
	return s.manager.IsEven(&a.raw)
}
func (s *RationalService) Uint64(a *OwnedRational) (uint64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Uint64(&a.raw)
}
func (s *RationalService) Int64(a *OwnedRational) (int64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Int64(&a.raw)
}
func (s *RationalService) Unsigned(a *OwnedRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Unsigned(&a.raw)
}
func (s *RationalService) Int32(a *OwnedRational) (int32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Int32(&a.raw)
}
func (s *RationalService) Hash(a *OwnedRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Hash(&a.raw)
}
func (s *RationalService) Bitsize(a *OwnedRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return 0, e
	}
	return s.manager.Bitsize(&a.raw)
}
func (s *RationalService) StorageSize(a *OwnedRational) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return 0, e
	}
	return s.manager.StorageSize(&a.raw)
}
func (s *RationalService) Float64(a *OwnedRational) (float64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.rationalScalarOwner(a); e != nil {
		return 0, e
	}
	return s.manager.Float64(&a.raw)
}
func (s *RationalService) IsSmall(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return false, e
	}
	return a.raw.IsSmall(), nil
}
func (s *RationalService) IsBig(a *OwnedRational) (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return false, e
	}
	return !a.raw.IsSmall(), nil
}
func (s *RationalService) IsPowerOfTwo(a *OwnedRational) (bool, uint32, error) {
	if e := s.lock(); e != nil {
		return false, 0, e
	}
	defer s.mu.Unlock()
	if e := s.check(a); e != nil {
		return false, 0, e
	}
	return s.manager.IsPowerOfTwo(&a.raw)
}
func (s *RationalService) facadeAddMul(a, c, k *OwnedRational, subtract bool) (_lexErr error) {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	for _, v := range []*OwnedRational{a, c, k} {
		if e := s.check(v); e != nil {
			return e
		}
	}
	one, e := s.manager.IsOne(&c.raw)
	if e != nil {
		return e
	}
	minus, e := s.manager.IsMinusOne(&c.raw)
	if e != nil {
		return e
	}
	op := rationalFacadeAdd
	if subtract {
		op = rationalFacadeSub
	}
	if one {
		return s.facadeAssignLocked(a, k, op)
	}
	if minus {
		if subtract {
			op = rationalFacadeAdd
		} else {
			op = rationalFacadeSub
		}
		return s.facadeAssignLocked(a, k, op)
	}
	temp := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	tempComplete := false
	defer s.endLexicalLocal(&temp.raw, &tempComplete)
	if e := s.manager.Copy(&temp.raw, &k.raw); e != nil {
		return e
	}
	tempComplete = true
	if e := s.facadeAssignLocked(&temp, c, rationalFacadeMul); e != nil {
		return e
	}
	return s.facadeAssignLocked(a, &temp, op)
}
func (s *RationalService) AddMulAssign(a, c, k *OwnedRational) error {
	return s.facadeAddMul(a, c, k, false)
}
func (s *RationalService) SubMulAssign(a, c, k *OwnedRational) error {
	return s.facadeAddMul(a, c, k, true)
}
func (s *RationalService) ModHat(a, b *OwnedRational) (_lexOut OwnedRational, _lexErr error) {
	if e := s.lock(); e != nil {
		return OwnedRational{}, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return OwnedRational{}, e
	}
	defer s.endLexicalRational(&_lexOut, &_lexErr)
	if e := s.check(a); e != nil {
		return OwnedRational{}, e
	}
	if e := s.check(b); e != nil {
		return OwnedRational{}, e
	}
	pos, e := s.manager.IsPos(&b.raw)
	if e != nil {
		return OwnedRational{}, e
	}
	if !pos {
		return OwnedRational{}, ErrRationalFacadeDomain
	}
	r, e := s.facadeValueLocked(a, b, rationalFacadeMod)
	if e != nil {
		return OwnedRational{}, e
	}
	defer s.destroyLexicalRational(&r.raw)
	nonneg, e := s.manager.IsNonNeg(&r.raw)
	if e != nil {
		return OwnedRational{}, e
	}
	if !nonneg {
		return OwnedRational{}, ErrRationalFacadeDomain
	}
	r2 := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	r2Complete := false
	defer s.endLexicalLocal(&r2.raw, &r2Complete)
	if e := s.manager.Copy(&r2.raw, &r.raw); e != nil {
		return OwnedRational{}, e
	}
	r2Complete = true
	two := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	twoComplete := false
	defer s.endLexicalLocal(&two.raw, &twoComplete)
	if e := s.manager.SetInt64(&two.raw, 2); e != nil {
		return OwnedRational{}, e
	}
	twoComplete = true
	if e := s.facadeAssignLocked(&r2, &two, rationalFacadeMul); e != nil {
		return OwnedRational{}, e
	}
	// Source rational(2) is destroyed at this full-expression boundary, before
	// the comparison and optional subtraction. Keep a valid empty value for
	// deferred cleanup on every later error path.
	if e := s.destroyLexicalRational(&two.raw); e != nil {
		return OwnedRational{}, e
	}
	two.raw = NewRational()
	lt, e := s.facadeCompareLocked(b, &r2, 1)
	if e != nil {
		return OwnedRational{}, e
	}
	if lt {
		if e := s.facadeAssignLocked(&r, b, rationalFacadeSub); e != nil {
			return OwnedRational{}, e
		}
	}
	return s.facadeTakeLocked(&r.raw)
}
func (s *RationalService) NumBits(a *OwnedRational) (_lexValue uint32, _lexErr error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return 0, e
	}
	defer s.endLexicalError(&_lexErr)
	if e := s.check(a); e != nil {
		return 0, e
	}
	two := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	twoComplete := false
	defer s.endLexicalLocal(&two.raw, &twoComplete)
	if e := s.manager.SetInt64(&two.raw, 2); e != nil {
		return 0, e
	}
	twoComplete = true
	integer, e := s.manager.IsInt(&a.raw)
	if e != nil {
		return 0, e
	}
	if !integer {
		return 0, ErrRationalNotInteger
	}
	neg, e := s.manager.IsNeg(&a.raw)
	if e != nil {
		return 0, e
	}
	if neg {
		return 0, ErrRationalFacadeDomain
	}
	n := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	nComplete := false
	defer s.endLexicalLocal(&n.raw, &nComplete)
	if e := s.manager.Copy(&n.raw, &a.raw); e != nil {
		return 0, e
	}
	nComplete = true
	count := uint32(1)
	for {
		quotient, e := s.facadeValueLocked(&n, &two, rationalFacadeIDiv)
		if e != nil {
			return 0, e
		}
		e = s.manager.Copy(&n.raw, &quotient.raw)
		if cleanup := s.destroyLexicalRational(&quotient.raw); e == nil {
			e = cleanup
		}
		if e != nil {
			return 0, e
		}
		pos, e := s.manager.IsPos(&n.raw)
		if e != nil {
			return 0, e
		}
		if !pos {
			return count, nil
		}
		count++
	}
}
