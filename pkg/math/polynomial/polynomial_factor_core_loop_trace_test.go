package polynomial

import (
	"bytes"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"strings"
	"testing"
)

func TestPolynomialFactorCoreLoopTraceMultiplicityAndRefusingWriter(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		p := polynomialQueryTerms(t, m.core, []int32{1, 3, 3, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, {}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		var out bytes.Buffer
		writer := &polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		if refuse {
			tags = diagnostic.NewTraceTagsWithWriter(writer)
		}
		defer func() { tags.Finalize(); serviceMust(t, m.CleanupFactor()); serviceMust(t, f.CloseChecked()) }()
		tags.Enable("factor")
		m.core.monomials.config.Trace = tags
		serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
		polynomialFactorAssertReconstruction(t, m, p, f, 1)
		if len(f.degrees) != 1 || f.degrees[0] != 3 {
			t.Fatal("lost multiplicity", f.degrees)
		}
		if tags.ShouldTrace("factor") {
			if refuse {
				if writer.calls == 0 {
					t.Fatal("writer not reached")
				}
			} else {
				text := out.String()
				last := -1
				for _, want := range []string{"factor_core main loop j: 1\nA: ", "factor_core main loop j: 2\nA: ", "factor_core main loop j: 3\nA: "} {
					at := strings.Index(text, want)
					if at <= last {
						t.Fatalf("missing/out-of-order %q in %q", want, text)
					}
					last = at
				}
				if strings.Count(text, ":6795 ") != 3 || strings.Count(text, ":6807 ") != 2 || strings.Count(text, "const C: 1\n") != 2 {
					t.Fatal("source loop/constant records", text)
				}
			}
			if !m.matchesFactorCore(&m.factorRoot) || m.factorRoot.frame.traceDisplay.frame == nil || m.factorRoot.frame.traceDisplay.generation == 0 {
				t.Fatal("loop diagnostics lost actual formatter receipt")
			}
		} else if out.Len() != 0 || writer.calls != 0 {
			t.Fatal("erased TRACE emitted output")
		}
		tags.Finalize()
		serviceMust(t, f.CloseChecked())
	}
}

func TestPolynomialFactorCoreLoopTraceEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			p := polynomialQueryTerms(t, m.core, []int32{1, 3, 3, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, {}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("factor")
			m.core.monomials.config.Trace = tags
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, m.CleanupFactor())
				serviceMust(t, f.CloseChecked())
			}()
			m.originalBudget.FailAfter(ordinal)
			cause := m.Factor(p, f, DefaultUPolynomialFactorParams())
			m.originalBudget.FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual loop diagnostics reservation", cause)
			}
			if cause == nil {
				polynomialFactorAssertReconstruction(t, m, p, f, 1)
				if tags.ShouldTrace("factor") && !strings.Contains(out.String(), "factor_core main loop j: 3\nA: ") {
					t.Fatal("success omitted final actual loop trace")
				}
				success = true
			}
		})
	}
	if !success {
		t.Fatal("actual loop diagnostics never reached success")
	}
}

func TestPolynomialFactorCoreTraceBIGIntegerEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixtureWithLimit(t, 4<<20)
			z := m.core.integer
			var base, triple unumeral.Int
			serviceMust(t, z.SetString(&base, "18446744073709551616"))
			three := unumeral.NewInt(3)
			serviceMust(t, z.Mul(&base, &three, &triple))
			var monos []Monomial
			for _, powers := range [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, {}} {
				mo, e := m.core.monomials.MkMonomial(powers)
				serviceMust(t, e)
				monos = append(monos, mo)
			}
			p, e := m.core.MkTerms([]unumeral.Int{base, triple, triple, base}, monos)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(p))
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("factor")
			m.core.monomials.config.Trace = tags
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, m.CleanupFactor())
				serviceMust(t, f.CloseChecked())
				serviceMust(t, m.core.DecRef(p))
				serviceMust(t, z.Del(&triple))
				serviceMust(t, z.Del(&base))
			}()
			m.originalBudget.FailAfter(ordinal)
			cause := m.Factor(p, f, DefaultUPolynomialFactorParams())
			m.originalBudget.FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual BIG integer trace refusal", cause)
			}
			if cause == nil {
				polynomialFactorAssertReconstruction(t, m, p, f, 1)
				if tags.ShouldTrace("factor") && (!strings.Contains(out.String(), "i: 18446744073709551616\n") || m.factorRoot.frame.traceNumeral.Character == nil) {
					t.Fatal("BIG integer record omitted actual character invocation", out.String())
				}
				serviceMust(t, f.Reset())
				writer := &polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: true}
				refused := diagnostic.NewTraceTagsWithWriter(writer)
				refused.Enable("factor")
				defer refused.Finalize()
				m.core.monomials.config.Trace = refused
				serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
				polynomialFactorAssertReconstruction(t, m, p, f, 1)
				if refused.ShouldTrace("factor") && (writer.calls == 0 || m.factorRoot.frame.traceNumeral.Character == nil) {
					t.Fatal("refused stream suppressed actual BIG integer formatting")
				}
				success = true
			}
		})
	}
	if !success {
		t.Fatal("actual BIG integer trace never reached success")
	}
}
