// Source: src/util/mpf.cpp:1660-1693; checked Runtime TLS release profile.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"testing"
)

func TestArbitraryFloatBinaryStringTemporarySourceStop(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var lane umemory.Block
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			armed = false
			calls++
			s := &m.binaryString
			if !s.executing || !s.tmpLive || s.tmp.Cell == 0 || !s.bitsLive {
				t.Fatal("must fail after actual tmp COPY", s.tmp, s.tmpLive, s.bitsLive)
			}
			serviceMust(t, m.memory.SetMaxSize(0))
			// These are real public allocate/reallocate/deallocate operations on the
			// checked signed-host TLS profile, not a fabricated destructor callback.
			contextTLSNearMinimum(t, m.memory)
			delta := int64(-99950) - m.memory.State().LocalSize
			if e := m.memory.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("failed public TLS realloc", e)
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 40000, false)}
	digits := make([]uint32, 1250)
	digits[len(digits)-1] = 1
	serviceMust(t, m.integer.SetLimbs(&x.significand, digits))
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsZero(&x)
	serviceMust(t, e)
	_, e = m.BiasExp(8, 0)
	serviceMust(t, e)
	lane, e = r.Allocate(8)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(94000 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	var out utext.OwnedString
	body := m.ToStringBinary(&x, 0, 0, &out)
	armed = false
	s := &m.binaryString
	if calls != 1 || !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || !errors.Is(body, arena.ErrBounds) || s.sourceCause == nil || !s.tmpLive || s.bitsLive || !s.outLive || !out.Matches("#b0 #b01111111 ") {
		t.Fatal("actual source tmp stop missing", calls, body, s.sourceCause, s.tmpLive, s.bitsLive, s.outLive)
	}
	before, used, value, cause := r.State(), budget.Used(), x, s.sourceCause
	if e := m.finishBinaryStringSource(); e != cause {
		t.Fatal("unsticky binary source", e)
	}
	if e := m.SourceFinalize(); e != cause {
		t.Fatal("class ignored younger stop", e)
	}
	if r.State() != before || budget.Used() != used || x != value || m.sourceEnded || !m.rationalReady || !out.Matches("#b0 #b01111111 ") {
		t.Fatal("source stop touched older owners")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane, pad}, &x.significand, &s.tmp)
	if budget.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("source stop graph", budget.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupBinaryString())
	if s.sourceCause != cause || s.tmpLive || s.outLive || s.active || !out.Closed {
		t.Fatal("owning join erased source history")
	}
	contextTLSFinish(t, r, func() error { return m.CleanupValue(&x) })
	contextTLSFinish(t, r, m.Close)
	for _, block := range []*umemory.Block{&lane, &pad, &recovery} {
		contextTLSFinish(t, r, func() error {
			if *block == (umemory.Block{}) {
				return nil
			}
			return r.Deallocate(block)
		})
	}
	contextTLSFinish(t, r, r.Close)
	if budget.Used() != 0 {
		t.Fatal("source stop refunds", budget.Used())
	}
}
