package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSATSolverAssignConflictStatusSourceOrder(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 3)
	j := newSATBinaryJustification(MakeLiteral(2, true))
	satOK(t, s.assign(MakeLiteral(0, true), j))
	if v.assignment[0] != lbool.False || v.assignment[1] != lbool.True || !satJustificationTestValue(t, &v.justification, 0).None() || v.level[0] != 0 || v.phase[0] != 1 || !v.assignedSinceGC[0] {
		t.Fatal("base assignment")
	}
	v.scopeLevel = 3
	satOK(t, s.assign(MakeLiteral(1, false), j))
	if satJustificationTestValue(t, &v.justification, 1) != j || v.level[1] != 3 || v.phase[1] != 0 {
		t.Fatal("scoped assignment")
	}
	c, e := s.clauses.MkClause([]Literal{MakeLiteral(0, false), MakeLiteral(2, false)}, false)
	satOK(t, e)
	z, e := s.status(c)
	satOK(t, e)
	if z != lbool.Undef {
		t.Fatal(z)
	}
	satOK(t, s.assign(MakeLiteral(2, true), j))
	z, e = s.status(c)
	satOK(t, e)
	if z != lbool.False {
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
func TestSATSolverAssignMissingExtensionAfterActualTrail(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 1)
	satOK(t, v.setExternal(BoolVar(0), true))
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
