// Copyright (c) 2006, 2007 Microsoft Corporation.
// Actual source diagnostic byte streams stay in main; core tests moved to memory.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func smallObjectMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func smallObjectAlloc(t *testing.T, a *utilSmallObjectAllocator, n uint64) utilSmallObjectRef {
	t.Helper()
	r, e := a.Allocate(n)
	smallObjectMust(t, e)
	return r
}
func smallObjectDiagnosticFixture(t *testing.T) (*utilMemoryRuntime, *arena.Budget, *arena.BytesRegion, *NumericTextStream) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	rt, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
	smallObjectMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
	smallObjectMust(t, e)
	stream, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	smallObjectMust(t, e)
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
func smallObjectID(t *testing.T, r *arena.BytesRegion, text string) GlobalParamCString {
	t.Helper()
	ref, e := r.Allocate(uint32(len(text) + 1))
	smallObjectMust(t, e)
	for i := range text {
		smallObjectMust(t, r.Write(ref, uint32(i), text[i]))
	}
	smallObjectMust(t, r.Write(ref, uint32(len(text)), 0))
	return GlobalParamCString{Origin: r.OwnerID(), Ref: ref, Present: true}
}
func TestSmallObjectAllocatorBorrowedIDMutationAndWholeCStringWidth(t *testing.T) {
	rt, _, region, out := smallObjectDiagnosticFixture(t)
	id := smallObjectID(t, region, "abc\x00ignored")
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, BorrowedID: region, IDRef: id, Diagnostics: out})
	smallObjectMust(t, e)
	_ = smallObjectAlloc(t, a, 1)
	smallObjectMust(t, region.Write(id.Ref, 0, 'z'))
	smallObjectMust(t, a.Close())
	if rt.Stats().PhysicalBytes != 0 {
		t.Fatal("diagnostic preceded chunk free")
	}
	text, e := out.String()
	smallObjectMust(t, e)
	if text != "Memory leak detected for small object allocator 'zbc'. 1 bytes leaked\n" {
		t.Fatal("borrowed mutable ID/NUL", text)
	}
	smallObjectMust(t, out.ResetText())
	smallObjectMust(t, out.SetWidth(8))
	smallObjectMust(t, smallObjectCStringInto(out, region, id))
	text, e = out.String()
	smallObjectMust(t, e)
	if text != "     zbc" || out.State().Width != 0 {
		t.Fatal("C-string inserted as per-character fields", text, out.State())
	}
}
func TestSmallObjectAllocatorNullStaleIDErrorPrefixes(t *testing.T) {
	for _, stale := range []bool{false, true} {
		rt, _, region, out := smallObjectDiagnosticFixture(t)
		id := GlobalParamCString{}
		if stale {
			id = smallObjectID(t, region, "old")
			region.Reset()
		}
		a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, BorrowedID: region, IDRef: id, Diagnostics: out})
		smallObjectMust(t, e)
		_ = smallObjectAlloc(t, a, 1)
		e = a.Close()
		if stale {
			if e != arena.ErrInvalidSlot {
				t.Fatal("stale spelling not checked at insertion", e)
			}
		} else if e != ErrParamsStreamFailure {
			t.Fatal("null ostream insertion adapter", e)
		}
		if rt.Stats().PhysicalBytes != 0 {
			t.Fatal("ID checked before destructor physical prefix")
		}
		text, e := out.String()
		smallObjectMust(t, e)
		if text != "Memory leak detected for small object allocator '" {
			t.Fatal("ID error prefix", stale, text)
		}
	}
}
func TestSmallObjectAllocatorDiagnosticOutputFailureAfterFree(t *testing.T) {
	rt, b, _, out := smallObjectDiagnosticFixture(t)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, ID: "a long borrowed static allocator identifier", Diagnostics: out})
	smallObjectMust(t, e)
	_ = smallObjectAlloc(t, a, 1)
	b.FailAfter(0)
	e = a.Close()
	if e != arena.ErrInjectedFailure || rt.Stats().PhysicalBytes != 0 || out.Stats().Position != 64 {
		t.Fatal("output refusal hid actual destructor prefix", e, rt.Stats(), out.Stats())
	}
	b.FailAfter(-1)
	text, e := out.String()
	smallObjectMust(t, e)
	want := "Memory leak detected for small object allocator 'a long borrowed static allocator identifier'. 1 bytes leaked\n"
	if text != want[:64] {
		t.Fatal("defined emitted byte prefix", text)
	}
	smallObjectMust(t, a.Close())
}
func TestSmallObjectAllocatorVerboseUsesRealRuntimeAndLexicalLifetime(t *testing.T) {
	rt, _, _, out := smallObjectDiagnosticFixture(t)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, Verbose: out, Verbosity: 20})
	smallObjectMust(t, e)
	var refs [1021]utilSmallObjectRef
	for i := range refs {
		refs[i] = smallObjectAlloc(t, a, 1)
	}
	for _, r := range refs {
		smallObjectMust(t, a.Deallocate(1, r))
	}
	smallObjectMust(t, out.SetFloatMode(NumericScientificFloat))
	smallObjectMust(t, out.SetPrecision(7))
	smallObjectMust(t, a.Consolidate())
	text, e := out.String()
	smallObjectMust(t, e)
	want := "(allocator-consolidate :wasted-size 8168 :memory 0.01)\n(end-allocator-consolidate :wasted-size 8168 :memory 0.02)\n"
	if text != want || out.State().FloatMode != NumericFixedFloat || out.State().Precision != 2 || rt.State().GlobalSize != 8200 {
		t.Fatal("real counter/lexical scratch/end-stream state", text, out.State(), rt.State())
	}
	smallObjectMust(t, a.Close())
}
