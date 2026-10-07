// Licensed under the MIT license.
// Source mpz.cpp:1412 and scoped_numeral.h:31-33: DEFAULT before assignment.
// Source: src/util/mpz.cpp, src/util/scoped_numeral.h.
package util

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func smt2ScopeFixture(t *testing.T) (*IntegerManager, *arena.Budget, Int) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	var a Int
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	if e = m.SetLimbs(&a, digits); e != nil {
		t.Fatal(e)
	}
	if e = m.Neg(&a); e != nil {
		t.Fatal(e)
	}
	positive := NewInt(19)
	if _, e = m.prepareSMT2(&positive, true, 0); e != nil {
		t.Fatal(e)
	}
	return m, b, a
}

func TestSMT2FreshDefaultCopyFailureNullDestructor(t *testing.T) {
	// Locate a real failed assignment independently. Its Go failure token does
	// not imply that the fresh source nullptr assignment completed.
	for fault := int64(0); fault < 40; fault++ {
		control, cb, ca := smt2ScopeFixture(t)
		var tmp Int
		cb.FailAfter(fault)
		e := control.Copy(&tmp, &ca)
		selected := numericPhysicalReservationRefusal(e) && tmp.Cell == control.FailedNullRef
		cb.FailAfter(-1)
		if e = control.CloseChecked(); e != nil || cb.Used() != 0 {
			t.Fatal(e, cb.Used())
		}
		if !selected {
			continue
		}
		m, b, a := smt2ScopeFixture(t)
		b.FailAfter(fault)
		_, e = m.prepareSMT2(&a, true, 0)
		if !numericPhysicalReservationRefusal(e) || errors.Is(e, arena.ErrInvalidSlot) {
			b.FailAfter(-1)
			m.Close()
			t.Fatalf("fresh source null-pointer destructor must preserve only assignment failure: %v", e)
		}
		integerPhysicalAudit(t, m, &a)
		if state := m.SMT2DisplayState(); state.Pending || state.Guard || state.CleanupCause != nil {
			t.Fatal("source null destructor invented cleanup refusal", state)
		}
		if _, smallErr := m.prepareSMT2(&Int{Val: -7}, true, 0); smallErr != nil {
			t.Fatal("fresh failed copy poisoned unrelated completed local", smallErr)
		}
		b.FailAfter(-1)
		if e = m.CloseChecked(); e != nil || b.Used() != 0 {
			t.Fatal(e, b.Used())
		}
		return
	}
	t.Fatal("missing actual failed-copy pointer prefix")
}

