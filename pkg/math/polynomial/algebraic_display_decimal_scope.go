// Source: algebraic_numbers.cpp:2736-2746; scoped l,u and reached get_interval.
package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

type algebraicDecimalDisplayScope struct {
	values                  [2]unumeral.BinaryRational
	remaining               uint8
	copy                    unumeral.IntegerCopyAllocationReceipt
	refinement              algebraicRefinementScope
	display                 unumeral.BinaryDisplayInvocationReceipt
	sourceStop, displayStop error
}

func (s *algebraicDecimalDisplayScope) pending(m *AlgebraicRootCore) bool {
	return s.remaining != 0 || m.config.Rationals.Integers().MatchesCopyAllocationInvocation(&s.copy) || s.refinement.pending(m) || m.config.Binary.DisplayInvocationPending(&s.display)
}
func (m *AlgebraicRootCore) cleanupDecimalDisplay() error { return m.finishDecimalDisplay(true) }
func (m *AlgebraicRootCore) finishDecimalDisplay(owning bool) error {
	if m.decimalDisplayRunning {
		return ErrAlgebraicPending
	}
	m.decimalDisplayRunning = true
	defer func() { m.decimalDisplayRunning = false }()
	s := &m.decimalDisplay
	if !owning {
		if s.displayStop != nil {
			return s.displayStop
		}
		if s.refinement.stop != nil {
			return s.refinement.stop
		}
		if e := s.refinement.evaluationSourceCause(m); e != nil {
			return e
		}
	} else {
		if e := m.config.Binary.CleanupDisplayInvocation(&s.display); e != nil {
			return e
		}
		if e := s.refinement.cleanup(); e != nil {
			return e
		}
		if e := m.config.Rationals.Integers().CleanupCopyAllocationInvocation(&s.copy); e != nil {
			return e
		}
	}
	for s.remaining != 0 {
		value := &s.values[s.remaining-1]
		var e error
		if owning {
			e = m.config.Binary.Del(value)
		} else {
			e = unumeral.SourceBinaryDel(m.config.Binary, value)
		}
		if e != nil {
			return e
		}
		s.remaining--
	}
	return nil
}
