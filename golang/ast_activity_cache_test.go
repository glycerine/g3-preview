// Source: src/ast/act_cache.cpp:56-225; src/ast/act_cache.h:23-53;
// src/util/chashtable.h:106-174,312-348,443-479.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"fmt"
	"testing"
)

func TestASTActivityCacheSourceOwnershipReplacementAndReset(t *testing.T) {
	m := astTestManager(t)
	limit := uint32(1)
	c, e := newASTActivityCache(m, &limit)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	if c.maxUnused != 1024 || c.table.slots != 8 || len(c.table.cells) != 10 || len(c.queue) != 0 {
		t.Fatal("source cache constructor", c.maxUnused, c.table.slots, len(c.table.cells))
	}
	size, e := c.Size()
	satOK(t, e)
	capacity, e := c.Capacity()
	satOK(t, e)
	empty, e := c.Empty()
	satOK(t, e)
	if size != 0 || capacity != 10 || !empty {
		t.Fatal("source cache queries", size, capacity, empty)
	}
	p, q, r := goalTestAtom(t, m, "cache-p"), goalTestAtom(t, m, "cache-q"), goalTestAtom(t, m, "cache-r")
	refs := func(h ASTHandle) uint32 { n, e := m.GetRefCount(h); satOK(t, e); return n }
	bp, bq, br := refs(p), refs(q), refs(r)
	satOK(t, c.Insert(p, q))
	if c.table.size != 1 || c.unused != 1 || refs(p) != bp+1 || refs(q) != bq+1 {
		t.Fatal("source new entry ownership")
	}
	satOK(t, c.Insert(p, q))
	if c.unused != 1 || len(c.queue) != 1 || refs(q) != bq+1 {
		t.Fatal("same value insertion changed ownership/activity")
	}
	satOK(t, c.Insert(p, r))
	if c.unused != 1 || refs(q) != bq || refs(r) != br+1 {
		t.Fatal("source replacement INC before DEC")
	}
	found, e := c.Find(p)
	satOK(t, e)
	if found != r || c.unused != 0 {
		t.Fatal("first find marks active", found, c.unused)
	}
	found, e = c.Find(p)
	satOK(t, e)
	if found != r || c.unused != 0 {
		t.Fatal("repeat find")
	}
	satOK(t, c.Reset())
	if c.table.size != 0 || c.unused != 0 || c.qhead != 0 || len(c.queue) != 0 || refs(p) != bp || refs(q) != bq || refs(r) != br {
		t.Fatal("source reset ownership/state")
	}
	// Null values are used by the original proof cache, but still have an entry.
	satOK(t, c.Insert(p, ASTHandle{}))
	found, e = c.Find(p)
	satOK(t, e)
	if !found.IsNull() || c.table.size != 1 || c.unused != 0 {
		t.Fatal("null tagged value", found, c.table.size, c.unused)
	}
	satOK(t, c.Cleanup())
	if c.table.size != 0 || len(c.table.cells) != 10 || c.queue != nil {
		t.Fatal("cleanup reinitializes map and finalizes queue")
	}
	copyOwner := *c
	if e = copyOwner.Insert(p, q); e != ErrASTHandle {
		t.Fatal("copied cache owner", e)
	}
	satOK(t, c.Close())
	if _, e = c.Find(p); e != ErrASTHandle {
		t.Fatal("closed cache owner", e)
	}
}

