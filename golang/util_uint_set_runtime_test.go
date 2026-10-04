// Source: src/util/uint_set.h; src/test/uint_set.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-12-07.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUintSetRuntimeSourceBoundary(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	for _, v := range []uint32{0, 31, 32, 63, 100} {
		heapMust(t, s.Insert(v))
	}
	for v := uint32(0); v < 128; v++ {
		got, e := s.Contains(v)
		heapMust(t, e)
		want := v == 0 || v == 31 || v == 32 || v == 63 || v == 100
		if got != want {
			t.Fatal(v, got, want)
		}
	}
	n, e := s.NumElems()
	heapMust(t, e)
	if n != 5 {
		t.Fatal(n)
	}
	it, e := s.Begin()
	heapMust(t, e)
	for _, want := range []uint32{0, 31, 32, 63, 100} {
		v, e := it.Value()
		heapMust(t, e)
		if v != want {
			t.Fatal(v, want)
		}
		heapMust(t, it.Advance())
	}
	end, e := s.End()
	heapMust(t, e)
	if !it.Equal(end) {
		t.Fatal("iterator endpoint")
	}
	heapMust(t, s.Reset())
	empty, e := s.Empty()
	heapMust(t, e)
	if !empty {
		t.Fatal("reset")
	}
	heapMust(t, s.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestUintSetRuntimeIndependentWorlds(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	a, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	c, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	for x := uint32(0); x < 32; x++ {
		for y := uint32(0); y < 32; y++ {
			heapMust(t, a.Reset())
			heapMust(t, c.Reset())
			for i := uint32(0); i < 5; i++ {
				if x&(1<<i) != 0 {
					heapMust(t, a.Insert(i*31))
				}
				if y&(1<<i) != 0 {
					heapMust(t, c.Insert(i*31))
				}
			}
			sub, e := a.SubsetOf(c)
			heapMust(t, e)
			if sub != (x&^y == 0) {
				t.Fatal(x, y, sub)
			}
			eq, e := a.Equal(c)
			heapMust(t, e)
			if eq != (x == y) {
				t.Fatal(x, y, eq)
			}
			heapMust(t, a.Union(c))
			n, e := a.NumElems()
			heapMust(t, e)
			want := uint32(0)
			for i := uint32(0); i < 5; i++ {
				if (x|y)&(1<<i) != 0 {
					want++
				}
			}
			if n != want {
				t.Fatal(x, y, n, want)
			}
			heapMust(t, a.Intersect(c))
			eq, e = a.Equal(c)
			heapMust(t, e)
			if !eq {
				t.Fatal(x, y, "intersection")
			}
		}
	}
	if cleanup := a.Close(); cleanup != nil {
		if !a.destroyed || cleanup != memory.ErrRuntimeOwner {
			t.Fatal("unexpected assignment source unwind", cleanup)
		}
		uintSetRuntimeAudit(t, r, a, c)
		heapMust(t, a.CleanupStopped())
	}
	heapMust(t, c.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func uintSetRuntimeAudit(t *testing.T, r *memory.Runtime, sets ...*RuntimeUintSet) {
	t.Helper()
	seen := map[memory.Block]bool{}
	var physical, bitmap, cold uint64
	for _, s := range sets {
		if s == nil || s.closed {
			continue
		}
		cold += s.ownerBytes
		blocks := []memory.Block{s.words.raw.block, s.words.raw.pending}
		if s.words.raw.pendingReplaces || s.destroyed {
			blocks[0] = memory.Block{}
		}
		for _, h := range blocks {
			if h == (memory.Block{}) {
				continue
			}
			if seen[h] {
				t.Fatal("duplicate owning block", h)
			}
			seen[h] = true
			n, e := r.Size(h)
			heapMust(t, e)
			stride := uint64(r.SizeTBits() / 8)
			physical += n + stride
			bitmap += ((n + stride + 63) / 64) * 8
		}
	}
	st := r.Stats()
	if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
		t.Fatal("named physical graph", st, len(seen), physical, bitmap)
	}
	if r.Budget().Used() != st.TotalBytes+cold {
		t.Fatal("once-only cold graph", r.Budget().Used(), st.TotalBytes, cold)
	}
}
func TestUintSetRuntimeEveryActualRefusal(t *testing.T) {
	for route := 0; route < 4; route++ {
		failures, success := 0, 0
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 24)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
			heapMust(t, e)
			a, e := NewRuntimeUintSet(r, b)
			heapMust(t, e)
			c, e := NewRuntimeUintSet(r, b)
			heapMust(t, e)
			heapMust(t, a.Insert(1))
			heapMust(t, c.Insert(511))
			old := a.words.raw.block
			b.FailAfter(ordinal)
			var copy *RuntimeUintSet
			switch route {
			case 0:
				e = a.Insert(511)
			case 1:
				e = a.Union(c)
			case 2:
				copy, e = c.Clone(r, b)
			case 3:
				e = a.Assign(c)
			}
			uintSetRuntimeAudit(t, r, a, c, copy)
			if e != nil {
				failures++
				intHeapActualRefusal(t, r, e)
				has, probe := c.Contains(511)
				heapMust(t, probe)
				if !has {
					t.Fatal("source modified")
				}
				if route == 3 {
					if a.words.raw.block != old {
						t.Fatal("source assignment dangling member changed before copy publication", a.words.raw.block, old)
					}
					if _, probe = r.Size(old); probe == nil {
						t.Fatal("assignment did not destroy first")
					}
				}
			} else {
				success++
				var dest *RuntimeUintSet = a
				if route == 2 {
					dest = copy
				}
				has, e := dest.Contains(511)
				heapMust(t, e)
				if !has {
					t.Fatal("successful endpoint")
				}
			}
			b.FailAfter(-1)
			if copy != nil {
				if cleanup := copy.Close(); cleanup != nil {
					if !copy.words.raw.pendingReplaces || cleanup != memory.ErrRuntimeOwner {
						t.Fatal("unexpected source unwind guard", cleanup)
					}
					uintSetRuntimeAudit(t, r, a, c, copy)
					heapMust(t, copy.CleanupStopped())
				}
			}
			if cleanup := a.Close(); cleanup != nil {
				if !a.destroyed || cleanup != memory.ErrRuntimeOwner {
					t.Fatal("unexpected assignment source unwind", cleanup)
				}
				uintSetRuntimeAudit(t, r, a, c)
				heapMust(t, a.CleanupStopped())
			}
			heapMust(t, c.Close())
			heapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("refund", b.Used())
			}
			if success != 0 {
				t.Logf("route%d refusals%d firstsuccess%d", route, failures, ordinal)
				break
			}
		}
		if success != 1 || failures == 0 {
			t.Fatal(route, failures, success)
		}
	}
}
func TestUintSetRuntimeLiveIteratorAndCopyCapacity(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(1))
	heapMust(t, s.Insert(63))
	it, e := s.Begin()
	heapMust(t, e)
	heapMust(t, s.Insert(511))
	heapMust(t, s.Remove(63))
	heapMust(t, s.Insert(62))
	heapMust(t, it.Advance())
	v, e := it.Value()
	heapMust(t, e)
	if v != 62 {
		t.Fatal(v)
	}
	heapMust(t, it.Advance())
	v, e = it.Value()
	heapMust(t, e)
	if v != 64 {
		t.Fatal("captured endpoint", v)
	}
	clone, e := s.Clone(r, b)
	heapMust(t, e)
	eq, e := clone.Equal(s)
	heapMust(t, e)
	if !eq {
		t.Fatal("copy")
	}
	heapMust(t, s.Reset())
	heapMust(t, s.Insert(0))
	cap, e := s.words.capacity()
	heapMust(t, e)
	heapMust(t, clone.Assign(s))
	got, e := clone.words.capacity()
	heapMust(t, e)
	if got != cap {
		t.Fatal(got, cap)
	}
	uintSetRuntimeAudit(t, r, s, clone)
	heapMust(t, s.Close())
	heapMust(t, clone.Close())
	heapMust(t, r.Close())
}
func TestUintSetRuntimeOwnerGuardsAndWarmPlateau(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	foreign := arena.NewBudget(1 << 24)
	before := foreign.Used()
	if _, e = NewRuntimeUintSet(r, foreign); e != arena.ErrConfig || foreign.Used() != before {
		t.Fatal(e)
	}
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(127))
	copy := *s
	if e = copy.Close(); e != ErrSATOwner {
		t.Fatal(e)
	}
	heapMust(t, s.Reset())
	heapMust(t, s.Insert(127))
	st := r.Stats()
	used := b.Used()
	b.FailAfter(0)
	for i := 0; i < 100; i++ {
		heapMust(t, s.Reset())
		heapMust(t, s.Insert(127))
		heapMust(t, s.Remove(127))
	}
	if r.Stats() != st || b.Used() != used {
		t.Fatal("physical plateau")
	}
	uintSetRuntimeAudit(t, r, s)
	b.FailAfter(-1)
	heapMust(t, s.Close())
	if _, e = s.Contains(0); e != arena.ErrClosed {
		t.Fatal(e)
	}
	heapMust(t, r.Close())
}

