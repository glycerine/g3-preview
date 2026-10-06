// Licensed under the MIT license.
// Source: src/util/mpz.h and src/util/mpz.cpp internal signed division wrappers/core.
// Knuth division translated from src/util/mpn.cpp.
// Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"math"
)

var ErrDivideByZero = errors.New("mpnum: integer division by zero")

const (
	quotientOnly         = 0
	remainderOnly        = 1
	quotientAndRemainder = 2
)

// MachineDivRem truncates toward zero; the remainder has the numerator's sign.
// Either output may alias either input. If q==r the source publication order
// is retained: the general/small path ends in remainder, whereas the
// shorter-numerator limb guard copies remainder then resets quotient to zero.
func (m *IntegerManager) MachineDivRem(a, b, q, r *Int) error {
	return m.machineDivRemWithAllocationContext(a, b, q, r, integerAllocationInvocationContext{})
}
func (m *IntegerManager) machineDivRemWithAllocationContext(a, b, q, r *Int, context integerAllocationInvocationContext) error {
	for _, n := range []*Int{a, b, q, r} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		av, bv := int64(a.val), int64(b.val)
		qv, rv := av/bv, av%bv
		// Only INT_MIN/-1 promotes a small quotient. Allocate before publishing.
		var prepared numericCellRef
		if qv > math.MaxInt32 && q.cell == 0 {
			var e error
			prepared, e = m.allocateWithSourceContext(6, 0, nil, context)
			if e != nil {
				return e
			}
		}
		if prepared != 0 {
			q.cell = prepared
			m.publishCell(prepared)
		}
		if e := m.setInt64WithAllocationContext(q, qv, context); e != nil {
			return e
		}
		return m.setInt64WithAllocationContext(r, rv, context)
	}
	return m.quotRemWithAllocationContext(a, b, q, r, quotientAndRemainder, context)
}
func (m *IntegerManager) MachineDiv(a, b, c *Int) error {
	return m.machineDivWithAllocationContext(a, b, c, integerAllocationInvocationContext{})
}
func (m *IntegerManager) machineDivWithAllocationContext(a, b, c *Int, context integerAllocationInvocationContext) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		return m.setInt64WithAllocationContext(c, int64(a.val)/int64(b.val), context)
	}
	var dummy Int
	return m.quotRemWithAllocationContext(a, b, c, &dummy, quotientOnly, context)
}
func (m *IntegerManager) Rem(a, b, c *Int) error { return m.remWithPNReceipt(a, b, c, nil) }
func (m *IntegerManager) remWithPNReceipt(a, b, c *Int, pn *integerPNormalizeScope) error {
	return m.remWithAllocationContext(a, b, c, pnAllocationContext(pn))
}
func (m *IntegerManager) remWithAllocationContext(a, b, c *Int, context integerAllocationInvocationContext) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if b.val == 0 {
		return ErrDivideByZero
	}
	if a.IsSmall() && b.IsSmall() {
		return m.setInt64WithAllocationContext(c, int64(a.val)%int64(b.val), context)
	}
	var dummy Int
	return m.quotRemWithAllocationContext(a, b, &dummy, c, remainderOnly, context)
}

