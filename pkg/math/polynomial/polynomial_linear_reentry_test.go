package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"testing"
)

type linearOperationProbeWriter struct {
	probe  func()
	called bool
}

func (w *linearOperationProbeWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		w.probe()
	}
	return len(p), nil
}

func TestLinearRationalExecutingOutputRefusesOwnerMutation(t *testing.T) {
	for _, trace := range []bool{false, true} {
		t.Run(map[bool]string{false: "display", true: "solve-trace"}[trace], func(t *testing.T) {
			q, _ := rationalManager(t, false)
			s, e := NewRationalLinearSolver(q)
			serviceMust(t, e)
			defer func() { serviceMust(t, s.Close()); serviceMust(t, q.BindTrace(nil)) }()
			serviceMust(t, s.Resize(1))
			a, b := unumeral.NewRational(), unumeral.NewRational()
			serviceMust(t, q.SetInt32(&a, 2))
			serviceMust(t, q.SetInt32(&b, 6))
			serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
			var nested [1]unumeral.Rational
			output := &linearOperationProbeWriter{probe: func() {
				before, aa, bb := q.Budget().Used(), s.A[0][0], s.b[0]
				header := s.header
				attempts := []func() error{s.Close, s.SourceFinalize, s.Flush, s.Reset, func() error { return s.Resize(1) }, func() error { return s.Add(0, nil, nil) }, func() error { _, e := s.Solve(nested[:]); return e }, func() error { return s.Display(io.Discard) }}
				for i, attempt := range attempts {
					if e := attempt(); !errors.Is(e, ErrPolynomialPending) {
						t.Fatalf("executing owner action %d: %v", i, e)
					}
				}
				if !s.running || s.closed || s.closing || s.sourceFinalized || s.header != header || s.sourceStage != 0 || s.sourceCause != nil || s.A[0][0] != aa || s.b[0] != bb || q.Budget().Used() != before {
					t.Fatal("callback changed executing owner")
				}
			}}
			if trace {
				tags := diagnostic.NewTraceTagsWithWriter(output)
				tags.Enable("linear_eq_solver")
				if !tags.ShouldTrace("linear_eq_solver") {
					t.Skip("source TRACE erased")
				}
				serviceMust(t, q.BindTrace(tags))
				result := []unumeral.Rational{unumeral.NewRational()}
				ok, e := s.Solve(result)
				serviceMust(t, e)
				text, e := q.String(&result[0])
				serviceMust(t, e)
				serviceMust(t, q.Del(&result[0]))
				if !ok || text != "3" {
					t.Fatal("continued solve changed result", ok, text)
				}
			} else {
				serviceMust(t, s.Display(output))
			}
			if !output.called || s.running {
				t.Fatal("missing callback or retained executing flag")
			}
			serviceMust(t, s.Reset())
		})
	}
}

func TestLinearModularExecutingOutputRefusesOwnerMutation(t *testing.T) {
	for _, trace := range []bool{false, true} {
		t.Run(map[bool]string{false: "display", true: "solve-trace"}[trace], func(t *testing.T) {
			q, _ := rationalManager(t, false)
			m, e := unumeral.NewIntegerModularManager(q.Integers())
			serviceMust(t, e)
			defer func() { serviceMust(t, m.BindTrace(nil)); serviceMust(t, m.Close()) }()
			serviceMust(t, m.SetZPUint64(11))
			s, e := NewModularLinearSolver(m)
			serviceMust(t, e)
			defer func() { serviceMust(t, s.Close()) }()
			serviceMust(t, s.Resize(1))
			a, b := unumeral.NewInt(2), unumeral.NewInt(6)
			serviceMust(t, s.Add(0, []unumeral.Int{a}, &b))
			var nested [1]unumeral.Int
			output := &linearOperationProbeWriter{probe: func() {
				before, aa, bb := q.Budget().Used(), s.A[0][0], s.b[0]
				header := s.header
				attempts := []func() error{s.Close, s.SourceFinalize, s.Flush, s.Reset, func() error { return s.Resize(1) }, func() error { return s.Add(0, nil, nil) }, func() error { _, e := s.Solve(nested[:]); return e }, func() error { return s.Display(io.Discard) }}
				for i, attempt := range attempts {
					if e := attempt(); !errors.Is(e, ErrPolynomialPending) {
						t.Fatalf("executing owner action %d: %v", i, e)
					}
				}
				if !s.running || s.closed || s.closing || s.sourceFinalized || s.header != header || s.sourceStage != 0 || s.sourceCause != nil || s.A[0][0] != aa || s.b[0] != bb || q.Budget().Used() != before {
					t.Fatal("callback changed executing owner")
				}
			}}
			if trace {
				tags := diagnostic.NewTraceTagsWithWriter(output)
				tags.Enable("linear_eq_solver")
				if !tags.ShouldTrace("linear_eq_solver") {
					t.Skip("source TRACE erased")
				}
				serviceMust(t, m.BindTrace(tags))
				var result [1]unumeral.Int
				ok, e := s.Solve(result[:])
				serviceMust(t, e)
				if !ok || result[0].Val != 3 {
					t.Fatal("continued solve changed result", ok, result)
				}
				serviceMust(t, q.Integers().Del(&result[0]))
			} else {
				serviceMust(t, s.Display(output))
			}
			if !output.called || s.running {
				t.Fatal("missing callback or retained executing flag")
			}
			serviceMust(t, s.Reset())
		})
	}
}
