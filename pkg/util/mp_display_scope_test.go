// Licensed under the MIT license.
// Source: src/util/mpq.cpp:156-184 raw locals do not have numeral destructors.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func rawDisplayFixture(t *testing.T) (*RationalManager, *arena.Budget, Rational) {
	return rawDisplayFixtureProfile(t, SmallAllocatorProfile{})
}
func rawDisplayFixtureProfile(t *testing.T, profile SmallAllocatorProfile) (*RationalManager, *arena.Budget, Rational) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	cfg := IntegerConfig{Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{PageSize: 2, Budget: b}}}
	m, e := NewRationalManager(RationalConfig{Integers: cfg})
	if e != nil {
		t.Fatal(e)
	}
	a := NewRational()
	if e = m.Z.SetString(&a.Num, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = m.Z.SetString(&a.Den, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	return m, b, a
}

func TestRawDecimalFailedDenominatorRetainsCopiedNumerator(t *testing.T) {
	// Independently locate the real allocation prefix between the two source
	// Copy calls, rather than relying on an allocator ordinal constant.
	for fault := int64(0); fault < 40; fault++ {
		control, cb, ca := rawDisplayFixtureProfile(t, SmallAllocatorProfile{PointerBits: 64, Debug: true})
		var num, den Int
		cb.FailAfter(fault)
		e1 := control.Z.Copy(&num, &ca.Num)
		var e2 error
		if e1 == nil {
			e2 = control.Z.Copy(&den, &ca.Den)
		}
		selected := e1 == nil && e2 != nil && num.Cell != 0
		cb.FailAfter(-1)
		control.Close()
		if cb.Used() != 0 {
			t.Fatal("control cleanup", cb.Used())
		}
		if !selected {
			continue
		}
		m, b, a := rawDisplayFixtureProfile(t, SmallAllocatorProfile{PointerBits: 64, Debug: true})
		before := m.Z.Cells.Stats().Live
		b.FailAfter(fault)
		_, e := m.prepareDecimal(&a, 0, false)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("source denominator allocation refusal", fault, e)
		}
		if got := m.Z.Cells.Stats().Live; got != before+1 {
			b.FailAfter(-1)
			m.Close()
			t.Fatalf("raw NUM must survive before explicit end label: got live%d want%d", got, before+1)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
		if !m.Z.RawDisplay.active || m.Z.RawDisplay.cursor != 0 || !errors.Is(m.Z.RawDisplay.original, e) {
			t.Fatal("raw source body/end-label state", m.Z.RawDisplay)
		}
		if _, blocked := m.prepareDecimal(&a, 0, false); blocked != ErrRawDisplayPending {
			t.Fatal("actual raw-scope reuse", blocked)
		}
		// Unrelated integer reads and caller deletion do not reuse the raw scope.
		if _, readErr := m.Z.Compare(&a.Num, &a.Den); readErr != nil {
			t.Fatal("unrelated comparison", readErr)
		}
		b.FailAfter(-1)
		if cleanErr := m.Z.CleanupRawDisplay(); cleanErr != nil {
			t.Fatal("explicit owning abort", cleanErr)
		}
		if m.Z.RawDisplay.active || m.Z.Cells.Stats().Live != before {
			t.Fatal("actual raw roots not released")
		}
		if pendingErr := m.Z.CleanupPendingAllocation(); pendingErr != nil {
			t.Fatal("separate physical allocation continuation", pendingErr)
		}
		if _, outputErr := m.prepareDecimal(&a, 0, false); outputErr != nil {
			t.Fatal("reuse after owning abort", outputErr)
		}
		if m.Z.RawDisplay.active {
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
