// Source: src/util/mpq.cpp:157-164; actual second RAW COPY allocation transport.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"testing"
)

func TestRationalDecimalStreamDenominatorCopyTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	a := Rational{Num: NewInt(-3), Den: NewInt(1)}
	var out utext.OwnedStringStream
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			s := &m.rational.decimalStream
			if !s.executing || s.count != 3 || s.sourceCursor != 0 || s.values[1].Val != -3 || s.values[0] != NewInt(0) || out.Buffer.Position != 7 {
				t.Fatal("actual second COPY declaration order missing", s.count, s.values, out.Buffer.Position)
			}
			before, values, used := a, s.values, b.Used()
			for i, e := range []error{m.rational.DisplayDecimalInto(&out, &a, 10, false), m.rational.CleanupDecimalStream(), m.rational.Neg(&a), m.rational.Del(&a), m.integer.SetInt32(&a.Num, 7), m.rational.CloseChecked(), m.rational.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.Close(), m.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("decimal COPY callback guard", i, e)
				}
			}
			if a != before || s.values != values || b.Used() != used || out.Buffer.Position != 7 {
				t.Fatal("callback changed lexical graph")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, out.Construct(b))
	serviceMust(t, out.AppendCString("prefix:"))
	serviceMust(t, m.integer.Mul2kInPlace(&a.Den, 40000))
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.rational.DisplayDecimalInto(&out, &a, 10, false)
	armed = false
	s := &m.rational.decimalStream
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls != 1 || !s.active || s.count != 3 || s.sourceCursor != 0 || s.bodySucceeded {
		t.Fatal("actual second COPY transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &a.Num, &a.Den, &s.values[0], &s.values[1], &s.values[2], &s.values[3])
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+out.Buffer.String.StorageBytes() {
		t.Fatal("immediate second COPY graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.rational.CleanupDecimalStream())
	if s.sourceCursor != 0 || s.ownCursor != 4 || s.count != 3 {
		t.Fatal("owning cleanup credited unreached ten or end label")
	}
	got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "prefix:" {
		t.Fatal("borrowed prefix changed", got)
	}
	out.Buffer.OwnAbort()
	serviceMust(t, m.rational.Del(&a))
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
		t.Fatal("decimal COPY refunds", b.Used())
	}
}
