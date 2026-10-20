package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// With exponent zero, the source monomial factory returns its actual unit
// without consulting x. This gives a defined continuation after2539 without
// fabricating a variable, mutating a reference count or retrying a failed body.
func TestPolynomialVariableConstructorAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := action
			if action == "disabled" {
				choices = ""
			}
			output := polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			c := polynomialCellFixture(t)
			before := c.monomials.nextVar
			p, e := c.MkVariable(^uint32(0), 0)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || p != (Polynomial{}) || c.singletonReference || c.pending.value != (Polynomial{}) {
					t.Fatal(e, p)
				}
				_, e = output.Seek(0, 0)
				serviceMust(t, e)
				text, e := io.ReadAll(output)
				serviceMust(t, e)
				if !strings.Contains(string(text), "Line: 2539\nis_valid(x)") {
					t.Fatal(string(text))
				}
			} else {
				serviceMust(t, e)
				term, e := c.TermMonomial(p, 0)
				serviceMust(t, e)
				coefficient, e := c.Coefficient(p, 0)
				serviceMust(t, e)
				if term != c.monomials.unit || coefficient != unumeral.NewInt(1) {
					t.Fatal(term, coefficient)
				}
				serviceMust(t, c.CleanupValue(&p))
			}
			if c.monomials.nextVar != before {
				t.Fatal("source ignored zero-exponent variable changed variable table")
			}
		})
	}
}

func TestPolynomialConstructorClosedAssertionOwnerPaths(t *testing.T) {
	for _, which := range []string{"variable", "rational-empty", "linear-empty", "rational-linear-empty"} {
		t.Run(which, func(t *testing.T) {
			c := polynomialCellFixture(t)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetInt64(&q, 0))
			t.Cleanup(func() { serviceMust(t, c.config.Rationals.Del(&q)) })
			var p Polynomial
			t.Cleanup(func() {
				if p != (Polynomial{}) && p != c.zero && p != c.unit {
					serviceMust(t, c.CleanupValue(&p))
				}
			})
			polynomialClosedCellAssertionContext(t)
			zero := unumeral.Int{}
			var body error
			switch which {
			case "variable":
				p, body = c.MkVariable(0, 1)
			case "rational-empty":
				body = c.rational2Numeral(nil, 0)
			case "linear-empty":
				p, body = c.MkLinearTake(nil, nil, &zero)
			case "rational-linear-empty":
				p, body = c.MkRationalLinear(nil, nil, &q)
			}
			if polynomialSourceDebug {
				if !errors.Is(body, arena.ErrClosed) || p != (Polynomial{}) || c.singletonReference || c.linearConstantActive || len(c.rationalTerms.values) != 0 || len(c.linearTerms.values) != 0 {
					t.Fatal(which, p, body)
				}
			} else {
				serviceMust(t, body)
			}
		})
	}
}

// Fractions are real caller-owned rationals. Source Continue copies their
// numerators exactly; this checks diagnostics, not mathematical fraction-to-
// integer conversion, and never changes the original rational payload.
func TestPolynomialRationalConstructorAssertionActions(t *testing.T) {
	for _, site := range []string{"coefficient", "constant"} {
		t.Run(site, func(t *testing.T) {
			for _, action := range []string{"T", "C", "disabled"} {
				t.Run(action, func(t *testing.T) {
					choices := action
					if action == "disabled" {
						choices = ""
					}
					output := polynomialAssertionActionsFixture(t, choices)
					if polynomialSourceDebug && action == "disabled" {
						serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
					}
					c := polynomialCellFixture(t)
					fraction := unumeral.NewRational()
					zero := unumeral.NewRational()
					serviceMust(t, c.config.Rationals.SetFraction(&fraction, 3, 2))
					defer func() {
						serviceMust(t, c.config.Rationals.Del(&fraction))
						serviceMust(t, c.config.Rationals.Del(&zero))
					}()
					var p Polynomial
					var body error
					line := "2626\nc.is_int()"
					if site == "constant" {
						p, body = c.MkRationalLinear(nil, nil, &fraction)
					} else {
						line = "2560\nas[i].is_int()"
						p, body = c.MkRationalLinear([]unumeral.Rational{fraction}, []uint32{0}, &zero)
					}
					if polynomialSourceDebug && action == "T" {
						if !errors.Is(body, ErrPolynomialAssertion) || p != (Polynomial{}) || c.linearConstantActive || len(c.rationalTerms.values) != 0 {
							t.Fatal(p, body)
						}
						_, e := output.Seek(0, 0)
						serviceMust(t, e)
						text, e := io.ReadAll(output)
						serviceMust(t, e)
						if !strings.Contains(string(text), "Line: "+line) {
							t.Fatal(string(text))
						}
					} else {
						serviceMust(t, body)
						coefficient, e := c.Coefficient(p, 0)
						serviceMust(t, e)
						if coefficient != unumeral.NewInt(3) {
							t.Fatal(coefficient)
						}
						if len(c.rationalTerms.values) != 0 || len(c.linearTerms.values) != 0 || c.linearConstantActive {
							t.Fatal("source converted fields not retired")
						}
						serviceMust(t, c.CleanupValue(&p))
					}
					if unumeral.IsIntegerRational(&fraction) {
						t.Fatal("caller fraction was modified")
					}
				})
			}
		})
	}
}
