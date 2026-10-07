// Source: src/util/id_gen.h; src/util/vector.h.
package util

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Added finite source-profile checks: id_gen.h has no dedicated test driver.
func TestGeneratorRawOrderAndLifecycle(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(^uint32(0), r); e != nil {
		t.Fatal(e)
	}
	for _, want := range []uint32{^uint32(0), 0} {
		v, e := g.Next()
		if e != nil || v != want {
			t.Fatal(v, e, want)
		}
	}
	for _, v := range []uint32{17, 19, 17, 23, 29} {
		if e = g.Recycle(v); e != nil {
			t.Fatal(e)
		}
	}
	if n, e := g.Capacity(); e != nil || n != 5 {
		t.Fatal(n, e)
	}
	h := g.RawRoots()[0]
	if n, e := r.Size(h); e != nil || n != 28 {
		t.Fatal(n, e)
	}
	// Independent byte sequence: cap=5,size=5 and the exact source LIFO data.
	want := []byte{5, 0, 0, 0, 5, 0, 0, 0, 17, 0, 0, 0, 19, 0, 0, 0, 17, 0, 0, 0, 23, 0, 0, 0, 29, 0, 0, 0}
	for i, w := range want {
		v, e := r.Read(h, uint64(i))
		if e != nil || v != w {
			t.Fatal(i, v, e, w)
		}
	}
	for _, want := range []uint32{29, 23, 17, 19, 17, 1} {
		v, e := g.Next()
		if e != nil || v != want {
			t.Fatal(v, e, want)
		}
	}
	if e = g.Reset(40); e != nil {
		t.Fatal(e)
	}
	if g.RawRoots()[0] != h {
		t.Fatal("reset released source retained storage")
	}
	if e = g.Cleanup(50); e != nil {
		t.Fatal(e)
	}
	if g.RawRoots() != [2]Block{} {
		t.Fatal("cleanup roots")
	}
	if v, e := g.Next(); e != nil || v != 50 {
		t.Fatal(v, e)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestGeneratorRealRefusalAndOOM(t *testing.T) {
	for _, grow := range []bool{false, true} {
		b := arena.NewBudget(1 << 20)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		if e != nil {
			t.Fatal(e)
		}
		var g Generator
		if e = g.Init(7, r); e != nil {
			t.Fatal(e)
		}
		if grow {
			for _, v := range []uint32{7, 8} {
				if e = g.Recycle(v); e != nil {
					t.Fatal(e)
				}
			}
		}
		h := g.RawRoots()[0]
		b.FailAfter(0)
		if e = g.Recycle(9); !errors.Is(e, Failure{Kind: FailureMemout}) {
			t.Fatal(e)
		}
		if !r.State().OutOfMemory {
			t.Fatal("real Runtime allocation refusal must set source OOM")
		}
		if g.RawRoots()[0] != h {
			t.Fatal("failed allocation replaced actual token")
		}
		b.FailAfter(-1)
		// recycle checks real Runtime OOM before the vector, including after refusal.
		if e = g.Recycle(10); e != nil {
			t.Fatal(e)
		}
		want := uint32(7)
		if grow {
			want = 8
		}
		if v, e := g.Next(); e != nil || v != want {
			t.Fatal(v, e, want)
		}
		if e = g.CloseChecked(); e != nil {
			t.Fatal(e)
		}
		if e = r.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestSourceVectorGrowthWidths(t *testing.T) {
	if n, b, e := VectorGrowth(0x40000000); e != nil || n != 0x60000000 || b != 0x80000008 {
		t.Fatal(n, b, e)
	}
	for _, p := range [][2]uint32{{0, 2}, {2, 3}, {3, 5}, {5, 8}, {8, 12}, {12, 18}, {18, 27}} {
		n, _, e := VectorGrowth(p[0])
		if e != nil || n != p[1] {
			t.Fatal(p, n, e)
		}
	}
	for _, old := range []uint32{0x55555555, 0xaaaaaaaa, ^uint32(0)} {
		_, _, e := VectorGrowth(old)
		if !errors.Is(e, Failure{Kind: FailureVectorOverflow}) {
			t.Fatal(old, e)
		}
	}
}

func TestGeneratorPendingPublicationAndOwnerGuards(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{ThreadLocal: true})
	if e != nil {
		t.Fatal(e)
	}
	filler, e := r.Allocate(99976)
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	copied := g
	if _, e = copied.Next(); !errors.Is(e, ErrOwner) {
		t.Fatal(e)
	}
	if e = r.SetMaxSize(1); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(42); !errors.Is(e, Failure{Kind: FailureMemout}) {
		t.Fatal(e)
	}
	roots := g.RawRoots()
	if roots[0] != (Block{}) || roots[1] == (Block{}) {
		t.Fatal("initial mem must remain lexical until source vector publication", roots)
	}
	if st := r.Stats(); st.LiveBlocks != 2 {
		t.Fatal(st)
	}
	if _, e = g.Next(); !errors.Is(e, ErrInterrupted) {
		t.Fatal(e)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if g.RawRoots() != [2]Block{} {
		t.Fatal("owning cleanup did not release pending mem")
	}
	if _, e = g.Next(); !errors.Is(e, ErrOwner) {
		t.Fatal(e)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Deallocate(&filler); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestGeneratorWarmRawStorage(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(41); e != nil {
		t.Fatal(e)
	}
	n := testing.AllocsPerRun(100, func() {
		v, e := g.Next()
		if e != nil || v != 41 {
			panic("ID order")
		}
		if e = g.Recycle(v); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal("warm raw ID generation allocated", n)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestGeneratorDebugSourceScanAndHash(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(17, r); e != nil {
		t.Fatal(e)
	}
	for _, v := range []uint32{3, 4, 6, 0x04030201, 0xfefdfcfb} {
		if e = g.Recycle(v); e != nil {
			t.Fatal(e)
		}
	}
	if v, e := g.SetNextID(3); e != nil || v != 5 {
		t.Fatal(v, e)
	}
	var sink idTestText
	if e = g.DisplayFreeIDs(&sink); e != nil || sink.text != "3 4 6 67305985 4278058235" {
		t.Fatal(sink.text, e)
	}
	// Independent literal byte fixture; high bytes exercise signed-char tails.
	bytes := string([]byte{3, 0, 0, 0, 4, 0, 0, 0, 6, 0, 0, 0, 1, 2, 3, 4, 0xfb, 0xfc, 0xfd, 0xfe})
	if v, e := g.ShowHash(); e != nil || v != UintPair(String(bytes, 17), 5) {
		t.Fatal(v, e)
	}
	if e = g.Reset(99); e != nil {
		t.Fatal(e)
	}
	// Explicit empty &free_ids[0] diagnostic adapter avoids source invalid access.
	if v, e := g.ShowHash(); e != nil || v != UintPair(String("", 17), 99) {
		t.Fatal(v, e)
	}
	if e = g.Recycle(^uint32(0)); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(0); e != nil {
		t.Fatal(e)
	}
	if v, e := g.SetNextID(^uint32(0)); e != nil || v != 1 {
		t.Fatal(v, e)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}

type idTestText struct{ text string }

func (s *idTestText) WriteLiteral(v string) error { s.text += v; return nil }
func (s *idTestText) WriteUnsigned(v uint32) error {
	s.text += strconv.FormatUint(uint64(v), 10)
	return nil
}

func TestGeneratorBorrowedRuntimeLifetime(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = g.Next(); e == nil {
		t.Fatal("borrowed runtime ended before generator")
	}
}

// Malformed unsigned-vector geometry profile: source growth arithmetic accepts
// these headers, then ILP32 SIZE_T allocation extent wraps before header write.
func TestGeneratorWrappedReallocPhysicalCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(1); e != nil {
		t.Fatal(e)
	}
	h := g.RawRoots()[0]
	for _, off := range []uint64{0, 4} {
		if e = r.WriteUint32(h, off, 0x55555553); e != nil {
			t.Fatal(e)
		}
	}
	if e = g.Recycle(2); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if st := r.Stats(); st.LiveBlocks != 1 || st.PhysicalBytes != 0 {
		t.Fatal(st)
	}
	if g.RawRoots()[0] == h {
		t.Fatal("actual realloc token generation did not advance")
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal("actual partial block must be releasable", e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestGeneratorNonNullZeroCapacitySourceBranch(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(42); e != nil {
		t.Fatal(e)
	}
	h := g.RawRoots()[0]
	if e = r.WriteUint32(h, 0, 0); e != nil {
		t.Fatal(e)
	}
	if e = r.WriteUint32(h, 4, 0); e != nil {
		t.Fatal(e)
	}
	var want error = Failure{Kind: FailureVectorOverflow}
	if idSourceDebug {
		want = ErrAssertion
	}
	if e = g.Recycle(99); !errors.Is(e, want) {
		t.Fatal(e, want)
	}
	if g.RawRoots()[0] != h {
		t.Fatal("nonnull cap0 source error must precede realloc")
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestGeneratorSourceFinalizeRetainsUnpublishedMem(t *testing.T) {
	r, e := NewRuntime(RuntimeConfig{ThreadLocal: true})
	if e != nil {
		t.Fatal(e)
	}
	filler, e := r.Allocate(99976)
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(17, r); e != nil {
		t.Fatal(e)
	}
	if e = r.SetMaxSize(1); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(42); !errors.Is(e, Failure{Kind: FailureMemout}) {
		t.Fatal(e)
	}
	pending := g.RawRoots()[1]
	if pending == (Block{}) {
		t.Fatal("missing real initial mem")
	}
	if e = g.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if g.RawRoots()[1] != pending {
		t.Fatal("source vector destructor consumed unpublished lexical mem")
	}
	if _, e = g.Next(); !errors.Is(e, ErrOwner) {
		t.Fatal(e)
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Deallocate(&filler); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestGeneratorSourceFinalizePreservesHeaderGuard(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	var g Generator
	if e = g.Init(0, r); e != nil {
		t.Fatal(e)
	}
	if e = g.Recycle(1); e != nil {
		t.Fatal(e)
	}
	h := g.RawRoots()[0]
	for _, off := range []uint64{0, 4} {
		if e = r.WriteUint32(h, off, 0x55555553); e != nil {
			t.Fatal(e)
		}
	}
	if e = g.Recycle(2); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	h = g.RawRoots()[0]
	if e = g.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if g.RawRoots()[0] != h {
		t.Fatal("source header guard released physical short block")
	}
	if e = g.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
