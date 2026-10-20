package polynomial

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAlgebraicUnaryIntervalTraceSourceFrontier(t *testing.T) {
	for _, root := range []bool{false, true} {
		t.Run(map[bool]string{false: "power", true: "root"}[root], func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			c.factor = false
			a, b := algebraicResultantInputs(t, c, up)
			serviceMust(t, c.Del(&b))
			var out Algebraic
			var text strings.Builder
			probed := false
			writer := &algebraicBinaryIntervalProbeWriter{output: &text, probe: func() {
				probed = true
				s := &c.binary
				if !s.running || !s.intervalConstructed || !s.saved[0].owned || s.saved[0].restored || len(s.sequences) != 1 || s.sequences[0] == nil || out != (Algebraic{}) {
					t.Fatal("unary interval trace moved past source target selection")
				}
				for i, action := range []func() error{c.Close, c.CleanupRetained, c.cleanupBinary, func() error { return c.Del(&a) }} {
					if e := action(); !errors.Is(e, ErrAlgebraicPending) {
						t.Fatalf("unary trace callback %d: %v", i, e)
					}
				}
			}}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			if root {
				serviceMust(t, c.Root(&a, 3, &out))
			} else {
				serviceMust(t, c.Power(&a, 3, &out))
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if tags.ShouldTrace("algebraic") {
				interval := "(1, 8)"
				if root {
					interval = "(1, 2)"
				}
				want := "-------- [algebraic] mk_unary src/math/polynomial/algebraic_numbers.cpp:1123 ---------\nr_i: " + interval + "\nlV: 1, uV: 0\na.m_interval: (1, 2)\n------------------------------------------------\n"
				if !probed || strings.Count(text.String(), want) != 1 {
					t.Fatal("literal unary selection diagnostic", text.String())
				}
			} else if probed || text.Len() != 0 {
				t.Fatal("erased unary diagnostic evaluated operands")
			}
			if root {
				algebraicExpectBetween(t, c, &out, 1, 2)
			} else {
				algebraicExpectBetween(t, c, &out, 2, 3)
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
		})
	}
}

func TestAlgebraicUnaryIntervalTraceDisabledDoesNotRead(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceUnaryIntervals(nil, ^uint32(0), ^uint32(0)))
}

func TestAlgebraicUnaryPowerTraceMultiwordReservationPrefixes(t *testing.T) {
	succeeded, refused := false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicLargeDyadicPromotionRoot(t, c)
			var out Algebraic
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Power(&a, 3, &out)
			budget.FailAfter(-1)
			if body == nil {
				succeeded = true
				algebraicExpectBetween(t, c, &out, 1, 2)
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("power trace failed outside an actual reservation", body)
				}
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
			if c.binary.active || c.rootTracePending() || c.coefficientNormalizationPending() {
				t.Fatal("power cleanup retained an accepted child")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !refused {
		t.Fatal("power trace sweep missed refusal or success", succeeded, refused)
	}
}
