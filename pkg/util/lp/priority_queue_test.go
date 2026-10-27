package lp

import (
	"math"
	"math/rand/v2"
	"testing"
)

// Check both directions independently of IsConsistent, whose source assertions
// intentionally do not check every form of a malformed active heap.
func lpCheckPriorityHeap(t *testing.T, h *priorityHeap, objects map[uint32]bool) {
	t.Helper()
	if int(h.size) != len(objects) {
		t.Fatal(h.size, len(objects))
	}
	seen := make(map[uint32]bool)
	for pos := uint32(1); pos <= h.size; pos++ {
		if uint64(pos) >= uint64(len(h.heap)) {
			t.Fatal("missing heap slot", pos)
		}
		object := h.heap[pos]
		if !objects[object] || seen[object] || uint64(object) >= uint64(len(h.inverse)) || h.inverse[object] != int32(pos) {
			t.Fatal("heap/inverse bijection", pos, object, h.inverse)
		}
		seen[object] = true
	}
	for object, pos := range h.inverse {
		if !objects[uint32(object)] && pos != -1 {
			t.Fatal("stale membership", object, pos)
		}
	}
}

// Original src/test/lp.cpp -tbq scenario, including its debug-only final drain.
// Assertions execute in every Go profile, strengthening the source driver.
func TestLPPriorityQueueSource(t *testing.T) {
	q := NewIntPriorityQueue(10)
	for _, p := range [][2]int32{{2, 2}, {1, 1}, {9, 9}, {8, 8}, {5, 25}, {3, 3}, {4, 4}, {7, 30}, {6, 6}, {0, 0}, {5, 5}, {7, 7}} {
		if err := q.Enqueue(uint32(p[0]), p[1]); err != nil {
			t.Fatal(err)
		}
	}
	for want := uint32(0); want < 10; want++ {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatal(got, want, err)
		}
	}
	for _, p := range [][2]int32{{2, 2}, {1, 1}, {9, 9}, {8, 8}, {5, 5}, {3, 3}, {4, 4}, {7, 2}, {0, 1}, {6, 6}, {7, 7}, {33, 1000}, {20, 0}} {
		if err := q.Enqueue(uint32(p[0]), p[1]); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := q.Dequeue(); err != nil || got != 20 {
		t.Fatal(got, err)
	}
	if err := q.Remove(33); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	for want := uint32(0); want < 10; want++ {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatal(got, want, err)
		}
	}
	if !q.IsEmpty() {
		t.Fatal(q.Size())
	}
}

func TestLPIntPriorityQueueIndependentMapOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	q := NewIntPriorityQueue(1)
	oracle := make(map[uint32]int32)
	random := rand.New(rand.NewPCG(1987, 413))
	for step := 0; step < 12000; step++ {
		object := uint32(random.IntN(128))
		switch random.IntN(8) {
		case 0, 1, 2, 3:
			value := int32(random.IntN(91))
			value -= 45
			if step%17 == 0 {
				value = math.MinInt32
			}
			if err := q.Enqueue(object, value); err != nil {
				t.Fatal(step, err)
			}
			oracle[object] = value
		case 4:
			if err := q.Remove(object); err != nil {
				t.Fatal(step, err)
			}
			delete(oracle, object)
		case 5:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				if err != nil {
					t.Fatal(step, err)
				}
				value, ok := oracle[got]
				if !ok {
					t.Fatal("unknown object", got)
				}
				for _, other := range oracle {
					if other < value {
						t.Fatal("not minimum", step, got, value, other)
					}
				}
				delete(oracle, got)
				// Dequeue leaves the last priority value stored outside the queue.
				stored, err := q.Priority(got)
				if err != nil || stored != value {
					t.Fatal(stored, value, err)
				}
			}
		case 6:
			if step%19 == 0 {
				if err := q.Clear(); err != nil {
					t.Fatal(err)
				}
				clear(oracle)
			}
		case 7:
			q.Resize(uint32(len(q.priorities) + random.IntN(3)))
		}
		members := make(map[uint32]bool, len(oracle))
		for object, value := range oracle {
			members[object] = true
			stored, err := q.Priority(object)
			if err != nil || stored != value {
				t.Fatal(step, stored, value, err)
			}
		}
		lpCheckPriorityHeap(t, &q.priorityHeap, members)
		good, err := q.IsConsistent()
		if err != nil || !good {
			t.Fatal(step, good, err)
		}
	}
}
func TestLPIntPriorityQueueTieAndResizePrefixes(t *testing.T) {
	q := NewIntPriorityQueue(4)
	for object := uint32(0); object < 3; object++ {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []uint32{0, 2, 1} {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatal(got, want, err)
		}
	}
	// Zero capacity followed by object zero retains the source's zero doubling.
	empty := NewIntPriorityQueue(0)
	if err := empty.Enqueue(0, 1); err != ErrBounds || empty.Size() != 0 {
		t.Fatal(err, empty.Size())
	}
	if _, err := empty.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	if _, err := empty.Priority(0); err != ErrBounds {
		t.Fatal(err)
	}
	// The auto-growth shift wraps before its attempted access, rather than
	// becoming an invented overflow-safe capacity policy.
	if err := q.Enqueue(1<<31, 1); err != ErrBounds || len(q.priorities) != 0 || len(q.heap) != 1 {
		t.Fatal(err, len(q.priorities), len(q.heap))
	}
	q = NewIntPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Clear(); err != ErrBounds {
		t.Fatal(err)
	}
	if q.Size() != 2 || q.inverse[0] != -1 {
		t.Fatal("clear prefix", q.Size(), q.inverse)
	}
	// The source inverse pass misses active objects outside its retained extent.
	good, err := q.IsConsistent()
	if err != ErrBounds {
		t.Fatal(good, err)
	}
	// Removing the top publishes a replacement before its inverse-index guard.
	// The stopped call leaves a real mismatch at the old root's inverse entry.
	q = NewIntPriorityQueue(5)
	for _, object := range []uint32{0, 1, 4} {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(4)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 || q.heap[1] != 4 {
		t.Fatal(err, q.Size(), q.heap)
	}
	good, err = q.IsConsistent()
	if err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(good, err)
	}
	q = NewIntPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(1, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(1)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 {
		t.Fatal(err, q.Size())
	}
	// With existing heap storage the empty native call executes the source's
	// unsigned decrement. Debug asserts before any mutation.
	q = NewIntPriorityQueue(1)
	_, err = q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || q.Size() != 0 {
			t.Fatal(err, q.Size())
		}
	} else if err != nil || q.Size() != math.MaxUint32 {
		t.Fatal(err, q.Size())
	}
}

func TestLPUintPriorityQueueIndependentMapOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	q := NewUintPriorityQueue(1)
	oracle := make(map[uint32]uint32)
	random := rand.New(rand.NewPCG(1987, 413))
	for step := 0; step < 12000; step++ {
		object := uint32(random.IntN(128))
		switch random.IntN(8) {
		case 0, 1, 2, 3:
			value := uint32(random.IntN(91))

			if step%17 == 0 {
				value = math.MaxUint32
			}
			if err := q.Enqueue(object, value); err != nil {
				t.Fatal(step, err)
			}
			oracle[object] = value
		case 4:
			if err := q.Remove(object); err != nil {
				t.Fatal(step, err)
			}
			delete(oracle, object)
		case 5:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				if err != nil {
					t.Fatal(step, err)
				}
				value, ok := oracle[got]
				if !ok {
					t.Fatal("unknown object", got)
				}
				for _, other := range oracle {
					if other < value {
						t.Fatal("not minimum", step, got, value, other)
					}
				}
				delete(oracle, got)
				// Dequeue leaves the last priority value stored outside the queue.
				stored, err := q.Priority(got)
				if err != nil || stored != value {
					t.Fatal(stored, value, err)
				}
			}
		case 6:
			if step%19 == 0 {
				if err := q.Clear(); err != nil {
					t.Fatal(err)
				}
				clear(oracle)
			}
		case 7:
			q.Resize(uint32(len(q.priorities) + random.IntN(3)))
		}
		members := make(map[uint32]bool, len(oracle))
		for object, value := range oracle {
			members[object] = true
			stored, err := q.Priority(object)
			if err != nil || stored != value {
				t.Fatal(step, stored, value, err)
			}
		}
		lpCheckPriorityHeap(t, &q.priorityHeap, members)
		good, err := q.IsConsistent()
		if err != nil || !good {
			t.Fatal(step, good, err)
		}
	}
}
func TestLPUintPriorityQueueTieAndResizePrefixes(t *testing.T) {
	q := NewUintPriorityQueue(4)
	for object := uint32(0); object < 3; object++ {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []uint32{0, 2, 1} {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatal(got, want, err)
		}
	}
	// Zero capacity followed by object zero retains the source's zero doubling.
	empty := NewUintPriorityQueue(0)
	if err := empty.Enqueue(0, 1); err != ErrBounds || empty.Size() != 0 {
		t.Fatal(err, empty.Size())
	}
	if _, err := empty.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	if _, err := empty.Priority(0); err != ErrBounds {
		t.Fatal(err)
	}
	// The auto-growth shift wraps before its attempted access, rather than
	// becoming an invented overflow-safe capacity policy.
	if err := q.Enqueue(1<<31, 1); err != ErrBounds || len(q.priorities) != 0 || len(q.heap) != 1 {
		t.Fatal(err, len(q.priorities), len(q.heap))
	}
	q = NewUintPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Clear(); err != ErrBounds {
		t.Fatal(err)
	}
	if q.Size() != 2 || q.inverse[0] != -1 {
		t.Fatal("clear prefix", q.Size(), q.inverse)
	}
	// The source inverse pass misses active objects outside its retained extent.
	good, err := q.IsConsistent()
	if err != ErrBounds {
		t.Fatal(good, err)
	}
	// Removing the top publishes a replacement before its inverse-index guard.
	// The stopped call leaves a real mismatch at the old root's inverse entry.
	q = NewUintPriorityQueue(5)
	for _, object := range []uint32{0, 1, 4} {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(4)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 || q.heap[1] != 4 {
		t.Fatal(err, q.Size(), q.heap)
	}
	good, err = q.IsConsistent()
	if err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(good, err)
	}
	q = NewUintPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(1, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(1)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 {
		t.Fatal(err, q.Size())
	}
	// With existing heap storage the empty native call executes the source's
	// unsigned decrement. Debug asserts before any mutation.
	q = NewUintPriorityQueue(1)
	_, err = q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || q.Size() != 0 {
			t.Fatal(err, q.Size())
		}
	} else if err != nil || q.Size() != math.MaxUint32 {
		t.Fatal(err, q.Size())
	}
}

