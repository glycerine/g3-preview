// Source: src/util/mpf.cpp:1519-1569.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

// ToStringInto writes the source result into its actual owned text context.
// The returned c_str view borrows that result until the caller's result DT.
func (m *ArbitraryFloatManager) ToStringInto(a *ArbitraryFloat, text ArbitraryFloatFormatText) (view ArbitraryFloatFormattedCString, err error) {
	if e := m.valueGuard(a); e != nil {
		return view, e
	}
	if text == nil {
		return view, ErrPolynomialOwner
	}
	s := text.NumericInvocation()
	if e := m.beginFormatInvocation(s, text); e != nil {
		return view, e
	}
	defer func() {
		s.executing = false
		s.bodyCause = err
		source := m.finishFormatSource(s)
		err = polynomialNewtonJoin(err, source)
		if err == nil {
			s.returned = true
			view, err = text.CString()
			if err != nil {
				s.bodyCause = err
				s.returned = false
			}
		}
	}()
	nan, e := m.IsNaN(a)
	if e != nil {
		return view, e
	}
	if nan {
		return view, text.AssignLiteral("NaN")
	}
	inf, e := m.IsInf(a)
	if e != nil {
		return view, e
	}
	if inf {
		if a.Sign() {
			return view, text.AssignLiteral("-oo")
		}
		return view, text.AssignLiteral("+oo")
	}
	zero, e := m.IsZero(a)
	if e != nil {
		return view, e
	}
	if zero {
		if a.Sign() {
			return view, text.AssignLiteral("-zero")
		}
		return view, text.AssignLiteral("+zero")
	}
	prefix := ""
	if a.Sign() {
		prefix = "-"
	}
	if e := text.AssignLiteral(prefix); e != nil {
		return view, e
	}
	// Scoped NUM constructors are reached only on the finite branch.
	s.num = Int{}
	s.constructed = 1
	s.denom = Int{}
	s.constructed = 2
	context := s.allocationContext()
	z := m.integer
	if e := z.setInt64WithAllocationContext(&s.num, 0, context); e != nil {
		return view, e
	}
	if e := z.setInt64WithAllocationContext(&s.denom, 1, context); e != nil {
		return view, e
	}
	denormal, e := m.IsDenormal(a)
	if e != nil {
		return view, e
	}
	exponent := a.exponent
	if denormal {
		exponent, e = m.minExponent(a.EBits())
		if e != nil {
			return view, e
		}
	} else {
		if e := z.setInt64WithAllocationContext(&s.num, 1, context); e != nil {
			return view, e
		}
		if e := z.shiftWithAllocationContext(&s.num, a.SBits()-1, &s.num, true, context); e != nil {
			return view, e
		}
	}
	if e := z.addSubWithAllocationContext(&s.num, &a.significand, &s.num, false, context); e != nil {
		return view, e
	}
	if e := z.shiftWithAllocationContext(&s.denom, a.SBits()-1, &s.denom, true, context); e != nil {
		return view, e
	}
	s.q = NewRational()
	s.constructed = 3
	if e := m.rational.setIntegerWithAllocationContext(&s.q, &s.num, context); e != nil {
		return view, e
	}
	if e := m.rational.divIntWithAllocationContext(&s.q, &s.denom, &s.q, context); e != nil {
		return view, e
	}
	s.stream, e = text.NewArbitraryFloatFormatStream()
	if e != nil {
		return view, e
	}
	if e := m.displayFormatDecimal(s, s.stream, &s.q, a.SBits()); e != nil {
		return view, e
	}
	integer, e := m.rational.IsInt(&s.q)
	if e != nil {
		return view, e
	}
	if integer {
		if e := s.stream.AppendCString(".0"); e != nil {
			return view, e
		}
	}
	if e := s.stream.AppendCString(" "); e != nil {
		return view, e
	}
	if e := s.stream.AppendInt64(exponent); e != nil {
		return view, e
	}
	return view, text.AppendStreamString(s.stream)
}
func (m *ArbitraryFloatManager) finishFormatSource(s *ArbitraryFloatFormatInvocation) error {
	if !m.matchesFormatFrame(s) {
		return ErrPolynomialOwner
	}
	if s.executing || s.finishing {
		return ErrPolynomialPending
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	if s.sourceCause != nil {
		return s.sourceCause
	}
	if e := m.formatChildSource(s); e != nil {
		s.sourceCause = e
		return e
	}
	for s.sourceCursor < 5 {
		var e error
		switch s.sourceCursor {
		case 0:
			if s.stream != nil {
				e = s.stream.SourceFinalize()
			}
		case 1:
			if s.constructed >= 3 {
				_, e = upolynomialSourceNumeralDel(m.integer, &s.q.num)
			}
		case 2:
			if s.constructed >= 3 {
				_, e = upolynomialSourceNumeralDel(m.integer, &s.q.den)
			}
		case 3:
			if s.constructed >= 2 {
				_, e = upolynomialSourceNumeralDel(m.integer, &s.denom)
			}
		case 4:
			if s.constructed >= 1 {
				_, e = upolynomialSourceNumeralDel(m.integer, &s.num)
			}
		}
		if e != nil {
			s.sourceCause = e
			return e
		}
		s.sourceCursor++
	}
	return nil
}
