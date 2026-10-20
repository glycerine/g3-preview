package polynomial

import (
	"bytes"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"
)

func TestPolynomialUnivariateFactorTracePublicDispatchAndStreamRefusal(t *testing.T) {
	for _, degree := range []uint32{2} {
		for _, refuse := range []bool{false, true} {
			m := polynomialPSCFullFixture(t)
			p := polynomialQueryTerms(t, m.core, []int32{-1, 1}, [][]MonomialPower{{{0, degree}}, {}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var out bytes.Buffer
			writer := &polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			if refuse {
				tags = diagnostic.NewTraceTagsWithWriter(writer)
			}
			defer func() { tags.Finalize(); serviceMust(t, m.CleanupGCD()); serviceMust(t, f.CloseChecked()) }()
			tags.Enable("factor")
			m.core.monomials.config.Trace = tags
			serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
			polynomialFactorAssertReconstruction(t, m, p, f, 2)
			if tags.ShouldTrace("factor") {
				line, label := ":6689 ", "factor square free univariate:\n"
				if refuse {
					if writer.calls == 0 {
						t.Fatal("refusing writer was not called")
					}
				} else if !strings.Contains(out.String(), line) || !strings.Contains(out.String(), label) {
					t.Fatalf("missing live leaf record: %q", out.String())
				}
			} else if out.Len() != 0 || writer.calls != 0 {
				t.Fatal("disabled TRACE emitted output")
			}
			tags.Finalize()
			serviceMust(t, m.CleanupGCD())
			serviceMust(t, f.CloseChecked())
		}
	}
}
