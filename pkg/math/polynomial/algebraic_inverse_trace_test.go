package polynomial

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAlgebraicInverseSourceTracesAndCachedIndex(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, _, _ := algebraicControlledFixture(t)
		a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
		var output strings.Builder
		broken := &linearTraceFailWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if fail {
			tags = diagnostic.NewTraceTagsWithWriter(broken)
		}
		tags.Enable("algebraic_bug")
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		serviceMust(t, c.Inv(&a))
		location, e := c.location(&a)
		serviceMust(t, e)
		flags, e := c.config.Allocator.ReadUint32(location.raw, 16)
		serviceMust(t, e)
		if tags.ShouldTrace("algebraic_bug") {
			if flags>>3 != 2 {
				t.Fatal("inversion overwrote display's actual cached root index", flags>>3)
			}
			if fail {
				if broken.calls != 1 {
					t.Fatal("sticky trace failure retried output", broken.calls)
				}
			} else {
				want := "-------- [algebraic_bug] inv src/math/polynomial/algebraic_numbers.cpp:1630 ---------\nbefore inv: (#^2 - 2, 2)\n(1, 2)\n------------------------------------------------\n" +
					"-------- [algebraic_bug] inv src/math/polynomial/algebraic_numbers.cpp:1641 ---------\ninv new_bounds: 1/2, 1\n------------------------------------------------\n" +
					"-------- [algebraic_bug] inv src/math/polynomial/algebraic_numbers.cpp:1644 ---------\nafter inv: (-2 #^2 + 1, 2)\n(1/2, 1)\n------------------------------------------------\n"
				if output.String() != want {
					t.Fatalf("literal inversion trace: %q", output.String())
				}
			}
		} else if output.Len() != 0 || broken.calls != 0 || flags>>3 != 0 {
			t.Fatal("erased trace formatted or cached root index", output.String(), flags>>3)
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		algebraicExpectBetween(t, c, &a, 0, 1)
		serviceMust(t, c.Del(&a))
	}
}

func TestAlgebraicInverseTraceCallbackKeepsActualCell(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	output := &linearOperationProbeWriter{probe: func() {
		before := a
		for i, action := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&a) }, func() error { return c.Inv(&a) }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("inverse trace callback action %d: %v", i, e)
			}
		}
		if a != before || c.closed || c.closing {
			t.Fatal("trace callback changed inversion receiver")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Inv(&a))
	if tags.ShouldTrace("algebraic_bug") != output.called || c.inversion.executing || c.inversion.remaining != 0 {
		t.Fatal("trace callback or completed inverse lifetime", output.called, c.inversion)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicInverseTraceEveryReservationPrefix(t *testing.T) {
	succeeded, refused := false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicBigBinaryBoundsRoot(t, c)
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("algebraic_bug")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Inv(&a)
			budget.FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("inversion trace failed outside an actual reservation", body)
				}
			} else {
				succeeded = true
				algebraicExpectBetween(t, c, &a, 0, 1)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
			if c.inversion.remaining != 0 || c.inversion.traceDisplayPending || c.rootDisplay.active {
				t.Fatal("trace child or older inverse locals retained after owning cleanup")
			}
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !refused {
		t.Fatal("inversion trace reservation frontier did not reach both refusal and success", succeeded, refused)
	}
}

func TestAlgebraicInverseTraceBigBoundsSuccess(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicBigBinaryBoundsRoot(t, c)
	var trace strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("algebraic_bug")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Inv(&a))
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	algebraicExpectBetween(t, c, &a, 0, 1)
	serviceMust(t, c.CleanupRetained())
	algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
	serviceMust(t, c.Del(&a))
}
