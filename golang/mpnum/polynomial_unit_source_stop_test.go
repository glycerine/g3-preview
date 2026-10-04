// Source: src/math/polynomial/polynomial.cpp:810-851. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestMonomialUnitSourceStoppedCauseDoesNotRepeatDEC(t *testing.T) {
	m := monomialFixture(t)
	b := m.memory.Budget()
	b.FailAfter(0)
	defer b.FailAfter(-1)
	cause := m.SourceFinalize()
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal("actual ID recycle reservation", cause)
	}
	monomialPhysicalAudit(t, m)
	before := m.memory.State()
	if retry := m.SourceFinalize(); retry != cause || m.memory.State() != before {
		t.Fatal("source replayed actual consumed unit DEC", cause, retry, before, m.memory.State())
	}
}
