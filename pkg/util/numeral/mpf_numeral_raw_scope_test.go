package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatNumeralRawLocalSourceEndAndOwningRetry(t *testing.T) {
	for _, op := range []string{"inv", "power"} {
		t.Run(op, func(t *testing.T) {
			budget := arena.NewBudget(1 << 27)
			runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32})
			serviceMust(t, e)
			core, e := NewArbitraryFloatManager(IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, core.Close()); serviceMust(t, runtime.Close()) })
			m, e := NewArbitraryFloatNumeralManager(core, 11, 53)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.Close()) })
			serviceMust(t, core.integer.SetLimbs(&m.rawLocal.significand, []uint32{7, 0, 1}))
			m.rawPending = true
			ref := m.rawLocal.significand.Cell
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
			if e := m.finishRawLocalSource(); !errors.Is(e, arena.ErrBounds) || !m.rawPending || m.rawLocal.significand.Cell != ref {
				t.Fatal("source deletion lost raw float numeral", e)
			}
			var x, out ArbitraryFloat
			serviceMust(t, m.SetInt32(&x, 2))
			t.Cleanup(func() { serviceMust(t, core.Del(&x)); serviceMust(t, core.Del(&out)) })
			if op == "inv" {
				e = m.Inv(&x)
			} else {
				e = m.Power(&x, 3, &out)
			}
			if !errors.Is(e, ErrArbitraryFloatNumeralRawPending) {
				t.Fatal("pending raw local allowed another invocation", e)
			}
			if e := m.Close(); !errors.Is(e, arena.ErrBounds) || m.closed {
				t.Fatal("close discarded stopped raw local", e)
			}
			if e := m.CleanupRawLocal(); !errors.Is(e, arena.ErrBounds) || !m.rawPending || m.rawLocal.significand.Cell != ref {
				t.Fatal("owning refusal lost raw local", e)
			}
			core.integer.Store(ref, original)
			corrupted = false
			serviceMust(t, m.CleanupRawLocal())
			if m.rawPending || m.rawLocal.significand.Cell != 0 {
				t.Fatal("owning retry retained raw local")
			}
			if op == "inv" {
				serviceMust(t, m.Inv(&x))
				got, e := core.ToDouble(&x)
				if e != nil || got != 0.5 {
					t.Fatal("inverse after retry", got, e)
				}
			} else {
				serviceMust(t, m.Power(&x, 3, &out))
				got, e := core.ToDouble(&out)
				if e != nil || got != 8 {
					t.Fatal("power after retry", got, e)
				}
			}
		})
	}
}

func TestArbitraryFloatNumeralPowerStoppedBodyRetainsRawLocal(t *testing.T) {
	core, _, budget := mpfNumeralHelpersFixture(t)
	m, e := NewArbitraryFloatNumeralManager(core, 11, 120)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	var input, out ArbitraryFloat
	serviceMust(t, m.SetInt32(&input, 2))
	t.Cleanup(func() { serviceMust(t, core.Del(&input)); serviceMust(t, core.Del(&out)) })
	budget.FailAfter(0)
	e = m.Power(&input, 3, &out)
	if !numericPhysicalReservationRefusal(e) || !m.rawPending {
		t.Fatal("power copy did not retain stopped source local", e, m.rawPending)
	}
	budget.FailAfter(-1)
	serviceMust(t, m.CleanupRawLocal())
	if m.rawPending || m.rawLocal.significand.Cell != 0 {
		t.Fatal("stopped body cleanup retained raw local")
	}
}

func TestArbitraryFloatNumeralPowerMaskWrapRetainsPublishedResult(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, e := NewArbitraryFloatNumeralManager(core, 11, 53)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	var input, out ArbitraryFloat
	serviceMust(t, m.SetInt32(&input, 1))
	t.Cleanup(func() { serviceMust(t, core.Del(&input)); serviceMust(t, core.Del(&out)) })
	exponent := uint32(1 << 31)
	e = m.Power(&input, exponent, &out)
	var domain *PowerDomainError
	if !errors.As(e, &domain) || domain.Exponent != exponent || domain.Reason != PowerMaskOverflow || !m.rawPending {
		t.Fatal("mask wrap did not retain source prefix", e, m.rawPending)
	}
	got, e := core.ToDouble(&out)
	if e != nil || got != 1 {
		t.Fatal("mask wrap lost published result", got, e)
	}
	if e := m.Power(&input, 3, &out); !errors.Is(e, ErrArbitraryFloatNumeralRawPending) {
		t.Fatal("mask wrap allowed reentry", e)
	}
	serviceMust(t, m.CleanupRawLocal())
	serviceMust(t, m.Power(&input, 3, &out))
}
