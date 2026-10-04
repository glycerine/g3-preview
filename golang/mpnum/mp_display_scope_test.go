// Licensed under the MIT license.
// Source: mpq.cpp156-184 raw locals do not have numeral destructors.
package mpnum

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func rawDisplayFixture(t *testing.T) (*RationalManager, *arena.Budget, Rational) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	cfg := IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{PageSize: 2, Budget: b}}}
	m, e := NewRationalManager(RationalConfig{Integers: cfg})
	if e != nil {
		t.Fatal(e)
	}
	a := NewRational()
	if e = m.z.SetString(&a.num, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = m.z.SetString(&a.den, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	return m, b, a
}

func TestRawDecimalFailedDenominatorRetainsCopiedNumerator(t *testing.T) {
	// Independently locate the real allocation prefix between the two source
	// Copy calls, rather than relying on an allocator ordinal constant.
	for fault := int64(0); fault < 40; fault++ {
		control, cb, ca := rawDisplayFixture(t)
		var num, den Int
		cb.FailAfter(fault)
		e1 := control.z.Copy(&num, &ca.num)
		var e2 error
		if e1 == nil {
			e2 = control.z.Copy(&den, &ca.den)
		}
		selected := e1 == nil && e2 != nil && num.cell != 0
		cb.FailAfter(-1)
		control.Close()
		if cb.Used() != 0 {
			t.Fatal("control cleanup", cb.Used())
		}
		if !selected {
			continue
		}
		m, b, a := rawDisplayFixture(t)
		before := m.z.cells.Stats().Live
		b.FailAfter(fault)
		_, e := m.prepareDecimal(&a, 0, false)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("source denominator allocation refusal", fault, e)
		}
		if got := m.z.cells.Stats().Live; got != before+1 {
			b.FailAfter(-1)
			m.Close()
			t.Fatalf("raw NUM must survive before explicit end label: got live%d want%d", got, before+1)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
		if !m.z.rawDisplay.active || m.z.rawDisplay.cursor != 0 || !errors.Is(m.z.rawDisplay.original, e) {
			t.Fatal("raw source body/end-label state", m.z.rawDisplay)
		}
		if _, blocked := m.prepareDecimal(&a, 0, false); blocked != ErrRawDisplayPending {
			t.Fatal("actual raw-scope reuse", blocked)
		}
		// Unrelated integer reads and caller deletion do not reuse the raw scope.
		if _, readErr := m.z.Compare(&a.num, &a.den); readErr != nil {
			t.Fatal("unrelated comparison", readErr)
		}
		b.FailAfter(-1)
		if cleanErr := m.z.CleanupRawDisplay(); cleanErr != nil {
			t.Fatal("explicit owning abort", cleanErr)
		}
		if m.z.rawDisplay.active || m.z.cells.Stats().Live != before {
			t.Fatal("actual raw roots not released")
		}
		if pendingErr := m.z.CleanupPendingAllocation(); pendingErr != nil {
			t.Fatal("separate physical allocation continuation", pendingErr)
		}
		if _, outputErr := m.prepareDecimal(&a, 0, false); outputErr != nil {
			t.Fatal("reuse after owning abort", outputErr)
		}
		if m.z.rawDisplay.active {
			t.Fatal("successful source end label not completed")
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal("owning raw-local cleanup", b.Used())
		}
		return
	}
	t.Fatal("no actual denominator-copy refusal prefix")
}
