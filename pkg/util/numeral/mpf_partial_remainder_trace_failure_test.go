// Source: src/util/mpf.cpp:1301; nested component formatter failure cleanup.
package numeral

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatPartialRemainderComponentTraceRefusalCleansChild(t *testing.T) {
	probe := diagnostic.NewTraceTagsWithWriter(&bytes.Buffer{})
	probe.Enable("mpf_dbg_rem")
	if !probe.ShouldTrace("mpf_dbg_rem") {
		return
	}
	found := false
	for ordinal := int64(0); ordinal < 256 && !found; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m, _, budget := mpfNumeralHelpersFixture(t)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("mpf_dbg_rem")
			serviceMust(t, m.BindTrace(tags))
			const hidden = 1 << 23
			x := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), exponent: 40, significand: NewInt(hidden)}
			y := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), exponent: 0, significand: NewInt(hidden + hidden/2)}
			beforeX, beforeY := x, y
			budget.FailAfter(ordinal)
			body := m.PartialRemainder(&x, &y, 0, false)
			budget.FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatalf("unexpected remainder failure: %v", body)
			}
			child := &m.hexfloatComponents
			if body != nil && child.bodyCause != nil {
				found = true
				if x != beforeX || y != beforeY {
					t.Fatal("nested trace refusal changed the caller's operands")
				}
				if child.active || child.executing || child.finishing {
					t.Fatalf("nested formatter retained a failed scope: %+v", *child)
				}
			}
			serviceMust(t, m.CleanupPartialRemainder())
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.Del(&y))
			if e := m.Close(); e != nil {
				z := m.integer
				if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
					t.Fatalf("unexpected manager source stop: %v", e)
				}
				index := int(z.SourceFinalizeStage-2) / 2
				ref := z.Tmp[index]
				if z.SourceFinalizeStage%2 == 1 {
					ref = z.Arg[index]
				}
				if ref != 0 {
					t.Fatalf("unproved source scratch hole: %v", ref)
				}
				serviceMust(t, m.Close())
			}
		})
	}
	if !found {
		t.Fatal("no reached component formatter allocation refusal")
	}
}
