// Source: src/util/vector.h:45–201; actual nested output ownership and lifetime.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestSATConsequenceRowsCapturedInnerPayloadSurvivesOuterMoveAndDiesAtClose(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]satLiteral{0, 2}))
	oldOuter := out.raw
	row, e := out.captureRow(0)
	satOK(t, e)
	satOK(t, out.appendLiterals([]satLiteral{4}))
	satOK(t, out.appendLiterals([]satLiteral{6}))
	if out.raw == oldOuter {
		t.Fatal("outer did not relocate")
	}
	if _, e = s.runtime.memory.Read(oldOuter, 8); e == nil {
		t.Fatal("captured outer still alive")
	}
	for i, want := range []satLiteral{0, 2} {
		got, e := row.read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.CloseChecked())
	if _, e = row.read(0); e == nil {
		t.Fatal("captured inner still alive after actual owner close")
	}
}

// An uninitialized source payload is a checked undefined-domain fixture. The
// failed inner copy constructor has no source element destructor; source outer
// destruction precedes separately declared Go owning abort of that partial copy.
func TestSATConsequenceRowsCloseDestroysSourceOuterBeforeFailedCopyAbort(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	f, e := s.consequenceFrame()
	satOK(t, e)
	satOK(t, f.vector(s, 0, nil))
	t.Cleanup(func() { satOK(t, s.consequenceDrop(f, nil)) })
	src := &f.vectors[0]
	satOK(t, src.push(6))
	satOK(t, s.runtime.memory.WriteUint32(src.raw.block, 4, 2)) // second byte-word not initialized
	e = out.appendColumn(src)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	outer := out.raw
	inner := out.rows[0].raw.block
	if inner == (memory.Block{}) {
		t.Fatal("lost actual partial constructor")
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 0 {
		t.Fatal("failed ctor published size", n)
	}
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.CloseChecked())
	blocks := reflect.ValueOf(s.runtime.memory).Elem().FieldByName("blocks")
	outerSerial := blocks.Index(int(outer.Slot() - 1)).FieldByName("release").FieldByName("serial").Uint()
	innerSerial := blocks.Index(int(inner.Slot() - 1)).FieldByName("release").FieldByName("serial").Uint()
	if outerSerial == 0 || innerSerial <= outerSerial {
		t.Fatal("source outer must be released before aborted inner", outerSerial, innerSerial)
	}
}

func TestSATConsequenceRowsSourceFinalizeThenReuseOwnsNewSourceVector(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]satLiteral{2}))
	satOK(t, out.SourceFinalize())
	satOK(t, out.appendLiterals([]satLiteral{4}))
	outer := out.raw
	inner := out.rows[0].raw.block
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.SourceFinalize())
	if _, e = s.runtime.memory.Read(outer, 0); e == nil {
		t.Fatal("second source finalize retained new outer")
	}
	satOK(t, out.CloseChecked())
	blocks := reflect.ValueOf(s.runtime.memory).Elem().FieldByName("blocks")
	a := blocks.Index(int(inner.Slot() - 1)).FieldByName("release").FieldByName("serial").Uint()
	b := blocks.Index(int(outer.Slot() - 1)).FieldByName("release").FieldByName("serial").Uint()
	if a == 0 || b <= a {
		t.Fatal("constructed new row destructor must precede its source outer", a, b)
	}
}

func TestSATConsequenceRowsStoppedSourceFinalizePreservesCauseAndNamedAbort(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	satOK(t, out.appendLiterals([]satLiteral{2}))
	satOK(t, s.runtime.memory.WriteUint32(out.raw, 8, 0)) // explicit source-pointer-domain corruption
	e = out.SourceFinalize()
	if !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satFullSolverRuntimeAudit(t, s, out)
	if er := out.appendLiterals([]satLiteral{4}); !errors.Is(er, arena.ErrClosed) {
		t.Fatal("stopped source destructor permits mutation", er)
	}
	if er := out.CloseChecked(); !errors.Is(er, e) {
		t.Fatal("lost original source guard", er, e)
	}
	satOK(t, out.CleanupStopped())
	satFullSolverRuntimeAudit(t, s, out)
}
