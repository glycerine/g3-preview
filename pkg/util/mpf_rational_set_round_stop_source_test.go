// Source: src/util/mpf.cpp:233-300, src/util/mpf.cpp:1984-2027.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Initial straight-line TLS scan did not reach Round; retained as a recipe
// diagnostic. Its actual earlier stops are preserved in review evidence.
func rationalSetInitialRoundNUMStopDiagnostic(t *testing.T) {
	found := false
	for offset := int64(-99000); offset >= -99980; offset -= 20 {
		if rationalSetRoundNUMStopRecipe(t, offset) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("recipe did not reach actual youngest Round NUM DT")
	}
}
func rationalSetRoundNUMStopRecipe(t *testing.T, offset int64) bool {
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	q := NewRational()
	zero := NewRational()
	ex := NewInt(-5000)
	var out, after ArbitraryFloat
	serviceMust(t, m.rational.SetFraction(&q, 13, 8))
	serviceMust(t, m.SetExponentRational(&out, 8, 2001, ArbitraryFloatNearestEven, &ex, &q))
	serviceMust(t, m.CleanupExponentRationalSet())
	serviceMust(t, m.CleanupRound())
	serviceMust(t, m.integer.EnsureTmp(0, 80))
	serviceMust(t, m.integer.EnsureTmp(1, 80))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := offset - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); e != arena.ErrBounds {
		t.Fatal(e)
	}
	var receipt ArbitraryFloatRationalSetReceipt
	body := m.SetExponentRationalWithReceipt(&out, 8, 2001, ArbitraryFloatNearestEven, &ex, &q, &receipt)
	s := &m.RoundScope
	t.Log("actual reached offset", offset, "child", body, "source", s.sourceCause, "flags", s.stickyConstructed, s.capConstructed, s.sigmaConstructed, "parent", m.rationalSet.sourceCause)
	matched := s.sourceCause == arena.ErrBounds && s.stickyConstructed && s.capConstructed && s.sigmaConstructed && m.rationalSet.sourceCause == s.sourceCause && m.rationalSet.sqLive && m.rationalSet.tLive && m.rationalSet.pLive && m.rationalSet.expLive && m.rationalSet.sigLive && errors.Is(body, s.sourceCause)
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &q.Num, &q.Den, &out.significand, &s.sticky, &s.cap, &s.sigma, &m.rationalSet.sig.Num, &m.rationalSet.sig.Den, &m.rationalSet.exp, &m.rationalSet.p, &m.rationalSet.p2, &m.rationalSet.prev, &m.rationalSet.t.Num, &m.rationalSet.t.Den, &m.rationalSet.sq.Num, &m.rationalSet.sq.Den, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
	if b.Used() != rt.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("before recovery named cold graph")
	}
	if matched {
		used := b.Used()
		stage := m.completedMaps
		sourceEnded := m.sourceEnded
		scope := m.rationalSet
		round := m.RoundScope
		if e = m.SourceFinalize(); e != s.sourceCause || m.completedMaps != stage || m.sourceEnded != sourceEnded || b.Used() != used || m.rationalSet != scope || m.RoundScope != round {
			t.Fatal("younger stop crossed older owner", e)
		}
	}
	recovery, e := rt.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, false))
	serviceMust(t, m.CleanupValue(&out))
	serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, true))
	if matched {
		h, e := m.rationalSetHistory(&receipt)
		serviceMust(t, e)
		beforeBody, beforeSource, beforeOwn := h.bodyCause, h.sourceCause, h.ownCause
		oldCopy, oldRound, oldFrame, oldInvocation := h.gcd, h.roundChild, h.roundFrame, h.roundInvocation
		b.FailAfter(0)
		serviceMust(t, m.SetExponentRational(&after, 8, 24, ArbitraryFloatNearestEven, &ex, &zero))
		b.FailAfter(-1)
		current := m.rationalSet
		afterBefore := after
		used := b.Used()
		serviceMust(t, m.CleanupExponentRationalSetWithReceipt(&receipt, true))
		if h.bodyCause != beforeBody || h.sourceCause != beforeSource || h.ownCause != beforeOwn || h.gcd != oldCopy || h.roundChild != oldRound || h.roundFrame != oldFrame || h.roundInvocation != oldInvocation || m.rationalSet != current || after != afterBefore || b.Used() != used {
			t.Fatal("old ACK/history touched later inline invocation")
		}
	}
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.Del(&after))

	if !matched {
		if source := m.SourceFinalize(); source != nil && source != m.rationalSet.sourceCause && !m.integer.sourceGuard {
			t.Fatal("unclassified enclosing source stop", source)
		}
	}
	contextTLSFinish(t, rt, m.Close)
	for _, p := range []*Block{&lane, &recovery} {
		contextTLSFinish(t, rt, func() error {
			if *p == (Block{}) {
				return nil
			}
			return rt.Deallocate(p)
		})
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	return matched
}
