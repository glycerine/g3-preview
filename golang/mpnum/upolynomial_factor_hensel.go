// Licensed under the MIT license.
package mpnum

import "math"

// Both literal to_zp_manager overloads, with reads at each source dereference.
func (m *UPolynomialManager) toZPInPlace(p *upolyVectorState) error {
	for i := uint32(0); i < p.size; i++ {
		a, e := m.read(m.stateView(p), i)
		if e != nil {
			return e
		}
		if e = m.modular.PNormalize(a); e != nil {
			return e
		}
	}
	return m.trim(p)
}
func (m *UPolynomialManager) toZPCopy(p UPolynomialView, out *upolyVectorState) error {
	if e := m.reset(out); e != nil {
		return e
	}
	for i := uint32(0); i < p.count; i++ {
		var local Int
		a, e := m.read(p, i)
		if e != nil {
			return e
		}
		if e = m.modular.Copy(&local, a); e != nil {
			m.modular.Del(&local)
			return e
		}
		target, e := m.pushDefault(out)
		if e != nil {
			m.modular.Del(&local)
			return e
		}
		*target = local // literal nontrivial mpz move, no coefficient copy or replay
	}
	return m.trim(out)
}
func (s *UPolynomialFactorizationService) HenselLiftPair(a, b, r *Int, u, A, v, B, C UPolynomialView, outA, outB UPolynomial) error {
	if e := s.check(); e != nil {
		return e
	}
	m := s.manager
	da, e := m.descriptor(outA)
	if e != nil {
		return e
	}
	db, e := m.descriptor(outB)
	if e != nil {
		return e
	}
	return s.henselPair(a, b, r, u, A, v, B, C, &da.state, &db.state)
}
func (s *UPolynomialFactorizationService) henselPair(a, b, r *Int, u, A, v, B, C UPolynomialView, outA, outB *upolyVectorState) error {
	m := s.manager
	rm, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return e
	}
	defer rm.Close()
	if e = rm.modular.SetZP(r); e != nil {
		return e
	}
	if modularSourceDebug {
		if upolyViewDegree(C) != upolyViewDegree(A)+upolyViewDegree(B) || upolyViewDegree(u) >= upolyViewDegree(B) || upolyViewDegree(v) >= upolyViewDegree(A) {
			return ErrUPolynomialAssertion
		}
	}
	var f, Vf, t, S, T, tmp upolyVectorState
	defer m.dropLocalVector(&f)
	if e = m.mul(A, B, &f); e != nil {
		return e
	}
	if e = m.addSubState(C, m.stateView(&f), &f, true); e != nil {
		return e
	}
	if e = m.divScalar(m.stateView(&f), b); e != nil {
		return e
	}
	if e = rm.toZPInPlace(&f); e != nil {
		return e
	}
	defer rm.dropLocalVector(&Vf)
	defer rm.dropLocalVector(&t)
	defer rm.dropLocalVector(&S)
	if e = rm.mul(v, rm.stateView(&f), &Vf); e != nil {
		return e
	}
	var d uint32
	if e = rm.divRemCore(rm.stateView(&Vf), A, &d, &t, &S); e != nil {
		return e
	}
	defer rm.dropLocalVector(&T)
	defer rm.dropLocalVector(&tmp)
	if e = rm.mul(u, rm.stateView(&f), &T); e != nil {
		return e
	}
	if e = rm.mul(B, rm.stateView(&t), &tmp); e != nil {
		return e
	}
	if e = rm.addSubState(rm.stateView(&T), rm.stateView(&tmp), &T, false); e != nil {
		return e
	}
	if e = m.mulScalarState(&S, b); e != nil {
		return e
	}
	if e = m.mulScalarState(&T, b); e != nil {
		return e
	}
	if e = m.addSubState(A, m.stateView(&S), outA, false); e != nil {
		return e
	}
	if e = m.addSubState(B, m.stateView(&T), outB, false); e != nil {
		return e
	}
	if modularSourceDebug && s.config.FactorizationDebugTag {
		ok, e := s.checkHenselPair(C, a, b, r, A, B, m.stateView(outA), m.stateView(outB))
		if e != nil {
			return e
		}
		if !ok {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (s *UPolynomialFactorizationService) henselQuadratic(C UPolynomialView, zpe *UPolynomialManager, A, B *upolyVectorState, e uint32) error {
	m := s.manager
	zp, err := NewRelatedUPolynomialManager(m)
	if err != nil {
		return err
	}
	defer zp.Close()
	p, err := zpe.modular.borrowedModulus()
	if err != nil {
		return err
	}
	if err = zp.modular.SetZP(p); err != nil {
		return err
	}
	var U, V, D, Al, Bl upolyVectorState
	defer m.dropLocalVector(&U)
	defer m.dropLocalVector(&V)
	defer m.dropLocalVector(&D)
	if err = zp.extendedGCD(zp.stateView(A), zp.stateView(B), &U, &V, &D); err != nil {
		return err
	}
	if modularSourceDebug {
		if D.size != 1 {
			return ErrUPolynomialAssertion
		}
		one, err := zp.modular.IsOne(&zp.values(&D)[0])
		if err != nil {
			return err
		}
		if !one {
			return ErrUPolynomialAssertion
		}
	}
	defer m.dropLocalVector(&Al)
	defer m.dropLocalVector(&Bl)
	for k := uint32(1); k < e; {
		if err = m.checkpoint(); err != nil {
			return err
		}
		pe, err := zpe.modular.borrowedModulus()
		if err != nil {
			return err
		}
		if err = s.henselPair(pe, pe, pe, m.stateView(&U), m.stateView(A), m.stateView(&V), m.stateView(B), C, &Al, &Bl); err != nil {
			return err
		}
		if err = s.henselQuadraticIteration(zpe, pe, &U, &V, A, B, &Al, &Bl); err != nil {
			return err
		}
		if k > math.MaxUint32/2 {
			return ErrUPolynomialNontermination
		}
		k *= 2
	}
	return nil
}
func (s *UPolynomialFactorizationService) henselQuadraticIteration(zpe *UPolynomialManager, pe *Int, U, V, A, B, Al, Bl *upolyVectorState) error {
	m := s.manager
	var tmp1, g, S, T, t, tmp2 upolyVectorState
	defer m.dropLocalVector(&tmp1)
	defer m.dropLocalVector(&g)
	zero := NewInt(0)
	if e := m.scopedPush(&g, &zero); e != nil {
		return e
	}
	if e := m.config.Integer.SetInt64(&m.values(&g)[g.size-1], 1); e != nil {
		return e
	}
	if e := m.mul(m.stateView(Al), m.stateView(U), &tmp1); e != nil {
		return e
	}
	if e := m.addSubState(m.stateView(&g), m.stateView(&tmp1), &g, true); e != nil {
		return e
	}
	if e := m.mul(m.stateView(Bl), m.stateView(V), &tmp1); e != nil {
		return e
	}
	if e := m.addSubState(m.stateView(&g), m.stateView(&tmp1), &g, true); e != nil {
		return e
	}
	if e := m.divScalar(m.stateView(&g), pe); e != nil {
		return e
	}
	if e := zpe.toZPInPlace(&g); e != nil {
		return e
	}
	defer m.dropLocalVector(&S)
	defer m.dropLocalVector(&T)
	defer m.dropLocalVector(&t)
	defer m.dropLocalVector(&tmp2)
	if e := zpe.mul(m.stateView(&g), m.stateView(V), &tmp1); e != nil {
		return e
	}
	var d uint32
	// A can be newly allocated by the prime core even when input factors
	// borrow a fourth store. Bind its actual origin at this lifted-core read.
	if e := zpe.adoptState(m, A); e != nil {
		return e
	}
	if e := zpe.divRemCore(m.stateView(&tmp1), m.stateView(A), &d, &t, &T); e != nil {
		return e
	}
	if e := zpe.mul(m.stateView(&g), m.stateView(U), &tmp1); e != nil {
		return e
	}
	if e := zpe.adoptState(m, B); e != nil {
		return e
	}
	if e := zpe.mul(m.stateView(&t), m.stateView(B), &tmp2); e != nil {
		return e
	}
	if e := zpe.addSubState(m.stateView(&tmp1), m.stateView(&tmp2), &S, false); e != nil {
		return e
	}
	if e := m.mulScalarState(&S, pe); e != nil {
		return e
	}
	if e := m.mulScalarState(&T, pe); e != nil {
		return e
	}
	if e := m.addSubState(m.stateView(U), m.stateView(&S), U, false); e != nil {
		return e
	}
	if e := m.addSubState(m.stateView(V), m.stateView(&T), V, false); e != nil {
		return e
	}
	if e := zpe.modular.SetPSquare(); e != nil {
		return e
	}
	for _, vec := range [4]*upolyVectorState{U, V, Al, Bl} {
		if e := zpe.toZPInPlace(vec); e != nil {
			return e
		}
	}
	*A, *Al = *Al, *A
	*B, *Bl = *Bl, *B
	if modularSourceDebug && s.config.FactorizationDebugTag {
		ok, e := s.checkQuadraticHensel(zpe, m.stateView(U), m.stateView(A), m.stateView(V), m.stateView(B))
		if e != nil {
			return e
		}
		if !ok {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (s *UPolynomialFactorizationService) HenselLiftFactors(f UPolynomialView, pf *UPolynomialFactors, e uint32, lf *UPolynomialFactors) error {
	if err := s.check(); err != nil {
		return err
	}
	if err := pf.check(); err != nil {
		return err
	}
	if err := lf.check(); err != nil {
		return err
	}
	if pf.manager.config.Integer != s.manager.config.Integer || lf.manager.config.Integer != s.manager.config.Integer {
		return ErrUPolynomialOwner
	}
	// Explicit physical provenance adapter for actual independent cores. These
	// cold retain prefixes precede the source numeric body, without copying
	// coefficients or changing either source arithmetic manager's modulus.
	m, prime, lift := s.manager, pf.manager, lf.manager
	for _, binding := range [4]struct {
		receiver *UPolynomialManager
		store    *UPolynomialCoefficientStore
	}{
		{m, prime.store}, {m, lift.store}, {prime, m.store}, {lift, m.store},
	} {
		if err := binding.receiver.BorrowCoefficientStore(binding.store); err != nil {
			return err
		}
	}
	for i := uint32(0); i < pf.count; i++ {
		if err := m.adoptState(prime, &pf.factors[i]); err != nil {
			return err
		}
		if err := lift.adoptState(prime, &pf.factors[i]); err != nil {
			return err
		}
	}
	for i := uint32(0); i < lf.count; i++ {
		if err := m.adoptState(lift, &lf.factors[i]); err != nil {
			return err
		}
	}
	return s.henselFactors(f, pf, e, lf)
}
func (s *UPolynomialFactorizationService) henselFactors(f UPolynomialView, pf *UPolynomialFactors, e uint32, lf *UPolynomialFactors) error {
	m := s.manager
	zp := pf.manager
	zpe := lf.manager
	if modularSourceDebug {
		if pf.totalFactors == 0 {
			return ErrUPolynomialAssertion
		}
		one, err := m.config.Integer.IsOne(&pf.constant)
		if err != nil {
			return err
		}
		if !one {
			return ErrUPolynomialAssertion
		}
	}
	p, err := zp.modular.borrowedModulus()
	if err != nil {
		return err
	}
	if err = zpe.modular.SetZP(p); err != nil {
		return err
	}
	var A, B, C, parts upolyVectorState
	defer m.dropLocalVector(&A)
	defer m.dropLocalVector(&B)
	defer m.dropLocalVector(&C)
	defer m.dropLocalVector(&parts)
	if err = m.setView(&parts, f); err != nil {
		return err
	}
	// Source converts unsigned count-1 to signed int for loop endpoint.
	end := int32(pf.count - 1)
	for i := int32(0); i < end; i++ {
		if modularSourceDebug {
			mult, err := pf.multiplicity(uint32(i))
			if err != nil {
				return err
			}
			if mult != 1 {
				return ErrUPolynomialAssertion
			}
		}
		if err = zp.setView(&A, zp.stateView(&pf.factors[i])); err != nil {
			return err
		}
		if i > 0 {
			// parts was grown by the lifted core in the prior source division.
			// Adopt its actual physical origin at this subsequent cross-core read.
			if err = zp.adoptState(m, &parts); err != nil {
				return err
			}
			if err = zp.toZPCopy(m.stateView(&parts), &C); err != nil {
				return err
			}
		} else {
			if err = s.henselInitialC(f, pf, &C); err != nil {
				return err
			}
		}
		if err = zp.divState(m.stateView(&C), m.stateView(&A), &B); err != nil {
			return err
		}
		p, err := zp.modular.borrowedModulus()
		if err != nil {
			return err
		}
		if err = zpe.modular.SetZP(p); err != nil {
			return err
		}
		if err = s.henselQuadratic(m.stateView(&parts), zpe, &A, &B, e); err != nil {
			return err
		}
		if modularSourceDebug && s.config.FactorizationDebugTag {
			ok, err := s.checkIndividualLift(zp, zp.stateView(&pf.factors[i]), zpe, m.stateView(&A))
			if err != nil {
				return err
			}
			if !ok {
				return ErrUPolynomialAssertion
			}
		}
		if i == 0 {
			if err = zpe.toZPCopy(f, &parts); err != nil {
				return err
			}
		}
		if err = zpe.divState(m.stateView(&parts), m.stateView(&A), &parts); err != nil {
			return err
		}
		if err = lf.pushSwap(m, &A, 1); err != nil {
			return err
		}
	}
	var lcInv Int
	defer m.config.Integer.Del(&lcInv)
	top, err := m.read(f, f.count-1)
	if err != nil {
		return err
	}
	if err = zpe.modular.Copy(&lcInv, top); err != nil {
		return err
	}
	if err = zpe.modular.Inv(&lcInv); err != nil {
		return err
	}
	if err = zpe.mulScalarState(&B, &lcInv); err != nil {
		return err
	}
	if err = lf.pushSwap(m, &B, 1); err != nil {
		return err
	}
	if modularSourceDebug && s.config.FactorizationDebugTag {
		ok, err := s.checkHenselFactors(f, pf, lf, e)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUPolynomialAssertion
		}
	}
	return nil
}
func (s *UPolynomialFactorizationService) henselInitialC(f UPolynomialView, pf *UPolynomialFactors, C *upolyVectorState) error {
	m := s.manager
	zp := pf.manager
	if e := pf.multiply(C); e != nil {
		return e
	}
	var lc Int
	defer m.config.Integer.Del(&lc)
	top, e := m.read(f, f.count-1)
	if e != nil {
		return e
	}
	if e = zp.modular.Copy(&lc, top); e != nil {
		return e
	}
	return zp.mulScalarState(C, &lc)
}
