package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestSimpleAlgebraicValuesOwnedCopiesDuplicatesAndEvaluation(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	values, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, values.Close()) }()
	m, e := values.Manager()
	serviceMust(t, e)
	if m != c {
		t.Fatal("mapping manager identity")
	}
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	serviceMust(t, values.Push(0, &root))
	serviceMust(t, c.Del(&root))
	var three Algebraic
	n := unumeral.NewInt(3)
	serviceMust(t, c.SetInteger(&three, &n))
	serviceMust(t, values.Push(0, &three))
	serviceMust(t, c.Del(&three))
	first, e := values.Algebraic(0)
	serviceMust(t, e)
	algebraicExpectBetween(t, c, first, 1, 2)
	yes, e := values.Contains(0)
	serviceMust(t, e)
	no, e := values.Contains(1)
	serviceMust(t, e)
	if !yes || no {
		t.Fatal("contains", yes, no)
	}
	// Evaluate x^2-2 using the real mapping and recursive algebraic evaluator.
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}})
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	serviceMust(t, pm.EvaluateAlgebraic(p, c, values, &out))
	zero, e := c.IsZero(&out)
	serviceMust(t, e)
	if !zero {
		t.Fatal("mapping evaluation of sqrt(2)^2-2")
	}
	if e = c.Close(); !errors.Is(e, ErrAlgebraicLive) {
		t.Fatal("mapping did not retain numerical owner", e)
	}
}

func TestSimpleAlgebraicValuesMissingAndFailedPushPrefix(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	values, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "C")
	missing, e := values.Algebraic(99)
	if polynomialSourceDebug {
		serviceMust(t, e)
		if missing == nil || *missing != (Algebraic{}) {
			t.Fatal("source static zero", missing)
		}
	} else if !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
		t.Fatal("missing lookup diagnostic", e)
	}
	var a Algebraic
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&a, &one))
	c.config.Rationals.Budget().FailAfter(0)
	e = values.Push(7, &a)
	c.config.Rationals.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("expected actual value-copy refusal", e)
	}
	contains, e := values.Contains(7)
	serviceMust(t, e)
	if !contains || len(values.variables) != 1 || values.values.Len() != 0 {
		t.Fatal("source variable/value prefix changed")
	}
	if _, e = values.Algebraic(7); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("failed copy exposed nonexistent cell", e)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, values.Close())
	serviceMust(t, c.Del(&a))
	// Failed-source prefix is not rolled back; a fresh mapping is independently reusable.
	fresh, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	serviceMust(t, fresh.Push(1, &a))
	serviceMust(t, fresh.Close())
}
