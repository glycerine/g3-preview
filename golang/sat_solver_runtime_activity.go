// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Actual solver m_activity. Its uint32 payload belongs solely to the borrowed
// Runtime. This cold owner is distinct from the separately ported public domain.
type satSolverRuntimeActivity struct {
	self       *satSolverRuntimeActivity
	values     satSolverRuntimeUintColumn
	budget     *arena.Budget
	ownerBytes uint64
	closed     bool
}

func newSATSolverRuntimeActivity(b *arena.Budget, r *memory.Runtime) (*satSolverRuntimeActivity, error) {
	if r == nil {
		return nil, ErrSATOwner
	}
	if _, e := r.AllocationSize(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(satSolverRuntimeActivity{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	a := &satSolverRuntimeActivity{budget: b, ownerBytes: n}
	a.self = a
	a.values.bind(r)
	return a, nil
}
func (a *satSolverRuntimeActivity) open() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed {
		return arena.ErrClosed
	}
	return nil
}
func (a *satSolverRuntimeActivity) Get(i uint32) (uint32, error) {
	if e := a.open(); e != nil {
		return 0, e
	}
	return a.values.read(i)
}
func (a *satSolverRuntimeActivity) Set(i, x uint32) error {
	if e := a.open(); e != nil {
		return e
	}
	return a.values.write(i, x)
}
func (a *satSolverRuntimeActivity) Push(x uint32) error {
	if e := a.open(); e != nil {
		return e
	}
	return a.values.push(x)
}
func (a *satSolverRuntimeActivity) Size() uint32 {
	if a.open() != nil {
		return 0
	}
	n, _ := a.values.size()
	return n
}
func (a *satSolverRuntimeActivity) Stats() satStorageVectorStats {
	if a.open() != nil {
		return satStorageVectorStats{}
	}
	return satStorageVectorStats{OwnerBytes: a.ownerBytes}
}
func (a *satSolverRuntimeActivity) Close() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed {
		return nil
	}
	if e := a.values.finalize(); e != nil {
		return e
	}
	a.budget.Release(a.ownerBytes)
	a.ownerBytes = 0
	a.closed = true
	return nil
}

// sat_solver.h:517-528 and sat_solver.cpp:2804-2812. Queue event precedes
// rescaling; neither multiplication nor addition is widened beyond uint32.
func (v *satSolverVariableState) incActivity(id satBoolVar) error {
	if e := v.open(); e != nil {
		return e
	}
	act, e := v.activity.Get(uint32(id))
	if e != nil {
		return e
	}
	if e = v.activity.Set(uint32(id), act+v.activityInc); e != nil {
		return e
	}
	if e = v.queue.ActivityIncreased(id); e != nil {
		return e
	}
	act, e = v.activity.Get(uint32(id))
	if e != nil {
		return e
	}
	if act > 1<<24 {
		return v.rescaleActivity()
	}
	return nil
}
func (v *satSolverVariableState) decayActivity() { v.activityInc *= 11; v.activityInc /= 10 }
func (v *satSolverVariableState) rescaleActivity() error {
	if e := v.open(); e != nil {
		return e
	}
	c := &v.activity.values.raw
	n, e := c.size()
	if e != nil {
		return e
	}
	h := c.block
	for i := uint32(0); i < n; i++ {
		x, e := satAssignmentReadWord(c.runtime, h, 8+4*uint64(i))
		if e != nil {
			return e
		}
		if e = satAssignmentWriteWord(c.runtime, h, 8+4*uint64(i), x>>14); e != nil {
			return e
		}
	}
	v.activityInc >>= 14
	return nil
}
