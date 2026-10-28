package polynomial

// Source: algebraic_numbers.cpp:1630,1644,1663-1666.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicDivisionInverseDisplayStopProtectsScopedInverse(t *testing.T) {
	for _, marker := range []string{":1630 ", ":1644 "} {
		t.Run(marker, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			b := algebraicLargeDyadicPromotionRoot(t, c)
			var zero, out Algebraic
			w := &algebraicBinaryDisplayStopWriter{core: c, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("algebraic_bug")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.Div(&zero, &b, &out)
			var stop error
			if tags.ShouldTrace("algebraic_bug") {
				stop = c.rootDisplay.sourceStop
				if !errors.Is(body, arena.ErrBounds) || stop == nil || w.ref == 0 || !c.division.active || c.division.inverse.slot == 0 || c.division.sourceStop != stop || c.inversion.sourceStop != stop || out.slot != 0 {
					t.Fatal("division lost actual nested source stop or scoped inverse", body, c.division.sourceStop, stop)
				}
				l, e := c.locationCleanup(&c.division.inverse)
				serviceMust(t, e)
				if l.stage != 0 || l.coefficientCursor != 0 {
					t.Fatal("older scoped inverse destruction started before younger display ended")
				}
				c.config.Rationals.Integers().Store(w.ref, w.original)
			} else {
				serviceMust(t, body)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.division.active || c.division.inverse.slot != 0 || c.rootTracePending() || c.inversion.remaining != 0 {
				t.Fatal("division acknowledgement retained inverse or diagnostic")
			}
			if stop != nil && c.division.sourceStop != stop {
				t.Fatal("owning cleanup overwrote original division source history")
			}
			serviceMust(t, c.Div(&zero, &b, &out))
			if c.division.sourceStop != nil || c.division.active {
				t.Fatal("new accepted division inherited old source stop")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
		})
	}
}

type divisionInverseTraceBudgetWriter struct {
	strings.Builder
	core   *AlgebraicRootCore
	prefix int64
	armed  bool
}

func (w *divisionInverseTraceBudgetWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *divisionInverseTraceBudgetWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if !w.armed && strings.Contains(w.String(), ":1630 ") {
		w.armed = true
		w.core.config.Rationals.Budget().FailAfter(w.prefix)
	}
	if w.armed && strings.Contains(w.String(), "------------------------------------------------\n") {
		w.core.config.Rationals.Budget().FailAfter(-1)
	}
	return n, e
}
func TestAlgebraicDivisionInverseDisplayRAWDoesNotStopOlderSourceDestructor(t *testing.T) {
	reached := false
	for prefix := int64(0); prefix < 512 && !reached; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			b := algebraicBigBinaryBoundsRoot(t, c)
			var zero, out Algebraic
			w := &divisionInverseTraceBudgetWriter{core: c, prefix: prefix}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("algebraic_bug")
			if !tags.ShouldTrace("algebraic_bug") {
				serviceMust(t, c.Del(&b))
				reached = true
				return
			}
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			body := c.Div(&zero, &b, &out)
			c.config.Rationals.Budget().FailAfter(-1)
			if !w.armed {
				t.Fatal("actual inversion diagnostic not reached", body)
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("diagnostic did not stop at actual numeric reservation", body)
			}
			if c.rootTracePending() && c.rootTraceSourceCause() == nil {
				reached = true
				if body == nil || c.division.active || c.division.inverse.slot != 0 || c.division.sourceStop != nil {
					t.Fatal("unjoined formatting RAW stopped completed older scoped inverse destructor", body, c.division.sourceStop)
				}
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
		}) {
			return
		}
	}
	if !reached {
		t.Fatal("no actual inversion display RAW suffix retained")
	}
}

type divisionInverseDestructorStopWriter struct {
	strings.Builder
	core     *AlgebraicRootCore
	final    bool
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
}

func (w *divisionInverseDestructorStopWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *divisionInverseDestructorStopWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if strings.Contains(w.String(), ":1644 ") {
		w.final = true
	}
	if w.final && w.ref == 0 && strings.HasSuffix(w.String(), "------------------------------------------------\n") {
		l, e := w.core.locationCleanup(&w.core.division.inverse)
		if e != nil {
			return n, e
		}
		r, e := w.core.load(l)
		if e != nil {
			return n, e
		}
		w.ref = r.interval.BorrowUpper().BorrowNumerator().Cell
		if w.ref != 0 {
			z := w.core.config.Rationals.Integers()
			w.original = z.Cell(w.ref)
			guard := w.original
			guard.Backing.Requested = uint64(^uint32(0)) + 1
			z.Store(w.ref, guard)
		}
	}
	return n, e
}
func TestAlgebraicDivisionScopedInverseDestructorPreservesCursor(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	b := algebraicBigBinaryBoundsRoot(t, c)
	var zero, out Algebraic
	w := &divisionInverseDestructorStopWriter{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.Div(&zero, &b, &out)
	var stop error
	if tags.ShouldTrace("algebraic_bug") {
		stop = c.division.sourceStop
		if !errors.Is(body, arena.ErrBounds) || w.ref == 0 || stop == nil || !c.division.active || c.division.cleaning || c.division.inverse.slot == 0 || c.inversion.sourceStop != nil {
			t.Fatal("actual private inverse destructor not retained", body, stop)
		}
		l, e := c.locationCleanup(&c.division.inverse)
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		if l.stage != 2 || r.poly.slot != 0 || l.coefficientInitialized != 0 || r.interval.BorrowLower().BorrowNumerator().Cell != 0 || r.interval.BorrowUpper().BorrowNumerator().Cell != w.ref {
			t.Fatal("private inverse source destructor cursor lost", l.stage)
		}
		c.config.Rationals.Integers().Store(w.ref, w.original)
	} else {
		serviceMust(t, body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.division.active || c.division.cleaning || c.division.inverse.slot != 0 || c.division.sourceStop != stop {
		t.Fatal("owning inverse suffix cleanup changed source history")
	}
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&b))
}
