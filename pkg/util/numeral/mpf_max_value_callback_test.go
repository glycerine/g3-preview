package numeral

// Source: src/util/mpf.cpp:1898-1904; actual returning-failure transport.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatMkMaxValueCopyTransportPublication(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 7, significand: NewInt(9)}
	var receipt ArbitraryFloatValueCopyReceipt
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			if x.EBits() != 17 || x.SBits() != 40000 || !x.Sign() || x.exponent != 65535 || !m.valueExecuting || receipt.frame == nil {
				t.Fatal("COPY must follow scalar publication", x, receipt)
			}
			before, used := x, b.Used()
			for i, e := range []error{m.MkMaxValue(8, 24, false, &x, nil), m.Close(), m.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize(), m.CleanupValueCopy(&receipt, false), m.CleanupValueCopy(&receipt, true), m.CleanupValue(&x)} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("executing COPY guard", i, e)
				}
			}
			if x != before || b.Used() != used {
				t.Fatal("transport changed published prefix")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	_, e = m.topExponent(17)
	serviceMust(t, e)
	_, e = m.powerTwoMinusOneWrapper(39999, false)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.MkMaxValue(17, 40000, true, &x, &receipt)
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 || receipt.frame == nil {
		t.Fatal("actual COPY transport missing", body, calls, receipt)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &x.significand)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate constructor graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, m.CleanupValueCopy(&receipt, false))
	serviceMust(t, m.CleanupValue(&x))
	serviceMust(t, m.CleanupValueCopy(&receipt, true))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("constructor refunds", b.Used())
	}
}