func TestUintSetRuntimeSourceDestructorAndOwningAbort(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(1600000))
	old := s.words.raw.block
	// Actual public TLS synchronization reaches the guarded signed-overflow
	// domain; no private counter writes or invented deallocation refusals.
	satHeapTLSNearMinimum(t, r)
	e = s.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || s.words.raw.block != (memory.Block{}) || s.closed {
		t.Fatal("postphysical source prefix", e, s.closed)
	}
	if _, e = r.Size(old); e == nil {
		t.Fatal("source block remains live")
	}
	uintSetRuntimeAudit(t, r, s)
	cause := s.destructorStop
	if e = s.Close(); e != cause {
		t.Fatal("ordinary retry erased source stop", e, cause)
	}
	heapMust(t, s.CleanupStopped())
	if s.destructorStop != cause {
		t.Fatal("lost history")
	}
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// The source random driver's ENSURE expressions are executed with a fixed,
// independently generated campaign. libc rand's platform stream is not claimed.
func TestUintSetRuntimeSourceRandomCampaign(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	rng := uint32(19)
	next := func() uint32 { rng = rng*1664525 + 1013904223; return rng }
	assertions := 0
	for round := 0; round < 200; round++ {
		n := uint32(1) + next()%100
		heapMust(t, s.Reset())
		model := make([]bool, n)
		count := uint32(0)
		ops := next() % 1000
		for j := uint32(0); j < ops; j++ {
			op := next() % 3
			v := next() % n
			if op < 2 {
				if !model[v] {
					count++
				}
				model[v] = true
				heapMust(t, s.Insert(v))
			} else {
				if model[v] {
					count--
				}
				model[v] = false
				heapMust(t, s.Remove(v))
			}
			got, e := s.NumElems()
			heapMust(t, e)
			if got != count {
				t.Fatal(round, j, got, count)
			}
			empty, e := s.Empty()
			heapMust(t, e)
			if empty != (count == 0) {
				t.Fatal(round, j, "empty")
			}
			assertions += 2
			for k := uint32(0); k < n; k++ {
				got, e := s.Contains(k)
				heapMust(t, e)
				if got != model[k] {
					t.Fatal(round, j, k)
				}
				assertions++
			}
		}
	}
	t.Logf("source ENSURE-expression executions %d", assertions)
	uintSetRuntimeAudit(t, r, s)
	heapMust(t, s.Close())
	heapMust(t, r.Close())
}
func TestUintSetRuntimeHashDisplayAndStreamPrefixes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	for _, v := range []uint32{1, 2, 44} {
		heapMust(t, s.Insert(v))
	}
	h, e := s.Hash()
	heapMust(t, e)
	if h != 8198 {
		t.Fatal(h)
	}
	out := &uintSetPrefixWriter{remaining: 100}
	heapMust(t, s.Display(out))
	want := "{1, 2, 44}"
	if string(out.bytes) != want {
		t.Fatal(string(out.bytes))
	}
	for n := 0; n < len(want); n++ {
		w := &uintSetPrefixWriter{remaining: n}
		if e = s.Display(w); e != uintSetStreamRefusal {
			t.Fatal(n, e)
		}
		if string(w.bytes) != want[:n] {
			t.Fatal(n, string(w.bytes), want[:n])
		}
		uintSetRuntimeAudit(t, r, s)
	}
	heapMust(t, s.Close())
	heapMust(t, r.Close())
}

