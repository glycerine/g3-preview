// Source: src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// These are the four literal RAW declarations. Failure omits the end label's
// DEL statements; explicit owning cleanup has its own cursor after older NUMs.
type rationalDecimalScope struct {
	originalPending                        memory.Block
	originalCell                           sourceCellBacking
	cellRetired                            bool
	transferred                            memory.SmallAllocatorPendingAllocation
	buffers                                [4]naturalPNBufferReceipt
	verification                           naturalPNVerificationReceipt
	parent                                 *ArbitraryFloatFormatInvocation
	values                                 [4]Int // ten,n1,d1,v1: source end-label DEL order
	count                                  uint8
	executing, bodySucceeded, ended, owned bool
	cursor, ownCursor                      uint8
	numeric                                polynomialNumericRoots
	division                               integerRawInvocationReceipt
	insertion                              NumeralInsertionReceipt
	bodyCause, sourceCause, ownCause       error
}

func (m *ArbitraryFloatManager) displayFormatDecimal(s *ArbitraryFloatFormatInvocation, out RationalSourceStream, a *Rational, prec uint32) (err error) {
	d := &s.decimal
	if d.parent != nil {
		return ErrPolynomialPending
	}
	*d = rationalDecimalScope{parent: s, executing: true, count: 3}
	context := s.allocationContext()
	context.formatDecimal = d
	z := m.integer
	n1, d1, v1, ten := &d.values[1], &d.values[2], &d.values[3], &d.values[0]
	defer func() {
		d.executing = false
		d.bodyCause = err
		if err == nil {
			d.bodySucceeded = true
		}
		err = polynomialNewtonJoin(err, m.finishFormatDecimal(d))
	}()
	if e := z.copyFromOwnerWithAllocationContext(n1, z, &a.num, nil, context); e != nil {
		return e
	}
	if e := z.copyFromOwnerWithAllocationContext(d1, z, &a.den, nil, context); e != nil {
		return e
	}
	if a.num.val < 0 {
		if e := out.AppendCString("-"); e != nil {
			return e
		}
		if e := z.Neg(n1); e != nil {
			return e
		}
	}
	*ten = NewInt(10)
	d.count = 4
	if e := z.divWithAllocationContext(n1, d1, v1, context); e != nil {
		return e
	}
	if e := z.insertSourceIntegerTracked(out, v1, nil, &d.insertion); e != nil {
		return e
	}
	if e := z.remWithAllocationContext(n1, d1, n1, context); e != nil {
		return e
	}
	if n1.val == 0 {
		return nil
	}
	if e := out.AppendCString("."); e != nil {
		return e
	}
	for i := uint32(0); i < prec; i++ {
		if e := z.mulWithAllocationContext(n1, ten, n1, context); e != nil {
			return e
		}
		if e := z.divWithAllocationContext(n1, d1, v1, context); e != nil {
			return e
		}
		if modularSourceDebug {
			less, e := z.Less(v1, ten)
			if e != nil {
				return e
			}
			if !less {
				return ErrIntegerAssertion
			}
		}
		if e := z.insertSourceIntegerTracked(out, v1, nil, &d.insertion); e != nil {
			return e
		}
		if e := z.remWithAllocationContext(n1, d1, n1, context); e != nil {
			return e
		}
		if n1.val == 0 {
			return nil
		}
	}
	return out.AppendCString("?")
}
func (m *ArbitraryFloatManager) finishFormatDecimal(d *rationalDecimalScope) error {
	if d.parent == nil {
		return nil
	}
	if !m.matchesFormatFrame(d.parent) || d != &d.parent.decimal {
		return ErrPolynomialOwner
	}
	if d.sourceCause != nil {
		return d.sourceCause
	}
	if e := m.integer.rawInvocationSourceCause(&d.division); e != nil {
		d.sourceCause = e
		return e
	}
	if e := m.integer.natural.pnVerificationSourceCause(&d.verification); e != nil {
		d.sourceCause = e
		return e
	}
	if d.insertion.sourceCause != nil {
		d.sourceCause = d.insertion.sourceCause
		return d.sourceCause
	}
	if !d.bodySucceeded || d.ended {
		return nil
	}
	for d.cursor < 4 {
		// ten was constructed after the first three declarations.
		if d.cursor == 0 && d.count < 4 {
			d.cursor++
			continue
		}
		_, e := upolynomialSourceNumeralDel(m.integer, &d.values[d.cursor])
		if e != nil {
			d.sourceCause = e
			return e
		}
		d.cursor++
	}
	d.ended = true
	return nil
}
func (m *ArbitraryFloatManager) cleanupFormatDecimal(d *rationalDecimalScope, abort bool) error {
	if d.parent == nil {
		return nil
	}
	if !m.matchesFormatFrame(d.parent) || d != &d.parent.decimal {
		return ErrPolynomialOwner
	}
	if d.executing {
		return ErrPolynomialPending
	}
	if !abort {
		if e := m.cleanupFormatInsertion(&d.insertion, false); e != nil {
			return e
		}
		if e := m.integer.natural.cleanupPNVerification(&d.verification, false); e != nil {
			return e
		}
		return m.integer.cleanupRawInvocation(&d.division, false)
	}
	for d.ownCursor < 4 {
		if d.ownCursor != 0 || d.count >= 4 {
			if e := m.integer.Del(&d.values[d.ownCursor]); e != nil {
				if d.ownCause == nil {
					d.ownCause = e
				}
				return e
			}
		}
		d.ownCursor++
	}
	if e := m.cleanupFormatInsertion(&d.insertion, true); e != nil {
		return e
	}
	if e := m.integer.cleanupRawInvocation(&d.division, true); e != nil {
		return e
	}
	if e := m.integer.natural.cleanupPNVerification(&d.verification, true); e != nil {
		return e
	}
	for i := len(d.buffers) - 1; i >= 0; i-- {
		if e := d.buffers[i].cleanup(); e != nil {
			return e
		}
	}
	if e := m.integer.cleanupFormatAllocation(&d.numeric, &d.transferred, &d.cellRetired); e != nil {
		return e
	}
	d.owned = true
	return nil
}

