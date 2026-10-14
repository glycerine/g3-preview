package interval

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryRootScratchStoppedSourceSuffixAndOwningRetry(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	core, e := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := core.Close(); e != nil {
			t.Error(e)
		}
		if e := runtime.Close(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("arbitrary root scratch retained budget", budget.Used())
		}
	})
	m, e := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	s := arbitraryRootScratch{m: m}
	t.Cleanup(func() {
		for i := len(s.owners) - 1; i >= 0; i-- {
			if e := s.owners[i].Cleanup(); e != nil {
				t.Error(e)
			}
		}
	})
	z, e := core.BorrowIntegerManager()
	if e != nil {
		t.Fatal(e)
	}
	var refs [3]numeral.NumericCellRef
	for i := range refs {
		value, e := s.new()
		if e != nil {
			t.Fatal(e)
		}
		sig, e := core.BorrowSignificand(value)
		if e != nil {
			t.Fatal(e)
		}
		if e := z.SetLimbs(sig, []uint32{uint32(i + 7), 0, 1}); e != nil {
			t.Fatal(e)
		}
		refs[i] = sig.Cell
	}
	original := z.Cell(refs[1])
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(refs[1], bad)
	corrupted := true
	t.Cleanup(func() {
		if corrupted {
			z.Store(refs[1], original)
		}
	})
	e = s.close()
	var pending *ArbitraryRootCleanupError
	if !errors.As(e, &pending) || !errors.Is(e, arena.ErrBounds) || s.cursor != 1 || s.closed || s.owners[1] == nil {
		t.Fatal("source stopped suffix", e, s.cursor)
	}
	if _, e := s.owners[2].Get(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("completed last destructor replayed", e)
	}
	if _, e := s.owners[0].Get(); e != nil {
		t.Fatal("older scope lost before stopped destructor", e)
	}
	if e := s.close(); !errors.Is(e, arena.ErrBounds) || s.cursor != 1 {
		t.Fatal("source destructor replayed", e, s.cursor)
	}
	if e := pending.Cleanup(); !errors.Is(e, arena.ErrBounds) || s.cursor != 1 {
		t.Fatal("owning refusal lost suffix", e, s.cursor)
	}
	z.Store(refs[1], original)
	corrupted = false
	if e := pending.Cleanup(); e != nil || !s.closed || s.owners != nil {
		t.Fatal("owning retry did not drain source suffix", e)
	}
	if e := pending.Cleanup(); e != nil {
		t.Fatal("completed cleanup replayed", e)
	}
}
