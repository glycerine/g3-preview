// Source: src/math/polynomial/upolynomial.cpp:2764-2778,2797-2807, reached
// by src/math/polynomial/algebraic_numbers.cpp:2721-2767.
// Width outlives each midpoint. MPBQ_DEBUG snapshots retain their separate
// diagnostic service ownership; this policy covers the numerical body.
package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

const (
	boundRefinementWidth = 0
	boundRefinementMid   = 1
)

type algebraicBoundRefinement struct {
	owner     *AlgebraicRootCore
	values    [2]unumeral.BinaryRational
	live      uint8
	hooks     unumeral.NumericInvocationHooks
	roots     unumeral.NumericAllocationRoots
	stop      error
	executing bool
}

func (s *algebraicBoundRefinement) NumericBound(executing bool) bool {
	m := s.owner
	return m != nil && m.self == m && !m.closed && s == &m.bounds.refinement && (!executing || s.executing && m.bounds.executing)
}
func (s *algebraicBoundRefinement) NumericRoots(z *unumeral.IntegerManager, local *unumeral.NumericAllocationRoots) *unumeral.NumericAllocationRoots {
	if !s.NumericBound(true) || s.owner.config.Rationals.Integers() != z {
		return nil
	}
	if local != nil {
		return local
	}
	return &s.roots
}
func (s *algebraicBoundRefinement) pending(m *AlgebraicRootCore) bool {
	if s.owner != m {
		return false
	}
	if s.live != 0 || s.stop != nil || unumeral.NumericAllocationPending(&s.roots) {
		return true
	}

	return false
}
func (s *algebraicBoundRefinement) context() unumeral.NumericAllocationContext {
	return unumeral.NumericAllocationContext{}.WithCaller(&s.hooks)
}
func (s *algebraicBoundRefinement) construct(which uint8) (*unumeral.BinaryRational, error) {
	if !s.NumericBound(true) || s.live&(1<<which) != 0 {
		return nil, ErrAlgebraicPending
	}
	s.values[which] = unumeral.BinaryRational{}
	s.live |= 1 << which
	return &s.values[which], nil
}
func (s *algebraicBoundRefinement) sourceFinish(which uint8) error {
	if s.stop != nil {
		return s.stop
	}
	m := s.owner
	if e := m.bounds.evaluationSourceCause(m); e != nil {
		s.stop = e
		return e
	}

	if s.live&(1<<which) == 0 {
		return nil
	}
	_, e := unumeral.SourceNumeralDel(m.config.Rationals.Integers(), s.values[which].BorrowNumerator())
	if e != nil {
		s.stop = e
		return e
	}
	s.live &^= 1 << which
	return nil
}
func (s *algebraicBoundRefinement) cleanup() error {
	if s.owner == nil {
		return nil
	}
	if s.executing {
		return ErrAlgebraicPending
	}
	z := s.owner.config.Rationals.Integers()
	if e := s.roots.Cleanup(z); e != nil {
		return e
	}

	for i := uint8(2); i != 0; {
		i--
		if s.live&(1<<i) != 0 {
			if e := z.CleanupRawNumeral(s.values[i].BorrowNumerator()); e != nil {
				return e
			}
			s.live &^= 1 << i
		}
	}
	s.stop = nil
	return nil
}
func (m *AlgebraicRootCore) refineBoundInterval(v UPolynomialView, lower, upper *unumeral.BinaryRational, k uint32) (bool, error) {
	s := &m.bounds.refinement
	if s.pending(m) {
		return false, ErrAlgebraicPending
	}
	s.owner = m
	s.hooks.Policy = s
	s.executing = true
	defer func() { s.executing = false }()
	return m.config.Polynomials.refineToPrecisionWithEvaluationReceipt(v, m.config.Binary, lower, upper, k, &m.bounds.evaluation, s)
}
