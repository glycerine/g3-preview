// Source: src/sat/sat_solver.cpp:3119-3150,3252-3636; src/util/vector.h:389-435.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func satLiteralRowsTestInput(t *testing.T, r *util.Runtime) *satUintColumn {
	t.Helper()
	c := &satUintColumn{}
	c.Bind(r)
	t.Cleanup(func() { satOK(t, c.Finalize()) })
	return c
}
func TestSATConsequenceRowsNativeCopiesAndCapturedLifetime(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	input := []Literal{0, 2}
	satOK(t, out.appendLiterals(input))
	input[0] = 9
	row, e := out.captureRow(0)
	satOK(t, e)
	before, physical := s.budget.Used(), s.runtime.memory.Stats()
	s.budget.FailAfter(0)
	for i := uint32(0); i < 257; i++ {
		satOK(t, out.appendUnsigned([]uint32{2*i + 4}))
	}
	s.budget.FailAfter(-1)
	if s.budget.Used() != before || s.runtime.memory.Stats() != physical {
		t.Fatal("Go output backing charged Runtime")
	}
	for i, want := range []Literal{0, 2} {
		got, e := row.read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal("caller mutation or outer growth changed result", got, want)
		}
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 258 {
		t.Fatal(n)
	}
	for i := uint32(1); i < n; i++ {
		got, e := out.captureRow(i)
		satOK(t, e)
		literal, e := got.read(0)
		satOK(t, e)
		if got.count != 1 || literal != Literal(2*(i-1)+4) {
			t.Fatal("result order", i, literal)
		}
	}
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.CloseChecked())
	if _, e := row.read(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed borrowed row", e)
	}
}
func TestSATSolverLiteralRowsFailedInputPreservesPublishedRows(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{2}))
	src := satLiteralRowsTestInput(t, s.runtime.memory)
	satOK(t, src.Push(6))
	satOK(t, src.Finalize())
	if e := out.appendColumn(src); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 1 {
		t.Fatal("failed input published a row", n)
	}
	// A fresh native input publishes an independent copy.
	src.Bind(s.runtime.memory)
	satOK(t, src.Push(6))
	satOK(t, out.appendColumn(src))
	satOK(t, src.Write(0, 8))
	satOK(t, src.Finalize())
	row, e := out.captureRow(1)
	satOK(t, e)
	got, e := row.read(0)
	satOK(t, e)
	if got != 6 {
		t.Fatal("result aliases native input", got)
	}
	satFullSolverRuntimeAudit(t, s, out)
}
func TestSATSolverLiteralRowsFinalizeInvalidatesViewsAndAllowsReuse(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	input := []uint32{2, 4}
	satOK(t, out.appendUnsigned(input))
	input[0] = 8
	old, e := out.captureRow(0)
	satOK(t, e)
	satOK(t, out.SourceFinalize())
	satOK(t, out.appendUnsigned([]uint32{6}))
	if _, e := old.read(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("old result view survived reuse", e)
	}
	current, e := out.captureRow(0)
	satOK(t, e)
	got, e := current.read(0)
	satOK(t, e)
	if got != 6 {
		t.Fatal(got)
	}
	satOK(t, out.SourceFinalize())
	n, e := out.rowCount()
	satOK(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	satOK(t, out.CloseChecked())
	if e := out.appendUnsigned(nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATSolverLiteralRowsOwnerAndInputGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	before := s.budget.Used()
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	copy := *out
	if copy.appendLiterals(nil) == nil || copy.CloseChecked() == nil {
		t.Fatal("copied owner accepted")
	}
	if e := out.appendColumn(nil); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	other, e := util.NewRuntime(util.RuntimeConfig{Budget: s.budget})
	satOK(t, e)
	if e := out.checkRuntime(other); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, other.Close())
	satOK(t, out.appendLiterals(nil))
	empty, e := out.captureRow(0)
	satOK(t, e)
	if empty.count != 0 {
		t.Fatal(empty.count)
	}
	if _, e := empty.read(0); e == nil {
		t.Fatal("read empty row")
	}
	if _, e := out.captureRow(1); e == nil {
		t.Fatal("read missing row")
	}
	satOK(t, out.CloseChecked())
	satOK(t, out.CloseChecked())
	if s.budget.Used() != before {
		t.Fatal("owner header retained", s.budget.Used(), before)
	}
}
