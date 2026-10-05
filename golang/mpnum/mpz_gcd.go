// Licensed under the MIT license.
// Source: src/util/mpz.cpp gcd_core and gcd(_MP_INTERNAL / LEHMER_GCD),
// src/util/mpz.h addmul/submul. Original author: Leonardo de Moura.
package mpnum

import "math"

// U32GCD and U64GCD are concrete translations of the source binary gcd_core.
// The source template is specialized explicitly to obey the no-generics rule.
func U32GCD(u, v uint32) uint32 {
	if u == 0 {
		return v
	}
	if v == 0 {
		return u
	}
	var k uint
	for (u|v)&1 == 0 {
		u >>= 1
		v >>= 1
		k++
	}
	for u&1 == 0 {
		u >>= 1
	}
	for {
		for v&1 == 0 {
			v >>= 1
		}
		if u < v {
			v -= u
		} else {
			diff := u - v
			u = v
			v = diff
		}
		v >>= 1
		if v == 0 {
			break
		}
	}
	return u << k
}
func U64GCD(u, v uint64) uint64 {
	if u == 0 {
		return v
	}
	if v == 0 {
		return u
	}
	var k uint
	for (u|v)&1 == 0 {
		u >>= 1
		v >>= 1
		k++
	}
	for u&1 == 0 {
		u >>= 1
	}
	for {
		for v&1 == 0 {
			v >>= 1
		}
		if u < v {
			v -= u
		} else {
			diff := u - v
			u = v
			v = diff
		}
		v >>= 1
		if v == 0 {
			break
		}
	}
	return u << k
}

// AddMul stores a+b*c, preserving the source +/-1 shortcut predicates (only
// inline +/-1 qualify). The nil public adapter retains its explicitly
// incomplete eager failure cleanup; source-owned RAW uses end-only DEL.
func (m *IntegerManager) AddMul(a, b, c, d *Int) error {
	return m.addMulWithSourceReceipt(a, b, c, d, nil, nil)
}

// One literal arithmetic body serves public and explicitly source-owned calls.
// The nil-receipt failure adapter temporarily preserves historical eager RAW
// cleanup; it is an incomplete deviation, not source AddMul lifetime credit.
func (m *IntegerManager) addMulWithSourceReceipt(a, b, c, d *Int, receipt *integerAddMulInvocationReceipt, arithmetic *polynomialArithmeticInvocationReceipt) (err error) {
	for _, n := range []*Int{a, b, c, d} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	context := integerAllocationInvocationContext{arithmetic: arithmetic}
	if b.IsSmall() && b.val == 1 {
		return m.addSubWithAllocationContext(a, c, d, false, context)
	}
	if b.IsSmall() && b.val == -1 {
		return m.addSubWithAllocationContext(a, c, d, true, context)
	}
	var legacyTmp Int
	endReached := false
	tmp := &legacyTmp
	var frame *integerAddMulScope
	if receipt != nil {
		frame = &m.addMulInvocation
		if frame.active || frame.invocation == ^uint64(0) {
			return ErrPolynomialPending
		}
		*frame = integerAddMulScope{active: true, executing: true, invocation: frame.invocation + 1, stage: 1}
		if arithmetic != nil {
			frame.parent = *arithmetic
		}
		*receipt = integerAddMulInvocationReceipt{m, frame, frame.invocation}
		tmp = &frame.tmp
		context = integerAllocationInvocationContext{addMul: receipt}
		defer func() {
			frame.executing = false
			if frame.stage < 3 {
				frame.bodyCause = err
			}
			if tmp.IsSmall() && !polynomialNegNumericPending(&frame.numeric) && frame.sourceCause == nil {
				frame.active = false
			}
		}()
	} else {
		// Source RAW has no destructor on body failure. This legacy ergonomic
		// adapter remains separately incomplete until all public users carry source
		// ownership; its mathematical Mul/Add engine is shared with the source path.
		defer func() {
			if err != nil && !endReached {
				_ = m.Del(tmp)
			}
		}()
	}
	if e := m.mulWithAllocationContext(b, c, tmp, context); e != nil {
		return e
	}
	if frame != nil {
		frame.stage = 2
		if arithmetic != nil {
			context = integerAllocationInvocationContext{arithmetic: arithmetic}
		}
	}
	if e := m.addSubWithAllocationContext(a, tmp, d, false, context); e != nil {
		return e
	}
	if frame != nil {
		frame.stage = 3
		return m.finishAddMulSource(frame)
	}
	endReached = true
	return m.Del(tmp)
}

// SubMul stores a-b*c with the source corresponding shortcut branches.
func (m *IntegerManager) SubMul(a, b, c, d *Int) error {
	for _, n := range []*Int{a, b, c, d} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.IsSmall() && b.val == 1 {
		return m.Sub(a, c, d)
	}
	if b.IsSmall() && b.val == -1 {
		return m.Add(a, c, d)
	}
	var tmp Int
	defer m.Del(&tmp)
	if e := m.Mul(b, c, &tmp); e != nil {
		return e
	}
	return m.Sub(a, &tmp, d)
}

