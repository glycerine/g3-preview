package lp

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpExactPriorityOracleLess(a, b [2]*big.Rat) bool {
	if c := a[0].Cmp(b[0]); c != 0 {
		return c < 0
	}
	return a[1].Cmp(b[1]) < 0
}

func TestLPRationalPairSourceRelations(t *testing.T) {
	h := newLPPairHarness(t)
	for _, row := range []struct {
		ax, ay, bx, by string
		less, greater  bool
	}{
		{"1/3", "7/11", "2/3", "-8/9", true, false},
		{"1/3", "7/11", "1/3", "8/11", true, false},
		{"1/3", "7/11", "1/3", "7/11", false, false},
		{"2/3", "-8/9", "1/3", "7/11", false, true},
		{"1/3", "8/11", "1/3", "7/11", false, true},
	} {
		a, b := h.pair(row.ax, row.ay), h.pair(row.bx, row.by)
		less, err := a.Less(h.m, b)
		h.must(err)
		greater, err := a.Greater(h.m, b)
		h.must(err)
		if less != row.less || greater != row.greater {
			t.Fatal(row, less, greater)
		}
	}
	// Definition-level raw scalar contract: numerically equal unreduced X does
	// not pass the source structural equality gate. This is not a claim that LP
	// value constructors produce unnormalized rationals.
	a, b := h.pair("1/2", "-3"), h.pair("1/2", "4")
	b.X.Num = numeral.NewInt(2)
	b.X.Den = numeral.NewInt(4)
	less, err := a.Less(h.m, b)
	h.must(err)
	greater, err := a.Greater(h.m, b)
	h.must(err)
	if less || greater {
		t.Fatal("unreduced equality gate", less, greater)
	}
}

