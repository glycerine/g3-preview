// Source: src/util/vector.h:45–201; actual nested output ownership and lifetime.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func satLiteralRowsTestInput(t *testing.T, r *util.Runtime) *RuntimeUintColumn {
	t.Helper()
	c := &RuntimeUintColumn{}
	c.Bind(r)
	t.Cleanup(func() { satOK(t, c.Finalize()) })
	return c
}

func TestSATConsequenceRowsCapturedInnerPayloadSurvivesOuterMoveAndDiesAtClose(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{0, 2}))
	oldOuter := out.storageState().Raw
	row, e := out.captureRow(0)
	satOK(t, e)
	satOK(t, out.appendLiterals([]Literal{4}))
	satOK(t, out.appendLiterals([]Literal{6}))
	if out.storageState().Raw == oldOuter {
		t.Fatal("outer did not relocate")
	}
	if _, e = s.runtime.memory.Read(oldOuter, 8); e == nil {
		t.Fatal("captured outer still alive")
	}
	for i, want := range []Literal{0, 2} {
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

func TestSATSolverLiteralRowsCloseDestroysSourceOuterBeforeFailedCopyAbort(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	src := satLiteralRowsTestInput(t, s.runtime.memory)
	satOK(t, src.Push(6))
	satOK(t, s.runtime.memory.WriteUint32(src.Raw.Block, 4, 2))
	e = out.appendColumn(src)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	outer := out.storageState().Raw
	inner := out.storageState().Rows[0].Block
	if inner == (util.Block{}) {
		t.Fatal("lost actual partial constructor")
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 0 {
		t.Fatal("failed constructor published size", n)
	}
	satOK(t, out.CloseChecked())
	blocks := testFieldBySourceName(reflect.ValueOf(s.runtime.memory).Elem(), "blocks")
	outerSerial := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(outer.Slot()-1)), "release"), "serial").Uint()
	innerSerial := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(inner.Slot()-1)), "release"), "serial").Uint()
	if outerSerial == 0 || innerSerial <= outerSerial {
		t.Fatal("source outer must be released before aborted inner", outerSerial, innerSerial)
	}
}

func TestSATSolverLiteralRowsSourceFinalizeThenReuseOwnsNewSourceVector(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendUnsigned([]uint32{2}))
	satOK(t, out.SourceFinalize())
	satOK(t, out.appendUnsigned([]uint32{4}))
	outer := out.storageState().Raw
	inner := out.storageState().Rows[0].Block
	satFullSolverRuntimeAudit(t, s, out)
	satOK(t, out.SourceFinalize())
	if _, e = s.runtime.memory.Read(outer, 0); e == nil {
		t.Fatal("second source finalize retained new outer")
	}
	satOK(t, out.CloseChecked())
	blocks := testFieldBySourceName(reflect.ValueOf(s.runtime.memory).Elem(), "blocks")
	a := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(inner.Slot()-1)), "release"), "serial").Uint()
	b := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(outer.Slot()-1)), "release"), "serial").Uint()
	if a == 0 || b <= a {
		t.Fatal("constructed new row destructor must precede its source outer", a, b)
	}
}

func TestSATSolverLiteralRowsStoppedSourceFinalizePreservesCauseAndNamedAbort(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	satOK(t, out.appendUnsigned([]uint32{2}))
	satOK(t, s.runtime.memory.WriteUint32(out.storageState().Raw, 8, 0))
	e = out.SourceFinalize()
	if !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if er := out.appendUnsigned([]uint32{4}); !errors.Is(er, arena.ErrClosed) {
		t.Fatal("stopped source destructor permits mutation", er)
	}
	if er := out.CloseChecked(); !errors.Is(er, e) {
		t.Fatal("lost original source guard", er, e)
	}
	satOK(t, out.CleanupStopped())
	satFullSolverRuntimeAudit(t, s, out)
}

func TestSATSolverLiteralRowsExpansionReadsActualOldPointer(t *testing.T) {
	for _, tc := range []struct {
		index uint32
		word  uint64
	}{{0, 0}, {1, 0}, {1, 4}} {
		t.Run(fmt.Sprintf("%d-%d", tc.index, tc.word), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
			satOK(t, e)
			satOK(t, out.appendUnsigned([]uint32{2}))
			satOK(t, out.appendUnsigned([]uint32{4}))
			old := out.storageState().Raw
			original := [2]util.Block{out.storageState().Rows[0].Block, out.storageState().Rows[1].Block}
			satOK(t, s.runtime.memory.WriteUint32(old, 8+uint64(tc.index)*8+tc.word, 0))
			e = out.appendUnsigned([]uint32{6})
			if !errors.Is(e, ErrSATOwner) {
				t.Errorf("source move ignored actual old pointer: %v", e)
			}
			state := out.storageState()
			if state.OldRaw != old || state.Raw == old || len(state.OldRows) != 2 {
				t.Errorf("move consumed old source ownership before pointer read")
			} else {
				for i := uint32(0); i < 2; i++ {
					if i < tc.index {
						if state.Rows[i].Block != original[i] || state.OldRows[i].Block != (util.Block{}) {
							t.Error("lost actual moved prefix", i)
						}
					} else if state.OldRows[i].Block != original[i] || state.Rows[i].Block != (util.Block{}) {
						t.Error("consumed unmoved source suffix", i)
					}
				}
			}
			if er := out.CloseChecked(); er != nil {
				satOK(t, out.CleanupStopped())
			}
			satFullSolverRuntimeAudit(t, s, out)
		})
	}
}
