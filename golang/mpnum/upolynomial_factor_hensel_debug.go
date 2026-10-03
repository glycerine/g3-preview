// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Source runtime CASSERT tag helpers, independent of g3debug build selection.
package mpnum

func (s *UPolynomialFactorizationService) checkHenselPair(C UPolynomialView, a, b, r *Int, A, B, Al, Bl UPolynomialView) (bool, error) {
	m := s.manager
	var br Int
	defer m.config.Integer.Del(&br)
	if e := m.config.Integer.Mul(b, r, &br); e != nil {
		return false, e
	}
	brCore, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	defer brCore.Close()
	if e = brCore.SetZP(&br); e != nil {
		return false, e
	}
	if Al.count != A.count {
		return false, nil
	}
	if Bl.count != B.count {
		return false, nil
	}
	atop, e := m.read(A, A.count-1)
	if e != nil {
		return false, e
	}
	ltop, e := m.read(Al, Al.count-1)
	if e != nil {
		return false, e
	}
	same, e := m.config.Integer.Equal(atop, ltop)
	if e != nil || !same {
		return false, e
	}
	var test1 upolyVectorState
	defer m.dropLocalVector(&test1)
	if e = m.mul(Al, Bl, &test1); e != nil {
		return false, e
	}
	if e = m.addSubState(C, m.stateView(&test1), &test1, true); e != nil {
		return false, e
	}
	if e = brCore.toZPInPlace(&test1); e != nil {
		return false, e
	}
	if test1.size != 0 {
		return false, nil
	}
	bCore, e := NewRelatedUPolynomialManager(m)
	if e != nil {
		return false, e
	}
	defer bCore.Close()
	if e = bCore.SetZP(b); e != nil {
		return false, e
	}
	var test2a, test2b upolyVectorState
	defer m.dropLocalVector(&test2a)
	defer m.dropLocalVector(&test2b)
	if e = bCore.toZPCopy(A, &test2a); e != nil {
		return false, e
	}
	if e = bCore.toZPCopy(Al, &test2b); e != nil {
		return false, e
	}
	same, e = m.EqualViews(m.stateView(&test2a), m.stateView(&test2b))
	if e != nil || !same {
		return false, e
	}
	var test3a, test3b upolyVectorState
	defer m.dropLocalVector(&test3a)
	defer m.dropLocalVector(&test3b)
	if e = bCore.toZPCopy(B, &test3a); e != nil {
		return false, e
	}
	if e = bCore.toZPCopy(Bl, &test3b); e != nil {
		return false, e
	}
	return m.EqualViews(m.stateView(&test3a), m.stateView(&test3b))
}
func (s *UPolynomialFactorizationService) checkQuadraticHensel(zpe *UPolynomialManager, U, A, V, B UPolynomialView) (bool, error) {
	m := s.manager
	var tmp1, tmp2, one upolyVectorState
	defer m.dropLocalVector(&tmp1)
	defer m.dropLocalVector(&tmp2)
	defer m.dropLocalVector(&one)
	if e := zpe.mul(U, A, &tmp1); e != nil {
		return false, e
	}
	if e := zpe.mul(V, B, &tmp2); e != nil {
		return false, e
	}
	if e := zpe.addSubState(m.stateView(&tmp1), m.stateView(&tmp2), &one, false); e != nil {
		return false, e
	}
	if one.size != 1 {
		return false, nil
	}
	return m.config.Integer.IsOne(&m.values(&one)[0])
}
func (s *UPolynomialFactorizationService) checkIndividualLift(zp *UPolynomialManager, A UPolynomialView, zpe *UPolynomialManager, Al UPolynomialView) (bool, error) {
	var converted upolyVectorState
	defer zp.dropLocalVector(&converted)
	if e := zp.toZPCopy(Al, &converted); e != nil {
		return false, e
	}
	return zp.EqualViews(A, zp.stateView(&converted))
}
func (s *UPolynomialFactorizationService) checkHenselFactors(f UPolynomialView, pf, lf *UPolynomialFactors, e uint32) (bool, error) {
	m := s.manager
	zp, zpe := pf.manager, lf.manager
	p, err := zp.modular.borrowedModulus()
	if err != nil {
		return false, err
	}
	pe, err := zpe.modular.borrowedModulus()
	if err != nil {
		return false, err
	}
	var power Int
	defer m.modular.Del(&power)
	if err = m.modular.Power(p, e, &power); err != nil {
		return false, err
	}
	ge, err := m.modular.GreaterEqual(pe, &power)
	if err != nil || !ge {
		return false, err
	}
	var multp, fp upolyVectorState
	defer m.dropLocalVector(&multp)
	defer m.dropLocalVector(&fp)
	if err = pf.multiply(&multp); err != nil {
		return false, err
	}
	if err = zp.toZPCopy(f, &fp); err != nil {
		return false, err
	}
	top, err := m.read(m.stateView(&fp), fp.size-1)
	if err != nil {
		return false, err
	}
	if err = zp.mulScalarState(&multp, top); err != nil {
		return false, err
	}
	same, err := m.EqualViews(m.stateView(&multp), m.stateView(&fp))
	if err != nil || !same {
		return false, err
	}
	if lf.count != pf.count {
		return false, nil
	}
	var multpe, fpe upolyVectorState
	defer m.dropLocalVector(&multpe)
	defer m.dropLocalVector(&fpe)
	if err = lf.multiply(&multpe); err != nil {
		return false, err
	}
	if err = zpe.toZPCopy(f, &fpe); err != nil {
		return false, err
	}
	top, err = m.read(m.stateView(&fpe), fpe.size-1)
	if err != nil {
		return false, err
	}
	if err = zpe.mulScalarState(&multpe, top); err != nil {
		return false, err
	}
	return m.EqualViews(m.stateView(&multpe), m.stateView(&fpe))
}
