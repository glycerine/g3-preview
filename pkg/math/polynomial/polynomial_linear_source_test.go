// Source: src/test/polynomial.cpp:1712-1739; src/math/polynomial/linear_eq_solver.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The source driver returns after tst_psc, so this fixture is dormant there.
// Port its VERIFY and three ENSURE sites explicitly without active-driver credit.
func TestLinearRationalOriginalPolynomialFixture(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(3))
	rows := [3][3]int64{{2, 1, -1}, {-3, -1, 2}, {-2, 1, 2}}
	rhs := [3]int64{8, -11, -3}
	for i := range rows {
		as := [3]util.Rational{}
		for j := range as {
			as[j] = util.NewRational()
			serviceMust(t, q.SetInt64(&as[j], rows[i][j]))
		}
		b := util.NewRational()
		serviceMust(t, q.SetInt64(&b, rhs[i]))
		serviceMust(t, s.Add(uint32(i), as[:], &b))
		for j := range as {
			serviceMust(t, q.Del(&as[j]))
		}
		serviceMust(t, q.Del(&b))
	}
	xs := [3]util.Rational{util.NewRational(), util.NewRational(), util.NewRational()}
	ok, e := s.Solve(xs[:])
	serviceMust(t, e)
	if !ok {
		t.Fatal("source VERIFY solve")
	}
	for i, want := range [3]int64{2, 3, -1} {
		r := util.NewRational()
		serviceMust(t, q.SetInt64(&r, want))
		equal, e := q.Equal(&xs[i], &r)
		serviceMust(t, e)
		if !equal {
			t.Fatal("source ENSURE coefficient", i)
		}
		serviceMust(t, q.Del(&xs[i]))
		serviceMust(t, q.Del(&r))
	}
	serviceMust(t, s.Close())
}

func TestLinearRationalSourcePivotAndSingularPublication(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	row0 := []util.Rational{util.NewRational(), util.NewRational()}
	row1 := []util.Rational{util.NewRational(), util.NewRational()}
	serviceMust(t, q.SetInt64(&row0[1], 1))
	serviceMust(t, q.SetInt64(&row1[0], 1))
	b0, b1 := util.NewRational(), util.NewRational()
	serviceMust(t, q.SetInt64(&b0, 3))
	serviceMust(t, q.SetInt64(&b1, 7))
	serviceMust(t, s.Add(0, row0, &b0))
	serviceMust(t, s.Add(1, row1, &b1))
	xs := []util.Rational{util.NewRational(), util.NewRational()}
	ok, e := s.Solve(xs)
	serviceMust(t, e)
	if !ok {
		t.Fatal("pivot not found")
	}
	// Actual source swaps A rows only. Do not repair b or replace the algorithm.
	for i, want := range []*util.Rational{&b0, &b1} {
		eq, e := q.Equal(&xs[i], want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("source row-only pivot", i)
		}
	}
	serviceMust(t, s.Reset())
	ok, e = s.Solve(xs)
	serviceMust(t, e)
	if ok {
		t.Fatal("zero matrix singular")
	}
	for i, want := range []*util.Rational{&b0, &b1} {
		eq, e := q.Equal(&xs[i], want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("singular result publication", i)
		}
	}
	for i := range xs {
		serviceMust(t, q.Del(&xs[i]))
		serviceMust(t, q.Del(&row0[i]))
		serviceMust(t, q.Del(&row1[i]))
	}
	serviceMust(t, q.Del(&b0))
	serviceMust(t, q.Del(&b1))
	serviceMust(t, s.Flush())
	serviceMust(t, s.Resize(0))
	ok, e = s.Solve(nil)
	serviceMust(t, e)
	if !ok {
		t.Fatal("empty system")
	}
	serviceMust(t, s.Close())
}

