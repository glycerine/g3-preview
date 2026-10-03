// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete u_map<uint32> coverage for util/map.h and hashtable.h.
package utilhash

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/rand/v2"
	"testing"
)

func newUM(t *testing.T) *UintMap {
	t.Helper()
	m, e := NewUintMap(UintMapConfig{})
	mapMust(t, e)
	t.Cleanup(m.Close)
	return m
}
func mapMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func mapKeys(t *testing.T, m *UintMap) []uint32 {
	t.Helper()
	var keys []uint32
	for it := m.Begin(); !it.AtEnd(); {
		p, e := it.Entry()
		mapMust(t, e)
		keys = append(keys, p.Key)
		mapMust(t, it.Next())
	}
	return keys
}
func expectMapKeys(t *testing.T, m *UintMap, want ...uint32) {
	t.Helper()
	got := mapKeys(t, m)
	if len(got) != len(want) {
		t.Fatal(got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatal(got, want)
		}
	}
}

// Assertion categories from active src/test/map.cpp tst1, specialized to the
// u_map uint32 source variant. Original string hashing/equality remain separate.
func TestUintMapSourceMapAssertions(t *testing.T) {
	m := newUM(t)
	mapMust(t, m.Insert(10, 35))
	if !m.Contains(10) || m.FindIterator(10).Equal(m.End()) {
		t.Fatal("first key missing")
	}
	p, e := m.FindIterator(10).Entry()
	mapMust(t, e)
	if p.Value != 35 || m.Size() != 1 {
		t.Fatal("first value/size")
	}
	mapMust(t, m.Insert(20, 32))
	for _, v := range []UintMapPair{{10, 35}, {20, 32}} {
		if !m.Contains(v.Key) || m.FindIterator(v.Key).Equal(m.End()) {
			t.Fatal("key missing")
		}
		p, e := m.FindIterator(v.Key).Entry()
		mapMust(t, e)
		if p.Value != v.Value {
			t.Fatal("value")
		}
	}
	if m.Size() != 2 {
		t.Fatal("second size")
	}
	mapMust(t, m.Remove(20))
	if m.Size() != 1 || m.Contains(20) || !m.Contains(10) {
		t.Fatal("remove")
	}
}
func TestUintMapLastTombstoneAndWrap(t *testing.T) {
	m := newUM(t)
	for _, k := range []uint32{0, 8, 16} {
		mapMust(t, m.Insert(k, k+100))
	}
	mapMust(t, m.Remove(0))
	mapMust(t, m.Remove(8))
	if m.Deleted() != 2 {
		t.Fatal("expected two tombstones")
	}
	mapMust(t, m.Insert(24, 124))
	expectMapKeys(t, m, 24, 16)
	if m.Deleted() != 1 {
		t.Fatal("last tombstone reuse")
	}
	mapMust(t, m.Insert(16, 216))
	if m.MustFind(16) != 216 || m.Size() != 2 {
		t.Fatal("overwrite")
	}
	mapMust(t, m.Reset())
	for _, k := range []uint32{7, 15, 23} {
		mapMust(t, m.Insert(k, k))
	}
	expectMapKeys(t, m, 15, 23, 7)
	mapMust(t, m.Remove(7))
	mapMust(t, m.Insert(31, 31))
	expectMapKeys(t, m, 15, 23, 31)
	if m.Contains(7) || m.Deleted() != 0 {
		t.Fatal("wrapped tombstone")
	}
	mapMust(t, m.Remove(23))
	if m.Deleted() != 0 {
		t.Fatal("next-free deletion must not create tombstone")
	}
}
func TestUintMapStrictGrowthAndInsertIfAbsent(t *testing.T) {
	m := newUM(t)
	for k := uint32(0); k < 7; k++ {
		mapMust(t, m.Insert(k, k+10))
		if m.Capacity() != 8 {
			t.Fatal("growth happened before strict threshold", k)
		}
	}
	mapMust(t, m.Insert(0, 88))
	if m.Capacity() != 16 || m.Size() != 7 || m.MustFind(0) != 88 {
		t.Fatal("existing-key update must grow before lookup")
	}
	inserted, ref, e := m.InsertIfNotThereCore(0, 99)
	mapMust(t, e)
	p, e := ref.Entry()
	mapMust(t, e)
	if inserted || p.Value != 88 {
		t.Fatal("existing value overwritten")
	}
	inserted, ref, e = m.InsertIfNotThereCore(50, 500)
	mapMust(t, e)
	p, e = ref.Entry()
	mapMust(t, e)
	if !inserted || p.Key != 50 || p.Value != 500 {
		t.Fatal("new insert-if-absent")
	}
	r, e := m.InsertIfNotThere(50, 900)
	mapMust(t, e)
	mapMust(t, r.SetValue(600))
	if m.MustFind(50) != 600 {
		t.Fatal("mutable value ref")
	}
	r, e = m.InsertIfNotThere2(50, 999)
	mapMust(t, e)
	p, e = r.Entry()
	mapMust(t, e)
	if p.Value != 600 {
		t.Fatal("insert-if-not-there2")
	}
}
func clusteredMap(t *testing.T, oom func() bool, budget *arena.Budget) *UintMap {
	t.Helper()
	m, e := NewUintMap(UintMapConfig{Budget: budget, OutOfMemory: oom})
	mapMust(t, e)
	t.Cleanup(m.Close)
	for i := uint32(0); i < 66; i++ {
		mapMust(t, m.Insert(i*1024, i))
	}
	if m.Capacity() != 128 {
		t.Fatal("cluster capacity", m.Capacity())
	}
	return m
}
func TestUintMapCleanupThresholdAndOutOfMemory(t *testing.T) {
	m := clusteredMap(t, nil, nil)
	for i := uint32(0); i < 64; i++ {
		mapMust(t, m.Remove(i*1024))
	}
	if m.Deleted() != 64 || m.Size() != 2 {
		t.Fatal("cleanup at <=64 tombstones")
	}
	mapMust(t, m.Remove(64*1024))
	if m.Deleted() != 0 || m.Size() != 1 || m.Capacity() != 128 {
		t.Fatal("same capacity cleanup threshold")
	}
	expectMapKeys(t, m, 65*1024)
	calls := 0
	skip := true
	s := clusteredMap(t, func() bool { calls++; return skip }, nil)
	for i := uint32(0); i < 65; i++ {
		mapMust(t, s.Remove(i*1024))
	}
	if calls != 1 || s.Deleted() != 65 || s.Size() != 1 {
		t.Fatal("OOM cleanup skip", calls, s.Deleted(), s.Size())
	}
	skip = false
	mapMust(t, s.Insert(80*1024, 80))
	mapMust(t, s.Remove(80*1024))
	if s.Deleted() != 0 || calls != 2 {
		t.Fatal("cleanup resumes after OOM clears", s.Deleted(), calls)
	}
}
func TestUintMapResetFinalizeAndClone(t *testing.T) {
	m := clusteredMap(t, nil, nil)
	mapMust(t, m.Reset())
	if m.Capacity() != 128 {
		t.Fatal("reset shrank table with too few preexisting free buckets")
	}
	mapMust(t, m.Insert(9, 90))
	mapMust(t, m.Reset())
	if m.Capacity() != 64 {
		t.Fatal("sparse reset should halve once")
	}
	mapMust(t, m.Reset())
	if m.Capacity() != 64 {
		t.Fatal("empty reset must return early")
	}
	large := clusteredMap(t, nil, nil)
	mapMust(t, large.Finalize())
	if large.Capacity() != 64 || !large.Empty() {
		t.Fatal("finalize large")
	}
	small := newUM(t)
	mapMust(t, small.Insert(0, 100))
	mapMust(t, small.Insert(8, 108))
	mapMust(t, small.Insert(16, 116))
	mapMust(t, small.Remove(0))
	copy, e := small.Clone()
	mapMust(t, e)
	t.Cleanup(copy.Close)
	expectMapKeys(t, copy, 8, 16)
	if copy.Capacity() != 8 || copy.Deleted() != 0 || copy.Size() != 2 {
		t.Fatal("copy rehash tombstones")
	}
	mapMust(t, copy.Insert(8, 908))
	if small.MustFind(8) != 108 {
		t.Fatal("clone alias")
	}
	mapMust(t, small.Finalize())
	if small.Capacity() != 8 || !small.Empty() || small.Deleted() != 0 {
		t.Fatal("small finalize")
	}
}
func TestUintMapMethodsRefsAndIterators(t *testing.T) {
	m := newUM(t)
	mapMust(t, m.Insert(0, 0))
	mapMust(t, m.Insert(^uint32(0), ^uint32(0)))
	if v, ok := m.Find(0); !ok || v != 0 {
		t.Fatal("zero value valid")
	}
	out := uint32(123)
	if m.FindInto(15, &out) || out != 123 || m.Get(15, 77) != 77 {
		t.Fatal("find misses preserve output/default")
	}
	if !m.FindInto(0, &out) || out != 0 || m.Get(0, 77) != 0 {
		t.Fatal("find hits")
	}
	r, ok := m.FindCore(0)
	if !ok {
		t.Fatal("find_core")
	}
	mapMust(t, m.Insert(0, 42))
	p, e := r.Entry()
	mapMust(t, e)
	if p.Value != 42 {
		t.Fatal("value update preserves nonstructural ref")
	}
	mapMust(t, m.Insert(7, 70))
	if _, e := r.Entry(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("membership mutation invalidates snapshot", e)
	}
	it := m.Begin()
	old, e := it.PostNext()
	mapMust(t, e)
	p, e = old.Entry()
	mapMust(t, e)
	if p.Key != 0 {
		t.Fatal("postincrement old position")
	}
	for !it.AtEnd() {
		mapMust(t, it.Next())
	}
	if !it.Equal(m.End()) {
		t.Fatal("end equality")
	}
	mapMust(t, m.Erase(7))
	if _, e := old.Entry(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale iterator", e)
	}
	if !m.FindIterator(1234).Equal(m.End()) || m.CollisionCount() != 0 || !m.CheckInvariant() {
		t.Fatal("source wrappers/invariant")
	}
	ref, ok := m.FindCore(0)
	if !ok {
		t.Fatal("ref exists")
	}
	m.Close()
	if _, e := ref.Entry(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ref after close", e)
	}
}
func TestUintMapAssignUnionIntersectionSwap(t *testing.T) {
	a, b := newUM(t), newUM(t)
	mapMust(t, a.Insert(0, 10))
	mapMust(t, a.Insert(8, 18))
	mapMust(t, b.Insert(8, 88))
	mapMust(t, b.Insert(16, 116))
	mapMust(t, a.Union(b))
	if a.Size() != 3 || a.MustFind(8) != 88 {
		t.Fatal("union overwrites value")
	}
	mapMust(t, a.Intersect(b))
	if a.Contains(0) || a.Size() != 2 || a.MustFind(8) != 88 {
		t.Fatal("intersection key semantics")
	}
	mapMust(t, b.Insert(8, 999))
	if a.MustFind(8) != 88 {
		t.Fatal("separate table storage")
	}
	mapMust(t, a.Assign(b))
	if a.MustFind(8) != 999 || a.Size() != 2 {
		t.Fatal("assignment")
	}
	mapMust(t, a.Assign(a))
	mapMust(t, a.Union(a))
	mapMust(t, a.Intersect(a))
	c := newUM(t)
	mapMust(t, c.Insert(3, 33))
	mapMust(t, a.Swap(c))
	if !a.Contains(3) || !c.Contains(16) || c.MustFind(8) != 999 {
		t.Fatal("swap")
	}
}
func TestUintMapIndependentBoundedOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(76, 93))
	m := newUM(t)
	oracle := make(map[uint32]uint32)
	for step := 0; step < 6000; step++ {
		k, v := uint32(r.IntN(257)), r.Uint32()
		switch r.IntN(7) {
		case 0, 1:
			mapMust(t, m.Insert(k, v))
			oracle[k] = v
		case 2:
			mapMust(t, m.Remove(k))
			delete(oracle, k)
		case 3:
			old, had := oracle[k]
			inserted, ref, e := m.InsertIfNotThereCore(k, v)
			mapMust(t, e)
			p, e := ref.Entry()
			mapMust(t, e)
			if inserted == had {
				t.Fatal(step, "insert flag")
			}
			if had {
				if p.Value != old {
					t.Fatal(step, "preserved value")
				}
			} else {
				oracle[k] = v
			}
		case 4:
			if step%31 == 0 {
				mapMust(t, m.Reset())
				clear(oracle)
			}
		case 5:
			copy, e := m.Clone()
			mapMust(t, e)
			if copy.Size() != uint32(len(oracle)) {
				t.Fatal("clone cardinality")
			}
			for k, v := range oracle {
				if g, ok := copy.Find(k); !ok || g != v {
					t.Fatal("clone contents")
				}
			}
			copy.Close()
		case 6:
			if step%47 == 0 {
				mapMust(t, m.Finalize())
				clear(oracle)
			}
		}
		if m.Size() != uint32(len(oracle)) || m.Empty() != (len(oracle) == 0) || !m.CheckInvariant() {
			t.Fatal(step, "size/invariant")
		}
		seen := make(map[uint32]bool)
		for it := m.Begin(); !it.AtEnd(); {
			p, e := it.Entry()
			mapMust(t, e)
			if w, ok := oracle[p.Key]; !ok || w != p.Value || seen[p.Key] {
				t.Fatal(step, "iteration contents")
			}
			seen[p.Key] = true
			mapMust(t, it.Next())
		}
		if len(seen) != len(oracle) {
			t.Fatal(step, "iteration cardinality")
		}
		for probe := uint32(0); probe < 257; probe++ {
			g, ok := m.Find(probe)
			w, had := oracle[probe]
			if ok != had || (ok && g != w) {
				t.Fatal(step, probe, "lookup")
			}
		}
	}
}
func TestUintMapBudgetFailureAndPeakAccounting(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	probe, e := NewUintMap(UintMapConfig{Budget: b})
	mapMust(t, e)
	initial := b.Used()
	if initial <= 128 {
		t.Fatal("cold owner not charged", initial)
	}
	root := initial - 128
	probe.Close()
	if b.Used() != 0 {
		t.Fatal("constructor close accounting")
	}
	for _, limit := range []uint64{0, root, initial - 1} {
		budget := arena.NewBudget(limit)
		if m, e := NewUintMap(UintMapConfig{Budget: budget}); !errors.Is(e, arena.ErrBudget) || m != nil || budget.Used() != 0 {
			t.Fatal("failed construction leaks charge", limit, e, budget.Used())
		}
	}
	// The final doubled table would fit, but old+new allocation peak must fail.
	budget := arena.NewBudget(root + 128 + 256 - 1)
	m, e := NewUintMap(UintMapConfig{Budget: budget})
	mapMust(t, e)
	t.Cleanup(m.Close)
	for i := uint32(0); i < 7; i++ {
		mapMust(t, m.Insert(i, i+10))
	}
	ref, _ := m.FindCore(0)
	before := budget.Used()
	if e := m.Insert(0, 999); !errors.Is(e, arena.ErrBudget) {
		t.Fatal("growth peak budget", e)
	}
	if m.Capacity() != 8 || m.Size() != 7 || m.MustFind(0) != 10 || budget.Used() != before {
		t.Fatal("failed expansion changed source state")
	}
	if _, e := ref.Entry(); e != nil {
		t.Fatal("failed expansion invalidated refs")
	}
	fail := arena.NewBudget(1 << 20)
	f, e := NewUintMap(UintMapConfig{Budget: fail})
	mapMust(t, e)
	t.Cleanup(f.Close)
	for i := uint32(0); i < 7; i++ {
		mapMust(t, f.Insert(i, i))
	}
	fail.FailAfter(0)
	if _, e := f.Clone(); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("clone injection", e)
	}
	used := fail.Used()
	if e := f.Insert(7, 7); !errors.Is(e, arena.ErrInjectedFailure) || fail.Used() != used || f.Contains(7) {
		t.Fatal("failed growth injection", e)
	}
	fail.FailAfter(-1)
}
func TestUintMapCleanupFailurePreservesCompletedRemoval(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	m := clusteredMap(t, nil, budget)
	for i := uint32(0); i < 64; i++ {
		mapMust(t, m.Remove(i*1024))
	}
	before := budget.Used()
	budget.FailAfter(0)
	if e := m.Remove(64 * 1024); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("cleanup failure", e)
	}
	if m.Contains(64*1024) || !m.Contains(65*1024) || m.Size() != 1 || m.Deleted() != 65 || budget.Used() != before || !m.CheckInvariant() {
		t.Fatal("source removal should complete before failed cleanup")
	}
	// Source counts preexisting FREE slots, not deleted ones. This reset needs
	// no allocation and succeeds even while reservation failure is injected.
	mapMust(t, m.Reset())
	if m.Capacity() != 128 || !m.Empty() || m.Deleted() != 0 || budget.Used() != before {
		t.Fatal("reset must count tombstones as occupied for overhead")
	}
	budget.FailAfter(-1)
}
func TestUintMapSparseResetAndFinalizeFailureAtomic(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	m := clusteredMap(t, nil, budget)
	mapMust(t, m.Reset())
	mapMust(t, m.Insert(9, 90))
	ref, _ := m.FindCore(9)
	used := budget.Used()
	budget.FailAfter(0)
	if e := m.Reset(); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if m.Size() != 1 || m.Capacity() != 128 || m.MustFind(9) != 90 || budget.Used() != used {
		t.Fatal("failed sparse reset changed state")
	}
	if _, e := ref.Entry(); e != nil {
		t.Fatal("failed reset invalidated reference", e)
	}
	if e := m.Finalize(); !errors.Is(e, arena.ErrInjectedFailure) || !m.Contains(9) {
		t.Fatal("failed finalize", e)
	}
	budget.FailAfter(-1)
	mapMust(t, m.Finalize())
	if m.Capacity() != 64 || !m.Empty() {
		t.Fatal("finalize after failure")
	}
}
func TestUintMapSwapDifferentBudgetsAndRefs(t *testing.T) {
	aBudget, bBudget := arena.NewBudget(1<<20), arena.NewBudget(1<<20)
	a, e := NewUintMap(UintMapConfig{Budget: aBudget})
	mapMust(t, e)
	t.Cleanup(a.Close)
	b, e := NewUintMap(UintMapConfig{Budget: bBudget})
	mapMust(t, e)
	t.Cleanup(b.Close)
	for i := uint32(0); i < 8; i++ {
		mapMust(t, b.Insert(i, i+1))
	}
	mapMust(t, a.Insert(50, 500))
	aUsed, bUsed := aBudget.Used(), bBudget.Used()
	ref, _ := a.FindCore(50)
	aBudget.FailAfter(0)
	if e := a.Swap(b); !errors.Is(e, arena.ErrInjectedFailure) || a.Size() != 1 || b.Size() != 8 || aBudget.Used() != aUsed || bBudget.Used() != bUsed {
		t.Fatal("failed owner transfer", e)
	}
	aBudget.FailAfter(-1)
	bBudget.FailAfter(0)
	if e := a.Swap(b); !errors.Is(e, arena.ErrInjectedFailure) || aBudget.Used() != aUsed || bBudget.Used() != bUsed {
		t.Fatal("second reservation rollback", e)
	}
	bBudget.FailAfter(-1)
	mapMust(t, a.Swap(b))
	if a.Capacity() != 16 || b.Capacity() != 8 || aBudget.Used() != aUsed+128 || bBudget.Used() != bUsed-128 {
		t.Fatal("successful owner transfer")
	}
	if _, e := ref.Entry(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("swap must invalidate source-owner reference", e)
	}
	a.Close()
	b.Close()
	if aBudget.Used() != 0 || bBudget.Used() != 0 {
		t.Fatal("close transferred owner bytes")
	}
}
func TestUintMapWarmScalarOperationsNoAllocation(t *testing.T) {
	m := newUM(t)
	mapMust(t, m.Insert(1, 1))
	var err error
	if n := testing.AllocsPerRun(100, func() {
		if e := m.Insert(1, 2); e != nil {
			err = e
		}
		m.Find(1)
		m.Get(1, 0)
		ref, _ := m.FindCore(1)
		if e := ref.SetValue(3); e != nil {
			err = e
		}
		for it := m.Begin(); !it.AtEnd(); {
			it.Entry()
			if e := it.Next(); e != nil {
				err = e
			}
		}
	}); n != 0 {
		t.Fatal("warmed scalar lookup/ref/iteration allocations", n)
	}
	mapMust(t, err)
}

