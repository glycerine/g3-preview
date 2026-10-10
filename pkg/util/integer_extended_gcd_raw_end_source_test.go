// Source: src/util/mpz.cpp:915-985; src/util/memory_manager.cpp.
// This is the real guarded signed-host TLS arithmetic domain, not ordinary mathematical exhaustion.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerPublicExtendedGCDRawBodyAndEndLifetimes(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	m, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	var a, b, x, y, g Int
	digits := make([]uint32, 64)
	digits[0] = 7
	digits[63] = 1
	serviceMust(t, m.SetLimbs(&a, digits))
	serviceMust(t, m.EnsureTmp(0, 64))
	serviceMust(t, m.EnsureTmp(1, 64))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed allocation guard", e)
	}
	returned := m.ExtendedGCD(&a, &b, &x, &y, &g)
	s := &m.ExtendedRaw
	r := integerRawInvocationReceipt{m, s, s.Invocation}
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("actual reached source error missing", returned)
	}
	captured := returned.Error()
	source := m.RawInvocationSourceCause(&r)
	if SourceExtendedGCDDebug {
		if s.BodySucceeded || s.ended || source != nil || !errors.Is(s.BodyCause, arena.ErrBounds) || g != (Int{}) {
			t.Fatal("literal diagnostic failed-body prefix", s, g, returned)
		}
	} else {
		cmp, e := m.Compare(&g, &a)
		serviceMust(t, e)
		if cmp != 0 || !s.BodySucceeded || source == nil {
			t.Fatal("source output before reached end DEL", s, g, a, returned)
		}
	}
	named := []*Int{&a, &b, &x, &y, &g}
	for i := uint8(0); i < s.Count; i++ {
		named = append(named, &s.Values[i])
	}
	integerPhysicalAuditWithRaw(t, m, []umemory.Block{lane}, named...)
	if budget.Used() != integerActualOwnedBytes(m) {
		t.Fatal("immediate cold RAW graph", budget.Used(), integerActualOwnedBytes(m))
	}
	before := s.Values
	own := m.CleanupRawInvocation(&r, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("false own phase", own)
	}
	if m.RawInvocationSourceCause(&r) != source || SourceExtendedGCDDebug && s.Values != before {
		t.Fatal("false owning phase changed unexecuted RAW end DEL/history")
	}
	integerPhysicalAuditWithRaw(t, m, []umemory.Block{lane}, named...)
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupRawInvocation(&r, true))
	if m.RawInvocationSourceCause(&r) != source || returned.Error() != captured {
		t.Fatal("owning acknowledgement changed source cause")
	}
	integerPhysicalAuditWithRaw(t, m, []umemory.Block{lane, recovery}, &a, &b, &x, &y, &g)
	two, zero := NewInt(2), NewInt(0)
	serviceMust(t, m.ExtendedGCD(&two, &zero, &x, &y, &g))
	if s.Invocation == r.Invocation || s.sourceCause != nil || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("new actual invocation mutated old returned error")
	}
	for _, v := range []*Int{&a, &b, &x, &y, &g} {
		serviceMust(t, m.Del(v))
	}
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, m.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("explicit complete owner graph remains", budget.Used())
	}
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("reached RAW end DEL source stop discarded", returned)
	}
}
