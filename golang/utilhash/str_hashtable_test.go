// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Concrete str_hashtable coverage; the active string->int map is separate.
package utilhash

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"strings"
	"testing"
)

func newSH(t *testing.T) (*StrHashtable, *arena.BytesRegion) {
	t.Helper()
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	mapMust(t, e)
	t.Cleanup(func() { mapMust(t, r.Close()) })
	h, e := NewStrHashtable(r, StrHashtableConfig{})
	mapMust(t, e)
	t.Cleanup(h.Close)
	return h, r
}
func strStore(t *testing.T, h *StrHashtable, s string, offset uint32) StrKey {
	t.Helper()
	r, e := h.region.Allocate(offset + uint32(len(s)) + 1)
	mapMust(t, e)
	for i, b := range []byte(s) {
		mapMust(t, h.region.Write(r, offset+uint32(i), b))
	}
	mapMust(t, h.region.Write(r, offset+uint32(len(s)), 0))
	k, e := h.CString(r, offset)
	mapMust(t, e)
	return k
}
func strContains(t *testing.T, h *StrHashtable, k StrKey) bool {
	t.Helper()
	ok, e := h.Contains(k)
	mapMust(t, e)
	return ok
}
func TestStrHashtableSourceHashCStringAndEquality(t *testing.T) {
	h, _ := newSH(t)
	// Seed17 constants from original hash.cpp, GCC little-endian/signed char.
	fixtures := []struct {
		s    string
		hash uint32
	}{{"", 3465257388}, {"foo", 1484660743}, {"boo", 2067241310}, {"abcdefghijk", 306724403}, {"abcdefghijkl", 1918021779}, {"abcdefghijklm", 58929046}, {"\x80\xff", 599191466}}
	for _, f := range fixtures {
		k := strStore(t, h, f.s, 8)
		got, e := h.Hash(k)
		mapMust(t, e)
		if got != f.hash {
			t.Fatalf("hash %q=%d want %d", f.s, got, f.hash)
		}
		mapMust(t, h.Insert(k))
		r, ok, e := h.FindCore(k)
		mapMust(t, e)
		if !ok {
			t.Fatal("missing")
		}
		cached, e := r.Hash()
		mapMust(t, e)
		if cached != f.hash {
			t.Fatal("cached hash")
		}
	}
	a := strStore(t, h, "foo\x00ignored", 3)
	if a.Length() != 3 || !strContains(t, h, a) {
		t.Fatal("C-string prefix")
	}
	mapMust(t, h.Insert(a))
	if h.Size() != uint32(len(fixtures)) {
		t.Fatal("NUL suffix participates")
	}
	stored, ok, e := h.Find(a)
	mapMust(t, e)
	if !ok || stored != a {
		t.Fatal("Insert must overwrite entire borrowed pointer")
	}
	b := strStore(t, h, "foo\x00other", 2)
	r, e := h.InsertIfNotThere(b)
	mapMust(t, e)
	stored, e = r.Key()
	mapMust(t, e)
	if stored != a {
		t.Fatal("insert-if-absent replaced pointer")
	}
	eq, e := h.Equal(a, b)
	mapMust(t, e)
	if !eq {
		t.Fatal("strcmp prefix equality")
	}
	if _, e := h.CString(arena.BytesRegionRef{}, 0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("null pointer accepted", e)
	}
	raw, e := h.region.Allocate(2)
	mapMust(t, e)
	mapMust(t, h.region.Write(raw, 0, 1))
	mapMust(t, h.region.Write(raw, 1, 2))
	if _, e := h.CString(raw, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("unterminated C string", e)
	}
	if reflect.TypeOf(strHashEntry{}).Size() != 32 {
		t.Fatal("entry must be six uint32 scalar words and uint64 origin")
	}
}
func TestStrHashtableBorrowedLifetimeAndReplacement(t *testing.T) {
	h, r := newSH(t)
	a := strStore(t, h, "foo", 0)
	mapMust(t, h.Insert(a))
	entry, ok, e := h.FindCore(a)
	mapMust(t, e)
	if !ok {
		t.Fatal("missing")
	}
	same := strStore(t, h, "foo", 8)
	mapMust(t, entry.SetKey(same))
	k, e := entry.Key()
	mapMust(t, e)
	if k != same {
		t.Fatal("owned replacement")
	}
	if e := entry.SetKey(strStore(t, h, "bar", 0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("unequal cached-hash replacement", e)
	}
	clone, e := h.Clone()
	mapMust(t, e)
	defer clone.Close()
	ck, ok, e := clone.Find(same)
	mapMust(t, e)
	if !ok || ck != same || clone.region != r {
		t.Fatal("Clone copied borrowed spelling")
	}
	mapMust(t, h.Remove(same))
	if !r.IsLive(a.Allocation()) || !r.IsLive(same.Allocation()) {
		t.Fatal("Remove freed bytes")
	}
	mapMust(t, h.Reset())
	h.Close()
	if !r.IsLive(same.Allocation()) {
		t.Fatal("Close freed bytes")
	}
	mark, e := r.PushScope()
	mapMust(t, e)
	temporary := strStore(t, clone, "scoped", 0)
	mapMust(t, clone.Insert(temporary))
	ref, _, e := clone.FindCore(temporary)
	mapMust(t, e)
	mapMust(t, r.Rollback(mark))
	if _, e := ref.Key(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale spelling returned", e)
	}
	if _, _, e := clone.FindCString("scoped"); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("lookup dereferenced rolled-back string", e)
	}
}
func collisionStrKeys(t *testing.T, h *StrHashtable, bucket, mask uint32, n int) []StrKey {
	t.Helper()
	var keys []StrKey
	for i := 0; len(keys) < n; i++ {
		s := fmt.Sprintf("collision%d", i)
		if String(s, 17)&mask == bucket {
			keys = append(keys, strStore(t, h, s, 0))
		}
	}
	return keys
}
func TestStrHashtableSourceProbeThresholdAndTombstones(t *testing.T) {
	h, _ := newSH(t)
	keys := collisionStrKeys(t, h, 7, 7, 8)
	for _, k := range keys[:6] {
		mapMust(t, h.Insert(k))
	}
	if h.Capacity() != 8 {
		t.Fatal("strict growth")
	}
	r, ok, e := h.FindCore(keys[0])
	mapMust(t, e)
	if !ok || r.slot != 7 {
		t.Fatal("wrapped first bucket")
	}
	// Before growth, wrapped physical chain has key1 at0 and key2 at1.
	mapMust(t, h.Remove(keys[1]))
	mapMust(t, h.Remove(keys[2]))
	if h.Deleted() != 2 {
		t.Fatal("tombstone state")
	}
	mapMust(t, h.Insert(keys[6]))
	r, ok, e = h.FindCore(keys[6])
	mapMust(t, e)
	if !ok || r.slot != 1 || h.Deleted() != 1 {
		t.Fatal("last tombstone reuse", r.slot)
	}
	mapMust(t, h.Insert(keys[7]))
	mapMust(t, h.Insert(strStore(t, h, "seventh", 0)))
	if h.Capacity() != 8 {
		t.Fatal("strict > threshold")
	}
	mapMust(t, h.Insert(keys[0]))
	if h.Capacity() != 16 {
		t.Fatal("growth before duplicate lookup")
	}
	if !h.CheckInvariant() {
		t.Fatal("invariant")
	}
	r, ok, e = h.FindCore(keys[0])
	mapMust(t, e)
	if !ok {
		t.Fatal("rehash key")
	}
	it, e := h.FindIterator(keys[0])
	mapMust(t, e)
	old, e := it.PostNext()
	mapMust(t, e)
	rr, e := old.Ref()
	mapMust(t, e)
	if rr.slot != r.slot {
		t.Fatal("postfix old bucket")
	}
	before := h.Size()
	inserted, _, e := h.InsertIfNotThereCore(keys[0])
	mapMust(t, e)
	if inserted || h.Size() != before {
		t.Fatal("duplicate core")
	}
	_, e = h.InsertIfNotThere2(keys[0])
	mapMust(t, e)
}
func TestStrHashtableOperationsAndOutput(t *testing.T) {
	h, _ := newSH(t)
	a := strStore(t, h, "foo", 0)
	b := strStore(t, h, "boo", 0)
	mapMust(t, h.Insert(a))
	mapMust(t, h.Insert(b))
	text, e := h.Dump()
	mapMust(t, e)
	if text != "[boo foo]" {
		t.Fatal("ascending bucket output", text)
	}
	out := a
	ok, e := h.FindInto(strStore(t, h, "missing", 0), &out)
	mapMust(t, e)
	if ok || out != a {
		t.Fatal("missing find changes output")
	}
	ok, e = h.FindInto(b, &out)
	mapMust(t, e)
	if !ok || out != b {
		t.Fatal("found data")
	}
	clone, e := h.Clone()
	mapMust(t, e)
	defer clone.Close()
	mapMust(t, clone.Erase(b))
	mapMust(t, h.Intersect(clone))
	if h.Size() != 1 || strContains(t, h, b) {
		t.Fatal("intersection")
	}
	mapMust(t, h.Union(clone))
	mapMust(t, clone.Assign(h))
	mapMust(t, h.Assign(h))
	mapMust(t, h.Union(h))
	mapMust(t, h.Intersect(h))
	mapMust(t, h.Reset())
	if !h.Empty() {
		t.Fatal("reset")
	}
	mapMust(t, h.Swap(clone))
	if h.Size() != 1 || clone.Size() != 0 {
		t.Fatal("swap")
	}
	mapMust(t, h.Finalize())
	if !h.Empty() || h.CollisionCount() != 0 {
		t.Fatal("finalize")
	}
}
func TestStrHashtableIndependentOracleAndCStringLookup(t *testing.T) {
	h, _ := newSH(t)
	oracle := make(map[string]bool)
	keys := make([]StrKey, 64)
	for i := range keys {
		keys[i] = strStore(t, h, fmt.Sprintf("key%d", i), uint32(i%9))
	}
	for step := 0; step < 900; step++ {
		index := (step*37 + 11) % 64
		s := fmt.Sprintf("key%d", index)
		if step%4 == 0 {
			mapMust(t, h.Remove(keys[index]))
			delete(oracle, s)
		} else {
			mapMust(t, h.Insert(keys[index]))
			oracle[s] = true
		}
		if h.Size() != uint32(len(oracle)) || !h.CheckInvariant() {
			t.Fatal("size/invariant", step)
		}
		for i, k := range keys {
			got, e := h.Contains(k)
			mapMust(t, e)
			if got != oracle[fmt.Sprintf("key%d", i)] {
				t.Fatal("oracle", step, i)
			}
		}
	}
	seen := make(map[string]bool)
	for it := h.Begin(); !it.AtEnd(); {
		r, e := it.Ref()
		mapMust(t, e)
		k, e := r.Key()
		mapMust(t, e)
		s, e := h.Text(k)
		mapMust(t, e)
		if !oracle[s] || seen[s] {
			t.Fatal("iteration", s)
		}
		seen[s] = true
		mapMust(t, it.Next())
	}
	if len(seen) != len(oracle) {
		t.Fatal("iteration incomplete")
	}
	for s := range oracle {
		r, ok, e := h.PrepareFindCString(s + "\x00suffix")
		mapMust(t, e)
		if !ok {
			t.Fatal("prepared lookup")
		}
		k, e := r.Key()
		mapMust(t, e)
		text, e := h.Text(k)
		mapMust(t, e)
		if text != s {
			t.Fatal("lookup spelling")
		}
	}
	if _, ok, e := h.FindCString("absent"); e != nil || ok {
		t.Fatal("absent CString", e)
	}
	if allocations := testing.AllocsPerRun(50, func() {
		if _, _, e := h.FindCString("key0"); e != nil {
			panic(e)
		}
		if e := h.Insert(keys[0]); e != nil {
			panic(e)
		}
	}); allocations != 0 {
		t.Fatal("warmed lookup allocates", allocations)
	}
}
func TestStrHashtableBudgetsSwapAndStaleViews(t *testing.T) {
	for fail := int64(0); fail < 3; fail++ {
		r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
		mapMust(t, e)
		budget := arena.NewBudget(1 << 20)
		budget.FailAfter(fail)
		h, e := NewStrHashtable(r, StrHashtableConfig{Budget: budget})
		if e == nil {
			h.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if budget.Used() != 0 {
			t.Fatal("constructor leak")
		}
		mapMust(t, r.Close())
	}
	for which := 0; which < 2; which++ {
		for fail := int64(0); fail < 2; fail++ {
			a, ra := newSH(t)
			b, rb := newSH(t)
			a.Close()
			b.Close()
			ba, bb := arena.NewBudget(1<<20), arena.NewBudget(1<<20)
			a, e := NewStrHashtable(ra, StrHashtableConfig{Budget: ba})
			mapMust(t, e)
			b, e = NewStrHashtable(rb, StrHashtableConfig{Budget: bb})
			mapMust(t, e)
			ka, kb := strStore(t, a, "foo", 0), strStore(t, b, "boo", 0)
			mapMust(t, a.Insert(ka))
			mapMust(t, b.Insert(kb))
			ref, _, e := a.FindCore(ka)
			mapMust(t, e)
			ua, ub := ba.Used(), bb.Used()
			if which == 0 {
				ba.FailAfter(fail)
			} else {
				bb.FailAfter(fail)
			}
			e = a.Swap(b)
			ba.FailAfter(-1)
			bb.FailAfter(-1)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) || a.region != ra || b.region != rb || ba.Used() != ua || bb.Used() != ub {
					t.Fatal("swap failure")
				}
				if _, e := ref.Key(); e != nil {
					t.Fatal("failed swap invalidated borrow")
				}
			} else {
				if a.region != rb || b.region != ra || !strContains(t, a, kb) {
					t.Fatal("swap owner binding")
				}
				if _, e := ref.Key(); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("swap old view")
				}
			}
			a.Close()
			b.Close()
			if ba.Used() != 0 || bb.Used() != 0 {
				t.Fatal("swap leak")
			}
		}
	}
	h, r := newSH(t)
	k := strStore(t, h, "x", 0)
	mapMust(t, h.Insert(k))
	it := h.Begin()
	entry, e := it.Ref()
	mapMust(t, e)
	mapMust(t, h.Remove(k))
	if _, e := entry.Key(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("old ref")
	}
	if e := it.Next(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("old iterator")
	}
	if !r.IsLive(k.Allocation()) {
		t.Fatal("borrowed allocation freed")
	}
	mapMust(t, r.Close())
	if _, _, e := h.FindCString("x"); e != nil {
		t.Fatal("empty table needs no byte reads", e)
	}
	if _, e := h.CString(k.Allocation(), 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed region", e)
	}
}

