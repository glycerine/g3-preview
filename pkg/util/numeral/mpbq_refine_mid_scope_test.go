package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryRefineRetainsMidpointAfterActualBodyFailure(t *testing.T) {
	floor := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 70), big.NewInt(17))
	ceil := new(big.Int).Add(floor, big.NewInt(1))
	numerator := new(big.Int).Add(new(big.Int).Mul(floor, big.NewInt(3)), big.NewInt(1))
	found := false
	for fault := int64(0); fault < 256 && !found; fault++ {
		m, budget := binaryManager(t)
		qm := m.Diagnostics.Manager
		q := NewRational()
		l, u := NewBinaryRational(), NewBinaryRational()
		serviceMust(t, qm.SetString(&q, numerator.String()+"/3"))
		serviceMust(t, m.Z.SetString(&l.Num, floor.String()))
		serviceMust(t, m.Z.SetString(&u.Num, ceil.String()))
		var receipt BinaryRefineInvocationReceipt
		budget.FailAfter(fault)
		e := m.RefineUpperWithReceipt(qm, &q, &l, &u, &receipt)
		if e != nil && m.refinePending && m.refineMid.Num.Cell != 0 {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal("unexpected midpoint body failure", fault, e)
			}
			found = true
		}
		budget.FailAfter(-1)
		if found {
			if e := m.open(); !errors.Is(e, ErrBinaryRefinePending) {
				t.Fatal("new binary operation accepted retained midpoint", e)
			}
			if !m.RefineInvocationPending(&receipt) {
				t.Fatal("actual midpoint owner was not issued to its caller")
			}
			before, pending := receipt, m.refineMid
			if e := m.RefineUpperWithReceipt(qm, &q, &l, &u, &receipt); !errors.Is(e, ErrBinaryRefinePending) || receipt != before {
				t.Fatal("refused invocation replaced actual caller receipt", e)
			}
			stale := receipt
			stale.invocation++
			copied := *m
			copied.self = &copied
			serviceMust(t, m.CleanupRefineInvocation(&stale, true))
			serviceMust(t, copied.CleanupRefineInvocation(&receipt, true))
			if pending != m.refineMid || !m.RefineInvocationPending(&receipt) {
				t.Fatal("stale receipt or copied manager consumed actual midpoint")
			}
		}
		serviceMust(t, m.CleanupRefineInvocation(&receipt, false))
		serviceMust(t, m.CleanupRefineInvocation(&receipt, true))
		serviceMust(t, m.Del(&l))
		serviceMust(t, m.Del(&u))
		serviceMust(t, qm.Del(&q))
		m.Close()
	}
	if !found {
		t.Fatal("reservation sweep never stopped after midpoint allocation")
	}
}

func TestBinaryRefineMidOwningCleanupPhysicalRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, qm.Z.SetLimbs(&m.refineMid.Num, []uint32{7, 0, 1}))
	m.refinePending = true
	ref := m.refineMid.Num.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.CleanupRefineMid(); !errors.Is(e, arena.ErrBounds) || !m.refinePending || m.refineMid.Num.Cell != ref {
		t.Fatal("physical refusal lost midpoint", e, m.refineMid)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryRefinePending) {
		t.Fatal("retained midpoint allowed reentry", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupRefineMid())
	if m.refinePending || m.refineMid.Num.Cell != 0 {
		t.Fatal("owning retry retained midpoint")
	}
}

func TestBinaryRefineImportedOwningCleanupPhysicalRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, qm.Z.SetLimbs(&m.refineImported.Num, []uint32{7, 0, 1}))
	serviceMust(t, qm.Z.SetLimbs(&m.refineImported.Den, []uint32{9, 0, 1}))
	m.refineImportedPending = true
	den := m.refineImported.Den.Cell
	original := qm.Z.Cell(den)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(den, bad)
	if e := m.CleanupRefineImported(); !errors.Is(e, arena.ErrBounds) || !m.refineImportedPending || m.refineImportedCursor != 1 || m.refineImported.Num.Cell != 0 || m.refineImported.Den.Cell != den {
		t.Fatal("physical refusal lost imported denominator", e, m.refineImported)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryRefinePending) {
		t.Fatal("retained import allowed reentry", e)
	}
	qm.Z.Store(den, original)
	serviceMust(t, m.CleanupRefineImported())
	if m.refineImportedPending || m.refineImported.Num.Cell != 0 || m.refineImported.Den.Cell != 0 {
		t.Fatal("owning retry retained imported rational")
	}
}
