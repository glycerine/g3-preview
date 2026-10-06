// Source: src/util/vector.h:59-100; src/util/mpf.h:65-73;
// src/util/memory_manager.cpp:274-289.
// Source author: Leonardo de Moura; Christoph Wintersteiger.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// GrowRuntimeValueStorage is the concrete source vector allocator leaf. Its
// returned block is acquired by the calling vector at this actual outcome;
// this method neither publishes m_data nor infers ownership from counters.
func (m *ArbitraryFloatManager) GrowRuntimeValueStorage(old memory.Block, request uint64) (memory.Block, error) {
	if e := m.valueGuard(&ArbitraryFloat{}); e != nil {
		return old, e
	}
	m.valueExecuting = true
	defer func() { m.valueExecuting = false }()
	if old == (memory.Block{}) {
		return m.memory.Allocate(request)
	}
	h := old
	e := m.memory.Reallocate(&h, request)
	return h, e
}

// The backlink is borrowed, lifecycle-only cold metadata installed at the
// actual completed MPQ/Integer member publication. It acquires no value root.
// Ordinary idle retained parents never exclude child use; only the concrete
// synchronous external transport reached by this exact parent does so.
func (z *IntegerManager) mpfLifecycleCallbackActive() bool {
	if z == nil || z.self != z {
		return false
	}
	p := z.mpfParent
	if p == nil || !p.valueExecuting {
		return false
	}
	return p.self == p && p.integer == z && p.rational != nil && p.rational.self == p.rational && p.rational.z == z && p.rational.mpfParent == p && p.memory == z.memory && p.memory != nil && p.memory.FailureTransportActive()
}
func (q *RationalManager) mpfLifecycleCallbackActive() bool {
	return q != nil && q.self == q && q.z != nil && q.mpfParent != nil && q.mpfParent.rational == q && q.z.mpfLifecycleCallbackActive()
}
