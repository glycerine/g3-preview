// Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"
)

func TestSATSolverAssignConflictStatusSourceOrder(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 3)
	j := newSATBinaryJustification(satMakeLiteral(2, true))
	satOK(t, s.assign(satMakeLiteral(0, true), j))
	if satAssignmentTestValue(t, &v.assignment, 0) != int8(lFalse) || satAssignmentTestValue(t, &v.assignment, 1) != int8(lTrue) || !satJustificationTestValue(t, &v.justification, 0).None() || satScalarTestUint(t, &v.level, 0) != 0 || satAssignmentTestValue(t, &v.phase, 0) != 1 || satAssignmentTestValue(t, &v.assignedSinceGC, 0) != 1 {
		t.Fatal("base assignment")
	}
	v.scopeLevel = 3
	satOK(t, s.assign(satMakeLiteral(1, false), j))
	if satJustificationTestValue(t, &v.justification, 1) != j || satScalarTestUint(t, &v.level, 1) != 3 || satAssignmentTestValue(t, &v.phase, 1) != 0 {
		t.Fatal("scoped assignment")
	}
	c, e := s.clauses.MkClause([]satLiteral{satMakeLiteral(0, false), satMakeLiteral(2, false)}, false)
	satOK(t, e)
	z, e := s.status(c)
	satOK(t, e)
	if z != lUndef {
		t.Fatal(z)
	}
	satOK(t, s.assign(satMakeLiteral(2, true), j))
	z, e = s.status(c)
	satOK(t, e)
	if z != lFalse {
		t.Fatal(z)
	}
	satOK(t, s.assign(satMakeLiteral(0, false), j))
	if !s.inconsistent || s.conflict != j || s.notLiteral != satMakeLiteral(0, true) {
		t.Fatal("conflict")
	}
	s.setConflict(satJustification{}, satMakeLiteral(2, false))
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
	if satAssignmentTestValue(t, &v.assignment, 0) != int8(lTrue) || satScalarTestUint(t, &v.level, 0) != 1 || satAssignmentTestValue(t, &v.phase, 0) != 0 || satAssignmentTestValue(t, &v.assignedSinceGC, 0) != 1 {
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
	satOK(t, v.external.write(0, 1))
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
