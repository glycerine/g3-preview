// Source: src/util/mpf.cpp:1695-1711; src/util/mpf.h:275-286.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatToRationalActualReusedFrameRejectsSavedContext(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var foreign *RationalManager
	var old integerAllocationInvocationContext
	calls := 0
	armed := false
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			s := &m.rationalConversion
			current := s.allocationContext()
			if current.conversionFrame(m.integer, false) != s || current.conversionFrame(m.integer, true) != nil {
				t.Fatal("callback history/acquisition partition")
			}
			if current.conversionFrame(foreign.z, false) != nil {
				t.Fatal("foreign Integer adopted source tuple")
			}
			copied := *s
			invalid := current
			invalid.mpfConversion = &copied
			if invalid.conversionFrame(m.integer, false) != nil {
				t.Fatal("copied scope accepted")
			}
			if calls == 1 {
				old = current
			} else if old.conversionFrame(m.integer, false) != nil || old.mpfConversionInvocation == current.mpfConversionInvocation {
				t.Fatal("saved context retargeted to reused embedded frame")
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	m, e = NewArbitraryFloatManager(cfg)
	serviceMust(t, e)
	foreign, e = NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	limbs := make([]uint32, 1200)
	limbs[1199] = 1
	var sig Int
	serviceMust(t, m.integer.SetLimbs(&sig, limbs))
	var x ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 8, 40000, false, 0, &sig))
	serviceMust(t, m.integer.Del(&sig))
	var pads []memory.Block
	for i := 0; i < 2; i++ {
		pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
		serviceMust(t, e)
		pads = append(pads, pad)
		out := NewRational()
		serviceMust(t, rt.SetMaxSize(1))
		armed = true
		body := m.ToRational(&x, foreign, &out)
		armed = false
		if !errors.Is(body, memory.Failure{Kind: memory.FailureMemout}) || m.rationalConversion.numericPhase != 1 || calls != i+1 {
			t.Fatal("actual initial-copy callback", body, calls, m.rationalConversion.numericPhase)
		}
		raw := append([]memory.Block(nil), pads...)
		if block := m.rationalConversion.transferred.Block(); block != (memory.Block{}) {
			raw = append(raw, block)
		}
		integerPhysicalAuditWithRaw(t, m.integer, raw, &x.significand, &m.rationalConversion.a.significand)
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupRationalConversion())
		serviceMust(t, foreign.Del(&out))
		used, state := b.Used(), rt.State()
		if ref, e := m.integer.allocateWithSourceContext(6, 1, nil, old); ref != 0 || e != memory.ErrRuntimeOwner {
			t.Fatal("invalid explicit saved context fell back to ordinary allocation", ref, e)
		}
		if b.Used() != used || rt.State() != state {
			t.Fatal("rejected explicit context mutated allocator")
		}

	}
	serviceMust(t, foreign.CloseChecked())
	serviceMust(t, m.Del(&x))
	for i := range pads {
		serviceMust(t, rt.Deallocate(&pads[i]))
	}
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
