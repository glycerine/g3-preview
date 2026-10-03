// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package utilhash

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"go/types"
	"reflect"
	"testing"
)

func newSIM(t *testing.T, c StrInt32MapConfig) (*StrInt32Map, *arena.BytesRegion) {
	t.Helper()
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	mapMust(t, e)
	t.Cleanup(func() { mapMust(t, r.Close()) })
	h, e := NewStrInt32Map(r, c)
	mapMust(t, e)
	t.Cleanup(h.Close)
	return h, r
}
func simKey(t *testing.T, r *arena.BytesRegion, s string) StrKey {
	t.Helper()
	ref, e := r.Allocate(uint32(len(s)) + 1)
	mapMust(t, e)
	for i, b := range []byte(s) {
		mapMust(t, r.Write(ref, uint32(i), b))
	}
	mapMust(t, r.Write(ref, uint32(len(s)), 0))
	k, e := BorrowCString(r, ref, 0)
	mapMust(t, e)
	return k
}
func simContains(t *testing.T, h *StrInt32Map, k StrKey) bool {
	t.Helper()
	b, e := h.Contains(k)
	mapMust(t, e)
	return b
}
func TestStrInt32MapSourceMap14Assertions(t *testing.T) {
	h, r := newSIM(t, StrInt32MapConfig{})
	foo, boo := simKey(t, r, "foo"), simKey(t, r, "boo")
	mapMust(t, h.Insert(foo, 35))
	// src/test/map.cpp tst1, all 14 ENSURE statements in source order.
	if !simContains(t, h, foo) {
		t.Fatal("assertion 1")
	}
	it, e := h.FindIterator(foo)
	mapMust(t, e)
	if it.Equal(h.End()) {
		t.Fatal("assertion 2")
	}
	v, e := it.Value()
	mapMust(t, e)
	if v != 35 {
		t.Fatal("assertion 3")
	}
	if h.Size() != 1 {
		t.Fatal("assertion 4")
	}
	mapMust(t, h.Insert(boo, 32))
	if !simContains(t, h, foo) {
		t.Fatal("assertion 5")
	}
	it, e = h.FindIterator(foo)
	mapMust(t, e)
	if it.Equal(h.End()) {
		t.Fatal("assertion 6")
	}
	v, e = it.Value()
	mapMust(t, e)
	if v != 35 {
		t.Fatal("assertion 7")
	}
	if !simContains(t, h, boo) {
		t.Fatal("assertion 8")
	}
	it, e = h.FindIterator(boo)
	mapMust(t, e)
	if it.Equal(h.End()) {
		t.Fatal("assertion 9")
	}
	v, e = it.Value()
	mapMust(t, e)
	if v != 32 {
		t.Fatal("assertion 10")
	}
	if h.Size() != 2 {
		t.Fatal("assertion 11")
	}
	mapMust(t, h.Remove(boo))
	if h.Size() != 1 {
		t.Fatal("assertion 12")
	}
	if simContains(t, h, boo) {
		t.Fatal("assertion 13")
	}
	if !simContains(t, h, foo) {
		t.Fatal("assertion 14")
	}
}
func TestStrInt32MapForeignBorrowAndMutableValues(t *testing.T) {
	h, r := newSIM(t, StrInt32MapConfig{})
	_, foreign := newSIM(t, StrInt32MapConfig{})
	a, b := simKey(t, r, "foo"), simKey(t, foreign, "foo\x00suffix")
	mapMust(t, h.Insert(a, -2147483648))
	old, ok, e := h.FindCore(a)
	mapMust(t, e)
	if !ok {
		t.Fatal("missing")
	}
	made, ref, e := h.InsertIfNotThereCoreFrom(foreign, b, 99)
	mapMust(t, e)
	if made || h.Stats().DirectoryBytes != 0 {
		t.Fatal("duplicate must not adopt")
	}
	v, e := ref.Value()
	mapMust(t, e)
	if v != -2147483648 {
		t.Fatal("absent insertion overwrote")
	}
	mapMust(t, h.InsertFrom(foreign, b, 2147483647))
	k, e := old.Key()
	mapMust(t, e)
	v, e = old.Value()
	mapMust(t, e)
	if k != b || v != 2147483647 || h.Stats().DirectoryBytes == 0 {
		t.Fatal("complete overwrite")
	}
	mapMust(t, old.SetValue(-7))
	v, found, e := h.Find(a)
	mapMust(t, e)
	if !found || v != -7 {
		t.Fatal("mutable find")
	}
	out := int32(123)
	found, e = h.FindInto(simKey(t, r, "missing"), &out)
	mapMust(t, e)
	if found || out != 123 {
		t.Fatal("miss changed output")
	}
	v, e = h.Get(a, 5)
	mapMust(t, e)
	if v != -7 {
		t.Fatal("get")
	}
	v, e = h.Get(simKey(t, r, "absent"), 5)
	mapMust(t, e)
	if v != 5 {
		t.Fatal("default")
	}
	mapMust(t, h.Remove(a))
	retained := false
	for _, entry := range h.table {
		if entry.key == b && entry.value == -7 && entry.state != strHashUsed {
			retained = true
		}
	}
	if !retained {
		t.Fatal("default entry removal discarded payload")
	}
	mapMust(t, h.Reset())
	if h.Stats().DirectoryBytes == 0 {
		t.Fatal("borrow binding retained through reset")
	}
}
func TestStrInt32MapIndependentOracleAndBuckets(t *testing.T) {
	h, r := newSIM(t, StrInt32MapConfig{})
	_, s := newSIM(t, StrInt32MapConfig{})
	keys := [2][31]StrKey{}
	for i := 0; i < 31; i++ {
		keys[0][i] = simKey(t, r, fmt.Sprintf("word%d", i))
		keys[1][i] = simKey(t, s, fmt.Sprintf("word%d", i))
	}
	present := [31]bool{}
	values := [31]int32{}
	stored := [31]StrKey{}
	for step := 0; step < 700; step++ {
		i := (step*17 + step/9) % 31
		pool := (step / 3) % 2
		key := keys[pool][i]
		owner := r
		if pool == 1 {
			owner = s
		}
		value := int32(step*97 - 20000)
		switch step % 5 {
		case 0, 1:
			mapMust(t, h.InsertFrom(owner, key, value))
			present[i] = true
			values[i] = value
			stored[i] = key
		case 2:
			made, _, e := h.InsertIfNotThereCoreFrom(owner, key, value)
			mapMust(t, e)
			if made == present[i] {
				t.Fatal("insert absent")
			}
			if made {
				present[i] = true
				values[i] = value
				stored[i] = key
			}
		case 3:
			mapMust(t, h.RemoveFrom(owner, key))
			present[i] = false
		case 4:
			ref, ok, e := h.FindFrom(owner, key)
			mapMust(t, e)
			if ok != present[i] {
				t.Fatal("query")
			}
			if ok {
				v, e := ref.Value()
				mapMust(t, e)
				if v != values[i] {
					t.Fatal("value")
				}
			}
		}
		var count uint32
		for j := 0; j < 31; j++ {
			v, ok, e := h.FindFromValue(r, keys[0][j])
			mapMust(t, e)
			if ok != present[j] || ok && v != values[j] {
				t.Fatalf("oracle step %d key%d", step, j)
			}
			if ok {
				count++
			}
		}
		if h.Size() != count || !h.CheckInvariant() {
			t.Fatal("shape")
		}
		seen := [31]bool{}
		previous := -1
		for it := h.Begin(); !it.AtEnd(); {
			ref, e := it.Ref()
			mapMust(t, e)
			if int(ref.slot) <= previous {
				t.Fatal("bucket order")
			}
			previous = int(ref.slot)
			k, e := it.Key()
			mapMust(t, e)
			match := -1
			for j := 0; j < 31; j++ {
				if present[j] && k == stored[j] {
					match = j
					break
				}
			}
			if match < 0 || seen[match] {
				t.Fatal("iteration")
			}
			seen[match] = true
			mapMust(t, it.Next())
		}
	}
	if uint64(reflect.TypeOf(strInt32Entry{}).Size()) != strInt32EntryBytes {
		t.Fatal("physical byte accounting")
	}
}
func TestStrInt32MapCopyOperatorsAndBudget(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	h, r := newSIM(t, StrInt32MapConfig{Budget: budget})
	other, s := newSIM(t, StrInt32MapConfig{})
	a, b, c := simKey(t, r, "foo"), simKey(t, s, "foo"), simKey(t, s, "boo")
	mapMust(t, h.Insert(a, 1))
	mapMust(t, other.Insert(b, 2))
	mapMust(t, other.Insert(c, 3))
	mapMust(t, h.Union(other))
	if h.Size() != 2 {
		t.Fatal("union")
	}
	v, ok, e := h.Find(a)
	mapMust(t, e)
	if !ok || v != 2 {
		t.Fatal("union overwrite")
	}
	used := budget.Used()
	for prefix := int64(0); prefix < 3; prefix++ {
		budget.FailAfter(prefix)
		copy, e := h.Clone()
		if copy != nil {
			copy.Close()
		}
		if !errors.Is(e, arena.ErrInjectedFailure) || budget.Used() != used {
			t.Fatalf("clone prefix%d %v", prefix, e)
		}
	}
	budget.FailAfter(-1)
	copy, e := h.Clone()
	mapMust(t, e)
	mapMust(t, copy.Insert(a, 8))
	v, _, e = h.Find(a)
	mapMust(t, e)
	if v != 2 {
		t.Fatal("clone aliases values")
	}
	copy.Close()
	mapMust(t, other.Remove(c))
	mapMust(t, h.Intersect(other))
	if h.Size() != 1 {
		t.Fatal("intersection")
	}
	mapMust(t, h.Assign(other))
	v, _, e = h.Find(a)
	mapMust(t, e)
	if v != 2 {
		t.Fatal("assign")
	}
	mapMust(t, h.Assign(h))
	mapMust(t, h.Union(h))
	mapMust(t, h.Intersect(h))
	mapMust(t, h.Swap(h))
	h.Close()
	if budget.Used() != 0 {
		t.Fatal("close leaked")
	}
}