func TestLPRationalPriorityQueueIndependentExactOracle(t *testing.T) {
	h := newLPPairHarness(t)
	q := NewRationalPriorityQueue(h.m, 1)
	values := make([]*RationalPair, 64)
	expected := make([][2]*big.Rat, 64)
	random := rand.New(rand.NewPCG(1977, 214))
	for i := range values {
		x := fmt.Sprintf("%d/%d", random.IntN(11)-5, random.IntN(19)+1)
		y := fmt.Sprintf("%d/%d", random.IntN(41)-20, random.IntN(31)+1)
		if i%17 == 0 {
			x = strings.Repeat("1234567890", 16) + "/7"
			y = "-" + strings.Repeat("9876543210", 17) + "/11"
		}
		values[i] = h.pair(x, y)
		expected[i][0], _ = new(big.Rat).SetString(x)
		expected[i][1], _ = new(big.Rat).SetString("0")
	}
	oracle := make(map[uint32][2]*big.Rat)
	for step := 0; step < 5000; step++ {
		object := uint32(random.IntN(64))
		switch random.IntN(7) {
		case 0, 1, 2, 3:
			k := random.IntN(len(values))
			h.must(q.Enqueue(object, &values[k].X))
			oracle[object] = expected[k]
		case 4:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				h.must(err)
				value, ok := oracle[got]
				if !ok {
					t.Fatal("unknown exact priority", got)
				}
				for _, other := range oracle {
					if lpExactPriorityOracleLess(other, value) {
						t.Fatal("not exact minimum", step, got, value, other)
					}
				}
				delete(oracle, got)
			}
		case 5:
			if step%23 == 0 {
				h.must(q.Clear())
				clear(oracle)
			}
		case 6:
			h.must(q.Resize(uint32(len(q.priorities.Data) + random.IntN(3))))
		}
		members := make(map[uint32]bool, len(oracle))
		for object, want := range oracle {
			members[object] = true
			lpCheckExactPermutationCoordinate(t, h, &q.priorities.Data[object], want[0])

		}
		lpCheckPriorityHeap(t, &q.priorityHeap, members)
		if q.decreaseArg != nil || q.calls.pending() {
			t.Fatal("completed queue retained call owner")
		}
	}
	h.must(q.Close())
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	if h.m.ComparisonInvocationState().Active {
		t.Fatal("completed comparison retained")
	}
}
func TestLPRationalPriorityQueueExactLifetimeAndPrefixContracts(t *testing.T) {
	h := newLPPairHarness(t)
	v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
	q := NewRationalPriorityQueue(h.m, 4)
	for object := uint32(0); object < 3; object++ {
		h.must(q.Enqueue(object, &v.X))
	}
	// Every stored priority owns independent numerals, including removed slots.
	for i := range q.priorities.Data {
		if i >= 3 {
			continue
		}
		if q.priorities.Data[i].Num.Cell == v.X.Num.Cell {
			t.Fatal("shallow priority copy")
		}
	}
	for _, want := range []uint32{0, 2, 1} {
		got, err := q.Dequeue()
		h.must(err)
		if got != want {
			t.Fatal(got, want)
		}
	}
	old, err := h.m.String(&q.priorities.Data[0])
	h.must(err)
	h.must(h.m.SetString(&v.X, "1"))
	retained, err := h.m.String(&q.priorities.Data[0])
	h.must(err)
	if retained != old {
		t.Fatal("borrowed external priority", retained, old)
	}
	h.must(q.Clear())
	h.must(q.Resize(2))
	h.must(q.Resize(6))
	zero, err := h.m.IsZero(&q.priorities.Data[5])
	h.must(err)
	if !zero {
		t.Fatal("uninitialized growth")
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	q = NewRationalPriorityQueue(h.m, 0)
	if err := q.Enqueue(0, &v.X); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err := q.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(q.Close())
	q = NewRationalPriorityQueue(h.m, 4)
	h.must(q.Enqueue(0, &v.X))
	h.must(q.Enqueue(3, &v.X))
	h.must(q.Resize(2))
	if err := q.Enqueue(1, &v.X); err != ErrBounds || q.Size() != 3 || q.inverse[1] != -1 {
		t.Fatal(err, q.Size(), q.inverse)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	q = NewRationalPriorityQueue(h.m, 1)
	_, err = q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || q.Size() != 0 {
			t.Fatal(err, q.Size())
		}
	} else if err != nil || q.Size() != ^uint32(0) {
		t.Fatal(err, q.Size())
	}
	h.must(q.Close())
}

func TestLPPairPriorityQueueIndependentExactOracle(t *testing.T) {
	h := newLPPairHarness(t)
	q := NewPairPriorityQueue(h.m, 1)
	values := make([]*RationalPair, 64)
	expected := make([][2]*big.Rat, 64)
	random := rand.New(rand.NewPCG(1977, 214))
	for i := range values {
		x := fmt.Sprintf("%d/%d", random.IntN(11)-5, random.IntN(19)+1)
		y := fmt.Sprintf("%d/%d", random.IntN(41)-20, random.IntN(31)+1)
		if i%17 == 0 {
			x = strings.Repeat("1234567890", 16) + "/7"
			y = "-" + strings.Repeat("9876543210", 17) + "/11"
		}
		values[i] = h.pair(x, y)
		expected[i][0], _ = new(big.Rat).SetString(x)
		expected[i][1], _ = new(big.Rat).SetString(y)
	}
	oracle := make(map[uint32][2]*big.Rat)
	for step := 0; step < 5000; step++ {
		object := uint32(random.IntN(64))
		switch random.IntN(7) {
		case 0, 1, 2, 3:
			k := random.IntN(len(values))
			h.must(q.Enqueue(object, values[k]))
			oracle[object] = expected[k]
		case 4:
			if len(oracle) > 0 {
				got, err := q.Dequeue()
				h.must(err)
				value, ok := oracle[got]
				if !ok {
					t.Fatal("unknown exact priority", got)
				}
				for _, other := range oracle {
					if lpExactPriorityOracleLess(other, value) {
						t.Fatal("not exact minimum", step, got, value, other)
					}
				}
				delete(oracle, got)
			}
		case 5:
			if step%23 == 0 {
				h.must(q.Clear())
				clear(oracle)
			}
		case 6:
			h.must(q.Resize(uint32(len(q.priorities.Data) + random.IntN(3))))
		}
		members := make(map[uint32]bool, len(oracle))
		for object, want := range oracle {
			members[object] = true
			lpCheckExactPermutationCoordinate(t, h, &q.priorities.Data[object].X, want[0])
			lpCheckExactPermutationCoordinate(t, h, &q.priorities.Data[object].Y, want[1])
		}
		lpCheckPriorityHeap(t, &q.priorityHeap, members)
		if q.decreaseArg != nil || q.calls.pending() {
			t.Fatal("completed queue retained call owner")
		}
	}
	h.must(q.Close())
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	if h.m.ComparisonInvocationState().Active {
		t.Fatal("completed comparison retained")
	}
}
func TestLPPairPriorityQueueExactLifetimeAndPrefixContracts(t *testing.T) {
	h := newLPPairHarness(t)
	v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
	q := NewPairPriorityQueue(h.m, 4)
	for object := uint32(0); object < 3; object++ {
		h.must(q.Enqueue(object, v))
	}
	// Every stored priority owns independent numerals, including removed slots.
	for i := range q.priorities.Data {
		if i >= 3 {
			continue
		}
		if q.priorities.Data[i].X.Num.Cell == v.X.Num.Cell || q.priorities.Data[i].Y.Num.Cell == v.Y.Num.Cell {
			t.Fatal("shallow priority copy")
		}
	}
	for _, want := range []uint32{0, 2, 1} {
		got, err := q.Dequeue()
		h.must(err)
		if got != want {
			t.Fatal(got, want)
		}
	}
	old, err := h.m.String(&q.priorities.Data[0].X)
	h.must(err)
	h.must(h.m.SetString(&v.X, "1"))
	retained, err := h.m.String(&q.priorities.Data[0].X)
	h.must(err)
	if retained != old {
		t.Fatal("borrowed external priority", retained, old)
	}
	h.must(q.Clear())
	h.must(q.Resize(2))
	h.must(q.Resize(6))
	zero, err := h.m.IsZero(&q.priorities.Data[5].X)
	h.must(err)
	if !zero {
		t.Fatal("uninitialized growth")
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	q = NewPairPriorityQueue(h.m, 0)
	if err := q.Enqueue(0, v); err != ErrBounds {
		t.Fatal(err)
	}
	if _, err := q.Dequeue(); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(q.Close())
	q = NewPairPriorityQueue(h.m, 4)
	h.must(q.Enqueue(0, v))
	h.must(q.Enqueue(3, v))
	h.must(q.Resize(2))
	if err := q.Enqueue(1, v); err != ErrBounds || q.Size() != 3 || q.inverse[1] != -1 {
		t.Fatal(err, q.Size(), q.inverse)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	q = NewPairPriorityQueue(h.m, 1)
	_, err = q.Dequeue()
	if sourceDebugTests {
		if err != ErrAssertion || q.Size() != 0 {
			t.Fatal(err, q.Size())
		}
	} else if err != nil || q.Size() != ^uint32(0) {
		t.Fatal(err, q.Size())
	}
	h.must(q.Close())
}

func TestLPRationalPriorityQueueAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"new", "decrease", "increase", "dequeue"} {
			failures, successes, lexicalStops := 0, 0, 0
			for fail := int64(0); fail < 100; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%v/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					digits := strings.Repeat("1234567890", 100)
					old := h.pair(digits+"/7", digits+"/11")
					next := h.pair("9"+digits+"/13", "8"+digits+"/17")
					low := h.pair(digits+"/"+strings.Repeat("9876543210", 200)+"7", digits+"/"+strings.Repeat("7654321090", 200)+"9")
					q := NewRationalPriorityQueue(h.m, 4)
					h.must(q.Enqueue(0, &old.X))
					if op == "decrease" || op == "increase" {
						h.must(q.Enqueue(1, &next.X))
					}
					if op == "dequeue" {
						a, b, c := h.pair(digits+"/11", "1"), h.pair(digits+"/13", "2"), h.pair(digits+"/17", "3")
						for i, v := range []*RationalPair{a, b, c} {
							h.must(q.Enqueue(uint32(i+1), &v.X))
						}
					}
					h.budget.FailAfter(fail)
					var err error
					switch op {
					case "new":
						err = q.Enqueue(1, &next.X)
					case "decrease":
						err = q.Enqueue(1, &low.X)
					case "increase":
						err = q.Enqueue(0, &next.X)
					case "dequeue":
						_, err = q.Dequeue()
					}
					h.budget.FailAfter(-1)
					if err == nil {
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal("unexpected reservation error", err)
						}
						failures++
						if q.decreaseArg != nil {
							lexicalStops++
						}
					}
					if q.ready() != nil {
						if err := q.Enqueue(0, &old.X); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if _, err := q.Dequeue(); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if err := q.Resize(4); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if err := q.Clear(); err != ErrExactQueuePending {
							t.Fatal(err)
						}
					}
					h.must(q.Close())
					h.must(q.Close())
					lpExactVectorNamedCensus(t, h, nil)
					if state := h.m.ComparisonInvocationState(); state.Active {
						t.Fatal("orphan comparison", state)
					}
					if state := h.m.RawInvocationState(); state.Active != [3]bool{} {
						t.Fatal("orphan numerical call", state)
					}
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete reservation campaign", op, failures, successes)
			}
			if op == "decrease" && lexicalStops == 0 {
				t.Fatal("by-value argument stop not reached")
			}
		}
	}
}

