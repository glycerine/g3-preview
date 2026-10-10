// Source: src/util/mpf.cpp:1590-1606; actual integer-display transport.
package util

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestArbitraryFloatRawStringNormalClassificationTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), significand: NewInt(1)}
	var out utext.OwnedString
	armed, calls := false, 0
	want := "[+ " + new(big.Int).Lsh(big.NewInt(1), 8192).String() + " 0"
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			s := &m.rawString
			if !s.executing || s.out != &out || s.numberStreamLive || !out.Matches(want) {
				t.Fatal("transport must follow number and exponent output", s.executing, s.numberStreamLive, out.Matches(want))
			}
			before, used := x, b.Used()
			var later utext.OwnedString
			for i, e := range []error{m.ToStringRaw(&x, &later), m.CleanupRawString(), m.Close(), m.SourceFinalize(), m.CleanupValue(&x), m.integer.SetInt32(&x.significand, 7), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("actual display callback guard", i, e)
				}
			}
			if x != before || b.Used() != used || !out.Matches(want) {
				t.Fatal("transport changed stopped raw string")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, m.integer.Mul2kInPlace(&x.significand, 8192))
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.ToStringRaw(&x, &out)
	armed = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 || !m.rawString.active {
		t.Fatal("actual integer display transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &x.significand)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate raw display graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, m.CleanupRawString())
	serviceMust(t, m.CleanupValue(&x))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("raw display refunds", b.Used())
	}
}
