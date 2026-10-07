// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp; src/util/small_object_allocator.cpp.
package util

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Every physical root comes from a named caller, actual allocator chunk chain,
// or explicit pending field. Runtime records are only the comparison census.
func fixedRuntimeAudit(t *testing.T, m *RuntimeFixedBitVectorManager, values ...RuntimeFixedBitVector) {
	fixedRuntimeAuditExtra(t, m, values, nil)
}
func fixedRuntimeAuditExtra(t *testing.T, m *RuntimeFixedBitVectorManager, values []RuntimeFixedBitVector, extras []Block) {
	t.Helper()
	r := m.runtime
	rv := reflect.ValueOf(r).Elem()
	av := reflect.ValueOf(m.allocator).Elem()
	records := testFieldBySourceName(rv, "blocks")
	type key struct{ slot, gen uint32 }
	seen := map[key]bool{}
	add := func(h reflect.Value) {
		if testFieldBySourceName(h, "slot").Uint() == 0 {
			return
		}
		if testFieldBySourceName(h, "origin").Uint() != testFieldBySourceName(rv, "origin").Uint() {
			t.Fatal("foreign physical provider")
		}
		k := key{uint32(testFieldBySourceName(h, "slot").Uint()), uint32(testFieldBySourceName(h, "generation").Uint())}
		if int(k.slot) > records.Len() {
			t.Fatal("root bounds")
		}
		rec := records.Index(int(k.slot) - 1)
		if !testFieldBySourceName(rec, "live").Bool() || uint32(testFieldBySourceName(rec, "generation").Uint()) != k.gen {
			t.Fatal("dead named root", k)
		}
		seen[k] = true
	}
	for _, h := range extras {
		add(reflect.ValueOf(h))
	}
	for _, v := range values {
		if v.inline || v.object == (ObjectRef{}) {
			continue
		}
		h, _, e := m.allocator.RawLocation(v.object)
		heapMust(t, e)
		add(reflect.ValueOf(h))
	}
	pending, e := m.allocator.PendingRoots()
	heapMust(t, e)
	for _, h := range pending {
		add(reflect.ValueOf(h))
	}
	locations := testFieldBySourceName(av, "locations")
	heads := testFieldBySourceName(av, "chunks")
	visited := map[uint64]bool{}
	width := uint64(m.profile.PointerBits / 8)
	for i := 0; i < heads.Len(); i++ {
		id := heads.Index(i).Uint()
		for id != 0 {
			if visited[id] {
				t.Fatal("chunk ownership cycle")
			}
			visited[id] = true
			if id > uint64(locations.Len()) {
				t.Fatal("chunk bounds")
			}
			loc := locations.Index(int(id) - 1)
			raw := testFieldBySourceName(loc, "raw")
			add(raw)
			k := int(testFieldBySourceName(raw, "slot").Uint()) - 1
			data := testFieldBySourceName(records.Index(k), "data")
			var next uint64
			for j := uint64(0); j < width; j++ {
				next |= data.Index(int(width+j)).Uint() << (8 * j)
			}
			id = next
		}
	}
	var physical, bitmap uint64
	live := 0
	for i := 0; i < records.Len(); i++ {
		rec := records.Index(i)
		if !testFieldBySourceName(rec, "live").Bool() {
			continue
		}
		live++
		k := key{uint32(i + 1), uint32(testFieldBySourceName(rec, "generation").Uint())}
		if !seen[k] {
			t.Fatal("unnamed physical root", k)
		}
		physical += uint64(testFieldBySourceName(rec, "data").Len())
		bitmap += uint64(testFieldBySourceName(rec, "initialized").Len()) * 8
	}
	st := r.Stats()
	if live != len(seen) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
		t.Fatal("physical bijection", st, live, len(seen), physical, bitmap)
	}
	want := st.TotalBytes + m.ownerBytes + m.allocator.Stats().TotalBytes
	if m.budget.Used() != want {
		t.Fatal("once-only cold+raw graph", m.budget.Used(), want)
	}
}
func TestFixedBitVectorRuntimeIndependentWorldsAndRange(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	a, e := m.Allocate0()
	heapMust(t, e)
	c, e := m.Allocate0()
	heapMust(t, e)
	for x := uint32(0); x < 32; x++ {
		for y := uint32(0); y < 32; y++ {
			heapMust(t, m.Fill0(a))
			heapMust(t, m.Fill0(c))
			for i := uint32(0); i < 5; i++ {
				heapMust(t, m.SetBool(a, i, x&(1<<i) != 0))
				heapMust(t, m.SetBool(c, i, y&(1<<i) != 0))
			}
			eq, e := m.Equal(a, c)
			heapMust(t, e)
			if eq != (x == y) {
				t.Fatal(x, y, "eq")
			}
			has, e := m.Contains(a, c)
			heapMust(t, e)
			if has != (x&y == y) {
				t.Fatal(x, y, "contains")
			}
			heapMust(t, m.Union(a, c))
			heapMust(t, m.Intersect(a, c))
			for i := uint32(0); i < 65; i++ {
				v, e := m.Get(a, i)
				heapMust(t, e)
				if v != (i < 5 && y&(1<<i) != 0) {
					t.Fatal(x, y, i)
				}
			}
		}
	}
	for _, lo := range []uint32{0, 1, 7, 31, 32, 33} {
		for _, length := range []uint32{1, 7, 32} {
			if lo+length > 65 {
				continue
			}
			heapMust(t, m.Fill0(a))
			want := make([]bool, 65)
			for i := range want {
				want[i] = i%3 == 0
				heapMust(t, m.SetBool(a, uint32(i), want[i]))
			}
			for i := uint32(0); i < length; i++ {
				want[lo+i] = want[i]
			}
			heapMust(t, m.SetRange(a, a, lo+length-1, lo))
			for i, w := range want {
				v, e := m.Get(a, uint32(i))
				heapMust(t, e)
				if v != w {
					t.Fatal("ascending self range", lo, length, i)
				}
			}
		}
	}
	for _, lo := range []uint32{0, 1, 7, 31, 32, 33} {
		for _, count := range []uint32{1, 7, 32} {
			if lo+count > 65 {
				continue
			}
			want := make([]bool, 65)
			source := make([]bool, 65)
			for i := range want {
				want[i] = i%2 == 0
				source[i] = i%3 == 0
				heapMust(t, m.SetBool(a, uint32(i), want[i]))
				heapMust(t, m.SetBool(c, uint32(i), source[i]))
			}
			for i := uint32(0); i < count; i++ {
				want[lo+i] = source[i]
			}
			heapMust(t, m.SetRange(a, c, lo+count-1, lo))
			for i, w := range want {
				got, e := m.Get(a, uint32(i))
				heapMust(t, e)
				if got != w {
					t.Fatal("independent nonalias range", lo, count, i)
				}
			}
		}
	}
	heapMust(t, arena.ValidateRecordType(reflect.TypeOf(RuntimeFixedBitVector{})))

	fixedRuntimeAudit(t, m, a, c)
	heapMust(t, m.Deallocate(a))
	heapMust(t, m.Deallocate(c))
	fixedRuntimeFinish(t, m, r, b)
}
func TestFixedBitVectorRuntimeUninitializedInlineAndUnsignedWrap(t *testing.T) {
	for _, n := range []uint32{0, 1, ^uint32(0)} {
		m, r, b := fixedRuntimeFixture(t, n)
		v, e := m.Allocate()
		heapMust(t, e)
		w, e := m.NumWords()
		heapMust(t, e)
		if w != (n+31)/32 {
			t.Fatal(n, w)
		}
		if _, e = m.Word(v, 0); !errors.Is(e, ErrRuntimeUninitialized) {
			t.Fatal("source default word uninitialized", n, e)
		}
		eq, e := m.Equal(v, v)
		heapMust(t, e)
		if !eq {
			t.Fatal("pointer identity before reads")
		}
		if n == 0 {
			heapMust(t, m.Fill0(v))
			heapMust(t, m.Fill1(v))
			if m.zeroInitialized != 0 {
				t.Fatal("zero bytes initialized inlineword")
			}
			h, e := m.Hash(RuntimeFixedBitVector{})
			heapMust(t, e)
			if h != String("", 0) {
				t.Fatal(h)
			}
			last, e := m.LastWord(RuntimeFixedBitVector{})
			heapMust(t, e)
			if last != 0 {
				t.Fatal(last)
			}
		}
		if n == 1 {
			heapMust(t, m.Fill0(v))
			_, e = m.Get(v, 31)
			heapMust(t, e)
			e = m.SetBit(v, 31)
			if sourceDebug {
				if !errors.Is(e, ErrRuntimeVectorAssertion) {
					t.Fatal(e)
				}
			} else {
				heapMust(t, e)
				g, e := m.Get(v, 31)
				heapMust(t, e)
				if !g {
					t.Fatal("physical spare bit")
				}
			}
		}
		if n == ^uint32(0) {
			if _, e = m.Hash(v); !errors.Is(e, ErrRuntimeUninitialized) {
				t.Fatal("wrap first actual inline read", e)
			}
		}
		fixedRuntimeAudit(t, m, v)
		heapMust(t, m.Deallocate(v))
		fixedRuntimeFinish(t, m, r, b)
	}
}
func TestFixedBitVectorRuntimeAllocateCopyRetainsReachedCandidate(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	src, e := m.Allocate()
	heapMust(t, e)
	heapMust(t, m.putWord(src, 0, 0x12345678))
	dst, e := m.AllocateCopy(src)
	if !errors.Is(e, ErrRuntimeUninitialized) || dst.object == (ObjectRef{}) {
		t.Fatal("allocate BEFORE first unread source byte", dst, e)
	}
	fixedRuntimeAudit(t, m, src, dst)
	w, e := m.Word(dst, 0)
	heapMust(t, e)
	if w != 0x12345678 {
		t.Fatal("copied prefix", w)
	}
	if _, e = m.Word(dst, 1); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	heapMust(t, m.Deallocate(src))
	heapMust(t, m.Deallocate(dst))
	fixedRuntimeFinish(t, m, r, b)
}
func TestFixedBitVectorRuntimeEveryActualAllocationRefusal(t *testing.T) {
	for route := 0; route < 4; route++ {
		success, failed := 0, 0
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 24)
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
			heapMust(t, e)
			var m *RuntimeFixedBitVectorManager
			var src, out RuntimeFixedBitVector
			if route != 0 {
				m, e = NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
				heapMust(t, e)
				if route == 3 {
					src, e = m.Allocate1()
					heapMust(t, e)
				}
			}
			b.FailAfter(ordinal)
			if route == 0 {
				m, e = NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			} else if route == 1 {
				out, e = m.Allocate0()
			} else if route == 2 {
				out, e = m.Allocate1()
			} else {
				out, e = m.AllocateCopy(src)
			}
			if m != nil {
				fixedRuntimeAudit(t, m, src, out)
			} else if b.Used() != r.Stats().TotalBytes {
				t.Fatal("constructor unwind", b.Used(), r.Stats())
			}
			if e != nil {
				failed++
				intHeapActualRefusal(t, r, e)
			} else {
				success++
			}
			b.FailAfter(-1)
			if m != nil {
				for _, v := range []RuntimeFixedBitVector{src, out} {
					if v.object != (ObjectRef{}) {
						heapMust(t, m.Deallocate(v))
					}
				}
				heapMust(t, m.allocator.CleanupPending())
				heapMust(t, m.Close())
			}
			heapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(route, ordinal, b.Used())
			}
			if e == nil {
				t.Logf("route%d firstsuccess%d", route, ordinal)
				break
			}
		}
		if failed == 0 || success != 1 {
			t.Fatal(route, failed, success)
		}
	}
}
func TestFixedBitVectorRuntimeHashAndDisplayPrefixes(t *testing.T) {
	for n := uint32(0); n <= 40; n++ {
		m, r, b := fixedRuntimeFixture(t, n)
		v, e := m.Allocate1()
		heapMust(t, e)
		h, e := m.Hash(v)
		heapMust(t, e)
		if h != String(string(bytes.Repeat([]byte{255}, int(n/8))), n) {
			t.Fatal(n, h)
		}
		var text bytes.Buffer
		heapMust(t, m.Display(&text, v))
		if text.String() != string(bytes.Repeat([]byte{'1'}, int(n))) {
			t.Fatal(n, text.String())
		}
		heapMust(t, m.Deallocate(v))
		fixedRuntimeFinish(t, m, r, b)
	}
	m, r, b := fixedRuntimeFixture(t, 10)
	v, e := m.Allocate0()
	heapMust(t, e)
	heapMust(t, m.SetBit(v, 9))
	for limit := 0; limit < 10; limit++ {
		w := &fixedRuntimeWriter{remaining: limit}
		e = m.Display(w, v)
		if !errors.Is(e, io.ErrClosedPipe) || string(w.bytes) != "1000000000"[:limit] {
			t.Fatal(limit, e, string(w.bytes))
		}
	}
	heapMust(t, m.Deallocate(v))
	fixedRuntimeFinish(t, m, r, b)
}

