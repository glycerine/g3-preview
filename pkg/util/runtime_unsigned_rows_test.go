// Source: src/util/vector.h:45–201; actual nested output ownership and lifetime.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRuntimeUnsignedRowsCloseDestroysSourceOuterBeforeFailedCopyAbort(t *testing.T) {
	m, budget := memoryRuntimeFixture(t, false, 64)
	out, e := NewRuntimeUnsignedRows(m, budget)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
	src := unsignedRowsTestInput(t, m)
	memoryRuntimeRequire(t, src.Push(6))
	memoryRuntimeRequire(t, m.WriteUint32(src.Raw.Block, 4, 2)) // second byte-word not initialized
	e = out.AppendColumn(src)
	if !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	outer := out.StorageState().Raw
	inner := out.StorageState().Rows[0].Block
	if inner == (Block{}) {
		t.Fatal("lost actual partial constructor")
	}
	n, e := out.RowCount()
	memoryRuntimeRequire(t, e)
	if n != 0 {
		t.Fatal("failed ctor published size", n)
	}
	unsignedRowsTestAudit(t, m, out, src)
	memoryRuntimeRequire(t, out.CloseChecked())
	blocks := testFieldBySourceName(reflect.ValueOf(m).Elem(), "blocks")
	outerSerial := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(outer.Slot()-1)), "release"), "serial").Uint()
	innerSerial := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(inner.Slot()-1)), "release"), "serial").Uint()
	if outerSerial == 0 || innerSerial <= outerSerial {
		t.Fatal("source outer must be released before aborted inner", outerSerial, innerSerial)
	}
}

func TestRuntimeUnsignedRowsSourceFinalizeThenReuseOwnsNewSourceVector(t *testing.T) {
	m, budget := memoryRuntimeFixture(t, false, 64)
	out, e := NewRuntimeUnsignedRows(m, budget)
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { memoryRuntimeRequire(t, out.CloseChecked()) })
	memoryRuntimeRequire(t, unsignedRowsTestAppend(out, []uint32{2}))
	memoryRuntimeRequire(t, out.SourceFinalize())
	memoryRuntimeRequire(t, unsignedRowsTestAppend(out, []uint32{4}))
	outer := out.StorageState().Raw
	inner := out.StorageState().Rows[0].Block
	unsignedRowsTestAudit(t, m, out)
	memoryRuntimeRequire(t, out.SourceFinalize())
	if _, e = m.Read(outer, 0); e == nil {
		t.Fatal("second source finalize retained new outer")
	}
	memoryRuntimeRequire(t, out.CloseChecked())
	blocks := testFieldBySourceName(reflect.ValueOf(m).Elem(), "blocks")
	a := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(inner.Slot()-1)), "release"), "serial").Uint()
	b := testFieldBySourceName(testFieldBySourceName(blocks.Index(int(outer.Slot()-1)), "release"), "serial").Uint()
	if a == 0 || b <= a {
		t.Fatal("constructed new row destructor must precede its source outer", a, b)
	}
}

func TestRuntimeUnsignedRowsStoppedSourceFinalizePreservesCauseAndNamedAbort(t *testing.T) {
	m, budget := memoryRuntimeFixture(t, false, 64)
	out, e := NewRuntimeUnsignedRows(m, budget)
	memoryRuntimeRequire(t, e)
	memoryRuntimeRequire(t, unsignedRowsTestAppend(out, []uint32{2}))
	memoryRuntimeRequire(t, m.WriteUint32(out.StorageState().Raw, 8, 0)) // explicit source-pointer-domain corruption
	e = out.SourceFinalize()
	if !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	unsignedRowsTestAudit(t, m, out)
	if er := unsignedRowsTestAppend(out, []uint32{4}); !errors.Is(er, arena.ErrClosed) {
		t.Fatal("stopped source destructor permits mutation", er)
	}
	if er := out.CloseChecked(); !errors.Is(er, e) {
		t.Fatal("lost original source guard", er, e)
	}
	memoryRuntimeRequire(t, out.CleanupStopped())
	unsignedRowsTestAudit(t, m, out)
}