func TestLinearRationalSourceAddReachedInput(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	b := util.NewRational()
	serviceMust(t, q.SetInt64(&b, 9))
	if e = s.Add(0, nil, &b); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached input guard", e)
	}
	// RHS publication precedes reading the borrowed coefficient array.
	actual, e := s.RightHandSide(0)
	serviceMust(t, e)
	eq, e := q.Equal(&actual, &b)
	serviceMust(t, e)
	if !eq {
		t.Fatal("source RHS prefix")
	}
	serviceMust(t, q.Del(&b))
	serviceMust(t, s.Close())
}

func TestLinearModularWholeSourceSolve(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := util.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	serviceMust(t, m.SetZPUint64(97))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	serviceMust(t, s.Resize(3))
	rows := [3][3]int64{{2, 1, -1}, {-3, -1, 2}, {-2, 1, 2}}
	rhs := [3]int64{8, -11, -3}
	for i := range rows {
		as := [3]util.Int{}
		b := util.Int{}
		for j := range as {
			serviceMust(t, m.SetInt64(&as[j], rows[i][j]))
		}
		serviceMust(t, m.SetInt64(&b, rhs[i]))
		serviceMust(t, s.Add(uint32(i), as[:], &b))
		for j := range as {
			serviceMust(t, m.Del(&as[j]))
		}
		serviceMust(t, m.Del(&b))
	}
	xs := [3]util.Int{}
	ok, e := s.Solve(xs[:])
	serviceMust(t, e)
	if !ok {
		t.Fatal("modular solve")
	}
	for i, want := range [3]int64{2, 3, -1} {
		a := util.Int{}
		serviceMust(t, m.SetInt64(&a, want))
		eq, e := m.Equal(&xs[i], &a)
		serviceMust(t, e)
		if !eq {
			t.Fatal("modular original fixture", i)
		}
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&xs[i]))
	}
	serviceMust(t, s.Reset())
	ok, e = s.Solve(xs[:])
	serviceMust(t, e)
	if ok {
		t.Fatal("modular singular")
	}
	serviceMust(t, s.Close())
	serviceMust(t, m.Close())
}

func TestLinearNativeOuterRowReachedIndexGuard(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	row, e := s.row(1)
	want := arena.ErrBounds
	if polynomialSourceDebug {
		want = ErrPolynomialAssertion
	}
	if !errors.Is(e, want) || row != nil {
		t.Fatal("reached matrix row guard", row, e)
	}
	serviceMust(t, s.Close())
}

// A fresh failed RHS Copy leaves a source NULL pointer. Literal flush must
// continue through rows and members; owning cleanup remains a separate action.
func TestLinearRationalFreshNullRHSFlush(t *testing.T) {
	q, budget := rationalPublicationDirectOwner(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	defer func() {
		budget.FailAfter(-1)
		if e := s.Close(); e != nil {
			serviceMust(t, s.Close())
		}
	}()
	serviceMust(t, s.Resize(1))
	rhs := util.NewRational()
	serviceMust(t, q.SetString(&rhs, "79228162514264337593543950337"))
	defer func() { budget.FailAfter(-1); serviceMust(t, q.Del(&rhs)) }()
	baseline := q.Z.Memory.Stats().LiveBlocks
	budget.FailAfter(0)
	e = s.Add(0, []util.Rational{util.NewRational()}, &rhs)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual RHS copy refusal", e)
	}
	actual, e := s.RightHandSide(0)
	serviceMust(t, e)
	if (*actual.BorrowNumerator()).Cell != q.Z.FailedNullRef || (*actual.BorrowDenominator()) != util.NewInt(1) {
		t.Fatal("default RHS source prefix", actual)
	}
	linearDefaultPhysicalCensus(t, q, s, baseline)
	budget.FailAfter(-1)
	serviceMust(t, s.SourceFinalize())
	if !s.sourceFinalized || s.sourceCause != nil || s.n != 0 {
		t.Fatal("defined NULL source flush continuation", s.sourceCause, s.n)
	}
	linearDefaultPhysicalCensus(t, q, s, baseline)
}
