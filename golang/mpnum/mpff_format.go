// Copyright (c) 2012 Microsoft Corporation. Source all MPFF displays.
package mpnum

import "github.com/glycerine/g3/golang/arena"

func (m *FastFloatManager) appendText(n uint32, s string) (uint32, error) {
	return m.natural.appendDisplay(n, s)
}
func (m *FastFloatManager) appendUnsigned(n uint32, v uint64) (uint32, error) {
	end, e := checkedTextEnd(n, 20)
	if e != nil {
		return 0, e
	}
	if e = m.natural.reserveText(end); e != nil {
		return 0, e
	}
	start := n
	for {
		m.natural.textWrite(n, '0'+byte(v%10))
		n++
		v /= 10
		if v == 0 {
			break
		}
	}
	m.natural.reverseText(start, n)
	return n, nil
}
func (m *FastFloatManager) appendSigned(n uint32, v int64) (uint32, error) {
	mag := uint64(v)
	if v < 0 {
		var e error
		n, e = m.appendText(n, "-")
		if e != nil {
			return 0, e
		}
		mag = 0 - mag
	}
	return m.appendUnsigned(n, mag)
}
func (m *FastFloatManager) prepareDisplay(a *FastFloat, raw, smt, decimal bool) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	n := uint32(0)
	var e error
	if a.negative() {
		prefix := "-"
		if smt {
			prefix = "(- "
		}
		n, e = m.appendText(n, prefix)
		if e != nil {
			return 0, e
		}
	}
	if raw {
		end, e := checkedTextEnd(n, uint64(m.precision)*8)
		if e != nil {
			return 0, e
		}
		if e = m.natural.reserveText(end); e != nil {
			return 0, e
		}
		s := m.sig(a)
		for i := m.precision; i > 0; {
			i--
			v := s.read(i)
			for j := uint32(8); j > 0; {
				j--
				m.natural.textWrite(n, "0123456789abcdef"[(v>>(j*4))&15])
				n++
			}
		}
		n, e = m.appendText(n, "*2^")
		if e != nil {
			return 0, e
		}
		return m.appendSigned(n, int64(a.exponent))
	}
	m.toBuffer(0, a, true, false)
	v := m.buffer(0, m.precision)
	tz := ffNTZ(v)
	exp := int64(a.exponent)
	shift := uint32(0)
	if exp < 0 {
		if int64(tz) >= -exp {
			shift = uint32(-exp)
			exp = 0
		} else {
			shift = tz
			exp += int64(tz)
		}
	}
	if shift != 0 {
		ffShift(v, v, shift, false)
	}
	if smt && exp != 0 {
		prefix := "(* "
		if exp < 0 {
			prefix = "(/ "
		}
		n, e = m.appendText(n, prefix)
		if e != nil {
			return 0, e
		}
	}
	n, e = m.prepareNatural(m.buffer(0, m.precision), n)
	if e != nil {
		return 0, e
	}
	if smt && decimal {
		n, e = m.appendText(n, ".0")
		if e != nil {
			return 0, e
		}
	}
	if exp != 0 {
		mag := uint64(exp)
		if exp < 0 {
			mag = uint64(-exp)
		}
		if !smt {
			op := "*"
			if exp < 0 {
				op = "/"
			}
			n, e = m.appendText(n, op)
			if e != nil {
				return 0, e
			}
			if mag <= 63 {
				return m.appendUnsigned(n, uint64(1)<<uint(mag))
			}
			n, e = m.appendText(n, "2^")
			if e != nil {
				return 0, e
			}
			return m.appendUnsigned(n, mag)
		}
		if mag <= 63 {
			n, e = m.appendUnsigned(n, uint64(1)<<uint(mag))
			if e == nil && decimal {
				n, e = m.appendText(n, ".0")
			}
		} else {
			prefix := " (^ 2"
			if decimal {
				prefix = " (^ 2.0"
			}
			n, e = m.appendText(n, prefix)
			if e == nil {
				n, e = m.appendText(n, " ")
			}
			if e == nil {
				n, e = m.appendUnsigned(n, mag)
			}
			if e == nil && decimal {
				n, e = m.appendText(n, ".0")
			}
			if e == nil {
				n, e = m.appendText(n, ")")
			}
		}
		if e != nil {
			return 0, e
		}
		n, e = m.appendText(n, ")")
		if e != nil {
			return 0, e
		}
	}
	if smt && a.negative() {
		return m.appendText(n, ")")
	}
	return n, nil
}
func (m *FastFloatManager) decimalResize(i int, n uint32) error {
	v := &m.decimal[i]
	inline := uint32(1024)
	if i == 2 {
		inline = 16
	}
	if v.ref == (arena.VectorRef{}) {
		r, e := m.words.Allocate(0, inline)
		if e != nil {
			return e
		}
		v.ref = r
		v.capacity = inline
	}
	for v.capacity < n {
		if v.capacity > ^uint32(0)/8 {
			return arena.ErrBounds
		}
		c := v.capacity * 2
		r, e := m.words.Reserve(v.ref, c)
		if e != nil {
			return e
		}
		v.ref = r
		v.capacity = c
	}
	r, e := m.words.Resize(v.ref, n)
	if e != nil {
		return e
	}
	v.ref = r
	v.size = n
	return nil
}
func (m *FastFloatManager) dv(i int, n uint32) ffLimbView {
	return ffLimbView{m: m, v: &m.decimal[i], size: n}
}
func (m *FastFloatManager) decimalDivide(a, b, q, r ffLimbView) error {
	av := limbView{words: m.words, ref: a.v.ref, size: a.size}
	bv := limbView{words: m.words, ref: b.v.ref, size: b.size}
	qv := limbView{words: m.words, ref: q.v.ref, size: a.size - b.size + 1}
	rv := limbView{words: m.words, ref: r.v.ref, size: b.size}
	_, e := m.natural.divide(av, bv, qv, rv)
	return e
}
func (m *FastFloatManager) prepareDecimal(a *FastFloat, prec, maxPower uint32) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	exp := int64(a.exponent)
	integer, e := m.IsInteger(a)
	if e != nil {
		return 0, e
	}
	if exp >= int64(maxPower) || exp <= -int64(maxPower)-int64(m.bits) || integer {
		return m.prepareDisplay(a, false, false, false)
	}
	n := uint32(0)
	if a.negative() {
		n, e = m.appendText(n, "-")
		if e != nil {
			return 0, e
		}
	}
	exp = -exp
	numWords := uint32(1 + exp/32)
	extra := uint32(1)
	if numWords > m.precision {
		extra += numWords - m.precision
	}
	sz := m.precision + extra
	for i := 0; i < 4; i++ {
		count := sz
		if i == 2 {
			count = m.precision
		}
		if i == 3 {
			count = numWords
		}
		if e = m.decimalResize(i, count); e != nil {
			return 0, e
		}
		ffZero(m.dv(i, count))
	}
	b1, b2, b3, pw := m.dv(0, sz), m.dv(1, sz), m.dv(2, m.precision), m.dv(3, numWords)
	ffCopy(m.sig(a), b1)
	pw.write(0, 1)
	ffShift(pw, pw, uint32(exp), true)
	if numWords > m.precision {
		n, e = m.appendText(n, "0")
	} else {
		numer := b1
		numer.size = m.precision
		if e = m.decimalDivide(numer, pw, b3, b2); e != nil {
			return 0, e
		}
		n, e = m.prepareNatural(b3, n)
		if e != nil {
			return 0, e
		}
		if sourceExtendedGCDDebug && !(!ffIsZero(b2)) {
			return 0, ErrFastFloatInvariant
		}
		ffCopy(b2, b1)
	}
	if e != nil {
		return 0, e
	}
	n, e = m.appendText(n, ".")
	if e != nil {
		return 0, e
	}
	sz1 := b1.size
	for sz1 > 0 && b1.read(sz1-1) == 0 {
		sz1--
	}
	if sourceExtendedGCDDebug && !(sz1 > 0) {
		return 0, ErrFastFloatInvariant
	}
	for i := uint32(0); sz1 > 0; i++ {
		if i >= prec {
			return m.appendText(n, "?")
		}
		a1 := b1
		a1.size = sz1
		out := b2
		out.size = sz1 + 1
		ffMul(a1, ffLimbView{size: 1, literal: 10}, out)
		sz2 := out.size
		for sz2 > 0 && out.read(sz2-1) == 0 {
			sz2--
		}
		if sourceExtendedGCDDebug && !(sz2 > 0) {
			return 0, ErrFastFloatInvariant
		}
		if numWords > sz2 {
			n, e = m.appendText(n, "0")
			sz1 = sz2
			src := b2
			src.size = sz2
			dst := b1
			dst.size = sz1
			ffCopy(src, dst)
		} else {
			src := b2
			src.size = sz2
			if e = m.decimalDivide(src, pw, b3, b1); e != nil {
				return 0, e
			}
			n, e = m.appendUnsigned(n, uint64(b3.read(0)))
			sz1 = numWords
			for sz1 > 0 && b1.read(sz1-1) == 0 {
				sz1--
			}
		}
		if e != nil {
			return 0, e
		}
	}
	return n, nil
}
func (m *FastFloatManager) String(a *FastFloat) (string, error) {
	n, e := m.prepareDisplay(a, false, false, false)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *FastFloatManager) RationalString(a *FastFloat) (string, error) { return m.String(a) }