type fixedRuntimeWriter struct {
	remaining int
	bytes     []byte
}

func (w *fixedRuntimeWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.ErrClosedPipe
	}
	w.bytes = append(w.bytes, p...)
	w.remaining -= len(p)
	return len(p), nil
}

func TestFixedBitVectorRuntimePaddingMasksResetValueAndShortCircuits(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	a, e := m.Allocate1()
	heapMust(t, e)
	c, e := m.Allocate1()
	heapMust(t, e)
	heapMust(t, m.putWord(c, 2, 1))
	eq, e := m.Equal(a, c)
	heapMust(t, e)
	if !eq {
		t.Fatal("last unsigned padding masked")
	}
	has, e := m.Contains(c, a)
	heapMust(t, e)
	if !has {
		t.Fatal("contains masks both finalwords")
	}
	ha, e := m.Hash(a)
	heapMust(t, e)
	hc, e := m.Hash(c)
	heapMust(t, e)
	if ha != hc {
		t.Fatal("floor-byte cutoff excludes finalbit/padding")
	}
	heapMust(t, m.ResetValue(c))
	last, e := m.LastWord(c)
	heapMust(t, e)
	if last != 0 {
		t.Fatal(last)
	}
	raw, e := m.Allocate()
	heapMust(t, e)
	heapMust(t, m.putWord(raw, 0, 1))
	eq, e = m.Equal(c, raw)
	heapMust(t, e)
	if eq {
		t.Fatal("first difference before unread laterword")
	}
	has, e = m.Contains(c, raw)
	heapMust(t, e)
	if has {
		t.Fatal("first missing bit before unread suffix")
	}
	fixedRuntimeAudit(t, m, a, c, raw)
	for _, v := range []RuntimeFixedBitVector{a, c, raw} {
		heapMust(t, m.Deallocate(v))
	}
	fixedRuntimeFinish(t, m, r, b)
}

