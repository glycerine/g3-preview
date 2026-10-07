// Source: src/sat/sat_var_queue.h:31-67; src/util/heap.h; src/sat/sat_simplifier.cpp:875–919 and src/sat/sat_solver.h case_split_queue.
package sat

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

type satVarQueue struct {
	literalSource                                             *satSimplifier
	runtimeValues, runtimeIndices                             RuntimeUintColumn
	runtimeSourceStop                                         error
	runtimeSourceCursor, runtimeAbortCursor, runtimeCompleted uint8
	runtimeClosing                                            bool
	self                                                      *satVarQueue
	activity                                                  *satActivityDomain
	runtimeActivity                                           *satSolverRuntimeActivity
	values, indices                                           []int32
	budget                                                    *arena.Budget
	ownerBytes, epoch                                         uint64
	closed                                                    bool
}
type satVarQueueStats struct{ OwnerBytes, ValuesBytes, IndicesBytes uint64 }

func NewSATVarQueue(a *satActivityDomain, b *arena.Budget) (*satVarQueue, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	q, e := newSATVarQueue(a, b)
	if e != nil {
		return nil, e
	}
	if e = q.initialize(); e != nil {
		if cleanup := q.Close(); cleanup != nil {
			return q, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return q, nil
}
func (q *satVarQueue) initialize() error {
	if q.runtimeBacked() {
		return q.initializeRuntime()
	}
	var e error
	if e = q.grow(&q.values, 1); e == nil {
		q.values = append(q.values, -1)
		e = q.reserve(128)
	}
	return e
}
func newSATVarQueue(a *satActivityDomain, b *arena.Budget) (*satVarQueue, error) {
	n := uint64(reflect.TypeOf(satVarQueue{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	q := &satVarQueue{activity: a, budget: b, ownerBytes: n, epoch: 1}
	q.self = q
	return q, nil
}
func (q *satVarQueue) open() error {
	if q == nil || q.self != q {
		return ErrSATOwner
	}

	if q.closed || q.runtimeClosing {
		return arena.ErrClosed
	}
	return nil
}
func (q *satVarQueue) change() error {
	if q.runtimeBacked() {
		return q.open()
	}
	if e := q.open(); e != nil {
		return e
	}
	if q.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (q *satVarQueue) grow(v *[]int32, need uint64) error {
	for need > uint64(cap(*v)) {
		n, e := satVectorNextCapacity(uint64(cap(*v)), 4)
		if e != nil {
			return e
		}
		if e = q.budget.Reserve(n * 4); e != nil {
			return e
		}
		next := make([]int32, len(*v), int(n))
		copy(next, *v)
		q.budget.Release(uint64(cap(*v)) * 4)
		*v = next
		if q.epoch != ^uint64(0) {
			q.epoch++
		}
	}
	return nil
}
func (q *satVarQueue) reserve(n uint32) error {
	if q.runtimeBacked() {
		return q.reserveRuntime(n)
	}
	if uint64(n) > uint64(^uint32(0)>>1) {
		return arena.ErrBounds
	}
	if uint64(n) <= uint64(len(q.indices)) {
		return nil
	}
	if e := q.grow(&q.indices, uint64(n)); e != nil {
		return e
	}
	old := len(q.indices)
	q.indices = q.indices[:int(n)]
	clear(q.indices[old:])
	return nil
}

// Compatibility signatures transport a Runtime guard as its actual typed panic.
// Solver algorithms use the checked error-returning methods instead.
func (q *satVarQueue) Size() uint32 {
	if q != nil && q.runtimeBacked() {
		n, e := q.sizeChecked()
		if e != nil {
			panic(e)
		}
		return n
	}
	if q == nil || q.closed || len(q.values) == 0 {
		return 0
	}
	return uint32(len(q.values) - 1)
}
func (q *satVarQueue) Empty() bool {
	if q != nil && q.runtimeBacked() {
		yes, e := q.emptyChecked()
		if e != nil {
			panic(e)
		}
		return yes
	}
	return q != nil && !q.closed && len(q.values) == 1
}
func (q *satVarQueue) Contains(v BoolVar) bool {
	if q != nil && q.runtimeBacked() {
		yes, e := q.containsChecked(v)
		if e != nil {
			panic(e)
		}
		return yes
	}
	return q != nil && !q.closed && uint64(v) < uint64(len(q.indices)) && q.indices[v] != 0
}
func (q *satVarQueue) lt(a, b int32) (bool, error) {
	if a < 0 || b < 0 {
		return false, arena.ErrBounds
	}
	if q.literalSource != nil {
		x, e := q.literalSource.literalWeight(uint32(a))
		if e != nil {
			return false, e
		}
		y, e := q.literalSource.literalWeight(uint32(b))
		if e != nil {
			return false, e
		}
		return x < y, nil
	}
	x, e := q.getActivity(uint32(a))
	if e != nil {
		return false, e
	}
	y, e := q.getActivity(uint32(b))
	if e != nil {
		return false, e
	}
	return x > y, nil
}
func (q *satVarQueue) put(i int32, v int32) error {
	if i < 0 || int64(i) >= int64(len(q.values)) {
		return arena.ErrBounds
	}
	q.values[i] = v
	if v < 0 || int64(v) >= int64(len(q.indices)) {
		return arena.ErrBounds
	}
	q.indices[v] = i
	return nil
}
func (q *satVarQueue) up(i int32) error {
	if i < 1 || int64(i) >= int64(len(q.values)) {
		return arena.ErrBounds
	}
	v := q.values[i]
	for {
		p := i / 2
		if p == 0 {
			break
		}
		less, e := q.lt(v, q.values[p])
		if e != nil {
			return e
		}
		if !less {
			break
		}
		if e = q.put(i, q.values[p]); e != nil {
			return e
		}
		i = p
	}
	return q.put(i, v)
}
func (q *satVarQueue) down(i int32) error {
	if i < 1 || int64(i) >= int64(len(q.values)) {
		return arena.ErrBounds
	}
	v := q.values[i]
	for {
		left := uint64(i) * 2
		if left >= uint64(len(q.values)) {
			break
		}
		c := int32(left)
		if left+1 < uint64(len(q.values)) {
			less, e := q.lt(q.values[c+1], q.values[c])
			if e != nil {
				return e
			}
			if less {
				c++
			}
		}
		less, e := q.lt(q.values[c], v)
		if e != nil {
			return e
		}
		if !less {
			break
		}
		if e = q.put(i, q.values[c]); e != nil {
			return e
		}
		i = c
	}
	return q.put(i, v)
}
func (q *satVarQueue) insert(v BoolVar) error {
	if q != nil && q.runtimeBacked() {
		return q.insertRuntime(v)
	}
	if e := q.change(); e != nil {
		return e
	}
	if uint64(v) >= uint64(len(q.indices)) {
		return arena.ErrBounds
	}
	if satStorageSourceDebug && q.Contains(v) {
		return ErrSATAssertion
	}
	if len(q.values) >= int(^uint32(0)>>1) {
		return arena.ErrBounds
	}
	i := int32(len(q.values))
	q.indices[v] = i
	if q.epoch != ^uint64(0) {
		q.epoch++
	}
	if e := q.grow(&q.values, uint64(len(q.values))+1); e != nil {
		return e
	}
	q.values = append(q.values, int32(v))
	return q.up(i)
}
func (q *satVarQueue) MkVar(v BoolVar) error {
	if e := q.change(); e != nil {
		return e
	}
	if e := q.reserve(uint32(v) + 1); e != nil {
		return e
	}
	return q.insert(v)
}
func (q *satVarQueue) UnassignVar(v BoolVar) error {
	if q.runtimeBacked() {
		yes, e := q.containsChecked(v)
		if e != nil {
			return e
		}
		if !yes {
			return q.insertRuntime(v)
		}
		return nil
	}
	if e := q.open(); e != nil {
		return e
	}
	if uint64(v) > uint64(^uint32(0)>>1) {
		return arena.ErrBounds
	}
	if !q.Contains(v) {
		return q.insert(v)
	}
	return nil
}
func (q *satVarQueue) ActivityIncreased(v BoolVar) error {
	if q.runtimeBacked() {
		yes, e := q.containsChecked(v)
		if e != nil {
			return e
		}
		if yes {
			if satStorageSourceDebug {
				present, e := q.containsChecked(v)
				if e != nil {
					return e
				}
				if !present {
					return ErrSATAssertion
				}
			}
			i, e := q.indexRuntime(int32(v))
			if e != nil {
				return e
			}
			return q.upRuntime(i)
		}
		return nil
	}
	if e := q.change(); e != nil {
		return e
	}
	if uint64(v) > uint64(^uint32(0)>>1) {
		return arena.ErrBounds
	}
	if q.Contains(v) {
		if q.epoch != ^uint64(0) {
			q.epoch++
		}
		return q.up(q.indices[v])
	}
	return nil
}
func (q *satVarQueue) DelVar(v BoolVar) error {
	if q != nil && q.runtimeBacked() {
		present, e := q.containsChecked(v)
		if e != nil {
			return e
		}
		if !present {
			return nil
		}
		return q.eraseRuntime(v)
	}
	if e := q.change(); e != nil {
		return e
	}
	if uint64(v) > uint64(^uint32(0)>>1) {
		return arena.ErrBounds
	}
	if !q.Contains(v) {
		return nil
	}
	i := q.indices[v]
	if i < 1 || int64(i) >= int64(len(q.values)) {
		return arena.ErrBounds
	}
	lastPos := len(q.values) - 1
	if int(i) == lastPos {
		q.indices[v] = 0
		q.values = q.values[:lastPos]
	} else {
		last := q.values[lastPos]
		if e := q.put(i, last); e != nil {
			return e
		}
		q.indices[v] = 0
		q.values = q.values[:lastPos]
		p := i / 2
		if p != 0 {
			less, e := q.lt(last, q.values[p])
			if e != nil {
				return e
			}
			if less {
				if q.epoch != ^uint64(0) {
					q.epoch++
				}
				return q.up(i)
			}
		}
		if e := q.down(i); e != nil {
			return e
		}
	}
	if q.epoch != ^uint64(0) {
		q.epoch++
	}
	return nil
}
func (q *satVarQueue) NextVar() (BoolVar, error) {
	if q != nil && q.runtimeBacked() {
		return q.nextRuntime()
	}
	if e := q.change(); e != nil {
		return 0, e
	}
	if len(q.values) < 2 {
		if satStorageSourceDebug {
			return 0, ErrSATAssertion
		}
		return 0, arena.ErrBounds
	}
	v := q.values[1]
	if v < 0 || int64(v) >= int64(len(q.indices)) {
		return 0, arena.ErrBounds
	}
	if len(q.values) == 2 {
		q.indices[v] = 0
		q.values = q.values[:1]
	} else {
		last := q.values[len(q.values)-1]
		if e := q.put(1, last); e != nil {
			return 0, e
		}
		q.indices[v] = 0
		q.values = q.values[:len(q.values)-1]
		if e := q.down(1); e != nil {
			return BoolVar(v), e
		}
	}
	if q.epoch != ^uint64(0) {
		q.epoch++
	}
	return BoolVar(v), nil
}
func (q *satVarQueue) Reset() error {
	if q != nil && q.runtimeBacked() {
		return q.resetRuntime()
	}
	if e := q.open(); e != nil {
		return e
	}
	if q.Empty() {
		return nil
	}
	if e := q.change(); e != nil {
		return e
	}
	clear(q.indices)
	q.values = q.values[:0]
	if e := q.grow(&q.values, 1); e != nil {
		return e
	}
	q.values = append(q.values, -1)
	q.values = q.values[:1]
	q.values[0] = -1
	if q.epoch != ^uint64(0) {
		q.epoch++
	}
	return nil
}
func (q *satVarQueue) CheckInvariant() (bool, error) {
	if q != nil && q.runtimeBacked() {
		return q.invariantRuntime()
	}
	if e := q.open(); e != nil {
		return false, e
	}
	if !satStorageSourceDebug {
		return true, nil
	}
	for i := 1; i < len(q.values); i++ {
		v := q.values[i]
		if v < 0 || int64(v) >= int64(len(q.indices)) || q.indices[v] != int32(i) {
			return false, ErrSATAssertion
		}
		if i > 1 {
			less, e := q.lt(v, q.values[i/2])
			if e != nil {
				return false, e
			}
			if less {
				return false, ErrSATAssertion
			}
		}
	}
	return true, nil
}
func (q *satVarQueue) Stats() satVarQueueStats {
	if q != nil && q.runtimeBacked() {
		return satVarQueueStats{OwnerBytes: q.ownerBytes}
	}
	return satVarQueueStats{q.ownerBytes, uint64(cap(q.values)) * 4, uint64(cap(q.indices)) * 4}
}
func (q *satVarQueue) Close() error {
	if q != nil && q.runtimeBacked() {
		return q.closeRuntime()
	}
	if q == nil || q.self != q {
		return ErrSATOwner
	}
	if q.closed {
		return nil
	}
	q.budget.Release(q.ownerBytes + uint64(cap(q.values)+cap(q.indices))*4)
	q.ownerBytes = 0
	q.values = nil
	q.indices = nil
	q.closed = true
	return nil
}
func (q *satVarQueue) Clone(b *arena.Budget) (*satVarQueue, error) {
	if q != nil && q.runtimeBacked() {
		return q.cloneRuntime(b)
	}
	if e := q.open(); e != nil {
		return nil, e
	}
	v, e := newSATVarQueue(q.activity, b)
	if e != nil {
		return nil, e
	}
	v.runtimeActivity = q.runtimeActivity
	v.literalSource = q.literalSource
	n := uint64(cap(q.values)) * 4
	if e = b.Reserve(n); e != nil {
		v.Close()
		return nil, e
	}
	if n != 0 {
		v.values = make([]int32, len(q.values), cap(q.values))
		copy(v.values, q.values)
	}
	n = uint64(cap(q.indices)) * 4
	if e = b.Reserve(n); e != nil {
		v.Close()
		return nil, e
	}
	if n != 0 {
		v.indices = make([]int32, len(q.indices), cap(q.indices))
		copy(v.indices, q.indices)
	}
	return v, nil
}
func (q *satVarQueue) MoveClone(b *arena.Budget) (*satVarQueue, error) {
	if q != nil && q.runtimeBacked() {
		return q.moveRuntime(b)
	}
	if e := q.change(); e != nil {
		return nil, e
	}
	v, e := newSATVarQueue(q.activity, b)
	if e != nil {
		return nil, e
	}
	v.runtimeActivity = q.runtimeActivity
	v.literalSource = q.literalSource
	n := uint64(cap(q.values)+cap(q.indices)) * 4
	if b != q.budget {
		if e = b.Reserve(n); e != nil {
			v.Close()
			return nil, e
		}
		q.budget.Release(n)
	}
	v.values = q.values
	v.indices = q.indices
	q.values = nil
	q.indices = nil
	if q.epoch != ^uint64(0) {
		q.epoch++
	}
	return v, nil
}

func (q *satVarQueue) getActivity(i uint32) (uint32, error) {
	if q.runtimeActivity != nil {
		return q.runtimeActivity.Get(i)
	}
	return q.activity.Get(i)
}
func newSATSolverRuntimeVarQueue(a *satSolverRuntimeActivity, b *arena.Budget) (*satVarQueue, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	if b == nil || a.budget != b || a.values.Raw.Runtime.Budget() != b {
		return nil, ErrSATOwner
	}
	q, e := newSATVarQueue(nil, b)
	if e != nil {
		return nil, e
	}
	q.runtimeActivity = a
	q.bindRuntime(a.values.Raw.Runtime)
	if e = q.initialize(); e != nil {
		if cleanup := q.Close(); cleanup != nil {
			return q, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return q, nil
}