// Permanent original hash.cpp seed17 fixtures (little endian, signed char).
func TestStrInt32MapHashCachedCopiesAndDefaultPayload(t *testing.T) {
	h, r := newSIM(t, StrInt32MapConfig{})
	fixtures := []struct {
		s    string
		hash uint32
	}{{"", 3465257388}, {"foo", 1484660743}, {"boo", 2067241310}, {"abcdefghijklm", 58929046}, {"\x80\xff", 599191466}}
	for i, f := range fixtures {
		k := simKey(t, r, f.s)
		hash, e := h.Hash(k)
		mapMust(t, e)
		if hash != f.hash {
			t.Fatal("permanent hash fixture")
		}
		mapMust(t, h.Insert(k, int32(i)))
	}
	a := simKey(t, r, "foo")
	ref, ok, e := h.FindCore(a)
	mapMust(t, e)
	if !ok {
		t.Fatal("missing")
	}
	cached, e := ref.Hash()
	mapMust(t, e)
	if cached != 1484660743 {
		t.Fatal("cached")
	}
	equal := simKey(t, r, "foo\x00ignored")
	mapMust(t, ref.SetKey(equal))
	if e := ref.SetKey(simKey(t, r, "different")); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("mutated hash/equality")
	}
	old := ref
	mapMust(t, h.rehash(16))
	if _, e = old.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reallocation borrow")
	}
	mapMust(t, h.Reset())
	for _, entry := range h.table {
		if entry.state != strHashFree {
			t.Fatal("reset states")
		}
	}
	retained := false
	for _, entry := range h.table {
		if entry.key == equal && entry.value == 1 {
			retained = true
		}
	}
	if !retained {
		t.Fatal("reset discarded default_entry payload")
	}
	copy, e := h.Clone()
	mapMust(t, e)
	defer copy.Close()
	for _, entry := range copy.table {
		if entry.key != (StrKey{}) || entry.value != 0 {
			t.Fatal("clone copied FREE payload")
		}
	}
}
func simCollisionKeys(t *testing.T, r *arena.BytesRegion, start, mask uint32, n int) []StrKey {
	t.Helper()
	keys := make([]StrKey, 0, n)
	// Fixture selection uses baseline hash, while table layout expectations below
	// are explicit finite source probe sequences, not a second table implementation.
	for i := 0; len(keys) < n; i++ {
		s := fmt.Sprintf("collide%d", i)
		if String(s, 17)&mask == start {
			keys = append(keys, simKey(t, r, s))
		}
	}
	return keys
}
func TestStrInt32MapWrappedTombstonesGrowthAndCleanup(t *testing.T) {
	h, r := newSIM(t, StrInt32MapConfig{})
	keys := simCollisionKeys(t, r, 7, 7, 8)
	for i, k := range keys[:6] {
		mapMust(t, h.Insert(k, int32(i)))
	}
	if h.Capacity() != 8 {
		t.Fatal("strict growth")
	}
	mapMust(t, h.Remove(keys[1]))
	mapMust(t, h.Remove(keys[2]))
	mapMust(t, h.Insert(keys[6], 66))
	ref, ok, e := h.FindCore(keys[6])
	mapMust(t, e)
	if !ok || ref.slot != 1 || h.Deleted() != 1 {
		t.Fatal("last wrapped tombstone")
	}
	mapMust(t, h.Insert(keys[7], 77))
	mapMust(t, h.Insert(simKey(t, r, "seventh"), 7))
	if h.Capacity() != 8 {
		t.Fatal("threshold equality")
	}
	mapMust(t, h.Insert(keys[0], 90))
	if h.Capacity() != 16 {
		t.Fatal("growth before overwrite")
	}
	v, _, e := h.Find(keys[0])
	mapMust(t, e)
	if v != 90 {
		t.Fatal("overwrite after growth")
	}
	for _, oom := range []bool{false, true} {
		calls := 0
		chain, rr := newSIM(t, StrInt32MapConfig{OutOfMemory: func() bool { calls++; return oom }})
		kk := simCollisionKeys(t, rr, 0, 255, 130)
		for i, k := range kk {
			mapMust(t, chain.Insert(k, int32(i)))
		}
		for _, k := range kk[:66] {
			mapMust(t, chain.Erase(k))
		}
		if calls != 1 || chain.Size() != 64 || !chain.CheckInvariant() {
			t.Fatal("cleanup threshold")
		}
		if oom && chain.Deleted() != 66 || !oom && chain.Deleted() != 0 {
			t.Fatal("OOM cleanup")
		}
	}
	budget := arena.NewBudget(1 << 20)
	chain, rr := newSIM(t, StrInt32MapConfig{Budget: budget})
	kk := simCollisionKeys(t, rr, 0, 255, 127)
	for i, k := range kk {
		mapMust(t, chain.Insert(k, int32(i)))
	}
	for _, k := range kk[:64] {
		mapMust(t, chain.Remove(k))
	}
	budget.FailAfter(0)
	e = chain.Remove(kk[64])
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || chain.Size() != 62 || chain.Deleted() != 65 || simContains(t, chain, kk[64]) {
		t.Fatal("removal before failed cleanup")
	}
	for _, entry := range chain.table {
		if entry.state == strHashDeleted && entry.key == (StrKey{}) {
			t.Fatal("deleted default entry lost payload")
		}
	}
}
func TestStrInt32MapResetFinalizeThresholdsAndEpoch(t *testing.T) {
	for _, count := range []int{7, 8} {
		h, r := newSIM(t, StrInt32MapConfig{InitialCapacity: 32})
		for i := 0; i < count; i++ {
			mapMust(t, h.Insert(simKey(t, r, fmt.Sprintf("reset%d", i)), int32(i)))
		}
		mapMust(t, h.Reset())
		want := uint32(32)
		if count == 7 {
			want = 16
		}
		if h.Capacity() != want {
			t.Fatal("strict prior FREE shrink")
		}
		mapMust(t, h.Reset())
		if h.Capacity() != want {
			t.Fatal("clean reset shrink")
		}
	}
	h, r := newSIM(t, StrInt32MapConfig{InitialCapacity: 128})
	key := simKey(t, r, "foo")
	mapMust(t, h.Insert(key, 1))
	mapMust(t, h.Finalize())
	if h.Capacity() != 64 || !h.Empty() || !r.IsLive(key.ref) {
		t.Fatal("finalize")
	}
	mapMust(t, h.Insert(key, 2))
	ref, _, e := h.FindCore(key)
	mapMust(t, e)
	it := h.Begin()
	mapMust(t, h.Insert(key, 3))
	v, e := ref.Value()
	mapMust(t, e)
	if v != 3 {
		t.Fatal("overwrite invalidates")
	}
	old, e := it.PostNext()
	mapMust(t, e)
	v, e = old.Value()
	mapMust(t, e)
	if v != 3 {
		t.Fatal("post increment")
	}
	h.epoch = ^uint64(0)
	before := h.Stats()
	table := append([]strInt32Entry(nil), h.table...)
	for _, action := range []func() error{func() error { return h.Insert(simKey(t, r, "new"), 4) }, func() error { return h.Remove(key) }, h.Reset, h.Finalize} {
		if e := action(); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("exhaustion", e)
		}
		if h.Stats() != before || !reflect.DeepEqual(table, h.table) {
			t.Fatal("exhaustion mutation")
		}
	}
	_, found, e := h.PrepareFindCString("new")
	if !errors.Is(e, arena.ErrBounds) || found {
		t.Fatal("prep miss")
	}
	_, found, e = h.PrepareFindCString("foo")
	mapMust(t, e)
	if !found {
		t.Fatal("prep hit")
	}
	mapMust(t, h.Insert(key, 8))
	h.Close()
}
func TestStrInt32MapMultiOwnerBudgetsSwapLifetimeAndWarm(t *testing.T) {
	ab, bb := arena.NewBudget(1<<20), arena.NewBudget(1<<20)
	a, r := newSIM(t, StrInt32MapConfig{Budget: ab})
	b, s := newSIM(t, StrInt32MapConfig{Budget: bb, InitialCapacity: 16})
	_, third := newSIM(t, StrInt32MapConfig{})
	ka, kb, kc := simKey(t, r, "a"), simKey(t, s, "b"), simKey(t, third, "c")
	mapMust(t, a.Insert(ka, 1))
	mapMust(t, a.InsertFrom(third, kc, 3))
	mapMust(t, b.Insert(kb, 2))
	ai := a.Begin()
	au, bu := ab.Used(), bb.Used()
	bb.FailAfter(0)
	e := a.Swap(b)
	bb.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || ab.Used() != au || bb.Used() != bu || !simContains(t, a, ka) {
		t.Fatal("second graph reserve rollback")
	}
	if _, e = ai.Value(); e != nil {
		t.Fatal("failed swap invalidates")
	}
	mapMust(t, a.Swap(b))
	if _, e = ai.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("swap stale iterator")
	}
	if !simContains(t, a, kb) || !simContains(t, b, ka) || !simContains(t, b, kc) {
		t.Fatal("swap bindings")
	}
	stats := a.Stats()
	if ab.Used() != stats.OwnerBytes+stats.TableBytes+stats.DirectoryBytes {
		t.Fatal("whole graph charges")
	}
	q := simKey(t, third, "b")
	before := a.Stats()
	ab.FailAfter(0)
	if got := testing.AllocsPerRun(100, func() {
		ref, ok, e := a.FindFrom(third, q)
		if e != nil || !ok {
			panic("query")
		}
		if e = ref.SetValue(4); e != nil {
			panic(e)
		}
	}); got != 0 {
		t.Fatal("warm foreign query allocations", got)
	}
	ab.FailAfter(-1)
	if a.Stats() != before {
		t.Fatal("query adopted")
	}
	_, e = third.PushScope()
	mapMust(t, e)
	expired := simKey(t, third, "scoped")
	mapMust(t, a.InsertFrom(third, expired, 9))
	ref, ok, e := a.FindFrom(third, expired)
	mapMust(t, e)
	if !ok {
		t.Fatal("scoped miss")
	}
	mapMust(t, third.PopScope())
	if _, e = ref.Key(); e == nil {
		t.Fatal("expired spelling")
	}
	mapMust(t, a.RemoveFrom(r, ka)) // absent query need not inspect stale spelling with a different hash
	a.Close()
	b.Close()
	if ab.Used() != 0 || bb.Used() != 0 {
		t.Fatal("close graph leak")
	}
}

