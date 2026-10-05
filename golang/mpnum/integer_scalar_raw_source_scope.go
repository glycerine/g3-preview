// Source: src/util/mpz.h:468-485; src/util/mpz.cpp:915-988.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type integerScalarRawScope struct {
	manager                                 *IntegerManager
	parent                                  polynomialScalarDivInvocationReceipt
	invocation                              uint64
	active, executing, bodySucceeded, ended bool
	values                                  [6]Int
	count, cursor                           uint8
	sourceCause                             error
}

func (m *IntegerManager) scalarRawFrameAddress(s *integerScalarRawScope, r *polynomialScalarDivInvocationReceipt) bool {
	return m != nil && m.self == m && s != nil && r != nil && r.manager != nil && r.manager.self == r.manager && r.manager.integer == m && r.frame == &r.manager.scalarDivision && (s == &r.frame.division || s == &r.frame.extended)
}
func (m *IntegerManager) scalarRawParentIdentity(s *integerScalarRawScope, r *polynomialScalarDivInvocationReceipt) bool {
	return m.scalarRawFrameAddress(s, r) && r.invocation != 0 && r.invocation == r.frame.invocation && s.manager == m && s.parent.manager == r.manager && s.parent.frame == r.frame && s.parent.invocation == r.invocation
}

func (m *IntegerManager) beginScalarRaw(s *integerScalarRawScope, r *polynomialScalarDivInvocationReceipt, count uint8) error {
	if !m.scalarRawFrameAddress(s, r) || m.scalarDivAllocationRoots(r) == nil {
		return ErrPolynomialOwner
	}
	if s.active || s.invocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	*s = integerScalarRawScope{manager: m, parent: *r, invocation: s.invocation + 1, active: true, executing: true, count: count}
	return nil
}
func (m *IntegerManager) finishScalarRawSource(s *integerScalarRawScope) error {
	if s == nil || !m.scalarRawParentIdentity(s, &s.parent) {
		return ErrPolynomialOwner
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active || !s.bodySucceeded || s.ended {
		return nil
	}
	for s.cursor < s.count {
		_, e := upolynomialSourceNumeralDel(m, &s.values[s.cursor])
		if e != nil {
			s.sourceCause = e
			return e
		}
		s.cursor++
	}
	s.ended = true
	s.active = false
	return nil
}
func (m *IntegerManager) cleanupScalarRaw(s *integerScalarRawScope, r *polynomialScalarDivInvocationReceipt, abort bool) error {
	if !m.scalarRawFrameAddress(s, r) {
		return ErrPolynomialOwner
	}
	if !m.scalarRawParentIdentity(s, r) || !r.frame.active || !s.active {
		return nil
	}
	if s.executing {
		return ErrPolynomialPending
	}
	if !abort && !s.bodySucceeded {
		return nil
	}
	for s.cursor < s.count {
		if e := m.Del(&s.values[s.cursor]); e != nil {
			return e
		}
		s.cursor++
	}
	s.active = false
	return nil
}
