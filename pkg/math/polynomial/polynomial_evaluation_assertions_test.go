package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialEvaluationSourceAssertionActions(t *testing.T) {
	sites := []struct {
		line       int32
		expression string
	}{
		{6440, "start < end"},
		{6441, "end <= p->size()"},
		{6442, "is_valid(x)"},
		{6447, "m->degree_of(x) > 0"},
		{6453, "x2v.contains(y)"},
		{6460, "x2v.contains(x)"},
		{6470, "end == i+1"},
		{6483, "d_j <= d"},
		{6489, "j == end || p->m(j)->degree_of(x) < d"},
		{6492, "j == i+1"},
		{6516, "size(p)==1"},
		{6532, "m_x == x"},
		{6536, "is_univariate(p)"},
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
				e := polynomialEvaluationSourceAssert(site.line, site.expression, func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) || reads != 1 {
						t.Fatal("evaluation assertion did not throw", e, reads)
					}
				} else if polynomialSourceDebug && action == "disabled" {
					if e != nil || reads != 0 {
						t.Fatal("disabled evaluation assertion evaluated predicate", e, reads)
					}
				} else if e != nil || (polynomialSourceDebug && reads != 1) || (!polynomialSourceDebug && reads != 0) {
					t.Fatal("Continue/release evaluation assertion behavior", e, reads)
				}
			})
		}
	}
}
