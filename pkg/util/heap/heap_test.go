// Source assertions: src/test/heap.cpp, including its original finite counts.
package heap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func testRNG(seed uint32) *rand.ChaCha8 {
	var key [32]byte
	binary.LittleEndian.PutUint32(key[:], seed)
	return rand.NewChaCha8(key)
}

func testInt(t *testing.T, bounds int32, c Config) *Int {
	t.Helper()
	h, e := New(bounds, c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { h.Close() })
	return h
}
func heapMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestHeapSourceAssertions(t *testing.T) {
	const n = 10000
	for seed := uint32(0); seed < 3; seed++ {
		rng := testRNG(seed)
		h := testInt(t, n, Config{})
		var present [n]bool
		for i := 0; i < n*3; i++ {
			v := int32(rng.Uint64() % n)
			if !h.Contains(v) {
				if present[v] {
					t.Fatal("source set absence")
				}
				heapMust(t, h.Insert(v))
				present[v] = true
			} else if !present[v] {
				t.Fatal("source set membership")
			}
		}
		if !h.CheckInvariant() {
			t.Fatal("source heap invariant")
		}
		for v, yes := range present {
			if yes && !h.Contains(int32(v)) {
				t.Fatal("source set iteration")
			}
		}
		last := int32(-1)
		for !h.Empty() {
			m1, e := h.Min()
			heapMust(t, e)
			m2, e := h.EraseMin()
			heapMust(t, e)
			if m1 != m2 || m2 <= -1 {
				t.Fatal("source minimum assertions")
			}
			if m2 <= last {
				t.Fatal("added independent sorted extraction")
			}
			last = m2
		}
		var priorities [n]int32
		for i := range priorities {
			priorities[i] = int32(rng.Uint64())
		}
		mutable := testInt(t, n, Config{Less: func(a, b int32) bool {
			if a < 0 || b < 0 || a >= n || b >= n {
				t.Fatal("source comparator bounds")
			}
			return priorities[a] < priorities[b]
		}})
		var log strings.Builder
		for i := 0; i < n*10; i++ {
			// Original print-only diagnostics are captured separately from APIs.
			fmt.Fprintf(&log, "i: %d\n", i)
			if i%1000 == 0 {
				fmt.Fprintf(&log, "i: %d\n", i)
			}
			cmd := rng.Uint64() % 10
			if cmd <= 3 {
				v := int32(rng.Uint64() % n)
				if !mutable.Contains(v) {
					heapMust(t, mutable.Insert(v))
					if !mutable.Contains(v) {
						t.Fatal("source inserted membership")
					}
				}
			} else if cmd <= 6 {
				v := int32(rng.Uint64() % n)
				if mutable.Contains(v) {
					heapMust(t, mutable.Erase(v))
					if mutable.Contains(v) {
						t.Fatal("source erased membership")
					}
				}
			} else if cmd <= 8 {
				v := int32(rng.Uint64() % n)
				old, next := priorities[v], int32(rng.Uint64())
				if mutable.Contains(v) {
					priorities[v] = next
					if old < next {
						heapMust(t, mutable.Increased(v))
					} else {
						heapMust(t, mutable.Decreased(v))
					}
				}
			} else if !mutable.CheckInvariant() {
				t.Fatal("source command invariant")
			}
		}
		if !mutable.CheckInvariant() {
			t.Fatal("source final invariant")
		}
		text := log.String()
		if !strings.HasPrefix(text, "i: 0\ni: 0\n") || !strings.HasSuffix(text, "i: 99999\n") || strings.Count(text, "\n") != 100100 {
			t.Fatal("source diagnostic shape")
		}
	}
}

