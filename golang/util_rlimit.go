// Licensed under the MIT license.
// Translation of src/util/rlimit.h/.cpp, Nikolaj Bjorner, 2015-09-28.
package g3

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/glycerine/g3/golang/arena"
)

// The source OpenMP critical section is process-wide, including recursive
// propagation. Counts/scopes/suspension are confined to their operation owner;
// cancellation and child registration may run concurrently with increments.
var utilReslimitCancelMu sync.Mutex

// This is a cold resource service, with explicitly borrowed child services.
// Children must form an acyclic graph and outlive their registration. Do not
// copy an initialized service (atomic state and owned vectors would alias).
type utilReslimit struct {
	cancelCount           atomic.Uint32
	suspended             bool
	steps, limit          uint64
	limits                []uint64
	children              []*utilReslimit
	budget                *arena.Budget
	headerCharged, closed bool
}

func newUtilReslimit(budget *arena.Budget) (*utilReslimit, error) {
	if err := budget.Reserve(uint64(reflect.TypeOf(utilReslimit{}).Size())); err != nil {
		return nil, err
	}
	return &utilReslimit{budget: budget, headerCharged: true}, nil
}

func utilLimitCapacity(old int, width uint64) (int, error) {
	next := uint64(2)
	if old != 0 {
		next = (3*uint64(old) + 1) / 2
	}
	if next <= uint64(old) || next > uint64(^uint(0)>>1)/width || next > (uint64(^uint32(0))-8)/width {
		return 0, arena.ErrBounds
	}
	return int(next), nil
}

func (r *utilReslimit) checkOpen() {
	if r.closed {
		panic("g3: closed resource limit")
	}
}
func (r *utilReslimit) count() uint64 { r.checkOpen(); return r.steps }
func (r *utilReslimit) inc() bool     { return r.incOffset(1) }
func (r *utilReslimit) incOffset(offset uint32) bool {
	r.checkOpen()
	r.steps += uint64(offset)
	return (r.cancelCount.Load() == 0 && (r.limit == 0 || r.steps <= r.limit)) || r.suspended
}
func (r *utilReslimit) push(delta uint32) error {
	r.checkOpen()
	if len(r.limits) == cap(r.limits) {
		n, err := utilLimitCapacity(cap(r.limits), 8)
		if err != nil {
			return err
		}
		if err := r.budget.Reserve(uint64(n) * 8); err != nil {
			return err
		}
		v := make([]uint64, len(r.limits), n)
		copy(v, r.limits)
		r.budget.Release(uint64(cap(r.limits)) * 8)
		r.limits = v
	}
	next := r.steps + uint64(delta)
	if next <= r.steps {
		next = 0
	}
	r.limits = append(r.limits, r.limit)
	if r.limit == 0 || next < r.limit {
		r.limit = next
	}
	// Deliberately local: source push/pop do not call set_cancel.
	r.cancelCount.Store(0)
	return nil
}
func (r *utilReslimit) pop() {
	r.checkOpen()
	if len(r.limits) == 0 {
		panic("g3: resource limit stack underflow")
	}
	if r.steps > r.limit && r.limit > 0 {
		r.steps = r.limit
	}
	r.limit = r.limits[len(r.limits)-1]
	r.limits = r.limits[:len(r.limits)-1]
	r.cancelCount.Store(0)
}
func (r *utilReslimit) cancelFlag() bool { return r.cancelCount.Load() > 0 && !r.suspended }
func (r *utilReslimit) cancelMessage() string {
	if r.cancelCount.Load() > 0 {
		return Z3CanceledMessage
	}
	return Z3MaxResourceMessage
}
func (r *utilReslimit) pushChild(child *utilReslimit) error {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	r.checkOpen()
	if child == nil || child == r {
		panic("g3: invalid resource limit child")
	}
	width := uint64(reflect.TypeOf((*utilReslimit)(nil)).Size())
	if len(r.children) == cap(r.children) {
		n, err := utilLimitCapacity(cap(r.children), width)
		if err != nil {
			return err
		}
		if err := r.budget.Reserve(uint64(n) * width); err != nil {
			return err
		}
		v := make([]*utilReslimit, len(r.children), n)
		copy(v, r.children)
		r.budget.Release(uint64(cap(r.children)) * width)
		r.children = v
	}
	// Source registration does not copy the parent's existing cancel flag.
	r.children = append(r.children, child)
	return nil
}
func (r *utilReslimit) popChild() {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	r.checkOpen()
	if len(r.children) == 0 {
		panic("g3: resource child stack underflow")
	}
	i := len(r.children) - 1
	r.children[i] = nil
	r.children = r.children[:i]
}
func (r *utilReslimit) setCancel(f uint32) {
	r.cancelCount.Store(f)
	for _, child := range r.children {
		child.setCancel(f)
	}
}
func (r *utilReslimit) cancel() { r.incCancel() }
func (r *utilReslimit) incCancel() {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	r.checkOpen()
	r.setCancel(r.cancelCount.Load() + 1)
}
func (r *utilReslimit) decCancel() {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	r.checkOpen()
	if f := r.cancelCount.Load(); f > 0 {
		r.setCancel(f - 1)
	}
}
func (r *utilReslimit) resetCancel() {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	r.checkOpen()
	r.setCancel(0)
}
func (r *utilReslimit) close() {
	utilReslimitCancelMu.Lock()
	defer utilReslimitCancelMu.Unlock()
	if r.closed {
		return
	}
	r.budget.Release(uint64(cap(r.limits)) * 8)
	r.budget.Release(uint64(cap(r.children)) * uint64(reflect.TypeOf((*utilReslimit)(nil)).Size()))
	if r.headerCharged {
		r.budget.Release(uint64(reflect.TypeOf(utilReslimit{}).Size()))
	}
	r.limits = nil
	r.children = nil
	r.closed = true
}

// Explicit close replaces each single C++ RAII destructor; repeated close is
// harmless. Scope values must not be copied while active. Use defer scope.close.
type utilLimitScope struct {
	owner  *utilReslimit
	active bool
}

func newUtilLimitScope(r *utilReslimit, delta uint32) (utilLimitScope, error) {
	if err := r.push(delta); err != nil {
		return utilLimitScope{}, err
	}
	return utilLimitScope{r, true}, nil
}
func (s *utilLimitScope) close() {
	if s.active {
		s.owner.pop()
		s.active = false
	}
}

type utilSuspendLimit struct {
	owner  *utilReslimit
	active bool
}

func newUtilSuspendLimit(r *utilReslimit) utilSuspendLimit {
	r.checkOpen()
	r.suspended = true
	return utilSuspendLimit{r, true}
}
func (s *utilSuspendLimit) close() {
	if s.active {
		s.owner.suspended = false
		s.active = false
	}
}

type utilChildLimits struct {
	owner  *utilReslimit
	size   uint32
	closed bool
}

func newUtilChildLimits(r *utilReslimit) utilChildLimits { return utilChildLimits{owner: r} }
func (s *utilChildLimits) pushChild(r *utilReslimit) error {
	if s.closed {
		panic("g3: closed child scope")
	}
	if s.size == ^uint32(0) {
		return arena.ErrBounds
	}
	if err := s.owner.pushChild(r); err != nil {
		return err
	}
	s.size++
	return nil
}
func (s *utilChildLimits) close() {
	if s.closed {
		return
	}
	for s.size > 0 {
		s.owner.popChild()
		s.size--
	}
	s.closed = true
}