var uintSetStreamRefusal = errors.New("uintset test stream refusal")

type uintSetPrefixWriter struct {
	remaining int
	bytes     []byte
}

func (w *uintSetPrefixWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n := w.remaining
		w.bytes = append(w.bytes, p[:n]...)
		w.remaining = 0
		return n, uintSetStreamRefusal
	}
	w.bytes = append(w.bytes, p...)
	w.remaining -= len(p)
	return len(p), nil
}
func TestUintSetRuntimeIteratorResetAndEndSourceProfiles(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	s, e := NewRuntimeUintSet(r, b)
	heapMust(t, e)
	heapMust(t, s.Insert(1))
	it, e := s.Begin()
	heapMust(t, e)
	old, e := it.PostAdvance()
	heapMust(t, e)
	v, e := old.Value()
	heapMust(t, e)
	if v != 1 {
		t.Fatal(v)
	}
	heapMust(t, s.Reset())
	copyBeforeReset := old
	heapMust(t, copyBeforeReset.Advance())
	v, e = copyBeforeReset.Value()
	heapMust(t, e)
	if v != 32 {
		t.Fatal("reset live iterator", v)
	}
	// Incrementing an already-at-end source iterator is outside its domain:
	// DEBUG asserts after increment, release reaches the retained extent guard.
	e = it.Advance()
	if satStorageSourceDebug {
		if e != ErrSATAssertion {
			t.Fatal(e)
		}
	} else {
		if e != arena.ErrBounds {
			t.Fatal(e)
		}
		v, e = it.Value()
		heapMust(t, e)
		if v != 64 {
			t.Fatal(v)
		}
	}
	heapMust(t, s.Close())
	if _, e = old.Value(); e != arena.ErrClosed {
		t.Fatal(e)
	}
	heapMust(t, r.Close())
}