// Source dump is byte-oriented; embedded NUL terminates each borrowed spelling.
func TestStrHashtableDumpCStringBytes(t *testing.T) {
	h, _ := newSH(t)
	k := strStore(t, h, "\x80\xff\x00tail", 0)
	mapMust(t, h.Insert(k))
	s, e := h.Dump()
	mapMust(t, e)
	if !strings.EqualFold(s, "[\x80\xff]") {
		t.Fatal(s)
	}
}

func TestStrHashtableKeyOwnerAndStats(t *testing.T) {
	a, region := newSH(t)
	b, _ := newSH(t)
	ka, kb := strStore(t, a, "same", 0), strStore(t, b, "same", 0)
	if ka == kb {
		t.Fatal("numerically matching refs alias foreign region")
	}
	mapMust(t, a.Insert(ka))
	if _, e := a.Contains(kb); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("foreign key accepted", e)
	}
	shared, e := NewStrHashtable(region, StrHashtableConfig{})
	mapMust(t, e)
	defer shared.Close()
	same, e := shared.CString(ka.Allocation(), ka.Offset())
	mapMust(t, e)
	if same != ka {
		t.Fatal("same region identity differs by table")
	}
	mapMust(t, shared.Insert(same))
	mapMust(t, a.Assign(shared))
	mapMust(t, a.Union(shared))
	mapMust(t, a.Intersect(shared))
	mapMust(t, b.Insert(kb))
	mapMust(t, a.Union(b))
	mapMust(t, a.Intersect(b))
	mapMust(t, a.Assign(b))
	budget := arena.NewBudget(1 << 20)
	charged, e := NewStrHashtable(region, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	s := charged.Stats()
	if s.OwnerBytes == 0 || s.TableBytes != 8*strHashEntryBytes || budget.Used() != s.OwnerBytes+s.TableBytes || s.Capacity != 8 || s.Size != 0 || s.Deleted != 0 {
		t.Fatal("table graph accounting", s, budget.Used())
	}
	charged.Close()
	s = charged.Stats()
	if s.OwnerBytes != 0 || s.TableBytes != 0 || budget.Used() != 0 {
		t.Fatal("closed table charges", s)
	}
}

