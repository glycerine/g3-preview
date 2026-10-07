package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverAssignConflictStatusSourceOrder(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 3)
	j := newSATBinaryJustification(MakeLiteral(2, true))
	satOK(t, s.assign(MakeLiteral(0, true), j))
	if satAssignmentTestValue(t, &v.assignment, 0) != int8(util.LFalse) || satAssignmentTestValue(t, &v.assignment, 1) != int8(util.LTrue) || !satJustificationTestValue(t, &v.justification, 0).None() || satScalarTestUint(t, &v.level, 0) != 0 || satAssignmentTestValue(t, &v.phase, 0) != 1 || satAssignmentTestValue(t, &v.assignedSinceGC, 0) != 1 {
		t.Fatal("base assignment")
	}
	v.scopeLevel = 3
	satOK(t, s.assign(MakeLiteral(1, false), j))
	if satJustificationTestValue(t, &v.justification, 1) != j || satScalarTestUint(t, &v.level, 1) != 3 || satAssignmentTestValue(t, &v.phase, 1) != 0 {
		t.Fatal("scoped assignment")
	}
	c, e := s.clauses.MkClause([]Literal{MakeLiteral(0, false), MakeLiteral(2, false)}, false)
	satOK(t, e)
	z, e := s.status(c)
	satOK(t, e)
	if z != util.LUndef {
		t.Fatal(z)
	}
	satOK(t, s.assign(MakeLiteral(2, true), j))
	z, e = s.status(c)
	satOK(t, e)
	if z != util.LFalse {
		t.Fatal(z)
	}
	satOK(t, s.assign(MakeLiteral(0, false), j))
	if !s.inconsistent || s.conflict != j || s.notLiteral != MakeLiteral(0, true) {
		t.Fatal("conflict")
	}
	s.setConflict(satJustification{}, MakeLiteral(2, false))
	if s.conflict != j {
		t.Fatal("first conflict retained")
	}
}
func TestSATSolverAssignTrailRefusalAndMissingExtensionPrefix(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 1
	b.FailAfter(0)
	e := s.assignCore(0, satJustification{})
	if !satAssignmentActualRefusal(v.memory, e) {
		t.Fatal("actual initial trail refusal", e)
	}
	if satAssignmentTestValue(t, &v.assignment, 0) != int8(util.LTrue) || satScalarTestUint(t, &v.level, 0) != 1 || satAssignmentTestValue(t, &v.phase, 0) != 0 || satAssignmentTestValue(t, &v.assignedSinceGC, 0) != 1 {
		t.Fatal("scalar prefix")
	}
	n, x := v.trail.Size(v.trailRef)
	satOK(t, x)
	if n != 0 {
		t.Fatal(n)
	}
	satPropagationAssertGraph(t, s)
	b.FailAfter(-1)
}
func TestSATSolverAssignMissingExtensionAfterActualTrail(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	satOK(t, v.external.Write(0, 1))
	s.extensionPresent = true
	e := s.assignCore(0, satJustification{})
	if !errors.Is(e, ErrSATMissingSourceExtBinding) {
		t.Fatal(e)
	}
	n, x := v.trail.Size(v.trailRef)
	satOK(t, x)
	if n != 1 {
		t.Fatal(n)
	}
}
