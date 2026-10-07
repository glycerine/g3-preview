// Source: src/util/scoped_numeral_buffer.h; src/util/buffer.h.
// Licensed under the MIT license. Author: Leonardo de Moura.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedBufferReviewDestructiveCopyBeforeSourceDestructor(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	var b scopedIntegerModularBuffer128
	b.construct(m)
	var small, large Int
	defer func() {
		contextTLSFinish(t, r, b.cleanup)
		contextTLSFinish(t, r, func() error { return z.Del(&small) })
		contextTLSFinish(t, r, func() error { return z.Del(&large) })
		contextTLSFinish(t, r, m.Close)
		contextTLSFinish(t, r, z.CloseChecked)
		contextTLSFinish(t, r, r.Close)
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}()
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, z.SetLimbs(&small, digits))
	bigger := make([]uint32, 80000)
	bigger[79999] = 1
	serviceMust(t, z.SetLimbs(&large, bigger))
	serviceMust(t, b.push(&small))
	old, e := b.get(0)
	serviceMust(t, e)
	contextTLSNearMinimum(t, r)
	body := b.set(0, &large)
	if !errors.Is(body, arena.ErrBounds) || !ValidNumericRef(z.Cells, old.Cell) || !z.Cell(old.Cell).Released || !z.Cell(old.Cell).CleanupPending {
		t.Fatal("actual destructive old physical release stop", body)
	}
	scopedBufferAudit(t, &b, &small, &large)
	state := r.State()
	stats := r.Stats()
	source := b.sourceClose()
	if !errors.Is(source, arena.ErrInvalidSlot) || b.count != 1 || b.closed || r.State() != state || r.Stats() != stats {
		t.Fatal("literal source leaf acknowledged already-consumed old block", source, b.count, b.closed, r.State(), state)
	}
}

func TestScopedBufferReviewStoppedSourceBlocksMutation(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewIntegerModularManager(z)
	serviceMust(t, e)
	var b scopedIntegerModularBuffer128
	b.construct(m)
	var input Int
	limbs := make([]uint32, 40000)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, z.SetLimbs(&input, limbs))
	serviceMust(t, b.push(&input))
	serviceMust(t, b.push(&input))
	first, e := b.get(0)
	serviceMust(t, e)
	older, e := b.get(1)
	serviceMust(t, e)
	before := r.Stats()
	contextTLSNearMinimum(t, r)
	cause := b.sourceClose()
	if !errors.Is(cause, arena.ErrBounds) || b.count != 2 || !ValidNumericRef(z.Cells, first.Cell) || !z.Cell(first.Cell).Released {
		t.Fatal("actual first payload physically consumed before source stop", cause, b.count, r.State())
	}
	now, e := b.get(1)
	serviceMust(t, e)
	if now != older || !ValidNumericRef(z.Cells, older.Cell) || z.Cell(older.Cell).Released || r.Stats().LiveBlocks+1 != before.LiveBlocks {
		t.Fatal("older actual completed numeral changed", now, older, r.Stats(), before)
	}
	state := r.State()
	stats := r.Stats()
	if again := b.sourceClose(); again != cause || r.State() != state || r.Stats() != stats {
		t.Fatal("source stop replayed physical accounting", again, cause)
	}
	scopedBufferAudit(t, &b, &input)
	count := b.count
	zero := NewInt(0)
	for name, call := range map[string]func() error{
		"push":   func() error { return b.push(&zero) },
		"set":    func() error { return b.set(1, &zero) },
		"resize": func() error { return b.Resize(count + 1) },
	} {
		if e := call(); e != cause || b.count != count || r.State() != state || r.Stats() != stats {
			t.Errorf("stopped ordinary %s changed source context: %v count%d want%d", name, e, b.count, count)
		}
	}
	contextTLSFinish(t, r, b.cleanup)
	integerPhysicalAudit(t, z, &input)
	contextTLSFinish(t, r, func() error { return z.Del(&input) })
	contextTLSFinish(t, r, m.Close)
	contextTLSFinish(t, r, z.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if budget.Used() != 0 {
		t.Fatal("full physical refund", budget.Used())
	}
}

// Source shrink's size precondition is DEBUG-only. In release, the empty
// iteration above the current size preserves the source logical count.
func TestScopedBufferReviewReleaseShrinkAboveSize(t *testing.T) {
	_, b := scopedBufferFixture(t)
	serviceMust(t, b.Resize(3))
	e := b.shrink(4)
	if modularSourceDebug {
		if e != ErrNumericAssertion {
			t.Fatal("source DEBUG-only precondition", e)
		}
	} else if e != nil {
		t.Fatal("release source performs no indexed access or count change", e)
	}
	if b.count != 3 {
		t.Fatal("source count changed", b.count)
	}
}
