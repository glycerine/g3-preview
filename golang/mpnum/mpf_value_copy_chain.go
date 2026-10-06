// Source: src/util/scoped_numeral_vector.h:25-65; src/util/mpf.cpp:316-333.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Christoph Wintersteiger.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// Embedded, charged with the concrete vector owner. Links are acquired only
// by the existing allocating SET record leaf. They borrow exact issued records;
// this chain creates no second numeral/storage owner or allocation registry.
type ArbitraryFloatValueCopyChain struct {
	self             *ArbitraryFloatValueCopyChain
	manager          *ArbitraryFloatManager
	head             *arbitraryFloatValueCopyCall
	executing, ended bool
}

func (c *ArbitraryFloatValueCopyChain) Bind(m *ArbitraryFloatManager) error {
	if c == nil || m == nil || m.self != m || m.closed || m.sourceEnded {
		return memory.ErrRuntimeOwner
	}
	if c.self != nil {
		if c.self != c || c.manager != m || c.ended {
			return memory.ErrRuntimeOwner
		}
		return nil
	}
	*c = ArbitraryFloatValueCopyChain{self: c, manager: m}
	return nil
}
func (c *ArbitraryFloatValueCopyChain) guard() error {
	if c == nil || c.self != c || c.manager == nil || c.manager.self != c.manager || c.ended {
		return memory.ErrRuntimeOwner
	}
	if c.executing || c.manager.valueExecuting || c.manager.memory.FailureTransportActive() {
		return ErrPolynomialPending
	}
	return nil
}
func (c *ArbitraryFloatValueCopyChain) CopyToRuntime(block memory.Block, offset uint64, sourceManager *ArbitraryFloatManager, source *ArbitraryFloat) error {
	return c.CopyToRuntimeWithReceipt(block, offset, sourceManager, source, nil)
}
func (c *ArbitraryFloatValueCopyChain) CopyToRuntimeWithReceipt(block memory.Block, offset uint64, sourceManager *ArbitraryFloatManager, source *ArbitraryFloat, receipt *ArbitraryFloatValueCopyReceipt) error {
	if e := c.guard(); e != nil {
		return e
	}
	c.executing = true
	defer func() { c.executing = false }()
	return c.manager.copyToRuntimeFromOwnerWithReceiptAndChain(block, offset, sourceManager, source, receipt, c)
}
func (c *ArbitraryFloatValueCopyChain) Cleanup(abort bool) error {
	if c != nil && c.self == nil && c.manager == nil && c.head == nil {
		return nil
	}
	if e := c.guard(); e != nil {
		return e
	}
	for q := c.head; q != nil; q = q.callerNext {
		if q.caller != c || q.self != q || q.manager != c.manager {
			return memory.ErrRuntimeOwner
		}
		if !q.callerFalseAck {
			if e := c.manager.CleanupValueCopy(&q.issued, false); e != nil {
				return e
			}
			q.callerFalseAck = true
		}
		if abort && !q.callerTrueAck {
			if e := c.manager.CleanupValueCopy(&q.issued, true); e != nil {
				return e
			}
			q.callerTrueAck = true
		}
	}
	return nil
}
func (c *ArbitraryFloatValueCopyChain) End() error {
	if c != nil && c.self == nil && c.manager == nil && c.head == nil {
		return nil
	}
	if c != nil && c.self == c && c.ended {
		return nil
	}
	if e := c.guard(); e != nil {
		return e
	}
	for q := c.head; q != nil; q = q.callerNext {
		if q.caller != c || !q.callerFalseAck || !q.callerTrueAck {
			return ErrPolynomialPending
		}
	}
	c.ended = true
	return nil
}
