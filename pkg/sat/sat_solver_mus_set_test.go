// Source: src/sat/sat_types.h:153–289; src/sat/sat_mus.cpp:132–209.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"
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

func TestSATSolverMUSCapturedNativeAssumptions(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{1, 3}))
	var assumptions satUintColumn
	assumptions.Bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, assumptions.Finalize()) })
	satOK(t, assumptions.Push(0))
	satOK(t, assumptions.Push(2))
	r, e := s.checkSpan(2, satConflictColumnSpan(&assumptions))
	satOK(t, e)
	if r != lbool.False {
		t.Fatal(r)
	}
}
