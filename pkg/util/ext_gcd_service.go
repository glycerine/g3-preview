// Source: src/util/ext_gcd.h, concrete rational specialization.
package util

// ExtendedGCD translates the wrapper Euclidean algorithm, which differs from
// the manager's coefficient algorithm. Outputs publish in source order and
// retain any completed prefix on failure. Inputs are copied before outputs.
func (s *RationalService) ExtendedGCD(inA, inB, gcd, x, y *OwnedRational) (_lexErr error) {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	for _, v := range []*OwnedRational{inA, inB, gcd, x, y} {
		if err := s.check(v); err != nil {
			return err
		}
	}
	a := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	b := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	lastX := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	lastY := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	tmp := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	quotient := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}
	// Copy/value constructors are incomplete until their actual source bodies
	// return. Unreached future declarations have no source destructor.
	aComplete, bComplete, lastXComplete, lastYComplete, tmpComplete, quotientComplete := false, false, false, false, false, false
	defer s.endLexicalLocal(&a.Raw, &aComplete)
	defer s.endLexicalLocal(&b.Raw, &bComplete)
	defer s.endLexicalLocal(&lastX.Raw, &lastXComplete)
	defer s.endLexicalLocal(&lastY.Raw, &lastYComplete)
	defer s.endLexicalLocal(&tmp.Raw, &tmpComplete)
	defer s.endLexicalLocal(&quotient.Raw, &quotientComplete)
	if err := s.Manager.Copy(&a.Raw, &inA.Raw); err != nil {
		return err
	}
	aComplete = true
	if err := s.Manager.Copy(&b.Raw, &inB.Raw); err != nil {
		return err
	}
	bComplete = true
	// Source assignments construct a small rational then deep-copy it.
	zero, one := OwnedRational{Raw: NewRational(), ownerID: s.ownerID}, OwnedRational{Raw: Rational{Num: NewInt(1), Den: NewInt(1)}, ownerID: s.ownerID}
	if err := s.Manager.Copy(&x.Raw, &zero.Raw); err != nil {
		return err
	}
	if err := s.Manager.Copy(&y.Raw, &one.Raw); err != nil {
		return err
	}
	if err := s.Manager.SetInt64(&lastX.Raw, 1); err != nil {
		return err
	}
	lastXComplete = true
	if err := s.Manager.SetInt64(&lastY.Raw, 0); err != nil {
		return err
	}
	lastYComplete = true
	// These source default constructors complete before entering the loop.
	tmpComplete, quotientComplete = true, true
	for {
		isZero, err := s.Manager.IsZero(&b.Raw)
		if err != nil {
			return err
		}
		if isZero {
			break
		}
		if err := s.Manager.Copy(&tmp.Raw, &b.Raw); err != nil {
			return err
		}
		if err := s.extendedGCDIntegerLocked(&quotient, &a, &b, false); err != nil {
			return err
		}
		if err := s.extendedGCDIntegerLocked(&b, &a, &b, true); err != nil {
			return err
		}
		if err := s.Manager.Copy(&a.Raw, &tmp.Raw); err != nil {
			return err
		}
		if err := s.Manager.Copy(&tmp.Raw, &x.Raw); err != nil {
			return err
		}
		if err := s.extendedGCDCoefficientLocked(&quotient, x, &lastX); err != nil {
			return err
		}
		if err := s.Manager.Copy(&lastX.Raw, &tmp.Raw); err != nil {
			return err
		}
		if err := s.Manager.Copy(&tmp.Raw, &y.Raw); err != nil {
			return err
		}
		if err := s.extendedGCDCoefficientLocked(&quotient, y, &lastY); err != nil {
			return err
		}
		if err := s.Manager.Copy(&lastY.Raw, &tmp.Raw); err != nil {
			return err
		}
	}
	if err := s.Manager.Copy(&gcd.Raw, &a.Raw); err != nil {
		return err
	}
	if err := s.Manager.Copy(&x.Raw, &lastX.Raw); err != nil {
		return err
	}
	return s.Manager.Copy(&y.Raw, &lastY.Raw)
}

// div/mod construct a completed default rational before manager arithmetic,
// then assign its value; its destructor runs at the full-expression boundary.
func (s *RationalService) extendedGCDIntegerLocked(dst, a, b *OwnedRational, modulo bool) (_lexErr error) {
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	result := NewRational()
	defer s.destroyLexicalRational(&result)
	var e error
	if modulo {
		e = s.Manager.Mod(&a.Raw, &b.Raw, &result)
	} else {
		e = s.Manager.IDiv(&a.Raw, &b.Raw, &result)
	}
	if e != nil {
		return e
	}
	return s.Manager.Copy(&dst.Raw, &result)
}

// The coefficient product and difference are completed returned values only
// after each free operator succeeds. Failed inner COPY storage waits for the
// caller's six completed locals to unwind, then owning abort runs separately.
func (s *RationalService) extendedGCDCoefficientLocked(quotient, current, last *OwnedRational) (_lexErr error) {
	if e := s.beginLexical(); e != nil {
		return e
	}
	defer s.endLexicalError(&_lexErr)
	product, e := s.facadeValueLocked(quotient, current, rationalFacadeMul)
	if e != nil {
		return e
	}
	defer s.destroyLexicalRational(&product.Raw)
	difference, e := s.facadeValueLocked(last, &product, rationalFacadeSub)
	if e != nil {
		return e
	}
	defer s.destroyLexicalRational(&difference.Raw)
	return s.Manager.Copy(&current.Raw, &difference.Raw)
}