func TestStrHashtableMultiOriginQueriesAndAdoption(t *testing.T) {
	a, ra := newSH(t)
	b, rb := newSH(t)
	ka, kb := strStore(t, a, "equal", 0), strStore(t, b, "equal", 0)
	mapMust(t, a.Insert(ka))
	before := a.Stats()
	ref, ok, e := a.FindFrom(rb, kb)
	mapMust(t, e)
	if !ok || a.Stats() != before {
		t.Fatal("foreign query adopted metadata")
	}
	key, e := ref.Key()
	mapMust(t, e)
	if key != ka {
		t.Fatal("foreign strcmp")
	}
	inserted, ref, e := a.InsertIfNotThereCoreFrom(rb, kb)
	mapMust(t, e)
	if inserted || a.Stats() != before {
		t.Fatal("duplicate adopted incoming pointer")
	}
	key, e = ref.Key()
	mapMust(t, e)
	if key != ka {
		t.Fatal("duplicate pointer replacement")
	}
	mapMust(t, a.InsertFrom(rb, kb))
	after := a.Stats()
	if after.DirectoryBytes == 0 {
		t.Fatal("retained foreign pointer not accounted")
	}
	key, e = ref.Key()
	mapMust(t, e)
	if key != kb {
		t.Fatal("equal-key pointer did not replace")
	}
	if ok, e := a.ContainsFrom(ra, ka); e != nil || !ok {
		t.Fatal("query opposite owner", e)
	}
	if _, _, e := a.FindFrom(ra, kb); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("foreign key owner mismatch", e)
	}
	borrowed, e := BorrowCString(rb, kb.Allocation(), kb.Offset())
	mapMust(t, e)
	if borrowed != kb {
		t.Fatal("independent borrow identity")
	}
	if allocs := testing.AllocsPerRun(50, func() {
		if _, _, e := a.FindFrom(ra, ka); e != nil {
			panic(e)
		}
		if _, e := a.ContainsFrom(rb, kb); e != nil {
			panic(e)
		}
	}); allocs != 0 {
		t.Fatal("foreign queries allocate", allocs)
	}
	mapMust(t, a.Reset())
	if a.Stats().DirectoryBytes != after.DirectoryBytes || !rb.IsLive(kb.Allocation()) {
		t.Fatal("reset discarded retained binding or bytes")
	}
}

