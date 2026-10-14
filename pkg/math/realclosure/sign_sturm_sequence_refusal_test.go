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

func TestRealClosureSturmTemporaryRefusalPrecedesOlderRows(t *testing.T) {
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
	temporary, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	older, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.numbers.SetString(&temporary.rational, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	if err := store.numbers.SetString(&older.rational, "18446744073709551619"); err != nil {
		t.Fatal(err)
	}
	rows := make([]realPolynomial, 1)
	if err := store.setPolynomial(&rows[0], []*realValue{older}); err != nil {
		t.Fatal(err)
	}
	if err := store.release(older); err != nil {
		t.Fatal(err)
	}
	z := service.Manager.Z
	tempRef, rowRef := temporary.rational.Num.Cell, older.rational.Num.Cell
	if tempRef == 0 || rowRef == 0 || tempRef == rowRef {
		t.Fatal("temporary and row need distinct real numeric cells")
	}
	tempCell, rowCell := z.Cell(tempRef), z.Cell(rowRef)
	badTemp, badRow := tempCell, rowCell
	badTemp.Backing.Requested = uint64(math.MaxUint32) + 1
	badRow.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(tempRef, badTemp)
	z.Store(rowRef, badRow)
	if err := store.releaseSturmCoefficients([]*realValue{first, temporary}); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("temporary destructor refusal", err)
	}
	if !first.deleted || temporary.deleted || len(store.pendingSturmCoefficients) != 1 || len(store.pendingSturmCoefficients[0]) != 1 {
		t.Fatal("temporary prefix or suffix ownership")
	}
	if _, err := store.rootCountUnboundedRational(nil, false); !errors.Is(err, errRealClosureSequencePending) {
		t.Fatal("pending temporary allowed a new query", err)
	}
	if err := store.closePolynomialSequence(rows); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("older row destructor refusal", err)
	}
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) || older.deleted {
		t.Fatal("younger refusal must precede older row cleanup", err)
	}
	z.Store(tempRef, tempCell)
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) || !temporary.deleted || older.deleted || len(store.pendingSturmCoefficients) != 0 {
		t.Fatal("completed temporary must precede refused row", err)
	}
	z.Store(rowRef, rowCell)
	if err := store.cleanupPendingPolynomialSequence(); err != nil {
		t.Fatal(err)
	}
	if !older.deleted || rows[0].coefficients != nil || store.pendingSequence != nil || store.pendingSturmCoefficients != nil {
		t.Fatal("both lexical owners were not drained")
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

func TestRealClosureSturmScratchRefusalRetainsValue(t *testing.T) {
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
	scratch, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.numbers.SetString(&scratch.rational, "18446744073709551617"); err != nil {
		t.Fatal(err)
	}
	z := service.Manager.Z
	ref := scratch.rational.Num.Cell
	if ref == 0 {
		t.Fatal("scratch has no physical numeric cell")
	}
	original := z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, bad)
	if err := store.releaseSturmScratch(scratch); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("scratch destructor refusal", err)
	}
	if scratch.deleted || len(store.pendingSturmCoefficients) != 1 || len(store.pendingSturmCoefficients[0]) != 1 || store.pendingSturmCoefficients[0][0] != scratch {
		t.Fatal("scratch owner did not retain the refused value")
	}
	if err := store.sequenceReady(); !errors.Is(err, errRealClosureSequencePending) {
		t.Fatal("pending scratch allowed a new sequence", err)
	}
	if err := store.cleanupPendingPolynomialSequence(); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("refused scratch retry", err)
	}
	z.Store(ref, original)
	if err := store.cleanupPendingPolynomialSequence(); err != nil {
		t.Fatal(err)
	}
	if !scratch.deleted || store.pendingSturmCoefficients != nil {
		t.Fatal("scratch did not drain")
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
