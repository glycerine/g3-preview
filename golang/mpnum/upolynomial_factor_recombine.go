// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
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
func (s *UPolynomialFactorizationService) factorSquareFree(f UPolynomialView, fs *UPolynomialFactors, k uint32, params UPolynomialFactorParams) (bool, error) {
	m := s.manager
	nm := m.modular
	if modularSourceDebug {
		if e := s.primitiveDiagnostic(f); e != nil {
			return false, e
		}
	}
	var fpp upolyVectorState
	defer m.dropLocalVector(&fpp)
	if e := m.setView(&fpp, f); e != nil {
		return false, e
	}
	if e := s.flipLeadingIfNegative(&fpp, fs, k); e != nil {
		return false, e
	}
	var p Int
	defer nm.Del(&p)
	if e := nm.SetInt32(&p, 2); e != nil {
		return false, e
	}
	zp, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	defer zp.Close()
	if e = zp.SetZP(&p); e != nil {
		return false, e
	}
	best, e := zp.NewFactors()
	if e != nil {
		return false, e
	}
	defer best.Close()
	var bestP Int
	defer nm.Del(&bestP)
	if e = nm.SetInt32(&bestP, 2); e != nil {
		return false, e
	}
	var ds upolyFactorDegreeState
	defer ds.close(m.config.Integer.budget)
	// Explicit construction adapter for the ill-formed C++ degree_set(zp_upm):
	// run the existing factors constructor on its already created EMPTY zp_fs.
	// That source body pushes true and gives max_degree()==0 for first swap.
	if e = ds.init(best); e != nil {
		return false, e
	}
	iterator, e := NewPrimeIterator(s.config.Primes)
	if e != nil {
		return false, e
	}
	var gcdTmp Int
	defer nm.Del(&gcdTmp)
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
			return false, fs.PushCopy(m.stateView(&fpp), k)
		}
		if e = nm.SetUint64(&p, next); e != nil {
			return false, e
		}
		if e = zp.SetZP(&p); e != nil {
			return false, e
		}
		top, e := m.read(m.stateView(&fpp), fpp.size-1)
		if e != nil {
			return false, e
		}
		if e = nm.GCD(&p, top, &gcdTmp); e != nil {
			return false, e
		}
		one, e := nm.IsOne(&gcdTmp)
		if e != nil {
			return false, e
		}
		if !one {
			continue
		}
		route, e := s.factorPrimeTrial(zp, &fpp, fs, best, k, &p, &bestP, &ds, &trials)
		if e != nil {
			return false, e
		}
		if route == 2 {
			return true, nil
		}
	}
	if e = zp.SetZP(&bestP); e != nil {
		return false, e
	}
	exponent, e := s.mignotteBound(m.stateView(&fpp), &bestP)
	if e != nil {
		return false, e
	}
	zpe, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	defer zpe.Close()
	if e = zpe.SetZP(&bestP); e != nil {
		return false, e
	}
	lifted, e := zpe.NewFactors()
	if e != nil {
		return false, e
	}
	defer lifted.Close()
	if e = s.henselFactors(m.stateView(&fpp), best, exponent, lifted); e != nil {
		return false, e
	}
	var leading Int
	defer nm.Del(&leading)
	top, e := m.read(m.stateView(&fpp), fpp.size-1)
	if e != nil {
		return false, e
	}
	if e = zpe.modular.Copy(&leading, top); e != nil {
		return false, e
	}
	if e = m.mulScalarState(&fpp, &leading); e != nil {
		return false, e
	}
	var it upolyFactorCombinationState
	it.factors = lifted
	defer it.close()
	if e = it.init(lifted, &ds); e != nil {
		return false, e
	}
	var trial, quotient upolyVectorState
	defer m.dropLocalVector(&trial)
	defer m.dropLocalVector(&quotient)
	var trialContent Int
	defer nm.Del(&trialContent)
	result, remove := true, false
	counter := uint32(0)
	for {
		more, e := it.next(remove)
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
		degree, e := it.degree()
		if e != nil {
			return false, e
		}
		left := degree <= best.totalDegree/2
		used, e := s.selectFactorTrial(&it, &leading, &fpp, &trial, left)
		if e != nil {
			return false, e
		}
		if !used {
			remove = false
			continue
		}
		if e = zpe.mulScalarState(&trial, &leading); e != nil {
			return false, e
		}
		exact, e := m.exactDivState(m.stateView(&fpp), m.stateView(&trial), &quotient)
		if e != nil {
			return false, e
		}
		if exact {
			if !left {
				trial, quotient = quotient, trial
			}
			if e = m.primitiveState(m.stateView(&trial), &trial, &trialContent); e != nil {
				return false, e
			}
			if e = fs.PushCopy(m.stateView(&trial), k); e != nil {
				return false, e
			}
			if e = m.primitiveState(m.stateView(&quotient), &fpp, &trialContent); e != nil {
				return false, e
			}
			top, e := m.read(m.stateView(&fpp), fpp.size-1)
			if e != nil {
				return false, e
			}
			if e = nm.Copy(&leading, top); e != nil {
				return false, e
			}
			if e = m.mulScalarState(&fpp, &leading); e != nil {
				return false, e
			}
			remove = true
		} else {
			remove = false
		}
	}
	if fpp.size > 1 {
		if e = m.divScalar(m.stateView(&fpp), &leading); e != nil {
			return false, e
		}
		if e = fs.PushCopy(m.stateView(&fpp), k); e != nil {
			return false, e
		}
	} else if modularSourceDebug {
		if fpp.size != 1 {
			return false, ErrUPolynomialAssertion
		}
		one, e := nm.IsOne(&m.values(&fpp)[fpp.size-1])
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
