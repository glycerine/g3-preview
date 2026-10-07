// Source: src/util/ref_vector.h:96-100; src/math/polynomial/polynomial.cpp:5626.
package polynomial

type polynomialReferenceVectorPushReceipt struct {
	manager    *MultivariatePolynomialCore
	owner      *PolynomialRefVector
	frame      *polynomialReferenceVector
	invocation uint64
}

func (v *PolynomialRefVector) pushWithReceipt(p Polynomial, r *polynomialReferenceVectorPushReceipt) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.state.pushSource(v.owner, p, v, r)
}
func (r *polynomialReferenceVectorPushReceipt) matches() bool {
	return r != nil && r.manager != nil && r.manager.self == r.manager && r.owner != nil && r.owner.self == r.owner && r.owner.owner == r.manager && r.frame == &r.owner.state && r.invocation != 0 && r.invocation == r.frame.pushInvocation
}
func (r *polynomialReferenceVectorPushReceipt) cleanup() (err error) {
	if !r.matches() {
		return nil
	}
	s := r.frame
	if s.pushExecuting {
		return ErrPolynomialPending
	}
	if !s.pushActive {
		return nil
	}
	defer func() {
		if err != nil && s.pushCleanupCause == nil {
			s.pushCleanupCause = err
		}
	}()
	if e := polynomialReferenceDrop(r.manager, &s.incoming, &s.incomingDec); e != nil {
		return e
	}
	s.pushActive = false
	return nil
}
