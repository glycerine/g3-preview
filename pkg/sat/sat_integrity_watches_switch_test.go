package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Source: src/sat/sat_integrity_checker.cpp:155-196. DEBUG_CODE still visits
// every watch-list slot, but an empty watch vector reaches no SASSERT site.
func TestSATIntegrityEmptyWatchVectorSkipsSourceAssertionSwitch(t *testing.T) {
	g, _ := satGCFixture(t, 0)
	checker, err := newSATIntegrityChecker(g.lifecycle(), g)
	satOK(t, err)
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.Close())
	if yes, err := checker.checkWatches(); err != nil || !yes {
		t.Fatal("empty watches read the debug owner", yes, err)
	}
	if !satStorageSourceDebug {
		return
	}
	v := g.lifecycle().propagation.variables
	_, err = v.mkVar(false, false)
	satOK(t, err)
	yes, err := checker.checkWatches()
	if yes || !errors.Is(err, arena.ErrClosed) {
		t.Fatal("reached watch slot skipped the debug owner", yes, err)
	}
}