func TestStrInt32MapConstructorGrowthAndUnionFailurePrefixes(t *testing.T) {
	_, primary := newSIM(t, StrInt32MapConfig{})
	budget := arena.NewBudget(1 << 20)
	for prefix := int64(0); prefix < 2; prefix++ {
		budget.FailAfter(prefix)
		h, e := NewStrInt32Map(primary, StrInt32MapConfig{Budget: budget})
		if h != nil {
			h.Close()
		}
		if !errors.Is(e, arena.ErrInjectedFailure) || budget.Used() != 0 {
			t.Fatal("constructor unwind", prefix, e)
		}
	}
	budget.FailAfter(-1)
	h, e := NewStrInt32Map(primary, StrInt32MapConfig{Budget: budget})
	mapMust(t, e)
	defer h.Close()
	keys := make([]StrKey, 8)
	for i := range keys {
		keys[i] = simKey(t, primary, fmt.Sprintf("growth%d", i))
	}
	for i, k := range keys[:7] {
		mapMust(t, h.Insert(k, int32(i)))
	}
	before := h.Stats()
	used := budget.Used()
	old, _, e := h.FindCore(keys[0])
	mapMust(t, e)
	budget.FailAfter(0)
	e = h.Insert(keys[0], 99)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || budget.Used() != used {
		t.Fatal("growth before duplicate failure")
	}
	v, e := old.Value()
	mapMust(t, e)
	if v != 0 {
		t.Fatal("failed duplicate mutated")
	}
	// Source ascending bucket order places boo at6, foo at7 in an empty cap8.
	destination, e := NewStrInt32Map(primary, StrInt32MapConfig{Budget: budget})
	mapMust(t, e)
	defer destination.Close()
	source, _ := newSIM(t, StrInt32MapConfig{})
	_, ownerA := newSIM(t, StrInt32MapConfig{})
	_, ownerB := newSIM(t, StrInt32MapConfig{})
	boo, foo := simKey(t, ownerA, "boo"), simKey(t, ownerB, "foo")
	mapMust(t, source.InsertFrom(ownerA, boo, 32))
	mapMust(t, source.InsertFrom(ownerB, foo, 35))
	budget.FailAfter(1)
	e = destination.Union(source)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || destination.Size() != 1 {
		t.Fatal("union ordered prefix", e)
	}
	v, ok, e := destination.FindFromValue(ownerA, boo)
	mapMust(t, e)
	if !ok || v != 32 {
		t.Fatal("first bucket prefix")
	}
	_, ok, e = destination.FindFrom(ownerB, foo)
	mapMust(t, e)
	if ok {
		t.Fatal("failed second published")
	}
	// Directory old+new physical peak is larger than its retained replacement.
	root := uint64(reflect.TypeOf(StrInt32Map{}).Size())
	dir := uint64(reflect.TypeOf(strOwner{}).Size())
	tight := arena.NewBudget(root + 8*strInt32EntryBytes + 3*dir - 1)
	limited, e := NewStrInt32Map(primary, StrInt32MapConfig{Budget: tight})
	mapMust(t, e)
	mapMust(t, limited.InsertFrom(ownerA, boo, 32))
	prior := limited.Stats()
	e = limited.InsertFrom(ownerB, foo, 35)
	if !errors.Is(e, arena.ErrBudget) || limited.Stats() != prior {
		t.Fatal("directory old/new peak", e)
	}
	limited.Close()
	if tight.Used() != 0 {
		t.Fatal("directory charge leak")
	}
}

