// Licensed under the MIT license.
// Actual Mignotte bound, prime image trials and subset recombination.
package mpnum

import "math"

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
func (s *UPolynomialFactorizationService) mignotteBound(f UPolynomialView, p *Int) (uint32, error) {
	m := s.manager
	nm := m.modular
	if modularSourceDebug && upolyViewDegree(f) < 2 {
		return 0, ErrUPolynomialAssertion
	}
	n := upolyViewDegree(f) / 2
	var norm, bound, tmp Int
	defer nm.Del(&norm)
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
			if e = nm.AddMul(&norm, a, a, &norm); e != nil {
				return 0, e
			}
		}
	}
	if _, e := nm.RootInPlace(&norm, 2); e != nil {
		return 0, e
	}
	defer nm.Del(&bound)
	if e := nm.SetInt32(&bound, 1); e != nil {
		return 0, e
	}
	if e := nm.Mul2k(&bound, n, &bound); e != nil {
		return 0, e
	}
	defer nm.Del(&tmp)
	top, e := m.read(f, f.count-1)
	if e != nil {
		return 0, e
	}
	if e = nm.Copy(&tmp, top); e != nil {
		return 0, e
	}
	if e = nm.Abs(&tmp); e != nil {
		return 0, e
	}
	if e = nm.Add(&norm, &tmp, &norm); e != nil {
		return 0, e
	}
	if e = nm.Mul(&bound, &norm, &bound); e != nil {
		return 0, e
	}
	if e = nm.Copy(&tmp, p); e != nil {
		return 0, e
	}
	exponent := uint32(1)
	for {
		le, e := nm.LessEqual(&tmp, &bound)
		if e != nil {
			return 0, e
		}
		if !le {
			return exponent, nil
		}
		if e = nm.Mul(&tmp, &tmp, &tmp); e != nil {
			return 0, e
		}
		if exponent > math.MaxUint32/2 {
			return 0, ErrUPolynomialNontermination
		}
		exponent *= 2
	}
}

// Trial result: 0 continue (bad prime), 1 completed ordinary candidate,
// 2 source early return true (irreducible/trivial degree set).
func (s *UPolynomialFactorizationService) factorPrimeTrial(zp *UPolynomialManager, fpp *upolyVectorState, fs, best *UPolynomialFactors, k uint32, p, bestP *Int, ds *upolyFactorDegreeState, trials *uint32) (uint8, error) {
	m := s.manager
	var fzp upolyVectorState
	defer m.dropLocalVector(&fzp)
	if e := zp.toZPCopy(m.stateView(fpp), &fzp); e != nil {
		return 0, e
	}
	squareFree, e := zp.IsSquareFreeView(zp.stateView(&fzp))
	if e != nil {
		return 0, e
	}
	if !squareFree {
		return 0, nil
	}
	if e = zp.monicLocal(zp.stateView(&fzp)); e != nil {
		return 0, e
	}
	current, e := zp.NewFactors()
	if e != nil {
		return 0, e
	}
	defer current.Close()
	factored, e := zp.factorZPBerlekamp(zp.stateView(&fzp), current, false)
	if e != nil {
		return 0, e
	}
	if !factored {
		return 2, fs.PushCopy(m.stateView(fpp), k)
	}
	var imageDegrees upolyFactorDegreeState
	defer imageDegrees.close(m.config.Integer.budget)
	if e = imageDegrees.init(current); e != nil {
		return 0, e
	}
	if ds.bits-1 == 0 {
		*ds, imageDegrees = imageDegrees, *ds
	} else {
		if e = ds.intersect(&imageDegrees); e != nil {
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
		if e = m.modular.Copy(bestP, p); e != nil {
			return 0, e
		}
	}
	return 1, nil
}
func (s *UPolynomialFactorizationService) factorSquareFree(f UPolynomialView, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (completed bool, failure error) {
	m := s.manager
	nm := m.modular
	if modularSourceDebug {
		if e := s.primitiveDiagnostic(f); e != nil {
			return false, e
		}
	}
	c, e := s.newSquareFreeScope()
	if e != nil {
		return false, e
	}
	defer func() {
		if e := s.CleanupRetained(); e != nil && failure == nil {
			failure = e
		}
	}()
	c.constructed = 1
	if e := m.setView(&c.fpp, f); e != nil {
		return false, e
	}
	if e := s.flipLeadingIfNegative(&c.fpp, fs, k); e != nil {
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
	if e = s.henselFactors(m.stateView(&c.fpp), best, exponent, lifted); e != nil {
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
		if e = zpe.mulScalarState(&c.trial, &c.leading); e != nil {
			return false, e
		}
		exact, e := m.exactDivState(m.stateView(&c.fpp), m.stateView(&c.trial), &c.quotient)
		if e != nil {
			return false, e
		}
		if exact {
			if !left {
				c.trial, c.quotient = c.quotient, c.trial
			}
			if e = m.primitiveState(m.stateView(&c.trial), &c.trial, &c.trialContent); e != nil {
				return false, e
			}
			if e = fs.PushCopy(m.stateView(&c.trial), k); e != nil {
				return false, e
			}
			if e = m.primitiveState(m.stateView(&c.quotient), &c.fpp, &c.trialContent); e != nil {
				return false, e
			}
			top, e := m.read(m.stateView(&c.fpp), c.fpp.size-1)
			if e != nil {
				return false, e
			}
			if e = nm.Copy(&c.leading, top); e != nil {
				return false, e
			}
			if e = m.mulScalarState(&c.fpp, &c.leading); e != nil {
				return false, e
			}
			remove = true
		} else {
			remove = false
		}
	}
	if c.fpp.size > 1 {
		if e = m.divScalar(m.stateView(&c.fpp), &c.leading); e != nil {
			return false, e
		}
		if e = fs.PushCopy(m.stateView(&c.fpp), k); e != nil {
			return false, e
		}
	} else if modularSourceDebug {
		if c.fpp.size != 1 {
			return false, ErrUPolynomialAssertion
		}
		one, e := nm.IsOne(&m.values(&c.fpp)[c.fpp.size-1])
		if e != nil {
			return false, e
		}
		if !one {
			return false, ErrUPolynomialAssertion
		}
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
func (s *UPolynomialFactorizationService) selectFactorTrial(it *upolyFactorCombinationState, leading *Int, fpp, trial *upolyVectorState, left bool) (bool, error) {
	m := s.manager
	var tmp Int
	defer m.modular.Del(&tmp)
	var e error
	if left {
		e = it.leftTail(leading, &tmp)
	} else {
		e = it.right(leading, &tmp, nil)
	}
	if e != nil {
		return false, e
	}
	first, e := m.read(m.stateView(fpp), 0)
	if e != nil {
		return false, e
	}
	divides, e := m.modular.Divides(&tmp, first)
	if e != nil || !divides {
		return false, e
	}
	if left {
		return true, it.left(trial)
	}
	return true, it.right(nil, nil, trial)
}
