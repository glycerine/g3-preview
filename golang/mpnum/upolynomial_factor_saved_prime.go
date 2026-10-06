// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175;
// src/util/mpzzp.h:122-123.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj).
package mpnum

// The post-loop SET uses the actual already-constructed saved-prime NUM and
// related manager. Metadata creates no source declaration or extra Copy.
func (s *UPolynomialFactorizationService) restoreSavedPrime(fs *UPolynomialFactors) (failure error) {
	if s == nil || s.scope == nil || s.scope.constructed != 7 || s.scope.zp == nil {
		return ErrUPolynomialOwner
	}
	q, e := s.beginPrimePreparation(fs)
	if e != nil {
		return e
	}
	r := q.receipt
	q.enterCall()
	defer q.leaveCall()
	defer func() { q.executing = false; failure = q.finish(&r, failure) }()
	c := q.owner
	return c.zp.modular.setZPWithAllocationContext(&c.bestP, c.zp.modular.primePreparationAllocationContext(q))
}

// Reached post-loop source preparation, including actual constructor frontiers.
func (s *UPolynomialFactorizationService) prepareFactorLift(fs *UPolynomialFactors) (failure error) {
	if s == nil || s.scope == nil || s.scope.constructed != 7 || s.scope.zp == nil {
		return ErrUPolynomialOwner
	}
	q, e := s.beginPrimePreparation(fs)
	if e != nil {
		return e
	}
	r := q.receipt
	q.enterCall()
	defer q.leaveCall()
	defer func() { q.executing = false; failure = q.finish(&r, failure) }()
	c, m := q.owner, q.owner.manager
	if e = c.zp.modular.setZPWithAllocationContext(&c.bestP, c.zp.modular.primePreparationAllocationContext(q)); e != nil {
		return e
	}
	exponent, e := s.mignotteBoundWithReceipt(m.stateView(&c.fpp), &c.bestP, &q.mignotte)
	if e != nil {
		return e
	}
	zpe, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return e
	}
	c.zpe = zpe
	c.constructed = 8
	if e = q.bindLift(zpe); e != nil {
		return e
	}
	if e = zpe.modular.setZPWithAllocationContext(&c.bestP, zpe.modular.primePreparationAllocationContext(q)); e != nil {
		return e
	}
	lifted, e := zpe.NewFactors()
	if e != nil {
		return e
	}
	c.lifted = lifted
	c.constructed = 9
	if e = s.beginHenselCaller(c); e != nil {
		return e
	}
	q.hensel = &c.hensel
	q.henselInvocation = c.hensel.invocation
	return s.henselFactorsWithCaller(m.stateView(&c.fpp), c.best, exponent, lifted, &c.hensel)
}