// Build a go/types scalar shape from the actual reflected field tree, so the
// architecture audit does not manually repeat the record declaration.
func simScalarType(t *testing.T, r reflect.Type) types.Type {
	t.Helper()
	switch r.Kind() {
	case reflect.Uint32:
		return types.Typ[types.Uint32]
	case reflect.Uint64:
		return types.Typ[types.Uint64]
	case reflect.Int32:
		return types.Typ[types.Int32]
	case reflect.Struct:
		fields := make([]*types.Var, r.NumField())
		for i := range fields {
			f := r.Field(i)
			fields[i] = types.NewVar(0, nil, f.Name, simScalarType(t, f.Type))
		}
		return types.NewStruct(fields, nil)
	default:
		t.Fatalf("non-scalar hot shape: %v", r)
		return nil
	}
}
func TestStrInt32MapConcreteArchitectureLayoutAndBudget(t *testing.T) {
	shape := simScalarType(t, reflect.TypeOf(strInt32Entry{}))
	for _, arch := range []string{"386", "amd64", "arm", "arm64", "wasm"} {
		sizes := types.SizesFor("gc", arch)
		if sizes == nil {
			t.Fatal("missing arch sizes", arch)
		}
		if got := sizes.Sizeof(shape); got != 40 {
			t.Fatalf("%s actual scalar record size %d, charged 40", arch, got)
		}
	}
	budget := arena.NewBudget(1 << 20)
	h, _ := newSIM(t, StrInt32MapConfig{Budget: budget, InitialCapacity: 16})
	s := h.Stats()
	actual := uint64(reflect.TypeOf(strInt32Entry{}).Size()) * 16
	if s.TableBytes != actual || budget.Used() != s.OwnerBytes+actual {
		t.Fatal("physical array accounting")
	}
	h.Close()
	if budget.Used() != 0 {
		t.Fatal("physical graph close")
	}
}
