// Source: src/sat/sat_solver.h:286,445; src/util/vector.h:59-120,187-201,342-387.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Solver bin_clause is std::pair<literal,literal>, not the distinct learned-bit
// sat::bin_clause record. Two complete source uint32 words occupy actual8 bytes.
type satSolverBinPair struct{ first, second uint32 }

func newSATSolverBinPair(a, b satLiteral) satSolverBinPair {
	return satSolverBinPair{a.index(), b.index()}
}
func (p satSolverBinPair) Literal1() satLiteral { return satLiteral(p.first) }
func (p satSolverBinPair) Literal2() satLiteral { return satLiteral(p.second) }

type satSolverBinPairVector struct {
	self            *satSolverBinPairVector
	raw             satSolverAssignmentColumn
	budget          *arena.Budget
	ownerBytes      uint64
	closing, closed bool
	sourceCause     error
	acknowledged    bool
}

func newSATSolverBinPairVector(r *memory.Runtime, b *arena.Budget) (*satSolverBinPairVector, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(satSolverBinPairVector{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	v := &satSolverBinPairVector{budget: b, ownerBytes: bytes}
	v.self = v
	v.raw.bind(r)
	return v, nil
}
func (v *satSolverBinPairVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	return v.raw.open()
}
func (v *satSolverBinPairVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.raw.size()
}
func (v *satSolverBinPairVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	return v.raw.shrink(0)
}
func (v *satSolverBinPairVector) Push(p satSolverBinPair) error {
	if e := v.open(); e != nil {
		return e
	}
	n, e := v.raw.size()
	if e != nil {
		return e
	}
	cap, e := v.raw.capacity()
	if e != nil {
		return e
	}
	if n == cap {
		if e = v.raw.expandStride(8); e != nil {
			return e
		}
	}
	if e = v.raw.runtime.WriteUint32(v.raw.block, 8+uint64(n)*8, p.first); e != nil {
		return e
	}
	if e = v.raw.runtime.WriteUint32(v.raw.block, 12+uint64(n)*8, p.second); e != nil {
		return e
	}
	return v.raw.runtime.WriteUint32(v.raw.block, 4, n+1)
}
func (v *satSolverBinPairVector) Get(i uint32) (satSolverBinPair, error) {
	if e := v.open(); e != nil {
		return satSolverBinPair{}, e
	}
	if satStorageSourceDebug {
		n, e := v.raw.size()
		if e != nil {
			return satSolverBinPair{}, e
		}
		if i >= n {
			return satSolverBinPair{}, ErrSATAssertion
		}
	}
	a, e := v.raw.runtime.ReadUint32(v.raw.block, 8+uint64(i)*8)
	if e != nil {
		return satSolverBinPair{}, e
	}
	b, e := v.raw.runtime.ReadUint32(v.raw.block, 12+uint64(i)*8)
	return satSolverBinPair{a, b}, e
}

// SourceFinalize preserves a reached source raw-pointer guard separately from
// explicit owning cleanup of an unpublished realloc replacement.
func (v *satSolverBinPairVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	if v.sourceCause != nil {
		return v.sourceCause
	}
	if v.raw.block != (memory.Block{}) {
		if e := v.raw.runtime.Deallocate(&v.raw.block); e != nil {
			v.sourceCause = e
			v.closing = true
			return e
		}
	}
	return nil
}
func (v *satSolverBinPairVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	if v.sourceCause != nil && !v.acknowledged {
		return v.sourceCause
	}
	if !v.acknowledged {
		if e := v.SourceFinalize(); e != nil {
			return e
		}
	}
	return v.owningAbort()
}
func (v *satSolverBinPairVector) CleanupStopped() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if !v.closing || v.sourceCause == nil {
		return ErrSATOwner
	}
	v.acknowledged = true
	return v.owningAbort()
}
func (v *satSolverBinPairVector) owningAbort() error {
	if e := v.raw.finalize(); e != nil {
		return e
	}
	v.budget.Release(v.ownerBytes)
	v.ownerBytes = 0
	v.closed = true
	return nil
}
