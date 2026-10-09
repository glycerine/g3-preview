// Source: src/math/polynomial/linear_eq_solver.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// ModularLinearSolver borrows its number-field manager. Matrix/RHS values live only in
// native Go slices. Returned observation values are nonowning scalar projections.
type ModularLinearSolver struct {
	self    *ModularLinearSolver
	manager *util.IntegerModularManager
	linearStorage
	A [][]util.Int
	b []util.Int
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
	*s = ModularLinearSolver{manager: m, linearStorage: linearStorage{integer: m.Integers(), header: header}}
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
func (s *ModularLinearSolver) row(i uint32) (*[]util.Int, error) {
	if uint64(i) >= uint64(len(s.A)) {
		if polynomialSourceDebug {
			return nil, ErrPolynomialAssertion
		}
		return nil, arena.ErrBounds
	}
	return &s.A[i], nil
}
func (s *ModularLinearSolver) swapRows(i, j uint32) error {
	if _, e := s.row(i); e != nil {
		return e
	}
	if _, e := s.row(j); e != nil {
		return e
	}
	s.A[i], s.A[j] = s.A[j], s.A[i]
	return nil
}
func (s *ModularLinearSolver) load(v *[]util.Int, i uint32) (util.Int, error) {
	if uint64(i) >= uint64(len(*v)) {
		if polynomialSourceDebug {
			return util.Int{}, ErrPolynomialAssertion
		}
		return util.Int{}, arena.ErrBounds
	}
	return (*v)[i], nil
}
func (s *ModularLinearSolver) copyInto(v *[]util.Int, i uint32, a *util.Int) error {
	if _, e := s.load(v, i); e != nil {
		return e
	}
	return s.manager.Copy(&(*v)[i], a)
}
func (s *ModularLinearSolver) setInto(v *[]util.Int, i uint32, value int64) error {
	if _, e := s.load(v, i); e != nil {
		return e
	}
	return s.manager.SetInt64(&(*v)[i], value)
}
func (s *ModularLinearSolver) divideInto(v *[]util.Int, i uint32, divisor *util.Int) error {
	if _, e := s.load(v, i); e != nil {
		return e
	}
	return s.manager.Div(&(*v)[i], divisor, &(*v)[i])
}
func (s *ModularLinearSolver) submulInto(v *[]util.Int, i uint32, b, c *util.Int) error {
	if _, e := s.load(v, i); e != nil {
		return e
	}
	return s.manager.SubMul(&(*v)[i], b, c, &(*v)[i])
}
func (s *ModularLinearSolver) deleteInto(v *[]util.Int, i uint32, owning bool) error {
	if uint64(i) >= uint64(len(*v)) {
		return arena.ErrBounds
	}
	if owning {
		return s.manager.Del(&(*v)[i])
	}
	return util.SourceIntegerDel(s.integer, &(*v)[i])
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
	s.A = make([][]util.Int, n)
	s.b = make([]util.Int, n)
	for i := range s.A {
		s.A[i] = make([]util.Int, n)
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
		if uint64(j) >= uint64(len(as)) {
			return arena.ErrBounds
		}
		if e = s.copyInto(r, j, &as[j]); e != nil {
			return e
		}
	}
	return nil
}
func (s *ModularLinearSolver) Solve(xs []util.Int) (bool, error) {
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
		if uint64(k) >= uint64(len(xs)) {
			return false, arena.ErrBounds
		}
		if e = s.manager.Copy(&xs[k], &s.b[k]); e != nil {
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
	if s.flushStage == 0 {
		if polynomialSourceDebug && len(s.b) != len(s.A) {
			return ErrPolynomialAssertion
		}
		s.flushStage = 1
	}
	for uint64(s.flushRow) < uint64(len(s.A)) {
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
		if polynomialSourceDebug && uint64(len(*r)) != uint64(s.n) {
			return ErrPolynomialAssertion
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
	// Numeric destruction completed in source order; native containers need no destructor.
	clear(s.A)
	s.A = s.A[:0]
	s.b = s.b[:0]
	s.n = 0
	s.flushRow, s.flushColumn, s.flushStage = 0, 0, 0
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
	s.b = nil
	s.A = nil
	s.sourceStage = 3
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
	// Explicit owning continuation releases actual numeral resources after a stopped source destructor.
	for i := range s.A {
		for j := range s.A[i] {
			if e := s.deleteInto(&s.A[i], uint32(j), true); e != nil {
				return e
			}
		}
	}
	for i := range s.b {
		if e := s.deleteInto(&s.b, uint32(i), true); e != nil {
			return e
		}
	}
	s.A, s.b = nil, nil
	s.integer.Budget().Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
