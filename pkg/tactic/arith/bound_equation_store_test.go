package arith

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestBoundEquationStoreCloseResumesReverseTrailAfterStoppedRelease(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	for x := uint32(0); x < 5; x++ {
		equationMust(t, s.MkVar(x, false))
	}
	coefficients := []numeral.Int{numeral.NewInt(1), numeral.NewInt(-1)}
	defer func() {
		for i := range coefficients {
			equationMust(t, m.Integers().Del(&coefficients[i]))
		}
	}()
	index, err := s.MkIntEquation(coefficients, []uint32{3, 4})
	equationMust(t, err)
	eq := s.constraints[index].eq
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	for x, number := range []string{
		"12345678901234567890123456789012345678901",
		"12345678901234567890123456789012345678902",
		"12345678901234567890123456789012345678903",
	} {
		equationMust(t, m.SetString(&value, number))
		var accepted bool
		if x == 1 {
			accepted, err = s.AssertUpper(uint32(x), &value, false)
		} else {
			accepted, err = s.AssertLower(uint32(x), &value, false)
		}
		equationMust(t, err)
		if !accepted {
			t.Fatal("fixture bound was not published", x)
		}
	}
	first := s.trail[0]
	released := s.trail[2]
	original := first.value.Num.Cell
	invalid := ^numeral.NumericCellRef(0)
	if original == 0 || released.value.Num.Cell == 0 {
		t.Fatal("fixture did not own physical numeral cells")
	}
	first.value.Num.Cell = invalid
	defer func() {
		if first.value.Num.Cell == invalid {
			first.value.Num.Cell = original
		}
		equationMust(t, s.Close())
	}()
	if err = s.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("stopped oldest bound release: %v", err)
	}
	if s.closeTrailRemaining != 1 || s.trail[1] != nil || s.trail[2] != nil ||
		s.variables[0].lower != nil || s.variables[1].upper != nil || s.variables[2].lower != nil || eq.closed {
		t.Fatal("reverse trail cleanup prefix was not retained")
	}
	if _, _, _, err = s.Lower(0, &value); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("partly closed store allowed a bound read: %v", err)
	}
	first.value.Num.Cell = original
	released.value.Num.Cell = invalid
	equationMust(t, s.Close())
	if !eq.closed {
		t.Fatal("constraint was not released after the trail completed")
	}
}

func TestBoundEquationStoreCloseResumesConstraintRelease(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	for x := uint32(0); x < 4; x++ {
		equationMust(t, s.MkVar(x, false))
	}
	makeEquation := func(x uint32) *LinearEquation {
		t.Helper()
		as := []numeral.Int{numeral.NewInt(0), numeral.NewInt(0)}
		for i, value := range []string{
			"12345678901234567890123456789012345678901",
			"12345678901234567890123456789012345678902",
		} {
			equationMust(t, m.Integers().SetString(&as[i], value))
		}
		index, e := s.MkIntEquation(as, []uint32{x, x + 1})
		equationMust(t, e)
		for i := range as {
			equationMust(t, m.Integers().Del(&as[i]))
		}
		return s.constraints[index].eq
	}
	first := makeEquation(0)
	second := makeEquation(2)
	original := second.as[0].Cell
	invalid := ^numeral.NumericCellRef(0)
	if original == 0 {
		t.Fatal("fixture did not own a physical coefficient cell")
	}
	second.as[0].Cell = invalid
	if err = s.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("stopped second constraint release: %v", err)
	}
	if !first.closed || second.closed || s.deleteIndex != 1 || s.closed {
		t.Fatal("constraint cleanup did not retain its completed prefix")
	}
	second.as[0].Cell = original
	equationMust(t, s.Close())
	if !second.closed || !s.closed {
		t.Fatal("constraint cleanup did not complete on retry")
	}
}

func TestBoundEquationStoreVariablesWatchesAndConstraintOwnership(t *testing.T) {
	m, _ := equationManager(t)
	z := m.Integers()
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	for _, x := range []uint32{0, 2, 5} {
		equationMust(t, s.MkVar(x, x == 0))
	}
	as := []numeral.Int{numeral.NewInt(6), numeral.NewInt(-4), numeral.NewInt(10)}
	xs := []uint32{5, 0, 2}
	index, err := s.MkIntEquation(as, xs)
	equationMust(t, err)
	if index != 0 {
		t.Fatalf("first constraint index = %d", index)
	}
	eq, dead, err := s.Constraint(index)
	equationMust(t, err)
	if dead || eq == nil {
		t.Fatal("new equation should be live")
	}
	var out bytes.Buffer
	equationMust(t, s.equations.Display(&out, eq))
	if got := out.String(); got != "-2*x0 + 5*x2 + 3*x5 = 0" {
		t.Fatalf("stored equation = %q", got)
	}
	for _, x := range []uint32{0, 2, 5} {
		w, err := s.Watches(x)
		equationMust(t, err)
		if !reflect.DeepEqual(w, []uint32{0}) {
			t.Fatalf("watches for %d = %v", x, w)
		}
	}
	equationMust(t, s.DelVar(2))
	_, dead, err = s.Constraint(index)
	equationMust(t, err)
	if !dead {
		t.Fatal("deleting a watched variable must kill its constraint")
	}
	equationMust(t, s.MkVar(2, false))
	w, err := s.Watches(2)
	equationMust(t, err)
	if len(w) != 0 {
		t.Fatal("reactivation retained old watches", w)
	}
	equationMust(t, s.DeleteConstraints())
	for _, x := range []uint32{0, 2, 5} {
		w, err := s.Watches(x)
		equationMust(t, err)
		if len(w) != 0 {
			t.Fatalf("deleted constraints left watch for %d: %v", x, w)
		}
	}
	for i := range as {
		equationMust(t, z.Del(&as[i]))
	}
}

func TestBoundEquationStoreRationalConstructionAndCancellation(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	for _, x := range []uint32{1, 3, 8} {
		equationMust(t, s.MkVar(x, false))
	}
	ras := []numeral.Rational{numeral.NewRational(), numeral.NewRational(), numeral.NewRational()}
	for i, value := range []string{"1/2", "-1/2", "2/3"} {
		equationMust(t, m.SetString(&ras[i], value))
	}
	ids := []uint32{1, 1, 8}
	index, err := s.MkRationalEquation(ras, ids)
	equationMust(t, err)
	if index != noEquationPosition {
		t.Fatalf("one surviving variable should not create a constraint: %d", index)
	}
	for i := range ras {
		equationMust(t, m.Del(&ras[i]))
	}
	for i, value := range []string{"1/2", "-1/3", "1/6"} {
		equationMust(t, m.SetString(&ras[i], value))
	}
	ids = []uint32{8, 1, 3}
	index, err = s.MkRationalEquation(ras, ids)
	equationMust(t, err)
	eq, dead, err := s.Constraint(index)
	equationMust(t, err)
	if eq == nil || dead {
		t.Fatal("rational equation was not retained")
	}
	var out bytes.Buffer
	equationMust(t, s.equations.Display(&out, eq))
	if got := out.String(); got != "-2*x1 + 1*x3 + 3*x8 = 0" {
		t.Fatalf("rational equation = %q", got)
	}
	for i := range ras {
		equationMust(t, m.Del(&ras[i]))
	}
}
