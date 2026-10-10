// Source: src/util/mpf.cpp:1584-1588; actual conversion allocation transport.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestArbitraryFloatSMT2StreamConversionTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var x ArbitraryFloat
	var out OwnedStringStream
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			s, c := &m.smt2Stream, &m.rationalConversion
			if !s.executing || !c.executing || !s.qLive || c.invocation == 0 || s.format != 0 {
				t.Fatal("actual nested conversion scope missing", s, c)
			}
			// The caller captures the child invocation only when the child returns.
			before, q, a, n, d, used := x, s.q, c.a, c.n, c.d, b.Used()
			var later OwnedStringStream
			for i, e := range []error{m.DisplaySMT2Into(&later, &x, true), m.CleanupSMT2Stream(), m.CleanupRationalConversion(), m.CleanupValue(&x), m.Del(&x), m.Close(), m.SourceFinalize(), m.integer.SetInt32(&x.significand, 7), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.rational.CloseChecked(), m.rational.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("nested conversion callback guard", i, e)
				}
			}
			if x != before || s.q != q || c.a != a || c.n != n || c.d != d || b.Used() != used || out.Buffer.Position != 0 {
				t.Fatal("callback changed actual lexical fields")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, out.Construct(b))
	sig := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 40000, false, 0, &sig))
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsZero(&x)
	serviceMust(t, e)
	_, e = m.powerTwoWrapper(x.SBits()-1, false)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.DisplaySMT2Into(&out, &x, true)
	armed = false
	s, c := &m.smt2Stream, &m.rationalConversion
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 || !s.active || s.conversion != c.invocation || s.format != 0 {
		t.Fatal("actual conversion transport missing", body, calls)
	}
	raw := []umemory.Block{pad}
	if c.transferred.Block() != (umemory.Block{}) {
		raw = append(raw, c.transferred.Block())
	}
	integerPhysicalAuditWithRaw(t, m.integer, raw, &x.significand, &s.q.Num, &s.q.Den, &c.a.significand, &c.n, &c.d, &m.rational.NTmp)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("immediate parent graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupSMT2Stream())
	serviceMust(t, m.CleanupValue(&x))
	out.Buffer.OwnAbort()
	serviceMust(t, r.Deallocate(&pad))
	if source := m.SourceFinalize(); source != nil {
		z := m.integer
		if !errors.Is(source, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
			t.Fatal(source)
		}
		index := int(z.SourceFinalizeStage-2) / 2
		ref := z.Tmp[index]
		if z.SourceFinalizeStage%2 == 1 {
			ref = z.Arg[index]
		}
		if ref != 0 {
			t.Fatal("unproved source scratch hole", ref)
		}
	}
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("nested transport refunds", b.Used())
	}
}
