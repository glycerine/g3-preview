package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

// Actual RAW and physical children reached by the retained to_mpq call.
type algebraicBoundConversion struct {
	owner      *AlgebraicRootCore
	hooks      unumeral.NumericInvocationHooks
	roots      unumeral.NumericAllocationRoots
	invocation unumeral.BinaryToRationalInvocation
	raw        []unumeral.IntegerRawInvocationReceipt
	stop       error
	executing  bool
}

func (s *algebraicBoundConversion) NumericBound(executing bool) bool {
	m := s.owner
	return m != nil && m.self == m && !m.closed && s == &m.bounds.conversion && (!executing || s.executing && m.bounds.executing)
}
func (s *algebraicBoundConversion) NumericRoots(z *unumeral.IntegerManager, local *unumeral.NumericAllocationRoots) *unumeral.NumericAllocationRoots {
	if !s.NumericBound(true) || s.owner.config.Rationals.Integers() != z {
		return nil
	}
	if local != nil {
		return local
	}
	return &s.roots
}
func (s *algebraicBoundConversion) NumericRecord(z *unumeral.IntegerManager, _ string, r unumeral.NumericInvocationReceipt) {
	if s.NumericBound(true) && s.owner.config.Rationals.Integers() == z && r.Raw != nil {
		s.raw = append(s.raw, *r.Raw)
	}
}
func (s *algebraicBoundConversion) pending(m *AlgebraicRootCore) bool {
	if s.owner != m {
		return false
	}
	if s.stop != nil || unumeral.NumericAllocationPending(&s.roots) || m.config.Rationals.MatchesRawPowerInvocation(&s.invocation.Power) || m.config.Rationals.MatchesRawNormalizeInvocation(&s.invocation.Normalize) {
		return true
	}
	for i := range s.raw {
		if m.config.Rationals.Integers().MatchesRawInvocation(&s.raw[i]) {
			return true
		}
	}
	return false
}
func (s *algebraicBoundConversion) cleanup() error {
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
	for i := len(s.raw) - 1; i >= 0; i-- {
		if e := z.CleanupRawInvocation(&s.raw[i], true); e != nil {
			return e
		}
	}
	if e := s.owner.config.Rationals.CleanupRawNormalizeInvocation(&s.invocation.Normalize); e != nil {
		return e
	}
	if e := s.owner.config.Rationals.CleanupRawPowerInvocation(&s.invocation.Power); e != nil {
		return e
	}
	clear(s.raw)
	s.raw = s.raw[:0]
	s.stop = nil
	return nil
}
func (m *AlgebraicRootCore) boundToRational(a *unumeral.BinaryRational, out *unumeral.Rational) error {
	s := &m.bounds.conversion
	if s.pending(m) {
		return ErrAlgebraicPending
	}
	clear(s.raw)
	s.raw = s.raw[:0]
	s.invocation = unumeral.BinaryToRationalInvocation{}
	s.owner = m
	s.hooks.Policy = s
	s.executing = true
	defer func() { s.executing = false }()
	context := unumeral.NumericAllocationContext{}.WithCaller(&s.hooks)
	e := m.config.Binary.ToRationalWithInvocation(m.config.Rationals, a, out, context, &s.invocation)
	z := m.config.Rationals.Integers()
	for i := len(s.raw) - 1; i >= 0 && s.stop == nil; i-- {
		s.stop = z.RawInvocationSourceCause(&s.raw[i])
	}
	if s.stop == nil {
		s.stop = m.config.Rationals.RawNormalizeInvocationSourceCause(&s.invocation.Normalize)
	}
	if s.stop == nil {
		s.stop = m.config.Rationals.RawPowerInvocationSourceCause(&s.invocation.Power)
	}
	return e
}
