package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicSexprRootIndexAssertionBeforeSyntaxAndErasedGate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			if enabled && !polynomialSourceDebug {
				t.Skip("source assertion compiled out")
			}
			c, _, sx, sy, _, _ := algebraicSexprRootFixture(t)
			invalid := sxPolySym(t, sx, sy, "y")
			if polynomialSourceDebug {
				d := diagnostic.BorrowProcessDebugContext()
				old, e := d.AssertionsEnabled()
				serviceMust(t, e)
				serviceMust(t, d.EnableAssertions(enabled))
				defer func() { serviceMust(t, d.EnableAssertions(old)) }()
				if enabled {
					polynomialAssertionActionsFixture(t, "T")
				}
			}
			var out Algebraic
			body := c.MkSexprRoot(sx, invalid, 0, &out)
			if enabled {
				var exception z3exception.DefaultException
				if !errors.As(body, &exception) || c.isolateTmp1.slot != 0 || c.rootConstruction.active {
					t.Fatal("index assertion after workspace/syntax", body)
				}
			} else {
				var syntax *SexprPolynomialError
				if !errors.As(body, &syntax) || c.isolateTmp1.slot == 0 {
					t.Fatal("erased index assertion did not reach syntax first", body)
				}
				serviceMust(t, c.CleanupRetained())
			}
		})
	}
}
func TestAlgebraicPolynomialRootSourceAssertionsBeforeWorkspace(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion compiled out")
	}
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 1, 0}}})
	polynomialAssertionActionsFixture(t, "TT")
	var out Algebraic
	for _, index := range []uint32{0, 1} {
		body := c.MkPolynomialRoot(pm, p, index, &out)
		var exception z3exception.DefaultException
		if !errors.As(body, &exception) || c.isolateTmp1.slot != 0 || c.rootConstruction.active || pm.algebraicIsolationBorrowers != 0 {
			t.Fatal("source assertion after conversion", index, body)
		}
	}
}
