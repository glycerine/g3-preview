// Licensed under the MIT license.
// Source: src/util/mpbq.cpp.
package mpnum

func (m *BinaryRationalManager) selectInteger(lower, upper *BinaryRational, r *Int) (bool, error) {
	if lower.k == 0 {
		return true, m.z.Copy(r, &lower.num)
	}
	if upper.k == 0 {
		return true, m.z.Copy(r, &upper.num)
	}
	if e := m.Ceil(lower, &m.selectIntTmp1); e != nil {
		return false, e
	}
	if e := m.Floor(upper, &m.selectIntTmp2); e != nil {
		return false, e
	}
	yes, e := m.z.LessEqual(&m.selectIntTmp1, &m.selectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.z.Copy(r, &m.selectIntTmp1)
}
func (m *BinaryRationalManager) rationalIntegerBound(qm *RationalManager, q *Rational, out *Int, upper bool) error {
	if isIntegerRational(q) {
		if e := m.z.CopyFrom(out, qm.z, &q.num); e != nil {
			return e
		}
		if upper {
			return m.z.Dec(out)
		}
		return m.z.Inc(out)
	}
	var tmp Int
	defer qm.z.Del(&tmp)
	var e error
	if upper {
		e = qm.FloorToInt(q, &tmp)
	} else {
		e = qm.CeilToInt(q, &tmp)
	}
	if e != nil {
		return e
	}
	return m.z.CopyFrom(out, qm.z, &tmp)
}
func (m *BinaryRationalManager) selectIntegerRationalLower(qm *RationalManager, lower *Rational, upper *BinaryRational, r *Int) (bool, error) {
	if upper.k == 0 {
		return true, m.z.Copy(r, &upper.num)
	}
	if e := m.rationalIntegerBound(qm, lower, &m.selectIntTmp1, false); e != nil {
		return false, e
	}
	if e := m.Floor(upper, &m.selectIntTmp2); e != nil {
		return false, e
	}
	yes, e := m.z.LessEqual(&m.selectIntTmp1, &m.selectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.z.Copy(r, &m.selectIntTmp1)
}
func (m *BinaryRationalManager) selectIntegerRationalUpper(qm *RationalManager, lower *BinaryRational, upper *Rational, r *Int) (bool, error) {
	if lower.k == 0 {
		return true, m.z.Copy(r, &lower.num)
	}
	if e := m.Ceil(lower, &m.selectIntTmp1); e != nil {
		return false, e
	}
	if e := m.rationalIntegerBound(qm, upper, &m.selectIntTmp2, true); e != nil {
		return false, e
	}
	yes, e := m.z.LessEqual(&m.selectIntTmp1, &m.selectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.z.Copy(r, &m.selectIntTmp1)
}
func (m *BinaryRationalManager) selectIntegerRationalBounds(qm *RationalManager, lower, upper *Rational, r *Int) (bool, error) {
	if e := m.rationalIntegerBound(qm, lower, &m.selectIntTmp1, false); e != nil {
		return false, e
	}
	if e := m.rationalIntegerBound(qm, upper, &m.selectIntTmp2, true); e != nil {
		return false, e
	}
	yes, e := m.z.LessEqual(&m.selectIntTmp1, &m.selectIntTmp2)
	if e != nil || !yes {
		return false, e
	}
	return true, m.z.Copy(r, &m.selectIntTmp1)
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
	aux := &m.selectSmallTmp
	yes, e = m.selectInteger(lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	minK, maxK := uint32(0), lower.k
	if upper.k < maxK {
		maxK = upper.k
	}
	l2k, u2k := &m.selectSmallTmp1, &m.selectSmallTmp2
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
	aux := &m.selectSmallTmp
	yes, e = m.selectIntegerRationalLower(qm, lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	l2k := NewRational()
	defer qm.drop(&l2k)
	two := Rational{num: NewInt(2), den: NewInt(1)}
	u2k := &m.selectSmallTmp2
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
	aux := &m.selectSmallTmp
	yes, e = m.selectIntegerRationalUpper(qm, lower, upper, aux)
	if e != nil {
		return e
	}
	if yes {
		return m.SetInteger(r, aux)
	}
	u2k := NewRational()
	defer qm.drop(&u2k)
	two := Rational{num: NewInt(2), den: NewInt(1)}
	l2k := &m.selectSmallTmp2
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
	aux := &m.selectSmallTmp
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
	two := Rational{num: NewInt(2), den: NewInt(1)}
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
	power, _, e := m.z.IsPowerOfTwo(&input.den)
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
	if a.k <= k {
		return nil
	}
	var old BinaryRational
	if e := m.debugApproxSnapshot(a, &old); e != nil {
		return e
	}
	defer m.debugApproxDrop(&old)
	neg := a.num.val < 0
	shift := a.k - k
	if e := m.z.Abs(&a.num); e != nil {
		return e
	}
	if e := m.z.MachineDiv2k(&a.num, shift, &a.num); e != nil {
		return e
	}
	if neg != plus {
		if e := m.z.Inc(&a.num); e != nil {
			return e
		}
	}
	if neg {
		if e := m.z.Neg(&a.num); e != nil {
			return e
		}
	}
	a.k = k
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
	if b.num.val == 0 {
		return ErrDivideByZero
	}
	power, kp, e := m.z.IsPowerOfTwo(&b.num)
	if e != nil {
		return e
	}
	if power {
		if e = m.debugAssert(b.k == 0 || kp == 0); e != nil {
			return e
		}
		if e = m.z.Copy(&c.num, &a.num); e != nil {
			return e
		}
		if b.k > 0 {
			if e = m.debugAssert(kp == 0); e != nil {
				return e
			}
			two := NewInt(2)
			if e = m.z.Power(&two, b.k, &m.divTmp1); e != nil {
				return e
			}
			if e = m.z.Mul(&c.num, &m.divTmp1, &c.num); e != nil {
				return e
			}
		}
		c.k = a.k + kp
		return m.normalize(c)
	}
	divides, e := m.z.Divides(&b.num, &a.num)
	if e != nil {
		return e
	}
	if divides {
		if e = m.z.Div(&a.num, &b.num, &c.num); e != nil {
			return e
		}
		if a.k >= b.k {
			c.k = a.k - b.k
		} else {
			if e = m.z.Mul2k(&c.num, b.k-a.k, &c.num); e != nil {
				return e
			}
			c.k = 0
		}
		return m.normalize(c)
	}
	neg := (a.num.val < 0) != (b.num.val < 0)
	absA, normA, absB := &m.divTmp1, &m.divTmp2, &m.divTmp3
	if e = m.z.Copy(absA, &a.num); e != nil {
		return e
	}
	if e = m.z.Abs(absA); e != nil {
		return e
	}
	if e = m.z.Copy(absB, &b.num); e != nil {
		return e
	}
	if e = m.z.Abs(absB); e != nil {
		return e
	}
	if a.k > b.k {
		if k >= a.k-b.k {
			e = m.z.Mul2k(absA, k-(a.k-b.k), normA)
		} else {
			e = m.z.MachineDiv2k(absA, (a.k-b.k)-k, normA)
		}
	} else {
		e = m.z.Mul2k(absA, k+b.k-a.k, normA)
	}
	if e != nil {
		return e
	}
	c.k = k
	if e = m.z.Div(normA, absB, &c.num); e != nil {
		return e
	}
	if neg != plus {
		if e = m.z.Inc(&c.num); e != nil {
			return e
		}
	}
	if neg {
		if e = m.z.Neg(&c.num); e != nil {
			return e
		}
	}
	return m.normalize(c)
}
