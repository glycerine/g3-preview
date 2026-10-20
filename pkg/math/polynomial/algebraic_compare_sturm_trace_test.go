package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"strings"
	"testing"
)

func TestAlgebraicCompareSturmTraceLiteralAndLiveSequence(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	c.minMagnitude = 1
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, false)
	var output strings.Builder
	observed := false
	w := &sturmComparisonTraceWriter{output: &output, probe: func() {
		if c.compareSequence == nil {
			t.Fatal("Sturm trace followed sequence destruction")
		}
		if c.compareSequence.closed || len(c.compareSequence.state.sizes) == 0 {
			t.Fatal("empty actual sequence")
		}
		for _, action := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&a) }, func() error { _, e := c.Compare(&a, &b); return e }} {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("Sturm executing borrow", e)
			}
		}
		observed = true
	}}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	for _, reverse := range []bool{false, true} {
		output.Reset()
		w.seen = false
		left, right := &a, &b
		want := -1
		if reverse {
			left, right = &b, &a
			want = 1
		}
		got, e := c.Compare(left, right)
		serviceMust(t, e)
		if got != want {
			t.Fatal("order", got, want)
		}
		if tags.ShouldTrace("algebraic") {
			text := "comparing using sturm\n(1, 2)\n(1, 2)\nV: -1, sign_lower(a): -1, sign_lower(b): -1\n"
			if reverse {
				text = strings.Replace(text, "V: -1", "V: 1", 1)
			}
			if !observed || !strings.Contains(output.String(), text) || strings.Count(output.String(), ":1817 ") != 1 || !strings.Contains(output.String(), "compare_core src/math/polynomial/algebraic_numbers.cpp:1817 ") {
				t.Fatal("literal Sturm record", output.String())
			}
		} else if output.Len() != 0 || observed {
			t.Fatal("erased diagnostic executed")
		}
		if c.compareSequence != nil {
			t.Fatal("sequence not destroyed after trace")
		}
	}
}

type sturmComparisonTraceWriter struct {
	output *strings.Builder
	probe  func()
	seen   bool
}

func (w *sturmComparisonTraceWriter) Write(p []byte) (int, error) {
	n, e := w.output.Write(p)
	if !w.seen && strings.Contains(w.output.String(), ":1817 ") {
		w.seen = true
		w.probe()
	}
	return n, e
}

func TestAlgebraicCompareSturmTraceZeroVariationEarlyReturn(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	c.minMagnitude = 1
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	b := algebraicCompareRoot(t, c, []int32{-4, 0, 0, 0, 1}, 1, 0, 2, 0, false)
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("algebraic")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != 0 || c.compareSturm != 1 {
		t.Fatal("independent equal polynomial roots", got, c.compareSturm)
	}
	if tags.ShouldTrace("algebraic") && !strings.Contains(output.String(), "V: 0, sign_lower(a): -1, sign_lower(b): -1\n") {
		t.Fatal("V==0 bypassed record", output.String())
	}
}

type sturmCharacterReadGuard struct {
	comparisonCharacterReadGuard
	sturm bool
}

func (w *sturmCharacterReadGuard) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *sturmCharacterReadGuard) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if strings.Contains(w.String(), ":1817 ") {
		w.sturm = true
	}
	r := &w.core.rootTrace.insertions[0]
	// Match only the actual Sturm record's completed bound formatter, rather
	// than an earlier outer/inner comparison diagnostic using the same tag.
	if w.sturm && !w.guarded && r.Active() && r.Character != nil && r.Character.Completed && strings.LastIndex(w.String(), "src/util/mpn.cpp:395 ") > strings.LastIndex(w.String(), ":1817 ") {
		w.guarded = true
		r.Character.Buffer = r.Character.Buffer[:0]
	}
	return n, e
}
func TestAlgebraicCompareSturmTraceActualBoundCharacterGuard(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	c, _, _ := algebraicCompareFixture(t)
	c.minMagnitude = 1
	a := algebraicBigBinaryBoundsRootWithMinimal(t, c, false)
	b := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, false)
	w := &sturmCharacterReadGuard{comparisonCharacterReadGuard: comparisonCharacterReadGuard{core: c}}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	tags.Enable("mpn_to_string")
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}()
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	_, body := c.Compare(&a, &b)
	if !w.sturm || !w.guarded || !errors.Is(body, arena.ErrBounds) || !c.rootTracePending() {
		t.Fatal("actual Sturm bound formatter not retained", body, w.sturm, w.guarded)
	}
	if c.compareSequence != nil || c.rootTraceSourceCause() != nil {
		t.Fatal("body read guard fabricated source destructor stop")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	got, e := c.Compare(&a, &b)
	serviceMust(t, e)
	if got != -1 {
		t.Fatal("guard changed original inputs", got)
	}
}

func TestAlgebraicCompareSturmTracePositiveLowerSignsAndRefusingStream(t *testing.T) {
	for _, refused := range []bool{false, true} {
		c, _, _ := algebraicCompareFixture(t)
		c.minMagnitude = 1
		a := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, -2, 0, -1, 0, false)
		b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, -2, 0, -1, 0, false)
		var output strings.Builder
		broken := &linearTraceFailWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if refused {
			tags = diagnostic.NewTraceTagsWithWriter(broken)
		}
		tags.Enable("algebraic")
		serviceMust(t, c.config.Rationals.BindTrace(tags))
		got, e := c.Compare(&a, &b)
		serviceMust(t, e)
		if got != -1 || c.compareSturm != 1 {
			t.Fatal("negative roots comparison", got, c.compareSturm)
		}
		if tags.ShouldTrace("algebraic") {
			if refused {
				if broken.calls != 1 {
					t.Fatal("sticky stream repeated writer", broken.calls)
				}
			} else if !strings.Contains(output.String(), "comparing using sturm\n(-2, -1)\n(-2, -1)\nV: 1, sign_lower(a): 1, sign_lower(b): 1\n") {
				t.Fatal("positive source lower signs", output.String())
			}
		}
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
	}
}
