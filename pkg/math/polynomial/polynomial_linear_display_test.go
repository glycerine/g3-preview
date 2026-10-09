// Source: src/math/polynomial/linear_eq_solver.h:140-150.
package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestLinearRationalSourceDisplaySpacingAndOrder(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	row := [2]util.Rational{util.NewRational(), util.NewRational()}
	rhs := util.NewRational()
	serviceMust(t, q.SetFractionSigned(&row[0], -3, 2))
	serviceMust(t, q.SetInt64(&row[1], 5))
	serviceMust(t, q.SetInt64(&rhs, 7))
	serviceMust(t, s.Add(0, row[:], &rhs))
	serviceMust(t, s.Add(1, row[:], &rhs))
	var out strings.Builder
	serviceMust(t, s.Display(&out))
	if out.String() != "-3/2 5 7\n-3/2 5 7\n" {
		t.Fatal("source row/space/newline order", out.String())
	}
	for i := range row {
		serviceMust(t, q.Del(&row[i]))
	}
	serviceMust(t, q.Del(&rhs))
	serviceMust(t, s.Close())
}

func TestLinearModularSourceDisplayUsesUnderlyingInteger(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := util.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	serviceMust(t, m.SetZPUint64(7))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	// display delegates to underlying integer, without the modular String
	// normalization assertion. The actual stored raw value remains unnormalized.
	r, e := s.row(0)
	serviceMust(t, e)
	a := util.Int{}
	serviceMust(t, q.Integers().SetInt64(&a, 9))
	(*r)[0] = a
	s.b[0] = a
	var out strings.Builder
	serviceMust(t, s.Display(&out))
	if out.String() != "9 9\n" {
		t.Fatal("underlying source display", out.String())
	}
	serviceMust(t, s.Close())
	serviceMust(t, m.Close())
}

type linearStoppedWriter struct {
	calls int
	seen  strings.Builder
	cause error
}

func (w *linearStoppedWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 2 {
		return 0, w.cause
	}
	return w.seen.Write(p)
}
func TestLinearSourceDisplayStoppedWriterBeforeRHSRead(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	a := util.NewRational()
	serviceMust(t, q.SetInt64(&a, 5))
	serviceMust(t, s.Add(0, []util.Rational{a}, &a))
	// Make the later RHS read guarded, without changing the reached row.
	saved := s.b
	s.b = nil
	marker := errors.New("stopped host output")
	out := linearStoppedWriter{cause: marker}
	e = s.Display(&out)
	s.b = saved
	if !errors.Is(e, marker) || out.seen.String() != "5" || out.calls != 2 {
		t.Fatal("source space output stops before RHS read", e, out.seen.String(), out.calls)
	}
	serviceMust(t, q.Del(&a))
	serviceMust(t, s.Close())
}

type linearShortWriter struct{}

func (linearShortWriter) Write(p []byte) (int, error) { return 0, nil }
func TestLinearSourceDisplayShortWriterGuard(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	if e = s.Display(linearShortWriter{}); !errors.Is(e, io.ErrShortWrite) {
		t.Fatal("host short write", e)
	}
	serviceMust(t, s.Close())
}
