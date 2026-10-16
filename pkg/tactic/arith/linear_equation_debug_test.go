package arith

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLinearEquationCoreSourceSortedDebugGuard(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	equationMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	equationMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	equationMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		equationMust(t, e)
		equationMust(t, debug.Close())
		equationMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatalf("debug owner retained %d bytes", budget.Used())
		}
	})
	m, l := equationManager(t)
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(3)}
	defer func() {
		for i := range as {
			equationMust(t, m.Integers().Del(&as[i]))
		}
	}()
	eq, err := l.mkCore(as, []uint32{4, 2})
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) || eq != nil {
			t.Fatalf("source sorted assertion: eq=%v err=%v", eq, err)
		}
		equationMust(t, debug.EnableAssertions(false))
		eq, err = l.mkCore(as, []uint32{4, 2})
		equationMust(t, err)
		equationMust(t, eq.Close())
		return
	}
	equationMust(t, err)
	equationMust(t, eq.Close())
}
