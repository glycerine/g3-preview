package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestAlgebraicRootTraceDisplayReceiptRejectsLaterInvocation(t *testing.T) {
	c, up, _ := algebraicCompareFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619"))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	writer := &algebraicBinaryDisplayStopWriter{core: c, marker: ":2398 "}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("isolate_roots")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.traceMultivariateIsolationLinearRoot(&a)
	if !tags.ShouldTrace("isolate_roots") {
		if body != nil || writer.ref != 0 {
			t.Fatal("erased root diagnostic", body)
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.config.Rationals.Del(&q))
		return
	}
	if !errors.Is(body, arena.ErrBounds) || writer.ref == 0 || !c.rootTracePending() {
		t.Fatal("actual trace display not retained", body)
	}
	old := c.rootTrace.root
	copied := *c
	copied.self = &copied
	serviceMust(t, copied.cleanupRootDisplayInvocation(&old))
	stale := old
	stale.invocation++
	serviceMust(t, c.cleanupRootDisplayInvocation(&stale))
	if !c.rootDisplayInvocationPending(&old) {
		t.Fatal("copied owner or stale receipt consumed actual display")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	z := c.config.Rationals.Integers()
	z.Store(writer.ref, writer.original)
	// The numerical owner may acknowledge its accepted child directly.
	serviceMust(t, c.cleanupRootDisplayInvocation(&old))
	serviceMust(t, c.DisplayMathematicaInto(&algebraicRootTextSink{}, &a))
	var ref unumeral.NumericCellRef
	var original unumeral.NumericCell
	sink := &upolyDisplaySink{numeralHook: func(string) error {
		frame := up.currentDisplay
		if ref == 0 && frame != nil && frame.constructed && frame.numeral.Cell != 0 {
			ref = frame.numeral.Cell
			original = z.Cell(ref)
			guard := original
			guard.Backing.Requested = uint64(^uint32(0)) + 1
			z.Store(ref, guard)
		}
		return nil
	}}
	location, e := c.locationCleanup(&a)
	serviceMust(t, e)
	var later algebraicRootDisplayInvocationReceipt
	next := c.displayRootAtLocationWithReceipt(sink, location, false, &later)
	if !errors.Is(next, arena.ErrBounds) || ref == 0 || later.invocation == old.invocation || !c.rootDisplayInvocationPending(&later) {
		t.Fatal("later actual display not retained", next)
	}
	frame, coefficients, negative := c.rootDisplay.display.frame, c.rootDisplay.coefficients, c.rootDisplay.negative
	used := c.config.Rationals.Budget().Used()
	serviceMust(t, c.cleanupRootDisplayInvocation(&old))
	serviceMust(t, c.cleanupRootTrace())
	if !c.rootDisplayInvocationPending(&later) || !frame.active || c.rootDisplay.coefficients != coefficients || c.rootDisplay.negative != negative || c.config.Rationals.Budget().Used() != used {
		t.Fatal("old trace receipt acquired replacement display")
	}
	z.Store(ref, original)
	serviceMust(t, c.cleanupRootDisplayInvocation(&later))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
