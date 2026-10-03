// Copyright (c) 2012 Microsoft Corporation. Source MPFF conversions, internal32.
package mpnum

// Selected source MPZ digit setter over a concrete ephemeral owner-indexed view.
// Retains existing MPZ storage staging; no temporary unbudgeted limb slice.
func ffSetInteger(z *IntegerManager, a *Int, v ffLimbView) error {
	if e := z.check(a); e != nil {
		return e
	}
	for v.size > 0 && v.read(v.size-1) == 0 {
		v.size--
	}
	if v.size == 0 {
		return z.Reset(a)
	}
	if v.size == 1 {
		return z.SetUint32(a, v.read(0))
	}
	ref := a.cell
	if ref == 0 || z.cell(ref).capacity < v.size {
		var e error
		ref, e = z.allocate(max(v.size, 6), v.size)
		if e != nil {
			return e
		}
	} else {
		z.resizeCell(ref, v.size)
	}
	for i := uint32(0); i < v.size; i++ {
		z.writeDigit(ref, i, v.read(i))
	}
	old := a.cell
	a.cell = ref
	a.val = 1
	if old != 0 && old != ref {
		z.free(old)
	}
	return nil
}
func (m *FastFloatManager) decompose(z *IntegerManager, a *Int) (bool, error) {
	if e := m.resize(&m.setBuffer, 0); e != nil {
		return false, e
	}
	size := uint32(1)
	if a.cell != 0 {
		size = z.cell(a.cell).size
	}
	for i := uint32(0); i < size; i++ {
		v := uint32(0)
		if a.cell == 0 {
			n := int64(a.val)
			if n < 0 {
				n = -n
			}
			v = uint32(n)
		} else {
			v = z.digit(a.cell, i)
		}
		if e := m.resize(&m.setBuffer, i+1); e != nil {
			return false, e
		}
		m.write(&m.setBuffer, i, v)
	}
	return a.val < 0, nil
}
func (m *FastFloatManager) SetInteger(a *FastFloat, z *IntegerManager, v *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if z == nil {
		return ErrFastFloatOwner
	}
	if e := z.check(v); e != nil {
		return e
	}
	ok, e := z.IsInt64(v)
	if e != nil {
		return e
	}
	if ok {
		n, e := z.Int64(v)
		if e != nil {
			return e
		}
		return m.SetInt64(a, n)
	}
	ok, e = z.IsUint64(v)
	if e != nil {
		return e
	}
	if ok {
		n, e := z.Uint64(v)
		if e != nil {
			return e
		}
		return m.SetUint64(a, n)
	}
	if e = m.allocate(a); e != nil {
		return e
	}
	negative, e := m.decompose(z, v)
	if e != nil {
		return e
	}
	a.sign(negative)
	for m.setBuffer.size < m.precision {
		if e = m.resize(&m.setBuffer, m.setBuffer.size+1); e != nil {
			return e
		}
		m.write(&m.setBuffer, m.setBuffer.size-1, 0)
	}
	w := ffLimbView{m: m, v: &m.setBuffer, size: m.setBuffer.size}
	lz := ffNLZ(w)
	if sourceExtendedGCDDebug && !(lz != 0) {
		return ErrFastFloatInvariant
	}
	ffShift(w, w, lz, true)
	s := m.sig(a)
	j := w.size
	for i := m.precision; i > 0; {
		i--
		j--
		s.write(i, w.read(j))
	}
	a.exponent = int32(j*32 - lz)
	if a.negative() != m.toPlus {
		for j > 0 {
			j--
			if w.read(j) != 0 {
				if e = m.incSig(a); e != nil {
					return e
				}
			}
		}
	}
	return m.debugCheck(a)
}
func (m *FastFloatManager) SetRational(a *FastFloat, qm *RationalManager, v *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if qm == nil {
		return ErrFastFloatOwner
	}
	if e := qm.check(v); e != nil {
		return e
	}
	var n, d FastFloat
	defer func() { m.Del(&d); m.Del(&n) }()
	if e := m.SetInteger(&n, qm.z, &v.num); e != nil {
		return e
	}
	e := func() error {
		old := m.toPlus
		m.toPlus = !old
		defer func() { m.toPlus = old }()
		return m.SetInteger(&d, qm.z, &v.den)
	}()
	if e != nil {
		return e
	}
	return m.Div(&n, &d, a)
}
func (m *FastFloatManager) Significand(a *FastFloat, z *IntegerManager, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if z == nil {
		return ErrFastFloatOwner
	}
	return ffSetInteger(z, out, m.sig(a))
}
func (m *FastFloatManager) ToInteger(a *FastFloat, z *IntegerManager, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if z == nil {
		return ErrFastFloatOwner
	}
	if e := z.check(out); e != nil {
		return e
	}
	if sourceExtendedGCDDebug {
		ok, e := m.IsInteger(a)
		if e != nil {
			return e
		}
		if !ok {
			return ErrFastFloatInvariant
		}
	}
	var e error
	exp := a.exponent
	if exp < 0 {
		if sourceExtendedGCDDebug && exp <= -int32(m.bits) {
			return ErrFastFloatInvariant
		}
		m.toBuffer(0, a, false, false)
		// Source negates an int32 here. INT_MIN is undefined signed
		// arithmetic; retain its completed buffer copy before diagnosing it.
		if exp == -2147483648 {
			return ErrFastFloatDomain
		}
		v := m.buffer(0, m.precision)
		ffShift(v, v, uint32(-exp), false)
		if e = ffSetInteger(z, out, v); e != nil {
			return e
		}
	} else {
		if e = ffSetInteger(z, out, m.sig(a)); e != nil {
			return e
		}
		if exp > 0 {
			p := NewInt(2)
			defer z.Del(&p)
			if e = z.Power(&p, uint32(exp), &p); e != nil {
				return e
			}
			if e = z.Mul(out, &p, out); e != nil {
				return e
			}
		}
	}
	if a.negative() {
		return z.Neg(out)
	}
	return nil
}
func ffSetRational(qm *RationalManager, out *Rational, v ffLimbView) error {
	if e := qm.check(out); e != nil {
		return e
	}
	if e := ffSetInteger(qm.z, &out.num, v); e != nil {
		return e
	}
	return qm.resetDenominator(out)
}
func (m *FastFloatManager) ToRational(a *FastFloat, qm *RationalManager, out *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if qm == nil {
		return ErrFastFloatOwner
	}
	if e := qm.check(out); e != nil {
		return e
	}
	exp := a.exponent
	if exp < 0 && exp > -int32(m.bits) && !ffHasLow(m.sig(a), uint32(-exp)) {
		m.toBuffer(0, a, false, false)
		v := m.buffer(0, m.precision)
		ffShift(v, v, uint32(-exp), false)
		if e := ffSetRational(qm, out, v); e != nil {
			return e
		}
	} else {
		if e := ffSetRational(qm, out, m.sig(a)); e != nil {
			return e
		}
		if exp != 0 {
			p := NewRational()
			defer qm.Del(&p)
			if e := qm.SetInt64(&p, 2); e != nil {
				return e
			}
			power := uint32(exp)
			if exp < 0 {
				power = uint32(-int64(exp))
			}
			if e := qm.Power(&p, power, &p); e != nil {
				return e
			}
			if exp < 0 {
				if e := qm.Div(out, &p, out); e != nil {
					return e
				}
			} else {
				if e := qm.Mul(out, &p, out); e != nil {
					return e
				}
			}
		}
	}
	if a.negative() {
		return qm.Neg(out)
	}
	return nil
}
