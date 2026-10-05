// Source: src/util/mpz.h:393-407 (plain RAW tmp; end-only explicit DEL).
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// This is actual source invocation state in the charged Integer owner. Body
// failure does not reach DEL; its published temporary is explicitly unjoined.
type integerAddMulScope struct {
	invocation                      uint64
	active, executing, ended, owned bool
	stage                           uint8
	tmp                             Int
	numeric                         polynomialNumericRoots
	parent                          polynomialArithmeticInvocationReceipt
	bodyCause, sourceCause          error
}
type integerAddMulInvocationReceipt struct {
	manager    *IntegerManager
	frame      *integerAddMulScope
	invocation uint64
}

func (m *IntegerManager) matchesAddMulInvocation(r *integerAddMulInvocationReceipt) bool {
	return r != nil && r.manager == m && m.self == m && !m.closed && r.frame == &m.addMulInvocation && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *IntegerManager) addMulInvocationSourceCause(r *integerAddMulInvocationReceipt) error {
	if r != nil && r.manager == m && r.frame == &m.addMulInvocation && r.invocation != 0 && r.invocation == r.frame.invocation {
		return r.frame.sourceCause
	}
	return nil
}
func (m *IntegerManager) finishAddMulSource(s *integerAddMulScope) error {
	if s != &m.addMulInvocation {
		return nil
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active || s.stage != 3 || s.ended || s.owned {
		return nil
	}
	_, e := upolynomialSourceNumeralDel(m, &s.tmp)
	if e != nil {
		s.sourceCause = e
		return e
	}
	s.ended = true
	if !polynomialNegNumericPending(&s.numeric) {
		s.active = false
	}
	return nil
}
func (m *IntegerManager) cleanupAddMulInvocation(r *integerAddMulInvocationReceipt, abort bool) error {
	if !m.matchesAddMulInvocation(r) {
		return nil
	}
	s := r.frame
	if s.executing {
		return ErrPolynomialPending
	}
	// Only reached end DEL may complete before older caller declarations.
	if s.stage == 3 && !s.ended && !s.owned {
		if e := m.Del(&s.tmp); e != nil {
			return e
		}
		s.owned = true
	}
	if !abort {
		return nil
	}
	if !s.ended && !s.owned {
		if e := m.Del(&s.tmp); e != nil {
			return e
		}
		s.owned = true
	}
	if e := s.numeric.cleanup(m); e != nil {
		return e
	}
	*s = integerAddMulScope{invocation: s.invocation, stage: s.stage, ended: s.ended, owned: s.owned, bodyCause: s.bodyCause, sourceCause: s.sourceCause}
	return nil
}
