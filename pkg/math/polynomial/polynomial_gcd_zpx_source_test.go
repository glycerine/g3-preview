// Source: src/math/polynomial/polynomial.cpp:4018-4051.
package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialGCDGlexZPXIndependentCoefficientModel(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("mgcd_detail")
	c.monomials.config.Trace = tags
	// Viewed over GF11[x], maximal remaining monomial y^2 has coefficient 3x+4.
	p := polynomialQueryTerms(t, c, []int32{3, 4, 5, 1}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 2}}, {{0, 3}, {1, 1}}, nil})
	a, e := full.lcGlexZPX(p, 0)
	serviceMust(t, e)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, r.root.assign(c, a))
	polynomialQueryExpect(t, c, a, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 4})
	if tags.ShouldTrace("mgcd_detail") {
		if !strings.Contains(trace.String(), "[mgcd_detail] lc_glex_ZpX src/math/polynomial/polynomial.cpp:4042 ---------\nmaximal monomial: x1^2\n") {
			t.Fatalf("missing maximal-monomial source trace: %q", trace.String())
		}
	} else if trace.Len() != 0 {
		t.Fatalf("ordinary build emitted trace: %q", trace.String())
	}
}

func TestPolynomialGCDICCPZPXContentAndPrimitiveIndependentModel(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	var conditionalTrace bytes.Buffer
	conditionalTags := diagnostic.NewTraceTagsWithWriter(&conditionalTrace)
	conditionalTags.Enable("content_bug")
	c.monomials.config.Trace = conditionalTags
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	// (x+1)(y+2) has GF11[x] content x+1 and primitive y+2.
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
	cr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cr.Close()); serviceMust(t, pr.Close()) })
	var ci unumeral.Int
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&ci)) })
	serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
	polynomialQueryExpect(t, c, cr.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	polynomialQueryExpect(t, c, pr.root.value, map[[3]uint32]int64{{0, 1, 0}: 1, {}: 2})
	if conditionalTrace.Len() != 0 {
		t.Fatalf("valid ZpX content emitted content_bug: %q", conditionalTrace.String())
	}
}

func TestPolynomialGCDICCPZPXSourceShortcutTable(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		serviceMust(t, c.modular.SetZPUint64(11))
		var trace bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&trace)
		tags.Enable("mgcd_detail")
		c.monomials.config.Trace = tags
		var p Polynomial
		var cw, pw map[[3]uint32]int64
		ciWant := int64(1)
		switch route {
		case 0:
			p = c.zero
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{}
			ciWant = 0
		case 1:
			p = polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{nil})
			cw = map[[3]uint32]int64{{}: 1}
			pw = cw
			ciWant = 3
		case 2:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1}
		case 3:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 2}}})
			cw = map[[3]uint32]int64{{2, 0, 0}: 1}
			pw = map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1}
		case 4:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{2, 1}}})
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{{1, 1, 0}: 1, {0, 0, 1}: 1}
		}
		cr, e := NewPolynomialRef(c)
		serviceMust(t, e)
		pr, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, cr.Close()); serviceMust(t, pr.Close()) })
		var ci unumeral.Int
		t.Cleanup(func() { serviceMust(t, c.integer.Del(&ci)) })
		serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
		got, e := c.integer.Int64(&ci)
		serviceMust(t, e)
		if got != ciWant {
			t.Fatal("source ci shortcut", route, got, ciWant)
		}
		polynomialQueryExpect(t, c, cr.root.value, cw)
		polynomialQueryExpect(t, c, pr.root.value, pw)
		if tags.ShouldTrace("mgcd_detail") {
			got := trace.String()
			if !strings.Contains(got, "[mgcd_detail] iccp_ZpX src/math/polynomial/polynomial.cpp:3878 ---------\niccp_ZpX, p: ") || !strings.Contains(got, "\nvar x0\n") {
				t.Fatalf("missing source entry trace for route %d: %q", route, got)
			}
			branches := []string{
				":3880 ---------\niccp_ZpX, p is zero\n",
				":3887 ---------\niccp_ZpX, p is constant\n",
				":3895 ---------\niccp_ZpX, degree(p, x) == 0\n",
				":3922 ---------\niccp_ZpX, all monomials contain x0, dividing by x0^2\n",
				":3954 ---------\niccp_ZpX, cheap case... invoking ic\n",
			}
			if !strings.Contains(got, branches[route]) {
				t.Fatalf("missing source shortcut trace for route %d: %q", route, got)
			}
		} else if trace.Len() != 0 {
			t.Fatalf("ordinary build emitted trace for route %d: %q", route, trace.String())
		}
		if route == 4 {
			tags.Disable("mgcd_detail")
			before := trace.String()
			serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
			if trace.String() != before {
				t.Fatalf("disabled tag emitted source trace: %q", trace.String())
			}
			refused := &algebraicIntervalRefusingWriter{stop: 10, err: errors.New("trace refused")}
			tags = diagnostic.NewTraceTagsWithWriter(refused)
			tags.Enable("mgcd_detail")
			c.monomials.config.Trace = tags
			serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
			polynomialQueryExpect(t, c, cr.root.value, cw)
			polynomialQueryExpect(t, c, pr.root.value, pw)
			if tags.ShouldTrace("mgcd_detail") && refused.writes != 10 {
				t.Fatalf("refused source stream writes: %d", refused.writes)
			}
		}
	}
}

func TestPolynomialGCDIntegerContentFieldChangeSourceInputAssertion(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	p := polynomialQueryTerms(t, c, []int32{12, 16}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	serviceMust(t, c.modular.SetZPUint64(11))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	var content unumeral.Int
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&content)) })
	e = c.IntegerContent(p, &content, r)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, []*unumeral.Int{&content}, r.header)
	serviceMust(t, c.CleanupIntegerContent())
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, []*unumeral.Int{&content}, r.header)
	if polynomialSourceDebug {
		if !errors.Is(e, unumeral.ErrModularAssertion) {
			t.Fatalf("source div requires normalized dividend: %v", e)
		}
		return
	}
	serviceMust(t, e)
	got, e := c.integer.Int64(&content)
	serviceMust(t, e)
	if got != 4 {
		t.Fatal("integer content source manager", got)
	}
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{1, 0, 0}: 3, {0, 1, 0}: 4})
}
