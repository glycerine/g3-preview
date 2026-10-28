package polynomial

// Source: src/math/polynomial/polynomial.cpp:3134-3182.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSparseExecutingTraceRefusesOwnerMutation(t *testing.T) {
	for _, operation := range []string{"reset", "cleanup", "source", "close", "add", "interpolate", "skeletonSource", "skeletonClose", "managerSource", "managerClose", "coreSource", "coreClose"} {
		t.Run(operation, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			serviceMust(t, m.modular.SetZPUint64(97))
			p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(m, p, 0)
			serviceMust(t, e)
			s, e := NewPolynomialSparseInterpolator(full, sk)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CloseChecked()); serviceMust(t, sk.CloseChecked()) })
			samples := []Polynomial{p}
			for _, x := range []int32{0, 1, 2} {
				in := unumeral.NewInt(x)
				q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
				samples = append(samples, q)
				ok, e := s.Add(&in, q)
				serviceMust(t, e)
				if !ok {
					t.Fatal("sample rejected", x)
				}
			}
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			w := &linearOperationProbeWriter{probe: func() {
				inputs, outputs := append([]unumeral.Int(nil), s.inputs...), append([]unumeral.Int(nil), s.outputs...)
				before, header := m.integer.Budget().Used(), s.header
				var cause error
				switch operation {
				case "reset":
					cause = s.Reset()
				case "cleanup":
					cause = s.CleanupInterpolation()
				case "source":
					cause = s.SourceFinalize()
				case "close":
					cause = s.CloseChecked()
				case "add":
					_, cause = s.Add(nil, Polynomial{})
				case "interpolate":
					_, cause = s.Interpolate(nil)
				case "skeletonSource":
					cause = sk.SourceFinalize()
				case "skeletonClose":
					cause = sk.CloseChecked()
				case "managerSource":
					cause = full.SourceFinalize()
				case "managerClose":
					cause = full.CloseChecked()
				case "coreSource":
					cause = m.SourceFinalizePrefix()
				case "coreClose":
					cause = m.CloseChecked()
				}
				if !errors.Is(cause, ErrPolynomialPending) {
					t.Errorf("executing sparse %s: %v", operation, cause)
				}
				if sk.closed || sk.closing || sk.sourceStarted || full.closing || m.closing || m.sourceStage != 0 || full.cleanupStage != 0 {
					t.Error("callback advanced borrowed owner destruction")
				}
				if !reflect.DeepEqual(inputs, s.inputs) || !reflect.DeepEqual(outputs, s.outputs) || s.header != header || s.sourceStarted || s.closed || s.closing || !s.interpolation.active || before != m.integer.Budget().Used() {
					t.Error("callback changed executing sparse owner")
				}
			}}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("sparse_interpolator")
			m.monomials.config.Trace = tags
			enabled := tags.ShouldTrace("sparse_interpolator")
			ok, e := s.Interpolate(out)
			tags.Disable("sparse_interpolator")
			if e != nil || !ok {
				t.Errorf("interpolation did not resume: %v %v", ok, e)
				return
			}
			if s.running || s.previous != nil || m.sparseCaller != nil {
				t.Fatal("returned interpolation retained execution guard")
			}
			if w.called != enabled {
				t.Fatal("source trace callback coverage mismatch", w.called, enabled)
			}
			same, e := m.Equal(out.root.value, p)
			serviceMust(t, e)
			if !same {
				t.Fatal("interpolated coefficients changed")
			}
			serviceMust(t, s.Reset())
			polynomialSparsePhysical(t, s, out, samples)
		})
	}
}

type sparseBorrowProbeWriter struct{ probe func() }

func (w *sparseBorrowProbeWriter) Write(p []byte) (int, error) { w.probe(); return len(p), nil }

func TestPolynomialSparseNestedBorrowRestoresCaller(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	var solvers [2]*PolynomialSparseInterpolator
	var skeletons [2]*PolynomialSkeleton
	var outputs [2]*PolynomialRef
	samples := []Polynomial{p}
	for i := range solvers {
		sk, e := NewPolynomialSkeleton(m, p, 0)
		serviceMust(t, e)
		skeletons[i] = sk
		s, e := NewPolynomialSparseInterpolator(full, sk)
		serviceMust(t, e)
		solvers[i] = s
		out, e := NewPolynomialRef(m)
		serviceMust(t, e)
		outputs[i] = out
		t.Cleanup(func() {
			serviceMust(t, out.Close())
			serviceMust(t, s.CloseChecked())
			serviceMust(t, sk.CloseChecked())
		})
		for _, x := range []int32{0, 1, 2} {
			in := unumeral.NewInt(x)
			q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
			samples = append(samples, q)
			ok, e := s.Add(&in, q)
			serviceMust(t, e)
			if !ok {
				t.Fatal("sample rejected")
			}
		}
	}
	unrelated, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, unrelated.CloseChecked()) })
	outerReached, innerReached := false, false
	writer := &sparseBorrowProbeWriter{probe: func() {
		if !outerReached {
			outerReached = true
			serviceMust(t, unrelated.CloseChecked())
			if m.sparseCaller != solvers[0] {
				t.Error("outer caller not bound")
			}
			ok, e := solvers[1].Interpolate(outputs[1])
			serviceMust(t, e)
			if !ok {
				t.Fatal("nested interpolation failed")
			}
			if m.sparseCaller != solvers[0] || solvers[1].running || solvers[1].previous != nil {
				t.Error("nested return lost outer caller")
			}
		} else if solvers[1].running && !innerReached {
			innerReached = true
			if m.sparseCaller != solvers[1] || solvers[1].previous != solvers[0] {
				t.Error("nested caller chain missing")
			}
			for _, sk := range skeletons {
				if e := sk.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
					t.Error("nested caller lost exact skeleton borrow", e)
				}
			}
			if e := m.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
				t.Error("nested caller lost core borrow", e)
			}
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("sparse_interpolator")
	m.monomials.config.Trace = tags
	enabled := tags.ShouldTrace("sparse_interpolator")
	ok, e := solvers[0].Interpolate(outputs[0])
	tags.Disable("sparse_interpolator")
	serviceMust(t, e)
	if !ok || outerReached != enabled || innerReached != enabled || m.sparseCaller != nil || m.sparseExecuting(nil) {
		t.Fatal("nested execution did not retire actual callers")
	}
	for i, s := range solvers {
		if s.running || s.previous != nil {
			t.Fatal("retained caller link")
		}
		if i == 0 || enabled {
			same, e := m.Equal(outputs[i].root.value, p)
			serviceMust(t, e)
			if !same {
				t.Fatal("nested result changed", i)
			}
		}
	}
	serviceMust(t, outputs[1].Close())
	serviceMust(t, solvers[1].CloseChecked())
	serviceMust(t, skeletons[1].CloseChecked())
	serviceMust(t, unrelated.CloseChecked())
	polynomialSparsePhysical(t, solvers[0], outputs[0], samples)
}
