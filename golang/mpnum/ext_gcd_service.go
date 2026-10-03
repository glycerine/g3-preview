// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/ext_gcd.h, concrete rational specialization.
package mpnum

// ExtendedGCD translates the wrapper Euclidean algorithm, which differs from
// the manager's coefficient algorithm. Outputs publish in source order and
// retain any completed prefix on failure. Inputs are copied before outputs.
func (s *RationalService) ExtendedGCD(inA, inB, gcd, x, y *OwnedRational) error {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
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
	// C++ locals are destroyed in reverse construction order, even on error.
	defer func() {
		for _, v := range []*OwnedRational{&quotient, &tmp, &lastY, &lastX, &b, &a} {
			s.manager.Del(&v.raw)
		}
	}()
	copy := func(dst, src *OwnedRational) error { return s.manager.Copy(&dst.raw, &src.raw) }
	if err := copy(&a, inA); err != nil {
		return err
	}
	if err := copy(&b, inB); err != nil {
		return err
	}
	// Source assignments construct a small rational then deep-copy it.
	zero, one := OwnedRational{raw: NewRational(), ownerID: s.ownerID}, OwnedRational{raw: Rational{num: NewInt(1), den: NewInt(1)}, ownerID: s.ownerID}
	if err := copy(x, &zero); err != nil {
		return err
	}
	if err := copy(y, &one); err != nil {
		return err
	}
	if err := s.manager.SetInt64(&lastX.raw, 1); err != nil {
		return err
	}
	// lastY, tmp and quotient were default-constructed as zero above.
	integerResult := func(dst *OwnedRational, modulo bool) error {
		result := NewRational()
		defer s.manager.Del(&result)
		var err error
		if modulo {
			err = s.manager.Mod(&a.raw, &b.raw, &result)
		} else {
			err = s.manager.IDiv(&a.raw, &b.raw, &result)
		}
		if err != nil {
			return err
		}
		return s.manager.Copy(&dst.raw, &result)
	}
	coefficient := func(current, last *OwnedRational) error {
		product, err := s.facadeValueLocked(&quotient, current, rationalFacadeMul)
		if err != nil {
			return err
		}
		defer s.manager.Del(&product.raw)
		difference, err := s.facadeValueLocked(last, &product, rationalFacadeSub)
		if err != nil {
			return err
		}
		defer s.manager.Del(&difference.raw)
		return copy(current, &difference)
	}
	for {
		isZero, err := s.manager.IsZero(&b.raw)
		if err != nil {
			return err
		}
		if isZero {
			break
		}
		if err := copy(&tmp, &b); err != nil {
			return err
		}
		if err := integerResult(&quotient, false); err != nil {
			return err
		}
		if err := integerResult(&b, true); err != nil {
			return err
		}
		if err := copy(&a, &tmp); err != nil {
			return err
		}
		if err := copy(&tmp, x); err != nil {
			return err
		}
		if err := coefficient(x, &lastX); err != nil {
			return err
		}
		if err := copy(&lastX, &tmp); err != nil {
			return err
		}
		if err := copy(&tmp, y); err != nil {
			return err
		}
		if err := coefficient(y, &lastY); err != nil {
			return err
		}
		if err := copy(&lastY, &tmp); err != nil {
			return err
		}
	}
	if err := copy(gcd, &a); err != nil {
		return err
	}
	if err := copy(x, &lastX); err != nil {
		return err
	}
	return copy(y, &lastY)
}
