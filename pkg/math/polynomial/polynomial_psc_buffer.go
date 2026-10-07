// Source: src/util/ref_buffer.h:38-99; src/util/buffer.h:29-46,146-151;
// src/math/polynomial/polynomial.cpp:5704. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// These actual scalar inline fields are charged by the reflected enclosing
// owner. Heap payload uses the actual Runtime only after the source16 capacity
// is exhausted. Handle16 replaces native pointer width; this is not native ABI.
type polynomialPSCRefBuffer struct {
	nextFrame                                                                   *polynomialPSCNextScope
	nextInvocation                                                              uint64
	invocation                                                                  uint64
	self                                                                        *polynomialPSCRefBuffer
	owner                                                                       *MultivariatePolynomialCore
	inline                                                                      [16]Polynomial
	raw, pending, sourceRaw                                                     util.Block
	count, capacity, initialized, copied, cursor                                uint32
	pendingOwns, rawDestroyed, interrupted, constructed, finalized, freeStarted bool
	incoming                                                                    Polynomial
	incomingDec, drainDec                                                       bool
	finishing                                                                   bool
	sourceCause, cleanupCause                                                   error
}

func (b *polynomialPSCRefBuffer) construct(m *MultivariatePolynomialCore) error {
	return b.constructWithNextReceipt(m, nil)
}
func (b *polynomialPSCRefBuffer) constructWithNextReceipt(m *MultivariatePolynomialCore, r *polynomialPSCNextInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if r != nil && (b != &m.pscBuffer || !m.matchesPSCNextInvocation(r) || !r.frame.executing) {
		return ErrPolynomialOwner
	}
	if b.invocation == ^uint64(0) || b.constructed && !b.finalized || b.incoming != (Polynomial{}) || b.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	*b = polynomialPSCRefBuffer{owner: m, capacity: 16, constructed: true, invocation: b.invocation + 1}
	b.self = b
	if r != nil {
		b.nextFrame = r.frame
		b.nextInvocation = r.invocation
	}
	return nil
}
func (b *polynomialPSCRefBuffer) check(m *MultivariatePolynomialCore) error {
	if b == nil || b.self != b || b.owner != m || !b.constructed {
		return ErrPolynomialOwner
	}
	if b.finalized {
		return arena.ErrClosed
	}
	return nil
}
func pscReadHandle(m *MultivariatePolynomialCore, raw util.Block, i uint32) (Polynomial, error) {
	o := uint64(i) * 16
	v := polynomialVector{raw: raw}
	owner, e := v.read64(m, o)
	if e != nil {
		return Polynomial{}, e
	}
	slot, e := m.memory.ReadUint32(raw, o+8)
	if e != nil {
		return Polynomial{}, e
	}
	generation, e := m.memory.ReadUint32(raw, o+12)
	if e != nil {
		return Polynomial{}, e
	}
	return Polynomial{owner: owner, slot: slot, generation: generation}, nil
}
func pscWriteHandle(m *MultivariatePolynomialCore, raw util.Block, i uint32, p Polynomial) error {
	o := uint64(i) * 16
	v := polynomialVector{raw: raw}
	if e := v.write64(m, o, p.owner); e != nil {
		return e
	}
	if e := m.memory.WriteUint32(raw, o+8, p.slot); e != nil {
		return e
	}
	return m.memory.WriteUint32(raw, o+12, p.generation)
}
func (b *polynomialPSCRefBuffer) physical(m *MultivariatePolynomialCore, i uint32) (Polynomial, error) {
	if b.rawDestroyed {
		return pscReadHandle(m, b.sourceRaw, i)
	}
	if i >= b.initialized {
		return Polynomial{}, arena.ErrBounds
	}
	if b.raw == (util.Block{}) {
		if i >= 16 {
			return Polynomial{}, arena.ErrBounds
		}
		return b.inline[i], nil
	}
	return pscReadHandle(m, b.raw, i)
}
func (b *polynomialPSCRefBuffer) get(m *MultivariatePolynomialCore, i uint32) (Polynomial, error) {
	if e := b.check(m); e != nil {
		return Polynomial{}, e
	}
	if polynomialSourceDebug && i >= b.count {
		return Polynomial{}, ErrPolynomialAssertion
	}
	return b.physical(m, i)
}
func (b *polynomialPSCRefBuffer) expand(m *MultivariatePolynomialCore) error {
	nc := b.capacity << 1
	var e error
	b.pending, e = m.memory.Allocate(uint64(nc) * 16)
	if e != nil {
		return e
	}
	for b.copied < b.count {
		p, e := b.physical(m, b.copied)
		if e != nil {
			return e
		}
		if e = pscWriteHandle(m, b.pending, b.copied, p); e != nil {
			return e
		}
		b.copied++
	}
	if b.raw != (util.Block{}) {
		b.freeStarted = true
		b.sourceRaw = b.raw
		e = m.memory.Deallocate(&b.raw)
		if b.raw == (util.Block{}) {
			b.pendingOwns = true
			b.rawDestroyed = true
		}
		if e != nil {
			return e
		}
	}
	b.raw = b.pending
	b.pending = util.Block{}
	b.capacity = nc
	b.copied = 0
	b.freeStarted = false
	b.pendingOwns = false
	b.rawDestroyed = false
	b.sourceRaw = util.Block{}
	return nil
}
func (b *polynomialPSCRefBuffer) push(m *MultivariatePolynomialCore, p Polynomial) error {
	return b.pushWithNextReceipt(m, p, nil)
}
func (b *polynomialPSCRefBuffer) pushWithNextReceipt(m *MultivariatePolynomialCore, p Polynomial, r *polynomialPSCNextInvocationReceipt) error {
	if e := b.check(m); e != nil {
		return e
	}
	if b.nextFrame != nil {
		if b != &m.pscBuffer || r == nil || !m.matchesPSCNextInvocation(r) || r.frame != b.nextFrame || r.invocation != b.nextInvocation || !r.frame.executing {
			return ErrPolynomialPending
		}
	}
	if b.interrupted || b.incoming != (Polynomial{}) {
		return ErrPolynomialPending
	}
	if p != (Polynomial{}) {
		if e := m.incRefCore(p); e != nil {
			return e
		}
	}
	b.incoming = p
	if b.count == b.capacity {
		if e := b.expand(m); e != nil {
			b.interrupted = true
			b.sourceCause = e
			return e
		}
	}
	if b.raw == (util.Block{}) {
		if b.count >= 16 {
			b.interrupted = true
			b.sourceCause = arena.ErrBounds
			return b.sourceCause
		}
		b.inline[b.count] = p
	} else if e := pscWriteHandle(m, b.raw, b.count, p); e != nil {
		b.interrupted = true
		b.sourceCause = e
		return e
	}
	// buffer::push_back stores first, then increments its source logical position.
	b.count++
	if b.initialized < b.count {
		b.initialized = b.count
	}
	b.incoming = Polynomial{}
	return nil
}
func (b *polynomialPSCRefBuffer) finalize(m *MultivariatePolynomialCore, owning bool) error {
	return b.finalizePhase(m, owning, owning)
}
func (b *polynomialPSCRefBuffer) finalizePhase(m *MultivariatePolynomialCore, owning, abort bool) error {
	if b == nil || b.self != b || b.owner != m {
		return ErrPolynomialOwner
	}
	if m == nil || m.self != m {
		return ErrPolynomialOwner
	}
	if b.finishing || m.callbackDepth != 0 {
		return ErrPolynomialPending
	}
	b.finishing = true
	defer func() { b.finishing = false }()
	if b.nextFrame != nil {
		if m == nil || m.self != m || b != &m.pscBuffer || b.nextFrame != &m.pscNext || b.nextInvocation != m.pscNext.invocation {
			return ErrPolynomialOwner
		}
		if b.nextFrame.executing {
			return ErrPolynomialPending
		}
	}
	if !b.finalized {
		if b.cleanupCause != nil && !owning {
			return b.cleanupCause
		}
		for b.cursor < b.count {
			var p Polynomial
			var e error
			if owning && b.pendingOwns {
				p, e = pscReadHandle(m, b.pending, b.cursor)
			} else {
				p, e = b.physical(m, b.cursor)
			}
			if e != nil {
				b.cleanupCause = e
				return e
			}
			if b.drainDec && !owning {
				return b.cleanupCause
			}
			if e = polynomialReferenceDrop(m, &p, &b.drainDec); e != nil {
				b.cleanupCause = e
				return e
			}
			b.drainDec = false
			b.cursor++
		}
		if b.raw != (util.Block{}) {
			if e := m.memory.Deallocate(&b.raw); e != nil {
				b.cleanupCause = e
				return e
			}
		}
		b.finalized = true
		if b.nextFrame != nil && b.nextFrame == &m.pscNext && b.nextInvocation == m.pscNext.invocation && b.invocation == m.pscNext.bufferInvocation {
			m.pscNext.bufferDestroyed = true
		}
		b.cleanupCause = nil
	}
	if abort {
		if e := polynomialReferenceDrop(m, &b.incoming, &b.incomingDec); e != nil {
			b.cleanupCause = e
			return e
		}
		if b.pending != (util.Block{}) {
			if e := m.memory.Deallocate(&b.pending); e != nil {
				b.cleanupCause = e
				return e
			}
		}
	}
	return nil
}
