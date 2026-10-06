// Source: src/math/polynomial/upolynomial.cpp:33-54;
// src/math/polynomial/upolynomial_factorization.cpp:1062,1120-1139,1172-1175.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj).
package mpnum

import "reflect"

type upolyPreparedFactorCursor struct {
	stage       uint8
	row, column uint32
}

// Owning rescue of the exact constructed member never replays a source SET
// against a modulus whose failed Copy left an unreadable pointer codec.
func (c *upolyFactorSquareFreeScope) closePreparedFactors(fs *UPolynomialFactors, index int, source bool) (cause error) {
	q := &c.primePreparation
	if !q.matches() || c.service.scope != c || index < 0 || index > 1 || fs == nil || fs.self != fs || (index == 0 && (c.constructed != 4 || c.best != fs || fs.manager != q.related)) || (index == 1 && (c.constructed != 9 || c.lifted != fs || fs.manager != q.lift)) {
		return ErrUPolynomialOwner
	}
	if q.executing || q.sourceExecuting || q.ownExecuting {
		return ErrUPolynomialCleanupPending
	}
	if source {
		if q.sourceCause != nil {
			return q.sourceCause
		}
		q.sourceExecuting = true
		defer func() { q.sourceExecuting = false }()
		q.enterCall()
		defer q.leaveCall()
		previousActive := q.active
		q.active = true
		defer func() {
			if cause != nil && q.sourceCause == nil {
				q.sourceCause = cause
			}
			q.active = previousActive || cause != nil || q.pending()
		}()
		st := &q.factorSource[index]
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
			if e := m.modular.setInt32WithSourceContext(&fs.constant, 1, m.modular.primePreparationAllocationContext(q)); e != nil {
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
	q.ownExecuting = true
	defer func() { q.ownExecuting = false }()
	q.enterCall()
	defer q.leaveCall()
	defer func() {
		if cause != nil && q.ownCause == nil {
			q.ownCause = cause
			q.active = true
		}
	}()
	st := &q.factorOwn[index]
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
