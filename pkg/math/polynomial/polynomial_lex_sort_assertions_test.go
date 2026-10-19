package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialLexSortSourceAssertionActions(t *testing.T) {
	sites := []struct {
		line       int32
		expression string
	}{
		{1316, "end > start"},
		{1333, "p.size() == end - start"},
		{1352, "d_j <= d"},
		{1356, "j == end || m_ms[j]->degree_of(x) < d"},
		{1453, "lex_compare(m_ms[i], m_ms[i+1]) > 0"},
	}
	for _, site := range sites {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(action+"/"+site.expression, func(t *testing.T) {
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				reads := 0
				e := polynomialLexSortSourceAssert(site.line, site.expression, func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("lex-sort assertion did not throw", e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled lex-sort assertion evaluated predicate", e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release lex-sort assertion behavior", e, reads)
				}
			})
		}
	}
}
