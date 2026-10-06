// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1299.
// Source author: Leonardo de Moura.
// Licensed under the MIT license.
// Actual Mignotte bound, prime image trials and subset recombination.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

func (s *UPolynomialFactorizationService) FactorSquareFree(f UPolynomialView, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (bool, error) {
	if e := s.check(); e != nil {
		return false, e
	}
	if e := s.manager.checkFactors(fs); e != nil {
		return false, e
	}
	return s.factorSquareFree(f, fs, k, params)
}
func (s *UPolynomialFactorizationService) FactorSquareFreeDefault(f UPolynomialView, fs *UPolynomialFactors, params UPolynomialFactorParams) (bool, error) {
	return s.FactorSquareFree(f, fs, 1, params)
}
func (s *UPolynomialFactorizationService) mignotteBound(f UPolynomialView, p *Int) (answer uint32, cause error) {
	m := s.manager
	nm := m.modular
	if modularSourceDebug && upolyViewDegree(f) < 2 {
		return 0, ErrUPolynomialAssertion
	}
	n := upolyViewDegree(f) / 2
	r, e := s.beginMignotteInvocation()
	if e != nil {
		return 0, e
	}
	q := r.frame
	q.constructed = 1
	defer func() { q.executing = false; cause = s.finishMignotteSource(&r, cause) }()
	norm, bound, tmp := &q.norm, &q.bound, &q.tmp
	for i := uint32(0); i < f.count; i++ {
		a, e := m.read(f, i)
		if e != nil {
			return 0, e
		}
		zero, e := nm.IsZero(a)
		if e != nil {
			return 0, e
		}
		if !zero {
			if e = s.mignotteNumericCall(&r, 0, a, a, norm, 0); e != nil {
				return 0, e
			}
		}
	}
	if e := s.mignotteNumericCall(&r, 1, nil, nil, norm, 2); e != nil {
		return 0, e
	}
	q.constructed = 2
	if e := s.mignotteNumericCall(&r, 2, nil, nil, bound, 1); e != nil {
		return 0, e
	}
	if e := s.mignotteNumericCall(&r, 3, bound, nil, bound, n); e != nil {
		return 0, e
	}
	q.constructed = 3
	top, e := m.read(f, f.count-1)
	if e != nil {
		return 0, e
	}
	if e = s.mignotteNumericCall(&r, 4, top, nil, tmp, 0); e != nil {
		return 0, e
	}
	if e = s.mignotteNumericCall(&r, 5, nil, nil, tmp, 0); e != nil {
		return 0, e
	}
	if e = s.mignotteNumericCall(&r, 6, norm, tmp, norm, 0); e != nil {
		return 0, e
	}
	if e = s.mignotteNumericCall(&r, 7, bound, norm, bound, 0); e != nil {
		return 0, e
	}
	if e = s.mignotteNumericCall(&r, 4, p, nil, tmp, 0); e != nil {
		return 0, e
	}
	exponent := uint32(1)
	for {
		le, e := nm.LessEqual(tmp, bound)
		if e != nil {
			return 0, e
		}
		if !le {
			return exponent, nil
		}
		if e = s.mignotteNumericCall(&r, 7, tmp, tmp, tmp, 0); e != nil {
			return 0, e
		}
		exponent *= 2
	}
}

// Trial result: 0 continue (bad prime), 1 completed ordinary candidate,
// 2 source early return true (irreducible/trivial degree set).
func (s *UPolynomialFactorizationService) factorPrimeTrial(zp *UPolynomialManager, fpp *upolyVectorState, fs, best *UPolynomialFactors, k uint32, p, bestP *Int, ds *upolyFactorDegreeState, trials *uint32) (route uint8, failure error) {
	m := s.manager
	r, e := s.beginPrimeTrialInvocation(zp)
	if e != nil {
		return 0, e
	}
	q := r.frame
	q.enterCall()
	defer q.leaveCall()
	defer func() { q.executing = false; failure = s.finishPrimeTrialSource(&r, failure) }()
	q.constructed = 1
	fzp := &q.fzp
	if e := zp.toZPCopyWithPrimeReceipt(m.stateView(fpp), fzp, &r); e != nil {
		return 0, e
	}
	squareFree, e := zp.IsSquareFreeView(zp.stateView(fzp))
	if e != nil {
		return 0, e
	}
	if !squareFree {
		return 0, nil
	}
	if e = zp.monicLocal(zp.stateView(fzp)); e != nil {
		return 0, e
	}
	current, e := zp.NewFactors()
	if e != nil {
		return 0, e
	}
	q.current = current
	q.constructed = 2
	factored, e := zp.factorZPBerlekamp(zp.stateView(fzp), current, false)
	if e != nil {
		return 0, e
	}
	if !factored {
		return 2, fs.PushCopy(m.stateView(fpp), k)
	}
	q.degreesStarted = true
	imageDegrees := &q.degrees
	if e = imageDegrees.init(current); e != nil {
		return 0, e
	}
	q.constructed = 3
	if ds.bits-1 == 0 {
		*ds, *imageDegrees = *imageDegrees, *ds
	} else {
		if e = ds.intersect(imageDegrees); e != nil {
			return 0, e
		}
	}
	trivial, e := ds.trivial()
	if e != nil {
		return 0, e
	}
	if trivial {
		return 2, fs.PushCopy(m.stateView(fpp), k)
	}
	*trials++
	if best.count == 0 || best.totalFactors > current.totalFactors {
		if e = best.Swap(current); e != nil {
			return 0, e
		}
		if e = m.modular.copyWithSourceContext(bestP, p, m.modular.primeTrialAllocationContext(&r)); e != nil {
			return 0, e
		}
	}
	return 1, nil
}
func (s *UPolynomialFactorizationService) factorSquareFree(f UPolynomialView, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (completed bool, failure error) {
	m := s.manager
	nm := m.modular
	if modularSourceDebug {
		if e := s.primitiveDiagnosticWithPreparation(f); e != nil {
			return false, e
		}
	}
	c, e := s.newSquareFreeScope()
	if e != nil {
		return false, e
	}
	defer func() {
		failure = s.finishSquareFreeScopeError(failure)
	}()
	c.constructed = 1
	if e := s.prepareSquareFree(f, fs, k); e != nil {
		return false, e
	}
	c.constructed = 2
	if e := nm.SetInt32(&c.p, 2); e != nil {
		return false, e
	}
	zp, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	c.zp = zp
	c.constructed = 3
	if e = zp.SetZP(&c.p); e != nil {
		return false, e
	}
	best, e := zp.NewFactors()
	if e != nil {
		return false, e
	}
	c.best = best
	c.constructed = 4
	c.constructed = 5
	if e = nm.SetInt32(&c.bestP, 2); e != nil {
		return false, e
	}
	c.constructed = 6
	// Explicit construction adapter for the ill-formed C++ degree_set(zp_upm):
	// run the existing factors constructor on its already created EMPTY zp_fs.
	// That source body pushes true and gives max_degree()==0 for first swap.
	if e = c.ds.init(best); e != nil {
		return false, e
	}
	iterator, e := NewPrimeIterator(s.config.Primes)
	if e != nil {
		return false, e
	}
	c.constructed = 7
	trials := uint32(0)
	for trials < params.PrimeTrials {
		if e = m.checkpoint(); e != nil {
			return false, e
		}
		next, e := iterator.Next()
		if e != nil {
			return false, e
		}
		if next > uint64(params.MaxPrime) {
			return false, fs.PushCopy(m.stateView(&c.fpp), k)
		}
		if e = nm.SetUint64(&c.p, next); e != nil {
			return false, e
		}
		if e = zp.SetZP(&c.p); e != nil {
			return false, e
		}
		top, e := m.read(m.stateView(&c.fpp), c.fpp.size-1)
		if e != nil {
			return false, e
		}
		if e = nm.GCD(&c.p, top, &c.gcdTmp); e != nil {
			return false, e
		}
		one, e := nm.IsOne(&c.gcdTmp)
		if e != nil {
			return false, e
		}
		if !one {
			continue
		}
		route, e := s.factorPrimeTrial(zp, &c.fpp, fs, best, k, &c.p, &c.bestP, &c.ds, &trials)
		if e != nil {
			return false, e
		}
		if route == 2 {
			return true, nil
		}
	}
	if e = zp.SetZP(&c.bestP); e != nil {
		return false, e
	}
	exponent, e := s.mignotteBound(m.stateView(&c.fpp), &c.bestP)
	if e != nil {
		return false, e
	}
	zpe, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	c.zpe = zpe
	c.constructed = 8
	if e = zpe.SetZP(&c.bestP); e != nil {
		return false, e
	}
	lifted, e := zpe.NewFactors()
	if e != nil {
		return false, e
	}
	c.lifted = lifted
	c.constructed = 9
	if e = s.beginHenselCaller(c); e != nil {
		return false, e
	}
	if e = s.henselFactorsWithCaller(m.stateView(&c.fpp), best, exponent, lifted, &c.hensel); e != nil {
		return false, e
	}
	c.constructed = 10
	top, e := m.read(m.stateView(&c.fpp), c.fpp.size-1)
	if e != nil {
		return false, e
	}
	if e = zpe.modular.Copy(&c.leading, top); e != nil {
		return false, e
	}
	if e = m.mulScalarState(&c.fpp, &c.leading); e != nil {
		return false, e
	}
	c.it.factors = lifted
	c.constructed = 11
	if e = c.it.init(lifted, &c.ds); e != nil {
		return false, e
	}
	c.constructed = 13
	c.constructed = 14
	result, remove := true, false
	counter := uint32(0)
	for {
		more, e := c.it.next(remove)
		if e != nil {
			return false, e
		}
		if !more {
			break
		}
		if e = m.checkpoint(); e != nil {
			return false, e
		}
		counter++
		if counter > params.MaxSearchSize {
			result = false
			break
		}
		degree, e := c.it.degree()
		if e != nil {
			return false, e
		}
		left := degree <= best.totalDegree/2
		used, e := s.selectFactorTrial(&c.it, &c.leading, &c.fpp, &c.trial, left)
		if e != nil {
			return false, e
		}
		if !used {
			remove = false
			continue
		}
		remove, e = s.publishRecombinationTrial(fs, k, left)
		if e != nil {
			return false, e
		}
	}
	if e := s.publishRecombinationRemainder(fs, k); e != nil {
		return false, e
	}
	return result, nil
}
func (s *UPolynomialFactorizationService) primitiveDiagnostic(f UPolynomialView) error {
	m := s.manager
	var content Int
	defer m.modular.Del(&content)
	if e := m.gcdSpan(f, &content); e != nil {
		return e
	}
	if f.count == 0 {
		return nil
	}
	one, e := m.modular.IsOne(&content)
	if e != nil {
		return e
	}
	if !one {
		return ErrUPolynomialAssertion
	}
	return nil
}
func (s *UPolynomialFactorizationService) selectFactorTrial(it *upolyFactorCombinationState, leading *Int, fpp, trial *upolyVectorState, left bool) (used bool, failure error) {
	q, e := s.beginRecombinationTrial(it.factors)
	if e != nil {
		return false, e
	}
	r := q.receipt
	q.enterCall()
	defer func() { q.executing = false; q.leaveCall(); failure = q.finish(&r, failure) }()
	context := it.factors.manager.modular.trialAllocationContext(q)
	if context.trialFrame(s.manager.config.Integer) == nil {
		return false, ErrUPolynomialOwner
	}
	if left {
		e = it.leftTailWithSourceContext(leading, &q.tmp, context)
	} else {
		e = it.rightWithSourceContext(leading, &q.tmp, nil, &q.addMul, context)
	}
	if e != nil {
		return false, e
	}
	first, e := s.manager.read(s.manager.stateView(fpp), 0)
	if e != nil {
		return false, e
	}
	divides, e := s.manager.modular.dividesWithAllocationContext(&q.tmp, first, s.manager.modular.trialAllocationContext(q))
	if e != nil || !divides {
		return false, e
	}
	if left {
		return true, it.leftWithSourceContext(trial, &q.addMul, context)
	}
	return true, it.rightWithSourceContext(nil, nil, trial, &q.addMul, context)
}

// Source: src/math/polynomial/upolynomial_factorization.cpp:1238-1273.
func (s *UPolynomialFactorizationService) publishRecombinationTrial(fs *UPolynomialFactors, k uint32, left bool) (remove bool, failure error) {
	q, e := s.beginRecombinationStep(fs)
	if e != nil {
		return false, e
	}
	r := q.receipt
	q.enterCall()
	defer func() { q.executing = false; q.leaveCall(); failure = q.finish(&r, failure) }()
	c := q.owner
	m := c.manager
	nm := m.modular
	zpe := q.related
	context := nm.recombinationAllocationContext(q)
	if e = zpe.mulScalarStateWithRecombination(&c.trial, &c.leading, q); e != nil {
		return false, e
	}
	exact, e := m.exactDivStateWithSourceContext(m.stateView(&c.fpp), m.stateView(&c.trial), &c.quotient, &q.addMul, context)
	if e != nil {
		return false, e
	}
	if exact {
		if !left {
			c.trial, c.quotient = c.quotient, c.trial
		}
		if e = m.primitiveStateWithSourceContext(m.stateView(&c.trial), &c.trial, &c.trialContent, context); e != nil {
			return false, e
		}
		if e = fs.pushCopyWithSourceContext(m.stateView(&c.trial), k, context); e != nil {
			return false, e
		}
		if e = m.primitiveStateWithSourceContext(m.stateView(&c.quotient), &c.fpp, &c.trialContent, context); e != nil {
			return false, e
		}
		top, e := m.read(m.stateView(&c.fpp), c.fpp.size-1)
		if e != nil {
			return false, e
		}
		if e = nm.copyWithSourceContext(&c.leading, top, context); e != nil {
			return false, e
		}
		if e = m.mulScalarStateWithRecombination(&c.fpp, &c.leading, q); e != nil {
			return false, e
		}
		return true, nil
	} else {
		return false, nil
	}
}

// Source: src/math/polynomial/upolynomial_factorization.cpp:1285-1292.
func (s *UPolynomialFactorizationService) publishRecombinationRemainder(fs *UPolynomialFactors, k uint32) (failure error) {
	q, e := s.beginRecombinationStep(fs)
	if e != nil {
		return e
	}
	r := q.receipt
	q.enterCall()
	defer func() { q.executing = false; q.leaveCall(); failure = q.finish(&r, failure) }()
	c := q.owner
	m := c.manager
	nm := m.modular
	context := m.modular.recombinationAllocationContext(q)
	if c.fpp.size > 1 {
		if e = m.divScalarWithSourceContext(m.stateView(&c.fpp), &c.leading, context); e != nil {
			return e
		}
		if e = fs.pushCopyWithSourceContext(m.stateView(&c.fpp), k, context); e != nil {
			return e
		}
	} else if modularSourceDebug {
		if c.fpp.size != 1 {
			return ErrUPolynomialAssertion
		}
		one, e := nm.IsOne(&m.values(&c.fpp)[c.fpp.size-1])
		if e != nil {
			return e
		}
		if !one {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
