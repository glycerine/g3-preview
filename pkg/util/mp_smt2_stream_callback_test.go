// Source: src/util/mpz.cpp:1415-1416; actual negative temporary COPY transport.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestRationalSMT2StreamCopyTransportGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	a := Rational{Num: NewInt(-1), Den: NewInt(3)}
	var out OwnedStringStream
	armed, calls := false, 0
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if armed {
			calls++
			s, p := &m.integer.smt2Stream, &m.rational.smt2Stream
			if !s.executing || !s.tmpLive || !p.executing || !m.integer.CopyAllocation.Executing {
				t.Fatal("actual negative COPY scopes missing")
			}
			before, tmp, used := a, s.tmp, b.Used()
			if out.Buffer.Position != 3 {
				t.Fatal("COPY should follow rational prefix and precede negative prefix", out.Buffer.Position)
			}
			for i, e := range []error{m.rational.DisplaySMT2Into(&out, &a, false), m.integer.DisplaySMT2Into(&out, &a.Num, false), m.rational.CleanupSMT2Stream(), m.integer.CleanupSMT2Stream(), m.rational.Del(&a), m.rational.Neg(&a), m.integer.SetInt32(&a.Num, 7), m.rational.CloseChecked(), m.rational.SourceFinalize(), m.integer.CloseChecked(), m.integer.SourceFinalize(), m.Close(), m.SourceFinalize()} {
				if !errors.Is(e, ErrNumericPending) {
					t.Fatal("negative COPY callback guard", i, e)
				}
			}
			if a != before || s.tmp != tmp || b.Used() != used || out.Buffer.Position != 3 {
				t.Fatal("callback changed lexical graph")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	serviceMust(t, out.Construct(b))
	serviceMust(t, m.integer.Mul2kInPlace(&a.Num, 40000))
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	body := m.rational.DisplaySMT2Into(&out, &a, false)
	armed = false
	if !errors.Is(body, Failure{Kind: FailureMemout}) || calls != 1 || !m.rational.smt2Stream.active || !m.integer.smt2Stream.active {
		t.Fatal("actual negative COPY transport missing", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &a.Num, &a.Den, &m.integer.smt2Stream.tmp)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+uint64(len(out.Buffer.String.External)+len(out.Buffer.String.Pending)) {
		t.Fatal("immediate COPY graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.rational.CleanupSMT2Stream())
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "(/ " {
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
		t.Fatal("COPY transport refunds", b.Used())
	}
}

func TestRationalSMT2StreamRefusedChildDoesNotAdoptOlderCopy(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 60; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		v := NewInt(-1)
		serviceMust(t, m.integer.Mul2kInPlace(&v, 4000))
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		b.FailAfter(ordinal)
		body := m.integer.DisplaySMT2Into(&out, &v, false)
		s := &m.integer.smt2Stream
		matched := body != nil && s.active && m.integer.CopyAllocation.Roots.Cell.Object != (ObjectRef{})
		b.FailAfter(-1)
		if matched {
			reached = true
			invocation, roots, tmp, used := s.invocation, m.integer.CopyAllocation.Roots, s.tmp, b.Used()
			var later OwnedStringStream
			serviceMust(t, later.Construct(b))
			a := Rational{Num: NewInt(7), Den: NewInt(3)}
			if e := m.rational.DisplaySMT2Into(&later, &a, false); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older child refusal", e)
			}
			if m.rational.smt2Stream.child != 0 {
				t.Fatal("refused integer child was adopted")
			}
			serviceMust(t, m.rational.CleanupSMT2Stream())
			if !s.active || s.invocation != invocation || m.integer.CopyAllocation.Roots != roots || s.tmp != tmp || b.Used() != used {
				t.Fatal("rational cleanup changed older COPY")
			}
			got, e := (OwnedStringView{&later.Buffer.String, later.Buffer.String.Generation, later.Buffer.Position}).String()
			serviceMust(t, e)
			if got != "(/ " {
				t.Fatal("refused borrowed prefix", got)
			}
			later.Buffer.OwnAbort()
			integerPhysicalAudit(t, m.integer, &v, &s.tmp)
		}
		serviceMust(t, m.integer.CleanupSMT2Stream())
		out.Buffer.OwnAbort()
		serviceMust(t, m.integer.Del(&v))
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
		if b.Used() != 0 {
			t.Fatal("older COPY refunds", b.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no actual retained COPY descriptor reached")
	}
}
