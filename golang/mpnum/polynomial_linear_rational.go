// Source: src/math/polynomial/linear_eq_solver.h. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2012-01-22.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// RationalLinearSolver borrows its number-field manager. Matrix/RHS values live only in
// Runtime bytes. Returned observation values are nonowning scalar projections.
type RationalLinearSolver struct {
	self    *RationalLinearSolver
	manager *RationalManager
	linearStorage
}

func NewRationalLinearSolver(m *RationalManager) (*RationalLinearSolver, error) {
	if e := m.checkOwnerReady(); e != nil {
		return nil, e
	}
	if modularSourceDebug && !m.Field() {
		return nil, ErrPolynomialAssertion
	}
	h := uint64(reflect.TypeOf(RationalLinearSolver{}).Size())
	if e := m.z.budget.Reserve(h); e != nil {
		return nil, e
	}
	s := &RationalLinearSolver{manager: m, linearStorage: linearStorage{runtime: m.z.memory, integer: m.z, header: h, A: linearNumeralVector{width: 4}, b: linearNumeralVector{width: 24}}}
	s.self = s
	return s, nil
}
func (s *RationalLinearSolver) identity() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrPolynomialOwner
	}
	return s.manager.checkOwnerReady()
}
func (s *RationalLinearSolver) open() error {
	if e := s.identity(); e != nil {
		return e
	}
	if s.closed || s.closing || s.sourceFinalized || s.sourceStage != 0 {
		return arena.ErrClosed
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	return nil
}
func (s *RationalLinearSolver) load(v *linearNumeralVector, i uint32) (Rational, error) {
	if modularSourceDebug {
		n, e := v.size(s.runtime)
		if e != nil {
			return Rational{}, e
		}
		if i >= n {
			return Rational{}, ErrPolynomialAssertion
		}
	}
	if i >= v.initialized {
		return Rational{}, arena.ErrBounds
	}
	return v.rationalAt(s.runtime, i)
}
func (s *RationalLinearSolver) put(v *linearNumeralVector, i uint32, a Rational) error {
	return v.putRational(s.runtime, i, a)
}
func (s *RationalLinearSolver) copyInto(v *linearNumeralVector, i uint32, a *Rational) error {
	b, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.Copy(&b, a)
	return polynomialNewtonJoin(e, s.put(v, i, b))
}
func (s *RationalLinearSolver) setInto(v *linearNumeralVector, i uint32, value int64) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.SetInt64(&a, value)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *RationalLinearSolver) divideInto(v *linearNumeralVector, i uint32, divisor *Rational) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.Div(&a, divisor, &a)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *RationalLinearSolver) submulInto(v *linearNumeralVector, i uint32, b, c *Rational) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.SubMul(&a, b, c, &a)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *RationalLinearSolver) deleteInto(v *linearNumeralVector, i uint32, owning bool) error {
	if i >= v.initialized {
		return arena.ErrBounds
	}
	a, e := v.rationalAt(s.runtime, i)
	if e != nil {
		return e
	}
	if owning {
		e = s.manager.Del(&a)
	} else {
		e = linearSourceRationalDel(s.manager, &a)
	}
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *RationalLinearSolver) RightHandSide(i uint32) (Rational, error) {
	if e := s.open(); e != nil {
		return Rational{}, e
	}
	return s.load(&s.b, i)
}
func (s *RationalLinearSolver) Resize(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if s.n == n {
		return nil
	}
	if e := s.Flush(); e != nil {
		return e
	}
	s.n = n
	for i := uint32(0); i < n; i++ {
		if e := s.appendRow(i); e != nil {
			return e
		}
		r, e := s.row(i)
		if e != nil {
			return e
		}
		for j := uint32(0); j < n; j++ {
			if e = r.appendDefault(s.runtime, 0); e != nil {
				return e
			}
		}
		if e = s.b.appendDefault(s.runtime, 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *RationalLinearSolver) Reset() error {
	if e := s.open(); e != nil {
		return e
	}
	for i := uint32(0); i < s.n; i++ {
		r, e := s.row(i)
		if e != nil {
			return e
		}
		for j := uint32(0); j < s.n; j++ {
			if e = s.setInto(r, j, 0); e != nil {
				return e
			}
		}
		if e = s.setInto(&s.b, i, 0); e != nil {
			return e
		}
	}
	return nil
}
func (s *RationalLinearSolver) Add(i uint32, as []Rational, b *Rational) error {
	if e := s.open(); e != nil {
		return e
	}
	if modularSourceDebug && i >= s.n {
		return ErrPolynomialAssertion
	}
	if e := s.copyInto(&s.b, i, b); e != nil {
		return e
	}
	r, e := s.row(i)
	if e != nil {
		return e
	}
	for j := uint32(0); j < s.n; j++ {
		if uint64(j) >= uint64(len(as)) {
			return arena.ErrBounds
		}
		if e = s.copyInto(r, j, &as[j]); e != nil {
			return e
		}
	}
	return nil
}
func (s *RationalLinearSolver) Solve(xs []Rational) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	for k := uint32(0); k < s.n; k++ {
		i := k
		for ; i < s.n; i++ {
			r, e := s.row(i)
			if e != nil {
				return false, e
			}
			a, e := s.load(r, k)
			if e != nil {
				return false, e
			}
			zero, e := s.manager.IsZero(&a)
			if e != nil {
				return false, e
			}
			if !zero {
				break
			}
		}
		if i == s.n {
			return false, nil
		}
		// Literal source swaps only matrix row descriptors, not right-hand sides.
		if e := s.swapRows(k, i); e != nil {
			return false, e
		}
		rk, e := s.row(k)
		if e != nil {
			return false, e
		}
		pivot, e := s.load(rk, k)
		if e != nil {
			return false, e
		}
		if modularSourceDebug {
			zero, e := s.manager.IsZero(&pivot)
			if e != nil {
				return false, e
			}
			if zero {
				return false, ErrPolynomialAssertion
			}
		}
		for j := k + 1; j < s.n; j++ {
			if e = s.divideInto(rk, j, &pivot); e != nil {
				return false, e
			}
		}
		if e = s.divideInto(&s.b, k, &pivot); e != nil {
			return false, e
		}
		if e = s.setInto(rk, k, 1); e != nil {
			return false, e
		}
		if modularSourceDebug {
			for j := uint32(0); j < k; j++ {
				a, e := s.load(rk, j)
				if e != nil {
					return false, e
				}
				zero, e := s.manager.IsZero(&a)
				if e != nil {
					return false, e
				}
				if !zero {
					return false, ErrPolynomialAssertion
				}
			}
		}
		for i := k + 1; i < s.n; i++ {
			ri, e := s.row(i)
			if e != nil {
				return false, e
			}
			factor, e := s.load(ri, k)
			if e != nil {
				return false, e
			}
			for j := k + 1; j < s.n; j++ {
				a, e := s.load(rk, j)
				if e != nil {
					return false, e
				}
				if e = s.submulInto(ri, j, &factor, &a); e != nil {
					return false, e
				}
			}
			b, e := s.load(&s.b, k)
			if e != nil {
				return false, e
			}
			if e = s.submulInto(&s.b, i, &factor, &b); e != nil {
				return false, e
			}
			if e = s.setInto(ri, k, 0); e != nil {
				return false, e
			}
		}
	}
	k := s.n
	for k > 0 {
		k--
		rk, e := s.row(k)
		if e != nil {
			return false, e
		}
		if modularSourceDebug {
			a, e := s.load(rk, k)
			if e != nil {
				return false, e
			}
			one, e := s.manager.IsOne(&a)
			if e != nil {
				return false, e
			}
			if !one {
				return false, ErrPolynomialAssertion
			}
		}
		b, e := s.load(&s.b, k)
		if e != nil {
			return false, e
		}
		if uint64(k) >= uint64(len(xs)) {
			return false, arena.ErrBounds
		}
		if e = s.manager.Copy(&xs[k], &b); e != nil {
			return false, e
		}
		i := k
		for i > 0 {
			i--
			ri, e := s.row(i)
			if e != nil {
				return false, e
			}
			a, e := s.load(ri, k)
			if e != nil {
				return false, e
			}
			if e = s.submulInto(&s.b, i, &a, &b); e != nil {
				return false, e
			}
			if e = s.setInto(ri, k, 0); e != nil {
				return false, e
			}
		}
	}
	return true, nil
}
func (s *RationalLinearSolver) flushSource() (err error) {
	if s.sourceCause != nil {
		return s.sourceCause
	}
	defer func() {
		if err != nil {
			s.sourceCause = err
		}
	}()
	sz, e := s.A.size(s.runtime)
	if e != nil {
		return e
	}
	if s.flushStage == 0 {
		if modularSourceDebug {
			bs, e := s.b.size(s.runtime)
			if e != nil {
				return e
			}
			if bs != sz {
				return ErrPolynomialAssertion
			}
		}
		s.flushStage = 1
	}
	for s.flushRow < sz {
		r, e := s.row(s.flushRow)
		if e != nil {
			return e
		}
		if s.flushStage == 1 {
			if e = s.deleteInto(&s.b, s.flushRow, false); e != nil {
				return e
			}
			s.flushStage = 2
		}
		if modularSourceDebug {
			rs, e := r.size(s.runtime)
			if e != nil {
				return e
			}
			if rs != s.n {
				return ErrPolynomialAssertion
			}
		}
		for s.flushColumn < s.n {
			if e = s.deleteInto(r, s.flushColumn, false); e != nil {
				return e
			}
			s.flushColumn++
		}
		s.flushColumn = 0
		s.flushRow++
		s.flushStage = 1
	}
	if s.flushStage < 3 {
		s.flushStage = 3
		s.flushRow = 0
	}
	// A.reset destroys child vectors in ascending source descriptor order.
	for s.flushRow < sz {
		r, e := s.row(s.flushRow)
		if e != nil {
			return e
		}
		if e = s.sourceBacking(r); e != nil {
			return e
		}
		s.flushRow++
	}
	if e = s.A.resetSize(s.runtime); e != nil {
		return e
	}
	if e = s.b.resetSize(s.runtime); e != nil {
		return e
	}
	s.n = 0
	s.flushRow = 0
	s.flushColumn = 0
	s.flushStage = 0
	return nil
}
func (s *RationalLinearSolver) Flush() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.flushSource()
}
func (s *RationalLinearSolver) SourceFinalize() (err error) {
	if e := s.identity(); e != nil {
		return e
	}
	if s.closed {
		return arena.ErrClosed
	}
	if s.sourceFinalized {
		return nil
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	defer func() {
		if err != nil {
			s.sourceCause = err
		}
	}()
	if s.sourceStage == 0 {
		if e := s.flushSource(); e != nil {
			return e
		}
		s.sourceStage = 1
	}
	if s.sourceStage == 1 {
		if e := s.sourceBacking(&s.b); e != nil {
			return e
		}
		s.sourceStage = 2
	}
	if s.sourceStage == 2 {
		if e := s.sourceBacking(&s.A); e != nil {
			return e
		}
		s.sourceStage = 3
	}
	s.sourceFinalized = true
	return nil
}
func (s *RationalLinearSolver) Close() error {
	if s == nil || s.self != s {
		return ErrPolynomialOwner
	}
	if s.closed {
		return nil
	}
	if e := s.identity(); e != nil {
		return e
	}
	if !s.closing {
		s.closing = true
		if e := s.SourceFinalize(); e != nil {
			return e
		}
	}
	// Only explicit owning continuation acknowledges physical release failures.
	// Every row below was created by appendRow and is an actual named member.
	for i := range s.rows {
		v := &s.rows[i]
		if e := s.ownVectorNumerals(v); e != nil {
			return e
		}
		if e := s.freeTokens(v); e != nil {
			return e
		}
	}
	if e := s.ownVectorNumerals(&s.b); e != nil {
		return e
	}
	if e := s.freeTokens(&s.b); e != nil {
		return e
	}
	if e := s.freeTokens(&s.A); e != nil {
		return e
	}
	s.integer.budget.Release(uint64(cap(s.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size()) + s.header)
	s.rows = nil
	s.header = 0
	s.closed = true
	return nil
}
