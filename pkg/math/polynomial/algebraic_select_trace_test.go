package polynomial

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicSelectTraceLiteralAndPreLocalBorrow(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, true)
	var result Algebraic
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&result))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}()
	var output strings.Builder
	calls := 0
	writer := &linearOperationProbeWriter{probe: func() {
		calls++
		if c.selection.active {
			t.Fatal("selection local constructed before entry trace")
		}
		for _, action := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&a) }, func() error { return c.Select(&a, &b, &result) }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("trace borrower", e)
			}
		}
	}}
	// Capture all chunks while checking the existing callback borrower.
	tags := diagnostic.NewTraceTagsWithWriter(&selectTraceWriter{output: &output, probe: writer})
	tags.Enable("algebraic_select")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Select(&a, &b, &result))
	if tags.ShouldTrace("algebraic_select") {
		want := "-------- [algebraic_select] select src/math/polynomial/algebraic_numbers.cpp:2523 ---------\nprev: (1, 2)\ncurr: (1, 2)\n------------------------------------------------\n"
		if output.String() != want || calls == 0 {
			t.Fatalf("entry record %q callbacks %d", output.String(), calls)
		}
	} else if output.Len() != 0 || calls != 0 {
		t.Fatal("erased trace executed")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	cmp, e := c.Compare(&a, &result)
	serviceMust(t, e)
	if cmp >= 0 {
		t.Fatal("lower sample", cmp)
	}
	cmp, e = c.Compare(&result, &b)
	serviceMust(t, e)
	if cmp >= 0 {
		t.Fatal("upper sample", cmp)
	}
}

type selectTraceWriter struct {
	output *strings.Builder
	probe  *linearOperationProbeWriter
}

func (w *selectTraceWriter) Write(p []byte) (int, error) {
	_, _ = w.probe.Write(p)
	return w.output.Write(p)
}

func TestAlgebraicSelectTraceRefusingWriterStillFormats(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	var a, b, out Algebraic
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}()
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&a, &one))
	two := unumeral.NewInt(2)
	serviceMust(t, c.SetInteger(&b, &two))
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic_select")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.Select(&a, &b, &out))
	if tags.ShouldTrace("algebraic_select") && w.calls != 1 {
		t.Fatal("sticky writer", w.calls)
	}
	if c.rootTracePending() || c.selection.active {
		t.Fatal("completed source owner retained")
	}
}

func TestAlgebraicSelectTraceEveryReservation(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	success, refused, beforeLocal := false, false, false
	for prefix := int64(0); prefix < 1024 && !success; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicBigBinaryBoundsRoot(t, c)
			var b, out Algebraic
			three := unumeral.NewInt(3)
			serviceMust(t, c.SetInteger(&b, &three))
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("algebraic_select")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Select(&a, &b, &out)
			budget.FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("non-reservation error", body)
				}
				if !c.selection.active && out == (Algebraic{}) {
					beforeLocal = true
				}
			} else {
				success = true
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootTracePending() || c.selection.active {
				t.Fatal("accepted child not acknowledged")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !success || !refused || !beforeLocal {
		t.Fatal("missing trace refusal or success", success, refused, beforeLocal)
	}
}

func TestAlgebraicSelectTraceBasicAndRootIntervals(t *testing.T) {
	for shape := 0; shape < 4; shape++ {
		t.Run(strconv.Itoa(shape), func(t *testing.T) {
			c, _, _ := algebraicControlledFixture(t)
			a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, true)
			defer func() {
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
			}()
			lower, upper := "(1, 2)", "(1, 2)"
			if shape&1 != 0 {
				one := unumeral.NewInt(1)
				serviceMust(t, c.SetInteger(&a, &one))
				lower = "[1, 1]"
			}
			if shape&2 != 0 {
				two := unumeral.NewInt(2)
				serviceMust(t, c.SetInteger(&b, &two))
				upper = "[2, 2]"
			}
			var output strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("algebraic_select")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			serviceMust(t, c.Select(&a, &b, &a))
			if tags.ShouldTrace("algebraic_select") && !strings.Contains(output.String(), "prev: "+lower+"\ncurr: "+upper+"\n") {
				t.Fatal("original intervals before separation/alias assignment", output.String())
			}
		})
	}
}
