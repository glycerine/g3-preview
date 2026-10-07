package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// One concrete source literal_vector, not a directory of arena lists. Scalar
// root identity names this vector object and survives its raw reallocations.
// Root identity/header are charged Go lifetime adapters; no source malloc event
// is emitted for them. Actual literal payload is the source unsigned m_val.
type satSolverRuntimeTrail struct {
	self               *satSolverRuntimeTrail
	values             RuntimeUintColumn
	budget             *arena.Budget
	origin, ownerBytes uint64
	live, closed       bool
}

func newSATSolverRuntimeTrail(b *arena.Budget, r *util.Runtime) (*satSolverRuntimeTrail, satSolverLiteralListRef, error) {
	if r == nil {
		return nil, satSolverLiteralListRef{}, ErrSATOwner
	}
	if _, e := r.AllocationSize(); e != nil {
		return nil, satSolverLiteralListRef{}, e
	}
	n := uint64(reflect.TypeOf(satSolverRuntimeTrail{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, satSolverLiteralListRef{}, e
	}
	id, e := satNextOwner()
	if e != nil {
		b.Release(n)
		return nil, satSolverLiteralListRef{}, e
	}
	t := &satSolverRuntimeTrail{budget: b, origin: id, ownerBytes: n, live: true}
	t.self = t
	t.values.Bind(r)
	return t, satSolverLiteralListRef{owner: id, generation: 1}, nil
}
func (t *satSolverRuntimeTrail) open() error {
	if t == nil || t.self != t {
		return ErrSATOwner
	}
	if t.closed {
		return arena.ErrClosed
	}
	return nil
}
func (t *satSolverRuntimeTrail) check(r satSolverLiteralListRef) error {
	if e := t.open(); e != nil {
		return e
	}
	if r.owner != t.origin {
		return ErrSATOwner
	}
	if !t.live || r.slot != 0 || r.generation != 1 {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (t *satSolverRuntimeTrail) Size(r satSolverLiteralListRef) (uint32, error) {
	if e := t.check(r); e != nil {
		return 0, e
	}
	return t.values.Size()
}
func (t *satSolverRuntimeTrail) Push(r satSolverLiteralListRef, l Literal) error {
	if e := t.check(r); e != nil {
		return e
	}
	return t.values.Push(uint32(l))
}
func (t *satSolverRuntimeTrail) Read(r satSolverLiteralListRef, i uint32) (Literal, error) {
	if e := t.check(r); e != nil {
		return 0, e
	}
	x, e := t.values.Read(i)
	return Literal(x), e
}
func (t *satSolverRuntimeTrail) Shrink(r satSolverLiteralListRef, n uint32) error {
	if e := t.check(r); e != nil {
		return e
	}
	return t.values.Shrink(n)
}
func (t *satSolverRuntimeTrail) IsLive(r satSolverLiteralListRef) bool { return t.check(r) == nil }
func (t *satSolverRuntimeTrail) Free(r satSolverLiteralListRef) error {
	if e := t.check(r); e != nil {
		return e
	}
	if e := t.values.Finalize(); e != nil {
		return e
	}
	t.live = false
	return nil
}
func (t *satSolverRuntimeTrail) Close() error {
	if t == nil || t.self != t {
		return ErrSATOwner
	}
	if t.closed {
		return nil
	}
	if e := t.values.Finalize(); e != nil {
		return e
	}
	t.live = false
	t.budget.Release(t.ownerBytes)
	t.ownerBytes = 0
	t.closed = true
	return nil
}
