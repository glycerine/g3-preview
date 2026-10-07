// Source: src/util/mpbq.cpp.
package util

func (m *BinaryRationalManager) selectInteger(lower, upper *BinaryRational, r *Int) (bool, error) {
	if lower.KField == 0 {
		return true, m.Z.Copy(r, &lower.Num)
	}
	if upper.KField == 0 {
		return true, m.Z.Copy(r, &upper.Num)
	}
	if e := m.Ceil(lower, &m.SelectIntTmp1); e != nil {
		return false, e
	}
	if e := m.Floor(upper, &m.SelectIntTmp2); e != nil {
		return false, e
	}
	yes, e := m.Z.LessEqual(&m.SelectIntTmp1, &m.SelectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.Z.Copy(r, &m.SelectIntTmp1)
}
func (m *BinaryRationalManager) rationalIntegerBound(qm *RationalManager, q *Rational, out *Int, upper bool) error {
	if isIntegerRational(q) {
		if e := m.Z.CopyFrom(out, qm.Z, &q.Num); e != nil {
			return e
		}
		if upper {
			return m.Z.Dec(out)
		}
		return m.Z.Inc(out)
	}
	var tmp Int
	defer qm.Z.Del(&tmp)
	var e error
	if upper {
		e = qm.FloorToInt(q, &tmp)
	} else {
		e = qm.CeilToInt(q, &tmp)
	}
	if e != nil {
		return e
	}
	return m.Z.CopyFrom(out, qm.Z, &tmp)
}
func (m *BinaryRationalManager) selectIntegerRationalLower(qm *RationalManager, lower *Rational, upper *BinaryRational, r *Int) (bool, error) {
	if upper.KField == 0 {
		return true, m.Z.Copy(r, &upper.Num)
	}
	if e := m.rationalIntegerBound(qm, lower, &m.SelectIntTmp1, false); e != nil {
		return false, e
	}
	if e := m.Floor(upper, &m.SelectIntTmp2); e != nil {
		return false, e
	}
	yes, e := m.Z.LessEqual(&m.SelectIntTmp1, &m.SelectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.Z.Copy(r, &m.SelectIntTmp1)
}
func (m *BinaryRationalManager) selectIntegerRationalUpper(qm *RationalManager, lower *BinaryRational, upper *Rational, r *Int) (bool, error) {
	if lower.KField == 0 {
		return true, m.Z.Copy(r, &lower.Num)
	}
	if e := m.Ceil(lower, &m.SelectIntTmp1); e != nil {
		return false, e
	}
	if e := m.rationalIntegerBound(qm, upper, &m.SelectIntTmp2, true); e != nil {
		return false, e
	}
	yes, e := m.Z.LessEqual(&m.SelectIntTmp1, &m.SelectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.Z.Copy(r, &m.SelectIntTmp1)
}
func (m *BinaryRationalManager) selectIntegerRationalBounds(qm *RationalManager, lower, upper *Rational, r *Int) (bool, error) {
	if e := m.rationalIntegerBound(qm, lower, &m.SelectIntTmp1, false); e != nil {
		return false, e
	}
	if e := m.rationalIntegerBound(qm, upper, &m.SelectIntTmp2, true); e != nil {
		return false, e
	}
	yes, e := m.Z.LessEqual(&m.SelectIntTmp1, &m.SelectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.Z.Copy(r, &m.SelectIntTmp1)
}
func (m *BinaryRationalManager) SelectSmall(lower, upper, r *BinaryRational) (bool, error) {
	if e := m.triple(lower, upper, r); e != nil {
		return false, e
	}
	bad, e := m.Greater(lower, upper)
	if e != nil || bad {
		return false, e
	}
	return true, m.SelectSmallCore(lower, upper, r)
}
func (m *BinaryRationalManager) SelectSmallCore(lower, upper, r *BinaryRational) error {
	if e := m.triple(lower, upper, r); e != nil {
		return e
	}
	yes, e := m.LessEqual(lower, upper)
	if e != nil {
		return e
	}
	if !yes {
		return ErrBinaryInterval
	}
	aux := &m.SelectSmallTmp
	yes, e = m.selectInteger(lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	minK, maxK := uint32(0), lower.KField
	if upper.KField < maxK {
		maxK = upper.KField
	}
	l2k, u2k := &m.SelectSmallTmp1, &m.SelectSmallTmp2
	if maxK <= 8 {
		k := uint32(0)
		if e = m.Copy(l2k, lower); e != nil {
			return e
		}
		if e = m.Copy(u2k, upper); e != nil {
			return e
		}
		for {
			k++
			if e = m.Mul2(l2k); e != nil {
				return e
			}
			if e = m.Mul2(u2k); e != nil {
				return e
			}
			yes, e = m.selectInteger(l2k, u2k, aux)
			if e != nil {
				return e
			}
			if yes {
				if e = m.SetIntegerPower(r, aux, k); e != nil {
					return e
				}
				break
			}
		}
	} else {
		for {
			mid := minK + (maxK-minK)/2
			if e = m.Copy(l2k, lower); e != nil {
				return e
			}
			if e = m.Copy(u2k, upper); e != nil {
				return e
			}
			if e = m.Mul2k(l2k, mid); e != nil {
				return e
			}
			if e = m.Mul2k(u2k, mid); e != nil {
				return e
			}
			yes, e = m.selectInteger(l2k, u2k, aux)
			if e != nil {
				return e
			}
			if yes {
				maxK = mid
			} else {
				minK = mid + 1
			}
			if minK == maxK {
				if maxK != mid {
					if e = m.Copy(l2k, lower); e != nil {
						return e
					}
					if e = m.Copy(u2k, upper); e != nil {
						return e
					}
					if e = m.Mul2k(l2k, maxK); e != nil {
						return e
					}
					if e = m.Mul2k(u2k, maxK); e != nil {
						return e
					}
					yes, e = m.selectInteger(l2k, u2k, aux)
					if e != nil {
						return e
					}
					if !yes {
						return ErrBinaryInvariant
					}
				}
				if e = m.SetIntegerPower(r, aux, maxK); e != nil {
					return e
				}
				break
			}
		}
	}
	return m.debugSelected(lower, upper, r)
}
func (m *BinaryRationalManager) SelectSmallRationalLower(qm *RationalManager, lower *Rational, upper, r *BinaryRational) error {
	if e := m.pair(upper, r); e != nil {
		return e
	}
	if qm == nil {
		return ErrBinaryOwner
	}
	if e := qm.check(lower); e != nil {
		return e
	}
	yes, e := m.GreaterRational(upper, qm, lower)
	if e != nil {
		return e
	}
	if !yes {
		return ErrBinaryInterval
	}
	aux := &m.SelectSmallTmp
	yes, e = m.selectIntegerRationalLower(qm, lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	l2k := NewRational()
	defer qm.drop(&l2k)
	two := Rational{Num: NewInt(2), Den: NewInt(1)}
	u2k := &m.SelectSmallTmp2
	if e = qm.Copy(&l2k, lower); e != nil {
		return e
	}
	if e = m.Copy(u2k, upper); e != nil {
		return e
	}
	for k := uint32(1); ; k++ {
		if e = qm.Mul(&l2k, &two, &l2k); e != nil {
			return e
		}
		if e = m.Mul2(u2k); e != nil {
			return e
		}
		yes, e = m.selectIntegerRationalLower(qm, &l2k, u2k, aux)
		if e != nil {
			return e
		}
		if yes {
			return m.SetIntegerPower(r, aux, k)
		}
	}
}
func (m *BinaryRationalManager) SelectSmallRationalUpper(qm *RationalManager, lower *BinaryRational, upper *Rational, r *BinaryRational) error {
	if e := m.pair(lower, r); e != nil {
		return e
	}
	if qm == nil {
		return ErrBinaryOwner
	}
	if e := qm.check(upper); e != nil {
		return e
	}
	yes, e := m.LessRational(lower, qm, upper)
	if e != nil {
		return e
	}
	if !yes {
		return ErrBinaryInterval
	}
	aux := &m.SelectSmallTmp
	yes, e = m.selectIntegerRationalUpper(qm, lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	u2k := NewRational()
	defer qm.drop(&u2k)
	two := Rational{Num: NewInt(2), Den: NewInt(1)}
	l2k := &m.SelectSmallTmp2
	if e = m.Copy(l2k, lower); e != nil {
		return e
	}
	if e = qm.Copy(&u2k, upper); e != nil {
		return e
	}
	for k := uint32(1); ; k++ {
		if e = m.Mul2(l2k); e != nil {
			return e
		}
		if e = qm.Mul(&u2k, &two, &u2k); e != nil {
			return e
		}
		yes, e = m.selectIntegerRationalUpper(qm, l2k, &u2k, aux)
		if e != nil {
			return e
		}
		if yes {
			return m.SetIntegerPower(r, aux, k)
		}
	}
}
func (m *BinaryRationalManager) SelectSmallRationalBounds(qm *RationalManager, lower, upper *Rational, r *BinaryRational) error {
	if e := m.check(r); e != nil {
		return e
	}
	if qm == nil {
		return ErrBinaryOwner
	}
	if e := qm.check(lower); e != nil {
		return e
	}
	if e := qm.check(upper); e != nil {
		return e
	}
	cmp, e := qm.Compare(lower, upper)
	yes := cmp < 0
	if e != nil {
		return e
	}
	if !yes {
		return ErrBinaryInterval
	}
	aux := &m.SelectSmallTmp
	yes, e = m.selectIntegerRationalBounds(qm, lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	l2k, u2k := NewRational(), NewRational()
	defer qm.drop(&l2k)
	defer qm.drop(&u2k)
	two := Rational{Num: NewInt(2), Den: NewInt(1)}
	if e = qm.Copy(&l2k, lower); e != nil {
		return e
	}
	if e = qm.Copy(&u2k, upper); e != nil {
		return e
	}
	for k := uint32(1); ; k++ {
		if e = qm.Mul(&l2k, &two, &l2k); e != nil {
			return e
		}
		if e = qm.Mul(&u2k, &two, &u2k); e != nil {
			return e
		}
		yes, e = m.selectIntegerRationalBounds(qm, &l2k, &u2k, aux)
		if e != nil {
			return e
		}
		if yes {
			return m.SetIntegerPower(r, aux, k)
		}
	}
}
func (m *BinaryRationalManager) RefineUpper(qm *RationalManager, q *Rational, l, u *BinaryRational) error {
	return m.refine(qm, q, l, u, true)
}
func (m *BinaryRationalManager) RefineLower(qm *RationalManager, q *Rational, l, u *BinaryRational) error {
	return m.refine(qm, q, l, u, false)
}
func (m *BinaryRationalManager) refine(qm *RationalManager, q *Rational, l, u *BinaryRational, upper bool) error {
	if e := m.pair(l, u); e != nil {
		return e
	}
	local := NewRational()
	defer m.dropImported(&local)
	input, e := m.importRational(qm, q, &local)
	if e != nil {
		return e
	}
	lo, e := m.LessRational(l, qm, q)
	if e != nil {
		return e
	}
	if !lo {
		return ErrBinaryInterval
	}
	hi, e := m.GreaterRational(u, qm, q)
	if e != nil {
		return e
	}
	if !lo || !hi {
		return ErrBinaryInterval
	}
	power, _, e := m.Z.IsPowerOfTwo(&input.Den)
	if e != nil {
		return e
	}
	if power {
		return ErrBinaryInterval
	}
	var mid BinaryRational
	cleaned := false
	defer func() {
		if !cleaned {
			m.Del(&mid)
		}
	}()
	for {
		if e = m.Add(l, u, &mid); e != nil {
			return e
		}
		if e = m.Div2(&mid); e != nil {
			return e
		}
		var stop bool
		if upper {
			stop, e = m.GreaterRational(&mid, qm, q)
		} else {
			stop, e = m.LessRational(&mid, qm, q)
		}
		if e != nil {
			return e
		}
		if upper == stop {
			if e = m.Swap(u, &mid); e != nil {
				return e
			}
		} else {
			if e = m.Swap(l, &mid); e != nil {
				return e
			}
		}
		if stop {
			if e = m.Del(&mid); e != nil {
				return e
			}
			cleaned = true
			return m.debugEnclosure(qm, q, l, u)
		}
	}
}
func (m *BinaryRationalManager) Approx(a *BinaryRational, k uint32, plus bool) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.KField <= k {
		return nil
	}
	var old BinaryRational
	if e := m.debugApproxSnapshot(a, &old); e != nil {
		return e
	}
	defer m.debugApproxDrop(&old)
	neg := a.Num.Val < 0
	shift := a.KField - k
	if e := m.Z.Abs(&a.Num); e != nil {
		return e
	}
	if e := m.Z.MachineDiv2k(&a.Num, shift, &a.Num); e != nil {
		return e
	}
	if neg != plus {
		if e := m.Z.Inc(&a.Num); e != nil {
			return e
		}
	}
	if neg {
		if e := m.Z.Neg(&a.Num); e != nil {
			return e
		}
	}
	a.KField = k
	if e := m.normalize(a); e != nil {
		return e
	}
	return m.debugApprox(&old, a, plus)
}
func (m *BinaryRationalManager) ApproxDivDefault(a, b, c *BinaryRational) error {
	return m.ApproxDiv(a, b, c, 32, false)
}
func (m *BinaryRationalManager) ApproxDiv(a, b, c *BinaryRational, k uint32, plus bool) error {
	if e := m.triple(a, b, c); e != nil {
		return e
	}
	if b.Num.Val == 0 {
		return ErrDivideByZero
	}
	power, kp, e := m.Z.IsPowerOfTwo(&b.Num)
	if e != nil {
		return e
	}
	if power {
		if e = m.debugAssert(b.KField == 0 || kp == 0); e != nil {
			return e
		}
		if e = m.Z.Copy(&c.Num, &a.Num); e != nil {
			return e
		}
		if b.KField > 0 {
			if e = m.debugAssert(kp == 0); e != nil {
				return e
			}
			two := NewInt(2)
			if e = m.Z.Power(&two, b.KField, &m.DivTmp1); e != nil {
				return e
			}
			if e = m.Z.Mul(&c.Num, &m.DivTmp1, &c.Num); e != nil {
				return e
			}
		}
		c.KField = a.KField + kp
		return m.normalize(c)
	}
	divides, e := m.Z.Divides(&b.Num, &a.Num)
	if e != nil {
		return e
	}
	if divides {
		if e = m.Z.Div(&a.Num, &b.Num, &c.Num); e != nil {
			return e
		}
		if a.KField >= b.KField {
			c.KField = a.KField - b.KField
		} else {
			if e = m.Z.Mul2k(&c.Num, b.KField-a.KField, &c.Num); e != nil {
				return e
			}
			c.KField = 0
		}
		return m.normalize(c)
	}
	neg := (a.Num.Val < 0) != (b.Num.Val < 0)
	absA, normA, absB := &m.DivTmp1, &m.DivTmp2, &m.DivTmp3
	if e = m.Z.Copy(absA, &a.Num); e != nil {
		return e
	}
	if e = m.Z.Abs(absA); e != nil {
		return e
	}
	if e = m.Z.Copy(absB, &b.Num); e != nil {
		return e
	}
	if e = m.Z.Abs(absB); e != nil {
		return e
	}
	if a.KField > b.KField {
		if k >= a.KField-b.KField {
			e = m.Z.Mul2k(absA, k-(a.KField-b.KField), normA)
		} else {
			e = m.Z.MachineDiv2k(absA, (a.KField-b.KField)-k, normA)
		}
	} else {
		e = m.Z.Mul2k(absA, k+b.KField-a.KField, normA)
	}
	if e != nil {
		return e
	}
	c.KField = k
	if e = m.Z.Div(normA, absB, &c.Num); e != nil {
		return e
	}
	if neg != plus {
		if e = m.Z.Inc(&c.Num); e != nil {
			return e
		}
	}
	if neg {
		if e = m.Z.Neg(&c.Num); e != nil {
			return e
		}
	}
	return m.normalize(c)
}
