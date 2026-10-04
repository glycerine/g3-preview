// Licensed under the MIT license.
// Source: src/util/mpz.cpp display_smt2, mpq.cpp display_smt2/display_decimal,
// and mpq.h to_rational_string/display_pp forwarding wrappers.
package mpnum

// appendDisplay copies static punctuation into the existing budgeted workspace.
// It preserves the complete prefix if preparation grows the byte owner.
func (m *NaturalManager) appendDisplay(offset uint32, text string) (uint32, error) {
	end, e := checkedTextEnd(offset, uint64(len(text)))
	if e != nil {
		return 0, e
	}
	if e = m.reserveText(end); e != nil {
		return 0, e
	}
	for i := 0; i < len(text); i++ {
		m.textWrite(offset+uint32(i), text[i])
	}
	return end, nil
}
func (m *IntegerManager) prepareSMT2(a *Int, decimal bool, offset uint32) (length uint32, result error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	input := a
	// Source scoped temporary is created only in the negative branch; Copy/Neg
	// precede emission of the negative SMT2 prefix.
	if a.val < 0 {
		if e := m.beginSMT2Display(); e != nil {
			return 0, e
		}
		defer m.finishSMT2Display(&length, &result)
		tmp := &m.smt2Display.value
		if e := m.copySMT2Temporary(a); e != nil {
			return 0, e
		}
		if e := m.Neg(tmp); e != nil {
			return 0, e
		}
		input = tmp
		var e error
		offset, e = m.natural.appendDisplay(offset, "(- ")
		if e != nil {
			return 0, e
		}
	}
	n, e := m.prepareInteger(input, offset)
	if e != nil {
		return 0, e
	}
	if decimal {
		n, e = m.natural.appendDisplay(n, ".0")
		if e != nil {
			return 0, e
		}
	}
	if a.val < 0 {
		n, e = m.natural.appendDisplay(n, ")")
		if e != nil {
			return 0, e
		}
	}
	return n, nil
}

// FormatSMT2Into follows source expression syntax. Length excludes any C NUL;
// short buffers and predictable preparation errors leave dst unchanged.
func (m *IntegerManager) FormatSMT2Into(a *Int, decimal bool, dst []byte) (int, error) {
	n, e := m.prepareSMT2(a, decimal, 0)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}

// SMT2String allocates only the final caller-owned host string after preparing
// complete text in the numeric owner's accounted workspace.
func (m *IntegerManager) SMT2String(a *Int, decimal bool) (string, error) {
	n, e := m.prepareSMT2(a, decimal, 0)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *RationalManager) prepareSMT2(a *Rational, decimal bool) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if isIntegerRational(a) {
		return m.z.prepareSMT2(&a.num, decimal, 0)
	}
	n, e := m.z.natural.appendDisplay(0, "(/ ")
	if e != nil {
		return 0, e
	}
	n, e = m.z.prepareSMT2(&a.num, decimal, n)
	if e != nil {
		return 0, e
	}
	n, e = m.z.natural.appendDisplay(n, " ")
	if e != nil {
		return 0, e
	}
	n, e = m.z.prepareSMT2(&a.den, decimal, n)
	if e != nil {
		return 0, e
	}
	return m.z.natural.appendDisplay(n, ")")
}
func (m *RationalManager) FormatSMT2Into(a *Rational, decimal bool, dst []byte) (int, error) {
	n, e := m.prepareSMT2(a, decimal)
	if e != nil {
		return 0, e
	}
	return m.z.natural.publishText(dst, n)
}
func (m *RationalManager) SMT2String(a *Rational, decimal bool) (string, error) {
	n, e := m.prepareSMT2(a, decimal)
	if e != nil {
		return "", e
	}
	return m.z.natural.finalString(n)
}

// prepareDecimal retains source Copy, sign/Neg, Div/display/Rem, then each
// Mul/Div/display/Rem action. No rounding or zero padding is introduced.
func (m *RationalManager) prepareDecimal(a *Rational, precision uint32, truncate bool) (length uint32, resultErr error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	if e := m.z.beginRawDisplay(3, [3]uint8{0, 1, 2}); e != nil {
		return 0, e
	}
	defer m.z.finishRawDisplay(&resultErr, &length)
	numerator, denominator, digit := &m.z.rawDisplay.roots[0], &m.z.rawDisplay.roots[1], &m.z.rawDisplay.roots[2]
	ten := NewInt(10)
	if e := m.z.Copy(numerator, &a.num); e != nil {
		return 0, e
	}
	if e := m.z.Copy(denominator, &a.den); e != nil {
		return 0, e
	}
	var offset uint32
	if a.num.val < 0 {
		var e error
		offset, e = m.z.natural.appendDisplay(offset, "-")
		if e != nil {
			return 0, e
		}
		if e = m.z.Neg(numerator); e != nil {
			return 0, e
		}
	}
	if e := m.z.Div(numerator, denominator, digit); e != nil {
		return 0, e
	}
	offset, e := m.z.prepareInteger(digit, offset)
	if e != nil {
		return 0, e
	}
	if e = m.z.Rem(numerator, denominator, numerator); e != nil {
		return 0, e
	}
	if numerator.val == 0 {
		return offset, nil
	}
	offset, e = m.z.natural.appendDisplay(offset, ".")
	if e != nil {
		return 0, e
	}
	for i := uint32(0); i < precision; i++ {
		if e = m.z.Mul(numerator, &ten, numerator); e != nil {
			return 0, e
		}
		if e = m.z.Div(numerator, denominator, digit); e != nil {
			return 0, e
		}
		offset, e = m.z.prepareInteger(digit, offset)
		if e != nil {
			return 0, e
		}
		if e = m.z.Rem(numerator, denominator, numerator); e != nil {
			return 0, e
		}
		if numerator.val == 0 {
			return offset, nil
		}
	}
	if !truncate {
		return m.z.natural.appendDisplay(offset, "?")
	}
	return offset, nil
}
func (m *RationalManager) FormatDecimalInto(a *Rational, precision uint32, truncate bool, dst []byte) (int, error) {
	n, e := m.prepareDecimal(a, precision, truncate)
	if e != nil {
		return 0, e
	}
	return m.z.natural.publishText(dst, n)
}
func (m *RationalManager) DecimalString(a *Rational, precision uint32, truncate bool) (string, error) {
	n, e := m.prepareDecimal(a, precision, truncate)
	if e != nil {
		return "", e
	}
	return m.z.natural.finalString(n)
}

// ToRationalString and FormatPrettyInto are the source thin forwarding variants,
// retaining the ordinary integer-or-fraction representation and host boundary.
func (m *RationalManager) ToRationalString(a *Rational) (string, error) { return m.String(a) }
func (m *RationalManager) FormatPrettyInto(a *Rational, dst []byte) (int, error) {
	return m.FormatInto(a, dst)
}
