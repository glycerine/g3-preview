// Source: src/util/mpq.cpp:123-127; actual denominator-display transport.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalOwnedStringDenominatorTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	a := Rational{Num: NewInt(3), Den: NewInt(1)}
	var out OwnedString
	armed, calls := false, 0
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if armed {
			calls++
			s := &m.rational.stringScope
			if !s.executing || !s.denominatorStreamLive || !s.outLive || s.out != &out || !out.Matches("3/") {
				t.Fatal("transport must follow numerator/slash publication", s, out)
			}
			before, used := a, b.Used()
			var later OwnedString
			for i, e := range []error{m.rational.ToOwnedString(&a, &later), m.rational.CleanupOwnedString(), m.rational.Neg(&a), m.rational.Del(&a), m.integer.SetInt32(&a.Num, 7), m.rational.CloseChecked(), m.rational.SourceFinalize(), m.Close(), m.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("actual denominator callback guard", i, e)
				}
			}
			if a != before || b.Used() != used || !out.Matches("3/") {
				t.Fatal("transport changed stopped rational string")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, m.integer.Mul2kInPlace(&a.Den, 8192))
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.rational.ToOwnedString(&a, &out)
	armed = false
	if !errors.Is(body, Failure{Kind: FailureMemout}) || calls != 1 || !m.rational.stringScope.active {
		t.Fatal("actual denominator transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &a.Num, &a.Den)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate denominator graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, m.rational.CleanupOwnedString())
	serviceMust(t, m.rational.Del(&a))
	serviceMust(t, m.Close())
	serviceMust(t, r.Deallocate(&pad))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("denominator transport refunds", b.Used())
	}
}
