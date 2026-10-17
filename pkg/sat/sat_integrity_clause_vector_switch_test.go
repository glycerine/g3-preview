package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Source: src/sat/sat_integrity_checker.cpp:102-110. The assertion switch is
// read only at a reached clause, not at entry to an empty vector traversal.
func TestSATIntegrityEmptyClauseVectorSkipsSourceAssertionSwitch(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	checker, err := newSATIntegrityChecker(g.lifecycle(), g)
	satOK(t, err)
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.Close())
	if yes, err := checker.checkClauseVector(nil); err != nil || !yes {
		t.Fatal("empty vector read the debug owner", yes, err)
	}
	yes, err := checker.checkClauseVector([]StoredClauseRef{{}})
	if satStorageSourceDebug {
		if yes || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("reached clause skipped the debug owner", yes, err)
		}
	} else if err != nil || !yes {
		t.Fatal("release build evaluated an erased assertion", yes, err)
	}
}
