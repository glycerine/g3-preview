package polynomial

// Source: algebraic_numbers.cpp:1630,1641,1644.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicInverseTraceRetainsActualDisplayAndOlderRationals(t *testing.T) {
	for _, marker := range []string{":1630 ", ":1644 "} {
		t.Run(marker, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicLargeDyadicPromotionRoot(t, c)
			writer := &algebraicBinaryDisplayStopWriter{core: c, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("algebraic_bug")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.Inv(&a)
			var stop error
			if tags.ShouldTrace("algebraic_bug") {
				if !errors.Is(body, arena.ErrBounds) || writer.ref == 0 || !c.rootTracePending() || !c.rootDisplayInvocationPending(&c.inversion.traceDisplay) || !c.inversion.traceDisplayPending {
					t.Fatal("inverse trace lost actual display invocation", body)
				}
				stop = c.rootDisplay.sourceStop
				remaining := uint8(0)
				if marker == ":1644 " {
					remaining = 2
				}
				if stop == nil || c.inversion.sourceStop != stop || c.inversion.remaining != remaining || c.inversion.sourceCursor != 0 {
					t.Fatal("display stop crossed older scoped rational frontier", body, c.inversion.remaining, c.inversion.sourceCursor)
				}
				c.config.Rationals.Integers().Store(writer.ref, writer.original)
			} else {
				serviceMust(t, body)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootTracePending() || c.rootDisplay.active || c.inversion.traceDisplayPending || c.inversion.remaining != 0 {
				t.Fatal("owning inverse trace cleanup retained child")
			}
			if stop != nil && c.inversion.sourceStop != stop {
				t.Fatal("owning cleanup rewrote original source history")
			}
			serviceMust(t, c.Del(&a))
		})
	}
}

func TestAlgebraicInverseDisplayReceiptDoesNotAcquireReplacement(t *testing.T) {
	c, up, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	writer := &algebraicBinaryDisplayStopWriter{core: c, marker: ":1630 "}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.Inv(&a)
	if !tags.ShouldTrace("algebraic_bug") {
		serviceMust(t, body)
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&a))
		return
	}
	if !errors.Is(body, arena.ErrBounds) || writer.ref == 0 || !c.rootDisplayInvocationPending(&c.inversion.traceDisplay) {
		t.Fatal("actual stopped inverse display", body)
	}
	old := c.inversion.traceDisplay
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	z := c.config.Rationals.Integers()
	z.Store(writer.ref, writer.original)
	serviceMust(t, c.cleanupRootDisplayInvocation(&old))
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
		t.Fatal("replacement root display not reached", next)
	}
	frame := c.rootDisplay.display.frame
	used := z.Budget().Used()
	serviceMust(t, c.cleanupInverse())
	serviceMust(t, c.cleanupRootTrace())
	if !c.rootDisplayInvocationPending(&later) || !frame.active || used != z.Budget().Used() {
		t.Fatal("old inverse owner acquired replacement display")
	}
	z.Store(ref, original)
	serviceMust(t, c.cleanupRootDisplayInvocation(&later))
	serviceMust(t, c.Del(&a))
}

type inverseTraceBorrowProbe struct {
	core  *AlgebraicRootCore
	t     *testing.T
	calls int
}

func (w *inverseTraceBorrowProbe) Write(p []byte) (int, error) {
	w.calls++
	c := w.core
	up, z := c.config.Polynomials, c.config.Rationals.Integers()
	if !c.rootTrace.running || up.factorTrace != &c.rootTrace.borrow {
		w.t.Error("inverse stream callback lacks actual numeric borrow")
		return len(p), nil
	}
	used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
	for _, call := range []func() error{z.SourceFinalize, z.CloseChecked, c.config.Rationals.SourceFinalize, c.config.Rationals.CloseChecked} {
		if e := call(); !errors.Is(e, ErrPolynomialPending) {
			w.t.Error("inverse callback acquired numeric owner", e)
		}
	}
	q, e := up.NewPolynomial()
	if !errors.Is(e, ErrUPolynomialCleanupPending) || q.slot != 0 {
		w.t.Error("inverse callback constructed UP local", e)
	}
	if e := up.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		w.t.Error("inverse callback closed UP owner", e)
	}
	if used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
		w.t.Error("refused inverse callback changed actual allocation graph")
	}
	return len(p), nil
}
func TestAlgebraicInverseTraceBorrowCoversEntireStream(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicBigBinaryBoundsRoot(t, c)
	w := &inverseTraceBorrowProbe{core: c, t: t}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Inv(&a))
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if (w.calls != 0) != tags.ShouldTrace("algebraic_bug") || c.rootTrace.running || c.rootTracePending() {
		t.Fatal("inverse stream borrow lifetime", w.calls)
	}
	serviceMust(t, c.Del(&a))
}
