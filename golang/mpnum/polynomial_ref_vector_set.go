// Source: src/util/ref_vector.h:67-71,128-132; src/util/vector.h:422-433.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

// This receipt names only one actual guarded caller-vector SET. It does not
// acquire the previously published caller range or its allocation backing.
type polynomialReferenceVectorSetReceipt struct {
	manager    *MultivariatePolynomialCore
	owner      *PolynomialRefVector
	frame      *polynomialReferenceVector
	invocation uint64
}

func (v *PolynomialRefVector) setWithReceipt(i uint32, p Polynomial, r *polynomialReferenceVectorSetReceipt) error {
	return v.setWithFactoryReceipt(i, p, r, nil)
}
func (v *PolynomialRefVector) setWithFactoryReceipt(i uint32, p Polynomial, r *polynomialReferenceVectorSetReceipt, factory *polynomialFactoryInvocationReceipt) (err error) {
	if e := v.open(); e != nil {
		return e
	}
	s := &v.state
	if s.pushActive || s.setActive || s.draining || s.incoming != (Polynomial{}) || s.setInvocation == ^uint64(0) {
		return ErrPolynomialPending
	}
	s.setInvocation++
	s.setIndex = i
	s.setActive = true
	s.setExecuting = true
	s.setOldConsumed = false
	s.setOldRead = false
	s.setBodyCause = nil
	s.setCleanupCause = nil
	if r != nil {
		*r = polynomialReferenceVectorSetReceipt{v.owner, v, s, s.setInvocation}
	}
	defer func() {
		s.setExecuting = false
		if err != nil && s.setBodyCause == nil {
			s.setBodyCause = err
		}
	}()
	if v.owner.matchesFactoryInvocation(factory) && factory.frame.result != p {
		return ErrPolynomialOwner
	}
	if p != (Polynomial{}) {
		if e := v.owner.incRefCore(p); e != nil {
			s.setActive = false
			return e
		}
	}
	s.incoming = p
	// Caller vector has completed its real INC before raw factory transfer.
	if e := v.owner.transferFactoryResult(factory, p); e != nil {
		return e
	}
	old, e := s.get(v.owner, i)
	if e != nil {
		return e
	}
	s.setOldRead = true
	if e = polynomialReferenceDrop(v.owner, &old, &s.incomingDec); e != nil {
		return e
	}
	s.setOldConsumed = true
	s.incomingDec = false
	if e = s.values.putMonomial(v.owner, i, Monomial{owner: p.owner, slot: p.slot, generation: p.generation}); e != nil {
		return e
	}
	s.incoming = Polynomial{}
	s.setActive = false
	return nil
}
func (v *polynomialReferenceVector) abortSet(m *MultivariatePolynomialCore) (err error) {
	if !v.setActive {
		return nil
	}
	if v.setExecuting {
		return ErrPolynomialPending
	}
	defer func() {
		if err != nil && v.setCleanupCause == nil {
			v.setCleanupCause = err
		}
	}()
	// Source SET increments incoming before the vector index assertion. If
	// that actual access failed, no older slot was read or acquired; owning
	// continuation must consume only the completed incoming INC.
	if !v.setOldRead {
		if e := polynomialReferenceDrop(m, &v.incoming, &v.incomingDec); e != nil {
			return e
		}
		v.setActive = false
		return nil
	}
	if !v.setOldConsumed {
		old, e := v.get(m, v.setIndex)
		if e != nil {
			return e
		}
		if e = polynomialReferenceDrop(m, &old, &v.incomingDec); e != nil {
			return e
		}
		v.incomingDec = false
		v.setOldConsumed = true
	}
	// The failed source SET did not publish incoming. Retire only its actual
	// old slot and incoming INC after the younger parent's declarations.
	if e := v.values.putMonomial(m, v.setIndex, Monomial{}); e != nil {
		return e
	}
	if e := polynomialReferenceDrop(m, &v.incoming, &v.incomingDec); e != nil {
		return e
	}
	v.setActive = false
	return nil
}
func (r *polynomialReferenceVectorSetReceipt) matches() bool {
	return r != nil && r.manager != nil && r.manager.self == r.manager && r.owner != nil && r.owner.owner == r.manager && r.owner.self == r.owner && r.frame == &r.owner.state && r.invocation != 0 && r.invocation == r.frame.setInvocation
}
func (r *polynomialReferenceVectorSetReceipt) cleanup() error {
	if !r.matches() {
		return nil
	}
	return r.frame.abortSet(r.owner.owner)
}
func (r *polynomialReferenceVectorSetReceipt) bodyCause() error {
	if !r.matches() {
		return nil
	}
	return r.frame.setBodyCause
}

// The reached SubresultantChain resize follows a completed reset. Only growth
// from that zero source pointer range is claimed here; no shrink adapter is
// substituted for the separate ref_vector shrink/destructor source algorithm.
func (v *PolynomialRefVector) resizeZeroRange(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	s := &v.state
	if s.pushActive || s.setActive || s.draining || s.incoming != (Polynomial{}) {
		return ErrPolynomialPending
	}
	old, e := s.values.size(v.owner)
	if e != nil {
		return e
	}
	if old != 0 {
		return ErrPolynomialPending
	}
	cap, e := s.values.capacity(v.owner)
	if e != nil {
		return e
	}
	for n > cap {
		if e = s.values.expand(v.owner); e != nil {
			return e
		}
		cap, e = s.values.capacity(v.owner)
		if e != nil {
			return e
		}
	}
	if n == 0 {
		return nil
	}
	if e = s.values.publishSize(v.owner, n); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if e = s.values.putMonomial(v.owner, i, Monomial{}); e != nil {
			return e
		}
		s.values.initialized = i + 1
	}
	return nil
}
