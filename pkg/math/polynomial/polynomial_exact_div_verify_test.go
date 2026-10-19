package polynomial

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialExactDivSourceVerifyActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choice := action
			if action == "disabled" {
				choice = "C"
			}
			file := polynomialVerifyActionsFixture(t, choice)
			if action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := polynomialExactDivSourceVerifyFailure()
			if !polynomialSourceDebug {
				if !errors.Is(e, ErrPolynomialUnreachable) || !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
					t.Fatal("release VERIFY undefined domain", e)
				}
			} else if action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("VERIFY Throw", e)
				}
			} else if !errors.Is(e, ErrPolynomialUnreachable) {
				t.Fatal("continued VERIFY did not retain typed stop", e)
			}
			data, e := os.ReadFile(file.Name())
			serviceMust(t, e)
			if !strings.HasPrefix(string(data), "Failed to verify: div(m_r, m_q, m_r_q)\n") {
				t.Fatal("VERIFY message", string(data))
			}
			if polynomialSourceDebug && !strings.Contains(string(data), "src/math/polynomial/polynomial.cpp\nLine: 5142\nUNREACHABLE CODE WAS REACHED.") {
				t.Fatal("VERIFY source site", string(data))
			}
		})
	}
}
