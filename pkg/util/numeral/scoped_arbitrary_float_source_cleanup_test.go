package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestScopedArbitraryFloatSourceDeletionAndOwningRetry(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32})
	serviceMust(t, e)
	core, e := NewArbitraryFloatManager(IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, core.Close()); serviceMust(t, runtime.Close()) })
	m, e := NewArbitraryFloatNumeralManager(core, 11, 53)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	s, e := NewScopedArbitraryFloatNumeral(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	value, e := s.Get()
	serviceMust(t, e)
	serviceMust(t, core.integer.SetLimbs(&value.significand, []uint32{7, 0, 1}))
	ref := value.significand.Cell
	original := core.integer.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	core.integer.Store(ref, bad)
	corrupted := true
	t.Cleanup(func() {
		if corrupted {
			core.integer.Store(ref, original)
		}
	})
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.significand.Cell != ref {
		t.Fatal("source destructor lost scoped significand", e)
	}
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.significand.Cell != ref {
		t.Fatal("stopped source destructor replayed", e)
	}
	if e := s.Cleanup(); !errors.Is(e, arena.ErrBounds) || s.closed || value.significand.Cell != ref {
		t.Fatal("owning refusal lost scoped significand", e)
	}
	core.integer.Store(ref, original)
	corrupted = false
	serviceMust(t, s.Cleanup())
	if !s.closed || value.significand.Cell != 0 {
		t.Fatal("owning retry retained scoped value")
	}
	serviceMust(t, s.Close())
}
