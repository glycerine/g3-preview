package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverAntecedentMapNativeColdConstruction(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, e := newSATSolverIndexMap(b)
	satOK(t, e)
	if len(m.entries) != 0 || b.Used() != m.ownerBytes {
		t.Fatal("native cold owner", b.Used(), m.ownerBytes)
	}
	copied := *m
	if e = copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("shallow owner copy accepted", e)
	}
	satOK(t, m.Close())
	satOK(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("native header leaked or double refunded", b.Used())
	}
	if _, e = m.contains(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed map accepted", e)
	}
}

func TestSATSolverAntecedentMapAllConstructorRefusalPrefixes(t *testing.T) {
	for fail := int64(0); fail < 2; fail++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(fail)
		m, e := newSATSolverIndexMap(b)
		if fail == 0 {
			if !errors.Is(e, arena.ErrInjectedFailure) || m != nil || b.Used() != 0 {
				t.Fatal("cold header refusal", m, e, b.Used())
			}
		} else {
			satOK(t, e)
			satOK(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("cold owner refund", b.Used())
			}
		}
	}
}

func TestSATSolverAntecedentMapNativeResetAndCloseInvalidateRows(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, e := newSATSolverIndexMap(b)
	satOK(t, e)
	source := satSolverIndexSet{}
	satOK(t, source.insert(17))
	satOK(t, m.insert(7, &source))
	row, e := m.find(7)
	satOK(t, e)
	it, e := row.capture()
	satOK(t, e)
	before := b.Used()
	b.FailAfter(0)
	satOK(t, m.reset())
	if len(m.entries) != 0 || b.Used() != before {
		t.Fatal("native reset changed cold charge")
	}
	if _, e = row.contains(17); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("reset retained borrowed row", e)
	}
	if _, e = it.next(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("reset retained borrowed iterator", e)
	}
	satOK(t, m.insert(7, &source))
	replacement, e := m.find(7)
	satOK(t, e)
	if replacement == row {
		t.Fatal("reset revived stale row identity")
	}
	satOK(t, m.Close())
	if _, e = replacement.contains(17); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("close retained borrowed row", e)
	}
	if b.Used() != 0 {
		t.Fatal("native close leaked header", b.Used())
	}
	yes, e := source.contains(17)
	satOK(t, e)
	if !yes {
		t.Fatal("map adopted caller's set")
	}
	satOK(t, source.close())
}
