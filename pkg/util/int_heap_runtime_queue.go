// Source: src/util/heap.h; src/util/vector.h. Licensed under the MIT license.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrSignedHeapIndexDomain = errors.New("sat: undefined source signed heap child shift")

// intHeapRuntimeQueue holds the runtime-backed storage of a utility heap.
// SAT-specific activities and literal weights belong to the SAT queue adapter.
type intHeapRuntimeQueue struct {
	generalSource                                             *IntHeap
	runtimeValues, runtimeIndices                             RuntimeUintColumn
	runtimeSourceStop                                         error
	runtimeSourceCursor, runtimeAbortCursor, runtimeCompleted uint8
	runtimeClosing                                            bool
	self                                                      *intHeapRuntimeQueue
	budget                                                    *arena.Budget
	ownerBytes                                                uint64
	closed                                                    bool
}

func (q *intHeapRuntimeQueue) open() error {
	if q == nil || q.self != q {
		return ErrRuntimeVectorOwner
	}
	if err := q.generalSource.alive(); err != nil {
		return err
	}
	if q.closed || q.runtimeClosing {
		return arena.ErrClosed
	}
	return nil
}
func (q *intHeapRuntimeQueue) change() error       { return q.open() }
func (q *intHeapRuntimeQueue) Close() error        { return q.closeRuntime() }
func (q *intHeapRuntimeQueue) runtimeBacked() bool { return q.runtimeValues.Raw.Self != nil }
func (q *intHeapRuntimeQueue) bindRuntime(r *Runtime) {
	q.runtimeValues.Bind(r)
	q.runtimeIndices.Bind(r)
	q.runtimeCompleted = 3
}
func (q *intHeapRuntimeQueue) sizeChecked() (uint32, error) {
	if e := q.open(); e != nil {
		return 0, e
	}
	if !q.runtimeBacked() {
		return 0, ErrRuntimeVectorOwner
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return 0, e
	}
	if n == 0 {
		return 0, nil
	}
	return n - 1, nil
}
func (q *intHeapRuntimeQueue) emptyChecked() (bool, error) {
	if e := q.open(); e != nil {
		return false, e
	}
	if !q.runtimeBacked() {
		return false, ErrRuntimeVectorOwner
	}
	n, e := q.runtimeValues.Size()
	return n == 1, e
}
func (q *intHeapRuntimeQueue) containsChecked(v int32) (bool, error) {
	if e := q.open(); e != nil {
		return false, e
	}
	if !q.runtimeBacked() {
		return false, ErrRuntimeVectorOwner
	}
	n, e := q.runtimeIndices.Size()
	if e != nil {
		return false, e
	}
	if int32(v) >= int32(n) {
		return false, nil
	}
	x, e := q.readUintRuntime(&q.runtimeIndices, uint32(v))
	return x != 0, e
}
func (q *intHeapRuntimeQueue) valueRuntime(i int32) (int32, error) {
	x, e := q.readUintRuntime(&q.runtimeValues, uint32(i))
	return int32(x), e
}
func (q *intHeapRuntimeQueue) indexRuntime(v int32) (int32, error) {
	x, e := q.readUintRuntime(&q.runtimeIndices, uint32(v))
	return int32(x), e
}
func (q *intHeapRuntimeQueue) putRuntime(i, v int32) error {
	if e := q.writeUintRuntime(&q.runtimeValues, uint32(i), uint32(v)); e != nil {
		return e
	}
	return q.writeUintRuntime(&q.runtimeIndices, uint32(v), uint32(i))
}
func (q *intHeapRuntimeQueue) lessRuntime(a, b int32) (bool, error) {
	return q.generalSource.lt(a, b), nil
}
func (q *intHeapRuntimeQueue) upRuntime(i int32) error {
	v, e := q.valueMutableRuntime(i)
	if e != nil {
		return e
	}
	for {
		parent := i >> 1
		if parent == 0 {
			break
		}
		p, e := q.valueMutableRuntime(parent)
		if e != nil {
			return e
		}
		less, e := q.lessRuntime(v, p)
		if e != nil {
			return e
		}
		if !less {
			break
		}
		p, e = q.valueMutableRuntime(parent)
		if e != nil {
			return e
		}
		if e = q.writeUintRuntime(&q.runtimeValues, uint32(i), uint32(p)); e != nil {
			return e
		}
		// heap.h:87 reaches operator[] again after the preceding value store.
		// Preserve that separate assertion/read before indexing the inverse.
		moved, e := q.valueMutableRuntime(i)
		if e != nil {
			return e
		}
		if e = q.writeUintRuntime(&q.runtimeIndices, uint32(moved), uint32(i)); e != nil {
			return e
		}
		i = parent
	}
	if e := q.putRuntime(i, v); e != nil {
		return e
	}
	return q.cassertRuntimeAt(92)
}
func intHeapLeft(i int32) (int32, error) {
	if i < 0 || i > int32(^uint32(0)>>2) {
		return 0, ErrSignedHeapIndexDomain
	}
	return i << 1, nil
}
func (q *intHeapRuntimeQueue) downRuntime(i int32) error {
	v, e := q.valueMutableRuntime(i)
	if e != nil {
		return e
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return e
	}
	sz := int32(n)
	for {
		left, e := intHeapLeft(i)
		if e != nil {
			return e
		}
		if left >= sz {
			break
		}
		right := left + 1
		c := left
		if right < sz {
			a, e := q.valueMutableRuntime(right)
			if e != nil {
				return e
			}
			b, e := q.valueMutableRuntime(left)
			if e != nil {
				return e
			}
			less, e := q.lessRuntime(a, b)
			if e != nil {
				return e
			}
			if less {
				c = right
			}
		}
		if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
			return gateErr
		} else if enabled && c>>1 != i {
			if e := q.assertionFailureRuntime("src/util/heap.h", 105, "parent(min_idx) == idx"); e != nil {
				return e
			}
		}
		child, e := q.valueMutableRuntime(c)
		if e != nil {
			return e
		}
		less, e := q.lessRuntime(child, v)
		if e != nil {
			return e
		}
		if !less {
			break
		}
		if e = q.putRuntime(i, child); e != nil {
			return e
		}
		i = c
	}
	if e := q.putRuntime(i, v); e != nil {
		return e
	}
	return q.cassertRuntimeAt(116)
}
func (q *intHeapRuntimeQueue) insertRuntime(v int32) error {
	if e := q.change(); e != nil {
		return e
	}
	if e := q.cassertRuntimeAt(231); e != nil {
		return e
	}
	if q.generalSource != nil && sourceDebug && q.generalSource.debugContext != nil {
		on, e := q.generalSource.debugContext.ConditionalAssertionEnabled("heap")
		if e != nil {
			return e
		}
		if on {
			yes, e := q.containsChecked(v)
			if e != nil {
				return e
			}
			if yes {
				if e = q.assertionFailureRuntime("src/util/heap.h", 232, "!contains(val)"); e != nil {
					return e
				}
			}
		}
	}

	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return gateErr
	} else if enabled {
		// SASSERT(is_valid_value(val)) evaluates its nested source SASSERT separately.
		if e := q.isValidValueRuntime(int32(v)); e != nil {
			return e
		}
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return e
	}
	i := int32(n)
	// Source publishes the reverse index BEFORE push can grow/refuse.
	if e = q.writeUintRuntime(&q.runtimeIndices, uint32(v), uint32(i)); e != nil {
		return e
	}
	if e = q.pushValuesRuntime(uint32(v)); e != nil {
		return e
	}
	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return gateErr
	} else if enabled {
		n, e = q.runtimeValues.Size()
		if e != nil {
			return e
		}
		if i != int32(n)-1 {
			if e = q.assertionFailureRuntime("src/util/heap.h", 237, "idx == static_cast<int>(m_values.size()) - 1"); e != nil {
				return e
			}
		}
	}
	if e := q.upRuntime(i); e != nil {
		return e
	}
	return q.cassertRuntimeAt(239)
}
func (q *intHeapRuntimeQueue) popRuntime() error {
	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return gateErr
	} else if enabled {
		n, e := q.runtimeValues.Size()
		if e != nil {
			return e
		}
		if n == 0 {
			if e := q.assertionFailureRuntime("src/util/vector.h", 344, "!empty()"); e != nil {
				return e
			}
		}
	}
	// vector.h:343–348 decrements the actual size word directly. shrink's
	// null no-op and logical-size assertion are not the pop_back body.
	c := &q.runtimeValues.Raw
	n, e := ReadRuntimeWord(c.Runtime, c.Block, 4)
	if e != nil {
		return e
	}
	return WriteRuntimeWord(c.Runtime, c.Block, 4, n-1)
}
func (q *intHeapRuntimeQueue) nextRuntime() (int32, error) {
	if e := q.change(); e != nil {
		return 0, e
	}
	if e := q.cassertRuntimeAt(174); e != nil {
		return 0, e
	}
	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return 0, gateErr
	} else if enabled {
		if q.generalSource == nil {
			empty, e := q.emptyChecked()
			if e != nil {
				return 0, e
			}
			if empty {
				return 0, ErrRuntimeVectorAssertion
			}
		}
		empty, e := q.emptyChecked()
		if e != nil {
			return 0, e
		}
		if empty {
			if e := q.assertionFailureRuntime("src/util/heap.h", 175, "!empty()"); e != nil {
				return 0, e
			}
		}
	}
	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return 0, gateErr
	} else if enabled {
		n, e := q.runtimeValues.Size()
		if e != nil {
			return 0, e
		}
		if n < 2 {
			if e := q.assertionFailureRuntime("src/util/heap.h", 176, "m_values.size() >= 2"); e != nil {
				return 0, e
			}
		}
	}
	result, e := q.valueMutableRuntime(1)
	if e != nil {
		return 0, e
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return 0, e
	}
	if n == 2 {
		if e = q.writeUintRuntime(&q.runtimeIndices, uint32(result), 0); e != nil {
			return 0, e
		}
		if e = q.popRuntime(); e != nil {
			return 0, e
		}
		if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
			return 0, gateErr
		} else if enabled {
			empty, e := q.emptyChecked()
			if e != nil {
				return 0, e
			}
			if !empty {
				if e := q.assertionFailureRuntime("src/util/heap.h", 181, "empty()"); e != nil {
					return 0, e
				}
			}
		}
	} else {
		last, e := q.backValuesRuntime()
		if e != nil {
			return 0, e
		}
		if e = q.putRuntime(1, last); e != nil {
			return 0, e
		}
		if e = q.writeUintRuntime(&q.runtimeIndices, uint32(result), 0); e != nil {
			return 0, e
		}
		if e = q.popRuntime(); e != nil {
			return 0, e
		}
		if e = q.downRuntime(1); e != nil {
			return int32(result), e
		}
	}
	if e := q.cassertRuntimeAt(191); e != nil {
		return 0, e
	}
	return int32(result), nil
}
func (q *intHeapRuntimeQueue) eraseRuntime(v int32) error {
	if e := q.change(); e != nil {
		return e
	}
	if e := q.cassertRuntimeAt(196); e != nil {
		return e
	}
	if enabled, gateErr := q.assertionsRuntime(); gateErr != nil {
		return gateErr
	} else if enabled {
		present, e := q.containsChecked(v)
		if e != nil {
			return e
		}
		if !present {
			if e := q.assertionFailureRuntime("src/util/heap.h", 197, "contains(val)"); e != nil {
				return e
			}
		}
	}
	i, e := q.indexMutableRuntime(int32(v))
	if e != nil {
		return e
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return e
	}
	if i == int32(n)-1 {
		if e = q.writeUintRuntime(&q.runtimeIndices, uint32(v), 0); e != nil {
			return e
		}
		if e := q.popRuntime(); e != nil {
			return e
		}
		return q.cassertRuntimeAt(217)
	}
	last, e := q.backValuesRuntime()
	if e != nil {
		return e
	}
	if e = q.putRuntime(i, last); e != nil {
		return e
	}
	if e = q.writeUintRuntime(&q.runtimeIndices, uint32(v), 0); e != nil {
		return e
	}
	if e = q.popRuntime(); e != nil {
		return e
	}
	parent := i >> 1
	if parent != 0 {
		p, e := q.valueMutableRuntime(i >> 1)
		if e != nil {
			return e
		}
		less, e := q.lessRuntime(last, p)
		if e != nil {
			return e
		}
		if less {
			if e := q.upRuntime(i); e != nil {
				return e
			}
			return q.cassertRuntimeAt(217)
		}
	}
	if e := q.downRuntime(i); e != nil {
		return e
	}
	return q.cassertRuntimeAt(217)
}
func (q *intHeapRuntimeQueue) resetRuntime() error {
	if e := q.open(); e != nil {
		return e
	}
	if e := q.cassertRuntimeAt(138); e != nil {
		return e
	}
	empty, e := q.emptyChecked()
	if e != nil {
		return e
	}
	if empty {
		return nil
	}
	n, e := q.runtimeIndices.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if e = q.resetIndexRuntime(i); e != nil {
			return e
		}
	}
	if e = q.resetValuesRuntime(); e != nil {
		return e
	}
	if e := q.pushValuesRuntime(^uint32(0)); e != nil {
		return e
	}
	return q.cassertRuntimeAt(145)
}
func (q *intHeapRuntimeQueue) sourceFinalize() error {
	if q == nil || q.self != q {
		return ErrRuntimeVectorOwner
	}
	if q.runtimeSourceStop != nil {
		return q.runtimeSourceStop
	}
	return q.finishSourceRuntime()
}
func (q *intHeapRuntimeQueue) finishSourceRuntime() error {
	for q.runtimeSourceCursor < 2 {
		c := &q.runtimeIndices.Raw
		mask := uint8(2)
		if q.runtimeSourceCursor == 1 {
			c = &q.runtimeValues.Raw
			mask = 1
		}
		if q.runtimeCompleted&mask != 0 && c.Block != (Block{}) {
			if e := c.Runtime.Deallocate(&c.Block); e != nil {
				if q.runtimeSourceStop == nil {
					q.runtimeSourceStop = e
				}
				return e
			}
		}
		q.runtimeSourceCursor++
	}
	return nil
}
func (q *intHeapRuntimeQueue) closeRuntime() error {
	if q == nil || q.self != q {
		return ErrRuntimeVectorOwner
	}
	if q.closed {
		return nil
	}
	q.runtimeClosing = true
	if e := q.sourceFinalize(); e != nil {
		return e
	}
	return q.abortRuntimePending()
}
func (q *intHeapRuntimeQueue) abortRuntimePending() error {
	for q.runtimeAbortCursor < 2 {
		c := &q.runtimeIndices.Raw
		if q.runtimeAbortCursor == 1 {
			c = &q.runtimeValues.Raw
		}
		// An incomplete copy constructor's published raw prefix has no source
		// destructor; this reachable buffer belongs solely to owning abort.
		if c.Block != (Block{}) {
			if e := c.Release(&c.Block); e != nil {
				return e
			}
		}
		if c.Pending != (Block{}) {
			if e := c.Release(&c.Pending); e != nil {
				return e
			}
			c.PendingReplaces = false
		}
		c.Closed = true
		q.runtimeAbortCursor++
	}
	q.budget.Release(q.ownerBytes)
	q.ownerBytes = 0
	q.closed = true
	return nil
}
func (q *intHeapRuntimeQueue) CleanupStopped() error {
	if q == nil || q.self != q {
		return ErrRuntimeVectorOwner
	}
	if q.closed {
		return nil
	}
	if q.runtimeSourceStop == nil {
		return ErrRuntimeVectorOwner
	}
	c := &q.runtimeIndices.Raw
	if q.runtimeSourceCursor == 1 {
		c = &q.runtimeValues.Raw
	}
	// Only actual consumed-token progress or the recorded successful physical
	// realloc transfer is acknowledged. No arbitrary source guard is repaired.
	if c.Block != (Block{}) {
		if !c.PendingReplaces {
			return q.runtimeSourceStop
		}
		c.Block = Block{}
	}
	q.runtimeSourceCursor++
	if e := q.finishSourceRuntime(); e != nil {
		return e
	}
	return q.abortRuntimePending()
}

