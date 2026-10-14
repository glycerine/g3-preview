package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinarySelectionScopedIntegerCleanupRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, qm.Z.SetLimbs(&m.selectIntTemporary, []uint32{3, 0, 1}))
	m.selectIntOwner = qm
	m.selectIntPending = true
	ref := m.selectIntTemporary.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.finishSelectIntegerTemporarySource(); !errors.Is(e, arena.ErrBounds) || !m.selectIntPending || m.selectIntTemporary.Cell != ref {
		t.Fatal("source scoped integer refusal lost owner", e, m.selectIntTemporary)
	}
	if e := m.open(); !errors.Is(e, ErrBinarySelectPending) {
		t.Fatal("selection reentered stopped source scope", e)
	}
	if e := m.CleanupSelectIntegerTemporary(); !errors.Is(e, arena.ErrBounds) || !m.selectIntPending {
		t.Fatal("owning retry lost stopped integer", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupSelectIntegerTemporary())
	if m.selectIntPending || m.selectIntOwner != nil || m.selectIntTemporary.Cell != 0 {
		t.Fatal("owning cleanup did not drain scoped integer")
	}
	q := NewRational()
	serviceMust(t, qm.SetString(&q, "18446744073709551617/2"))
	var out Int
	serviceMust(t, m.rationalIntegerBound(qm, &q, &out, false))
	if got := integerString(t, m.Z, &out); got != "9223372036854775809" {
		t.Fatal("selection did not resume", got)
	}
	if m.selectIntPending || m.selectIntTemporary.Cell != 0 {
		t.Fatal("successful scoped destructor retained integer")
	}
	serviceMust(t, m.Z.Del(&out))
	serviceMust(t, qm.Del(&q))
	serviceMust(t, qm.Z.SetLimbs(&m.selectIntTemporary, []uint32{5, 0, 1}))
	m.selectIntOwner = qm
	m.selectIntPending = true
	ref = m.selectIntTemporary.Cell
	original = qm.Z.Cell(ref)
	bad = original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.CloseChecked(); !errors.Is(e, arena.ErrBounds) || !m.selectIntPending || m.closed {
		t.Fatal("manager close discarded stopped borrowed scope", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CloseChecked())
}
