// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Source: src/util/mpz.cpp. Licensed under the MIT license.
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
	return m.GCDManyIndexed(integerArrayView{values: as, count: uint32(len(as))}, g)
}

// ExtendedGCD preserves source coefficient publication at every action. Outputs
// may alias each other or inputs; input signs are reread only after the loop.
// A failure keeps already-published coefficients and actual RAW locals.
// End DEL runs only after a successful body; explicit owning abort is separate.
func (m *IntegerManager) ExtendedGCD(r1, r2, a, b, g *Int) error {
	return m.extendedGCDWithAllocationContext(r1, r2, a, b, g, integerAllocationInvocationContext{})
}
func (m *IntegerManager) extendedGCDWithAllocationContext(r1, r2, a, b, g *Int, context integerAllocationInvocationContext) (err error) {
	for _, v := range []*Int{r1, r2, a, b, g} {
		if e := m.check(v); e != nil {
			return e
		}
	}
	var values *[6]Int
	var raw *integerScalarRawScope
	if r := context.scalarDiv; m.scalarDivAllocationRoots(r) != nil {
		raw = &r.frame.extended
		if e := m.beginScalarRaw(raw, r, 4); e != nil {
			return e
		}
		values = &raw.values
		defer func() {
			raw.executing = false
			if err == nil {
				raw.bodySucceeded = true
				err = m.finishScalarRawSource(raw)
			}
		}()
	} else {
		public := &m.extendedRaw
		receipt, e := m.beginRawInvocation(public, 4)
		if e != nil {
			return e
		}
		values = &public.values
		context.recordRecombinationRaw(m, receipt)
		context.recordRootUpperRaw(m, receipt)
		context.recordRemoveOneHalfRaw(m, receipt)
		context.recordPrimePreparationRaw(m, receipt)
		context.recordFactorFrontendRaw(m, receipt)
		context.recordContentRaw(m, receipt)
		context.recordNormalizationRaw(m, receipt)
		context.recordSOMNormalizationRaw(m, receipt)
		m.bindHenselRaw(context, receipt, false)
		m.bindHenselCallerRaw(context, receipt, false)
		m.bindExactDivRaw(context, receipt, false)
		m.bindScalarExactDivRaw(context, receipt, false)
		m.bindDividesRaw(context, receipt, false)
		context = context.withRawInvocation(receipt)
		defer func() {
			public.executing = false
			public.bodyCause = err
			if err == nil {
				public.bodySucceeded = true
				err = m.finishRawSource(public)
			}
		}()
	}
	tmp1, tmp2, aux, quot, nexta, nextb := &values[0], &values[1], &values[2], &values[3], &values[4], &values[5]
	if e := m.copyFromOwnerWithAllocationContext(tmp1, m, r1, nil, context); e != nil {
		return e
	}
	if e := m.copyFromOwnerWithAllocationContext(tmp2, m, r2, nil, context); e != nil {
		return e
	}
	if e := m.setInt64WithAllocationContext(a, 1, context); e != nil {
		return e
	}
	if e := m.setInt64WithAllocationContext(b, 0, context); e != nil {
		return e
	}
	if raw != nil {
		raw.count = 6
	} else {
		m.extendedRaw.count = 6
	}
	if e := m.setInt64WithAllocationContext(nexta, 0, context); e != nil {
		return e
	}
	if e := m.setInt64WithAllocationContext(nextb, 1, context); e != nil {
		return e
	}
	if e := m.absWithAllocationContext(tmp1, context); e != nil {
		return e
	}
	if e := m.absWithAllocationContext(tmp2, context); e != nil {
		return e
	}
	cmp, e := m.Compare(tmp1, tmp2)
	if e != nil {
		return e
	}
	if cmp < 0 {
		if e = m.Swap(tmp1, tmp2); e != nil {
			return e
		}
		if e = m.Swap(nexta, nextb); e != nil {
			return e
		}
		if e = m.Swap(a, b); e != nil {
			return e
		}
	}
	for tmp2.val > 0 {
		if e = m.gcdSourceGreaterEqual(tmp1, tmp2); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(aux, m, tmp2, nil, context); e != nil {
			return e
		}
		if e = m.machineDivWithAllocationContext(tmp1, tmp2, quot, context); e != nil {
			return e
		}
		if e = m.remWithAllocationContext(tmp1, tmp2, tmp2, context); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(tmp1, m, aux, nil, context); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(aux, m, nexta, nil, context); e != nil {
			return e
		}
		if e = m.mulWithAllocationContext(quot, nexta, nexta, context); e != nil {
			return e
		}
		if e = m.addSubWithAllocationContext(a, nexta, nexta, true, context); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(a, m, aux, nil, context); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(aux, m, nextb, nil, context); e != nil {
			return e
		}
		if e = m.mulWithAllocationContext(nextb, quot, nextb, context); e != nil {
			return e
		}
		if e = m.addSubWithAllocationContext(b, nextb, nextb, true, context); e != nil {
			return e
		}
		if e = m.copyFromOwnerWithAllocationContext(b, m, aux, nil, context); e != nil {
			return e
		}
	}
	if r1.val < 0 {
		if e = m.negWithAllocationContext(a, context); e != nil {
			return e
		}
	}
	if r2.val < 0 {
		if e = m.negWithAllocationContext(b, context); e != nil {
			return e
		}
	}
	if e = m.extendedGCDDiagnosticWithAllocationContext(r1, r2, a, b, tmp1, nexta, nextb, context); e != nil {
		return e
	}
	return m.copyFromOwnerWithAllocationContext(g, m, tmp1, nil, context)
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
	return m.dividesWithAllocationContext(a, b, integerAllocationInvocationContext{})
}
func (m *IntegerManager) dividesWithAllocationContext(a, b *Int, parent integerAllocationInvocationContext) (result bool, cause error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if e := m.check(b); e != nil {
		return false, e
	}
	receipt, e := m.beginDividesNumeralInvocation(parent)
	if e != nil {
		return false, e
	}
	s := receipt.frame
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = polynomialNewtonJoin(cause, m.finishDividesNumeralSource(s))
	}()
	if a.val == 0 {
		return b.val == 0, nil
	}
	context := integerAllocationInvocationContext{boolDivides: &s.receipt, boolDividesInvocation: s.invocation}
	if e := m.remWithAllocationContext(b, a, &s.numeral, context); e != nil {
		return false, e
	}
	return s.numeral.val == 0, nil
}
