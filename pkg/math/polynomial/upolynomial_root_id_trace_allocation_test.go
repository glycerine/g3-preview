package polynomial

import (
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strconv"
	"testing"
)

// Refuse each reached reservation on a fresh complete graph through first
// success. No global integer/display pending cleanup may acknowledge the child.
func TestUPolynomialRootIDTraceReachedReservations(t *testing.T) {
	success, failed, diagnosticFailure := false, false, false
	traceEnabled := false
	for after := int64(0); after < 2048 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			z := m.config.Integer
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("rem_bug")
			serviceMust(t, z.BindTrace(tags))
			traceEnabled = tags.ShouldTrace("rem_bug")
			z.Budget().FailAfter(after)
			var receipt upolyRootIDInvocationReceipt
			id, body := m.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
			z.Budget().FailAfter(-1)
			serviceMust(t, z.BindTrace(nil))
			q := &m.rootID
			if body == nil {
				success = true
				if id != 0 {
					t.Fatal("independent root count below lower1", id)
				}
			} else {
				failed = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected refusal", body)
				}
				for i := range q.remainderTrace.displays {
					child := &q.remainderTrace.displays[i]
					if m.matchesDisplayReceipt(child) && child.frame.bodyCause != nil {
						diagnosticFailure = true
					}
				}
				for i := range q.remainderTrace.insertions {
					if q.remainderTrace.insertions[i].Active() {
						diagnosticFailure = true
					}
				}
			}
			source, scalarSource := q.sourceCursor, q.scalar.sourceCursor
			serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
			if q.active || q.remainderTracePending() || source != q.sourceCursor || scalarSource != q.scalar.sourceCursor {
				t.Fatal("exact parent cleanup lost source history")
			}
			if m.display.active {
				t.Fatal("parent lost embedded diagnostic display")
			}
			for f := m.displayFrames; f != nil; f = f.next {
				if f.active {
					t.Fatal("parent lost allocating diagnostic display frame")
				}
			}
			algebraicComparisonAudit(t, c, &root)
			serviceMust(t, c.Del(&root))
		})
	}
	if !success || !failed || traceEnabled && !diagnosticFailure {
		t.Fatal("root-ID trace allocation campaign incomplete", success, failed, traceEnabled, diagnosticFailure)
	}
}

func TestUPolynomialRootIDTraceFailureTransportRejectsDisplayContext(t *testing.T) {
	reached := false
	for after := int64(0); after < 128 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			var m *UPolynomialManager
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || m.currentDisplay == nil {
					return f
				}
				armed = false
				reached = true
				display, z := m.currentDisplay, m.config.Integer
				context := display.context()
				used := z.Budget().Used()
				probe := unumeral.NewInt(0)
				if context.upolyDisplayFrame(z) != nil || display.NumericCharacterValid(z.Naturals(), true) {
					t.Error("failure transport acquired paused display context")
				}
				if e := z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7); e == nil || probe.Val != 0 || probe.Cell != 0 || used != z.Budget().Used() {
					t.Error("failure transport mutated numeral through display context", e, probe)
				}
				// Reuse the policy already issued to the allocating primitive,
				// as well as testing a freshly resolved invocation tuple.
				inflight := unumeral.NumericAllocationContext{}.WithCaller(&display.numeric.hooks)
				if e := z.WithAllocationContext(inflight).SetUint32(&probe, 9); e == nil || probe.Val != 0 || probe.Cell != 0 || used != z.Budget().Used() {
					t.Error("failure transport reused in-flight display policy", e, probe)
				}
				return f
			}
			c, owner, _ := algebraicNegTLSFixture(t, transport)
			m = owner
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			z := m.config.Integer
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("rem_bug")
			serviceMust(t, z.BindTrace(tags))
			if !tags.ShouldTrace("rem_bug") {
				serviceMust(t, z.BindTrace(nil))
				serviceMust(t, c.Del(&root))
				reached = true
				return
			}
			armed = true
			z.Budget().FailAfter(after)
			var receipt upolyRootIDInvocationReceipt
			_, body := m.getRootIDWithReceipt(v, r.interval.BorrowLower(), &receipt)
			z.Budget().FailAfter(-1)
			armed = false
			serviceMust(t, z.BindTrace(nil))
			if reached && body == nil {
				t.Fatal("missing actual allocation failure")
			}
			serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
			algebraicComparisonAudit(t, c, &root)
			serviceMust(t, c.Del(&root))
		})
	}
	if !reached {
		t.Fatal("did not reach actual display allocation failure transport")
	}
}