func TestHeapIndependentMutableOracle(t *testing.T) {
	const n = 97
	var priority [n]int32
	h := testInt(t, n, Config{Less: func(a, b int32) bool { return priority[a] < priority[b] }})
	var live [n]bool
	rng := testRNG(39)
	for step := 0; step < 5000; step++ {
		v := int32(rng.Uint64() % n)
		switch step % 5 {
		case 0:
			if live[v] {
				heapMust(t, h.Erase(v))
				live[v] = false
			}
		case 1:
			if !live[v] {
				priority[v] = int32(rng.Uint64())
				heapMust(t, h.Insert(v))
				live[v] = true
			}
		default:
			if live[v] {
				old := priority[v]
				priority[v] = int32(rng.Uint64())
				if priority[v] < old {
					heapMust(t, h.Decreased(v))
				} else {
					heapMust(t, h.Increased(v))
				}
			}
		}
		count := uint32(0)
		best := int32(1 << 30)
		for i, yes := range live {
			if h.Contains(int32(i)) != yes {
				t.Fatal("independent membership")
			}
			if yes {
				count++
				if priority[i] < best {
					best = priority[i]
				}
			}
		}
		if count != h.Size() || !h.CheckInvariant() {
			t.Fatal("independent count/invariant")
		}
		if count > 0 {
			top, e := h.Min()
			heapMust(t, e)
			if priority[top] != best {
				t.Fatal("independent best priority")
			}
		}
	}
}

func TestHeapSourceShapeAndLifetime(t *testing.T) {
	h := testInt(t, 20, Config{})
	for _, v := range []int32{4, 2, 6, 1, 3, 5, 7} {
		heapMust(t, h.Insert(v))
	}
	values, e := h.AppendValues([]int32{99})
	heapMust(t, e)
	want := []int32{99, 1, 2, 5, 4, 3, 6, 7}
	for i, v := range want {
		if values[i] != v {
			t.Fatal("source array layout", values)
		}
	}
	text, e := h.Display()
	heapMust(t, e)
	if text != "1\n 2\n  4\n  3\n 5\n  6\n  7\n" {
		t.Fatal("source display", text)
	}
	it := h.Begin()
	if v, e := it.Value(); e != nil || v != 1 {
		t.Fatal("iterator", v, e)
	}
	heapMust(t, h.Erase(3))
	if _, e := it.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale iterator", e)
	}
	if e := h.SetBounds(4); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("unsafe live shrink", e)
	}
	heapMust(t, h.Reserve(30))
	if h.Bounds() != 30 {
		t.Fatal("reserve bounds")
	}
	heapMust(t, h.Reserve(-1))
	heapMust(t, h.Clear())
	heapMust(t, h.SetBounds(0))
	if !h.Empty() || h.Contains(0) {
		t.Fatal("empty bounded heap")
	}
	if _, e := h.Min(); !errors.Is(e, ErrValue) {
		t.Fatal("empty minimum", e)
	}
	if e := h.Insert(-1); !errors.Is(e, ErrValue) {
		t.Fatal("negative key", e)
	}
	heapMust(t, h.Close())
	if e := h.Insert(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed heap", e)
	}
}

func TestHeapBudgetsFailuresSwapAndWarm(t *testing.T) {
	for prefix := int64(0); prefix < 5; prefix++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(prefix)
		h, e := New(32, Config{Budget: b})
		if e == nil {
			h.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("constructor leak")
		}
	}
	b := arena.NewBudget(1 << 20)
	h := testInt(t, 32, Config{Budget: b})
	heapMust(t, h.Insert(1))
	before := b.Used()
	b.FailAfter(0)
	if e := h.Insert(2); !errors.Is(e, arena.ErrInjectedFailure) || h.Size() != 1 || !h.Contains(2) || b.Used() != before {
		t.Fatal("growth publication", e)
	}
	b.FailAfter(-1)
	heapMust(t, h.Reset())
	heapMust(t, h.Insert(1))
	heapMust(t, h.Insert(2))
	for i := int32(3); i < 10; i++ {
		heapMust(t, h.Insert(i))
	}
	heapMust(t, h.Reset())
	if allocations := testing.AllocsPerRun(100, func() { h.Insert(1); h.Insert(2); h.EraseMin(); h.EraseMin() }); allocations != 0 {
		t.Fatal("warm allocations", allocations)
	}
	s := h.Stats()
	if s.OwnerBytes+s.ValuesBytes+s.IndicesBytes != b.Used() {
		t.Fatal("whole owned graph")
	}
	for fail := int64(0); fail < 2; fail++ {
		otherBudget := arena.NewBudget(1 << 20)
		other := testInt(t, 4, Config{Budget: otherBudget})
		heapMust(t, other.Insert(3))
		a, c := b.Used(), otherBudget.Used()
		b.FailAfter(fail)
		e := h.Swap(other)
		b.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != a || otherBudget.Used() != c || !other.Contains(3) {
				t.Fatal("failed swap", e)
			}
		} else {
			if !h.Contains(3) || !other.Empty() {
				t.Fatal("swapped backing")
			}
			heapMust(t, h.Swap(other))
		}
		heapMust(t, other.Close())
		if otherBudget.Used() != 0 {
			t.Fatal("swap leak")
		}
	}
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("owner close leak")
	}
}

