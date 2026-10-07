// Source: src/sat/sat_types.h:153–289; src/sat/sat_mus.cpp:132–209.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATMUSLiteralSetActualCopyIntersectionUnionAndCapturedIteration(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var a, b satMUSLiteralSet
	a.bind(s.runtime.memory)
	b.bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, b.close()); satOK(t, a.close()) })
	for _, l := range []Literal{6, 0, 4, 2, 6} {
		satOK(t, a.insert(l))
	}
	for _, l := range []Literal{4, 8, 0} {
		satOK(t, b.insert(l))
	}
	satOK(t, a.intersect(&b))
	n, e := a.size()
	satOK(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	for i, want := range []uint32{0, 4} {
		got, e := a.values.Read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satOK(t, a.union(&b))
	for i, want := range []uint32{0, 4, 8} {
		got, e := a.values.Read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satOK(t, b.assign(&a))
	satOK(t, a.reset())
	for i, want := range []uint32{0, 4, 8} {
		got, e := b.values.Read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal("copy owns distinct storage", got, want)
		}
	}
}
func TestSATSolverMUSCapturedRuntimeAssumptionsUseActualPointer(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{1, 3}))
	var assumptions RuntimeUintColumn
	assumptions.Bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, assumptions.Finalize()) })
	satOK(t, assumptions.Push(0))
	satOK(t, assumptions.Push(2))
	r, e := s.checkSpan(2, satConflictRuntimeSpan(&assumptions))
	satOK(t, e)
	if r != util.LFalse {
		t.Fatal(r)
	}
}

// copy_core multiplies sizeof(T) (size_t), unlike expand_vector's explicitly
// unsigned32 byte-extent temporaries. No enormous allocation is performed.
func TestSATMUSCopyCoreRequestUsesSourceSizeTPromotion(t *testing.T) {
	for _, test := range []struct {
		capacity, stride uint32
		request          uint64
	}{
		{0, 4, 8}, {2, 4, 16}, {^uint32(0), 1, 4294967303}, {^uint32(0), 4, 17179869188}, {1 << 30, 4, 4294967304},
	} {
		got := satMUSCopyRequest(test.capacity, test.stride)
		if got != test.request {
			t.Fatal(test, got)
		}
	}
}

// The oversized capacity header is an explicit malformed source-domain fixture.
// size==0 means copy_core never dereferences the missing huge payload. Actual
// SIZE_T32 arithmetic wraps the request to8 and publishes both source headers;
// this is not a valid large-polynomial or ordinary OOM claim.
func TestSATMUSCopyCoreActualSizeT32WrapPublishesEmptyHeaderPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 32})
	satOK(t, e)
	var src, dst RuntimeVectorColumn
	src.Bind(r)
	dst.Bind(r)
	t.Cleanup(func() {
		satOK(t, dst.Finalize())
		satOK(t, src.Finalize())
		satOK(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("actual graph leak", b.Used())
		}
	})
	src.Block, e = r.Allocate(8)
	satOK(t, e)
	satOK(t, r.WriteUint32(src.Block, 0, 0x40000000))
	satOK(t, r.WriteUint32(src.Block, 4, 0))
	before := r.State()
	e = satMUSCopyColumn(&dst, &src, 4)
	if e != nil {
		t.Fatal("source32 request wraps to8 before Runtime entry", e)
	}
	cap, e := dst.Capacity()
	satOK(t, e)
	size, e := dst.Size()
	satOK(t, e)
	if cap != 0x40000000 || size != 0 || r.Stats().LiveBlocks != 2 {
		t.Fatal(cap, size, r.Stats())
	}
	after := r.State()
	if after.GlobalSize-before.GlobalSize != 12 {
		t.Fatal("actual8request plus4mallocheader", before, after)
	}
	for _, test := range []struct {
		c, stride uint32
		want      uint64
	}{{0x40000000, 4, 8}, {^uint32(0), 1, 7}, {^uint32(0), 4, 4}} {
		if got := satMUSCopyRequestFor(test.c, test.stride, 32); got != test.want {
			t.Fatal(test, got)
		}
	}
}
