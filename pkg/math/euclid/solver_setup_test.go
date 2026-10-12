package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanPositionSourceBinarySearch(t *testing.T) {
	for _, tc := range []struct {
		xs   []uint32
		x    uint32
		want uint32
	}{
		{nil, 4, nullEquationIndex},
		{[]uint32{2, 7, 19}, 2, 0},
		{[]uint32{2, 7, 19}, 7, 1},
		{[]uint32{2, 7, 19}, 19, 2},
		{[]uint32{2, 7, 19}, 0, nullEquationIndex},
		{[]uint32{2, 7, 19}, 8, nullEquationIndex},
		{[]uint32{2, 7, 19}, 20, nullEquationIndex},
	} {
		if got := position(tc.xs, tc.x); got != tc.want {
			t.Fatalf("position(%v, %d) = %d, want %d", tc.xs, tc.x, got, tc.want)
		}
	}
}

func TestEuclideanSetupVariableJustificationAndEquationOwnership(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i, parameter := range []bool{false, true, false} {
		id, e := s.mkVar(parameter)
		euclidMust(t, e)
		if id != uint32(i) || s.parameter[i] != parameter || s.solved[i] != nullEquationIndex || len(s.occurrences[i]) != 0 || s.asBuffer.Size() != uint32(i+1) {
			t.Fatal("variable source publication", id, i, s.solved, s.parameter)
		}
	}
	for i := 0; i < 2; i++ {
		id, e := s.mkJustification()
		euclidMust(t, e)
		if id != uint32(i) || s.bsBuffer.Size() != uint32(i+1) {
			t.Fatal("justification source publication", id, i)
		}
	}
	z := m.Integers()
	as := []numeral.Int{{}, {}}
	euclidMust(t, z.SetString(&as[0], "123456789012345678901"))
	euclidMust(t, z.SetString(&as[1], "-11"))
	c := numeral.NewInt(5)
	b := []numeral.Rational{numeral.NewRational()}
	euclidMust(t, m.SetString(&b[0], "17/23"))
	eq, e := s.addEquation(as, []uint32{2, 0}, &c, b, []uint32{1}, true)
	euclidMust(t, e)
	if len(s.equations) != 1 || s.equations[0] != eq || eq.xs[0] != 0 || eq.xs[1] != 2 || eq.js[0] != 1 || position(eq.xs, 2) != 1 {
		t.Fatal("solver equation publication", eq.xs, eq.js)
	}
	if got, e := z.String(&eq.as[0]); e != nil || got != "-11" {
		t.Fatal("sorted owned equation coefficient", got, e)
	}
	second, e := s.addEquation([]numeral.Int{numeral.NewInt(1)}, []uint32{1}, &c, nil, nil, false)
	euclidMust(t, e)
	if len(s.equations) != 2 || s.equations[1] != second {
		t.Fatal("second owned equation")
	}
	euclidMust(t, s.Close())
	euclidMust(t, s.Close())
	if !eq.closed || !second.closed || !s.closed {
		t.Fatal("solver owner missed equation cleanup")
	}
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	euclidMust(t, z.Del(&c))
	euclidMust(t, m.Del(&b[0]))
}

func TestEuclideanSetupHeaderRefusalBeforeBuffers(t *testing.T) {
	m := euclidNumeralManager(t)
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	s, e := newSolverSetup(m)
	if s != nil || e == nil || m.BudgetField.Used() != before {
		t.Fatal("constructor header refusal", s, e)
	}
	m.BudgetField.FailAfter(-1)
}

func TestEuclideanSetupSolvedVariableQueue(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	if s.queue.Bounds() != 16 {
		t.Fatalf("initial source queue bound = %d", s.queue.Bounds())
	}
	for i := 0; i < 17; i++ {
		id, e := s.mkVar(false)
		euclidMust(t, e)
		if id != uint32(i) || s.queue.Bounds() < int32(i+1) {
			t.Fatalf("variable %d queue bound = %d", id, s.queue.Bounds())
		}
	}
	s.solved[0], s.solved[1], s.solved[2] = 5, 2, 9
	euclidMust(t, s.scheduleVarSubst([]uint32{2, 0, 3, 1}))
	if !s.queue.CheckInvariant() || s.queue.Contains(3) {
		t.Fatal("queue accepted an unsolved variable or lost its order")
	}
	for _, want := range []int32{1, 0, 2} {
		got, e := s.queue.EraseMin()
		euclidMust(t, e)
		if got != want {
			t.Fatalf("erase minimum = %d, want %d", got, want)
		}
	}
	if !s.queue.Empty() {
		t.Fatal("scheduled variables remain")
	}
	euclidMust(t, s.Close())
}
