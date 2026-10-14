package realclosure

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source: src/math/realclosure/realclosure.cpp:406-451 and
// src/util/ref_buffer.h:38-49. An invalid physical size refuses the actual
// final coefficient deletion; no mock release callback is used.
func TestRealClosureSturmSequenceRefusedDestructorRetainsRows(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
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
	store := &realValueStore{numbers: service.Manager}
	first, err := store.newRational(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	last, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.numbers.SetString(&last.rational, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	cellRef := last.rational.Num.Cell
	if cellRef == 0 {
		t.Fatal("test value did not allocate a physical numeric cell")
	}
	rows := make([]realPolynomial, 2)
	if err := store.setPolynomial(&rows[0], []*realValue{first}); err != nil {
		t.Fatal(err)
	}
	if err := store.setPolynomial(&rows[1], []*realValue{last}); err != nil {
		t.Fatal(err)
	}
	if err := store.release(first); err != nil {
		t.Fatal(err)
	}
	if err := store.release(last); err != nil {
		t.Fatal(err)
	}
	z := service.Manager.Z
	original := z.Cell(cellRef)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(cellRef, bad)
	if err := store.closePolynomialSequence(rows); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("actual destructor refusal", err)
	}
	if rows[0].coefficients != nil || !first.deleted || len(rows[1].coefficients) != 1 || last.deleted || len(store.pendingSequence) != 2 {
		t.Fatal("failed sequence cleanup lost the completed prefix or pending suffix")
	}
	for _, query := range []func() (int, error){
		func() (int, error) { return store.rootCountUnboundedRational(nil, false) },
		func() (int, error) { return store.tarskiUnboundedRational(nil, nil, false) },
		func() (int, error) { return store.rootCountBinaryIntervalRational(nil, nil, false) },
		func() (int, error) { return store.tarskiBinaryIntervalRational(nil, nil, nil, false) },
	} {
		if _, err := query(); !errors.Is(err, errRealClosureSequencePending) {
			t.Fatal("pending owner allowed another query", err)
		}
	}
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) || len(store.pendingSequence) != 2 {
		t.Fatal("refused retry lost the pending sequence", err)
	}
	z.Store(cellRef, original)
	if err := store.cleanupPendingPolynomialSequence(); err != nil {
		t.Fatal(err)
	}
	if !last.deleted || store.pendingSequence != nil || rows[1].coefficients != nil {
		t.Fatal("recovered cleanup did not retire the last coefficient")
	}
	if err := store.sequenceReady(); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("physical cleanup retained bytes", used)
	}
}
