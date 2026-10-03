// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpz.cpp array gcd, extended Euclid, lcm, divides.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

var errExtendedGCDInvariant = errors.New("mpnum: source extended gcd Bezout assertion failed")

// GCDMany preserves active source sequential order, including destination
// aliases to future array elements and early-one skipping of remaining inputs.
func (m *IntegerManager) GCDMany(as []Int, g *Int) error {
	if uint64(len(as)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if e := m.check(g); e != nil {
		return e
	}
	switch len(as) {
	case 0:
		return m.Reset(g)
	case 1:
		if e := m.Copy(g, &as[0]); e != nil {
			return e
		}
		return m.Abs(g)
	}
	if e := m.GCD(&as[0], &as[1], g); e != nil {
		return e
	}
	for i := 2; i < len(as); i++ {
		if isOneInteger(g) {
			return nil
		}
		if e := m.GCD(g, &as[i], g); e != nil {
			return e
		}
	}
	return nil
}

// ExtendedGCD preserves source coefficient publication at every action. Outputs
// may alias each other or inputs; input signs are reread only after the loop.
// A failure keeps already-published coefficients and always deletes locals.
func (m *IntegerManager) ExtendedGCD(r1, r2, a, b, g *Int) error {
	for _, v := range []*Int{r1, r2, a, b, g} {
		if e := m.check(v); e != nil {
			return e
		}
	}
	var tmp1, tmp2, aux, quot, nexta, nextb Int
	defer func() {
		m.Del(&tmp1)
		m.Del(&tmp2)
		m.Del(&aux)
		m.Del(&quot)
		m.Del(&nexta)
		m.Del(&nextb)
	}()
	if e := m.Copy(&tmp1, r1); e != nil {
		return e
	}
	if e := m.Copy(&tmp2, r2); e != nil {
		return e
	}
	if e := m.SetInt64(a, 1); e != nil {
		return e
	}
	if e := m.SetInt64(b, 0); e != nil {
		return e
	}
	if e := m.SetInt64(&nexta, 0); e != nil {
		return e
	}
	if e := m.SetInt64(&nextb, 1); e != nil {
		return e
	}
	if e := m.Abs(&tmp1); e != nil {
		return e
	}
	if e := m.Abs(&tmp2); e != nil {
		return e
	}
	cmp, e := m.Compare(&tmp1, &tmp2)
	if e != nil {
		return e
	}
	if cmp < 0 {
		if e = m.Swap(&tmp1, &tmp2); e != nil {
			return e
		}
		if e = m.Swap(&nexta, &nextb); e != nil {
			return e
		}
		if e = m.Swap(a, b); e != nil {
			return e
		}
	}
	for tmp2.val > 0 {
		if e = m.gcdSourceGreaterEqual(&tmp1, &tmp2); e != nil {
			return e
		}
		if e = m.Copy(&aux, &tmp2); e != nil {
			return e
		}
		if e = m.MachineDiv(&tmp1, &tmp2, &quot); e != nil {
			return e
		}
		if e = m.Rem(&tmp1, &tmp2, &tmp2); e != nil {
			return e
		}
		if e = m.Copy(&tmp1, &aux); e != nil {
			return e
		}
		if e = m.Copy(&aux, &nexta); e != nil {
			return e
		}
		if e = m.Mul(&quot, &nexta, &nexta); e != nil {
			return e
		}
		if e = m.Sub(a, &nexta, &nexta); e != nil {
			return e
		}
		if e = m.Copy(a, &aux); e != nil {
			return e
		}
		if e = m.Copy(&aux, &nextb); e != nil {
			return e
		}
		if e = m.Mul(&nextb, &quot, &nextb); e != nil {
			return e
		}
		if e = m.Sub(b, &nextb, &nextb); e != nil {
			return e
		}
		if e = m.Copy(b, &aux); e != nil {
			return e
		}
	}
	if r1.val < 0 {
		if e = m.Neg(a); e != nil {
			return e
		}
	}
	if r2.val < 0 {
		if e = m.Neg(b); e != nil {
			return e
		}
	}
	if e = m.extendedGCDDiagnostic(r1, r2, a, b, &tmp1, &nexta, &nextb); e != nil {
		return e
	}
	return m.Copy(g, &tmp1)
}

// LCM follows the signed source shortcuts; it is not canonical abs(a*b)/gcd.
func (m *IntegerManager) LCM(a, b, c *Int) error {
	for _, v := range []*Int{a, b, c} {
		if e := m.check(v); e != nil {
			return e
		}
	}
	if isOneInteger(b) {
		return m.Copy(c, a)
	}
	if isOneInteger(a) {
		return m.Copy(c, b)
	}
	cmp, e := m.Compare(a, b)
	if e != nil {
		return e
	}
	if cmp == 0 {
		return m.Copy(c, b)
	}
	var r Int
	defer m.Del(&r)
	if e = m.GCD(a, b, &r); e != nil {
		return e
	}
	cmp, e = m.Compare(&r, a)
	if e != nil {
		return e
	}
	if cmp == 0 {
		return m.Copy(c, b)
	}
	cmp, e = m.Compare(&r, b)
	if e != nil {
		return e
	}
	if cmp == 0 {
		return m.Copy(c, a)
	}
	if e = m.MachineDiv(a, &r, &r); e != nil {
		return e
	}
	return m.Mul(&r, b, c)
}
func (m *IntegerManager) Divides(a, b *Int) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(b); e != nil {
		return false, e
	}
	if a.val == 0 {
		return b.val == 0, nil
	}
	var tmp Int
	defer m.Del(&tmp)
	if e := m.Rem(b, a, &tmp); e != nil {
		return false, e
	}
	return tmp.val == 0, nil
}
