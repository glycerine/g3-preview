package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strconv"
	"strings"
	"testing"
)

type polynomialUnivariateBugWriter struct {
	polynomialQuadraticTraceWriter
}

func (w *polynomialUnivariateBugWriter) Write(p []byte) (int, error) {
	if w.m.factorUnivariate.executing {
		if e := w.m.upm.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			w.t.Fatalf("univariate diagnostic callback closed backend: %v", e)
		}
	}
	return w.polynomialQuadraticTraceWriter.Write(p)
}
func TestPolynomialUnivariateFactorBugTraceConvertedRecordsAndRefusal(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		p := polynomialQueryTerms(t, m.core, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, {}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		var out bytes.Buffer
		writer := &polynomialUnivariateBugWriter{polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}}
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		if refuse {
			tags = diagnostic.NewTraceTagsWithWriter(writer)
		}
		defer func() { tags.Finalize(); serviceMust(t, m.CleanupGCD()); serviceMust(t, f.CloseChecked()) }()
		tags.Enable("factor_bug")
		m.core.monomials.config.Trace = tags
		serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
		polynomialFactorAssertReconstruction(t, m, p, f, 2)
		if tags.ShouldTrace("factor_bug") {
			if refuse {
				if writer.calls == 0 {
					t.Fatal("writer not reached")
				}
			} else {
				text := out.String()
				for _, want := range []string{":6707 ", "factoring fs constant: 1\np:\n", ":6715 ", "uni-factor:\n", "\nfactor:\n", ":6720 ", "end-factors...\n"} {
					if !strings.Contains(text, want) {
						t.Fatalf("missing %q in %q", want, text)
					}
				}
				if strings.Count(text, "uni-factor:\n") != 2 {
					t.Fatal("converted factor record count", text)
				}
			}
			if !m.core.matchesDisplayReceipt(&m.factorUnivariate.tracePolynomial) || !m.upm.matchesDisplayReceipt(&m.factorUnivariate.traceUnivariate) {
				t.Fatal("completed diagnostics lost actual formatter receipts")
			}
		} else if out.Len() != 0 || writer.calls != 0 {
			t.Fatal("disabled factor_bug emitted")
		}
		tags.Finalize()
		serviceMust(t, f.CloseChecked())
	}
}

func TestPolynomialUnivariateFactorBugEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 512 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixtureWithLimit(t, 2<<20)
			p := polynomialQueryTerms(t, m.core, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, {}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("factor_bug")
			m.core.monomials.config.Trace = tags
			var receipt polynomialFactorUnivariateInvocationReceipt
			defer func() {
				tags.Finalize()
				m.core.integer.Budget().FailAfter(-1)
				serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, false))
				serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, true))
				serviceMust(t, f.CloseChecked())
			}()
			m.core.integer.Budget().FailAfter(ordinal)
			cause := m.factorSquareFreePrimitiveUnivariateWithReceipt(p, f, 1, DefaultUPolynomialFactorParams(), &receipt)
			m.core.integer.Budget().FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual factor_bug reservation", cause)
			}
			if cause == nil {
				polynomialFactorAssertReconstruction(t, m, p, f, 2)
				if tags.ShouldTrace("factor_bug") && (!m.core.matchesDisplayReceipt(&receipt.frame.tracePolynomial) || !m.upm.matchesDisplayReceipt(&receipt.frame.traceUnivariate)) {
					t.Fatal("completed diagnostics lost actual formatter receipts")
				}
				success = true
			}
		})
	}
	if !success {
		t.Fatal("no successful actual factor_bug reservation prefix")
	}
}

func TestPolynomialUnivariateFactorBugNegativeBackendConstant(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{-1, 1}, [][]MonomialPower{{{0, 2}}, {}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("factor_bug")
	m.core.monomials.config.Trace = tags
	var receipt polynomialFactorUnivariateInvocationReceipt
	defer func() {
		tags.Finalize()
		serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, false))
		serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, true))
		serviceMust(t, f.CloseChecked())
	}()
	serviceMust(t, m.factorSquareFreePrimitiveUnivariateWithReceipt(p, f, 3, DefaultUPolynomialFactorParams(), &receipt))
	if f.constant.Val != -1 {
		t.Fatal("odd multiplicity lost negative backend constant", f.constant)
	}
	if tags.ShouldTrace("factor_bug") && !strings.Contains(out.String(), "factoring fs constant: -1\np:\n") {
		t.Fatal("raw integer String lost negative backend constant", out.String())
	}
}