// Public oversized reallocations reach a guarded signed-overflow boundary.
// This is host/source undefined-domain evidence, not valid memory exhaustion.
func TestSMT2CompletedTemporaryGuardedLateFree(t *testing.T) {
	for scenario := 0; scenario < 3; scenario++ {
		bodyFails, hasInner := scenario == 1, scenario == 2
		t.Run([]string{"completed-body", "refused-output-prefix", "completed-inner-before-outer"}[scenario], func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			if e != nil {
				t.Fatal(e)
			}
			m, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{PageSize: 2, Budget: b}}})
			if e != nil {
				t.Fatal(e)
			}
			var a Int
			digits := make([]uint32, 32768)
			digits[0], digits[len(digits)-1] = 17, 1
			if e = m.SetLimbs(&a, digits); e != nil {
				t.Fatal(e)
			}
			if e = m.Neg(&a); e != nil {
				t.Fatal(e)
			}
			// The exact source DEFAULT -> assignment sequence, then scoped destruction.
			// Formatting's inner families are covered by the existing display sweeps.
			if e = m.beginSMT2Display(); e != nil {
				t.Fatal(e)
			}
			if e = m.copySMT2Temporary(&a); e != nil {
				t.Fatal(e)
			}
			if e = m.Neg(&m.Smt2Display.Value); e != nil {
				t.Fatal(e)
			}
			if hasInner {
				if _, e = m.Natural.appendDisplay(0, "(- "); e != nil {
					t.Fatal(e)
				}
				if e = m.Natural.beginIntegerCharacters(65537); e != nil {
					t.Fatal(e)
				}
			}
			ref := m.Smt2Display.Value.Cell
			raw := m.Cell(ref).Backing.Raw
			n, e := r.Size(raw)
			if e != nil || n <= 100000 {
				t.Fatal("real direct temporary extent", n, e)
			}
			first, e := r.Allocate(8)
			if e != nil {
				t.Fatal(e)
			}
			second, e := r.Allocate(8)
			if e != nil {
				t.Fatal(e)
			}
			if e = r.Write(first, 0, 13); e != nil {
				t.Fatal(e)
			}
			if e = r.Write(second, 0, 17); e != nil {
				t.Fatal(e)
			}
			before := r.State()
			delta := int64(-1000000)
			request := uint64(16) + uint64(delta) - 8
			old := first
			if e = r.Reallocate(&first, request); e != arena.ErrBounds || first != old {
				t.Fatal("old caller retained", e)
			}
			prefix := r.State()
			if prefix.GlobalSize != before.GlobalSize || prefix.LocalSize != before.LocalSize+delta || prefix.LocalCount != before.LocalCount+1 {
				t.Fatal("actual realloc prefix", before, prefix)
			}
			if v, e := r.Read(first, 0); e != nil || v != 13 {
				t.Fatal(v, e)
			}
			if b.Used() != integerActualOwnedBytes(m) {
				t.Fatal("whole actual budget graph")
			}
			if e = r.Deallocate(&first); e != nil {
				t.Fatal(e)
			}
			negative := r.State()
			if negative.GlobalSize != prefix.GlobalSize+prefix.LocalSize-16 || negative.LocalSize != 0 {
				t.Fatal("actual first free/sync", negative)
			}
			target := int64(math.MinInt64) + 1000 - negative.GlobalSize + 16
			request = uint64(target) + 16 - 8
			old = second
			if e = r.Reallocate(&second, request); e != arena.ErrBounds || second != old {
				t.Fatal("second caller retained", e)
			}
			if v, e := r.Read(second, 0); e != nil || v != 17 {
				t.Fatal(v, e)
			}
			if e = r.Deallocate(&second); e != nil {
				t.Fatal(e)
			}
			near := r.State()
			if near.GlobalSize != math.MinInt64+1000 || near.LocalSize != 0 {
				t.Fatal("public boundary recipe", near)
			}
			integerPhysicalAudit(t, m, &a)
			length := uint32(7)
			var result error
			var original error
			if bodyFails {
				// Real Go output-workspace reservation at the actual source prefix
				// insertion point; this is not a manufactured destructor error.
				b.FailAfter(0)
				_, original = m.Natural.appendDisplay(0, "(- ")
				if !numericPhysicalReservationRefusal(original) {
					t.Fatal("actual body prefix refusal", original)
				}
				b.FailAfter(-1)
				result = original
			}
			if hasInner {
				original = m.Natural.destroyIntegerCharacters()
				if original != arena.ErrBounds {
					t.Fatal("actual completed inner free", original)
				}
				result = original
			}
			innerStopped := r.State()
			m.finishSMT2Display(&length, &result)
			stopped := r.State()
			if !errors.Is(result, arena.ErrBounds) || (bodyFails && !errors.Is(result, original)) || length != 0 || stopped.GlobalSize != near.GlobalSize || stopped.LocalSize != -int64(n+8) && !hasInner || stopped.GlobalCount != near.GlobalCount || stopped.LocalCount != near.LocalCount {
				t.Fatal("completed source free/sync guard", result, length, near, stopped)
			}
			state := m.SMT2DisplayState()
			if !state.Pending || state.Guard || state.OriginalCause != original || state.CleanupCause != arena.ErrBounds || m.Smt2Display.Value.Cell != ref {
				t.Fatal("actual retained release cursor", state)
			}
			if hasInner {
				if stopped != innerStopped || m.Cell(ref).Released {
					t.Fatal("older temporary destroyed before completed child", stopped, innerStopped)
				}
				if _, e = r.Size(raw); e != nil {
					t.Fatal("older actual physical root lost", e)
				}
			} else if _, e = r.Size(raw); e != ErrRuntimeOwner {
				t.Fatal("physical free not completed", e)
			}
			integerPhysicalAudit(t, m, &a)
			if e = m.SourceFinalize(); e != ErrSMT2DisplayPending || m.SourceFinalizeStage != 0 {
				t.Fatal("parent before lexical unwind", e)
			}
			if hasInner {
				if e = m.CleanupSMT2Display(); e != arena.ErrBounds {
					t.Fatal("outer free after child acknowledgement", e)
				}
				afterOuter := r.State()
				if afterOuter.GlobalSize != stopped.GlobalSize || afterOuter.LocalSize != stopped.LocalSize-int64(n+8) || afterOuter.LocalCount != stopped.LocalCount || afterOuter.GlobalCount != stopped.GlobalCount {
					t.Fatal("actual next outer free", stopped, afterOuter)
				}
				if _, e = r.Size(raw); e != ErrRuntimeOwner {
					t.Fatal("outer physical free", e)
				}
				stopped = afterOuter
				integerPhysicalAudit(t, m, &a)
			}
			if e = m.CleanupSMT2Display(); e != nil || r.State() != stopped || m.SMT2DisplayState().Pending {
				t.Fatal("release acknowledgement replayed source", e, r.State(), stopped)
			}
			integerPhysicalAudit(t, m, &a)
			// The guarded counter remains unchanged. Each other actual raw free may
			// itself stop after physical release; bounded owning continuation consumes it.
			closed := false
			for i := 0; i < 64; i++ {
				e = m.CloseChecked()
				if e == nil {
					closed = true
					break
				}
				if !errors.Is(e, arena.ErrBounds) {
					t.Fatal("owning continuation", e)
				}
			}
			if !closed {
				t.Fatal("bounded actual owning cleanup")
			}
			if e = r.Close(); e != nil || b.Used() != 0 {
				t.Fatal(e, b.Used())
			}
		})
	}
}
