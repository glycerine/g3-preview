// Source: src/math/polynomial/polynomial.cpp:3134-3182.
package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSparseExecutingTraceRefusesOwnerMutation(t *testing.T) {
	for _, operation := range []string{"reset", "cleanup", "source", "close", "add", "interpolate"} {
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
				}
				if !errors.Is(cause, ErrPolynomialPending) {
					t.Errorf("executing sparse %s: %v", operation, cause)
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
			if s.running {
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
