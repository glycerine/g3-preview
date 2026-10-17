package euclid

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanPivotReadsSourceAssertionSwitches(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 0, nullEquationIndex)
	empty, err := newSolverSetup(m)
	euclidMust(t, err)
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
			t.Fatal("borrowed pivot debug owner", budget.Used())
		}
	})
	euclidMust(t, d.Close())
	one := numeral.NewInt(1)
	better, err := s.betterPivot(&one, 0, 1)
	if diagnostic.DebugBuild {
		if better || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("is_better source assertion read", better, err)
		}
	} else if !better || err != nil {
		t.Fatal("erased is_better source assertion", better, err)
	}
	found, err := empty.selectNextVar()
	if found || err != nil {
		t.Fatal("empty source branch bypasses assertion", found, err)
	}
	s.nextX = 42
	found, err = s.selectNextVar()
	if diagnostic.DebugBuild {
		if found || !errors.Is(err, arena.ErrClosed) || s.nextX != 42 {
			t.Fatal("post-trim assertion precedes pivot reset", found, err, s.nextX)
		}
	} else if !found || err != nil || s.nextX != 0 {
		t.Fatal("erased post-trim assertion", found, err, s.nextX)
	}
	_, err = diagnostic.BindProcessDebugContext(old)
	euclidMust(t, err)
	found, err = s.selectNextVar()
	if !found || err != nil || s.nextX != 0 {
		t.Fatal("valid source pivot after owner restoration", found, err, s.nextX)
	}
	euclidMust(t, s.Close())
	euclidMust(t, empty.Close())
}
