// Source: src/util/mpf.cpp:194-231; actual allocation callback during the
// formatted text's rational/exponent/conversion chain.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedLiteralAllocationCallbackGuards(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	var m *ArbitraryFloatManager
	var out ArbitraryFloat
	armed := false
	calls := 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			before, used := out, b.Used()
			invocation := m.formattedLiteral.invocation
			for index, call := range []func() error{m.SourceFinalize, m.Close, m.CleanupFormattedLiteral, func() error { return m.SetFormattedLiteral(&out, 8, 24, ArbitraryFloatNearestEven, "1p0") }} {
				if got := call(); got != ErrNumericPending {
					t.Fatalf("callback owner refusal %d: %v", index, got)
				}
			}
			if out != before || b.Used() != used || m.formattedLiteral.invocation != invocation {
				t.Fatal("callback changed formatted caller")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := m.SetFormattedLiteral(&out, 8, 4000, ArbitraryFloatNearestEven, "-123456789.25p17")
	armed = false
	if body == nil || calls == 0 {
		t.Fatal("no reached formatted allocation callback", body, calls)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.CleanupFormattedLiteral())
	serviceMust(t, m.CleanupValue(&out))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal("formatted source end", e)
	}
	serviceMust(t, m.Close())
	serviceMust(t, rt.Deallocate(&pad))
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("formatted callback owner refund", b.Used())
	}
}