func TestStrHashtableMultiOriginOperatorsCloneAndLifetime(t *testing.T) {
	a, ra := newSH(t)
	b, rb := newSH(t)
	c, rc := newSH(t)
	ka := strStore(t, a, "shared", 0)
	kb := strStore(t, b, "shared", 8)
	onlyB := strStore(t, b, "onlyb", 0)
	onlyC := strStore(t, c, "onlyc", 0)
	mapMust(t, a.Insert(ka))
	mapMust(t, b.Insert(kb))
	mapMust(t, b.Insert(onlyB))
	mapMust(t, b.InsertFrom(rc, onlyC))
	mapMust(t, a.Union(b))
	if a.Size() != 3 {
		t.Fatal("multi-owner union")
	}
	got, ok, e := a.Find(ka)
	mapMust(t, e)
	if !ok || got != kb {
		t.Fatal("source pointer overwrite")
	}
	clone, e := a.Clone()
	mapMust(t, e)
	defer clone.Close()
	if clone.Size() != 3 || clone.Stats().DirectoryBytes == 0 {
		t.Fatal("clone binding graph")
	}
	for _, k := range []StrKey{kb, onlyB, onlyC} {
		got, ok, e := clone.Find(k)
		mapMust(t, e)
		if !ok || got != k {
			t.Fatal("clone copied spelling")
		}
	}
	mapMust(t, c.Insert(onlyC))
	mapMust(t, a.Intersect(c))
	if a.Size() != 1 || !strContains(t, a, onlyC) {
		t.Fatal("foreign intersection")
	}
	mapMust(t, a.Assign(b))
	if a.Size() != 3 {
		t.Fatal("foreign assignment")
	}
	mapMust(t, a.Assign(a))
	mapMust(t, a.Union(a))
	mapMust(t, a.Intersect(a))
	mapMust(t, a.Swap(c))
	if a.region != rc || c.region != ra || c.Size() != 3 || a.Size() != 1 {
		t.Fatal("swap primary and bindings")
	}
	for _, k := range []StrKey{kb, onlyB, onlyC} {
		if !strContains(t, c, k) {
			t.Fatal("swap lost owner binding")
		}
	}
	mark, e := rb.PushScope()
	mapMust(t, e)
	temporary := strStore(t, b, "temporary", 0)
	mapMust(t, c.InsertFrom(rb, temporary))
	view, _, e := c.FindCore(temporary)
	mapMust(t, e)
	mapMust(t, rb.Rollback(mark))
	if _, e := view.Key(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("borrowed rollback not detected", e)
	}
	if !rc.IsLive(onlyC.Allocation()) || !ra.IsLive(ka.Allocation()) {
		t.Fatal("operators released region bytes")
	}
}