func (m *FastFloatManager) PrettyString(a *FastFloat) (string, error)   { return m.String(a) }
func (m *FastFloatManager) RawString(a *FastFloat) (string, error) {
	n, e := m.prepareDisplay(a, true, false, false)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *FastFloatManager) SMT2String(a *FastFloat, decimal bool) (string, error) {
	n, e := m.prepareDisplay(a, false, true, decimal)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *FastFloatManager) DecimalString(a *FastFloat, p, max uint32) (string, error) {
	n, e := m.prepareDecimal(a, p, max)
	if e != nil {
		return "", e
	}
	return m.natural.finalString(n)
}
func (m *FastFloatManager) DecimalStringDefault32_128(a *FastFloat) (string, error) {
	return m.DecimalString(a, 32, 128)
}
func (m *FastFloatManager) FormatInto(a *FastFloat, dst []byte) (int, error) {
	n, e := m.prepareDisplay(a, false, false, false)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}
func (m *FastFloatManager) FormatPrettyInto(a *FastFloat, dst []byte) (int, error) {
	return m.FormatInto(a, dst)
}
func (m *FastFloatManager) FormatRawInto(a *FastFloat, dst []byte) (int, error) {
	n, e := m.prepareDisplay(a, true, false, false)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}
func (m *FastFloatManager) FormatSMT2Into(a *FastFloat, decimal bool, dst []byte) (int, error) {
	n, e := m.prepareDisplay(a, false, true, decimal)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}
func (m *FastFloatManager) FormatDecimalInto(a *FastFloat, p, max uint32, dst []byte) (int, error) {
	n, e := m.prepareDecimal(a, p, max)
	if e != nil {
		return 0, e
	}
	return m.natural.publishText(dst, n)
}

func (m *FastFloatManager) prepareNatural(v ffLimbView, n uint32) (uint32, error) {
	for v.size > 1 && v.read(v.size-1) == 0 {
		v.size--
	}
	return m.natural.prepareNatural(limbView{words: m.words, ref: v.v.ref, size: v.size}, n)
}
