package numeral

// Source: src/util/mpf.cpp:1828-1848; actual cached-numeral ownership.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatPublicExponentBoundsEveryColdReservation(t *testing.T) {
	for kind, get := range []func(*ArbitraryFloatManager, uint32) (int64, error){(*ArbitraryFloatManager).MkTopExp, (*ArbitraryFloatManager).MkBotExp, (*ArbitraryFloatManager).MkMinExp, (*ArbitraryFloatManager).MkMaxExp} {
		for _, eb := range []uint32{2, 32, 63} {
			complete := false
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				m, r, b := mpfNumeralHelpersFixture(t)
				b.FailAfter(ordinal)
				_, body := get(m, eb)
				integerPhysicalAudit(t, m.integer)
				if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
					t.Fatal("immediate bound graph", kind, eb, ordinal, body, b.Used())
				}
				if body != nil && !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected bound refusal", kind, eb, ordinal, body)
				}
				b.FailAfter(-1)
				serviceMust(t, m.Close())
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("bound refunds", kind, eb, ordinal, b.Used())
				}
				if body == nil {
					complete = true
					break
				}
			}
			if !complete {
				t.Fatal("bound reservation horizon", kind, eb)
			}
		}
	}
}

func TestArbitraryFloatPublicExponentBoundsCacheTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			used := b.Used()
			if !m.valueExecuting || m.PowerWrappers == nil || !m.PowerWrappers.executing {
				t.Fatal("transport must reach actual cache acquisition")
			}
			for _, get := range []func(uint32) (int64, error){m.MkTopExp, m.MkBotExp, m.MkMinExp, m.MkMaxExp} {
				if _, e := get(8); !errors.Is(e, ErrNumericPending) {
					t.Fatal("executing cache bound guard", e)
				}
			}
			for _, e := range []error{m.Close(), m.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("executing cache owner guard", e)
				}
			}
			if b.Used() != used {
				t.Fatal("transport changed cache graph")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	_, body := m.MkTopExp(63)
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 {
		t.Fatal("actual cache transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad})
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate cache graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<27))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("cache refunds", b.Used())
	}
}