// GCD translates the internal backend's selected Lehmer algorithm, not the
// inactive Euclid/binary alternatives. It returns the nonnegative gcd, including
// gcd(0,0)=0. Destination may alias either input. All working Ints are explicit
// owner-local source copies. The zero fastpath Copies and Abs directly into the
// caller; errors preserve completed actions rather than a staged destination.
func (m *IntegerManager) GCD(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.IsSmall() && b.IsSmall() && a.val != math.MinInt32 && b.val != math.MinInt32 {
		av, bv := a.val, b.val
		if av < 0 {
			av = -av
		}
		if bv < 0 {
			bv = -bv
		}
		return m.SetInt64(c, int64(U32GCD(uint32(av), uint32(bv))))
	}
	if a.val == 0 {
		if e := m.Copy(c, b); e != nil {
			return e
		}
		return m.Abs(c)
	}
	if b.val == 0 {
		if e := m.Copy(c, a); e != nil {
			return e
		}
		return m.Abs(c)
	}
	var a1, b1, t, r, tmp Int
	defer func() { m.Del(&a1); m.Del(&b1); m.Del(&r); m.Del(&t); m.Del(&tmp) }()
	if e := m.Copy(&a1, a); e != nil {
		return e
	}
	if e := m.Copy(&b1, b); e != nil {
		return e
	}
	if e := m.Abs(&a1); e != nil {
		return e
	}
	if e := m.Abs(&b1); e != nil {
		return e
	}
	cmp, e := m.Compare(&a1, &b1)
	if e != nil {
		return e
	}
	if cmp < 0 {
		if e = m.Swap(&a1, &b1); e != nil {
			return e
		}
	}
	for {
		if e = m.gcdSourceGreaterEqual(&a1, &b1); e != nil {
			return e
		}
		if b1.IsSmall() {
			if a1.IsSmall() {
				return m.SetInt64(c, int64(U32GCD(uint32(a1.val), uint32(b1.val))))
			}
			for b1.val != 0 {
				if e = m.gcdSourceGreaterEqual(&a1, &b1); e != nil {
					return e
				}
				if e = m.Rem(&a1, &b1, &tmp); e != nil {
					return e
				}
				if e = m.Swap(&a1, &b1); e != nil {
					return e
				}
				if e = m.Swap(&b1, &tmp); e != nil {
					return e
				}
			}
			return m.Swap(c, &a1)
		}
		if sourceGCDDebug {
			if e = gcdSourceRequire(!a1.IsSmall()); e != nil {
				return e
			}
			if e = gcdSourceRequire(!b1.IsSmall()); e != nil {
				return e
			}
		}
		asize, bsize := m.cell(a1.cell).size, m.cell(b1.cell).size
		if sourceGCDDebug {
			if e = gcdSourceRequire(bsize <= asize); e != nil {
				return e
			}
		}
		ahat := int64(m.digit(a1.cell, asize-1))
		bhat := int64(0)
		if bsize == asize {
			bhat = int64(m.digit(b1.cell, bsize-1))
		}
		A, B, C, D := int64(1), int64(0), int64(0), int64(1)
		for {
			if sourceGCDDebug {
				limit := int64(math.MaxUint32) + 1
				if e = gcdSourceRequire(ahat+A <= limit); e != nil {
					return e
				}
				if e = gcdSourceRequire(ahat+B < limit); e != nil {
					return e
				}
				if e = gcdSourceRequire(bhat+C < limit); e != nil {
					return e
				}
				if e = gcdSourceRequire(bhat+D <= limit); e != nil {
					return e
				}
			}
			if bhat+C == 0 || bhat+D == 0 {
				break
			}
			quotient := (ahat + A) / (bhat + C)
			if quotient != (ahat+B)/(bhat+D) {
				break
			}
			T := A - quotient*C
			A = C
			C = T
			T = B - quotient*D
			B = D
			D = T
			T = ahat - quotient*bhat
			ahat = bhat
			bhat = T
		}
		if e = m.gcdSourceGreaterEqual(&a1, &b1); e != nil {
			return e
		}
		if B == 0 {
			if e = m.Rem(&a1, &b1, &t); e != nil {
				return e
			}
			if e = m.Swap(&a1, &b1); e != nil {
				return e
			}
			if e = m.Swap(&b1, &t); e != nil {
				return e
			}
			if e = m.gcdSourceGreaterEqual(&a1, &b1); e != nil {
				return e
			}
		} else {
			if e = m.SetInt64(&tmp, A); e != nil {
				return e
			}
			if e = m.Mul(&a1, &tmp, &t); e != nil {
				return e
			}
			if e = m.SetInt64(&tmp, B); e != nil {
				return e
			}
			if e = m.AddMul(&t, &tmp, &b1, &t); e != nil {
				return e
			}
			if e = m.SetInt64(&tmp, C); e != nil {
				return e
			}
			if e = m.Mul(&a1, &tmp, &r); e != nil {
				return e
			}
			if e = m.SetInt64(&tmp, D); e != nil {
				return e
			}
			if e = m.AddMul(&r, &tmp, &b1, &r); e != nil {
				return e
			}
			if e = m.Swap(&a1, &t); e != nil {
				return e
			}
			if e = m.Swap(&b1, &r); e != nil {
				return e
			}
			if e = m.gcdSourceGreaterEqual(&a1, &b1); e != nil {
				return e
			}
		}
	}
}