func TestASTActivityCacheOldestUnusedEvictionAndQueueCompression(t *testing.T) {
	m := astTestManager(t)
	limit := uint32(0)
	c, e := newASTActivityCache(m, &limit)
	satOK(t, e)
	defer func() { satOK(t, c.Close()) }()
	value := goalTestAtom(t, m, "cache-shared-value")
	var keys []ASTHandle
	for i := 0; i < 1024; i++ {
		k := goalTestAtom(t, m, fmt.Sprintf("cache-key-%d", i))
		keys = append(keys, k)
		satOK(t, c.Insert(k, value))
	}
	if c.unused != 1024 || c.table.size != 1024 {
		t.Fatal("source unused bound")
	}
	// The default overload captures the current canonical AST-table size.
	wantLimit := m.table.size
	large, e := newASTActivityCache(m, nil)
	satOK(t, e)
	if wantLimit <= 1024 || large.maxUnused != wantLimit {
		t.Fatal("source default current AST count", large.maxUnused, wantLimit)
	}
	satOK(t, large.Close())
	for i := 0; i < 900; i++ {
		got, e := c.Find(keys[i])
		satOK(t, e)
		if got != value {
			t.Fatal("activity mark", i)
		}
	}
	// Reach the limit again while leaving used entries in the old queue prefix.
	for i := 0; i < 900; i++ {
		k := goalTestAtom(t, m, fmt.Sprintf("cache-refill-%d", i))
		satOK(t, c.Insert(k, value))
	}
	fresh := goalTestAtom(t, m, "cache-trigger")
	satOK(t, c.Insert(fresh, value))
	if c.unused != 1024 || c.table.size != 1924 || c.qhead != 901 {
		t.Fatal("skip used keys then evict oldest unused", c.unused, c.table.size, c.qhead)
	}
	missing, e := c.Find(keys[900])
	satOK(t, e)
	if !missing.IsNull() {
		t.Fatal("oldest unused key survived")
	}
	for i := 0; i < 124; i++ {
		k := goalTestAtom(t, m, fmt.Sprintf("cache-evict-%d", i))
		satOK(t, c.Insert(k, value))
	}
	if c.qhead != 0 || len(c.queue) != 1024 {
		t.Fatal("source compression threshold", c.qhead, len(c.queue))
	}
	got, e := c.Find(keys[0])
	satOK(t, e)
	if got != value {
		t.Fatal("used entry was evicted")
	}
	if !c.CheckInvariant() {
		t.Fatal("source invariant method")
	}
}

func TestASTActivityCacheUsedReplacementPreservesSourceCounterQuirk(t *testing.T) {
	m := astTestManager(t)
	c, e := newASTActivityCache(m, nil)
	satOK(t, e)
	defer func() { satOK(t, c.Close()) }()
	p, q, r := goalTestAtom(t, m, "cache-used-p"), goalTestAtom(t, m, "cache-used-q"), goalTestAtom(t, m, "cache-used-r")
	if c.maxUnused != 1024 {
		t.Fatal("default max uses current AST table size and clamps")
	}
	satOK(t, c.Insert(p, q))
	_, e = c.Find(p)
	satOK(t, e)
	satOK(t, c.Insert(p, r))
	if c.unused != 0 || len(c.queue) != 1 {
		t.Fatal("source replacement does not increment unused or append queue")
	}
	got, e := c.Find(p)
	if astSourceDebug {
		if e != ErrASTDomain || c.unused != 0 {
			t.Fatal("source reached unused>0 assertion", e, c.unused)
		}
	} else {
		satOK(t, e)
		if got != r || c.unused != ^uint32(0) {
			t.Fatal("source counter underflow", got, c.unused)
		}
	}
}

func TestASTActivityCacheOriginalChainDisplacementRehashAndErase(t *testing.T) {
	m := astTestManager(t)
	c, e := newASTActivityCache(m, nil)
	satOK(t, e)
	defer func() { satOK(t, c.Close()) }()
	value := goalTestAtom(t, m, "chain-value")
	var keys []ASTHandle
	for i := 0; len(keys) < 3; i++ {
		k := goalTestAtom(t, m, fmt.Sprintf("chain-key-%d", i))
		hash, e := m.GetHash(k)
		satOK(t, e)
		if hash&15 == 0 {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		satOK(t, c.Insert(k, value))
	}
	if c.table.cells[0].key != keys[2] || c.table.cells[8].key != keys[0] || c.table.cells[9].key != keys[1] || c.table.next != 10 || c.table.used != 1 {
		t.Fatal("source head displacement / sequential cellar")
	}
	// Even a duplicate expands a full cellar before finding its existing entry.
	satOK(t, c.Insert(keys[1], value))
	if c.table.slots != 16 || len(c.table.cells) != 20 || c.table.cells[0].key != keys[0] || c.table.cells[16].key != keys[2] || c.table.cells[17].key != keys[1] || c.table.size != 3 || c.unused != 3 {
		t.Fatal("source copy order and grow-before-duplicate")
	}
	satOK(t, c.deleteUnused())
	if c.table.cells[0].key != keys[1] || c.table.free != 17 || c.table.size != 2 || c.qhead != 1 || c.unused != 2 {
		t.Fatal("source erase replaces head with next cellar cell")
	}
	collisions := c.table.collisions
	satOK(t, c.Reset())
	if c.table.collisions != collisions || c.table.slots != 8 {
		t.Fatal("source reset preserves lifetime collision count")
	}
}
