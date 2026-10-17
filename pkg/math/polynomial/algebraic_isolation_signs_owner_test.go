package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"reflect"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicIsolationSignsPreservesOlderPolynomialEvaluator(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {2, [3]uint32{}}})
	wrapped := polynomialLexWrappedPolynomial(t, pm)
	rv := polynomialEvalRationalMap{manager: c.config.Rationals}
	for i := range rv.values {
		rv.values[i] = unumeral.NewRational()
	}
	oldOut := unumeral.NewRational()
	defer func() {
		serviceMust(t, c.config.Rationals.Del(&oldOut))
		for i := range rv.values {
			serviceMust(t, c.config.Rationals.Del(&rv.values[i]))
		}
	}()
	older := pm.EvaluateRational(wrapped, &rv, &oldOut)
	if older == nil || !pm.evaluation.active || !pm.lexPending {
		t.Fatal("did not reach older lexical-sort failure", older)
	}
	generation := pm.evaluation.lexGeneration
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	signs := []int{71}
	e = c.IsolateMultivariateRootsWithSigns(pm, p, &algebraicIsolationAssignments{}, roots, &signs)
	if !errors.Is(e, ErrPolynomialPending) || len(signs) != 1 || signs[0] != 71 {
		t.Fatal("older evaluation refusal", e, signs)
	}
	serviceMust(t, c.CleanupRetained())
	if !pm.evaluation.active || !pm.lexPending || pm.evaluation.lexGeneration != generation {
		t.Fatal("new signs cleanup consumed older evaluator")
	}
	serviceMust(t, pm.CleanupEvaluation())
}

type algebraicIsolationSignsProbe struct {
	*algebraicIsolationAssignments
	core   *AlgebraicRootCore
	pm     *MultivariatePolynomialManager
	p      Polynomial
	roots  *AlgebraicVector
	t      *testing.T
	probes uint32
}

func (v *algebraicIsolationSignsProbe) Contains(x uint32) (bool, error) {
	if v.core.isolationSigns.constructed && v.core.signEvaluationRunning {
		v.probes++
		if e := v.core.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
			v.t.Fatal("signs callback cleanup", e)
		}
		if e := v.core.Close(); !errors.Is(e, ErrAlgebraicPending) {
			v.t.Fatal("signs callback algebraic close", e)
		}
		if e := v.pm.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
			v.t.Fatal("signs callback polynomial close", e)
		}
		if e := v.core.IsolateMultivariateRoots(v.pm, v.p, v, v.roots); !errors.Is(e, ErrAlgebraicPending) {
			v.t.Fatal("signs callback isolation", e)
		}
		var signs []int
		if e := v.core.IsolateMultivariateRootsWithSigns(v.pm, v.p, v, v.roots, &signs); !errors.Is(e, ErrAlgebraicPending) {
			v.t.Fatal("recursive signs callback", e)
		}
	}
	return v.algebraicIsolationAssignments.Contains(x)
}
func TestAlgebraicIsolationSignsExecutingOwnerRefusals(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}})
	values := algebraicIsolationAssignmentsFixture(t, c, up, false)
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	probe := algebraicIsolationSignsProbe{algebraicIsolationAssignments: values, core: c, pm: pm, p: p, roots: roots, t: t}
	var signs []int
	serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, &probe, roots, &signs))
	if probe.probes == 0 {
		t.Fatal("did not reach sampling assignment callback")
	}
	if len(signs) != 2 || signs[0] != -1 || signs[1] != 1 {
		t.Fatal(signs)
	}
}

func TestAlgebraicIsolationSignsNonzeroAssertionAndErasedGate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			if enabled && !polynomialSourceDebug {
				t.Skip("source assertions compiled out")
			}
			c, _, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}})
			roots, e := c.NewVector()
			serviceMust(t, e)
			defer func() { serviceMust(t, roots.Close()) }()
			// This intentionally invalid mapping assigns the univariate root variable
			// to zero. Isolation ignores it, but every sampled evaluation is zero.
			values := &algebraicIsolationAssignments{assigned: [3]bool{false, false, true}}
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
			signs := []int{71}
			body := c.IsolateMultivariateRootsWithSigns(pm, p, values, roots, &signs)
			if enabled {
				var exception z3exception.DefaultException
				if !errors.As(body, &exception) || len(signs) != 1 {
					t.Fatal("source nonzero assertion after publication", body, signs)
				}
			} else {
				serviceMust(t, body)
				if !reflect.DeepEqual(signs, []int{71, 0, 0}) {
					t.Fatal("erased nonzero assertion", signs)
				}
			}
			if c.isolationSigns.active || c.isolationSigns.constructed || roots.Len() != 1 {
				t.Fatal("assertion did not unwind actual sample", roots.Len())
			}
		})
	}
}
