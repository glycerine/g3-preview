// Source: src/sat/sat_solver.h:286,445; src/util/vector.h:59-120,187-201,342-387.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Solver bin_clause is std::pair<literal,literal>, not the distinct learned-bit
// sat::bin_clause record. Two complete source uint32 words occupy actual8 bytes.
type satSolverBinPair struct{ first, second uint32 }

func newSATSolverBinPair(a, b Literal) satSolverBinPair {
	return satSolverBinPair{a.index(), b.index()}
}
func (p satSolverBinPair) Literal1() Literal { return Literal(p.first) }
func (p satSolverBinPair) Literal2() Literal { return Literal(p.second) }

type BinClauseBuffer struct {
	self            *BinClauseBuffer
	raw             RuntimeVectorColumn
	budget          *arena.Budget
	ownerBytes      uint64
	closing, closed bool
	sourceCause     error
	acknowledged    bool
}

func NewBinClauseBuffer(r *util.Runtime, b *arena.Budget) (*BinClauseBuffer, error) {
	if r == nil || r.Budget() != b {
		return nil, ErrSATOwner
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(BinClauseBuffer{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	v := &BinClauseBuffer{budget: b, ownerBytes: bytes}
	v.self = v
	v.raw.Bind(r)
	return v, nil
}
func (v *BinClauseBuffer) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	return v.raw.Open()
}
func (v *BinClauseBuffer) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.raw.Size()
}
func (v *BinClauseBuffer) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	return v.raw.Shrink(0)
}
func (v *BinClauseBuffer) Push(p satSolverBinPair) error {
	if e := v.open(); e != nil {
		return e
	}
	n, e := v.raw.Size()
	if e != nil {
		return e
	}
	cap, e := v.raw.Capacity()
	if e != nil {
		return e
	}
	if n == cap {
		if e = v.raw.ExpandStride(8); e != nil {
			return e
		}
	}
	if e = v.raw.Runtime.WriteUint32(v.raw.Block, 8+uint64(n)*8, p.first); e != nil {
		return e
	}
	if e = v.raw.Runtime.WriteUint32(v.raw.Block, 12+uint64(n)*8, p.second); e != nil {
		return e
	}
	return v.raw.Runtime.WriteUint32(v.raw.Block, 4, n+1)
}
func (v *BinClauseBuffer) Get(i uint32) (satSolverBinPair, error) {
	if e := v.open(); e != nil {
		return satSolverBinPair{}, e
	}
	if satStorageSourceDebug {
		n, e := v.raw.Size()
		if e != nil {
			return satSolverBinPair{}, e
		}
		if i >= n {
			return satSolverBinPair{}, ErrSATAssertion
		}
	}
	a, e := v.raw.Runtime.ReadUint32(v.raw.Block, 8+uint64(i)*8)
	if e != nil {
		return satSolverBinPair{}, e
	}
	b, e := v.raw.Runtime.ReadUint32(v.raw.Block, 12+uint64(i)*8)
	return satSolverBinPair{a, b}, e
}

// SourceFinalize preserves a reached source raw-pointer guard separately from
// explicit owning cleanup of an unpublished realloc replacement.
func (v *BinClauseBuffer) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	if v.sourceCause != nil {
		return v.sourceCause
	}
	if v.raw.Block != (util.Block{}) {
		if e := v.raw.Runtime.Deallocate(&v.raw.Block); e != nil {
			v.sourceCause = e
			v.closing = true
			return e
		}
	}
	return nil
}
func (v *BinClauseBuffer) Close() error {
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
func (v *BinClauseBuffer) CleanupStopped() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if !v.closing || v.sourceCause == nil {
		return ErrSATOwner
	}
	v.acknowledged = true
	return v.owningAbort()
}
func (v *BinClauseBuffer) owningAbort() error {
	if e := v.raw.Finalize(); e != nil {
		return e
	}
	v.budget.Release(v.ownerBytes)
	v.ownerBytes = 0
	v.closed = true
	return nil
}