func TestLPRationalPriorityQueueOlderComparisonIdentity(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	q := NewRationalPriorityQueue(h.m, 2)
	h.must(q.Enqueue(0, &a.X))
	before, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	h.budget.FailAfter(0)
	_, err = h.m.Compare(&a.X, &b.X)
	h.budget.FailAfter(-1)
	if err == nil {
		t.Fatal("older comparison did not fail")
	}
	var roots numeral.NumericAllocationRoots
	h.must(roots.After(h.m.Z, before))
	older := h.m.ComparisonInvocationState()
	if !older.Active {
		t.Fatal(older)
	}
	if err := q.Enqueue(1, &b.X); err != numeral.ErrRationalRawArithmeticPending {
		t.Fatal(err)
	}
	if q.calls.pending() {
		t.Fatal("claimed older comparison")
	}
	h.must(q.Close())
	if got := h.m.ComparisonInvocationState(); got != older {
		t.Fatal("consumed older comparison", got, older)
	}
	generation := older.Generation
	h.must(h.m.CleanupComparisonInvocation(&generation))
	h.must(numeral.CleanupNumericAllocationRoots(h.m.Z, &roots))
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairPriorityQueueAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"new", "decrease", "increase", "dequeue"} {
			failures, successes, lexicalStops := 0, 0, 0
			for fail := int64(0); fail < 100; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%v/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					digits := strings.Repeat("1234567890", 100)
					old := h.pair(digits+"/7", digits+"/11")
					next := h.pair("9"+digits+"/13", "8"+digits+"/17")
					low := h.pair(digits+"/"+strings.Repeat("9876543210", 200)+"7", digits+"/"+strings.Repeat("7654321090", 200)+"9")
					q := NewPairPriorityQueue(h.m, 4)
					h.must(q.Enqueue(0, old))
					if op == "decrease" || op == "increase" {
						h.must(q.Enqueue(1, next))
					}
					if op == "dequeue" {
						a, b, c := h.pair(digits+"/11", "1"), h.pair(digits+"/13", "2"), h.pair(digits+"/17", "3")
						for i, v := range []*RationalPair{a, b, c} {
							h.must(q.Enqueue(uint32(i+1), v))
						}
					}
					h.budget.FailAfter(fail)
					var err error
					switch op {
					case "new":
						err = q.Enqueue(1, next)
					case "decrease":
						err = q.Enqueue(1, low)
					case "increase":
						err = q.Enqueue(0, next)
					case "dequeue":
						_, err = q.Dequeue()
					}
					h.budget.FailAfter(-1)
					if err == nil {
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal("unexpected reservation error", err)
						}
						failures++
						if q.decreaseArg != nil {
							lexicalStops++
						}
					}
					if q.ready() != nil {
						if err := q.Enqueue(0, old); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if _, err := q.Dequeue(); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if err := q.Resize(4); err != ErrExactQueuePending {
							t.Fatal(err)
						}
						if err := q.Clear(); err != ErrExactQueuePending {
							t.Fatal(err)
						}
					}
					h.must(q.Close())
					h.must(q.Close())
					lpExactVectorNamedCensus(t, h, nil)
					if state := h.m.ComparisonInvocationState(); state.Active {
						t.Fatal("orphan comparison", state)
					}
					if state := h.m.RawInvocationState(); state.Active != [3]bool{} {
						t.Fatal("orphan numerical call", state)
					}
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete reservation campaign", op, failures, successes)
			}
			if op == "decrease" && lexicalStops == 0 {
				t.Fatal("by-value argument stop not reached")
			}
		}
	}
}

