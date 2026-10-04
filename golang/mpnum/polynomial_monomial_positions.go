// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
// Concrete translation of polynomial.cpp monomial2pos (449–505).
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilid"
	"reflect"
)

// MonomialPositions borrows its manager and Runtime. Neither monomials nor the
// manager are retained. The borrower must be closed before those owners.
// raw is the source unsigned_vector; pending is an actual unreturned initial
// allocation, retained solely for explicit Go owning cleanup after an error.
// Implicit copies of this owning wrapper are rejected.
type MonomialPositions struct {
	self                            *MonomialPositions
	manager                         *MonomialManager
	raw, pending                    memory.Block
	header                          uint64
	interrupted, finalizing, closed bool
	partialRaw                      bool
}

func NewMonomialPositions(m *MonomialManager) (*MonomialPositions, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(MonomialPositions{}).Size())
	if e := m.config.Budget.Reserve(n); e != nil {
		return nil, e
	}
	p := &MonomialPositions{manager: m, header: n}
	p.self = p
	return p, nil
}
func (p *MonomialPositions) identity() error {
	if p == nil || p.self != p {
		return ErrMonomialOwner
	}
	if p.closed {
		return arena.ErrClosed
	}
	return nil
}
func (p *MonomialPositions) check() error {
	if e := p.identity(); e != nil {
		return e
	}
	if p.interrupted || p.finalizing {
		return ErrMonomialPending
	}
	return p.manager.check()
}
func (p *MonomialPositions) id(a Monomial) (uint32, error) {
	l, e := p.manager.location(a)
	if e != nil {
		return 0, e
	}
	// monomial::id reads this word only, not RC/degree/size/hash.
	return p.manager.allocator.ReadUint32(l.raw, 4)
}
func (p *MonomialPositions) size() (uint32, error) {
	if p.raw == (memory.Block{}) {
		return 0, nil
	}
	return p.manager.memory.ReadUint32(p.raw, 4)
}
func (p *MonomialPositions) capacity() (uint32, error) {
	if p.raw == (memory.Block{}) {
		return 0, nil
	}
	return p.manager.memory.ReadUint32(p.raw, 0)
}
func (p *MonomialPositions) expand() error {
	r := p.manager.memory
	old, e := p.capacity()
	if e != nil {
		return e
	}
	if p.raw != (memory.Block{}) && old == 0 {
		if modularSourceDebug {
			return ErrMonomialAssertion
		}
		return memory.Failure{Kind: memory.FailureVectorOverflow}
	}
	nc, nb, e := utilid.VectorGrowth(old)
	if e != nil {
		return e
	}
	if p.raw == (memory.Block{}) {
		p.pending, e = r.Allocate(uint64(nb))
		if e != nil {
			return e
		}
		if e = r.WriteUint32(p.pending, 0, nc); e != nil {
			return e
		}
		if e = r.WriteUint32(p.pending, 4, 0); e != nil {
			return e
		}
		p.raw = p.pending
		p.pending = memory.Block{}
		return nil
	}
	before := p.raw
	if e = r.Reallocate(&p.raw, uint64(nb)); e != nil {
		if p.raw != before {
			p.interrupted = true
			p.partialRaw = r.SizeTBits() == 32 && uint32(nb+4) < 4
		}
		return e
	}
	return r.WriteUint32(p.raw, 0, nc)
}
func (p *MonomialPositions) reserve(n uint32) error {
	old, e := p.size()
	if e != nil {
		return e
	}
	if n <= old {
		return nil
	}
	for {
		c, e := p.capacity()
		if e != nil {
			return e
		}
		if n <= c {
			break
		}
		if e = p.expand(); e != nil {
			return e
		}
	}
	r := p.manager.memory
	if e = r.WriteUint32(p.raw, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = r.WriteUint32(p.raw, 8+uint64(i)*4, ^uint32(0)); e != nil {
			return e
		}
	}
	return nil
}
func (p *MonomialPositions) read(id uint32) (uint32, error) {
	if modularSourceDebug {
		n, e := p.size()
		if e != nil {
			return 0, e
		}
		if id >= n {
			return 0, ErrMonomialAssertion
		}
	}
	return p.manager.memory.ReadUint32(p.raw, 8+uint64(id)*4)
}
func (p *MonomialPositions) fail(e error) error {
	if e != nil && p.pending != (memory.Block{}) {
		p.interrupted = true
	}
	return e
}
func (p *MonomialPositions) Get(a Monomial) (uint32, error) {
	if e := p.check(); e != nil {
		return 0, e
	}
	id, e := p.id(a)
	if e != nil {
		return 0, e
	}
	if e = p.reserve(id + 1); e != nil {
		return 0, p.fail(e)
	}
	return p.read(id)
}
func (p *MonomialPositions) Reset(a Monomial) error {
	if e := p.check(); e != nil {
		return e
	}
	id, e := p.id(a)
	if e != nil {
		return e
	}
	if modularSourceDebug {
		n, e := p.size()
		if e != nil {
			return e
		}
		if id >= n {
			return ErrMonomialAssertion
		}
	}
	return p.manager.memory.WriteUint32(p.raw, 8+uint64(id)*4, ^uint32(0))
}
func (p *MonomialPositions) Set(a Monomial, pos uint32) error {
	if e := p.check(); e != nil {
		return e
	}
	id, e := p.id(a)
	if e != nil {
		return e
	}
	if e = p.reserve(id + 1); e != nil {
		return p.fail(e)
	}
	if modularSourceDebug {
		v, e := p.read(id)
		if e != nil {
			return e
		}
		if v != ^uint32(0) {
			return ErrMonomialAssertion
		}
	}
	return p.manager.memory.WriteUint32(p.raw, 8+uint64(id)*4, pos)
}

// These concrete arrays replace only the two source Poly template loops. The
// actual polynomial imp overload remains unported; no stand-in Poly interface.
func (p *MonomialPositions) SetMonomials(xs []Monomial) error {
	if e := p.check(); e != nil {
		return e
	}
	if uint64(len(xs)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	for i, a := range xs {
		if e := p.Set(a, uint32(i)); e != nil {
			return e
		}
	}
	return nil
}
func (p *MonomialPositions) ResetMonomials(xs []Monomial) error {
	if e := p.check(); e != nil {
		return e
	}
	if uint64(len(xs)) > uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	for _, a := range xs {
		if e := p.Reset(a); e != nil {
			return e
		}
	}
	return nil
}

// SourceFinalize executes only the implicit vector member destructor. It does
// not destroy an allocation that failed before m_data publication.
func (p *MonomialPositions) SourceFinalize() error {
	if e := p.identity(); e != nil {
		return e
	}
	p.finalizing = true
	if p.raw != (memory.Block{}) {
		if e := p.manager.memory.Deallocate(&p.raw); e != nil {
			return e
		}
	}
	return nil
}

// CloseChecked is explicit Go owning cleanup, including pending lexical malloc.
// It stops at a refused release and retains all remaining scalar roots.
func (p *MonomialPositions) CloseChecked() error {
	if p == nil || p.self != p {
		return ErrMonomialOwner
	}
	if p.closed {
		return nil
	}
	p.finalizing = true
	if p.raw != (memory.Block{}) {
		var e error
		if p.partialRaw {
			e = p.manager.memory.CleanupPartial(&p.raw)
		} else {
			e = p.manager.memory.Deallocate(&p.raw)
		}
		if e != nil {
			return e
		}
		p.partialRaw = false
	}
	if p.pending != (memory.Block{}) {
		if e := p.manager.memory.Deallocate(&p.pending); e != nil {
			return e
		}
	}
	p.manager.config.Budget.Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
