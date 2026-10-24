package core

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSimplifyCountedReleaseAfterFailedCleanupReconstruction(t *testing.T) {
	m, c, g := simplifyFixture(t)
	r, err := NewSimplifyTactic(m, c, g, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, r.Close()) })
	simplifyMust(t, r.IncRef())
	m.Budget().FailAfter(0)
	err = r.Cleanup()
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.open() == nil {
		t.Fatal("expected a stopped, non-executable reconstruction", err)
	}
	// A source reference-counted destructor does not require an executable
	// implementation. This release must retire the real partial owner.
	simplifyMust(t, r.DecRef())
	if !r.closed {
		t.Fatal("final reference did not destroy the stopped owner")
	}
}
