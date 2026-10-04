// Licensed under the MIT license.
//go:build g3debug

// Source MPBQ_DEBUG rational actions.
package mpnum

func (m *BinaryRationalManager) debugSelected(l, u, r *BinaryRational) error {
	yes, e := m.LessEqual(l, r)
	if e != nil {
		return e
	}
	if e = m.debugAssert(yes); e != nil {
		return e
	}
	yes, e = m.LessEqual(r, u)
	if e != nil {
		return e
	}
	return m.debugAssert(yes)
}
func (m *BinaryRationalManager) debugEnclosure(qm *RationalManager, q *Rational, l, u *BinaryRational) error {
	lo, e := m.LessRational(l, qm, q)
	if e != nil {
		return e
	}
	if !lo {
		return ErrBinaryInvariant
	}
	hi, e := m.GreaterRational(u, qm, q)
	if e != nil {
		return e
	}
	return m.debugAssert(lo && hi)
}
func (m *BinaryRationalManager) debugRationalEqual(a *BinaryRational, qm *RationalManager, q *Rational) error {
	yes, e := m.EqualRational(a, qm, q)
	if e != nil {
		return e
	}
	return m.debugAssert(yes)
}

func binaryCheckDiagnostics(z *IntegerManager, s *RationalService) error {
	if s == nil || s.self != s || s.closed || s.manager == nil || s.manager.budget != z.budget {
		return ErrBinaryOwner
	}
	return nil
}
func (m *BinaryRationalManager) debugAssert(ok bool) error {
	if !ok {
		return ErrBinaryInvariant
	}
	return nil
}
func (m *BinaryRationalManager) debugValue(a *BinaryRational) (OwnedRational, error) {
	return m.ToOwnedRational(m.diagnostics, a)
}
func (m *BinaryRationalManager) debugInteger(a *Int) (OwnedRational, error) {
	s := m.diagnostics
	v, e := s.NewValue()
	if e != nil {
		return OwnedRational{}, e
	}
	if e = s.lock(); e != nil {
		s.Del(&v)
		return OwnedRational{}, e
	}
	e = s.manager.z.CopyFrom(&v.raw.num, m.z, a)
	s.mu.Unlock()
	if e != nil {
		s.Del(&v)
		return OwnedRational{}, e
	}
	return v, nil
}
func (m *BinaryRationalManager) debugDel(a *OwnedRational) {
	if a.ownerID != 0 {
		m.diagnostics.Del(a)
	}
}
func (m *BinaryRationalManager) debugEqualValue(old *OwnedRational, a *BinaryRational) error {
	v, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&v)
	same, e := m.diagnostics.Equal(old, &v)
	if e != nil {
		return e
	}
	return m.debugAssert(same)
}
func (m *BinaryRationalManager) debugArithmetic(a, b *OwnedRational, out *BinaryRational, op uint8) error {
	r, e := m.debugValue(out)
	if e != nil {
		return e
	}
	defer m.debugDel(&r)
	var want OwnedRational
	switch op {
	case 0:
		want, e = m.diagnostics.Add(a, b)
	case 1:
		want, e = m.diagnostics.Sub(a, b)
	case 2:
		want, e = m.diagnostics.Mul(a, b)
	}
	if e != nil {
		return e
	}
	defer m.debugDel(&want)
	same, e := m.diagnostics.Equal(&want, &r)
	if e != nil {
		return e
	}
	return m.debugAssert(same)
}
func (m *BinaryRationalManager) debugApproxSnapshot(a, old *BinaryRational) error {
	return m.Copy(old, a)
}
func (m *BinaryRationalManager) debugApproxDrop(old *BinaryRational) { m.Del(old) }
func (m *BinaryRationalManager) debugApprox(old, a *BinaryRational, plus bool) error {
	var yes bool
	var e error
	if plus {
		yes, e = m.Less(old, a)
	} else {
		yes, e = m.Less(a, old)
	}
	if e != nil {
		return e
	}
	return m.debugAssert(yes)
}
