// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Source core_manager polynomial division; outputs keep completed prefixes.
package mpnum

import "github.com/glycerine/g3/golang/arena"

func (m *UPolynomialManager) divScalar(v UPolynomialView, b *Int) error {
	if modularSourceDebug {
		zero, e := m.modular.IsZero(b)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	one, e := m.modular.IsOne(b)
	if e != nil || one {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		if modularSourceDebug {
			ok, e := m.modular.Divides(b, x)
			if e != nil {
				return e
			}
			if !ok {
				return ErrUPolynomialAssertion
			}
		}
		if e = m.modular.Div(x, b, x); e != nil {
			return e
		}
	}
	return nil
}
func (m *UPolynomialManager) DivScalar(p UPolynomial, b *Int) error {
	v, e := m.wholeView(p)
	if e != nil {
		return e
	}
	return m.divScalar(v, b)
}
func (m *UPolynomialManager) DivScalarView(v UPolynomialView, b *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.divScalar(v, b)
}

// Concrete borrowed divisor representation; immediate is used only for the
// actual source lexical mpz[-1,2]. Pointer receivers borrow that caller frame,
// never an address of a by-value numeral copy or a fake directory binding.
type upolyDivisorSpan struct {
	view      UPolynomialView
	immediate bool
	pair      [2]Int
}

func (b *upolyDivisorSpan) count() uint32 {
	if b.immediate {
		return 2
	}
	return b.view.count
}
func (m *UPolynomialManager) readDivisor(b *upolyDivisorSpan, i uint32) (*Int, error) {
	if b.immediate {
		if i >= 2 {
			return nil, arena.ErrBounds
		}
		return &b.pair[i], nil
	}
	return m.read(b.view, i)
}
func (m *UPolynomialManager) aliasDivisor(b *upolyDivisorSpan, s *upolyVectorState) bool {
	return !b.immediate && m.alias(b.view, s)
}
func (m *UPolynomialManager) divRemCore(a, b UPolynomialView, d *uint32, q, r *upolyVectorState) error {
	span := upolyDivisorSpan{view: b}
	return m.divRemCoreSpan(a, &span, d, q, r)
}
func (m *UPolynomialManager) divRemCoreSpan(a UPolynomialView, b *upolyDivisorSpan, d *uint32, q, r *upolyVectorState) error {
	if d == nil {
		return arena.ErrBounds
	}
	if modularSourceDebug && (m.alias(a, q) || m.aliasDivisor(b, q) || m.alias(a, r) || m.aliasDivisor(b, r)) {
		return ErrUPolynomialAssertion
	}
	*d = 0
	if modularSourceDebug && b.count() == 0 {
		return ErrUPolynomialAssertion
	}
	if b.count() == 1 {
		if e := m.setView(q, a); e != nil {
			return e
		}
		field, e := m.modular.Field()
		if e != nil {
			return e
		}
		if field {
			bn, e := m.readDivisor(b, 0)
			if e != nil {
				return e
			}
			if e = m.divScalar(m.stateView(q), bn); e != nil {
				return e
			}
		}
		return m.reset(r)
	}
	if e := m.reset(q); e != nil {
		return e
	}
	if e := m.setView(r, a); e != nil {
		return e
	}
	if a.count <= 1 {
		return nil
	}
	var qsz uint32
	if a.count >= b.count() {
		qsz = a.count - b.count() + 1
		if e := m.reserve(q, qsz); e != nil {
			return e
		}
	}
	bn, e := m.readDivisor(b, b.count()-1)
	if e != nil {
		return e
	}
	if modularSourceDebug {
		zero, e := m.modular.IsZero(bn)
		if e != nil {
			return e
		}
		if zero {
			return ErrUPolynomialAssertion
		}
	}
	var am Int
	// Source scoped_numeral destructor; cleanup on Go errors is the safe
	// exception adapter. The counter and caller/scratch prefixes stay published.
	defer m.modular.Del(&am)
	for {
		if e = m.checkpoint(); e != nil {
			return e
		}
		n := r.size
		if n < b.count() {
			return m.setSize(q, qsz)
		}
		delta := n - b.count()
		field, e := m.modular.Field()
		if e != nil {
			return e
		}
		bn, e = m.readDivisor(b, b.count()-1)
		if e != nil {
			return e
		}
		if field {
			if e = m.modular.Div(&m.values(r)[n-1], bn, &am); e != nil {
				return e
			}
			if e = m.modular.Add(&m.values(q)[delta], &am, &m.values(q)[delta]); e != nil {
				return e
			}
		} else {
			*d++
			if e = m.modular.Copy(&am, &m.values(r)[n-1]); e != nil {
				return e
			}
			for i := uint32(0); i < n-1; i++ {
				bn, e = m.readDivisor(b, b.count()-1)
				if e != nil {
					return e
				}
				if e = m.modular.Mul(&m.values(r)[i], bn, &m.values(r)[i]); e != nil {
					return e
				}
			}
			for i := uint32(0); i < qsz; i++ {
				bn, e = m.readDivisor(b, b.count()-1)
				if e != nil {
					return e
				}
				if e = m.modular.Mul(&m.values(q)[i], bn, &m.values(q)[i]); e != nil {
					return e
				}
			}
			if e = m.modular.Add(&m.values(q)[delta], &am, &m.values(q)[delta]); e != nil {
				return e
			}
		}
		for i := uint32(0); i < b.count()-1; i++ {
			x, e := m.readDivisor(b, i)
			if e != nil {
				return e
			}
			dst := &m.values(r)[i+delta]
			if e = m.modular.SubMul(dst, &am, x, dst); e != nil {
				return e
			}
		}
		if e = m.setSize(r, n-1); e != nil {
			return e
		}
	}
}
func (m *UPolynomialManager) DivRemViews(a, b UPolynomialView, q, r UPolynomial, d *uint32) error {
	qo, e := m.descriptor(q)
	if e != nil {
		return e
	}
	ro, e := m.descriptor(r)
	if e != nil {
		return e
	}
	if e = m.divRemCore(a, b, d, &m.div2, &m.div1); e != nil {
		return e
	}
	ro.state, m.div1 = m.div1, ro.state
	qo.state, m.div2 = m.div2, qo.state
	return nil
}
func (m *UPolynomialManager) DivRem(a, b, q, r UPolynomial, d *uint32) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.DivRemViews(av, bv, q, r, d)
}
func (m *UPolynomialManager) divState(a, b UPolynomialView, q *upolyVectorState) error {
	span := upolyDivisorSpan{view: b}
	return m.divSpanState(a, &span, q)
}
func (m *UPolynomialManager) divSpanState(a UPolynomialView, b *upolyDivisorSpan, q *upolyVectorState) error {
	var d uint32
	if e := m.divRemCoreSpan(a, b, &d, &m.div2, &m.div1); e != nil {
		return e
	}
	if e := m.reset(&m.div1); e != nil {
		return e
	}
	*q, m.div2 = m.div2, *q
	return nil
}
func (m *UPolynomialManager) DivViews(a, b UPolynomialView, q UPolynomial) error {
	out, e := m.descriptor(q)
	if e != nil {
		return e
	}
	return m.divState(a, b, &out.state)
}
func (m *UPolynomialManager) Div(a, b, q UPolynomial) error {
	av, e := m.wholeView(a)
	if e != nil {
		return e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return e
	}
	return m.DivViews(av, bv, q)
}

