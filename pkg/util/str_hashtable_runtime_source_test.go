// Source: src/util/hashtable.h:94-110,142-147,222-238,243-272,287-322;
// src/util/debug.cpp:44-68; src/util/str_hashtable.h:24-31.
package util

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableRuntimeActualWrapperTableAndLazyFieldInitialization(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	if h.table != nil {
		t.Fatal("Runtime route retained shadow hot slice")
	}
	roots, e := h.RuntimeRoots()
	if e != nil {
		t.Fatal(e)
	}
	if roots.Object == (Block{}) || roots.Table == (Block{}) || roots.Object == roots.Table {
		t.Fatal("actual source wrapper and eager table roots", roots)
	}
	if r.State().GlobalCount != 2 {
		t.Fatal("actual alloc(str_hashtable) then alloc_vect", r.State())
	}
	if h.Capacity() != 8 || h.Size() != 0 || h.Deleted() != 0 {
		t.Fatal(h.Stats())
	}
	// Only the pointer sentinel is initialized by the ascending entry ctor.
	if _, e := r.ReadUint32(roots.Table, 0); e != ErrRuntimeUninitialized {
		t.Fatal("cached hash initialized before reached set_hash", e)
	}
	for i := uint64(0); i < 8; i++ {
		v, e := r.ReadUint32(roots.Table, i*32+4)
		if e != nil || v != 0 {
			t.Fatal(i, v, e)
		}
	}
	if e := h.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 {
		t.Fatal("actual source table then wrapper destructor", r.Stats())
	}
	if e := h.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e := region.Close(); e != nil {
		t.Fatal(e)
	}
	if e := r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("named raw/cold graph did not refund", b.Used())
	}
}

