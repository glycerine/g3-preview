// Source: src/util/mpbq.h:116.
// Source: src/util/mpbq.h/.cpp.
package numeral

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
	x, y := &a.Num, &b.Num
	k := a.KField
	if a.KField < b.KField {
		if e = m.Z.Mul2k(&a.Num, b.KField-a.KField, &m.Tmp); e != nil {
			return e
		}
		k = b.KField
		if sub {
			x, y = &m.Tmp, &b.Num
		} else {
			x, y = &b.Num, &m.Tmp
		}
	}
	if a.KField > b.KField {
		if e = m.Z.Mul2k(&b.Num, a.KField-b.KField, &m.Tmp); e != nil {
			return e
		}
		x, y = &a.Num, &m.Tmp
	}
	if sub {
		e = m.Z.Sub(x, y, &r.Num)
	} else {
		e = m.Z.Add(x, y, &r.Num)
	}
	if e != nil {
		return e
	}
	r.KField = k
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
	if e := m.Z.check(b); e != nil {
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
	if a.KField != 0 {
		if e = m.Z.Mul2k(b, a.KField, &m.Tmp); e != nil {
			return e
		}
		y = &m.Tmp
	}
	if sub {
		e = m.Z.Sub(&a.Num, y, &r.Num)
	} else {
		e = m.Z.Add(&a.Num, y, &r.Num)
	}
	if e != nil {
		return e
	}
	r.KField = a.KField
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
	if e = m.Z.Mul(&a.Num, &b.Num, &r.Num); e != nil {
		return e
	}
	r.KField = a.KField + b.KField
	// Source rereads after publication, including output-operand aliases.
	if a.KField == 0 || b.KField == 0 {
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
	if e := m.Z.check(b); e != nil {
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
	if e = m.Z.Mul(&a.Num, b, &r.Num); e != nil {
		return e
	}
	r.KField = a.KField
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
	if e := m.Mul(b, c, &m.AddMulTmp); e != nil {
		return e
	}
	return m.Add(a, &m.AddMulTmp, r)
}
func (m *BinaryRationalManager) AddMulInteger(a *BinaryRational, b *Int, c, r *BinaryRational) error {
	if e := m.triple(a, c, r); e != nil {
		return e
	}
	if e := m.Z.check(b); e != nil {
		return e
	}
	if e := m.MulInteger(c, b, &m.AddMulTmp); e != nil {
		return e
	}
	return m.Add(a, &m.AddMulTmp, r)
}
func (m *BinaryRationalManager) Neg(a *BinaryRational) error {
	return m.negWithSourceContext(a, integerAllocationInvocationContext{})
}
func (m *BinaryRationalManager) negWithSourceContext(a *BinaryRational, context integerAllocationInvocationContext) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.Z.negWithAllocationContext(&a.Num, context)
}
func (m *BinaryRationalManager) Div2(a *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	old := a.KField == 0
	a.KField++
	if old {
		return m.normalize(a)
	}
	return nil
}
func (m *BinaryRationalManager) Div2k(a *BinaryRational, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	old := a.KField == 0
	a.KField += k
	if old {
		return m.normalize(a)
	}
	return nil
}
func (m *BinaryRationalManager) Mul2(a *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if a.KField != 0 {
		a.KField--
		return nil
	}
	return m.Z.Mul2k(&a.Num, 1, &a.Num)
}
func (m *BinaryRationalManager) Mul2k(a *BinaryRational, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if k == 0 {
		return nil
	}
	if a.KField < k {
		if e := m.Z.Mul2k(&a.Num, k-a.KField, &a.Num); e != nil {
			return e
		}
		a.KField = 0
	} else {
		a.KField -= k
	}
	return nil
}
func (m *BinaryRationalManager) PowerInPlace(a *BinaryRational, p uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.debugAssert(uint64(a.KField)*uint64(p) <= uint64(^uint32(0))); e != nil {
		return e
	}
	a.KField *= p
	return m.Z.Power(&a.Num, p, &a.Num)
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
	if a.KField != b.KField {
		return false, nil
	}
	return m.Z.Equal(&a.Num, &b.Num)
}
func (m *BinaryRationalManager) NotEqual(a, b *BinaryRational) (bool, error) {
	v, e := m.Equal(a, b)
	return !v, e
}
func (m *BinaryRationalManager) Less(a, b *BinaryRational) (bool, error) {
	if e := m.pair(a, b); e != nil {
		return false, e
	}
	if a.KField == b.KField {
		return m.Z.Less(&a.Num, &b.Num)
	}
	if a.KField < b.KField {
		if e := m.Z.Mul2k(&a.Num, b.KField-a.KField, &m.Tmp); e != nil {
			return false, e
		}
		return m.Z.Less(&m.Tmp, &b.Num)
	}
	if e := m.Z.Mul2k(&b.Num, a.KField-b.KField, &m.Tmp); e != nil {
		return false, e
	}
	return m.Z.Less(&a.Num, &m.Tmp)
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
	v, e := m.Z.Equal(&a.Num, b)
	return v && a.KField == 0, e
}
func (m *BinaryRationalManager) NotEqualInteger(a *BinaryRational, b *Int) (bool, error) {
	v, e := m.EqualInteger(a, b)
	return !v, e
}
func (m *BinaryRationalManager) compareInteger(a *BinaryRational, b *Int, le bool) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.Z.check(b); e != nil {
		return false, e
	}
	y := b
	if a.KField != 0 {
		if e := m.Z.Mul2k(b, a.KField, &m.Tmp); e != nil {
			return false, e
		}
		y = &m.Tmp
	}
	if le {
		return m.Z.LessEqual(&a.Num, y)
	}
	return m.Z.Less(&a.Num, y)
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
	x, y := &a.Num, &q.Num
	if a.KField != 0 || !isOneInteger(&q.Den) {
		if op == 0 {
			if e = m.Z.Mul2k(&q.Num, a.KField, &m.Tmp); e != nil {
				return false, e
			}
			if e = m.Z.Mul(&a.Num, &q.Den, &m.Tmp2); e != nil {
				return false, e
			}
			x, y = &m.Tmp, &m.Tmp2
		} else {
			if e = m.Z.Mul(&a.Num, &q.Den, &m.Tmp); e != nil {
				return false, e
			}
			if e = m.Z.Mul2k(&q.Num, a.KField, &m.Tmp2); e != nil {
				return false, e
			}
			x, y = &m.Tmp, &m.Tmp2
		}
	}
	if op == 0 {
		return m.Z.Equal(x, y)
	}
	if op == 1 {
		return m.Z.Less(x, y)
	}
	return m.Z.LessEqual(x, y)
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
	if a.Num.Val == 0 {
		return 0, nil
	}
	if a.Num.Val > 0 {
		v, e := m.Z.Log2(&a.Num)
		return int32(v - a.KField), e
	}
	v, e := m.Z.MLog2(&a.Num)
	return int32(v - a.KField + 1), e
}
func (m *BinaryRationalManager) MagnitudeUpper(a *BinaryRational) (int32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if a.Num.Val == 0 {
		return 0, nil
	}
	if a.Num.Val > 0 {
		v, e := m.Z.Log2(&a.Num)
		return int32(v - a.KField + 1), e
	}
	v, e := m.Z.MLog2(&a.Num)
	return int32(v - a.KField), e
}
func (m *BinaryRationalManager) LessOneDiv2k(a *BinaryRational, k uint32) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.Num.Val <= 0 {
		return true, nil
	}
	if a.KField <= k {
		return false, nil
	}
	one := NewInt(1)
	if e := m.Z.Mul2k(&one, a.KField-k, &m.Tmp); e != nil {
		return false, e
	}
	return m.Z.Less(&a.Num, &m.Tmp)
}
