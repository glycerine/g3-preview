// Source: src/util/memory_manager.cpp:89-113,153-160.
// Source author: Nikolaj Bjorner (2007-07-24).
// Licensed under the MIT license; see LICENSE.txt.
package util

// Close is owning abort of only actual parent acquisitions. It never simulates
// successful mem_finalize, resets source flags/counters, or closes borrowed
// Runtime/rational/symbol/debug/trace/prime headers. Stopped child receipts are
// handled by their actual same-engine owning cleanup, not a registry sweep.
func (p *MemoryProcess) Close() error {
	if p == nil || p.self != p {
		return ErrMemoryProcessOwner
	}
	if p.closed {
		return nil
	}
	p.closing = true
	if p.parameters != nil {
		if e := p.parameters.Close(); e != nil {
			return e
		}
		p.parameters = nil
	}
	if p.catalog != nil {
		if e := p.catalog.Close(); e != nil {
			return e
		}
		p.catalog = nil
	}
	if p.context != nil {
		if e := p.context.Close(); e != nil {
			return e
		}
		p.context = nil
	}
	p.config.Budget.Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
