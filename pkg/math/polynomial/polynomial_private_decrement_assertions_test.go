package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// The real full imp destructor must reach1371 before decrementing its private
// zero. Letting that DEC run would delete zero and stop later at the unit's
// coefficient assertion, so the concrete source cursor/count distinguishes it.
func TestPolynomialPrivateDestructorDecrementAssertionGate(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	zero, unit := c.zero, c.unit
	zr, e := c.readWord(zero, 0)
	serviceMust(t, e)
	ur, e := c.readWord(unit, 0)
	serviceMust(t, e)
	polynomialClosedCellAssertionContext(t)
	body := m.SourceFinalize()
	if !polynomialSourceDebug {
		serviceMust(t, body)
		if !m.sourceFinalized || m.bodyStage != 11 || !c.sourceDecDone[0] || !c.sourceDecDone[1] {
			t.Fatal("native source destructor did not complete")
		}
		return
	}
	if !errors.Is(body, arena.ErrClosed) || m.sourceCause != body || !m.sourceStarted || !c.sourceStarted || m.bodyStage != 0 || c.sourceStage != 0 || c.sourceDecDone != ([2]bool{}) || c.sourceDeleteNeeded != ([2]bool{}) {
		t.Fatal("private DEC assertion was not the first stopped source frontier", body, m.bodyStage, c.sourceStage, c.sourceDecDone, c.sourceDeleteNeeded)
	}
	afterZero, e := c.readWord(zero, 0)
	serviceMust(t, e)
	afterUnit, e := c.readWord(unit, 0)
	serviceMust(t, e)
	if zr != 1 || ur != 1 || afterZero != zr || afterUnit != ur {
		t.Fatal(zr, ur, afterZero, afterUnit)
	}
	zl, e := c.polynomialLocation(zero)
	serviceMust(t, e)
	ul, e := c.polynomialLocation(unit)
	serviceMust(t, e)
	if zl.deletionStarted || ul.deletionStarted || zl.deleteStage != 0 || ul.deleteStage != 0 {
		t.Fatal("source assertion advanced into deletion")
	}
	if again := m.SourceFinalize(); again != body {
		t.Fatal("source stop was replayed", again, body)
	}
	// The fixture restores the process owner before explicit owning cleanup;
	// cleanup retires the unchanged actual private roots, not a resumed body.
}
