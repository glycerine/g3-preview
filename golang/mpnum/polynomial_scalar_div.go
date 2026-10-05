// Source: src/math/polynomial/polynomial.cpp:2757-2768.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

type polynomialScalarDivScope struct {
	a                        Int
	returned                 Polynomial
	active, ended, executing bool
	normalize                integerPNormalizeInvocationReceipt
	buffers                  [4]naturalPNBufferReceipt
	verification             naturalPNVerificationReceipt
	transport                Int
	transportBuffer          *polynomialSOM
	transportSlot            uint32
	transportBound           bool
	division                 integerScalarRawScope
	extended                 integerScalarRawScope
	bodyCause, sourceCause   error
	invocation               uint64
	numeric                  polynomialNumericRoots
}
type polynomialScalarDivInvocationReceipt struct {
	manager    *MultivariatePolynomialCore
	frame      *polynomialScalarDivScope
	invocation uint64
}

func (m *MultivariatePolynomialCore) matchesScalarDivInvocation(r *polynomialScalarDivInvocationReceipt) bool {
	return m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.scalarDivision && r.invocation != 0 && r.invocation == r.frame.invocation && r.frame.active
}
func (m *MultivariatePolynomialCore) scalarDivInvocationSourceCause(r *polynomialScalarDivInvocationReceipt) error {
	if m != nil && m.self == m && r != nil && r.manager == m && r.frame == &m.scalarDivision && r.invocation != 0 && r.invocation == r.frame.invocation {
		return r.frame.sourceCause
	}
	return nil
}
func (m *MultivariatePolynomialCore) finishScalarDivSource() error {
	s := &m.scalarDivision
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if !s.active || s.ended {
		return nil
	}
	for _, e := range []error{s.division.sourceCause, s.extended.sourceCause, m.modular.pNormalizeInvocationSourceCause(&s.normalize), m.integer.natural.pnVerificationSourceCause(&s.verification)} {
		if e != nil {
			s.sourceCause = e
			return e
		}
	}
	_, e := upolynomialSourceNumeralDel(m.integer, &s.a)
	if e != nil {
		s.sourceCause = e
		return e
	}
	s.ended = true
	s.returned = Polynomial{}
	if !s.directPending(m) {
		s.active = false
	}
	return nil
}
func (m *MultivariatePolynomialCore) cleanupScalarDivInvocation(r *polynomialScalarDivInvocationReceipt, abort bool) error {
	if !m.matchesScalarDivInvocation(r) {
		return nil
	}
	return m.cleanupScalarDivPhase(abort)
}
func (m *MultivariatePolynomialCore) cleanupScalarDivPhase(abort bool) error {
	s := &m.scalarDivision
	if !s.active {
		return nil
	}
	if s.executing {
		return ErrPolynomialPending
	}
	r := polynomialScalarDivInvocationReceipt{m, s, s.invocation}
	if e := m.modular.cleanupPNormalizeInvocation(&s.normalize, false); e != nil {
		return e
	}
	if e := m.integer.natural.cleanupPNVerification(&s.verification, false); e != nil {
		return e
	}
	if e := m.integer.cleanupScalarRaw(&s.division, &r, false); e != nil {
		return e
	}
	if e := m.integer.cleanupScalarRaw(&s.extended, &r, false); e != nil {
		return e
	}
	if !s.ended {
		if e := m.deleteNumeral(&s.a); e != nil {
			return e
		}
		s.ended = true
	}
	if abort {
		if s.transportBound {
			if e := m.deleteNumeral(&s.transport); e != nil {
				return e
			}
			s.transportBound = false
			s.transportBuffer = nil
			s.transportSlot = 0
		}
		if e := m.modular.cleanupPNormalizeInvocation(&s.normalize, true); e != nil {
			return e
		}
		if e := m.integer.natural.cleanupPNVerification(&s.verification, true); e != nil {
			return e
		}
		if e := m.integer.cleanupScalarRaw(&s.division, &r, true); e != nil {
			return e
		}
		if e := m.integer.cleanupScalarRaw(&s.extended, &r, true); e != nil {
			return e
		}
		for i := len(s.buffers) - 1; i >= 0; i-- {
			if e := s.buffers[i].cleanup(); e != nil {
				return e
			}
		}
		if s.returned != (Polynomial{}) {
			if e := m.CleanupValue(&s.returned); e != nil {
				return e
			}
		}
		if e := s.numeric.cleanup(m.integer); e != nil {
			return e
		}
	}
	if s.returned == (Polynomial{}) && !s.directPending(m) {
		invocation := s.invocation
		*s = polynomialScalarDivScope{invocation: invocation, bodyCause: s.bodyCause, sourceCause: s.sourceCause, division: s.division, extended: s.extended, normalize: s.normalize, verification: s.verification}
	}
	return nil
}
func (m *MultivariatePolynomialCore) scalarDiv(p Polynomial, d *Int) (Polynomial, error) {
	return m.scalarDivWithReceipt(p, d, nil)
}
func (m *MultivariatePolynomialCore) scalarDivWithReceipt(p Polynomial, d *Int, receipt *polynomialScalarDivInvocationReceipt) (result Polynomial, cause error) {
	return m.scalarDivWithParentReceipts(p, d, receipt, nil, nil)
}
func (m *MultivariatePolynomialCore) scalarDivWithParentReceipts(p Polynomial, d *Int, receipt *polynomialScalarDivInvocationReceipt, appendReceipt *polynomialCheapAppendInvocationReceipt, factoryReceipt *polynomialFactoryInvocationReceipt) (result Polynomial, cause error) {
	if e := m.check(); e != nil {
		return Polynomial{}, e
	}
	size, e := m.cheap.ms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && size != 0 {
		return Polynomial{}, ErrPolynomialAssertion
	}
	// Source reads p->size before constructing scoped NUM a. The divisor is
	// read only by an actual loop DIV, including when p has no coefficients.
	n, e := m.readWord(p, 8)
	if e != nil {
		return Polynomial{}, e
	}
	s := &m.scalarDivision
	if s.active || s.invocation == ^uint64(0) {
		return Polynomial{}, ErrPolynomialPending
	}
	*s = polynomialScalarDivScope{active: true, executing: true, invocation: s.invocation + 1}
	directReceipt := polynomialScalarDivInvocationReceipt{manager: m, frame: s, invocation: s.invocation}
	if receipt != nil {
		*receipt = directReceipt
	}

	defer func() {
		s.executing = false
		s.bodyCause = cause
		s.returned = result
		if m.matchesFactoryInvocation(factoryReceipt) && m.factoryInvocation.result == result {
			s.returned = Polynomial{}
		}
		if e := m.finishScalarDivSource(); e != nil {
			result = Polynomial{}
			cause = polynomialNewtonJoin(cause, e)
		}
	}()
	for i := uint32(0); i < n; i++ {
		a, e := m.coefficientCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		e = m.modular.divWithScalarDivReceipt(&a, d, &s.a, &directReceipt)
		if e != nil {
			return Polynomial{}, e
		}
		term, e := m.monomialCore(p, i)
		if e != nil {
			return Polynomial{}, e
		}
		e = m.cheap.addConsumedScalarDivReceipts(&s.a, term, false, nil, appendReceipt, &directReceipt)
		if e != nil {
			return Polynomial{}, e
		}
	}
	return m.cheap.mkWithAppendReceipt(false, appendReceipt, factoryReceipt)
}

func (s *polynomialScalarDivScope) directPending(m *MultivariatePolynomialCore) bool {
	if s.transportBound || polynomialNegNumericPending(&s.numeric) || s.division.active || s.extended.active || m.modular.matchesPNormalizeInvocation(&s.normalize) || m.integer.natural.matchesPNVerification(&s.verification) {
		return true
	}
	for i := range s.buffers {
		if s.buffers[i].pending != (naturalPNBufferReceipt{}).pending {
			return true
		}
	}
	return false
}
