// Source: src/math/polynomial/linear_eq_solver.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// ModularLinearSolver borrows its number-field manager. Matrix/RHS values live only in
// Runtime bytes. Returned observation values are nonowning scalar projections.
type ModularLinearSolver struct {
	self    *ModularLinearSolver
	manager *util.IntegerModularManager
	linearStorage
}

func modularLinearConstructorGuard(m *util.IntegerModularManager) error {
	if e := m.Check(); e != nil {
		return e
	}
	if polynomialSourceDebug {
		field, e := m.Field()
		if e != nil {
			return e
		}
		if !field {
			return ErrPolynomialAssertion
		}
	}
	return nil
}
func (s *ModularLinearSolver) initializeDefault(m *util.IntegerModularManager, header uint64) {
	*s = ModularLinearSolver{manager: m, linearStorage: linearStorage{runtime: m.Integers().RuntimeOwner(), integer: m.Integers(), header: header, A: linearNumeralVector{width: 4}, b: linearNumeralVector{width: 12}}}
	s.self = s
}
func (s *ModularLinearSolver) constructDefault(m *util.IntegerModularManager) error {
	if e := modularLinearConstructorGuard(m); e != nil {
		return e
	}
	s.initializeDefault(m, 0)
	return nil
}
func NewModularLinearSolver(m *util.IntegerModularManager) (*ModularLinearSolver, error) {
	if e := modularLinearConstructorGuard(m); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(ModularLinearSolver{}).Size())
	if e := m.Integers().Budget().Reserve(header); e != nil {
		return nil, e
	}
	s := &ModularLinearSolver{}
	s.initializeDefault(m, header)
	return s, nil
}
func (s *ModularLinearSolver) identity() error {
	if s == nil || s.self != s || s.manager == nil {
		return ErrPolynomialOwner
	}
	return s.manager.Check()
}
func (s *ModularLinearSolver) open() error {
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
func (s *ModularLinearSolver) load(v *linearNumeralVector, i uint32) (util.Int, error) {
	if polynomialSourceDebug {
		n, e := v.size(s.runtime)
		if e != nil {
			return util.Int{}, e
		}
		if i >= n {
			return util.Int{}, ErrPolynomialAssertion
		}
	}
	if i >= v.initialized {
		return util.Int{}, arena.ErrBounds
	}
	return v.intAt(s.runtime, i)
}
func (s *ModularLinearSolver) put(v *linearNumeralVector, i uint32, a util.Int) error {
	return v.putInt(s.runtime, i, a)
}
func (s *ModularLinearSolver) copyInto(v *linearNumeralVector, i uint32, a *util.Int) error {
	b, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.Copy(&b, a)
	return polynomialNewtonJoin(e, s.put(v, i, b))
}
func (s *ModularLinearSolver) setInto(v *linearNumeralVector, i uint32, value int64) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.SetInt64(&a, value)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *ModularLinearSolver) divideInto(v *linearNumeralVector, i uint32, divisor *util.Int) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.Div(&a, divisor, &a)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *ModularLinearSolver) submulInto(v *linearNumeralVector, i uint32, b, c *util.Int) error {
	a, e := s.load(v, i)
	if e != nil {
		return e
	}
	e = s.manager.SubMul(&a, b, c, &a)
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *ModularLinearSolver) deleteInto(v *linearNumeralVector, i uint32, owning bool) error {
	if i >= v.initialized {
		return arena.ErrBounds
	}
	a, e := v.intAt(s.runtime, i)
	if e != nil {
		return e
	}
	if owning {
		e = s.manager.Del(&a)
	} else {
		e = util.SourceIntegerDel(s.integer, &a)
	}
	return polynomialNewtonJoin(e, s.put(v, i, a))
}
func (s *ModularLinearSolver) RightHandSide(i uint32) (util.Int, error) {
	if e := s.open(); e != nil {
		return util.Int{}, e
	}
	return s.load(&s.b, i)
}
func (s *ModularLinearSolver) Resize(n uint32) error {
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
func (s *ModularLinearSolver) Reset() error {
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
func (s *ModularLinearSolver) Add(i uint32, as []util.Int, b *util.Int) error {
	return s.AddIndexed(i, util.IntegerArrayView{Values: as, Count: uint32(len(as))}, b)
}
func (s *ModularLinearSolver) AddIndexed(i uint32, as util.IntegerArrayView, b *util.Int) error {
	if e := s.open(); e != nil {
		return e
	}
	if polynomialSourceDebug && i >= s.n {
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
		a, readErr := as.CheckedAt(s.runtime, j)
		if readErr != nil {
			return readErr
		}
		if as.SameRuntimeElement(s.runtime, r.raw, 8+uint64(j)*12, j) {
			e = s.manager.Copy(&a, &a)
		} else {
			e = s.copyInto(r, j, &a)
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *ModularLinearSolver) Solve(xs []util.Int) (bool, error) {
	return s.SolveIndexed(util.IntegerArrayView{Values: xs, Count: uint32(len(xs))})
}
func (s *ModularLinearSolver) SolveIndexed(xs util.IntegerArrayView) (bool, error) {
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
		if polynomialSourceDebug {
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
		if polynomialSourceDebug {
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
		if polynomialSourceDebug {
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
		x, e := xs.CheckedAt(s.runtime, k)
		if e != nil {
			return false, e
		}
		if xs.SameRuntimeElement(s.runtime, s.b.raw, 8+uint64(k)*12, k) {
			e = s.manager.Copy(&b, &b)
		} else {
			e = s.manager.Copy(&x, &b)
			e = polynomialNewtonJoin(e, xs.Store(k, x))
		}
		if e != nil {
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
			b, e = s.load(&s.b, k)
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
func (s *ModularLinearSolver) flushSource() (err error) {
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
		if polynomialSourceDebug {
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
		if polynomialSourceDebug {
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
func (s *ModularLinearSolver) Flush() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.flushSource()
}
func (s *ModularLinearSolver) SourceFinalize() (err error) {
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
func (s *ModularLinearSolver) Close() error {
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
	s.integer.Budget().Release(uint64(cap(s.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size()) + s.header)
	s.rows = nil
	s.header = 0
	s.closed = true
	return nil
}
