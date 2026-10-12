package euclid

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanPublicSolverBorrowedManagerSolveNormalizeReset(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := NewSolver(m)
	euclidMust(t, e)
	borrowed, e := s.Manager()
	euclidMust(t, e)
	if borrowed != m {
		t.Fatal("solver did not borrow caller manager")
	}
	x, e := s.MkVar()
	euclidMust(t, e)
	j, e := s.MkJustification()
	euclidMust(t, e)
	if x != 0 || j != 0 {
		t.Fatal("source variable and justification IDs", x, j)
	}
	a, c := numeral.NewInt(1), numeral.NewInt(1)
	euclidMust(t, s.AssertEq([]numeral.Int{a}, []uint32{x}, &c, j))
	var out bytes.Buffer
	euclidMust(t, s.Display(&out))
	if out.String() != "solution set:\ntodo:\n1*j0 |= 1*x0 + 1 = 0\n" {
		t.Fatalf("pre-solve display = %q", out.String())
	}
	consistent, e := s.Solve()
	euclidMust(t, e)
	if !consistent {
		t.Fatal("unit equation is inconsistent")
	}
	parameter, e := s.IsParameter(x)
	euclidMust(t, e)
	if parameter {
		t.Fatal("caller variable became parameter")
	}
	aPrime, cPrime := numeral.NewInt(0), numeral.NewInt(0)
	two, three := numeral.NewInt(2), numeral.NewInt(3)
	var js []uint32
	euclidMust(t, s.Normalize([]numeral.Int{two}, []uint32{x}, &three, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "0", "1")
	if !slices.Equal(js, []uint32{j}) {
		t.Fatalf("query justifications = %v", js)
	}
	out.Reset()
	euclidMust(t, s.Display(&out))
	if out.String() != "solution set:\n-1*j0 |= -1*x0 + -1 = 0\ntodo:\n" {
		t.Fatalf("solved display = %q", out.String())
	}
	euclidMust(t, s.Reset())
	borrowed, e = s.Manager()
	euclidMust(t, e)
	if borrowed != m {
		t.Fatal("reset changed borrowed manager")
	}
	out.Reset()
	euclidMust(t, s.Display(&out))
	if out.String() != "solution set:\ntodo:\n" {
		t.Fatalf("reset display = %q", out.String())
	}
	x, e = s.MkVar()
	euclidMust(t, e)
	if x != 0 {
		t.Fatalf("reset first variable = %d", x)
	}
	euclidMust(t, s.Close())
	if e := m.CheckOwnerReady(); e != nil {
		t.Fatalf("solver closed borrowed manager: %v", e)
	}
	for _, value := range []*numeral.Int{&a, &c, &aPrime, &cPrime, &two, &three} {
		euclidMust(t, z.Del(value))
	}
}

func TestEuclideanPublicSolverOwnedManagerAndClosedGuards(t *testing.T) {
	s, e := NewSolver(nil)
	euclidMust(t, e)
	m, e := s.Manager()
	euclidMust(t, e)
	_, e = s.MkVar()
	euclidMust(t, e)
	euclidMust(t, s.Close())
	euclidMust(t, s.Close())
	if used := m.BudgetField.Used(); used != 0 {
		t.Fatalf("owned manager retained %d bytes", used)
	}
	if e := m.CheckOwnerReady(); e == nil {
		t.Fatal("owned numeral manager survived solver close")
	}
	if _, e := s.MkVar(); !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("closed solver MkVar = %v", e)
	}
}

func TestEuclideanPublicSolverContradictionExplanationAndDisplay(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := NewSolver(m)
	euclidMust(t, e)
	x, e := s.MkVar()
	euclidMust(t, e)
	j0, e := s.MkJustification()
	euclidMust(t, e)
	j1, e := s.MkJustification()
	euclidMust(t, e)
	a, zero, one := numeral.NewInt(1), numeral.NewInt(0), numeral.NewInt(1)
	euclidMust(t, s.AssertEq([]numeral.Int{a}, []uint32{x}, &zero, j0))
	euclidMust(t, s.AssertEq([]numeral.Int{a}, []uint32{x}, &one, j1))
	consistent, e := s.Solve()
	euclidMust(t, e)
	if consistent {
		t.Fatal("contradictory equations solved")
	}
	bad, e := s.Inconsistent()
	euclidMust(t, e)
	if !bad {
		t.Fatal("missing contradiction flag")
	}
	js, e := s.GetJustification()
	euclidMust(t, e)
	if !slices.Equal(js, []uint32{j0, j1}) {
		t.Fatalf("explanation IDs = %v", js)
	}
	js[0] = 99
	js, e = s.GetJustification()
	euclidMust(t, e)
	if !slices.Equal(js, []uint32{j0, j1}) {
		t.Fatal("caller changed solver explanation")
	}
	var out bytes.Buffer
	euclidMust(t, s.Display(&out))
	if out.String() != "inconsistent: 1*j0 -1*j1 |= -1 = 0\nsolution set:\n-1*j1 |= -1*x0 + -1 = 0\ntodo:\n1*j0 -1*j1 |= -1 = 0\n" {
		t.Fatalf("contradiction display = %q", out.String())
	}
	if e := s.Display(shortEuclideanWriter{}); !errors.Is(e, io.ErrShortWrite) {
		t.Fatalf("short writer result = %v", e)
	}
	euclidMust(t, s.Close())
	for _, value := range []*numeral.Int{&a, &zero, &one} {
		euclidMust(t, z.Del(value))
	}
}

func TestEuclideanPublicSolverResetAfterGuardedAssertionError(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := NewSolver(m)
	euclidMust(t, e)
	_, e = s.MkVar()
	euclidMust(t, e)
	one := numeral.NewInt(1)
	if e := s.AssertEq([]numeral.Int{one}, nil, &one, nullEquationIndex); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("mismatched assertion = %v", e)
	}
	if _, e := s.Solve(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("stopped solver Solve = %v", e)
	}
	euclidMust(t, s.Reset())
	x, e := s.MkVar()
	euclidMust(t, e)
	if x != 0 {
		t.Fatalf("reset variable ID = %d", x)
	}
	euclidMust(t, s.Close())
	euclidMust(t, m.Integers().Del(&one))
}

type shortEuclideanWriter struct{}

func (shortEuclideanWriter) Write([]byte) (int, error) { return 0, nil }
