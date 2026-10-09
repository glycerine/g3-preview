// Source: src/util/mpf.cpp:1695-1711; src/util/mpz.cpp:730-817.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// This exercises the real signed-host TLS owner-primitive domain. It is not an
// ordinary mathematical allocation failure or an injected destructor callback.
func TestArbitraryFloatToRationalGCDSourceStopsOlderLocals(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	m, e := NewArbitraryFloatManager(cfg)
	serviceMust(t, e)
	var x ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 2048, false, 0, &one))
	out := NewRational()
	serviceMust(t, m.ToRational(&x, m.rational, &out))
	serviceMust(t, m.CleanupRationalConversion())
	old := make([]uint32, 15000)
	old[14999] = 1
	serviceMust(t, m.integer.SetLimbs(&m.rational.NTmp, old))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-50000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	returned := m.ToRational(&x, m.rational, &out)
	s := &m.rationalConversion
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &x.significand, &out.Num, &out.Den, &m.rational.NTmp, &s.a.significand, &s.n, &s.d)
	var maps uint64
	want := m.header + m.ValueCopyHistoryBytes() + m.RationalConversionHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + maps + integerActualOwnedBytes(m.integer)
	if b.Used() != want {
		t.Fatal("immediate named conversion physical/cold graph", b.Used(), want)
	}
	child := m.integer.RawInvocationSourceCause(&s.outputGCD)
	if !errors.Is(returned, arena.ErrBounds) || child == nil || s.sourceCause != child || s.sourceCursor != 0 {
		t.Fatal("actual youngest GCD source stop", returned, child, s.sourceCause, s.sourceCursor, s.numericPhase)
	}
	captured := returned
	cause := s.sourceCause
	nonce := s.invocation
	before := [3]Int{s.d, s.n, s.a.significand}
	state := rt.State()
	if e := m.finishRationalConversionSource(); e != cause || before != ([3]Int{s.d, s.n, s.a.significand}) || rt.State() != state {
		t.Fatal("sticky unread older suffix", e)
	}
	type powerSourceSnapshot struct {
		block             Block
		delDone, freeDone bool
	}
	var powersBefore []powerSourceSnapshot
	for w := m.PowerWrappers; w != nil; w = w.Next {
		powersBefore = append(powersBefore, powerSourceSnapshot{w.Block, w.SourceDelDone, w.sourceFreeDone})
	}
	ownerState, ownerBytes := rt.State(), b.Used()
	if e := m.SourceFinalize(); e != cause {
		t.Fatal("class source suffix ignored younger conversion stop", e, cause)
	}
	if !m.rationalReady || m.sourceEnded || rt.State() != ownerState || b.Used() != ownerBytes {
		t.Fatal("class source suffix mutated older owner", m.rationalReady, m.sourceEnded, rt.State(), ownerState)
	}
	for w, i := m.PowerWrappers, 0; w != nil; w, i = w.Next, i+1 {
		if got := (powerSourceSnapshot{w.Block, w.SourceDelDone, w.sourceFreeDone}); got != powersBefore[i] {
			t.Fatal("class source suffix touched older cached power", i, got, powersBefore[i])
		}
	}
	ownStop := m.Close()
	if !errors.Is(ownStop, arena.ErrBounds) || s.ownCause != ownStop {
		t.Fatal("actual first own stop", ownStop, s.ownCause)
	}
	for w, i := m.PowerWrappers, 0; w != nil; w, i = w.Next, i+1 {
		if got := (powerSourceSnapshot{w.Block, w.SourceDelDone, w.sourceFreeDone}); got != powersBefore[i] {
			t.Fatal("class owning stop touched older cached power", i, got, powersBefore[i])
		}
	}
	if !m.rationalReady || m.closed || m.ownCause != ownStop {
		t.Fatal("class owning stop lost original phase", m.rationalReady, m.closed, m.ownCause)
	}
	firstOwn := s.ownCause
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &x.significand, &out.Num, &out.Den, &m.rational.NTmp, &s.a.significand, &s.n, &s.d)
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, m.CleanupRationalConversion)
	body, source, own, ok := m.RationalConversionCauses(nonce)
	if !ok || source != cause || own != firstOwn || !errors.Is(captured, cause) || body == nil {
		t.Fatal("own ACK lost original history", body, source, ok)
	}
	contextTLSFinish(t, rt, func() error { return m.rational.Del(&out) })
	contextTLSFinish(t, rt, func() error { return m.Del(&x) })
	contextTLSFinish(t, rt, m.Close)
	if e := rt.Deallocate(&lane); e != nil && !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e := rt.Deallocate(&recovery); e != nil && !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