func (q *intHeapRuntimeQueue) valueMutableRuntime(i int32) (int32, error) {
	if q.generalSource == nil {
		return q.valueRuntime(i)
	}
	x, e := q.generalSource.readUintRuntimeAt(&q.runtimeValues, uint32(i), 304)
	return int32(x), e
}
func (q *intHeapRuntimeQueue) isValidValueRuntime(v int32) error {
	enabled, e := q.assertionsRuntime()
	if e != nil || !enabled {
		return e
	}
	valid := v >= 0
	if valid {
		n, e := q.runtimeIndices.Size()
		if e != nil {
			return e
		}
		valid = v < int32(n)
	}
	if !valid {
		return q.assertionFailureRuntime("src/util/heap.h", 57, "v >= 0 && v < static_cast<int>(m_value2indices.size())")
	}
	return nil
}

func (q *intHeapRuntimeQueue) indexMutableRuntime(v int32) (int32, error) {
	if q.generalSource == nil {
		return q.indexRuntime(v)
	}
	x, e := q.generalSource.readUintRuntimeAt(&q.runtimeIndices, uint32(v), 304)
	return int32(x), e
}
func (q *intHeapRuntimeQueue) backValuesRuntime() (int32, error) {
	enabled, e := q.assertionsRuntime()
	if e != nil {
		return 0, e
	}
	if enabled {
		n, e := q.runtimeValues.Size()
		if e != nil {
			return 0, e
		}
		if n == 0 {
			if e := q.assertionFailureRuntime("src/util/vector.h", 334, "!empty()"); e != nil {
				return 0, e
			}
		}
	}
	n, e := q.runtimeValues.Size()
	if e != nil {
		return 0, e
	}
	return q.valueMutableRuntime(int32(n) - 1)
}
