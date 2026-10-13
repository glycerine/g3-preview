// Source: src/util/hwf.cpp:409-427.
package numeral

// ToRational follows the source bit-field decomposition, including its raw
// exponent for subnormals. The destination and scratch integers belong to qm.
func (m *HardwareFloatManager) ToRational(x *HardwareFloat, qm *RationalManager, out *Rational) error {
	if m == nil || x == nil || qm == nil || out == nil {
		return ErrNumericOwner
	}
	if e := qm.checkSetter(out); e != nil {
		return e
	}
	if modularSourceDebug && !(m.IsNormal(x) || m.IsDenormal(x) || m.IsZero(x)) {
		return ErrHardwareFloatNonRegular
	}
	z := qm.Z
	var n, d Int
	defer z.Del(&d)
	defer z.Del(&n)
	sig := m.Significand(x)
	if m.IsNormal(x) {
		sig |= uint64(1) << 52
	}
	if e := z.SetUint64(&n, sig); e != nil {
		return e
	}
	if m.Sign(x) {
		if e := z.Neg(&n); e != nil {
			return e
		}
	}
	if e := z.SetUint64(&d, uint64(1)<<52); e != nil {
		return e
	}
	exp := m.Exponent(x)
	if exp >= 0 {
		if e := z.Mul2k(&n, uint32(exp), &n); e != nil {
			return e
		}
	} else {
		if e := z.Mul2k(&d, uint32(-exp), &d); e != nil {
			return e
		}
	}
	return qm.SetParts(out, &n, &d)
}
