// Actual source diagnostic byte streams stay in main; core tests moved to memory.
package diagnostic

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func sourceSmallObjectMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func sourceSmallObjectAlloc(t *testing.T, a *utilSmallObjectAllocator, n uint64) utilSmallObjectRef {
	t.Helper()
	r, e := a.Allocate(n)
	sourceSmallObjectMust(t, e)
	return r
}
func smallObjectDiagnosticFixture(t *testing.T) (*umemory.Runtime, *arena.Budget, *arena.BytesRegion, *utext.NumericTextStream) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	rt, e := newUtilMemoryRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	sourceSmallObjectMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
	sourceSmallObjectMust(t, e)
	stream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	sourceSmallObjectMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		stream.Close()
		region.Close()
		if e := rt.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("diagnostic borrowed graph leak", b.Used())
		}
	})
	return rt, b, region, stream
}
func smallObjectID(t *testing.T, r *arena.BytesRegion, text string) utext.RegionCString {
	t.Helper()
	ref, e := r.Allocate(uint32(len(text) + 1))
	sourceSmallObjectMust(t, e)
	for i := range text {
		sourceSmallObjectMust(t, r.Write(ref, uint32(i), text[i]))
	}
	sourceSmallObjectMust(t, r.Write(ref, uint32(len(text)), 0))
	return utext.RegionCString{Origin: r.OwnerID(), Ref: ref, Present: true}
}
func TestSmallObjectAllocatorBorrowedIDMutationAndWholeCStringWidth(t *testing.T) {
	rt, _, region, out := smallObjectDiagnosticFixture(t)
	id := smallObjectID(t, region, "abc\x00ignored")
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, BorrowedID: region, IDRef: id, Diagnostics: out})
	sourceSmallObjectMust(t, e)
	_ = sourceSmallObjectAlloc(t, a, 1)
	sourceSmallObjectMust(t, region.Write(id.Ref, 0, 'z'))
	sourceSmallObjectMust(t, a.Close())
	if rt.Stats().PhysicalBytes != 0 {
		t.Fatal("diagnostic preceded chunk free")
	}
	text, e := out.String()
	sourceSmallObjectMust(t, e)
	if text != "Memory leak detected for small object allocator 'zbc'. 1 bytes leaked\n" {
		t.Fatal("borrowed mutable ID/NUL", text)
	}
	sourceSmallObjectMust(t, out.ResetText())
	sourceSmallObjectMust(t, out.SetWidth(8))
	sourceSmallObjectMust(t, smallObjectCStringInto(out, region, id))
	text, e = out.String()
	sourceSmallObjectMust(t, e)
	if text != "     zbc" || out.State().Width != 0 {
		t.Fatal("C-string inserted as per-character fields", text, out.State())
	}
}
func TestSmallObjectAllocatorNullStaleIDErrorPrefixes(t *testing.T) {
	for _, stale := range []bool{false, true} {
		rt, _, region, out := smallObjectDiagnosticFixture(t)
		id := utext.RegionCString{}
		if stale {
			id = smallObjectID(t, region, "old")
			region.Reset()
		}
		a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, BorrowedID: region, IDRef: id, Diagnostics: out})
		sourceSmallObjectMust(t, e)
		_ = sourceSmallObjectAlloc(t, a, 1)
		e = a.Close()
		if stale {
			if e != arena.ErrInvalidSlot {
				t.Fatal("stale spelling not checked at insertion", e)
			}
		} else if e != utext.ErrNullCString {
			t.Fatal("null ostream insertion adapter", e)
		}
		if rt.Stats().PhysicalBytes != 0 {
			t.Fatal("ID checked before destructor physical prefix")
		}
		text, e := out.String()
		sourceSmallObjectMust(t, e)
		if text != "Memory leak detected for small object allocator '" {
			t.Fatal("ID error prefix", stale, text)
		}
	}
}
func TestSmallObjectAllocatorDiagnosticGoBufferAfterFree(t *testing.T) {
	rt, b, _, out := smallObjectDiagnosticFixture(t)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, ID: "a long borrowed static allocator identifier", Diagnostics: out})
	sourceSmallObjectMust(t, e)
	_ = sourceSmallObjectAlloc(t, a, 1)
	b.FailAfter(0)
	e = a.Close()
	want := "Memory leak detected for small object allocator 'a long borrowed static allocator identifier'. 1 bytes leaked\n"
	if e != nil || rt.Stats().PhysicalBytes != 0 || out.Stats().Position != uint64(len(want)) {
		t.Fatal("Go buffer diagnostic after destructor", e, rt.Stats(), out.Stats())
	}
	b.FailAfter(-1)
	text, e := out.String()
	sourceSmallObjectMust(t, e)
	if text != want {
		t.Fatal("diagnostic output", text)
	}
	sourceSmallObjectMust(t, a.Close())
}
func TestSmallObjectAllocatorVerboseUsesRealRuntimeAndLexicalLifetime(t *testing.T) {
	rt, _, _, out := smallObjectDiagnosticFixture(t)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, Verbose: out, Verbosity: 20})
	sourceSmallObjectMust(t, e)
	var refs [1021]utilSmallObjectRef
	for i := range refs {
		refs[i] = sourceSmallObjectAlloc(t, a, 1)
	}
	for _, r := range refs {
		sourceSmallObjectMust(t, a.Deallocate(1, r))
	}
	sourceSmallObjectMust(t, out.SetFloatMode(utext.NumericScientificFloat))
	sourceSmallObjectMust(t, out.SetPrecision(7))
	sourceSmallObjectMust(t, a.Consolidate())
	text, e := out.String()
	sourceSmallObjectMust(t, e)
	want := "(allocator-consolidate :wasted-size 8168 :memory 0.01)\n(end-allocator-consolidate :wasted-size 8168 :memory 0.01)\n"
	if text != want || out.State().FloatMode != utext.NumericFixedFloat || out.State().Precision != 2 || rt.State().GlobalSize != 8200 {
		t.Fatal("real counter/lexical scratch/end-stream state", text, out.State(), rt.State())
	}
	sourceSmallObjectMust(t, a.Close())
}
