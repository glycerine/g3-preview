//go:build g3debug

package euclid

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanJustificationAndRecentSolutionAssertionReads(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	public, err := NewSolver(m)
	euclidMust(t, err)
	one := numeral.NewInt(1)
	euclidMust(t, public.AssertEq(nil, nil, &one, nullEquationIndex))
	if public.setup.inconsistent == nullEquationIndex {
		t.Fatal("constant contradiction did not publish equation")
	}
	setup, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = setup.mkVar(false)
	euclidMust(t, err)
	minusOne, zero := numeral.NewInt(-1), numeral.NewInt(0)
	_, err = setup.addEquation([]numeral.Int{minusOne}, []uint32{0}, &zero, nil, nil, true)
	euclidMust(t, err)
	euclidMust(t, setup.promoteSolvedEquation(0, 0))
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	euclidMust(t, err)
	d, err := diagnostic.NewDebugContext(region, budget)
	euclidMust(t, err)
	old, err := diagnostic.BindProcessDebugContext(d)
	euclidMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(old)
		euclidMust(t, e)
		euclidMust(t, d.Close())
		euclidMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal("borrowed Euclidean assertion owner", budget.Used())
		}
	})
	euclidMust(t, d.Close())
	js, err := public.GetJustification()
	if js != nil || !errors.Is(err, arena.ErrClosed) {
		t.Fatal("get_justification source assertion switch", js, err)
	}
	err = setup.substituteMostRecentSolution(0)
	if !errors.Is(err, arena.ErrClosed) || len(setup.solution) != 1 {
		t.Fatal("recent-solution source assertion before trace", err, len(setup.solution))
	}
	_, err = diagnostic.BindProcessDebugContext(old)
	euclidMust(t, err)
	js, err = public.GetJustification()
	if err != nil || len(js) != 0 {
		t.Fatal("restored contradictory explanation", js, err)
	}
	euclidMust(t, setup.substituteMostRecentSolution(0))
	euclidMust(t, setup.Close())
	euclidMust(t, public.Close())
	euclidMust(t, z.Del(&minusOne))
	euclidMust(t, z.Del(&zero))
	euclidMust(t, z.Del(&one))
}
