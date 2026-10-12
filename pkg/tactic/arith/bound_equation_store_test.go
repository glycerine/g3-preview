package arith

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

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
