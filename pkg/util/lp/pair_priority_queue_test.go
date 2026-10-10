package lp

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

// Upstream test_upair_queue performs 100 enqueues, one removal and a drain but
// asserts no outcomes. This translation checks priorities and membership too.
func TestLPPairPriorityQueueSource(t *testing.T) {
	q := NewIntPairPriorityQueue(2)
	oracle := make(map[IndexPair]int32)
	random := rand.New(rand.NewPCG(12, 457))
	for k := 0; k < 100; k++ {
		pair := IndexPair{uint32(random.IntN(10)), uint32(random.IntN(10))}
		priority := int32(random.IntN(10))
		if err := q.Enqueue(pair.I, pair.J, priority); err != nil {
			t.Fatal(err)
		}
		oracle[pair] = priority
	}
	if err := q.Remove(5, 5); err != nil {
		t.Fatal(err)
	}
	delete(oracle, IndexPair{5, 5})
	for !q.IsEmpty() {
		pair, err := q.Dequeue()
		if err != nil {
			t.Fatal(err)
		}
		priority, ok := oracle[pair]
		if !ok {
			t.Fatal(pair)
		}
		for _, other := range oracle {
			if other < priority {
				t.Fatal(pair, priority, other)
			}
		}
		delete(oracle, pair)
	}
	if len(oracle) != 0 {
		t.Fatal(oracle)
	}
}

func TestLPIntPairPriorityQueueIndependentMapOracle(t *testing.T) {
	q := NewIntPairPriorityQueue(2)
	oracle := make(map[IndexPair]int32)
	random := rand.New(rand.NewPCG(1997, 651))
	for step := 0; step < 12000; step++ {
		pair := IndexPair{uint32(random.IntN(13)), uint32(random.IntN(13))}
		switch random.IntN(6) {
		case 0, 1, 2:
			priority := int32(random.IntN(123))
			priority -= 61
			if err := q.Enqueue(pair.I, pair.J, priority); err != nil {
				t.Fatal(step, err)
			}
			oracle[pair] = priority
		case 3:
			if err := q.Remove(pair.I, pair.J); err != nil {
				t.Fatal(step, err)
			}
			delete(oracle, pair)
		case 4:
			if len(oracle) > 0 {
				pair, err := q.Dequeue()
				if err != nil {
					t.Fatal(step, err)
				}
				priority, ok := oracle[pair]
				if !ok {
					t.Fatal("unknown pair", pair)
				}
				for _, other := range oracle {
					if other < priority {
						t.Fatal("not minimum", step, pair, priority, other)
					}
				}
				delete(oracle, pair)
			}
		case 5:
			if step%47 == 0 {
				q.Resize(uint32(len(q.queue.priorities) + random.IntN(7)))
			}
		}
		if int(q.Size()) != len(oracle) || len(q.index) != len(oracle) {
			t.Fatal(step, q.Size(), len(oracle), len(q.index))
		}
		objects := make(map[uint32]bool, len(oracle))
		for pair, priority := range oracle {
			if !q.Contains(pair.I, pair.J) {
				t.Fatal("missing pair", pair)
			}
			got, err := q.Priority(pair.I, pair.J)
			if err != nil || got != priority {
				t.Fatal(step, pair, got, priority, err)
			}
			slot, ok := q.index[pair]
			if !ok || q.pairs[slot] != pair {
				t.Fatal("pair/slot inverse", pair, slot)
			}
			objects[slot] = true
		}
		lpCheckPriorityHeap(t, &q.queue.priorityHeap, objects)
		if len(q.available)+len(oracle) != len(q.pairs) {
			t.Fatal("slot census", step)
		}
		for _, slot := range q.available {
			if objects[slot] {
				t.Fatal("occupied free slot", slot)
			}
		}
		good, err := q.IsCorrect()
		if err != nil || !good {
			t.Fatal(step, good, err)
		}
		if !q.Contains(pair.I, pair.J) {
			if got, err := q.Priority(pair.I, pair.J); err != nil || got != 0xFFFFFF {
				t.Fatal(got, err)
			}
		}
	}
}
func TestLPIntPairPriorityQueueSlotAndFailurePrefixes(t *testing.T) {
	q := NewIntPairPriorityQueue(2)
	if err := q.Enqueue(1, 2, 10); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{1, 2}] != 1 {
		t.Fatal(q.index)
	}
	if err := q.Enqueue(2, 1, 20); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{2, 1}] != 0 {
		t.Fatal(q.index)
	}
	if err := q.Remove(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 4, 30); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{3, 4}] != 1 {
		t.Fatal(q.index)
	}
	if err := q.Enqueue(5, 6, 40); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{5, 6}] != 3 || !reflect.DeepEqual(q.available, []uint32{2}) {
		t.Fatal(q.index, q.available)
	}
	// Public Resize changes only the underlying heap extent.
	beforePairs := append([]IndexPair(nil), q.pairs...)
	beforeAvailable := append([]uint32(nil), q.available...)
	q.Resize(10)
	if !reflect.DeepEqual(beforePairs, q.pairs) || !reflect.DeepEqual(beforeAvailable, q.available) {
		t.Fatal(q.pairs, q.available)
	}
	// Underlying enqueue can fail after the map publishes the selected slot.
	q = NewIntPairPriorityQueue(1)
	q.Resize(0)
	if err := q.Enqueue(7, 8, 9); err != ErrBounds {
		t.Fatal(err)
	}
	if q.Size() != 0 || !q.Contains(7, 8) || len(q.available) != 0 || q.pairs[0] != (IndexPair{7, 8}) {
		t.Fatal(q.Size(), q.index, q.available, q.pairs)
	}
	// Source diagnostic does not test map membership against heap membership.
	if good, err := q.IsCorrect(); err != nil || !good {
		t.Fatal(good, err)
	}
	if _, err := q.Priority(7, 8); err != ErrBounds {
		t.Fatal(err)
	}
	if err := q.Remove(7, 8); err != ErrBounds || !q.Contains(7, 8) {
		t.Fatal(err, q.index)
	}
	q.Resize(1)
	if err := q.Enqueue(7, 8, 9); err != nil {
		t.Fatal(err)
	}
	if got, err := q.Dequeue(); err != nil || got != (IndexPair{7, 8}) {
		t.Fatal(got, err)
	}
	q = NewIntPairPriorityQueue(0)
	if err := q.Enqueue(1, 2, 3); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	if len(q.index) != 0 || len(q.available) != 0 || q.Size() != 0 {
		t.Fatal(q.index, q.available, q.Size())
	}
	if _, err := q.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	// The available-slot diagnostic reports duplicates from the native empty
	// dequeue continuation, rather than silently repairing the free list.
	q = NewIntPairPriorityQueue(1)
	_, err := q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || !reflect.DeepEqual(q.available, []uint32{0}) {
			t.Fatal(err, q.available)
		}
	} else {
		if err != nil || !reflect.DeepEqual(q.available, []uint32{0, 0}) || q.availableCorrect() {
			t.Fatal(err, q.available)
		}
	}
}

