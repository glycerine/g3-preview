// Source: src/math/polynomial/polynomial.cpp:3157-3171; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type polynomialSparseSolverRefusingWriter struct {
	bytes.Buffer
	armed, refused bool
	cause          error
}

func (w *polynomialSparseSolverRefusingWriter) Write(p []byte) (int, error) {
	if w.armed && !w.refused {
		w.refused = true
		return 0, w.cause
	}
	n, err := w.Buffer.Write(p)
	if string(p) == "system of equations:\n" {
		w.armed = true
	}
	return n, err
}
func (w *polynomialSparseSolverRefusingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestPolynomialSparseInterpolationSourceTraceSequence(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	sk, err := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, err)
	s, err := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()); serviceMust(t, sk.CloseChecked()) })
	for _, x := range []int32{0, 1, 2} {
		input := unumeral.NewInt(x)
		output := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
		ok, e := s.Add(&input, output)
		serviceMust(t, e)
		if !ok {
			t.Fatal("sample rejected", x)
		}
	}
	r, err := NewPolynomialRef(m)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sparse_interpolator")
	m.monomials.config.Trace = tags
	ok, err := s.Interpolate(r)
	serviceMust(t, err)
	if !ok {
		t.Fatal("Vandermonde solve failed")
	}
	if tags.ShouldTrace("sparse_interpolator") {
		header := func(line int) string {
			if line == 3157 {
				return "-------- [sparse_interpolator] mk src/math/polynomial/polynomial.cpp:3157 ---------\n"
			}
			return "-------- [sparse_interpolator] mk src/math/polynomial/polynomial.cpp:3164 ---------\n"
		}
		footer := "------------------------------------------------\n"
		want := header(3157) + "adding new equation:\n1 0 0 \n" + footer +
			header(3157) + "adding new equation:\n1 1 1 \n" + footer +
			header(3157) + "adding new equation:\n1 2 4 \n" + footer +
			header(3164) + "find coefficients of:\n1\nx0\nx0^2\nsystem of equations:\n" +
			"1 0 0 2\n1 1 1 6\n1 2 4 12\n" + footer
		if output.String() != want {
			t.Fatalf("sparse interpolation source trace: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("ordinary build emitted sparse trace: %q", output.String())
	}
	tags.Disable("sparse_interpolator")
	serviceMust(t, s.Reset())

	refusal := errors.New("sparse trace refused")
	refused := &polynomialSparseSolverRefusingWriter{cause: refusal}
	tags = diagnostic.NewTraceTagsWithWriter(refused)
	tags.Enable("sparse_interpolator")
	m.monomials.config.Trace = tags
	for _, x := range []int32{0, 1, 2} {
		input := unumeral.NewInt(x)
		q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
		ok, e := s.Add(&input, q)
		serviceMust(t, e)
		if !ok {
			t.Fatal("sample rejected after trace refusal", x)
		}
	}
	ok, err = s.Interpolate(r)
	serviceMust(t, err)
	if !ok {
		t.Fatal("refused trace changed interpolation result")
	}
	if tags.ShouldTrace("sparse_interpolator") {
		if !refused.refused || !strings.HasSuffix(refused.String(), "system of equations:\n") {
			t.Fatalf("refused sparse matrix display prefix: refused=%v output=%q", refused.refused, refused.String())
		}
	} else if refused.Len() != 0 {
		t.Fatalf("ordinary build emitted sparse trace: %q", refused.String())
	}
	tags.Disable("sparse_interpolator")
}