func TestLPPairPriorityQueueOlderComparisonIdentity(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	q := NewPairPriorityQueue(h.m, 2)
	h.must(q.Enqueue(0, a))
	before, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	h.budget.FailAfter(0)
	_, err = h.m.Compare(&a.X, &b.X)
	h.budget.FailAfter(-1)
	if err == nil {
		t.Fatal("older comparison did not fail")
	}
	var roots numeral.NumericAllocationRoots
	h.must(roots.After(h.m.Z, before))
	older := h.m.ComparisonInvocationState()
	if !older.Active {
		t.Fatal(older)
	}
	if err := q.Enqueue(1, b); err != numeral.ErrRationalRawArithmeticPending {
		t.Fatal(err)
	}
	if q.calls.pending() {
		t.Fatal("claimed older comparison")
	}
	h.must(q.Close())
	if got := h.m.ComparisonInvocationState(); got != older {
		t.Fatal("consumed older comparison", got, older)
	}
	generation := older.Generation
	h.must(h.m.CleanupComparisonInvocation(&generation))
	h.must(numeral.CleanupNumericAllocationRoots(h.m.Z, &roots))
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPriorityQueueRetainedShrinkMutationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	values := []*RationalPair{h.pair("0", "0"), h.pair("1", "0"), h.pair("2", "0"), h.pair("3", "0")}
	for _, world := range []string{"clear", "decrease-child", "decrease-parent", "increase-left", "increase-right", "dequeue"} {
		q := NewRationalPriorityQueue(h.m, 4)
		objects := []uint32{3, 0, 1}
		if world == "clear" {
			objects = []uint32{0, 3}
		}
		if world == "increase-left" {
			objects = []uint32{0, 3, 1}
		}
		if world == "increase-right" {
			objects = []uint32{0, 1, 3}
		}
		if world == "dequeue" {
			objects = []uint32{0, 1, 3}
		}
		for priority, object := range objects {
			v := values[priority]
			h.must(q.Enqueue(object, &v.X))
		}
		h.must(q.Resize(2))
		var err error
		switch world {
		case "clear":
			err = q.Clear()
		case "decrease-child":
			v := values[0]
			err = q.Enqueue(1, &v.X)
		case "decrease-parent":
			v := values[0]
			err = q.Enqueue(0, &v.X)
		case "increase-left", "increase-right":
			v := values[3]
			err = q.Enqueue(0, &v.X)
		case "dequeue":
			_, err = q.Dequeue()
		}
		if err != ErrBounds {
			t.Fatal(world, err)
		}
		if world == "clear" && (q.Size() != 2 || q.inverse[0] != -1) {
			t.Fatal(world, q.Size(), q.inverse)
		}
		if strings.HasPrefix(world, "decrease") {
			if q.decreaseArg == nil || q.ready() != ErrExactQueuePending {
				t.Fatal("lost by-value owner", world)
			}
			lpCheckExactPermutationCoordinate(t, h, &q.decreaseArg.Data[0], big.NewRat(0, 1))
		}
		h.must(q.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	q := NewRationalPriorityQueue(h.m, 5)
	for priority, object := range []uint32{0, 1, 4} {
		v := values[priority]
		h.must(q.Enqueue(object, &v.X))
	}
	h.must(q.Resize(4))
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 || q.heap[1] != 4 || q.inverse[0] != 1 {
		t.Fatal(err, q.Size(), q.heap, q.inverse)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	// Wrapped growth destroys retained exact slots before the failed access.
	q = NewRationalPriorityQueue(h.m, 2)
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	h.must(q.Enqueue(0, &large.X))
	if err := q.Enqueue(1<<31, &large.X); err != ErrBounds || len(q.priorities.Data) != 0 || len(q.heap) != 1 || q.Size() != 1 {
		t.Fatal(err, q.Size(), q.heap)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairPriorityQueueRetainedShrinkMutationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	values := []*RationalPair{h.pair("0", "0"), h.pair("1", "0"), h.pair("2", "0"), h.pair("3", "0")}
	for _, world := range []string{"clear", "decrease-child", "decrease-parent", "increase-left", "increase-right", "dequeue"} {
		q := NewPairPriorityQueue(h.m, 4)
		objects := []uint32{3, 0, 1}
		if world == "clear" {
			objects = []uint32{0, 3}
		}
		if world == "increase-left" {
			objects = []uint32{0, 3, 1}
		}
		if world == "increase-right" {
			objects = []uint32{0, 1, 3}
		}
		if world == "dequeue" {
			objects = []uint32{0, 1, 3}
		}
		for priority, object := range objects {
			v := values[priority]
			h.must(q.Enqueue(object, v))
		}
		h.must(q.Resize(2))
		var err error
		switch world {
		case "clear":
			err = q.Clear()
		case "decrease-child":
			v := values[0]
			err = q.Enqueue(1, v)
		case "decrease-parent":
			v := values[0]
			err = q.Enqueue(0, v)
		case "increase-left", "increase-right":
			v := values[3]
			err = q.Enqueue(0, v)
		case "dequeue":
			_, err = q.Dequeue()
		}
		if err != ErrBounds {
			t.Fatal(world, err)
		}
		if world == "clear" && (q.Size() != 2 || q.inverse[0] != -1) {
			t.Fatal(world, q.Size(), q.inverse)
		}
		if strings.HasPrefix(world, "decrease") {
			if q.decreaseArg == nil || q.ready() != ErrExactQueuePending {
				t.Fatal("lost by-value owner", world)
			}
			lpCheckExactPermutationCoordinate(t, h, &q.decreaseArg.Data[0].X, big.NewRat(0, 1))
		}
		h.must(q.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	q := NewPairPriorityQueue(h.m, 5)
	for priority, object := range []uint32{0, 1, 4} {
		v := values[priority]
		h.must(q.Enqueue(object, v))
	}
	h.must(q.Resize(4))
	if _, err := q.Dequeue(); err != ErrBounds || q.Size() != 2 || q.heap[1] != 4 || q.inverse[0] != 1 {
		t.Fatal(err, q.Size(), q.heap, q.inverse)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
	// Wrapped growth destroys retained exact slots before the failed access.
	q = NewPairPriorityQueue(h.m, 2)
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	h.must(q.Enqueue(0, large))
	if err := q.Enqueue(1<<31, large); err != ErrBounds || len(q.priorities.Data) != 0 || len(q.heap) != 1 || q.Size() != 1 {
		t.Fatal(err, q.Size(), q.heap)
	}
	h.must(q.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func lpExactPriorityPhysicalRefusal(err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var failure umemory.Failure
	return errors.As(err, &failure) && failure.Kind == umemory.FailureMemout && failure.ErrorCode() == 101
}
