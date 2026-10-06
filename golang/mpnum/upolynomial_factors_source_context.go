// Source: src/math/polynomial/upolynomial.cpp:33-63.
// Concrete FS class destruction shared by its actual captured callers.
package mpnum

import "reflect"

func (fs *UPolynomialFactors) closeWithSourceContext(st *upolyPreparedFactorCursor, source bool, context integerAllocationInvocationContext) error {
	if fs == nil || fs.self != fs || fs.manager == nil || st == nil {
		return ErrUPolynomialOwner
	}
	if fs.closed {
		return nil
	}
	if e := fs.manager.check(); e != nil {
		return e
	}
	if context != (integerAllocationInvocationContext{}) && context.roots(fs.manager.config.Integer) == nil {
		return ErrUPolynomialOwner
	}
	if source {

		m := fs.manager
		z := m.config.Integer
		if st.stage == 0 {
			for st.row < fs.count {
				v := &fs.factors[st.row]
				for st.column < v.size {
					a, e := m.read(m.stateView(v), st.column)
					if e != nil {
						return e
					}
					if _, e = upolynomialSourceNumeralDel(z, a); e != nil {
						return e
					}
					st.column++
				}
				v.size = 0
				st.column = 0
				st.row++
			}
			st.stage = 1
		}
		if st.stage == 1 {
			for i := uint32(0); i < fs.count; i++ {
				m.freeBacking(&fs.factors[i])
			}
			fs.count = 0
			fs.degrees.size = 0
			st.stage = 2
		}
		if st.stage == 2 {
			if e := m.modular.setInt32WithSourceContext(&fs.constant, 1, context); e != nil {
				return e
			}
			fs.totalFactors = 0
			fs.totalDegree = 0
			st.stage = 3
		}
		if st.stage == 3 {
			if _, e := upolynomialSourceNumeralDel(z, &fs.constant); e != nil {
				return e
			}
			st.stage = 4
		}
		if !fs.closed {
			budget := z.budget
			budget.Release(uint64(cap(fs.degrees.values)) * 4)
			fs.degrees = upolySequenceUnsigned{}
			budget.Release(uint64(cap(fs.factors)) * uint64(reflect.TypeOf(upolyVectorState{}).Size()))
			fs.factors = nil
			budget.Release(fs.header)
			m.liveFactors--
			fs.closed = true
		}
		return nil
	}

	m := fs.manager
	z := m.config.Integer
	for st.row < uint32(len(fs.factors)) {
		v := &fs.factors[st.row]
		for st.column < v.size {
			a, e := m.read(m.stateView(v), st.column)
			if e != nil {
				return e
			}
			if e = z.cleanupRawNumeral(a); e != nil {
				return e
			}
			st.column++
		}
		v.size = 0
		m.freeBacking(v)
		st.column = 0
		st.row++
	}
	if st.stage == 0 {
		if e := z.cleanupRawNumeral(&fs.constant); e != nil {
			return e
		}
		st.stage = 1
	}
	if !fs.closed {
		b := z.budget
		b.Release(uint64(cap(fs.degrees.values)) * 4)
		fs.degrees = upolySequenceUnsigned{}
		b.Release(uint64(cap(fs.factors)) * uint64(reflect.TypeOf(upolyVectorState{}).Size()))
		fs.factors = nil
		b.Release(fs.header)
		m.liveFactors--
		fs.count = 0
		fs.totalFactors = 0
		fs.totalDegree = 0
		fs.closed = true
	}
	return nil
}
