// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
)

func (m *MonomialManager) writePower(a Monomial, i uint32, p MonomialPower) error {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	if e = m.allocator.WriteUint32(l.raw, 20+uint64(i)*8, p.Var); e != nil {
		return e
	}
	return m.allocator.WriteUint32(l.raw, 24+uint64(i)*8, p.Degree)
}
func (m *MonomialManager) swapPowers(a Monomial, i, j uint32) error {
	x, e := m.powerCore(a, i)
	if e != nil {
		return e
	}
	y, e := m.powerCore(a, j)
	if e != nil {
		return e
	}
	if e = m.writePower(a, i, y); e != nil {
		return e
	}
	return m.writePower(a, j, x)
}
func (m *MonomialManager) siftPowers(a Monomial, root, end uint32) error {
	for {
		child := uint64(root)*2 + 1
		if child >= uint64(end) {
			return nil
		}
		i := uint32(child)
		p, e := m.powerCore(a, i)
		if e != nil {
			return e
		}
		if i+1 < end {
			q, e := m.powerCore(a, i+1)
			if e != nil {
				return e
			}
			if p.Var < q.Var {
				i++
				p = q
			}
		}
		r, e := m.powerCore(a, root)
		if e != nil {
			return e
		}
		if r.Var >= p.Var {
			return nil
		}
		if e = m.swapPowers(a, root, i); e != nil {
			return e
		}
		root = i
	}
}
func (m *MonomialManager) renameObject(a Monomial, xs []uint32) error {
	h, e := m.headerCore(a)
	if e != nil {
		return e
	}
	for i := uint32(0); i < h.Size; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return e
		}
		if uint64(p.Var) >= uint64(len(xs)) {
			return arena.ErrBounds
		}
		p.Var = xs[p.Var]
		if e = m.writePower(a, i, p); e != nil {
			return e
		}
	}
	for i := h.Size / 2; i > 0; {
		i--
		if e = m.siftPowers(a, i, h.Size); e != nil {
			return e
		}
	}
	for end := h.Size; end > 1; {
		end--
		if e = m.swapPowers(a, 0, end); e != nil {
			return e
		}
		if e = m.siftPowers(a, 0, end); e != nil {
			return e
		}
	}
	var problem error
	hash := utilhash.WordReader(h.Size*8, 11, func(i uint32) uint32 {
		l, e := m.location(a)
		if e != nil {
			problem = e
			return 0
		}
		v, e := m.allocator.ReadUint32(l.raw, 20+uint64(i)*4)
		if e != nil {
			problem = e
		}
		return v
	})
	if problem != nil {
		return problem
	}
	l, e := m.location(a)
	if e != nil {
		return e
	}
	return m.allocator.WriteUint32(l.raw, 16, hash)
}
func (m *MonomialManager) Rename(xs []uint32) (err error) {
	if e := m.check(); e != nil {
		return e
	}
	if modularSourceDebug && (m.managerRefs > 1 || uint64(len(xs)) != uint64(m.nextVar)) {
		return ErrMonomialAssertion
	}
	if m.renameTable.raw != (memory.Block{}) || m.renameTable.pending != (memory.Block{}) || m.renameFound != (memory.Block{}) {
		return ErrMonomialPending
	}
	if modularSourceDebug {
		var capacity, size uint32
		for size < m.nextVar {
			if size == capacity {
				n := uint32(2)
				if capacity > 0 {
					n = (3*capacity + 1) >> 1
				}
				bytes := n + 8
				if capacity > 0 && (n <= capacity || bytes <= capacity+8) {
					return memory.Failure{Kind: memory.FailureVectorOverflow}
				}
				if m.renameFound == (memory.Block{}) {
					raw, e := m.memory.Allocate(uint64(bytes))
					m.renameFound = raw
					if e != nil {
						return e
					}
					if e = m.memory.WriteUint32(raw, 4, 0); e != nil {
						return e
					}
				} else {
					if e := m.memory.Reallocate(&m.renameFound, uint64(bytes)); e != nil {
						return e
					}
				}
				if e := m.memory.WriteUint32(m.renameFound, 0, n); e != nil {
					return e
				}
				capacity = n
			}
			if e := m.memory.Write(m.renameFound, uint64(size)+8, 0); e != nil {
				return e
			}
			size++
			if e := m.memory.WriteUint32(m.renameFound, 4, size); e != nil {
				return e
			}
		}
		for _, x := range xs {
			if x >= m.nextVar {
				return ErrMonomialAssertion
			}
			v, e := m.memory.Read(m.renameFound, uint64(x)+8)
			if e != nil {
				return e
			}
			if v != 0 {
				return ErrMonomialAssertion
			}
			if e = m.memory.Write(m.renameFound, uint64(x)+8, 1); e != nil {
				return e
			}
		}
		if e := m.memory.Deallocate(&m.renameFound); e != nil {
			return e
		}
	}
	t := &m.renameTable
	if e := t.construct(m, 8, 2); e != nil {
		return e
	}
	defer func() {
		if cleanup := t.sourceDestroy(m); cleanup != nil {
			err = errors.Join(err, cleanup)
		}
	}()
	for head := uint32(0); head < m.table.slots; head++ {
		next, e := m.table.next(m, head)
		if e != nil {
			return e
		}
		if next == monomialFreeLink {
			continue
		}
		for i := head; ; {
			k, e := m.table.key(m, i)
			if e != nil {
				return e
			}
			if k.kind != monomialObjectKey {
				return ErrMonomialPending
			}
			a := Monomial{m.origin, k.slot, m.locations[k.slot-1].generation}
			if e = m.renameObject(a, xs); e != nil {
				return e
			}
			if modularSourceDebug {
				_, found, e := t.find(m, k)
				if e != nil {
					return e
				}
				if found {
					return ErrMonomialAssertion
				}
			}
			_, inserted, e := t.insert(m, k, false)
			if e != nil {
				return e
			}
			if modularSourceDebug && !inserted {
				return ErrMonomialAssertion
			}
			next, e = m.table.next(m, i)
			if e != nil {
				return e
			}
			if next == 0 {
				break
			}
			i = uint32(next - 1)
		}
	}
	m.table, m.renameTable = m.renameTable, m.table
	return nil
}
func (m *MonomialManager) CleanupRename() error {
	if m == nil || m.self != m || m.closed {
		return ErrMonomialOwner
	}
	if m.renameFound != (memory.Block{}) {
		if e := m.memory.Deallocate(&m.renameFound); e != nil {
			return e
		}
	}
	return m.renameTable.cleanup(m)
}
