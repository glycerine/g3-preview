package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATCleanerInvariantUsesProcessDebugGate(t *testing.T) {
	g, _ := satGCFixture(t, 3)
	c := g.cleaner
	v := g.lifecycle().propagation.variables
	priorDecision := v.decision
	defer func() { v.decision = priorDecision }()
	v.decision = nil
	budget := arena.NewBudget(^uint64(0))
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	satOK(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	satOK(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	satOK(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		satOK(t, e)
		satOK(t, debug.Close())
		satOK(t, region.Close())
	})
	before := g.lifecycle().propagation.limit.Steps
	satOK(t, c.assertInvariant())
	satOK(t, debug.EnableAssertions(false))
	satOK(t, c.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	satOK(t, c.assertInvariant())
	satOK(t, debug.EnableDebug("cleaner_bug"))
	if satStorageSourceDebug {
		if err = c.assertInvariant(); !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("enabled nested checker assertion: %v", err)
		}
		if got := g.lifecycle().propagation.limit.Steps; got != before+1 {
			t.Fatalf("gated checker calls: got %d, want %d", got, before+1)
		}
	} else if got := g.lifecycle().propagation.limit.Steps; got != before {
		t.Fatalf("release CASSERT reached checker: %d", got)
	}
	v.decision = priorDecision
	satOK(t, c.assertInvariant())
	satOK(t, g.lifecycle().propagation.assign(0, satJustification{}))
	_, err = c.run(true)
	satOK(t, err)
	unbound, _ := satCleanerFixture(t, 2)
	err = unbound.assertInvariant()
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATMissingIntegrityBinding) {
			t.Fatalf("enabled checker missing owner binding: %v", err)
		}
	} else {
		satOK(t, err)
	}
}
