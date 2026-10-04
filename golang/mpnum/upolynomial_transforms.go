// Licensed under the MIT license.
package mpnum

// Concrete source overload dispatch: borrowed numerical parameters do not
// escape the call, and their fields are reread at the actual source sites.
func (m *UPolynomialManager) translate(v UPolynomialView, kind uint8, k uint32, z *Int, b *BinaryRational, q *Rational) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	if kind == 3 {
		if b == nil {
			return ErrUPolynomialOwner
		}
		if e := m.ComposePowerTwoDegreeDivideXViewInPlace(v, b.k); e != nil {
			return e
		}
	}
	if kind == 4 {
		if q == nil {
			return ErrUPolynomialOwner
		}
		if e := m.ComposeIntegerDegreeDivideXViewInPlace(v, &q.den); e != nil {
			return e
		}
	}
	var aux Int
	if kind == 1 {
		defer m.modular.Del(&aux)
	}
	n := v.count - 1
	for i := uint32(1); i <= n; i++ {
		if e := m.checkpoint(); e != nil {
			return e
		}
		for index := n - i; index <= n-1; index++ {
			a, e := m.read(v, index)
			if e != nil {
				return e
			}
			next, e := m.read(v, index+1)
			if e != nil {
				return e
			}
			switch kind {
			case 0:
				e = m.modular.Add(a, next, a)
			case 1:
				// The C++ inner unsigned k shadows the shift argument.
				e = m.modular.Mul2k(next, index, &aux)
				if e == nil {
					e = m.modular.Add(a, &aux, a)
				}
			case 2:
				e = m.modular.AddMul(a, z, next, a)
			case 3:
				if index > n-i {
					e = m.modular.Mul2kInPlace(a, b.k)
				}
				if e == nil {
					e = m.modular.AddMul(a, &b.num, next, a)
				}
			case 4:
				if index > n-i {
					e = m.modular.Mul(a, &q.den, a)
				}
				if e == nil {
					e = m.modular.AddMul(a, &q.num, next, a)
				}
			}
			if e != nil {
				return e
			}
		}
		if kind == 3 || kind == 4 {
			top, e := m.read(v, n)
			if e != nil {
				return e
			}
			if kind == 3 {
				e = m.modular.Mul2kInPlace(top, b.k)
			} else {
				e = m.modular.Mul(top, &q.den, top)
			}
			if e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UPolynomialManager) TranslateOneViewInPlace(v UPolynomialView) error {
	return m.translate(v, 0, 0, nil, nil, nil)
}
func (m *UPolynomialManager) TranslatePowerTwoViewInPlace(v UPolynomialView, k uint32) error {
	return m.translate(v, 1, k, nil, nil, nil)
}
func (m *UPolynomialManager) TranslateIntegerViewInPlace(v UPolynomialView, c *Int) error {
	return m.translate(v, 2, 0, c, nil, nil)
}
func (m *UPolynomialManager) TranslateBinaryViewInPlace(v UPolynomialView, b *BinaryRational) error {
	return m.translate(v, 3, 0, nil, b, nil)
}
func (m *UPolynomialManager) TranslateRationalViewInPlace(v UPolynomialView, b *Rational) error {
	return m.translate(v, 4, 0, nil, nil, b)
}
func (m *UPolynomialManager) translated(v UPolynomialView, out UPolynomial, kind uint8, k uint32, z *Int, b *BinaryRational, q *Rational) error {
	d, e := m.descriptor(out)
	if e != nil {
		return e
	}
	if e = m.setView(&d.state, v); e != nil {
		return e
	}
	return m.translate(m.stateView(&d.state).WithCount(v.count), kind, k, z, b, q)
}
func (m *UPolynomialManager) TranslateOne(v UPolynomialView, out UPolynomial) error {
	return m.translated(v, out, 0, 0, nil, nil, nil)
}
func (m *UPolynomialManager) TranslatePowerTwo(v UPolynomialView, k uint32, out UPolynomial) error {
	return m.translated(v, out, 1, k, nil, nil, nil)
}
func (m *UPolynomialManager) TranslateInteger(v UPolynomialView, c *Int, out UPolynomial) error {
	return m.translated(v, out, 2, 0, c, nil, nil)
}
func (m *UPolynomialManager) TranslateBinary(v UPolynomialView, b *BinaryRational, out UPolynomial) error {
	return m.translated(v, out, 3, 0, nil, b, nil)
}
func (m *UPolynomialManager) TranslateRational(v UPolynomialView, b *Rational, out UPolynomial) error {
	return m.translated(v, out, 4, 0, nil, nil, b)
}
func (m *UPolynomialManager) ComposeTwoDegreeDivideXViewInPlace(v UPolynomialView) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	k := v.count - 1
	for i := uint32(0); i < v.count-1; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.Mul2kInPlace(a, k); e != nil {
			return e
		}
		k--
	}
	return nil
}
func (m *UPolynomialManager) ComposePowerTwoDegreeDivideXViewInPlace(v UPolynomialView, k uint32) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	ki := k * v.count
	for i := uint32(0); i < v.count; i++ {
		ki -= k
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if !zero {
			if e = m.modular.Mul2kInPlace(a, ki); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UPolynomialManager) ComposePowerTwoXViewInPlace(v UPolynomialView, k uint32) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	ki := k
	for i := uint32(1); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.Mul2kInPlace(a, ki); e != nil {
			return e
		}
		ki += k
	}
	return nil
}
func (m *UPolynomialManager) ComposeIntegerDegreeDivideXViewInPlace(v UPolynomialView, a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	var ai Int
	defer m.modular.Del(&ai)
	if e := m.modular.Copy(&ai, a); e != nil {
		return e
	}
	for i := v.count - 1; i > 0; {
		i--
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(x)
		if e != nil {
			return e
		}
		if !zero {
			if e = m.modular.Mul(x, &ai, x); e != nil {
				return e
			}
		}
		if e = m.modular.Mul(&ai, a, &ai); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) ComposeIntegerXViewInPlace(v UPolynomialView, b *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	var bi Int
	defer m.modular.Del(&bi)
	if e := m.modular.Copy(&bi, b); e != nil {
		return e
	}
	for i := uint32(1); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if !zero {
			if e = m.modular.Mul(a, &bi, a); e != nil {
				return e
			}
		}
		if e = m.modular.Mul(&bi, b, &bi); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) ComposeBinaryXViewInPlace(v UPolynomialView, b *BinaryRational) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	if b == nil {
		return ErrUPolynomialOwner
	}
	var ci Int
	defer m.modular.Del(&ci)
	if e := m.modular.SetInt32(&ci, 1); e != nil {
		return e
	}
	k := b.k
	ki := k * v.count
	for i := uint32(0); i < v.count; i++ {
		ki -= k
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if !zero {
			if e = m.modular.Mul2kInPlace(a, ki); e != nil {
				return e
			}
			if e = m.modular.Mul(a, &ci, a); e != nil {
				return e
			}
		}
		if e = m.modular.Mul(&ci, &b.num, &ci); e != nil {
			return e
		}
	}
	if modularSourceDebug && ki != 0 {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (m *UPolynomialManager) ComposeRationalXViewInPlace(v UPolynomialView, q *Rational) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	if q == nil {
		return ErrUPolynomialOwner
	}
	var bc Int
	defer m.modular.Del(&bc)
	if e := m.modular.Power(&q.den, v.count-1, &bc); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if !zero {
			if e = m.modular.Mul(a, &bc, a); e != nil {
				return e
			}
		}
		if i < v.count-1 {
			if e = m.modular.Div(&bc, &q.den, &bc); e != nil {
				return e
			}
			if e = m.modular.Mul(&bc, &q.num, &bc); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *UPolynomialManager) PMinusXViewInPlace(v UPolynomialView) error {
	if e := m.check(); e != nil {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return e
		}
		if zero || i%2 == 0 {
			continue
		}
		if e = m.modular.Neg(a); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) ReciprocalViewInPlace(v UPolynomialView) error {
	if e := m.check(); e != nil {
		return e
	}
	if v.count <= 1 {
		return nil
	}
	i, j := uint32(0), v.count-1
	for {
		a, e := m.read(v, i)
		if e != nil {
			return e
		}
		b, e := m.read(v, j)
		if e != nil {
			return e
		}
		*a, *b = *b, *a
		i++
		j--
		if i >= j {
			return nil
		}
	}
}
