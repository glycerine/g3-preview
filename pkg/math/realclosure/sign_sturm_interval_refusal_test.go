package realclosure

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureSturmIntervalRefusedDestructorRetainsOwner(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32})
	if err != nil {
		t.Fatal(err)
	}
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 32},
	}})
	if err != nil {
		t.Fatal(err)
	}
	bounds, err := numeral.NewBinaryRationalManager(service.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: service})
	if err != nil {
		t.Fatal(err)
	}
	value := numeral.NewBinaryRational()
	if err := bounds.Z.SetString(&value.Num, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	interval, err := NewBinaryInterval(bounds)
	if err != nil {
		t.Fatal(err)
	}
	if err := interval.SetFinite(&value, &value, false, false); err != nil {
		t.Fatal(err)
	}
	cell := interval.lower.Num.Cell
	if cell == 0 {
		t.Fatal("interval did not own a physical numerator cell")
	}
	store := &realValueStore{numbers: service.Manager}
	z := service.Manager.Z
	original := z.Cell(cell)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(cell, bad)
	if err := store.releaseSturmInterval(interval); !errors.Is(err, arena.ErrBounds) || len(store.pendingSturmIntervals) != 1 {
		t.Fatal("refused interval release", err)
	}
	if err := store.sequenceReady(); !errors.Is(err, errRealClosureSequencePending) {
		t.Fatal("pending interval allowed another query", err)
	}
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) || len(store.pendingSturmIntervals) != 1 {
		t.Fatal("refused retry lost interval", err)
	}
	z.Store(cell, original)
	if err := store.cleanupPendingPolynomialSequence(); err != nil {
		t.Fatal(err)
	}
	if !interval.closed || len(store.pendingSturmIntervals) != 0 {
		t.Fatal("recovered cleanup retained interval")
	}
	valueCell := value.Num.Cell
	valueOriginal := z.Cell(valueCell)
	valueBad := valueOriginal
	valueBad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(valueCell, valueBad)
	if err := store.releaseSturmBinary(bounds, &value); !errors.Is(err, arena.ErrBounds) || len(store.pendingSturmBinaries) != 1 {
		t.Fatal("refused binary temporary release", err)
	}
	if err := store.sequenceReady(); !errors.Is(err, errRealClosureSequencePending) {
		t.Fatal("pending binary temporary allowed another query", err)
	}
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("refused binary retry", err)
	}
	z.Store(valueCell, valueOriginal)
	if err := store.cleanupPendingPolynomialSequence(); err != nil || len(store.pendingSturmBinaries) != 0 {
		t.Fatal("recovered binary cleanup", err)
	}
	if err := bounds.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("recovered interval retained bytes", used)
	}
}
