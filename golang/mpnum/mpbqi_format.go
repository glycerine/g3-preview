// Licensed under the MIT license.
// Source basic_interval display.
package mpnum

func (m *BinaryIntervalManager) appendBound(a *BinaryRational, offset uint32) (uint32, error) {
	n, e := m.bounds.z.prepareInteger(&a.num, offset)
	if e != nil || a.k == 0 {
		return n, e
	}
	n, e = m.bounds.z.natural.appendDisplay(n, "/2")
	if e != nil || a.k == 1 {
		return n, e
	}
	n, e = m.bounds.z.natural.appendDisplay(n, "^")
	if e != nil {
		return n, e
	}
	return m.bounds.appendUnsigned(n, a.k)
}
func (m *BinaryIntervalManager) prepareInterval(a *BinaryInterval) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	w := m.bounds.z.natural
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
	return m.bounds.z.natural.finalString(n)
}
func (m *BinaryIntervalManager) FormatInto(a *BinaryInterval, dst []byte) (int, error) {
	n, e := m.prepareInterval(a)
	if e != nil {
		return 0, e
	}
	return m.bounds.z.natural.publishText(dst, n)
}
