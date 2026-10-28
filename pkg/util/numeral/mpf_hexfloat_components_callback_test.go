package numeral

// Source: src/util/mpf.cpp:1608-1619; actual first COPY allocation transport.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestArbitraryFloatHexfloatComponentsActualCopyTransport(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	sig := NewInt(0)
	var text *ArbitraryFloatFormatText
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			s := &m.hexfloatComponents
			if !s.executing || !s.qLive || !s.sigLive || s.q.format != 0 || s.format != nil || text.Numeric.invocation != 0 {
				t.Fatal("initial COPY source prefix")
			}
			q, n, input, used, inv := s.q, s.sig, sig, budget.Used(), s.invocation
			_, nested := m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 40000, 0, text)
			for i, e := range []error{nested, m.CleanupHexfloatComponents(), m.CleanupValue(&s.q), m.Close(), m.SourceFinalize(), m.integer.SetInt32(&sig, 7), m.integer.Del(&s.sig), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("component copy callback guard", i, e)
				}
			}
			if s.q != q || s.sig != n || sig != input || s.invocation != inv || budget.Used() != used || text.Result.Length != 0 {
				t.Fatal("callback changed source prefix")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	digits := make([]uint32, 1250)
	digits[len(digits)-1] = 1
	serviceMust(t, m.integer.SetLimbs(&sig, digits))
	text, e = NewArbitraryFloatFormatText(budget)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	_, body := m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 40000, 0, text)
	armed = false
	s := &m.hexfloatComponents
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 || !s.active || s.format != nil {
		t.Fatal("actual component COPY transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &sig, &s.sig, &s.q.significand)
	if budget.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+text.StorageBytes() {
		t.Fatal("component callback graph", budget.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupHexfloatComponents())
	serviceMust(t, text.Close())
	serviceMust(t, m.integer.CleanupRawNumeral(&sig))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if budget.Used() != 0 {
		t.Fatal("component callback refunds", budget.Used())
	}
}