func TestStrHashtableMultiOriginFailureOrderAndAccounting(t *testing.T) {
	_, primary := newSH(t)
	peer, foreign := newSH(t)
	budget := arena.NewBudget(1 << 20)
	h, e := NewStrHashtable(primary, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	defer h.Close()
	local, e := BorrowCString(primary, strStore(t, h, "same", 0).Allocation(), 0)
	mapMust(t, e)
	incoming := strStore(t, peer, "same", 0)
	mapMust(t, h.Insert(local))
	ref, _, e := h.FindCore(local)
	mapMust(t, e)
	before := h.Stats()
	used := budget.Used()
	budget.FailAfter(0)
	e = h.InsertFrom(foreign, incoming)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || h.Stats() != before || budget.Used() != used {
		t.Fatal("adoption failure mutated graph", e)
	}
	key, e := ref.Key()
	mapMust(t, e)
	if key != local {
		t.Fatal("adoption failure pointer overwrite")
	}
	h.epoch = ^uint64(0)
	newKey := strStore(t, peer, "new", 0)
	e = h.InsertFrom(foreign, newKey)
	if !errors.Is(e, arena.ErrBounds) || h.Stats() != before || budget.Used() != used {
		t.Fatal("exhaustion adopted unused owner", e)
	}
	h.epoch = ref.epoch
	mapMust(t, h.InsertFrom(foreign, incoming))
	stats := h.Stats()
	if stats.DirectoryBytes == 0 || budget.Used() != stats.OwnerBytes+stats.TableBytes+stats.DirectoryBytes {
		t.Fatal("directory graph charge", stats, budget.Used())
	}
	// Source ascending buckets determine which partial Union insertion survives
	// the second cold directory growth failure; no bulk pre-adoption is allowed.
	source, second := newSH(t)
	thirdTable, third := newSH(t)
	s1 := strStore(t, source, "boo", 0)
	s2 := strStore(t, thirdTable, "foo", 0)
	mapMust(t, source.Insert(s1))
	mapMust(t, source.InsertFrom(third, s2))
	var order []StrKey
	for it := source.Begin(); !it.AtEnd(); {
		k, e := it.Key()
		mapMust(t, e)
		order = append(order, k)
		mapMust(t, it.Next())
	}
	destination, e := NewStrHashtable(primary, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	budget.FailAfter(1)
	e = destination.Union(source)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || destination.Size() != 1 {
		t.Fatal("source partial union prefix", e, destination.Stats())
	}
	if got, ok, e := destination.Find(order[0]); e != nil || !ok || got != order[0] {
		t.Fatal("wrong completed source bucket", got, e)
	}
	if _, ok, e := destination.FindFrom(third, order[1]); e != nil || ok {
		t.Fatal("failed source bucket inserted", e)
	}
	destination.Close()
	// Clone failures at header/table/directory prefix unwind all destination
	// charges while keeping source bindings and borrowed allocations live.
	mapMust(t, h.InsertFrom(second, s1))
	mapMust(t, h.InsertFrom(third, s2))
	used = budget.Used()
	for fail := int64(0); fail < 5; fail++ {
		budget.FailAfter(fail)
		clone, e := h.Clone()
		budget.FailAfter(-1)
		if e == nil {
			clone.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if budget.Used() != used {
			t.Fatal("failed clone binding leak", fail, budget.Used(), used)
		}
	}
}

func TestStrHashtableMultiOriginSwapGraphFailures(t *testing.T) {
	for failureOwner := 0; failureOwner < 2; failureOwner++ {
		for fail := int64(0); fail < 2; fail++ {
			_, ra := newSH(t)
			_, rb := newSH(t)
			third, rc := newSH(t)
			ba, bb := arena.NewBudget(1<<20), arena.NewBudget(1<<20)
			a, e := NewStrHashtable(ra, StrHashtableConfig{Budget: ba})
			mapMust(t, e)
			b, e := NewStrHashtable(rb, StrHashtableConfig{Budget: bb, InitialCapacity: 16})
			mapMust(t, e)
			ka, kb, kc := strStore(t, a, "a", 0), strStore(t, b, "b", 0), strStore(t, third, "c", 0)
			mapMust(t, a.Insert(ka))
			mapMust(t, b.Insert(kb))
			mapMust(t, a.InsertFrom(rc, kc))
			beforeA, beforeB := a.Stats(), b.Stats()
			usedA, usedB := ba.Used(), bb.Used()
			if failureOwner == 0 {
				ba.FailAfter(fail)
			} else {
				bb.FailAfter(fail)
			}
			e = a.Swap(b)
			ba.FailAfter(-1)
			bb.FailAfter(-1)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) || a.Stats() != beforeA || b.Stats() != beforeB || ba.Used() != usedA || bb.Used() != usedB {
					t.Fatal("partial table+directory transfer", e)
				}
			} else {
				if a.region != rb || b.region != ra || b.Stats().DirectoryBytes != beforeA.DirectoryBytes || a.Stats().DirectoryBytes != beforeB.DirectoryBytes || !strContains(t, b, kc) {
					t.Fatal("swap graph binding")
				}
				sa, sb := a.Stats(), b.Stats()
				if ba.Used() != sa.OwnerBytes+sa.TableBytes+sa.DirectoryBytes || bb.Used() != sb.OwnerBytes+sb.TableBytes+sb.DirectoryBytes {
					t.Fatal("wrong transferred directory charges")
				}
			}
			a.Close()
			b.Close()
			if ba.Used() != 0 || bb.Used() != 0 || !rc.IsLive(kc.Allocation()) {
				t.Fatal("swap close leaked/released spelling")
			}
		}
	}
}

func TestStrHashtableMultiOriginQueryMethodsAndCollisionOrder(t *testing.T) {
	h, _ := newSH(t)
	b, rb := newSH(t)
	c, rc := newSH(t)
	query, rq := newSH(t)
	keys := collisionStrKeys(t, h, 7, 7, 4)
	texts := make([]string, 4)
	for i, k := range keys {
		var e error
		texts[i], e = h.Text(k)
		mapMust(t, e)
	}
	kb, kc, kq := strStore(t, b, texts[1], 0), strStore(t, c, texts[2], 0), strStore(t, query, texts[0]+"\x00ignored", 3)
	mapMust(t, h.Insert(keys[0]))
	mapMust(t, h.InsertFrom(rb, kb))
	mapMust(t, h.InsertFrom(rc, kc))
	want := []StrKey{kb, kc, keys[0]}
	i := 0
	for it := h.Begin(); !it.AtEnd(); {
		k, e := it.Key()
		mapMust(t, e)
		if k != want[i] {
			t.Fatal("mixed-owner source wrapped buckets")
		}
		i++
		mapMust(t, it.Next())
	}
	if i != 3 {
		t.Fatal("bucket count")
	}
	before := h.Stats()
	out := StrKey{}
	ok, e := h.FindIntoFrom(rq, kq, &out)
	mapMust(t, e)
	if !ok || out != keys[0] || h.Stats() != before {
		t.Fatal("foreign output find adopted")
	}
	it, e := h.FindIteratorFrom(rq, kq)
	mapMust(t, e)
	ref, e := it.Ref()
	mapMust(t, e)
	if ref.slot != 7 {
		t.Fatal("foreign finditerator bucket")
	}
	mapMust(t, h.RemoveFrom(rq, kq))
	if h.Stats().DirectoryBytes != before.DirectoryBytes || h.Deleted() != 1 {
		t.Fatal("foreign Remove adopted owner")
	}
	// Entry mutation borrows a new owner but preserves its cached hash and epoch.
	replacement := strStore(t, query, texts[1], 0)
	stored, _, e := h.FindCore(kb)
	mapMust(t, e)
	epoch := h.epoch
	mapMust(t, stored.SetKeyFrom(rq, replacement))
	if h.epoch != epoch {
		t.Fatal("metadata adoption invalidated borrow")
	}
	got, e := stored.Key()
	mapMust(t, e)
	if got != replacement {
		t.Fatal("foreign SetKey pointer")
	}
	added := strStore(t, query, texts[3], 0)
	mapMust(t, h.InsertFrom(rq, added))
	addedRef, _, e := h.FindFrom(rq, added)
	mapMust(t, e)
	if addedRef.slot != 7 {
		t.Fatal("mixed-owner tombstone placement")
	}
	mapMust(t, h.EraseFrom(rq, added))
	if _, e := h.InsertIfNotThereFrom(rq, replacement); e != nil {
		t.Fatal(e)
	}
	if _, e := h.InsertIfNotThere2From(rq, replacement); e != nil {
		t.Fatal(e)
	}
	mapMust(t, rq.Close())
	if _, _, e := h.FindFrom(rq, replacement); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed query owner guard", e)
	}
}

