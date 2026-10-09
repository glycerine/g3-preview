// Source: src/sat/sat_types.h:153–289; src/sat/sat_mus.cpp:132–209.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATMUSLiteralSetActualCopyIntersectionUnionAndCapturedIteration(t *testing.T) {
	var a, b satMUSLiteralSet
	for _, l := range []Literal{6, 0, 4, 2, 6} {
		satOK(t, a.insert(l))
	}
	for _, l := range []Literal{4, 8, 0} {
		satOK(t, b.insert(l))
	}
	a.intersect(&b)
	if !reflect.DeepEqual(a.values, []Literal{0, 4}) {
		t.Fatal(a.values)
	}
	satOK(t, a.union(&b))
	if !reflect.DeepEqual(a.values, []Literal{0, 4, 8}) {
		t.Fatal(a.values)
	}
	b.assign(&a)
	a.reset()
	satOK(t, a.insert(2))
	if !reflect.DeepEqual(b.values, []Literal{0, 4, 8}) || b.contains(2) {
		t.Fatal("copy aliases source", b.values)
	}
	b.assign(&b)
	b.intersect(&b)
	satOK(t, b.union(&b))
	if !reflect.DeepEqual(b.values, []Literal{0, 4, 8}) {
		t.Fatal("self operations changed order", b.values)
	}
	satOK(t, satConsequenceSetRemove(&b, 0))
	if !reflect.DeepEqual(b.values, []Literal{8, 4}) || b.contains(0) {
		t.Fatal("remove must swap last", b.values)
	}
	a.close()
	b.close()
}
func TestSATMUSLiteralSetNativeGrowthAndCopy(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	before, used := s.runtime.memory.Stats(), s.budget.Used()
	s.budget.FailAfter(0)
	defer s.budget.FailAfter(-1)
	var a, b, copy satMUSLiteralSet
	for i := Literal(0); i < 1024; i++ {
		satOK(t, a.insert(i))
		if i&1 != 0 {
			satOK(t, b.insert(i))
		}
	}
	copy.assign(&a)
	a.intersect(&b)
	if a.size() != 512 || copy.size() != 1024 || a.contains(0) || !copy.contains(0) {
		t.Fatal("independent copy/intersection")
	}
	satOK(t, a.union(&copy))
	if a.size() != 1024 || a.values[0] != 1 || a.values[511] != 1023 || a.values[512] != 0 || a.values[1023] != 1022 {
		t.Fatal("union insertion order")
	}
	a.close()
	b.close()
	copy.close()
	if s.runtime.memory.Stats() != before || s.budget.Used() != used {
		t.Fatal("native set operation acquired Runtime storage")
	}
	satFullSolverRuntimeAudit(t, s)
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
	if r != lbool.False {
		t.Fatal(r)
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
	e = copyRuntimeVectorColumn(&dst, &src, 4)
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
}
