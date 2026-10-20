package polynomial

import (
	"errors"
	"io"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBasicRootRationalTraceEveryMultiwordReservation(t *testing.T) {
	n, _ := new(big.Int).SetString("18446744073709551619", 10)
	input := new(big.Int).Mul(n, n).String() + "/9"
	expected := n.String() + "/3"
	succeeded, refused := false, false
	rootRAW, scratchHole := false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetString(&q, input))
			var a, out Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("root_core")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Root(&a, 2, &out)
			budget.FailAfter(-1)
			if body == nil {
				succeeded = true
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("rational root trace failed outside actual reservation", body)
				}
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			if c.basicRootScope.root.Pending() && c.basicRootScope.root.SourceCause() == nil {
				rootRAW = true
				if c.basicRootScope.qConstructed {
					t.Fatal("unjoined integer-root RAW protected a completed older rational destructor")
				}
			}
			serviceMust(t, c.config.Rationals.Integers().CleanupPendingAllocation())
			serviceMust(t, c.CleanupRetained())
			if c.basicRootScope.active || c.rootTracePending() {
				t.Fatal("root trace retained a child after owning acknowledgement")
			}
			var retry Algebraic
			guardedScratch := false
			if body != nil {
				z := c.config.Rationals.Integers()
				before := z.Tmp
				for _, ref := range before {
					guardedScratch = guardedScratch || ref == 0 || !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released
				}
				call := c.Root(&a, 2, &retry)
				if guardedScratch {
					// mpz.h:156-162 destroys old scratch before replacement
					// allocation. A failed lifetime stays unavailable in Go.
					scratchHole = true
					if !errors.Is(call, arena.ErrInvalidSlot) || z.Tmp != before {
						t.Fatal("source-destroyed scratch was repaired or read", call)
					}
					serviceMust(t, c.CleanupRetained())
				} else {
					serviceMust(t, call)
				}
			}
			if !guardedScratch {
				serviceMust(t, c.config.Rationals.SetString(&q, expected))
				result := &out
				if body != nil {
					result = &retry
				}
				cmp, e := c.CompareRational(result, &q)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("independent exact rational square root", cmp)
				}
			}
			serviceMust(t, c.Del(&retry))
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}) {
			return
		}
	}
	if !succeeded || !refused {
		t.Fatal("rational root trace sweep missed refusal or success", succeeded, refused)
	}
	if !rootRAW || !scratchHole {
		t.Fatal("sweep missed actual RAW ownership or destructive source scratch failure", rootRAW, scratchHole)
	}
}

func TestAlgebraicBasicRootTraceExactAndPolynomialBranches(t *testing.T) {
	for _, tc := range []struct {
		input, expected, tag, body, setter string
		degree                             uint32
		line                               string
	}{
		{"16/81", "4/9", "root_core", "r_a_val: 4/9 a_val: 16/81\n", "", 2, "1279"},
		{"-8/27", "-2/3", "root_core", "r_a_val: -2/3 a_val: -8/27\n", "", 3, "1279"},
		{"2", "", "algebraic", "root_core:\nx^2 - 2\n", "a: (#^2 - 2, 2)\n", 2, "1320"},
		{"-2", "", "algebraic", "root_core:\nx^3 + 2\n", "a: (#^3 + 2, 1)\n", 3, "1320"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			for _, fail := range []bool{false, true} {
				c, _, _ := algebraicControlledFixture(t)
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&q, tc.input))
				var a, out Algebraic
				serviceMust(t, c.SetRational(&a, &q))
				var text strings.Builder
				broken := &linearTraceFailWriter{}
				tags := diagnostic.NewTraceTagsWithWriter(&text)
				if fail {
					tags = diagnostic.NewTraceTagsWithWriter(broken)
				}
				tags.Enable(tc.tag)
				serviceMust(t, c.config.Rationals.BindTrace(tags))
				serviceMust(t, c.Root(&a, tc.degree, &out))
				if tags.ShouldTrace(tc.tag) {
					want := "-------- [" + tc.tag + "] root_core src/math/polynomial/algebraic_numbers.cpp:" + tc.line + " ---------\n" + tc.body + "------------------------------------------------\n"
					if tc.setter != "" {
						// cpp:1323 calls the polynomial setter after the root_core record.
						want += "-------- [algebraic] set src/math/polynomial/algebraic_numbers.cpp:506 ---------\n" + tc.setter + "------------------------------------------------\n"
					}
					if fail {
						if broken.calls != 1 {
							t.Fatal("failed root stream retried output", broken.calls)
						}
					} else if text.String() != want {
						t.Fatal("literal root_core diagnostic", text.String(), want)
					}
				} else if text.Len() != 0 || broken.calls != 0 {
					t.Fatal("erased root_core diagnostic emitted output")
				}
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				if tc.expected != "" {
					serviceMust(t, c.config.Rationals.SetString(&q, tc.expected))
					cmp, e := c.CompareRational(&out, &q)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("independent rational root", cmp)
					}
				} else if tc.input == "2" {
					algebraicExpectBetween(t, c, &out, 1, 2)
				} else {
					algebraicExpectBetween(t, c, &out, -2, -1)
				}
				serviceMust(t, c.Del(&out))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
			}
		})
	}
}