func TestLPFloatPriorityQueueIndependentMapOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	q := NewFloatPriorityQueue(1)
	oracle := make(map[uint32]float64)
	random := rand.New(rand.NewPCG(1987, 413))
	for step := 0; step < 12000; step++ {
		object := uint32(random.IntN(128))
		switch random.IntN(8) {
		case 0, 1, 2, 3:
			value := float64(random.IntN(91))
			value -= 45
			if step%17 == 0 {
				value = math.Inf(-1)
			}
			if err := q.Enqueue(object, value); err != nil {
				t.Fatal(step, err)
			}
			oracle[object] = value
		case 4:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				if err != nil {
					t.Fatal(err)
				}
				value, ok := oracle[got]
				if !ok {
					t.Fatal(got)
				}
				for _, other := range oracle {
					if other < value {
						t.Fatal("not minimum", got, value, other)
					}
				}
				delete(oracle, got)
			}
		case 5:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				if err != nil {
					t.Fatal(step, err)
				}
				value, ok := oracle[got]
				if !ok {
					t.Fatal("unknown object", got)
				}
				for _, other := range oracle {
					if other < value {
						t.Fatal("not minimum", step, got, value, other)
					}
				}
				delete(oracle, got)
				// Dequeue leaves the last priority value stored outside the queue.
				stored, err := q.Priority(got)
				if err != nil || stored != value {
					t.Fatal(stored, value, err)
				}
			}
		case 6:
			if step%19 == 0 {
				if err := q.Clear(); err != nil {
					t.Fatal(err)
				}
				clear(oracle)
			}
		case 7:
			q.Resize(uint32(len(q.priorities) + random.IntN(3)))
		}
		members := make(map[uint32]bool, len(oracle))
		for object, value := range oracle {
			members[object] = true
			stored, err := q.Priority(object)
			if err != nil || stored != value {
				t.Fatal(step, stored, value, err)
			}
		}
		lpCheckPriorityHeap(t, &q.priorityHeap, members)
		good, err := q.IsConsistent()
		if err != nil || !good {
			t.Fatal(step, good, err)
		}
	}
}
func TestLPFloatPriorityQueueTieAndResizePrefixes(t *testing.T) {
	q := NewFloatPriorityQueue(4)
	for object := uint32(0); object < 3; object++ {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []uint32{0, 2, 1} {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatal(got, want, err)
		}
	}
	// Zero capacity followed by object zero retains the source's zero doubling.
	empty := NewFloatPriorityQueue(0)
	if err := empty.Enqueue(0, 1); err != ErrBounds || empty.Size() != 0 {
		t.Fatal(err, empty.Size())
	}
	if _, err := empty.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	if _, err := empty.Priority(0); err != ErrBounds {
		t.Fatal(err)
	}
	// The auto-growth shift wraps before its attempted access, rather than
	// becoming an invented overflow-safe capacity policy.
	if err := q.Enqueue(1<<31, 1); err != ErrBounds || len(q.priorities) != 0 || len(q.heap) != 1 {
		t.Fatal(err, len(q.priorities), len(q.heap))
	}
	q = NewFloatPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Clear(); err != ErrBounds {
		t.Fatal(err)
	}
	if q.Size() != 2 || q.inverse[0] != -1 {
		t.Fatal("clear prefix", q.Size(), q.inverse)
	}
	// The source inverse pass misses active objects outside its retained extent.
	good, err := q.IsConsistent()
	if err != ErrBounds {
		t.Fatal(good, err)
	}
	// Removing the top publishes a replacement before its inverse-index guard.
	// The stopped call leaves a real mismatch at the old root's inverse entry.
	q = NewFloatPriorityQueue(5)
	for _, object := range []uint32{0, 1, 4} {
		if err := q.Enqueue(object, 1); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(4)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 || q.heap[1] != 4 {
		t.Fatal(err, q.Size(), q.heap)
	}
	good, err = q.IsConsistent()
	if err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(good, err)
	}
	q = NewFloatPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(1, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(1)
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 {
		t.Fatal(err, q.Size())
	}
	// With existing heap storage the empty native call executes the source's
	// unsigned decrement. Debug asserts before any mutation.
	q = NewFloatPriorityQueue(1)
	_, err = q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || q.Size() != 0 {
			t.Fatal(err, q.Size())
		}
	} else if err != nil || q.Size() != math.MaxUint32 {
		t.Fatal(err, q.Size())
	}
}

