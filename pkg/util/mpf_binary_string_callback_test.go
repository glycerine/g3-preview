// Source: src/util/mpf.cpp:1635-1693; actual numeric allocation transport.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestArbitraryFloatBinaryStringActualCopyTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 40000, false), significand: NewInt(1)}
	var out OwnedString
	armed, calls := false, 0
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if armed {
			calls++
			s := &m.binaryString
			if !s.executing || !s.tmpLive || !s.bitsLive || s.out != &out || !out.Matches("#b0 #b01111111 ") {
				t.Fatal("transport must reach significand COPY", s.executing, s.tmpLive, s.bitsLive, out.Length)
			}
			before, used, inv := x, b.Used(), s.invocation
			var later OwnedString
			for i, e := range []error{m.ToStringBinary(&x, 0, 0, &later), m.CleanupBinaryString(), m.Close(), m.SourceFinalize(), m.CleanupValue(&x), m.integer.SetInt32(&x.significand, 7), m.integer.Del(&s.tmp), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("binary callback guard", i, e)
				}
			}
			if x != before || b.Used() != used || s.invocation != inv || !out.Matches("#b0 #b01111111 ") {
				t.Fatal("transport changed source prefix")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, m.integer.Mul2kInPlace(&x.significand, 39998))
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsZero(&x)
	serviceMust(t, e)
	_, e = m.BiasExp(8, 0)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.ToStringBinary(&x, 0, 0, &out)
	armed = false
	if !errors.Is(body, Failure{Kind: FailureMemout}) || calls != 1 || !m.binaryString.active {
		t.Fatal("actual binary COPY transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &x.significand, &m.binaryString.tmp)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("binary callback graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, m.CleanupBinaryString())
	serviceMust(t, m.CleanupValue(&x))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("binary callback refunds", b.Used())
	}
}