func (m *UPolynomialManager) dividesViews(a, b UPolynomialView) (bool, error) {
	if b.count == 0 {
		return false, nil
	}
	if a.count == 0 {
		return true, nil
	}
	if b.count > a.count {
		return false, nil
	}
	bn, e := m.read(b, b.count-1)
	if e != nil {
		return false, e
	}
	an, e := m.read(a, a.count-1)
	if e != nil {
		return false, e
	}
	ok, e := m.modular.Divides(bn, an)
	if e != nil || !ok {
		return false, e
	}
	var ratio Int
	defer m.modular.Del(&ratio)
	r := &m.div1
	if e = m.setView(r, a); e != nil {
		return false, e
	}
	n := a.count
	for {
		if n == 0 {
			return true, nil
		}
		if b.count > n {
			return false, nil
		}
		bn, e = m.read(b, b.count-1)
		if e != nil {
			return false, e
		}
		ok, e = m.modular.Divides(bn, &m.values(r)[n-1])
		if e != nil || !ok {
			return false, e
		}
		delta := n - b.count
		if e = m.modular.Div(&m.values(r)[n-1], bn, &ratio); e != nil {
			return false, e
		}
		for i := uint32(0); i < b.count-1; i++ {
			x, e := m.read(b, i)
			if e != nil {
				return false, e
			}
			zero, e := m.modular.IsZero(x)
			if e != nil {
				return false, e
			}
			if !zero {
				dst := &m.values(r)[i+delta]
				if e = m.modular.SubMul(dst, &ratio, x, dst); e != nil {
					return false, e
				}
			}
		}
		if e = m.modular.Reset(&m.values(r)[n-1]); e != nil {
			return false, e
		}
		if e = m.trim(r); e != nil {
			return false, e
		}
		n = r.size
	}
}
func (m *UPolynomialManager) DividesViews(a, b UPolynomialView) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return m.dividesViews(a, b)
}
func (m *UPolynomialManager) Divides(a, b UPolynomial) (bool, error) {
	av, e := m.wholeView(a)
	if e != nil {
		return false, e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return false, e
	}
	return m.DividesViews(av, bv)
}

