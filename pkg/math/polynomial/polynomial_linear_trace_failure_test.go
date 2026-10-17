package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"testing"
)

type linearTraceFailWriter struct{ calls int }

func (w *linearTraceFailWriter) Write([]byte) (int, error) { w.calls++; return 0, io.ErrClosedPipe }

func TestLinearRationalTraceFailbitStillReachesGuardBeforeSolveMutation(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()); serviceMust(t, q.BindTrace(nil)) }()
	serviceMust(t, s.Resize(2))
	row := []unumeral.Rational{unumeral.NewRational(), unumeral.NewRational()}
	b := unumeral.NewRational()
	serviceMust(t, q.SetInt32(&row[0], 2))
	serviceMust(t, q.SetInt32(&row[1], 4))
	serviceMust(t, q.SetInt32(&b, 6))
	serviceMust(t, s.Add(0, row, &b))
	out := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("linear_eq_solver")
	if !tags.ShouldTrace("linear_eq_solver") {
		t.Skip("source TRACE erased")
	}
	serviceMust(t, q.BindTrace(tags))
	// Deliberately guard the actual later RHS extent. This is malformed source
	// storage, not an allocation refusal; restore it only for fixture destruction.
	saved := s.b
	s.b = nil
	defer func() { s.b = saved }()
	var xs [2]unumeral.Rational
	_, e = s.Solve(xs[:])
	guard := error(arena.ErrBounds)
	if polynomialSourceDebug {
		guard = ErrPolynomialAssertion
	}
	if !errors.Is(e, guard) || out.calls != 1 {
		t.Fatal("failbit suppressed later source operand", e, out.calls)
	}
	value, e := q.String(&s.A[0][1])
	serviceMust(t, e)
	if value != "4" {
		t.Fatal("numeric normalization ran after trace computation guard", value)
	}
}

func TestLinearModularTraceFailbitStillReachesGuardBeforeSolveMutation(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := unumeral.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	defer func() { serviceMust(t, m.BindTrace(nil)); serviceMust(t, m.Close()) }()
	serviceMust(t, m.SetZPUint64(11))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(2))
	row := []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(4)}
	b := unumeral.NewInt(6)
	serviceMust(t, s.Add(0, row, &b))
	out := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("linear_eq_solver")
	if !tags.ShouldTrace("linear_eq_solver") {
		t.Skip("source TRACE erased")
	}
	serviceMust(t, m.BindTrace(tags))
	saved := s.b
	s.b = nil
	defer func() { s.b = saved }()
	var xs [2]unumeral.Int
	_, e = s.Solve(xs[:])
	guard := error(arena.ErrBounds)
	if polynomialSourceDebug {
		guard = ErrPolynomialAssertion
	}
	if !errors.Is(e, guard) || out.calls != 1 || s.A[0][1].Val != 4 {
		t.Fatal("failbit/computation order", e, out.calls, s.A[0][1])
	}
}

func TestLinearTraceStreamFailureDoesNotCancelValidSolve(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()); serviceMust(t, q.BindTrace(nil)) }()
	serviceMust(t, s.Resize(1))
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetInt32(&a, 2))
	serviceMust(t, q.SetInt32(&b, 6))
	serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
	out := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(out)
	tags.Enable("linear_eq_solver")
	serviceMust(t, q.BindTrace(tags))
	result := []unumeral.Rational{unumeral.NewRational()}
	ok, e := s.Solve(result)
	serviceMust(t, e)
	text, e := q.String(&result[0])
	serviceMust(t, e)
	serviceMust(t, q.Del(&result[0]))
	if !ok || text != "3" {
		t.Fatal("trace stream failure changed solution", ok, text)
	}
	want := 0
	if tags.ShouldTrace("linear_eq_solver") {
		want = 1
	}
	if out.calls != want {
		t.Fatal("failed stream was retried", out.calls, want)
	}
}
