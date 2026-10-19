package polynomial

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialPSCAssertionRuntimeActions(t *testing.T) {
	for _, line := range []int32{5672, 5673, 5699, 5700, 5711, 5716, 5727, 5755, 5790, 5805, 5806} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprintf("%d/%s", line, action), func(t *testing.T) {
				polynomialAssertionActionsFixture(t, action)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				reads := 0
				e := polynomialPSCSourceAssert(line, "false", func() (bool, error) {
					reads++
					return false, nil
				})
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrPolynomialAssertion) {
						t.Fatal("assertion did not throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				want := 0
				if polynomialSourceDebug && action != "disabled" {
					want = 1
				}
				if reads != want {
					t.Fatal("assertion predicate read count", reads, want)
				}
			})
		}
	}
}

func TestPolynomialPSCAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			out, e := NewPolynomialRefVector(m)
			serviceMust(t, e)
			defer func() { serviceMust(t, out.Close()) }()
			serviceMust(t, out.Push(m.unit))
			polynomialAssertionActionsFixture(t, strings.Repeat(action, 16))
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = m.PSCChain(m.unit, m.unit, 0, out)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) {
					t.Fatal("PSC assertion did not throw", e)
				}
			} else {
				serviceMust(t, e)
			}
			n, sizeErr := out.Size()
			serviceMust(t, sizeErr)
			p, getErr := out.Get(0)
			serviceMust(t, getErr)
			want := m.zero
			if polynomialSourceDebug && action == "T" {
				want = m.unit
			}
			if n != 1 || p != want {
				t.Fatal("assertion action/output reset ordering", n, p, want)
			}
		})
	}
}