func TestFixedBitVectorRuntimePooledEveryActualRefusalAndFreelistBytes(t *testing.T) {
	for _, debug := range []bool{false, true} {
		for route := 0; route < 2; route++ {
			success, failed := 0, 0
			for ordinal := int64(0); ordinal < 64; ordinal++ {
				b := arena.NewBudget(1 << 24)
				r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
				heapMust(t, e)
				m, e := NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: debug, Windows: debug}})
				heapMust(t, e)
				var src RuntimeFixedBitVector
				if route == 1 {
					src, e = m.Allocate1()
					heapMust(t, e)
				}
				before := testFieldBySourceName(reflect.ValueOf(m.allocator).Elem(), "allocSize").Uint()
				b.FailAfter(ordinal)
				var out RuntimeFixedBitVector
				if route == 0 {
					out, e = m.Allocate0()
				} else {
					out, e = m.AllocateCopy(src)
				}
				bodyError := e
				if got := testFieldBySourceName(reflect.ValueOf(m.allocator).Elem(), "allocSize").Uint(); got != before+12 {
					t.Fatal("requested source counter precedes actual reservation", got, before)
				}
				fixedRuntimeAudit(t, m, src, out)
				if e != nil {
					failed++
					intHeapActualRefusal(t, r, e)
				} else {
					success++
				}
				b.FailAfter(-1)
				for _, v := range []RuntimeFixedBitVector{src, out} {
					if v.object != (ObjectRef{}) {
						heapMust(t, m.Deallocate(v))
					}
				}
				heapMust(t, m.allocator.CleanupPending())
				e = m.Close()
				if debug && bodyError != nil {
					if !errors.Is(e, ErrAllocatorIO) {
						t.Fatal("unrolled failed requested-byte prefix reached real diagnostic", e)
					}
					if r.Stats().LiveBlocks != 0 {
						t.Fatal(r.Stats())
					}
					heapMust(t, m.CleanupStopped())
				} else {
					heapMust(t, e)
				}
				heapMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal(debug, route, ordinal, b.Used())
				}
				if bodyError == nil {
					t.Logf("pooleddebug%v route%d firstsuccess%d", debug, route, ordinal)
					break
				}
			}
			if success != 1 || failed == 0 {
				t.Fatal(debug, route, success, failed)
			}
		}
	}
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	m, e := NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64}})
	heapMust(t, e)
	v, e := m.Allocate1()
	heapMust(t, e)
	old, e := m.Capture(v)
	heapMust(t, e)
	heapMust(t, m.Deallocate(v))
	next, e := m.Allocate()
	heapMust(t, e)
	span, e := m.Capture(next)
	heapMust(t, e)
	if span.Block != old.Block || span.Offset != old.Offset {
		t.Fatal("actual source freelist reuse")
	}
	w, e := m.Word(next, 2)
	heapMust(t, e)
	if w != ^uint32(0) {
		t.Fatal("allocate did not initialize/zero old payload", w)
	}
	if _, e = m.Get(v, 0); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("checked stale ObjectRef generation", e)
	}
	fixedRuntimeAudit(t, m, next)
	heapMust(t, m.Deallocate(next))
	fixedRuntimeFinish(t, m, r, b)
}
