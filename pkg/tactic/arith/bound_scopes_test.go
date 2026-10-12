package arith

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestBoundEquationNestedScopesRestoreExactBoundsAndConflict(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, true))
	equationMust(t, s.MkVar(1, true))
	input := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&input)) }()
	setLower := func(v int32) {
		t.Helper()
		equationMust(t, m.SetInt32(&input, v))
		accepted, err := s.AssertLower(0, &input, false)
		equationMust(t, err)
		if !accepted {
			t.Fatalf("lower %d rejected", v)
		}
	}
	setLower(2)
	equationMust(t, m.SetInt32(&input, 4))
	_, err = s.AssertUpper(0, &input, false)
	equationMust(t, err)
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	ids := []uint32{0, 1}
	_, err = s.MkIntEquation(coeff, ids)
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	equationMust(t, s.Propagate())
	baseTrail, baseQhead, baseTimestamp := len(s.trail), s.qhead, s.timestamp
	if baseTrail != 4 {
		t.Fatalf("base trail = %d", baseTrail)
	}

	equationMust(t, s.Push())
	setLower(3)
	equationMust(t, s.Propagate())
	if level, err := s.ScopeLevel(); err != nil || level != 1 {
		t.Fatalf("level = %d, %v", level, err)
	}
	if len(s.trail) != 6 {
		t.Fatalf("first scope trail = %d", len(s.trail))
	}
	if !s.variables[1].upper.derived || s.variables[1].upper.level != 1 {
		t.Fatal("scoped derived bound lost its source level")
	}
	equationMust(t, s.Push())
	setLower(5) // conflicts with the old upper bound of 4
	conflict, err := s.ConflictVar()
	equationMust(t, err)
	if conflict != 0 {
		t.Fatalf("nested conflict = %d", conflict)
	}
	equationMust(t, s.Pop(1))
	conflict, err = s.ConflictVar()
	equationMust(t, err)
	if conflict != noEquationPosition || len(s.trail) != 6 {
		t.Fatalf("first pop conflict=%d trail=%d", conflict, len(s.trail))
	}
	out := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&out)) }()
	_, _, _, err = s.Upper(1, &out)
	equationMust(t, err)
	got, err := m.String(&out)
	equationMust(t, err)
	if got != "-3" {
		t.Fatalf("first pop x1 upper = %s", got)
	}
	equationMust(t, s.Pop(1))
	_, _, _, err = s.Upper(1, &out)
	equationMust(t, err)
	got, err = m.String(&out)
	equationMust(t, err)
	if got != "-2" || len(s.trail) != baseTrail || s.qhead != baseQhead || s.timestamp != baseTimestamp {
		t.Fatalf("base state not restored: upper=%s trail=%d qhead=%d ts=%d", got, len(s.trail), s.qhead, s.timestamp)
	}
	if level, err := s.ScopeLevel(); err != nil || level != 0 {
		t.Fatalf("final level = %d, %v", level, err)
	}
}

func TestBoundEquationScopedInsertionReinitializesAfterPop(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, true))
	equationMust(t, s.MkVar(1, true))
	input := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&input)) }()
	for _, tc := range []struct {
		lower bool
		value int32
	}{{true, 2}, {false, 4}} {
		equationMust(t, m.SetInt32(&input, tc.value))
		if tc.lower {
			_, err = s.AssertLower(0, &input, false)
		} else {
			_, err = s.AssertUpper(0, &input, false)
		}
		equationMust(t, err)
	}
	equationMust(t, s.Push())
	equationMust(t, s.Push())
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	ids := []uint32{0, 1}
	index, err := s.MkIntEquation(coeff, ids)
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	if len(s.reinitStack) != 1 || s.reinitStack[0] != index || len(s.trail) != 4 {
		t.Fatalf("scoped insertion reinit=%v trail=%d", s.reinitStack, len(s.trail))
	}
	if err := s.DeleteConstraints(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("scoped DeleteConstraints = %v", err)
	}
	equationMust(t, s.Pop(1))
	if len(s.scopes) != 1 || len(s.reinitStack) != 1 || len(s.trail) != 4 ||
		!s.variables[1].lower.derived || s.variables[1].lower.level != 1 {
		t.Fatalf("nested pop reinitialization scopes=%d reinit=%v trail=%d", len(s.scopes), s.reinitStack, len(s.trail))
	}
	equationMust(t, s.Pop(1))
	if len(s.scopes) != 0 || len(s.reinitStack) != 0 || len(s.trail) != 4 {
		t.Fatalf("pop reinitialization scopes=%d reinit=%v trail=%d", len(s.scopes), s.reinitStack, len(s.trail))
	}
	if s.variables[1].lower.derived || s.variables[1].upper.derived {
		t.Fatal("level-zero reinitialization kept scoped derivation kind")
	}
	out := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&out)) }()
	_, _, _, err = s.Lower(1, &out)
	equationMust(t, err)
	got, err := m.String(&out)
	equationMust(t, err)
	if got != "-4" {
		t.Fatalf("reinitialized lower = %s", got)
	}
	equationMust(t, s.DeleteConstraints())
}