func TestHeapEpochBoundsIteratorAndComparatorSwap(t *testing.T) {
	h := testInt(t, 8, Config{})
	heapMust(t, h.Insert(2))
	heapMust(t, h.Insert(4))
	end := h.End()
	it := h.Begin()
	heapMust(t, it.Next())
	if v, e := it.Value(); v != 4 || e != nil {
		t.Fatal(v, e)
	}
	heapMust(t, it.Next())
	if !it.Equal(end) {
		t.Fatal("iterator end")
	}
	if e := it.Next(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	heapMust(t, h.SetBounds(5))
	if !h.Contains(4) {
		t.Fatal("valid shrink lost membership")
	}
	heapMust(t, h.SetBounds(8))
	for i := int32(5); i < 8; i++ {
		if h.Contains(i) {
			t.Fatal("regrowth retained stale index")
		}
	}
	h.epoch = ^uint64(0)
	before := fmt.Sprint(h.values)
	for _, f := range []func() error{func() error { return h.Insert(3) }, func() error { return h.Erase(2) }, func() error { return h.Increased(2) }, func() error { return h.Decreased(2) }, func() error { return h.SetBounds(9) }, h.Reset} {
		if e := f(); !errors.Is(e, arena.ErrSlotLimit) {
			t.Fatal("epoch preflight", e)
		}
		if fmt.Sprint(h.values) != before {
			t.Fatal("exhausted epoch mutated")
		}
	}
	heapMust(t, h.Reserve(-1))
	heapMust(t, h.SetBounds(8))
	heapMust(t, h.Swap(h))
	heapMust(t, h.Close())
	aCalls, bCalls := 0, 0
	a := testInt(t, 8, Config{Less: func(x, y int32) bool { aCalls++; return x < y }})
	b := testInt(t, 8, Config{Less: func(x, y int32) bool { bCalls++; return x < y }})
	heapMust(t, a.Insert(1))
	heapMust(t, a.Insert(3))
	heapMust(t, b.Insert(2))
	heapMust(t, b.Insert(4))
	old := a.Begin()
	heapMust(t, a.Swap(b))
	if _, e := old.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("swap iterator", e)
	}
	aCalls, bCalls = 0, 0
	heapMust(t, a.Insert(5))
	if aCalls == 0 || bCalls != 0 {
		t.Fatal("swap moved borrowed comparator")
	}
	if !a.Contains(2) || !b.Contains(1) {
		t.Fatal("swap membership")
	}
}

func TestHeapSourceBoundsGrowth(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h := testInt(t, 32, Config{Budget: b})
	if cap(h.indices) != 41 {
		t.Fatal("source repeated resize expansion", cap(h.indices))
	}
	b.FailAfter(1)
	if e := h.SetBounds(100); !errors.Is(e, arena.ErrInjectedFailure) || h.Bounds() != 32 || cap(h.indices) != 62 {
		t.Fatal("source capacity-only failure prefix", e, h.Bounds(), cap(h.indices))
	}
	b.FailAfter(-1)
	heapMust(t, h.SetBounds(100))
	if cap(h.indices) != 140 {
		t.Fatal("source bounds capacity", cap(h.indices))
	}
	heapMust(t, h.SetBounds(10000))
	if cap(h.indices) != 12137 {
		t.Fatal("source large resize", cap(h.indices))
	}
}

