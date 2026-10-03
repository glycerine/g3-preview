// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/ext_gcd.h, concrete rational specialization.
package mpnum

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
	a := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	b := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	lastX := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	lastY := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	tmp := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	quotient := OwnedRational{raw: NewRational(), ownerID: s.ownerID}
	// Copy/value constructors are incomplete until their actual source bodies
	// return. Unreached future declarations have no source destructor.
	aComplete, bComplete, lastXComplete, lastYComplete, tmpComplete, quotientComplete := false, false, false, false, false, false
	defer s.endLexicalLocal(&a.raw, &aComplete)
	defer s.endLexicalLocal(&b.raw, &bComplete)
	defer s.endLexicalLocal(&lastX.raw, &lastXComplete)
	defer s.endLexicalLocal(&lastY.raw, &lastYComplete)
	defer s.endLexicalLocal(&tmp.raw, &tmpComplete)
	defer s.endLexicalLocal(&quotient.raw, &quotientComplete)
	if err := s.manager.Copy(&a.raw, &inA.raw); err != nil {
		return err
	}
	aComplete = true
	if err := s.manager.Copy(&b.raw, &inB.raw); err != nil {
		return err
	}
	bComplete = true
	// Source assignments construct a small rational then deep-copy it.
	zero, one := OwnedRational{raw: NewRational(), ownerID: s.ownerID}, OwnedRational{raw: Rational{num: NewInt(1), den: NewInt(1)}, ownerID: s.ownerID}
	if err := s.manager.Copy(&x.raw, &zero.raw); err != nil {
		return err
	}
	if err := s.manager.Copy(&y.raw, &one.raw); err != nil {
		return err
	}
	if err := s.manager.SetInt64(&lastX.raw, 1); err != nil {
		return err
	}
	lastXComplete = true
	if err := s.manager.SetInt64(&lastY.raw, 0); err != nil {
		return err
	}
	lastYComplete = true
	// These source default constructors complete before entering the loop.
	tmpComplete, quotientComplete = true, true
	for {
		isZero, err := s.manager.IsZero(&b.raw)
		if err != nil {
			return err
		}
		if isZero {
			break
		}
		if err := s.manager.Copy(&tmp.raw, &b.raw); err != nil {
			return err
		}
		if err := s.extendedGCDIntegerLocked(&quotient, &a, &b, false); err != nil {
			return err
		}
		if err := s.extendedGCDIntegerLocked(&b, &a, &b, true); err != nil {
			return err
		}
		if err := s.manager.Copy(&a.raw, &tmp.raw); err != nil {
			return err
		}
		if err := s.manager.Copy(&tmp.raw, &x.raw); err != nil {
			return err
		}
		if err := s.extendedGCDCoefficientLocked(&quotient, x, &lastX); err != nil {
			return err
		}
		if err := s.manager.Copy(&lastX.raw, &tmp.raw); err != nil {
			return err
		}
		if err := s.manager.Copy(&tmp.raw, &y.raw); err != nil {
			return err
		}
		if err := s.extendedGCDCoefficientLocked(&quotient, y, &lastY); err != nil {
			return err
		}
		if err := s.manager.Copy(&lastY.raw, &tmp.raw); err != nil {
			return err
		}
	}
	if err := s.manager.Copy(&gcd.raw, &a.raw); err != nil {
		return err
	}
	if err := s.manager.Copy(&x.raw, &lastX.raw); err != nil {
		return err
	}
	return s.manager.Copy(&y.raw, &lastY.raw)
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
		e = s.manager.Mod(&a.raw, &b.raw, &result)
	} else {
		e = s.manager.IDiv(&a.raw, &b.raw, &result)
	}
	if e != nil {
		return e
	}
	return s.manager.Copy(&dst.raw, &result)
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
	defer s.destroyLexicalRational(&product.raw)
	difference, e := s.facadeValueLocked(last, &product, rationalFacadeSub)
	if e != nil {
		return e
	}
	defer s.destroyLexicalRational(&difference.raw)
	return s.manager.Copy(&current.raw, &difference.raw)
}