// src/test/hashtable.cpp tst_hashtable is empty on this Linux baseline. These
// additional portable tests translate its Windows-only membership assertions
// to the concrete u_map numeric-key/value variant; int_set remains unported.
func TestUintMapWindowsSourceMembershipAssertions(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 7))
	m := newUM(t)
	values := make([]uint32, 10000)
	for i := 1; i < 10000; i++ {
		v := uint32(r.IntN(5000))
		values[i] = v
		mapMust(t, m.Insert(v, v))
		if !m.Contains(v) {
			t.Fatal(i, "insert")
		}
	}
	for i := 1; i < 10000; i++ {
		if !m.Contains(values[i]) {
			t.Fatal(i, "first membership")
		}
	}
	for i := 1; i < 10000; i += 2 {
		mapMust(t, m.Erase(values[i]))
		if m.Contains(values[i]) {
			t.Fatal(i, "erase")
		}
	}
	for i := 1; i < 10000; i += 2 {
		mapMust(t, m.Insert(values[i], values[i]))
	}
	for i := 1; i < 10000; i++ {
		if !m.Contains(values[i]) {
			t.Fatal(i, "restored membership")
		}
	}
	for run := 0; run < 100; run++ {
		mapMust(t, m.Reset())
		oracle := make(map[uint32]bool)
		for i, n := 0, r.IntN(1000); i < n; i++ {
			v := uint32(r.IntN(1000))
			if r.IntN(3) == 2 {
				mapMust(t, m.Erase(v))
				delete(oracle, v)
				if m.Contains(v) {
					t.Fatal("erase assertion")
				}
			} else {
				mapMust(t, m.Insert(v, v))
				oracle[v] = true
				if !m.Contains(v) {
					t.Fatal("insert assertion")
				}
			}
		}
		for v := range oracle {
			if !m.Contains(v) {
				t.Fatal("oracle element missing")
			}
		}
		keys := mapKeys(t, m)
		if len(keys) != len(oracle) || m.Size() != uint32(len(keys)) {
			t.Fatal("iteration cardinality")
		}
		for _, v := range keys {
			if !m.Contains(v) {
				t.Fatal("iterator membership")
			}
		}
	}
	mapMust(t, m.Reset())
	for _, v := range []uint32{10, 20, 30} {
		mapMust(t, m.Insert(v, v))
	}
	mapMust(t, m.Erase(20))
	copy, e := m.Clone()
	mapMust(t, e)
	t.Cleanup(copy.Close)
	for _, x := range []*UintMap{m, copy} {
		if !x.Contains(10) || x.Contains(20) || !x.Contains(30) {
			t.Fatal("source copy assertions")
		}
	}
	if copy.Size() != 2 {
		t.Fatal("source copy size")
	}
}
func TestUintMapStrictDeletedVersusSizeThreshold(t *testing.T) {
	m := newUM(t)
	for i := uint32(0); i < 200; i++ {
		mapMust(t, m.Insert(i*1024, i))
	}
	if m.Capacity() != 512 {
		t.Fatal("expected source growth")
	}
	for i := uint32(0); i < 100; i++ {
		mapMust(t, m.Remove(i*1024))
	}
	if m.Deleted() != 100 || m.Size() != 100 {
		t.Fatal("deleted equal size must not clean")
	}
	mapMust(t, m.Remove(100*1024))
	if m.Deleted() != 0 || m.Size() != 99 || m.Capacity() != 512 {
		t.Fatal("deleted > size cleanup")
	}
	keys := mapKeys(t, m)
	for i, k := range keys {
		if k != uint32(i+101)*1024 {
			t.Fatal("same-cap rehash source bucket order", i, k)
		}
	}
}
func TestUintMapStrictResetFreeThreshold(t *testing.T) {
	m := newUM(t)
	for i := uint32(0); i < 14; i++ {
		mapMust(t, m.Insert(i, i))
	}
	if m.Capacity() != 32 {
		t.Fatal("expected source growth")
	}
	for i := uint32(14); i > 8; {
		i--
		mapMust(t, m.Remove(i))
	}
	if m.Size() != 8 || m.Deleted() != 0 {
		t.Fatal("tail removals must mark free")
	}
	mapMust(t, m.Reset())
	if m.Capacity() != 32 {
		t.Fatal("exact 75 percent free must not shrink")
	}
	mapMust(t, m.Insert(0, 0))
	mapMust(t, m.Reset())
	if m.Capacity() != 16 {
		t.Fatal("more than75 percent free must shrink")
	}
	mapMust(t, m.Insert(0, 0))
	mapMust(t, m.Reset())
	if m.Capacity() != 16 {
		t.Fatal("source reset stops at16")
	}
}
