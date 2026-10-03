// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpz.cpp bitwise_or/and/xor/not, _MP_INTERNAL branch.
package mpnum

import "errors"

var ErrBitwiseDomain = errors.New("mpnum: bitwise source operand must be nonnegative")

func (m *IntegerManager) BitwiseOr(a, b, out *Int) error  { return m.bitwiseBinary(a, b, out, 0) }
func (m *IntegerManager) BitwiseAnd(a, b, out *Int) error { return m.bitwiseBinary(a, b, out, 1) }
func (m *IntegerManager) BitwiseXor(a, b, out *Int) error { return m.bitwiseBinary(a, b, out, 2) }

// bitwiseBinary preserves the source arithmetic chunk algorithm, including its
// final multiplier/division actions. Work is owner-budgeted source temporaries;
// predictable errors preserve the caller's output, including aliased operands.
func (m *IntegerManager) bitwiseBinary(a, b, out *Int, op uint8) error {
	for _, v := range []*Int{a, b, out} {
		if e := m.check(v); e != nil {
			return e
		}
	}
	if a.val < 0 || b.val < 0 {
		return ErrBitwiseDomain
	}
	if a.IsSmall() && b.IsSmall() {
		var v int32
		switch op {
		case 0:
			v = a.val | b.val
		case 1:
			v = a.val & b.val
		case 2:
			v = a.val ^ b.val
		}
		return m.SetInt64(out, int64(v))
	}
	var a1, b1, a2, b2, mult, tmp, work Int
	defer func() {
		m.Del(&a1)
		m.Del(&b1)
		m.Del(&a2)
		m.Del(&b2)
		m.Del(&mult)
		m.Del(&tmp)
		m.Del(&work)
	}()
	if e := m.Copy(&a1, a); e != nil {
		return e
	}
	if e := m.Copy(&b1, b); e != nil {
		return e
	}
	if e := m.SetInt64(&mult, 1); e != nil {
		return e
	}
	if e := m.Reset(&work); e != nil {
		return e
	}
	for a1.val != 0 && b1.val != 0 {
		if e := m.Mod(&a1, &m.two64, &a2); e != nil {
			return e
		}
		if e := m.Mod(&b1, &m.two64, &b2); e != nil {
			return e
		}
		u, e := m.Uint64(&a2)
		if e != nil {
			return e
		}
		v, e := m.Uint64(&b2)
		if e != nil {
			return e
		}
		switch op {
		case 0:
			v = u | v
		case 1:
			v = u & v
		case 2:
			v = u ^ v
		}
		if e = m.SetUint64(&tmp, v); e != nil {
			return e
		}
		if e = m.Mul(&tmp, &mult, &tmp); e != nil {
			return e
		}
		if e = m.Add(&work, &tmp, &work); e != nil {
			return e
		}
		if e = m.Mul(&mult, &m.two64, &mult); e != nil {
			return e
		}
		if e = m.Div(&a1, &m.two64, &a1); e != nil {
			return e
		}
		if e = m.Div(&b1, &m.two64, &b1); e != nil {
			return e
		}
	}
	if op != 1 {
		if a1.val != 0 {
			if e := m.Mul(&a1, &mult, &a1); e != nil {
				return e
			}
			if e := m.Add(&work, &a1, &work); e != nil {
				return e
			}
		}
		if b1.val != 0 {
			if e := m.Mul(&b1, &mult, &b1); e != nil {
				return e
			}
			if e := m.Add(&work, &b1, &work); e != nil {
				return e
			}
		}
	}
	return m.Move(out, &work)
}

// BitwiseNot complements within the specified source unsigned bit width.
// The source small path uses a signed shift at width63; unsigned mask arithmetic
// defines its intended low63 bits without signed overflow. The general path
// retains all 64-bit source chunk actions, including the final unused multiplier.
func (m *IntegerManager) BitwiseNot(width uint32, a, out *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.check(out); e != nil {
		return e
	}
	if a.val < 0 {
		return ErrBitwiseDomain
	}
	if a.IsSmall() && width <= 63 {
		mask := (uint64(1) << width) - 1
		return m.SetInt64(out, int64(^uint64(a.val)&mask))
	}
	var a1, a2, mult, tmp, work Int
	defer func() { m.Del(&a1); m.Del(&a2); m.Del(&mult); m.Del(&tmp); m.Del(&work) }()
	if e := m.Copy(&a1, a); e != nil {
		return e
	}
	if e := m.SetInt64(&mult, 1); e != nil {
		return e
	}
	if e := m.SetInt64(&work, 0); e != nil {
		return e
	}
	for width > 0 {
		if e := m.Mod(&a1, &m.two64, &a2); e != nil {
			return e
		}
		n, e := m.Uint64(&a2)
		if e != nil {
			return e
		}
		v := ^n
		if width < 64 {
			v &= (uint64(1) << width) - 1
		}
		if e = m.SetUint64(&tmp, v); e != nil {
			return e
		}
		if e = m.Mul(&tmp, &mult, &tmp); e != nil {
			return e
		}
		if e = m.Add(&work, &tmp, &work); e != nil {
			return e
		}
		if e = m.Mul(&mult, &m.two64, &mult); e != nil {
			return e
		}
		if e = m.Div(&a1, &m.two64, &a1); e != nil {
			return e
		}
		if width < 64 {
			width = 0
		} else {
			width -= 64
		}
	}
	return m.Move(out, &work)
}
