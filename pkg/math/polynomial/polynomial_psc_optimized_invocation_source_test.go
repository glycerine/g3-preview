// Source: src/math/polynomial/polynomial.cpp:5751-5831,5897-5902.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialPSCOptimizedCapturedChildrenEveryReservationPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 1024; prefix++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
		out, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		t.Cleanup(func() { full.originalBudget.FailAfter(-1); serviceMust(t, m.CleanupPSC()); serviceMust(t, out.Close()) })
		full.originalBudget.FailAfter(prefix)
		returned := m.PSCChain(p, q, 0, out)
		s := &m.pscLocal
		receipt := polynomialPSCInvocationReceipt{m, s, s.invocation}
		audit := func() {
			ps := []Polynomial{p, q, out.state.incoming}
			n, e := out.Size()
			serviceMust(t, e)
			for i := out.state.drain; i < n; i++ {
				v, e := out.Get(i)
				serviceMust(t, e)
				ps = append(ps, v)
			}
			for i := uint8(0); i < s.constructed; i++ {
				ps = append(ps, s.refs[i].root.value, s.refs[i].root.incoming)
			}
			next := &m.pscNext
			for i := uint8(0); i < next.constructed; i++ {
				ps = append(ps, next.refs[i].value, next.refs[i].incoming)
			}
			laz := &m.lazardLocal
			ps = append(ps, laz.x.value, laz.x.incoming, laz.c.value, laz.c.incoming)
			raw := []util.Block{out.state.values.raw, out.state.values.pending, out.state.values.old}
			h := &m.pscBuffer
			if h.constructed {
				raw = append(raw, h.raw, h.pending)
				ps = append(ps, h.incoming)
				if !h.finalized {
					for i := h.cursor; i < h.count; i++ {
						var v Polynomial
						var e error
						if h.pendingOwns {
							v, e = pscReadHandle(m, h.pending, i)
						} else {
							v, e = h.physical(m, i)
						}
						serviceMust(t, e)
						ps = append(ps, v)
					}
				}
			}
			headers := out.header + uint64(cap(out.state.values.retired))*uint64(reflect.TypeOf(util.Int{}).Size())
			polynomialImpPhysicalAuditNamedRaw(t, full, ps, nil, headers, raw)
		}
		audit()
		copied := *s
		wrong := receipt
		wrong.frame = &copied
		state := m.memory.State()
		serviceMust(t, m.cleanupPSCInvocation(&wrong, true))
		if m.memory.State() != state {
			t.Fatal("copied source frame consumed actual roots", prefix)
		}
		body, source := s.bodyCause, s.sourceCause
		own := m.cleanupPSCInvocation(&receipt, false)
		if own == nil {
			own = m.cleanupPSCInvocation(&receipt, true)
		}
		audit()
		full.originalBudget.FailAfter(-1)
		serviceMust(t, m.cleanupPSCInvocation(&receipt, false))
		serviceMust(t, m.cleanupPSCInvocation(&receipt, true))
		if s.bodyCause != body || s.sourceCause != source {
			t.Fatal("own ACK erased actual parent history", prefix)
		}
		if returned == nil {
			n, e := out.Size()
			serviceMust(t, e)
			if n != 3 {
				t.Fatal("actual optimized layout", n)
			}
			for i, want := range []int64{17109, 132, 16} {
				v, e := out.Get(uint32(i))
				serviceMust(t, e)
				polynomialQueryExpect(t, m, v, map[[3]uint32]int64{{}: want})
			}
			success = true
		}
		if returned != nil {
			oldText := returned.Error()
			serviceMust(t, m.PSCChain(p, q, 0, out))
			state := m.memory.State()
			serviceMust(t, m.cleanupPSCInvocation(&receipt, true))
			if m.memory.State() != state || returned.Error() != oldText || s.invocation == receipt.invocation {
				t.Fatal("old invocation acquired retry or changed returned error", prefix)
			}
		}
		serviceMust(t, out.Close())
		if success {
			t.Log("actual first success", prefix)
			break
		}
		_ = own
	}
	if !success {
		t.Fatal("actual optimized horizon")
	}
}

// Returning Go checkpoint reentry must refuse before the source caller's S.reset.
func TestPolynomialPSCOptimizedPublicReentryPreservesPartialOutput(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
	out, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupPSC()); serviceMust(t, out.Close()) })
	checked := false
	m.config.Checkpoint = func() bool {
		if !checked && m.pscLocal.executing {
			n, e := out.Size()
			serviceMust(t, e)
			if n > 0 {
				checked = true
				v, e := out.Get(0)
				serviceMust(t, e)
				state := m.memory.State()
				used := full.originalBudget.Used()
				if e := m.PSCChain(p, q, 0, out); e != ErrPolynomialPending {
					t.Fatal("public reentry accepted", e)
				}
				started, stage := m.sourceStarted, m.sourceStage
				if e := m.SourceFinalizePrefix(); e != ErrPolynomialPending || m.sourceStarted != started || m.sourceStage != stage {
					t.Fatal("executing optimized caller source owner progressed", e, m.sourceStarted, m.sourceStage)
				}
				nn, e := out.Size()
				serviceMust(t, e)
				vv, e := out.Get(0)
				serviceMust(t, e)
				if n != nn || v != vv || state != m.memory.State() || used != full.originalBudget.Used() {
					t.Fatal("reentry advanced source output RESET")
				}
			}
		}
		return true
	}
	serviceMust(t, m.PSCChain(p, q, 0, out))
	if !checked {
		t.Fatal("actual partial-output checkpoint absent")
	}
}
