package polynomial

// Source: algebraic_numbers.cpp:2578-2598; accepted display and lexical vars.

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicIsolationSignsTraceRetainsActualRootDisplay(t *testing.T) {
	for _, marker := range []string{":2578 ", ":2598 "} {
		t.Run(marker, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
			z := c.config.Rationals.Integers()
			n := unumeral.NewInt(0)
			serviceMust(t, z.SetString(&n, "-18446744073709551619"))
			x2, e := pm.core.monomials.MkMonomial([]MonomialPower{{Var: 2, Degree: 2}})
			serviceMust(t, e)
			unit, e := pm.core.monomials.Unit()
			serviceMust(t, e)
			p, e := pm.core.MkTerms([]unumeral.Int{unumeral.NewInt(1), n}, []Monomial{x2, unit})
			serviceMust(t, e)
			p = polynomialCaller(t, pm.core, p)
			serviceMust(t, z.Del(&n))
			roots, e := c.NewVector()
			serviceMust(t, e)
			values := &algebraicIsolationAssignments{}
			signs := []int{91}
			writer := &algebraicBinaryDisplayStopWriter{core: c, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("isolate_roots_bug")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.IsolateMultivariateRootsWithSigns(pm, p, values, roots, &signs)
			var stop error
			if tags.ShouldTrace("isolate_roots_bug") {
				if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || !c.rootDisplay.active {
					t.Fatal("actual signs display guard", body)
				}
				stop = c.rootDisplay.sourceStop
				if !c.rootTracePending() || !c.isolationSigns.active || !c.isolationSigns.borrowed || pm.algebraicIsolationBorrowers != 1 || c.isolationSigns.sourceStop != stop {
					t.Error("stopped trace lost actual child or older signs owner")
				}
				if len(signs) != 1 || signs[0] != 91 {
					t.Error("display stop appended signs")
				}
				if c.isolationSigns.traceVariablesConstructed != (marker == ":2578 ") {
					t.Error("trace variable lifetime crossed source boundary")
				}
				if marker == ":2578 " && len(c.isolationSigns.traceVariables.values) != 1 {
					t.Error("older trace variable vector destroyed before display")
				}
				if c.isolationSigns.constructed != (marker == ":2598 ") {
					t.Error("sample declaration moved across diagnostic")
				}
				z.Store(writer.ref, writer.original)
			} else if body != nil || writer.ref != 0 || len(signs) != 4 {
				t.Fatal("erased diagnostic changed signs", body, signs)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootDisplay.active || c.rootTracePending() || c.isolationSigns.active || c.isolationSigns.traceVariablesConstructed || pm.algebraicIsolationBorrowers != 0 {
				t.Fatal("owning signs cleanup retained actual child")
			}
			if stop != nil && c.rootDisplay.sourceStop != stop {
				t.Fatal("owning cleanup changed display source history")
			}
			serviceMust(t, roots.Close())
		})
	}
}

func TestAlgebraicRootDisplayTraceBasicCoefficientsSourceCleanup(t *testing.T) {
	for _, tc := range []struct{ input, polynomial string }{
		{"18446744073709551619/7", "7 # - 18446744073709551619"},
		{"-18446744073709551619/7", "7 # + 18446744073709551619"},
		{"7/18446744073709551619", "18446744073709551619 # - 7"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetString(&q, tc.input))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			var output strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("isolate_roots")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			serviceMust(t, c.traceMultivariateIsolationLinearRoot(&a))
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if c.rootTracePending() || c.rootDisplay.active || c.rootDisplay.coefficients.slot != 0 || c.rootDisplay.negativeOwned {
				t.Fatal("basic trace did not finish actual duplicate coefficients")
			}
			if tags.ShouldTrace("isolate_roots") {
				want := ":2398 ---------\nafter trivial solving p has only one root:\n(" + tc.polynomial + ", 1)\n"
				if !strings.Contains(output.String(), want) {
					t.Fatal("literal basic root trace", output.String(), want)
				}
			} else if output.Len() != 0 {
				t.Fatal("erased basic trace emitted output")
			}
			cmp, e := c.CompareRational(&a, &q)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("trace changed caller's basic rational", cmp)
			}
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		})
	}
}

func TestAlgebraicIsolationSignsPolynomialDisplayStopsBeforeVariables(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	pm, _, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, nil)
	z := c.config.Rationals.Integers()
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "-18446744073709551619"))
	x, e := pm.core.monomials.MkMonomial([]MonomialPower{{Var: 2, Degree: 2}})
	serviceMust(t, e)
	unit, e := pm.core.monomials.Unit()
	serviceMust(t, e)
	p, e := pm.core.MkTerms([]unumeral.Int{unumeral.NewInt(1), n}, []Monomial{x, unit})
	serviceMust(t, e)
	p = polynomialCaller(t, pm.core, p)
	serviceMust(t, z.Del(&n))
	roots, e := c.NewVector()
	serviceMust(t, e)
	signs := []int{91}
	writer := &algebraicRootTraceNumeralStop{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("isolate_roots_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.IsolateMultivariateRootsWithSigns(pm, p, &algebraicIsolationAssignments{}, roots, &signs)
	var frame *polynomialDisplayFrame
	var stop error
	if tags.ShouldTrace("isolate_roots_bug") {
		if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || !c.rootTracePending() {
			t.Fatal("actual input display guard", body)
		}
		frame = c.rootTrace.poly.frame
		stop = frame.sourceCause
		if c.isolationSigns.traceVariablesConstructed || c.isolationSigns.constructed || !c.isolationSigns.active || c.isolationSigns.sourceStop != stop || pm.algebraicIsolationBorrowers != 1 {
			t.Error("input display crossed variable/sample declarations or lost signs owner")
		}
		if c.rootDisplay.active || len(signs) != 1 || signs[0] != 91 {
			t.Error("input display stop reached root display or sign publication")
		}
		z.Store(writer.ref, writer.original)
	} else if body != nil || writer.ref != 0 || len(signs) != 4 {
		t.Fatal("erased input trace changed signs", body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() || c.isolationSigns.active || c.isolationSigns.traceVariablesConstructed || pm.algebraicIsolationBorrowers != 0 || frame != nil && (frame.active || frame.sourceCause != stop) {
		t.Fatal("input display cleanup lost actual source history")
	}
	serviceMust(t, roots.Close())
}