func TestLPFloatPriorityQueueIEEESourceComparisons(t *testing.T) {
	q := NewFloatPriorityQueue(4)
	for object, value := range []float64{math.NaN(), 0, math.Inf(1), math.Inf(-1)} {
		if err := q.Enqueue(uint32(object), value); err != nil {
			t.Fatal(err)
		}
	}
	// NaN strict comparisons cannot displace the root; source <= diagnostic fails.
	if q.heap[1] != 0 {
		t.Fatal(q.heap)
	}
	if good, err := q.IsConsistent(); err != nil || good {
		t.Fatal(good, err)
	}
	if err := q.Enqueue(0, 1); err != nil {
		t.Fatal(err)
	}
	if got, err := q.Dequeue(); err != nil || got != 3 {
		t.Fatal(got, err)
	}
	if err := q.Clear(); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(0, math.Copysign(0, -1)); err != nil {
		t.Fatal(err)
	}
	value, err := q.Priority(0)
	if err != nil || !math.Signbit(value) {
		t.Fatal(value, err)
	}
	if err := q.Enqueue(0, math.NaN()); err != nil {
		t.Fatal(err)
	}
	value, err = q.Priority(0)
	if err != nil || !math.IsNaN(value) {
		t.Fatal(value, err)
	}
}

func TestLPIntPriorityQueueRetainedShrinkMutationPrefixes(t *testing.T) {
	// New membership increments size and assigns priority before heap publication.
	q := NewIntPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Enqueue(1, 2); err != ErrBounds || q.Size() != 3 || q.priorities[1] != 2 || q.inverse[1] != -1 {
		t.Fatal(err, q.Size(), q.priorities, q.inverse)
	}
	q.Resize(0)
	if err := q.Clear(); err != ErrBounds || q.Size() != 3 {
		t.Fatal(err, q.Size())
	}
	// A decreased retained object's old position can itself be outside the
	// resized heap. Assignment precedes that actual child lookup.
	q = NewIntPriorityQueue(4)
	for _, entry := range [][2]uint32{{3, 0}, {0, 2}, {1, 3}} {
		if err := q.Enqueue(entry[0], int32(entry[1])); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(2)
	if err := q.Enqueue(1, 1); err != ErrBounds || q.priorities[1] != 1 {
		t.Fatal(err, q.priorities)
	}
	// Increasing at the same invalid position executes no access when it has no
	// children. This source continuation is retained rather than repaired.
	if err := q.Enqueue(1, 5); err != nil || q.priorities[1] != 5 {
		t.Fatal(err, q.priorities)
	}
	// A parent object can have fallen outside the priority extent even though
	// the child's inverse position remains in range.
	if err := q.Enqueue(0, 1); err != ErrBounds || q.priorities[0] != 1 {
		t.Fatal(err, q.priorities)
	}
	for _, objects := range [][]uint32{{0, 3, 1}, {0, 1, 3}} {
		q = NewIntPriorityQueue(4)
		for priority, object := range objects {
			if err := q.Enqueue(object, int32(priority)); err != nil {
				t.Fatal(err)
			}
		}
		q.Resize(2)
		if err := q.Enqueue(0, 9); err != ErrBounds || q.priorities[0] != 9 {
			t.Fatal(err, q.priorities)
		}
	}
	q = NewIntPriorityQueue(4)
	if err := q.Remove(4); err != ErrBounds {
		t.Fatal(err)
	}
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Remove(0); err != ErrBounds || q.Size() != 1 || q.heap[1] != 3 || q.inverse[0] != 1 {
		t.Fatal(err, q.Size(), q.heap, q.inverse)
	}
}

func TestLPUintPriorityQueueRetainedShrinkMutationPrefixes(t *testing.T) {
	// New membership increments size and assigns priority before heap publication.
	q := NewUintPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Enqueue(1, 2); err != ErrBounds || q.Size() != 3 || q.priorities[1] != 2 || q.inverse[1] != -1 {
		t.Fatal(err, q.Size(), q.priorities, q.inverse)
	}
	q.Resize(0)
	if err := q.Clear(); err != ErrBounds || q.Size() != 3 {
		t.Fatal(err, q.Size())
	}
	// A decreased retained object's old position can itself be outside the
	// resized heap. Assignment precedes that actual child lookup.
	q = NewUintPriorityQueue(4)
	for _, entry := range [][2]uint32{{3, 0}, {0, 2}, {1, 3}} {
		if err := q.Enqueue(entry[0], uint32(entry[1])); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(2)
	if err := q.Enqueue(1, 1); err != ErrBounds || q.priorities[1] != 1 {
		t.Fatal(err, q.priorities)
	}
	// Increasing at the same invalid position executes no access when it has no
	// children. This source continuation is retained rather than repaired.
	if err := q.Enqueue(1, 5); err != nil || q.priorities[1] != 5 {
		t.Fatal(err, q.priorities)
	}
	// A parent object can have fallen outside the priority extent even though
	// the child's inverse position remains in range.
	if err := q.Enqueue(0, 1); err != ErrBounds || q.priorities[0] != 1 {
		t.Fatal(err, q.priorities)
	}
	for _, objects := range [][]uint32{{0, 3, 1}, {0, 1, 3}} {
		q = NewUintPriorityQueue(4)
		for priority, object := range objects {
			if err := q.Enqueue(object, uint32(priority)); err != nil {
				t.Fatal(err)
			}
		}
		q.Resize(2)
		if err := q.Enqueue(0, 9); err != ErrBounds || q.priorities[0] != 9 {
			t.Fatal(err, q.priorities)
		}
	}
	q = NewUintPriorityQueue(4)
	if err := q.Remove(4); err != ErrBounds {
		t.Fatal(err)
	}
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Remove(0); err != ErrBounds || q.Size() != 1 || q.heap[1] != 3 || q.inverse[0] != 1 {
		t.Fatal(err, q.Size(), q.heap, q.inverse)
	}
}

func TestLPFloatPriorityQueueRetainedShrinkMutationPrefixes(t *testing.T) {
	// New membership increments size and assigns priority before heap publication.
	q := NewFloatPriorityQueue(4)
	if err := q.Enqueue(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 1); err != nil {
		t.Fatal(err)
	}
	q.Resize(2)
	if err := q.Enqueue(1, 2); err != ErrBounds || q.Size() != 3 || q.priorities[1] != 2 || q.inverse[1] != -1 {
		t.Fatal(err, q.Size(), q.priorities, q.inverse)
	}
	q.Resize(0)
	if err := q.Clear(); err != ErrBounds || q.Size() != 3 {
		t.Fatal(err, q.Size())
	}
	// A decreased retained object's old position can itself be outside the
	// resized heap. Assignment precedes that actual child lookup.
	q = NewFloatPriorityQueue(4)
	for _, entry := range [][2]uint32{{3, 0}, {0, 2}, {1, 3}} {
		if err := q.Enqueue(entry[0], float64(entry[1])); err != nil {
			t.Fatal(err)
		}
	}
	q.Resize(2)
	if err := q.Enqueue(1, 1); err != ErrBounds || q.priorities[1] != 1 {
		t.Fatal(err, q.priorities)
	}
	// Increasing at the same invalid position executes no access when it has no
	// children. This source continuation is retained rather than repaired.
	if err := q.Enqueue(1, 5); err != nil || q.priorities[1] != 5 {
		t.Fatal(err, q.priorities)
	}
	// A parent object can have fallen outside the priority extent even though
	// the child's inverse position remains in range.
	if err := q.Enqueue(0, 1); err != ErrBounds || q.priorities[0] != 1 {
		t.Fatal(err, q.priorities)
	}
	for _, objects := range [][]uint32{{0, 3, 1}, {0, 1, 3}} {
		q = NewFloatPriorityQueue(4)
		for priority, object := range objects {
			if err := q.Enqueue(object, float64(priority)); err != nil {
				t.Fatal(err)
			}
		}
		q.Resize(2)
		if err := q.Enqueue(0, 9); err != ErrBounds || q.priorities[0] != 9 {
			t.Fatal(err, q.priorities)
		}
	}

}