func TestLPUintPairPriorityQueueIndependentMapOracle(t *testing.T) {
	q := NewUintPairPriorityQueue(2)
	oracle := make(map[IndexPair]uint32)
	random := rand.New(rand.NewPCG(1997, 651))
	for step := 0; step < 12000; step++ {
		pair := IndexPair{uint32(random.IntN(13)), uint32(random.IntN(13))}
		switch random.IntN(6) {
		case 0, 1, 2:
			priority := uint32(random.IntN(123))

			if err := q.Enqueue(pair.I, pair.J, priority); err != nil {
				t.Fatal(step, err)
			}
			oracle[pair] = priority
		case 3:
			if err := q.Remove(pair.I, pair.J); err != nil {
				t.Fatal(step, err)
			}
			delete(oracle, pair)
		case 4:
			if len(oracle) > 0 {
				pair, err := q.Dequeue()
				if err != nil {
					t.Fatal(step, err)
				}
				priority, ok := oracle[pair]
				if !ok {
					t.Fatal("unknown pair", pair)
				}
				for _, other := range oracle {
					if other < priority {
						t.Fatal("not minimum", step, pair, priority, other)
					}
				}
				delete(oracle, pair)
			}
		case 5:
			if step%47 == 0 {
				q.Resize(uint32(len(q.queue.priorities) + random.IntN(7)))
			}
		}
		if int(q.Size()) != len(oracle) || len(q.index) != len(oracle) {
			t.Fatal(step, q.Size(), len(oracle), len(q.index))
		}
		objects := make(map[uint32]bool, len(oracle))
		for pair, priority := range oracle {
			if !q.Contains(pair.I, pair.J) {
				t.Fatal("missing pair", pair)
			}
			got, err := q.Priority(pair.I, pair.J)
			if err != nil || got != priority {
				t.Fatal(step, pair, got, priority, err)
			}
			slot, ok := q.index[pair]
			if !ok || q.pairs[slot] != pair {
				t.Fatal("pair/slot inverse", pair, slot)
			}
			objects[slot] = true
		}
		lpCheckPriorityHeap(t, &q.queue.priorityHeap, objects)
		if len(q.available)+len(oracle) != len(q.pairs) {
			t.Fatal("slot census", step)
		}
		for _, slot := range q.available {
			if objects[slot] {
				t.Fatal("occupied free slot", slot)
			}
		}
		good, err := q.IsCorrect()
		if err != nil || !good {
			t.Fatal(step, good, err)
		}
		if !q.Contains(pair.I, pair.J) {
			if got, err := q.Priority(pair.I, pair.J); err != nil || got != 0xFFFFFF {
				t.Fatal(got, err)
			}
		}
	}
}
func TestLPUintPairPriorityQueueSlotAndFailurePrefixes(t *testing.T) {
	q := NewUintPairPriorityQueue(2)
	if err := q.Enqueue(1, 2, 10); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{1, 2}] != 1 {
		t.Fatal(q.index)
	}
	if err := q.Enqueue(2, 1, 20); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{2, 1}] != 0 {
		t.Fatal(q.index)
	}
	if err := q.Remove(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(3, 4, 30); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{3, 4}] != 1 {
		t.Fatal(q.index)
	}
	if err := q.Enqueue(5, 6, 40); err != nil {
		t.Fatal(err)
	}
	if q.index[IndexPair{5, 6}] != 3 || !reflect.DeepEqual(q.available, []uint32{2}) {
		t.Fatal(q.index, q.available)
	}
	// Public Resize changes only the underlying heap extent.
	beforePairs := append([]IndexPair(nil), q.pairs...)
	beforeAvailable := append([]uint32(nil), q.available...)
	q.Resize(10)
	if !reflect.DeepEqual(beforePairs, q.pairs) || !reflect.DeepEqual(beforeAvailable, q.available) {
		t.Fatal(q.pairs, q.available)
	}
	// Underlying enqueue can fail after the map publishes the selected slot.
	q = NewUintPairPriorityQueue(1)
	q.Resize(0)
	if err := q.Enqueue(7, 8, 9); err != ErrBounds {
		t.Fatal(err)
	}
	if q.Size() != 0 || !q.Contains(7, 8) || len(q.available) != 0 || q.pairs[0] != (IndexPair{7, 8}) {
		t.Fatal(q.Size(), q.index, q.available, q.pairs)
	}
	// Source diagnostic does not test map membership against heap membership.
	if good, err := q.IsCorrect(); err != nil || !good {
		t.Fatal(good, err)
	}
	if _, err := q.Priority(7, 8); err != ErrBounds {
		t.Fatal(err)
	}
	if err := q.Remove(7, 8); err != ErrBounds || !q.Contains(7, 8) {
		t.Fatal(err, q.index)
	}
	q.Resize(1)
	if err := q.Enqueue(7, 8, 9); err != nil {
		t.Fatal(err)
	}
	if got, err := q.Dequeue(); err != nil || got != (IndexPair{7, 8}) {
		t.Fatal(got, err)
	}
	q = NewUintPairPriorityQueue(0)
	if err := q.Enqueue(1, 2, 3); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	if len(q.index) != 0 || len(q.available) != 0 || q.Size() != 0 {
		t.Fatal(q.index, q.available, q.Size())
	}
	if _, err := q.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	// The available-slot diagnostic reports duplicates from the native empty
	// dequeue continuation, rather than silently repairing the free list.
	q = NewUintPairPriorityQueue(1)
	_, err := q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || !reflect.DeepEqual(q.available, []uint32{0}) {
			t.Fatal(err, q.available)
		}
	} else {
		if err != nil || !reflect.DeepEqual(q.available, []uint32{0, 0}) || q.availableCorrect() {
			t.Fatal(err, q.available)
		}
	}
}
