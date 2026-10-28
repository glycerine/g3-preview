package numeral

// Source: src/util/mpf.cpp:1695-1711,1922-1945.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatToRationalAllocationCallbackGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var rt *umemory.Runtime
	var qm *RationalManager
	out := NewRational()
	var a ArbitraryFloat
	active := false
	calls := 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			used := b.Used()
			stage := m.integer.SourceFinalizeStage
			own := m.ownCause
			source := m.sourceCause
			if e := m.Close(); e != ErrNumericPending {
				t.Fatal("MPF own reentry", e)
			}
			if e := m.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("MPF source reentry", e)
			}
			if e := m.integer.CloseChecked(); e != ErrNumericPending {
				t.Fatal("Integer own reentry", e)
			}
			if e := m.integer.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("Integer source reentry", e)
			}
			if e := qm.CloseChecked(); e != ErrNumericPending {
				t.Fatal("Rational own reentry", e)
			}
			if e := qm.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("Rational source reentry", e)
			}
			if e := m.CleanupRationalConversion(); e != ErrNumericPending {
				t.Fatal("false child reentry", e)
			}
			if e := m.CleanupRationalConversion(); e != ErrNumericPending {
				t.Fatal("true child reentry", e)
			}
			if e := m.ToRational(&a, qm, &out); e != ErrNumericPending {
				t.Fatal("same manager format reentry", e)
			}
			if b.Used() != used || m.integer.SourceFinalizeStage != stage || m.ownCause != own || m.sourceCause != source || m.rationalConversion.ownCause != nil {
				t.Fatal("executing cleanup changed older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	qm, e = NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	sig := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 40000, false, 0, &sig))
	_, e = m.IsNaN(&a)
	serviceMust(t, e)
	_, e = m.IsZero(&a)
	serviceMust(t, e)
	_, e = m.powerTwoWrapper(a.SBits()-1, false)
	serviceMust(t, e)
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	body := m.ToRational(&a, qm, &out)
	active = false
	if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls == 0 {
		t.Fatal("actual callback leaf not reached", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad, m.rationalConversion.transferred.Block()}, &a.significand, &m.rational.NTmp)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, m.CleanupRationalConversion())
	if m.rationalConversion.bodyCause != body || m.rationalConversion.ownCause != nil {
		t.Fatal("callback failure history", m.rationalConversion.bodyCause, m.rationalConversion.ownCause)
	}
	serviceMust(t, qm.Del(&out))
	serviceMust(t, qm.CloseChecked())
	serviceMust(t, m.Del(&a))
	serviceMust(t, rt.Deallocate(&pad))
	if source := m.SourceFinalize(); source != nil {
		if !errors.Is(source, arena.ErrInvalidSlot) || !m.integer.sourceGuard || m.integer.SourceFinalizeStage < 2 || m.integer.SourceFinalizeStage > 5 {
			t.Fatal(source)
		}
		index := int(m.integer.SourceFinalizeStage-2) / 2
		ref := m.integer.Tmp[index]
		if m.integer.SourceFinalizeStage%2 == 1 {
			ref = m.integer.Arg[index]
		}
		if ref != 0 {
			t.Fatal("unproved source scratch hole", ref)
		}
	}
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