func TestHeapCopyAndAllocationSizeBoundary(t *testing.T) {
	if n, e := nextCapacity(0); n != 2 || e != nil {
		t.Fatal(n, e)
	}
	if n, e := nextCapacity(41); n != 62 || e != nil {
		t.Fatal(n, e)
	}
	if _, e := nextCapacity(800000000); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("source unsigned byte-size overflow", e)
	}
	b := arena.NewBudget(1 << 20)
	h := testInt(t, 32, Config{Budget: b})
	for _, v := range []int32{4, 2, 6, 1} {
		heapMust(t, h.Insert(v))
	}
	copyBudget := arena.NewBudget(1 << 20)
	c, e := h.Clone(copyBudget)
	heapMust(t, e)
	defer c.Close()
	if cap(c.values) != cap(h.values) || cap(c.indices) != cap(h.indices) {
		t.Fatal("source copied vector capacity or nonsource scratch", c.Stats())
	}
	heapMust(t, c.Erase(2))
	if !h.Contains(2) {
		t.Fatal("copy aliases source")
	}
	for prefix := int64(0); prefix < 3; prefix++ {
		before := copyBudget.Used()
		copyBudget.FailAfter(prefix)
		got, err := h.Clone(copyBudget)
		if got != nil {
			got.Close()
		}
		if !errors.Is(err, arena.ErrInjectedFailure) || copyBudget.Used() != before {
			t.Fatal("copy failed prefix", prefix, err)
		}
	}
	copyBudget.FailAfter(-1)
	it := c.Begin()
	for prefix := int64(0); prefix < 2; prefix++ {
		candidate := testInt(t, 4, Config{Budget: copyBudget})
		heapMust(t, candidate.Insert(1))
		before := copyBudget.Used()
		oldValues, oldIndices := uint64(cap(candidate.values))*4, uint64(cap(candidate.indices))*4
		copyBudget.FailAfter(prefix)
		err := candidate.Assign(h)
		want := before - oldValues
		if prefix == 1 {
			want += uint64(cap(h.values))*4 - oldIndices
		}
		if !errors.Is(err, arena.ErrInjectedFailure) || copyBudget.Used() != want {
			t.Fatal("source destructive member frontier", prefix, err, copyBudget.Used(), want)
		}
		copyBudget.FailAfter(-1)
		heapMust(t, candidate.CleanupStopped())
	}
	heapMust(t, c.Assign(h))
	if !c.Contains(2) || !c.CheckInvariant() {
		t.Fatal("assignment")
	}
	if _, e := it.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("assignment invalidation", e)
	}
	heapMust(t, c.Assign(c))
	heapMust(t, c.Close())
	if copyBudget.Used() != 0 {
		t.Fatal("copy owner leak")
	}
}

func TestHeapSwapSecondReservationAndFinitePeak(t *testing.T) {
	ab, bb := arena.NewBudget(1<<20), arena.NewBudget(1<<20)
	a := testInt(t, 32, Config{Budget: ab})
	b := testInt(t, 4, Config{Budget: bb})
	heapMust(t, a.Insert(1))
	heapMust(t, b.Insert(2))
	ai, bi := a.Begin(), b.Begin()
	au, bu := ab.Used(), bb.Used()
	bb.FailAfter(0)
	if e := a.Swap(b); !errors.Is(e, arena.ErrInjectedFailure) || ab.Used() != au || bb.Used() != bu || !a.Contains(1) || !b.Contains(2) {
		t.Fatal("second swap reserve rollback", e)
	}
	if _, e := ai.Value(); e != nil {
		t.Fatal("failed swap invalidated first owner", e)
	}
	if _, e := bi.Value(); e != nil {
		t.Fatal("failed swap invalidated second owner", e)
	}
	bb.FailAfter(-1)
	// Constructor bounds2 uses indices capacity2. Exact retained graph fits;
	// one further insert needs replacement values capacity3 while cap2 lives.
	peak := arena.NewBudget(ownerBytes + 2*4 + 2*4 + 3*4 - 1)
	p := testInt(t, 2, Config{Budget: peak})
	heapMust(t, p.Insert(0))
	used := peak.Used()
	if e := p.Insert(1); !errors.Is(e, arena.ErrBudget) || peak.Used() != used || !p.Contains(1) || !p.CheckInvariant() {
		t.Fatal("old plus new capacity refusal", e)
	}
}
