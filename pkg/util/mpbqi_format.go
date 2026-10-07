// Licensed under the MIT license.
// Source basic_interval display.
package util

func (m *BinaryIntervalManager) appendBound(a *BinaryRational, offset uint32) (uint32, error) {
	n, e := m.bounds.Z.prepareInteger(&a.Num, offset)
	if e != nil || a.KField == 0 {
		return n, e
	}
	n, e = m.bounds.Z.Natural.appendDisplay(n, "/2")
	if e != nil || a.KField == 1 {
		return n, e
	}
	n, e = m.bounds.Z.Natural.appendDisplay(n, "^")
	if e != nil {
		return n, e
	}
	return m.bounds.appendUnsigned(n, a.KField)
}
func (m *BinaryIntervalManager) prepareInterval(a *BinaryInterval) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	w := m.bounds.Z.Natural
	n, e := w.appendDisplay(0, "(")
	if e != nil {
		return 0, e
	}
	n, e = m.appendBound(&a.lower, n)
	if e != nil {
		return 0, e
	}
	n, e = w.appendDisplay(n, ", ")
	if e != nil {
		return 0, e
	}
	n, e = m.appendBound(&a.upper, n)
	if e != nil {
		return 0, e
	}
	return w.appendDisplay(n, ")")
}
func (m *BinaryIntervalManager) String(a *BinaryInterval) (string, error) {
	n, e := m.prepareInterval(a)
	if e != nil {
		return "", e
	}
	return m.bounds.Z.Natural.finalString(n)
}
func (m *BinaryIntervalManager) FormatInto(a *BinaryInterval, dst []byte) (int, error) {
	n, e := m.prepareInterval(a)
	if e != nil {
		return 0, e
	}
	return m.bounds.Z.Natural.publishText(dst, n)
}