// Cleanup is bound to the actually issued character/formatting tuples. A later
// unrelated insertion is never adopted through the Natural member address.
func (m *ArbitraryFloatManager) cleanupFormatInsertion(r *NumeralInsertionReceipt, abort bool) error {
	if r == nil || !r.active {
		return nil
	}
	n := m.integer.natural
	if r.natural != n || r.character != &n.charFormatting || r.invocation == 0 {
		return ErrPolynomialOwner
	}
	if r.characterOwned && r.invocation != r.character.invocation {
		return ErrPolynomialOwner
	}
	if r.formattingOwned && (r.formatting != &n.formatting || r.formattingInvocation != r.formatting.invocation) {
		return ErrPolynomialOwner
	}
	stop := func(e error) error {
		if e != nil && r.ownCause == nil {
			r.ownCause = e
		}
		return e
	}
	if !r.sourceDone {
		if r.formattingOwned {
			if e := n.unwindFormattingScope(r.formatting); e != nil {
				return stop(e)
			}
		}
		if r.characterOwned && r.characterCompleted {
			if e := n.destroyIntegerCharacters(); e != nil {
				return stop(e)
			}
			r.characterOwned = false
		}
		r.sourceDone = true
	}
	if !abort {
		return nil
	}
	if r.formattingOwned {
		if e := n.cleanupFormattingScope(r.formatting); e != nil {
			return stop(e)
		}
	}
	if r.characterOwned {
		if e := r.character.buffer.finalize(n.runtime); e != nil {
			return stop(e)
		}
		n.charFormatting = integerCharacterScope{invocation: r.invocation}
	}
	r.characterOwned = false
	r.formattingOwned = false
	r.active = false
	r.retired = true
	return nil
}
