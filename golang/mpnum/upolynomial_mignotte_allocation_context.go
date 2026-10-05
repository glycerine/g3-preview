// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/mpz.h:124-149,393-407; src/util/mpzzp.h:132,150-163,229-235,271-279.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

func (c integerAllocationInvocationContext) mignotteReceipt(m *IntegerManager) *upolyMignotteInvocationReceipt {
	r := c.mignotte
	if m == nil || m.self != m || r == nil || r.service == nil || !r.service.matchesMignotteInvocation(r) || r.service.manager.config.Integer != m || !r.frame.executing || r.frame.constructed == 0 || r.service.manager.callbackDepth != 0 || c.mignotteInvocation == 0 || c.mignotteInvocation != r.invocation {
		return nil
	}
	return r
}
func (m *IntegerModularManager) mignotteAllocationContext(r *upolyMignotteInvocationReceipt) integerAllocationInvocationContext {
	if m == nil || m.manager == nil || r == nil || r.service == nil || r.service.manager == nil || r.service.manager.modular != m {
		return integerAllocationInvocationContext{}
	}
	c := integerAllocationInvocationContext{mignotte: r, mignotteInvocation: r.invocation}
	if c.mignotteReceipt(m.manager) == nil {
		return integerAllocationInvocationContext{}
	}
	return c
}

// One source call dispatches into the original scalar engines. The explicit
// context reaches only actual allocation leaves; successful NUM publication
// remains in the declared norm/bound/tmp, never a mirrored numeral array.
func (s *UPolynomialFactorizationService) mignotteNumericCall(r *upolyMignotteInvocationReceipt, op uint8, a, b, out *Int, k uint32) error {
	if s == nil || s.self != s || r == nil || r.service != s {
		return ErrUPolynomialOwner
	}
	m := s.manager.modular
	if e := m.check(); e != nil {
		return e
	}
	c := m.mignotteAllocationContext(r)
	if c.mignotteReceipt(m.manager) == nil {
		return ErrUPolynomialOwner
	}
	r.frame.enterCall(s.manager)
	defer r.frame.leaveCall(s.manager)
	q := r.frame
	if !((out == &q.norm && q.constructed >= 1) || (out == &q.bound && q.constructed >= 2) || (out == &q.tmp && q.constructed >= 3)) {
		return ErrUPolynomialOwner
	}
	var e error
	switch op {
	case 0:
		e = m.addMulWithSourceContext(out, a, b, out, &r.frame.addMul, c)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.manager.addMulInvocationSourceCause(&r.frame.addMul)
		}
		return e
	case 1:
		_, e = m.rootInPlaceWithReceipt(out, k, &r.frame.root)
		if r.frame.sourceCause == nil {
			r.frame.sourceCause = m.manager.rootInvocationSourceCause(&r.frame.root)
		}
		return e
	case 2:
		return m.setInt64WithSourceContext(out, int64(int32(k)), c)
	case 3:
		return m.mul2kWithSourceContext(a, k, out, c)
	case 4:
		return m.copyWithSourceContext(out, a, c)
	case 5:
		return m.absWithSourceContext(out, c)
	case 6:
		return m.addWithSourceContext(a, b, out, c)
	case 7:
		return m.mulWithSourceContext(a, b, out, c)
	default:
		return ErrUPolynomialAssertion
	}
}