// Div implements the source Euclidean quotient adjustment for a negative
// numerator. The source rereads b after output publication; when c==b this
// preserves a known source alias defect (documented in ../PORT_PROGRESS.md#audit-mpnum-port-status-md).
func (m *IntegerManager) Div(a, b, c *Int) error {
	return m.divWithAllocationContext(a, b, c, integerAllocationInvocationContext{})
}
func (m *IntegerManager) divWithAllocationContext(a, b, c *Int, context integerAllocationInvocationContext) (err error) {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if a.val < 0 {
		var tmp *Int
		var raw *integerScalarRawScope
		if r := context.scalarDiv; m.scalarDivAllocationRoots(r) != nil {
			raw = &r.frame.division
			if e := m.beginScalarRaw(raw, r, 1); e != nil {
				return e
			}
			tmp = &raw.values[0]
			defer func() {
				raw.executing = false
				if err == nil {
					raw.bodySucceeded = true
					err = m.finishScalarRawSource(raw)
				}
			}()
		} else {
			public := &m.divisionRaw
			receipt, e := m.beginRawInvocation(public, 1)
			if e != nil {
				return e
			}
			tmp = &public.values[0]
			context.recordContentRaw(m, receipt)
			m.bindHenselRaw(context, receipt, true)
			m.bindHenselCallerRaw(context, receipt, true)
			m.bindExactDivRaw(context, receipt, true)
			m.bindScalarExactDivRaw(context, receipt, true)
			m.bindDividesRaw(context, receipt, true)
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
		if e := m.machineDivRemWithAllocationContext(a, b, c, tmp, context); e != nil {
			return e
		}
		if tmp.val != 0 {
			one := NewInt(1)
			return m.addSubWithAllocationContext(c, &one, c, b.val >= 0, context)
		}
		return nil
	}
	return m.machineDivWithAllocationContext(a, b, c, context)
}

// Mod implements source nonnegative modulus, including its reread of an
// overwritten divisor when c==b (the source alias defect).
func (m *IntegerManager) Mod(a, b, c *Int) error {
	for _, n := range []*Int{a, b, c} {
		if e := m.check(n); e != nil {
			return e
		}
	}
	if e := m.Rem(a, b, c); e != nil {
		return e
	}
	if c.val < 0 {
		if b.val > 0 {
			return m.Add(c, b, c)
		}
		return m.Sub(c, b, c)
	}
	return nil
}

func (m *IntegerManager) normalizedTmpSize(idx int, size uint32) uint32 {
	for size > 0 && m.digit(m.tmp[idx], size-1) == 0 {
		size--
	}
	return size
}
func (m *IntegerManager) quotRemCore(a, b, q, r *Int, mode int) error {
	return m.quotRemWithPNReceipt(a, b, q, r, mode, nil)
}
func (m *IntegerManager) quotRemWithPNReceipt(a, b, q, r *Int, mode int, pn *integerPNormalizeScope) error {
	return m.quotRemWithAllocationContext(a, b, q, r, mode, pnAllocationContext(pn))
}
func (m *IntegerManager) quotRemWithAllocationContext(a, b, q, r *Int, mode int, context integerAllocationInvocationContext) error {
	sa, ca := m.signCell(a, 0)
	sb, cb := m.signCell(b, 1)
	na, nb := m.cell(ca).size, m.cell(cb).size
	if nb > na {
		if mode != quotientOnly {
			if e := m.copyFromOwnerWithAllocationContext(r, m, a, nil, context); e != nil {
				return e
			}
		}
		if mode != remainderOnly {
			return m.Reset(q)
		}
		return nil
	}
	qsize, rsize := na-nb+1, nb
	if e := m.ensureTmpWithAllocationContext(0, qsize, context); e != nil {
		return e
	}
	if e := m.ensureTmpWithAllocationContext(1, rsize, context); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(0, qsize); e != nil {
		return e
	}
	if e := m.prepareTmpSpan(1, rsize); e != nil {
		return e
	}
	if e := m.divideCellsWithAllocationContext(ca, cb, context); e != nil {
		return e
	}
	sign := int32(1)
	if sa != sb {
		sign = -1
	}
	if mode != remainderOnly {
		if e := m.publishTmpAtWithAllocationContext(0, q, sign, qsize, context); e != nil {
			return e
		}
	}
	if mode != quotientOnly {
		return m.publishTmpAtWithAllocationContext(1, r, sa, rsize, context)
	}
	return nil
}

// divideCells borrows ephemeral indexed views; no arena backing slice escapes.
func (m *IntegerManager) divideCells(a, b numericCellRef) error {
	return m.divideCellsWithPNReceipt(a, b, nil)
}
func (m *IntegerManager) divideCellsWithPNReceipt(a, b numericCellRef, pn *integerPNormalizeScope) error {
	return m.divideCellsWithAllocationContext(a, b, pnAllocationContext(pn))
}
func (m *IntegerManager) divideCellsWithAllocationContext(a, b numericCellRef, context integerAllocationInvocationContext) error {
	ca, cb := m.cell(a), m.cell(b)
	cq, cr := m.cell(m.tmp[0]), m.cell(m.tmp[1])
	_, e := m.natural.divideWithAllocationContext(limbView{allocator: m.allocator, backing: ca.backing, size: ca.size}, limbView{allocator: m.allocator, backing: cb.backing, size: cb.size}, limbView{allocator: m.allocator, backing: cq.backing, size: cq.span}, limbView{allocator: m.allocator, backing: cr.backing, size: cr.span}, context)
	return e
}
