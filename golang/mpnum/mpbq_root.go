// Copyright (c) 2011 Microsoft Corporation. Source: src/util/mpbq.h/.cpp.
package mpnum

func (m *BinaryRationalManager) RootLower(a *BinaryRational, n uint32, out *BinaryRational) (bool, error) {
	if e := m.Copy(out, a); e != nil {
		return false, e
	}
	return m.RootLowerInPlace(out, n)
}
func (m *BinaryRationalManager) RootUpper(a *BinaryRational, n uint32, out *BinaryRational) (bool, error) {
	if e := m.Copy(out, a); e != nil {
		return false, e
	}
	return m.RootUpperInPlace(out, n)
}
func (m *BinaryRationalManager) RootLowerInPlace(a *BinaryRational, n uint32) (bool, error) {
	return m.root(a, n, true)
}
func (m *BinaryRationalManager) RootUpperInPlace(a *BinaryRational, n uint32) (bool, error) {
	return m.root(a, n, false)
}
func (m *BinaryRationalManager) root(a *BinaryRational, n uint32, lower bool) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if n == 0 {
		return false, ErrBinaryRootDegreeZero
	}
	exact, e := m.z.RootInPlace(&a.num, n)
	if e != nil {
		return false, e
	}
	if lower && !exact {
		if e = m.z.Dec(&a.num); e != nil {
			return false, e
		}
	}
	if a.k%n == 0 {
		a.k /= n
		return exact, m.normalize(a)
	}
	neg := a.num.val < 0
	a.k /= n
	if lower && !neg || !lower && neg {
		a.k++
	}
	return false, m.normalize(a)
}
func (m *BinaryRationalManager) Floor(a *BinaryRational, out *Int) error {
	if e := m.open(); e != nil {
		return e
	}
	return m.FloorInto(a, m.z, out)
}
func (m *BinaryRationalManager) Ceil(a *BinaryRational, out *Int) error {
	if e := m.open(); e != nil {
		return e
	}
	return m.CeilInto(a, m.z, out)
}
func (m *BinaryRationalManager) FloorInto(a *BinaryRational, z *IntegerManager, out *Int) error {
	return m.round(a, z, out, false)
}
func (m *BinaryRationalManager) CeilInto(a *BinaryRational, z *IntegerManager, out *Int) error {
	return m.round(a, z, out, true)
}
func (m *BinaryRationalManager) round(a *BinaryRational, z *IntegerManager, out *Int, ceil bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if z == nil || z.closed {
		return ErrBinaryOwner
	}
	if e := z.check(out); e != nil {
		return e
	}
	var local Int
	defer z.Del(&local)
	input := &a.num
	if z != m.z {
		if e := z.CopyFrom(&local, m.z, input); e != nil {
			return e
		}
		input = &local
	}
	if a.k == 0 {
		return z.Copy(out, input)
	}
	neg, pos := a.num.val < 0, a.num.val > 0
	if e := z.MachineDiv2k(input, a.k, out); e != nil {
		return e
	}
	one := NewInt(1)
	if ceil && pos {
		return z.Add(out, &one, out)
	}
	if !ceil && neg {
		return z.Sub(out, &one, out)
	}
	return nil
}
func (m *BinaryRationalManager) FromRational(qm *RationalManager, q *Rational, out *BinaryRational) (bool, error) {
	if e := m.check(out); e != nil {
		return false, e
	}
	local := NewRational()
	defer m.dropImported(&local)
	input, e := m.importRational(qm, q, &local)
	if e != nil {
		return false, e
	}
	if isOneInteger(&input.den) {
		if e = m.SetInteger(out, &input.num); e != nil {
			return false, e
		}
		return true, m.debugRationalEqual(out, qm, q)
	}
	yes, k, e := m.z.IsPowerOfTwo(&input.den)
	if e != nil {
		return false, e
	}
	if yes {
		if e = m.debugAssert(k >= 1); e != nil {
			return false, e
		}
		if e = m.SetIntegerPower(out, &input.num, k); e != nil {
			return false, e
		}
		return true, m.debugRationalEqual(out, qm, q)
	}
	k, e = m.z.Log2(&input.den)
	if e != nil {
		return false, e
	}
	return false, m.SetIntegerPower(out, &input.num, k+1)
}
func (m *BinaryRationalManager) ToRational(qm *RationalManager, a *BinaryRational, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if qm == nil || qm.z == nil {
		return ErrBinaryOwner
	}
	if e := qm.check(out); e != nil {
		return e
	}
	two := Rational{num: NewInt(2), den: NewInt(1)}
	if e := qm.Power(&two, a.k, out); e != nil {
		return e
	}
	if e := qm.Inv(out, out); e != nil {
		return e
	}
	var local Int
	defer qm.z.Del(&local)
	n := &a.num
	if qm.z != m.z {
		if e := qm.z.CopyFrom(&local, m.z, n); e != nil {
			return e
		}
		n = &local
	}
	return qm.MulInt(n, out, out)
}
func (m *BinaryRationalManager) ToOwnedRational(s *RationalService, a *BinaryRational) (OwnedRational, error) {
	if e := m.check(a); e != nil {
		return OwnedRational{}, e
	}
	if s == nil {
		return OwnedRational{}, ErrBinaryOwner
	}
	r, e := s.NewValue()
	if e != nil {
		return OwnedRational{}, e
	}
	defer s.Del(&r)
	if e = s.lock(); e != nil {
		return OwnedRational{}, e
	}
	e = s.manager.z.CopyFrom(&r.raw.num, m.z, &a.num)
	s.mu.Unlock()
	if e != nil {
		return OwnedRational{}, e
	}
	twok, e := s.NewValue()
	if e != nil {
		return OwnedRational{}, e
	}
	defer s.Del(&twok)
	two, e := s.NewValue()
	if e != nil {
		return OwnedRational{}, e
	}
	defer func() {
		if two.ownerID != 0 {
			s.Del(&two)
		}
	}()
	if e = s.SetInt64(&two, 2); e != nil {
		return OwnedRational{}, e
	}
	pw, e := s.Power(&two, a.k)
	if e != nil {
		return OwnedRational{}, e
	}
	defer func() {
		if pw.ownerID != 0 {
			s.Del(&pw)
		}
	}()
	if e = s.Copy(&twok, &pw); e != nil {
		return OwnedRational{}, e
	}
	s.Del(&pw)
	s.Del(&two)
	return s.Div(&r, &twok)
}
