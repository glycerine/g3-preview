//go:build g3debug

package euclid

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanApplySolutionAssertionStopsBeforeSubstitution(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	add := func(coefficient int32) *equation {
		t.Helper()
		a, c := numeral.NewInt(coefficient), numeral.NewInt(0)
		eq, e := s.addEquation([]numeral.Int{a}, []uint32{0}, &c, nil, nil, true)
		euclidMust(t, e)
		euclidMust(t, z.Del(&a))
		euclidMust(t, z.Del(&c))
		return eq
	}
	solved := add(-1)
	euclidMust(t, s.promoteSolvedEquation(0, 0))
	target := add(2)
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
			t.Fatal("borrowed apply-solution debug owner", budget.Used())
		}
	})
	euclidMust(t, d.Close())
	err = s.applySolutionOnce(0, target, false, false, nullEquationIndex, nullEquationIndex, nil)
	if !errors.Is(err, arena.ErrClosed) || !equalUint32Slices(target.xs, []uint32{0}) || len(target.as) != 1 || len(s.solution) != 1 {
		t.Fatal("source solved-variable assertion before target mutation", err, target.xs, len(target.as), len(s.solution))
	}
	value, err := z.String(&target.as[0])
	if err != nil || value != "2" {
		t.Fatal("stopped target coefficient", value, err)
	}
	_, err = diagnostic.BindProcessDebugContext(old)
	euclidMust(t, err)
	euclidMust(t, s.applySolutionOnce(0, target, false, false, nullEquationIndex, nullEquationIndex, nil))
	if len(target.xs) != 0 || len(target.as) != 0 {
		t.Fatal("restored source substitution", target.xs, target.as)
	}
	euclidMust(t, s.Close())
	if !solved.closed || !target.closed {
		t.Fatal("source equation owners survived close")
	}
}
