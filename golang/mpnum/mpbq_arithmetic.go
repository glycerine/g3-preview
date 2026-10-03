// Copyright (c) 2011 Microsoft Corporation. Source: src/util/mpbq.h/.cpp.
package mpnum

func (m *BinaryRationalManager) Add(a, b, r *BinaryRational) error { return m.addSub(a, b, r, false) }
func (m *BinaryRationalManager) Sub(a, b, r *BinaryRational) error { return m.addSub(a, b, r, true) }
func (m *BinaryRationalManager) addSub(a, b, r *BinaryRational, sub bool) error {
	if e := m.triple(a, b, r); e != nil {
		return e
	}
	da, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&da)
	db, e := m.debugValue(b)
	if e != nil {
		return e
	}
	defer m.debugDel(&db)
	x, y := &a.num, &b.num
	k := a.k
	if a.k < b.k {
		if e = m.z.Mul2k(&a.num, b.k-a.k, &m.tmp); e != nil {
			return e
		}
		k = b.k
		if sub {
			x, y = &m.tmp, &b.num
		} else {
			x, y = &b.num, &m.tmp
		}
	}
	if a.k > b.k {
		if e = m.z.Mul2k(&b.num, a.k-b.k, &m.tmp); e != nil {
			return e
		}
		x, y = &a.num, &m.tmp
	}
	if sub {
		e = m.z.Sub(x, y, &r.num)
	} else {
		e = m.z.Add(x, y, &r.num)
	}
	if e != nil {
		return e
	}
	r.k = k
	if e = m.normalize(r); e != nil {
		return e
	}
	op := uint8(0)
	if sub {
		op = 1
	}
	return m.debugArithmetic(&da, &db, r, op)
}
func (m *BinaryRationalManager) AddInteger(a *BinaryRational, b *Int, r *BinaryRational) error {
	return m.addSubInteger(a, b, r, false)
}
func (m *BinaryRationalManager) SubInteger(a *BinaryRational, b *Int, r *BinaryRational) error {
	return m.addSubInteger(a, b, r, true)
}
func (m *BinaryRationalManager) addSubInteger(a *BinaryRational, b *Int, r *BinaryRational, sub bool) error {
	if e := m.pair(a, r); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	da, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&da)
	db, e := m.debugInteger(b)
	if e != nil {
		return e
	}
	defer m.debugDel(&db)
	y := b
	if a.k != 0 {
		if e = m.z.Mul2k(b, a.k, &m.tmp); e != nil {
			return e
		}
		y = &m.tmp
	}
	if sub {
		e = m.z.Sub(&a.num, y, &r.num)
	} else {
		e = m.z.Add(&a.num, y, &r.num)
	}
	if e != nil {
		return e
	}
	r.k = a.k
	if e = m.normalize(r); e != nil {
		return e
	}
	op := uint8(0)
	if sub {
		op = 1
	}
	return m.debugArithmetic(&da, &db, r, op)
}
func (m *BinaryRationalManager) Mul(a, b, r *BinaryRational) error {
	if e := m.triple(a, b, r); e != nil {
		return e
	}
	da, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&da)
	db, e := m.debugValue(b)
	if e != nil {
		return e
	}
	defer m.debugDel(&db)
	if e = m.z.Mul(&a.num, &b.num, &r.num); e != nil {
		return e
	}
	r.k = a.k + b.k
	// Source rereads after publication, including output-operand aliases.
	if a.k == 0 || b.k == 0 {
		if e = m.normalize(r); e != nil {
			return e
		}
	}
	return m.debugArithmetic(&da, &db, r, 2)
}
func (m *BinaryRationalManager) MulInteger(a *BinaryRational, b *Int, r *BinaryRational) error {
	if e := m.pair(a, r); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	da, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&da)
	db, e := m.debugInteger(b)
	if e != nil {
		return e
	}
	defer m.debugDel(&db)
	if e = m.z.Mul(&a.num, b, &r.num); e != nil {
		return e
	}
	r.k = a.k
	if e = m.normalize(r); e != nil {
		return e
	}
	return m.debugArithmetic(&da, &db, r, 2)
}
func (m *BinaryRationalManager) AddMul(a, b, c, r *BinaryRational) error {
	if e := m.triple(a, b, c); e != nil {
		return e
	}
	if e := m.check(r); e != nil {
		return e
	}
	if e := m.Mul(b, c, &m.addMulTmp); e != nil {
		return e
	}
	return m.Add(a, &m.addMulTmp, r)
}
func (m *BinaryRationalManager) AddMulInteger(a *BinaryRational, b *Int, c, r *BinaryRational) error {
	if e := m.triple(a, c, r); e != nil {
		return e
	}
	if e := m.z.check(b); e != nil {
		return e
	}
	if e := m.MulInteger(c, b, &m.addMulTmp); e != nil {
		return e
	}
	return m.Add(a, &m.addMulTmp, r)
}
func (m *BinaryRationalManager) Neg(a *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Neg(&a.num)
}
func (m *BinaryRationalManager) Div2(a *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	old := a.k == 0
	a.k++
	if old {
		return m.normalize(a)
	}
	return nil
}
func (m *BinaryRationalManager) Div2k(a *BinaryRational, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	old := a.k == 0
	a.k += k
	if old {
		return m.normalize(a)
	}
	return nil
}
func (m *BinaryRationalManager) Mul2(a *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.k != 0 {
		a.k--
		return nil
	}
	return m.z.Mul2k(&a.num, 1, &a.num)
}
func (m *BinaryRationalManager) Mul2k(a *BinaryRational, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if k == 0 {
		return nil
	}
	if a.k < k {
		if e := m.z.Mul2k(&a.num, k-a.k, &a.num); e != nil {
			return e
		}
		a.k = 0
	} else {
		a.k -= k
	}
	return nil
}
func (m *BinaryRationalManager) PowerInPlace(a *BinaryRational, p uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.debugAssert(uint64(a.k)*uint64(p) <= uint64(^uint32(0))); e != nil {
		return e
	}
	a.k *= p
	return m.z.Power(&a.num, p, &a.num)
}
func (m *BinaryRationalManager) Power(a *BinaryRational, p uint32, out *BinaryRational) error {
	if e := m.Copy(out, a); e != nil {
		return e
	}
	return m.PowerInPlace(out, p)
}
func (m *BinaryRationalManager) Equal(a, b *BinaryRational) (bool, error) {
	if e := m.pair(a, b); e != nil {
		return false, e
	}
	if a.k != b.k {
		return false, nil
	}
	return m.z.Equal(&a.num, &b.num)
}
func (m *BinaryRationalManager) NotEqual(a, b *BinaryRational) (bool, error) {
	v, e := m.Equal(a, b)
	return !v, e
}
func (m *BinaryRationalManager) Less(a, b *BinaryRational) (bool, error) {
	if e := m.pair(a, b); e != nil {
		return false, e
	}
	if a.k == b.k {
		return m.z.Less(&a.num, &b.num)
	}
	if a.k < b.k {
		if e := m.z.Mul2k(&a.num, b.k-a.k, &m.tmp); e != nil {
			return false, e
		}
		return m.z.Less(&m.tmp, &b.num)
	}
	if e := m.z.Mul2k(&b.num, a.k-b.k, &m.tmp); e != nil {
		return false, e
	}
	return m.z.Less(&a.num, &m.tmp)
}
func (m *BinaryRationalManager) Greater(a, b *BinaryRational) (bool, error) { return m.Less(b, a) }
func (m *BinaryRationalManager) LessEqual(a, b *BinaryRational) (bool, error) {
	v, e := m.Greater(a, b)
	return !v, e
}
func (m *BinaryRationalManager) GreaterEqual(a, b *BinaryRational) (bool, error) {
	return m.LessEqual(b, a)
}
func (m *BinaryRationalManager) EqualInteger(a *BinaryRational, b *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	v, e := m.z.Equal(&a.num, b)
	return v && a.k == 0, e
}
func (m *BinaryRationalManager) NotEqualInteger(a *BinaryRational, b *Int) (bool, error) {
	v, e := m.EqualInteger(a, b)
	return !v, e
}
func (m *BinaryRationalManager) compareInteger(a *BinaryRational, b *Int, le bool) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.z.check(b); e != nil {
		return false, e
	}
	y := b
	if a.k != 0 {
		if e := m.z.Mul2k(b, a.k, &m.tmp); e != nil {
			return false, e
		}
		y = &m.tmp
	}
	if le {
		return m.z.LessEqual(&a.num, y)
	}
	return m.z.Less(&a.num, y)
}
func (m *BinaryRationalManager) LessInteger(a *BinaryRational, b *Int) (bool, error) {
	return m.compareInteger(a, b, false)
}
func (m *BinaryRationalManager) LessEqualInteger(a *BinaryRational, b *Int) (bool, error) {
	return m.compareInteger(a, b, true)
}
func (m *BinaryRationalManager) GreaterInteger(a *BinaryRational, b *Int) (bool, error) {
	v, e := m.LessEqualInteger(a, b)
	return !v, e
}
func (m *BinaryRationalManager) GreaterEqualInteger(a *BinaryRational, b *Int) (bool, error) {
	v, e := m.LessInteger(a, b)
	return !v, e
}
func (m *BinaryRationalManager) compareRational(a *BinaryRational, qm *RationalManager, b *Rational, op uint8) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	local := NewRational()
	defer m.dropImported(&local)
	q, e := m.importRational(qm, b, &local)
	if e != nil {
		return false, e
	}
	x, y := &a.num, &q.num
	if a.k != 0 || !isOneInteger(&q.den) {
		if op == 0 {
			if e = m.z.Mul2k(&q.num, a.k, &m.tmp); e != nil {
				return false, e
			}
			if e = m.z.Mul(&a.num, &q.den, &m.tmp2); e != nil {
				return false, e
			}
			x, y = &m.tmp, &m.tmp2
		} else {
			if e = m.z.Mul(&a.num, &q.den, &m.tmp); e != nil {
				return false, e
			}
			if e = m.z.Mul2k(&q.num, a.k, &m.tmp2); e != nil {
				return false, e
			}
			x, y = &m.tmp, &m.tmp2
		}
	}
	if op == 0 {
		return m.z.Equal(x, y)
	}
	if op == 1 {
		return m.z.Less(x, y)
	}
	return m.z.LessEqual(x, y)
}
func (m *BinaryRationalManager) EqualRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	return m.compareRational(a, qm, b, 0)
}
func (m *BinaryRationalManager) NotEqualRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	v, e := m.EqualRational(a, qm, b)
	return !v, e
}
func (m *BinaryRationalManager) LessRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	return m.compareRational(a, qm, b, 1)
}
func (m *BinaryRationalManager) LessEqualRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	return m.compareRational(a, qm, b, 2)
}
func (m *BinaryRationalManager) GreaterRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	v, e := m.LessEqualRational(a, qm, b)
	return !v, e
}
func (m *BinaryRationalManager) GreaterEqualRational(a *BinaryRational, qm *RationalManager, b *Rational) (bool, error) {
	v, e := m.LessRational(a, qm, b)
	return !v, e
}
func (m *BinaryRationalManager) MagnitudeLower(a *BinaryRational) (int32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.num.val == 0 {
		return 0, nil
	}
	if a.num.val > 0 {
		v, e := m.z.Log2(&a.num)
		return int32(v - a.k), e
	}
	v, e := m.z.MLog2(&a.num)
	return int32(v - a.k + 1), e
}
func (m *BinaryRationalManager) MagnitudeUpper(a *BinaryRational) (int32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.num.val == 0 {
		return 0, nil
	}
	if a.num.val > 0 {
		v, e := m.z.Log2(&a.num)
		return int32(v - a.k + 1), e
	}
	v, e := m.z.MLog2(&a.num)
	return int32(v - a.k), e
}
func (m *BinaryRationalManager) LessOneDiv2k(a *BinaryRational, k uint32) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.num.val <= 0 {
		return true, nil
	}
	if a.k <= k {
		return false, nil
	}
	one := NewInt(1)
	if e := m.z.Mul2k(&one, a.k-k, &m.tmp); e != nil {
		return false, e
	}
	return m.z.Less(&a.num, &m.tmp)
}