func TestStrHashtableRuntimeProbeGrowthAndIndependentContents(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	for i, s := range []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight"} {
		k := strStore(t, h, s, 0)
		mapMust(t, h.Insert(k))
		if h.Size() != uint32(i+1) {
			t.Fatal("source size publication", i, h.Size())
		}
		for _, q := range []string{"zero", "one", "absent"} {
			_, found, e := h.FindCString(q)
			mapMust(t, e)
			want := q == "zero" || q == "one" && i >= 1
			if found != want {
				t.Fatal(i, q, found, want)
			}
		}
	}
	if h.Capacity() != 16 {
		t.Fatal("source expansion before eighth insert", h.Capacity())
	}
	if h.table != nil {
		t.Fatal("shadow slice")
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeResetDestructiveRefusalPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 64})
	mapMust(t, e)
	mapMust(t, h.Insert(strStore(t, h, "held", 0)))
	before, e := h.RuntimeRoots()
	mapMust(t, e)
	b.FailAfter(0)
	if e = h.Reset(); e == nil {
		t.Fatal("expected reached replacement refusal")
	}
	after, e := h.RuntimeRoots()
	mapMust(t, e)
	if after.Table != (Block{}) || h.Capacity() != 32 || h.Size() != 1 {
		t.Fatal("source delete, half, allocate before counter reset", after, h.Capacity(), h.Size())
	}
	if _, e = r.Size(before.Table); e == nil {
		t.Fatal("old table retained transactionally")
	}
	if r.Stats().LiveBlocks != 1 {
		t.Fatal("only actual wrapper survives", r.Stats())
	}
	b.FailAfter(-1)
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func runtimeStringGraph(t *testing.T, b *arena.Budget, r *Runtime, region *arena.BytesRegion, owners ...*StrHashtable) {
	t.Helper()
	roots := map[Block]bool{}
	cold := uint64(0)
	for _, h := range owners {
		if h == nil {
			continue
		}
		rr, e := h.RuntimeRoots()
		mapMust(t, e)
		cold += h.rootBytes + h.directoryBytes + rr.AcquisitionBytes
		for _, token := range append([]Block{rr.Object, rr.Table}, rr.Acquisitions...) {
			if token == (Block{}) {
				continue
			}
			if roots[token] {
				t.Fatal("duplicate physical owner", token)
			}
			roots[token] = true
			if _, e = r.Size(token); e != nil {
				t.Fatal("named actual token", token, e)
			}
		}
	}
	rs := r.Stats()
	if uint32(len(roots)) != rs.LiveBlocks {
		t.Fatal("actual root bijection", roots, rs)
	}
	s := region.Stats()
	d := s.Descriptors
	regionBytes := s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if want := rs.TotalBytes + regionBytes + cold; b.Used() != want {
		t.Fatal("exact cold/physical graph", b.Used(), want, rs, regionBytes, cold)
	}
}
func TestStrHashtableRuntimeEveryConstructorReservationAndCopyPrefixes(t *testing.T) {
	for _, copying := range []bool{false, true} {
		success := -1
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			var src *StrHashtable
			if copying {
				src, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
				mapMust(t, e)
				mapMust(t, src.Insert(strStore(t, src, "copy", 0)))
			}
			b.FailAfter(ordinal)
			var h *StrHashtable
			if copying {
				h, e = src.Clone()
			} else {
				h, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
			}
			// Inspect the complete immediate named graph while refusal is still active.
			runtimeStringGraph(t, b, r, region, src, h)
			if e == nil {
				success = int(ordinal)
				if h.Size() != map[bool]uint32{false: 0, true: 1}[copying] {
					t.Fatal("copy source counters")
				}
			}
			b.FailAfter(-1)
			if h != nil {
				mapMust(t, h.CloseChecked())
			}
			if src != nil {
				mapMust(t, src.CloseChecked())
			}
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("prefix refund", ordinal, b.Used())
			}
			if e == nil {
				break
			}
		}
		if success < 0 {
			t.Fatal("did not reach first actual success", copying)
		}
		t.Log("first constructor success", copying, success)
	}
}
func TestStrHashtableRuntimeBorrowRefsCloneSwapAndRemoval(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	keys := []StrKey{strStore(t, h, "a", 0), strStore(t, h, "b", 0), strStore(t, h, "c", 0)}
	for _, k := range keys {
		mapMust(t, h.Insert(k))
	}
	ref, ok, e := h.FindCore(keys[0])
	mapMust(t, e)
	if !ok {
		t.Fatal("missing")
	}
	got, e := ref.Key()
	mapMust(t, e)
	if got != keys[0] {
		t.Fatal("raw pointer codec")
	}
	hash, e := ref.Hash()
	mapMust(t, e)
	want, e := h.Hash(keys[0])
	mapMust(t, e)
	if hash != want {
		t.Fatal("cached hash")
	}
	copy, e := h.Clone()
	mapMust(t, e)
	mapMust(t, h.Remove(keys[1]))
	if h.Size() != 2 || copy.Size() != 3 {
		t.Fatal("independent payload")
	}
	mapMust(t, h.Swap(copy))
	if h.Size() != 3 || copy.Size() != 2 {
		t.Fatal("pointer and scalar swaps")
	}
	n := 0
	for it := h.Begin(); !it.AtEnd(); mapMust(t, it.Next()) {
		_, e := it.Key()
		mapMust(t, e)
		n++
	}
	if n != 3 {
		t.Fatal("ascending used iteration", n)
	}
	if !h.CheckInvariant() || !copy.CheckInvariant() {
		t.Fatal("physical state counts")
	}
	runtimeStringGraph(t, b, r, region, h, copy)
	mapMust(t, h.CloseChecked())
	mapMust(t, copy.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeEveryGrowthReservationAndRealTLSRetry(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		b := arena.NewBudget(1 << 26)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		mapMust(t, e)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
		mapMust(t, e)
		for _, s := range []string{"a", "b", "c", "d", "e", "f", "g"} {
			mapMust(t, h.Insert(strStore(t, h, s, 0)))
		}
		k := strStore(t, h, "h", 0)
		old, e := h.RuntimeRoots()
		mapMust(t, e)
		b.FailAfter(ordinal)
		e = h.Insert(k)
		runtimeStringGraph(t, b, r, region, h)
		if e == nil {
			success = int(ordinal)
			if h.Size() != 8 || h.Capacity() != 16 {
				t.Fatal("growth publication")
			}
		} else {
			now, e := h.RuntimeRoots()
			mapMust(t, e)
			if now.Table != old.Table || h.Size() != 7 || h.Capacity() != 8 {
				t.Fatal("pre-free old table frontier")
			}
		}
		b.FailAfter(-1)
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		mapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if e == nil {
			break
		}
	}
	if success < 0 {
		t.Fatal("growth first success missing")
	}
	t.Log("first growth success", success)
	// Real postphysical TLS limit failure yields an unjoined NEW table. The
	// allocation engine does not read its OOM flag: after lifting the size limit,
	// the literal next expansion may succeed while the earlier block is retained.
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 2048})
	mapMust(t, e)
	mapMust(t, r.SetMaxSize(1))
	e = h.rehashRuntime(h.Capacity() << 1)
	if e == nil || !r.State().OutOfMemory {
		t.Fatal("actual reached TLS limit", e, r.State())
	}
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	if len(roots.Acquisitions) != 1 || h.Capacity() != 2048 {
		t.Fatal("actual unjoined table", roots)
	}
	runtimeStringGraph(t, b, r, region, h)
	pending := roots.Acquisitions[0]
	mapMust(t, r.SetMaxSize(0))
	mapMust(t, h.rehashRuntime(h.Capacity()<<1))
	roots, e = h.RuntimeRoots()
	mapMust(t, e)
	if h.Capacity() != 4096 || len(roots.Acquisitions) != 1 || roots.Acquisitions[0] != pending {
		t.Fatal("valid retry lost earlier physical root", roots)
	}
	runtimeStringGraph(t, b, r, region, h)
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeSourceDestructorPostphysicalStopOwningAcknowledgment(t *testing.T) {
	// Public Runtime operations reach the source signed-overflow guard. This is
	// a diagnostic unsafe-domain fixture, not ordinary free/OOM behavior.
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 8192})
	mapMust(t, e)
	first, e := r.Allocate(8)
	mapMust(t, e)
	second, e := r.Allocate(8)
	mapMust(t, e)
	delta := int64(-1000000)
	request := uint64(16) + uint64(delta) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	mapMust(t, r.Deallocate(&first))
	state := r.State()
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	mapMust(t, r.Deallocate(&second))
	before, e := h.RuntimeRoots()
	mapMust(t, e)
	cause := h.SourceFinalize()
	if !errors.Is(cause, arena.ErrBounds) {
		t.Fatal("real release synchronization", cause)
	}
	after, e := h.RuntimeRoots()
	mapMust(t, e)
	if after.Table != (Block{}) || after.Object != before.Object || h.raw.sourceStage != 0 {
		t.Fatal("table consumed before object DT", after)
	}
	runtimeStringGraph(t, b, r, region, h)
	if again := h.SourceFinalize(); again != cause {
		t.Fatal("source cause not sticky", again, cause)
	}
	b.FailAfter(0)
	if e = h.CloseChecked(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("object physically consumed before next real sync guard", e)
	}
	runtimeStringGraph(t, b, r, region, h)
	if h.raw.object != (Block{}) || h.closed {
		t.Fatal("actual owning stop prefix")
	}
	mapMust(t, h.CloseChecked())
	if h.raw.sourceCause != cause {
		t.Fatal("owning abort repaired source cause")
	}
	b.FailAfter(-1)
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeLexicalIntersectionHasOnlyTableAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	other, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	for _, s := range []string{"a", "b", "c"} {
		mapMust(t, h.Insert(strStore(t, h, s, 0)))
	}
	mapMust(t, other.Insert(strStore(t, other, "b", 0)))
	before := r.State().GlobalCount
	mapMust(t, h.Intersect(other))
	if got := r.State().GlobalCount - before; got != 1 {
		t.Fatal("lexical inline object must not malloc wrapper", got)
	}
	if h.Size() != 1 {
		t.Fatal("intersection size", h.Size())
	}
	for _, q := range []string{"a", "b", "c"} {
		_, found, e := h.FindCString(q)
		mapMust(t, e)
		if found != (q == "b") {
			t.Fatal("independent finite set oracle", q, found)
		}
	}
	if h.raw.lexical != nil {
		t.Fatal("completed lexical context was not retired")
	}
	runtimeStringGraph(t, b, r, region, h, other)
	for i := 0; i < 8; i++ {
		beforeBytes := b.Used()
		mapMust(t, h.Intersect(other))
		if b.Used() != beforeBytes {
			t.Fatal("warm lexical copy cold growth", beforeBytes, b.Used())
		}
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, other.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestStrHashtableRuntimeUnusedEntryWordsStayUnread(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	// Malformed source pointer-domain fixture: a USED sentinel with an unrelated
	// cached hash never reaches its uninitialized borrowed pointer metadata.
	query := "absent"
	hash := String(query, 17)
	slot := hash & 7
	mapMust(t, r.WriteUint32(roots.Table, uint64(slot)*32+4, strHashUsed))
	mapMust(t, r.WriteUint32(roots.Table, uint64(slot)*32, hash^1))
	_, found, e := h.FindCString(query)
	mapMust(t, e)
	if found {
		t.Fatal("hash mismatch")
	}
	_, found, e = h.FindCStringBytes([]byte(query))
	mapMust(t, e)
	if found {
		t.Fatal("byte query hash mismatch")
	}
	// Invariant/iterator state scans do not read the uninitialized data/hash of
	// free entries or the unused data words of this malformed USED entry.
	if h.Begin().AtEnd() {
		t.Fatal("state-only iterator scan")
	}
	ok, e := h.CheckInvariantChecked()
	if strHashtableSourceDebug {
		if !errors.Is(e, ErrStrHashtableAssertion) || ok {
			t.Fatal("literal false SASSERT has no bound action", ok, e)
		}
	} else {
		mapMust(t, e)
		if ok {
			t.Fatal("release inspection adapter counts differ in malformed fixture")
		}
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeRemoveDeletedOOMShortCircuit(t *testing.T) {
	for _, oom := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		mapMust(t, e)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 256})
		mapMust(t, e)
		var keys []StrKey
		for n := 0; len(keys) < 70; n++ {
			s := fmt.Sprintf("collision-%d", n)
			if String(s, 17)&255 != 0 {
				continue
			}
			k := strStore(t, h, s, 0)
			keys = append(keys, k)
			mapMust(t, h.Insert(k))
		}
		if oom {
			mapMust(t, r.SetMaxSize(1))
			bad, e := r.Allocate(1)
			if e == nil || bad != (Block{}) || !r.State().OutOfMemory {
				t.Fatal("real prephysical source limit", bad, e)
			}
			mapMust(t, r.SetMaxSize(0))
		}
		for i := 0; i < 64; i++ {
			mapMust(t, h.Remove(keys[i]))
		}
		old, e := h.RuntimeRoots()
		mapMust(t, e)
		count := r.State().GlobalCount
		if oom {
			b.FailAfter(0)
		}
		mapMust(t, h.Remove(keys[64]))
		if h.Size() != 5 {
			t.Fatal("source size--", h.Size())
		}
		now, e := h.RuntimeRoots()
		mapMust(t, e)
		if oom {
			if h.Deleted() != 65 || now.Table != old.Table || r.State().GlobalCount != count {
				t.Fatal("OOM removes no table / no reservation", h.Deleted(), now)
			}
		} else {
			if h.Deleted() != 0 || now.Table == old.Table || r.State().GlobalCount != count+1 {
				t.Fatal("source same-cap clean publication", h.Deleted(), now)
			}
		}
		runtimeStringGraph(t, b, r, region, h)
		b.FailAfter(-1)
		for i, k := range keys {
			got, e := h.Contains(k)
			mapMust(t, e)
			if got != (i >= 65) {
				t.Fatal("finite membership oracle", i, got)
			}
		}
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		mapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestStrHashtableRuntimeSizeT32WrappedArrayGuardPrefix(t *testing.T) {
	// Selected entry-stride multiplication reaches a size_t32 zero allocation.
	// The first entry constructor write is guarded; this malformed/wrapped source
	// extent is not ordinary exhaustion and does not request a giant host block.
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 1 << 27})
	if !errors.Is(e, arena.ErrBounds) || h == nil || h.raw.complete {
		t.Fatal("reached zero-extent entry write", h, e)
	}
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	if roots.Object != (Block{}) || roots.Table != (Block{}) || len(roots.Acquisitions) != 1 {
		t.Fatal("wrapper unwind vs failed array", roots)
	}
	n, e := r.Size(roots.Acquisitions[0])
	mapMust(t, e)
	if n != 0 {
		t.Fatal("actual size_t caller conversion", n)
	}
	runtimeStringGraph(t, b, r, region, h)
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeEveryResetAndFinalizeReplacementReservation(t *testing.T) {
	for _, finalize := range []bool{false, true} {
		first := -1
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			cap := uint32(64)
			if finalize {
				cap = 128
			}
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: cap})
			mapMust(t, e)
			mapMust(t, h.Insert(strStore(t, h, "one", 0)))
			old, e := h.RuntimeRoots()
			mapMust(t, e)
			b.FailAfter(ordinal)
			if finalize {
				e = h.Finalize()
			} else {
				e = h.Reset()
			}
			runtimeStringGraph(t, b, r, region, h)
			if _, guard := r.Size(old.Table); guard == nil {
				t.Fatal("source old table not consumed before replacement", ordinal)
			}
			if e == nil {
				first = int(ordinal)
				if h.Size() != 0 || h.Deleted() != 0 {
					t.Fatal("completed counter reset")
				}
			} else {
				if h.Size() != 1 {
					t.Fatal("source old size erased before replacement completed")
				}
				want := uint32(32)
				if finalize {
					want = 128
				}
				if h.Capacity() != want {
					t.Fatal("capacity publication frontier", h.Capacity(), want)
				}
			}
			b.FailAfter(-1)
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if e == nil {
				break
			}
		}
		if first < 0 {
			t.Fatal("replacement first success missing", finalize)
		}
		t.Log("replacement first success", finalize, first)
	}
}
func TestStrHashtableRuntimeForeignBorrowLifetimeAndTypedObserverGuard(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	fref, e := foreign.Allocate(4)
	mapMust(t, e)
	for i, v := range []byte{'f', 'o', 'o', 0} {
		mapMust(t, foreign.Write(fref, uint32(i), v))
	}
	key, e := BorrowCString(foreign, fref, 0)
	mapMust(t, e)
	mapMust(t, h.InsertFrom(foreign, key))
	ref, found, e := h.FindCString("foo")
	mapMust(t, e)
	if !found {
		t.Fatal("foreign borrow not retained")
	}
	mapMust(t, foreign.Close())
	if _, e = ref.Key(); e == nil {
		t.Fatal("reached foreign lifetime read silently succeeded")
	}
	// Cached movement does not dereference foreign spelling bytes.
	mapMust(t, h.rehashRuntime(16))
	if _, found, e = h.FindCString("foo"); e == nil || found {
		t.Fatal("actual strcmp lifetime guard", found, e)
	}
	mapMust(t, h.SourceFinalize())
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("no-error observer silently returned after ended object")
			}
		}()
		_ = h.Size()
	}()
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Source: src/util/hashtable.h:375-410,481-496,587-609.
func TestStrHashtableRuntimeEveryLexicalAndDeletedReplacementReservation(t *testing.T) {
	for _, lexical := range []bool{true, false} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			b := arena.NewBudget(1 << 28)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 256})
			mapMust(t, e)
			var other *StrHashtable
			var keys []StrKey
			if lexical {
				other, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
				mapMust(t, e)
				for _, s := range []string{"a", "b", "c"} {
					mapMust(t, h.Insert(strStore(t, h, s, 0)))
				}
				mapMust(t, other.Insert(strStore(t, other, "b", 0)))
			} else {
				for n := 0; len(keys) < 70; n++ {
					s := fmt.Sprintf("reservation-collision-%d", n)
					if String(s, 17)&255 != 0 {
						continue
					}
					k := strStore(t, h, s, 0)
					keys = append(keys, k)
					mapMust(t, h.Insert(k))
				}
				for i := 0; i < 64; i++ {
					mapMust(t, h.Remove(keys[i]))
				}
			}
			b.FailAfter(ordinal)
			if lexical {
				e = h.Intersect(other)
			} else {
				e = h.Remove(keys[64])
			}
			runtimeStringGraph(t, b, r, region, h, other)
			if e == nil {
				success = ordinal
			}
			if !lexical && (h.Size() != 5 || (e != nil && h.Deleted() != 65)) {
				t.Fatal("reached delete/count prefix", ordinal, h.Size(), h.Deleted(), e)
			}
			b.FailAfter(-1)
			if e == nil {
				if lexical {
					for _, s := range []string{"a", "b", "c"} {
						_, found, q := h.FindCString(s)
						mapMust(t, q)
						if found != (s == "b") {
							t.Fatal(s, found)
						}
					}
				} else {
					for i, k := range keys {
						found, q := h.Contains(k)
						mapMust(t, q)
						if found != (i >= 65) {
							t.Fatal(i, found)
						}
					}
				}
			}
			mapMust(t, h.CloseChecked())
			if other != nil {
				mapMust(t, other.CloseChecked())
			}
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("once-only prefix refund", ordinal, b.Used())
			}
			if e == nil {
				break
			}
		}
		if success < 0 {
			t.Fatal("first successful actual reservation not reached", lexical)
		}
		t.Log("first replacement success", lexical, success)
	}
}

func TestStrHashtableRuntimeOrdinaryAllocatorProfiles(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, tls := range []bool{false, true} {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: bits, ThreadLocal: tls})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
			mapMust(t, e)
			for _, s := range []string{"first", "second", "third"} {
				mapMust(t, h.Insert(strStore(t, h, s, 0)))
			}
			runtimeStringGraph(t, b, r, region, h)
			mapMust(t, h.SourceFinalize())
			runtimeStringGraph(t, b, r, region, h)
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(bits, tls, b.Used())
			}
		}
	}
}