func TestStrHashtableMultiOriginIndependentOracleAndDirectoryPeak(t *testing.T) {
	h, _ := newSH(t)
	p1, _ := newSH(t)
	p2, _ := newSH(t)
	pools := []*StrHashtable{h, p1, p2}
	keys := make([][]StrKey, 3)
	for owner, pool := range pools {
		keys[owner] = make([]StrKey, 12)
		for i := range keys[owner] {
			keys[owner][i] = strStore(t, pool, fmt.Sprintf("oracle%d", i), uint32(owner*3))
		}
	}
	oracle := make(map[int]StrKey)
	for step := 0; step < 450; step++ {
		index := (step*7 + 1) % 12
		owner := (step*5 + 1) % 3
		key := keys[owner][index]
		if step%4 == 0 {
			mapMust(t, h.RemoveFrom(pools[owner].region, key))
			delete(oracle, index)
		} else {
			mapMust(t, h.InsertFrom(pools[owner].region, key))
			oracle[index] = key
		}
		if h.Size() != uint32(len(oracle)) || !h.CheckInvariant() {
			t.Fatal("multi-origin oracle size", step)
		}
		for i := 0; i < 12; i++ {
			queryOwner := (step + i) % 3
			ref, found, e := h.FindFrom(pools[queryOwner].region, keys[queryOwner][i])
			mapMust(t, e)
			want, present := oracle[i]
			if found != present {
				t.Fatal("foreign membership", step, i)
			}
			if found {
				got, e := ref.Key()
				mapMust(t, e)
				if got != want {
					t.Fatal("independent latest pointer", step, i)
				}
			}
		}
	}
	seen := make(map[StrKey]bool)
	for it := h.Begin(); !it.AtEnd(); {
		k, e := it.Key()
		mapMust(t, e)
		seen[k] = true
		mapMust(t, it.Next())
	}
	for _, key := range oracle {
		if !seen[key] {
			t.Fatal("multi-origin iteration missing pointer")
		}
	}
	if len(seen) != len(oracle) {
		t.Fatal("multi-origin iteration duplicate")
	}
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	oneDirectory := uint64(reflect.TypeOf(strOwner{}).Size())
	limit := root + 8*strHashEntryBytes + 3*oneDirectory - 1
	budget := arena.NewBudget(limit)
	limited, e := NewStrHashtable(h.region, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	mapMust(t, limited.InsertFrom(p1.region, keys[1][0]))
	before := limited.Stats()
	used := budget.Used()
	e = limited.InsertFrom(p2.region, keys[2][1])
	if !errors.Is(e, arena.ErrBudget) || limited.Stats() != before || budget.Used() != used {
		t.Fatal("directory old+new peak not reserved", e)
	}
	limited.Close()
	if budget.Used() != 0 {
		t.Fatal("directory peak close leak")
	}
}

func TestStrHashtableCleanupResetAndFailurePeaks(t *testing.T) {
	for _, oom := range []bool{false, true} {
		r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
		mapMust(t, e)
		calls := 0
		h, e := NewStrHashtable(r, StrHashtableConfig{OutOfMemory: func() bool { calls++; return oom }})
		mapMust(t, e)
		keys := collisionStrKeys(t, h, 0, 255, 130)
		for _, k := range keys {
			mapMust(t, h.Insert(k))
		}
		if h.Capacity() != 256 {
			t.Fatal("chain capacity")
		}
		for _, k := range keys[:66] {
			mapMust(t, h.Remove(k))
		}
		if calls != 1 || h.Size() != 64 || !h.CheckInvariant() {
			t.Fatal("strict cleanup", calls, h.Stats())
		}
		if oom && h.Deleted() != 66 || !oom && h.Deleted() != 0 {
			t.Fatal("OOM cleanup suppression", h.Deleted())
		}
		for _, k := range keys {
			if !r.IsLive(k.Allocation()) {
				t.Fatal("cleanup released spelling")
			}
		}
		h.Close()
		mapMust(t, r.Close())
	}
	h, r := newSH(t)
	mapMust(t, h.rehash(128))
	key := strStore(t, h, "one", 0)
	mapMust(t, h.Insert(key))
	mapMust(t, h.Reset())
	if h.Capacity() != 64 || !r.IsLive(key.Allocation()) {
		t.Fatal("reset shrink/borrow")
	}
	mapMust(t, h.Reset())
	if h.Capacity() != 64 {
		t.Fatal("emptyclean reset shrank")
	}
	mapMust(t, h.rehash(256))
	mapMust(t, h.Insert(key))
	mapMust(t, h.Finalize())
	if h.Capacity() != 64 || !h.Empty() || !r.IsLive(key.Allocation()) {
		t.Fatal("finalize borrow")
	}
	budget := arena.NewBudget(1 << 20)
	charged, e := NewStrHashtable(r, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	keys := collisionStrKeys(t, charged, 0, 7, 8)
	for _, k := range keys[:7] {
		mapMust(t, charged.Insert(k))
	}
	before := charged.Stats()
	borrow, _, e := charged.FindCore(keys[0])
	mapMust(t, e)
	used := budget.Used()
	budget.FailAfter(0)
	e = charged.Insert(keys[7])
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || charged.Stats() != before || budget.Used() != used {
		t.Fatal("expansion failure changed table", e)
	}
	if _, e := borrow.Key(); e != nil {
		t.Fatal("failedgrow stale borrow", e)
	}
	budget.FailAfter(0)
	_, _, e = charged.PrepareFindCString("missing")
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || charged.Stats() != before {
		t.Fatal("prepared growth failure")
	}
	budget.FailAfter(0)
	clone, e := charged.Clone()
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || clone != nil || budget.Used() != used {
		t.Fatal("clone budget failure")
	}
	charged.Close()
	if budget.Used() != 0 {
		t.Fatal("budget leak")
	}
	// Source Reset destroys before allocation; this adaptation reserves the
	// replacement table first and retains contents on predictable failure.
	charged, e = NewStrHashtable(r, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	mapMust(t, charged.rehash(128))
	mapMust(t, charged.Insert(key))
	before = charged.Stats()
	budget.FailAfter(0)
	e = charged.Reset()
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || charged.Stats() != before || !strContains(t, charged, key) {
		t.Fatal("atomic reset failure")
	}
	budget.FailAfter(0)
	e = charged.Finalize()
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || charged.Stats() != before {
		t.Fatal("atomic finalize failure")
	}
	charged.Close()
	// A finite limit proving the new table is charged while the old is live.
	root := uint64(reflect.TypeOf(StrHashtable{}).Size())
	tight := arena.NewBudget(root + 24*strHashEntryBytes - 1)
	charged, e = NewStrHashtable(r, StrHashtableConfig{Budget: tight})
	mapMust(t, e)
	for _, k := range keys[:7] {
		mapMust(t, charged.Insert(k))
	}
	if e := charged.Insert(keys[7]); !errors.Is(e, arena.ErrBudget) {
		t.Fatal("old+new peak not bounded", e)
	}
	charged.Close()
	if tight.Used() != 0 {
		t.Fatal("peak close leak")
	}
}

func TestStrHashtableSourceInitialCapacity(t *testing.T) {
	_, region := newSH(t)
	for _, capacity := range []uint32{1, 2, 4, 8, 16} {
		h, e := NewStrHashtable(region, StrHashtableConfig{InitialCapacity: capacity})
		mapMust(t, e)
		if h.Capacity() != capacity {
			t.Fatal("source constructor capacity")
		}
		key := strStore(t, h, "foo", 0)
		mapMust(t, h.Insert(key))
		clone, e := h.Clone()
		mapMust(t, e)
		if clone.Capacity() != capacity || !strContains(t, clone, key) {
			t.Fatal("smallcap clone")
		}
		clone.Close()
		h.Close()
	}
	for _, capacity := range []uint32{3, 5, 1 << 31} {
		if _, e := NewStrHashtable(region, StrHashtableConfig{InitialCapacity: capacity}); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("invalid/overflow capacity", capacity, e)
		}
	}
}

func TestStrHashtablePartialRemovalAndStrictReset(t *testing.T) {
	_, region := newSH(t)
	budget := arena.NewBudget(1 << 20)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	keys := collisionStrKeys(t, h, 0, 255, 127)
	for _, k := range keys {
		mapMust(t, h.Insert(k))
	}
	for _, k := range keys[:64] {
		mapMust(t, h.Remove(k))
	}
	if h.Deleted() != 64 || h.Size() != 63 {
		t.Fatal("cleanup at equality64")
	}
	budget.FailAfter(0)
	e = h.Remove(keys[64])
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || h.Size() != 62 || h.Deleted() != 65 || strContains(t, h, keys[64]) || !h.CheckInvariant() {
		t.Fatal("removal must precede failedcleanup", e, h.Stats())
	}
	for _, entry := range h.table {
		if entry.state == strHashDeleted && entry.key != (StrKey{}) {
			t.Fatal("ptr tombstone retained borrowed key rather than sentinel")
		}
	}
	h.Close()
	if budget.Used() != 0 {
		t.Fatal("close budget")
	}
	for _, n := range []int{8, 7} {
		table, e := NewStrHashtable(region, StrHashtableConfig{InitialCapacity: 32})
		mapMust(t, e)
		for i := 0; i < n; i++ {
			mapMust(t, table.Insert(strStore(t, table, fmt.Sprintf("reset%d", i), 0)))
		}
		mapMust(t, table.Reset())
		want := uint32(32)
		if n == 7 {
			want = 16
		}
		if table.Capacity() != want {
			t.Fatal("reset strictpreexistingFREE threshold", n, table.Capacity())
		}
		table.Close()
	}
}

func TestStrHashtableEpochExhaustion(t *testing.T) {
	for _, action := range []string{"insert", "grow", "remove", "reset", "finalize", "swap", "swap-other"} {
		_, region := newSH(t)
		budget := arena.NewBudget(1 << 20)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: budget})
		mapMust(t, e)
		old := strStore(t, h, "old", 0)
		next := strStore(t, h, "new", 0)
		mapMust(t, h.Insert(old))
		if action == "grow" {
			for i := 0; i < 6; i++ {
				mapMust(t, h.Insert(strStore(t, h, fmt.Sprintf("epoch%d", i), 0)))
			}
		}
		if action == "finalize" {
			mapMust(t, h.rehash(128))
		}
		other, e := NewStrHashtable(region, StrHashtableConfig{Budget: budget})
		mapMust(t, e)
		h.epoch = ^uint64(0)
		if action == "swap-other" {
			h.epoch = 1
			other.epoch = ^uint64(0)
		}
		ref, _, e := h.FindCore(old)
		mapMust(t, e)
		before := h.Stats()
		used := budget.Used()
		table := make([]strHashEntry, len(h.table))
		copy(table, h.table)
		switch action {
		case "insert":
			e = h.Insert(next)
		case "grow":
			_, _, e = h.PrepareFindCString("old")
		case "remove":
			e = h.Remove(old)
		case "reset":
			e = h.Reset()
		case "finalize":
			e = h.Finalize()
		case "swap", "swap-other":
			e = h.Swap(other)
		}
		if !errors.Is(e, arena.ErrBounds) || h.Stats() != before || budget.Used() != used {
			t.Fatalf("%s failed atomically: %v %+v %+v", action, e, h.Stats(), before)
		}
		for i, entry := range h.table {
			if entry != table[i] {
				t.Fatal("mutated exhausted table", action)
			}
		}
		if _, e := ref.Key(); e != nil {
			t.Fatal("failed operation invalidated existing borrow", action, e)
		}
		// Close must invalidate and release even when no epoch increment remains.
		h.Close()
		other.Close()
		if budget.Used() != 0 {
			t.Fatal("exhausted close leaked budget")
		}
		if _, e := ref.Key(); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("close did not invalidate ref", e)
		}
	}
	h, _ := newSH(t)
	a := strStore(t, h, "same", 0)
	b := strStore(t, h, "same", 8)
	mapMust(t, h.Insert(a))
	h.epoch = ^uint64(0)
	mapMust(t, h.Insert(b))
	key, ok, e := h.Find(a)
	mapMust(t, e)
	if !ok || key != b {
		t.Fatal("equal-key overwrite rejected at maxepoch")
	}
	prepared, found, e := h.PrepareFindCString("same")
	mapMust(t, e)
	if !found {
		t.Fatal("prepared hit rejected at maxepoch")
	}
	if got, e := prepared.Key(); e != nil || got != b {
		t.Fatal("prepared hit borrow", got, e)
	}
	beforePreparation := h.Stats()
	if _, found, e := h.PrepareFindCString("not-interned"); !errors.Is(e, arena.ErrBounds) || found || h.Stats() != beforePreparation {
		t.Fatal("prepared miss must refuse before caller allocation", found, e)
	}
	mapMust(t, h.Remove(strStore(t, h, "missing", 0)))
	empty, _ := newSH(t)
	empty.epoch = ^uint64(0)
	mapMust(t, empty.Reset())
	mapMust(t, empty.Finalize())
	mapMust(t, empty.Swap(empty))
	// The removal consumes the last epoch; attempted cleanup then fails before
	// allocation, retaining source removal-before-cleanup partial semantics.
	_, region := newSH(t)
	budget := arena.NewBudget(1 << 20)
	chain, e := NewStrHashtable(region, StrHashtableConfig{Budget: budget})
	mapMust(t, e)
	keys := collisionStrKeys(t, chain, 0, 255, 127)
	for _, k := range keys {
		mapMust(t, chain.Insert(k))
	}
	for _, k := range keys[:64] {
		mapMust(t, chain.Remove(k))
	}
	chain.epoch = ^uint64(0) - 1
	used := budget.Used()
	e = chain.Remove(keys[64])
	if !errors.Is(e, arena.ErrBounds) || chain.epoch != ^uint64(0) || chain.Size() != 62 || chain.Deleted() != 65 || budget.Used() != used || strContains(t, chain, keys[64]) {
		t.Fatal("last epoch cleanup partial semantics", e, chain.Stats())
	}
	chain.Close()
	if budget.Used() != 0 {
		t.Fatal("chain close leaked")
	}
}