func TestAlgebraicBasicRootTraceDisabledDoesNotRead(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	serviceMust(t, c.traceBasicRootRational(nil))
	serviceMust(t, c.traceBasicRootPolynomial(UPolynomialView{count: ^uint32(0)}))
}

func TestAlgebraicBasicRootTraceCallbackBeforePublication(t *testing.T) {
	for _, exact := range []bool{false, true} {
		c, _, _ := algebraicControlledFixture(t)
		q := unumeral.NewRational()
		input, tag := "2", "algebraic"
		if exact {
			input, tag = "16/81", "root_core"
		}
		serviceMust(t, c.config.Rationals.SetString(&q, input))
		var a, out Algebraic
		serviceMust(t, c.SetRational(&a, &q))
		called := false
		writer := &linearOperationProbeWriter{probe: func() {
			called = true
			s := &c.basicRootScope
			if !s.active || !s.running || out != (Algebraic{}) || (s.p.slot == 0) != exact || (!exact && s.constructed != 2) {
				t.Fatal("root diagnostic moved beyond its lexical/publication frontier")
			}
			for i, action := range []func() error{c.Close, c.CleanupRetained, c.CleanupIsolation, func() error { return c.Del(&a) }} {
				if e := action(); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatalf("root callback %d: %v", i, e)
				}
			}
		}}
		tags := diagnostic.NewTraceTagsWithWriter(writer)
		tags.Enable(tag)
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		serviceMust(t, c.Root(&a, 2, &out))
		if called != tags.ShouldTrace(tag) || c.basicRootScope.active || c.rootTracePending() {
			t.Fatal("root diagnostic lifetime", called)
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.config.Rationals.Del(&q))
	}
}

func TestAlgebraicBasicRootPolynomialTraceOwnsStoppedDisplay(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619"))
	var a, out Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	writer := &algebraicBinaryDisplayStopWriter{core: c, marker: "root_core:\n"}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.Root(&a, 2, &out)
	var frame *upolyDisplayScope
	var stop error
	if tags.ShouldTrace("algebraic") {
		frame = c.rootTrace.univariate.frame
		if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || frame == nil || !frame.active || !c.basicRootScope.active || c.basicRootScope.constructed != 2 || c.basicRootScope.p.slot == 0 || out != (Algebraic{}) {
			t.Fatal("stopped polynomial display consumed older root locals", body, writer.ref)
		}
		stop = frame.sourceCause
		if stop == nil || c.basicRootScope.sourceStop != stop {
			t.Fatal("root parent lost actual display source stop", body)
		}
		c.config.Rationals.Integers().Store(writer.ref, writer.original)
	} else {
		serviceMust(t, body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() || c.basicRootScope.active || frame != nil && (frame.active || frame.sourceCause != stop || c.basicRootScope.sourceStop != stop || !errors.Is(body, stop)) {
		t.Fatal("root owning acknowledgement changed source history")
	}
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
