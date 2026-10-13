// Source: src/util/f2n.h:96-112, instantiated as f2n<mpf_manager>.
package numeral

// Inv follows the source local-one, divide, delete sequence. The local is
// released on error too, so the Go owner can still close after a failed call.
func (m *ArbitraryFloatNumeralManager) Inv(x *ArbitraryFloat) (cause error) {
	if e := m.guard(); e != nil {
		return e
	}
	var one ArbitraryFloat
	defer func() { cause = JoinNumericErrors(cause, m.core.Del(&one)) }()
	if e := m.SetInt32(&one, 1); e != nil {
		return e
	}
	if e := m.Div(&one, x, x); e != nil {
		return e
	}
	return m.check(x)
}

// InvFrom is the source two-numeral overload used by interval division.
func (m *ArbitraryFloatNumeralManager) InvFrom(in, out *ArbitraryFloat) error {
	if e := m.Copy(out, in); e != nil {
		return e
	}
	return m.Inv(out)
}

func (m *ArbitraryFloatNumeralManager) Increment(x *ArbitraryFloat) error {
	if e := m.guard(); e != nil {
		return e
	}
	return m.Add(x, &m.one, x)
}

func (m *ArbitraryFloatNumeralManager) Decrement(x *ArbitraryFloat) error {
	if e := m.guard(); e != nil {
		return e
	}
	return m.Sub(x, &m.one, x)
}

// Power uses the source's unsigned mask walk, including its final square.
// Once mask wraps to zero, continuing would loop forever in C++; the Go
// translation ends that unreachable-for-default-parameters path.
func (m *ArbitraryFloatNumeralManager) Power(a *ArbitraryFloat, exponent uint32, out *ArbitraryFloat) (cause error) {
	if e := m.guard(); e != nil {
		return e
	}
	var power ArbitraryFloat
	defer func() { cause = JoinNumericErrors(cause, m.core.Del(&power)) }()
	if e := m.Copy(&power, a); e != nil {
		return e
	}
	if e := m.SetInt32(out, 1); e != nil {
		return e
	}
	for mask := uint32(1); mask != 0 && mask <= exponent; mask <<= 1 {
		if mask&exponent != 0 {
			if e := m.Mul(out, &power, out); e != nil {
				return e
			}
		}
		if e := m.Mul(&power, &power, &power); e != nil {
			return e
		}
	}
	return m.check(out)
}
