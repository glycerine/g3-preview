// Source: algebraic_numbers.cpp:290,2999-3001; actual display NUM destructor.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type boundPredicateDisplayReleaseWriter struct {
	t       *testing.T
	core    *AlgebraicRootCore
	runtime *umemory.Runtime
	lane    umemory.Block
	calls   int
}

func (w *boundPredicateDisplayReleaseWriter) Write(p []byte) (int, error) {
	frame := w.core.config.Polynomials.currentDisplay
	if w.calls == 0 && frame != nil && frame.constructed && frame.numeral.Cell != 0 {
		w.calls++
		// The concrete source display NUM exists and its text has reached this
		// writer. Public Runtime operations arm its upcoming physical release.
		w.lane = algebraicCellDeletionTLSLane(w.t, w.runtime)
	}
	return len(p), nil
}

func TestAlgebraicBoundPredicateEntryDisplaySourceStop(t *testing.T) {
	w := &boundPredicateDisplayReleaseWriter{t: t}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic_bug")
	if !polynomialSourceDebug || !tags.ShouldTrace("algebraic_bug") {
		t.Skip("predicate entry TRACE compiled out")
	}
	c, _, rt := algebraicNegTLSFixture(t, nil)
	w.core, w.runtime = c, rt
	root := algebraicLargeDyadicPromotionRoot(t, c)
	original := root
	out := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.GetUpperRational(&root, &out)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	defer func() {
		contextTLSFinish(t, rt, c.CleanupRetained)
		contextTLSFinish(t, rt, func() error { return c.config.Rationals.Del(&out) })
		contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
		algebraicCellDeletionFinishLane(t, rt, &w.lane)
	}()
	if w.calls != 1 || !errors.Is(body, arena.ErrBounds) || c.rootDisplay.sourceStop == nil || !c.rootDisplay.active || c.predicate.active {
		t.Fatal("actual phase-zero display source stop not reached", w.calls, body)
	}
	if root != original || c.bounds.remaining != 1 || !errors.Is(c.bounds.sourceStop, c.rootDisplay.sourceStop) {
		t.Error("entry display source stop crossed older BIN", c.bounds.remaining, c.bounds.sourceStop)
	}
	if !c.predicateTrace.active {
		t.Error("predicate TRACE lost its actual stopped display")
	}
	before, stats, used := c.bounds.values, rt.Stats(), c.config.Rationals.Budget().Used()
	if e := c.finishBounds(false); !errors.Is(e, c.rootDisplay.sourceStop) {
		t.Error("source replay lost entry display stop", e)
	}
	if c.bounds.values != before || rt.Stats() != stats || c.config.Rationals.Budget().Used() != used {
		t.Error("source replay changed physical state")
	}
	contextTLSFinish(t, rt, c.CleanupRetained)
	if c.bounds.remaining != 0 || c.rootDisplay.active || c.predicateTrace.active {
		t.Error("owning cleanup retained accepted entry display")
	}
}

func TestAlgebraicBoundPredicateTraceDoesNotAcquireReplacementDisplay(t *testing.T) {
	w := &boundPredicateDisplayReleaseWriter{t: t}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic_bug")
	if !polynomialSourceDebug || !tags.ShouldTrace("algebraic_bug") {
		t.Skip("predicate entry TRACE compiled out")
	}
	c, _, rt := algebraicNegTLSFixture(t, nil)
	w.core, w.runtime = c, rt
	root := algebraicLargeDyadicPromotionRoot(t, c)
	out := unumeral.NewRational()
	later := &boundPredicateDisplayReleaseWriter{t: t, core: c, runtime: rt}
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		contextTLSFinish(t, rt, c.CleanupRetained)
		contextTLSFinish(t, rt, func() error { return c.config.Rationals.Del(&out) })
		contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
		algebraicCellDeletionFinishLane(t, rt, &w.lane)
		algebraicCellDeletionFinishLane(t, rt, &later.lane)
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.GetUpperRational(&root, &out)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if !errors.Is(body, arena.ErrBounds) || w.calls != 1 || !c.predicateTrace.active || !c.rootDisplay.active {
		t.Fatal("actual entry display not retained", body)
	}
	old := c.rootDisplay.invocation
	// The display's owner independently acknowledges the genuine stopped child.
	// Its older TRACE receipt is deliberately left pending until the next call.
	contextTLSFinish(t, rt, c.cleanupRootDisplay)
	// Retain and enlarge the live fixture lane through public Runtime accounting
	// so another independent release frontier can be reached. No source value
	// or consumed NUM descriptor is restored.
	request := uint64(8 - rt.State().LocalSize + 4000000)
	serviceMust(t, rt.Reallocate(&w.lane, request))
	l, e := c.locationCleanup(&root)
	serviceMust(t, e)
	var replacement algebraicRootDisplayInvocationReceipt
	body = c.displayRootAtLocationWithReceipt(algebraicPredicateTraceStream{later}, l, false, &replacement)
	if !errors.Is(body, arena.ErrBounds) || later.calls != 1 || replacement.invocation == old || !c.rootDisplayInvocationPending(&replacement) {
		t.Fatal("actual replacement display not reached", body)
	}
	frame := c.rootDisplay.display.frame
	stats, used := rt.Stats(), c.config.Rationals.Budget().Used()
	serviceMust(t, c.cleanupPredicateTrace())
	if !c.rootDisplayInvocationPending(&replacement) || !frame.active || stats != rt.Stats() || used != c.config.Rationals.Budget().Used() {
		t.Fatal("older predicate trace acquired replacement display")
	}
	contextTLSFinish(t, rt, c.cleanupRootDisplay)
}