func (m *UPolynomialManager) exactDivState(a, b UPolynomialView, out *upolyVectorState) (bool, error) {
	if b.count == 0 {
		return false, nil
	}
	if a.count == 0 {
		return true, m.reset(out)
	}
	if b.count > a.count {
		return false, nil
	}
	bn, e := m.read(b, b.count-1)
	if e != nil {
		return false, e
	}
	an, e := m.read(a, a.count-1)
	if e != nil {
		return false, e
	}
	ok, e := m.modular.Divides(bn, an)
	if e != nil || !ok {
		return false, e
	}
	b0, e := m.read(b, 0)
	if e != nil {
		return false, e
	}
	a0, e := m.read(a, 0)
	if e != nil {
		return false, e
	}
	ok, e = m.modular.Divides(b0, a0)
	if e != nil || !ok {
		return false, e
	}
	q := &m.exactDiv
	r := &m.div1
	if e = m.reset(q); e != nil {
		return false, e
	}
	deg := a.count - b.count
	if e = m.reserve(q, deg+1); e != nil {
		return false, e
	}
	if e = m.setView(r, a); e != nil {
		return false, e
	}
	if modularSourceDebug && r.size != a.count {
		return false, ErrUPolynomialAssertion
	}
	n := a.count
	for {
		if n == 0 {
			if e = m.setSize(q, deg+1); e != nil {
				return false, e
			}
			*out, *q = *q, *out
			return true, nil
		}
		reject := b.count > n
		if !reject {
			bn, e = m.read(b, b.count-1)
			if e != nil {
				return false, e
			}
			ok, e = m.modular.Divides(bn, &m.values(r)[n-1])
			if e != nil {
				return false, e
			}
			reject = !ok
		}
		if !reject {
			b0, e = m.read(b, 0)
			if e != nil {
				return false, e
			}
			ok, e = m.modular.Divides(b0, &m.values(r)[0])
			if e != nil {
				return false, e
			}
			reject = !ok
		}
		if reject {
			return false, m.reset(out)
		}
		delta := n - b.count
		dst := &m.values(q)[delta]
		bn, e = m.read(b, b.count-1)
		if e != nil {
			return false, e
		}
		if e = m.modular.Div(&m.values(r)[n-1], bn, dst); e != nil {
			return false, e
		}
		for i := uint32(0); i < b.count-1; i++ {
			x, e := m.read(b, i)
			if e != nil {
				return false, e
			}
			zero, e := m.modular.IsZero(x)
			if e != nil {
				return false, e
			}
			if !zero {
				row := &m.values(r)[i+delta]
				if e = m.modular.SubMul(row, dst, x, row); e != nil {
					return false, e
				}
			}
		}
		if e = m.modular.Reset(&m.values(r)[n-1]); e != nil {
			return false, e
		}
		if e = m.trim(r); e != nil {
			return false, e
		}
		n = r.size
	}
}
func (m *UPolynomialManager) ExactDivViews(a, b UPolynomialView, q UPolynomial) (bool, error) {
	out, e := m.descriptor(q)
	if e != nil {
		return false, e
	}
	return m.exactDivState(a, b, &out.state)
}
func (m *UPolynomialManager) ExactDiv(a, b, q UPolynomial) (bool, error) {
	av, e := m.wholeView(a)
	if e != nil {
		return false, e
	}
	bv, e := m.wholeView(b)
	if e != nil {
		return false, e
	}
	return m.ExactDivViews(av, bv, q)
}
