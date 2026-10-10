// Source: src/util/util.cpp:22-40; src/util/util.h:176-186;
// src/util/small_object_allocator.cpp:172-242.
package util

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestVerboseContextMutableAllocatorSelection(t *testing.T) {
	rt, b, _, out := smallObjectDiagnosticFixture(t)
	c, e := NewVerboseContext(out, b)
	smallObjectMust(t, e)
	defer c.Close()
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, VerboseContext: c})
	smallObjectMust(t, e)
	defer a.Close()
	smallObjectMust(t, c.SetVerbosityLevel(19))
	smallObjectMust(t, a.Consolidate())
	text, e := out.String()
	smallObjectMust(t, e)
	if text != "" {
		t.Fatal(text)
	}
	smallObjectMust(t, c.SetVerbosityLevel(20))
	smallObjectMust(t, a.Consolidate())
	text, e = out.String()
	smallObjectMust(t, e)
	if text != "(allocator-consolidate :wasted-size 0 :memory 0.00)\n(end-allocator-consolidate :wasted-size 0 :memory 0.00)\n" {
		t.Fatal(text)
	}
	second, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	smallObjectMust(t, e)
	defer second.Close()
	smallObjectMust(t, c.SetVerboseStream(second))
	smallObjectMust(t, c.SetVerbosityLevel(^uint32(0)))
	smallObjectMust(t, a.Consolidate())
	got, e := second.String()
	smallObjectMust(t, e)
	if got != text {
		t.Fatal(got, text)
	}
	still, e := out.String()
	smallObjectMust(t, e)
	if still != text {
		t.Fatal("rerouted old output", still)
	}
}
func TestVerboseContextBorrowedPointerAndLifetime(t *testing.T) {
	_, b, _, out := smallObjectDiagnosticFixture(t)
	c, e := NewVerboseContext(out, b)
	smallObjectMust(t, e)
	n, e := c.VerbosityLevel()
	smallObjectMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	copy := *c
	if _, e = copy.VerbosityLevel(); !errors.Is(e, ErrVerboseContextOwner) {
		t.Fatal(e)
	}
	smallObjectMust(t, out.Close())
	smallObjectMust(t, c.SetVerboseStream(out))
	selected, e := c.VerboseStream()
	smallObjectMust(t, e)
	if selected != out {
		t.Fatal("pointer altered")
	}
	if e = selected.AppendByte('x'); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("insertion must reach borrowed lifetime", e)
	}
	smallObjectMust(t, c.Close())
	if _, e = c.VerboseStream(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestVerboseContextColdRefusalAndExactCharge(t *testing.T) {
	_, b, _, out := smallObjectDiagnosticFixture(t)
	before := b.Used()
	b.FailAfter(0)
	c, e := NewVerboseContext(out, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || c != nil || b.Used() != before {
		t.Fatal(c, e, b.Used(), before)
	}
	b.FailAfter(-1)
	c, e = NewVerboseContext(out, b)
	smallObjectMust(t, e)
	if b.Used() != before+uint64(reflect.TypeOf(VerboseContext{}).Size()) {
		t.Fatal("cold owner charge", b.Used(), before)
	}
	smallObjectMust(t, c.Close())
	if b.Used() != before {
		t.Fatal("cold refund")
	}
}

func verboseContextPhysical(t *testing.T, rt *umemory.Runtime, b *arena.Budget, region *arena.BytesRegion, out *utext.NumericTextStream, c *VerboseContext, a *utilSmallObjectAllocator) {
	t.Helper()
	rs := region.Stats()
	ds := rs.Descriptors
	ss := out.Stats()
	want := rt.Stats().TotalBytes + rs.OwnerBytes + rs.PayloadBytes + rs.PageHeaderBytes + rs.DirectoryBytes + rs.LogBytes + rs.ScopeBytes + ds.OwnerBytes + ds.RecordBytes + ds.MetadataBytes + ds.DirectoryBytes + ss.OwnerBytes + ss.ScratchBytes + ss.OutputOwnerBytes + ss.OutputOverflowBytes + c.ownerBytes
	if a != nil {
		want += a.Stats().TotalBytes
	}
	if want != b.Used() {
		t.Fatal("once-only named physical/cold graph", want, b.Used())
	}
}
func TestVerboseContextAllocatorEveryOutputRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		t.Run(string(rune('A'+ordinal)), func(t *testing.T) {
			rt, b, region, out := smallObjectDiagnosticFixture(t)
			c, e := NewVerboseContext(out, b)
			smallObjectMust(t, e)
			defer c.Close()
			smallObjectMust(t, c.SetVerbosityLevel(20))
			a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, VerboseContext: c})
			smallObjectMust(t, e)
			defer a.Close()
			before := rt.State()
			b.FailAfter(ordinal)
			e = a.Consolidate()
			verboseContextPhysical(t, rt, b, region, out, c, a)
			if rt.State() != before || rt.Stats().LiveBlocks != 0 {
				t.Fatal("stream cold growth fabricated Runtime event")
			}
			if e != nil {
				failures++
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(ordinal, e)
				}
			} else {
				success++
			}
			b.FailAfter(-1)
			got, e := out.String()
			smallObjectMust(t, e)
			want := "(allocator-consolidate :wasted-size 0 :memory 0.00)\n(end-allocator-consolidate :wasted-size 0 :memory 0.00)\n"
			if len(got) > len(want) || got != want[:len(got)] {
				t.Fatal("source completed stream prefix", ordinal, got)
			}
			if success != 0 && got != want {
				t.Fatal("successful endpoint", got)
			}
		})
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal("actual refusal horizon", failures, success)
	}
	t.Log("refusals", failures, "first success", failures)
}
func TestVerboseContextClosedLevelReachedBeforeOutput(t *testing.T) {
	rt, b, region, out := smallObjectDiagnosticFixture(t)
	c, e := NewVerboseContext(out, b)
	smallObjectMust(t, e)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, VerboseContext: c})
	smallObjectMust(t, e)
	defer a.Close()
	smallObjectMust(t, c.Close())
	before := rt.State()
	used := b.Used()
	e = a.Consolidate()
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	verboseContextPhysical(t, rt, b, region, out, c, a)
	if before != rt.State() || used != b.Used() {
		t.Fatal("closed level getter performed later action")
	}
	got, e := out.String()
	smallObjectMust(t, e)
	if got != "" {
		t.Fatal(got)
	}
}

func TestVerboseContextWarmOutputAndNullAdapter(t *testing.T) {
	rt, b, region, out := smallObjectDiagnosticFixture(t)
	c, e := NewVerboseContext(out, b)
	smallObjectMust(t, e)
	defer c.Close()
	if e = c.SetVerboseStream(nil); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("null source-reference adapter", e)
	}
	selected, e := c.VerboseStream()
	smallObjectMust(t, e)
	if selected != out {
		t.Fatal("null setter changed pointer")
	}
	smallObjectMust(t, c.SetVerbosityLevel(20))
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64}, VerboseContext: c})
	smallObjectMust(t, e)
	defer a.Close()
	smallObjectMust(t, a.Consolidate())
	smallObjectMust(t, out.ResetText())
	used := b.Used()
	state := rt.State()
	b.FailAfter(0)
	e = a.Consolidate()
	verboseContextPhysical(t, rt, b, region, out, c, a)
	if e != nil || used != b.Used() || rt.State() != state {
		t.Fatal("warmed chain reserved cold or Runtime storage", e)
	}
	b.FailAfter(-1)
	if out.State().Precision != 2 || out.State().FloatMode != utext.NumericFixedFloat {
		t.Fatal("source persistent manipulators", out.State())
	}
}
