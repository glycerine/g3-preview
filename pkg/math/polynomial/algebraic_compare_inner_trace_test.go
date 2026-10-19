// Source: algebraic_numbers.cpp:1738-1743,1861,2708-2719.
package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCompareInnerTraceExactRecordsAndEarlyExits(t *testing.T) {
	for _, kind := range []string{"minimal", "nonminimal", "equal", "disjoint"} {
		t.Run(kind, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			minimal := kind != "nonminimal"
			a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, minimal)
			var b Algebraic
			switch kind {
			case "equal":
				serviceMust(t, c.Copy(&b, &a))
			case "disjoint":
				b = algebraicCompareRoot(t, c, []int32{-5, 0, 1}, 2, 0, 3, 0, minimal)
			default:
				b = algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, minimal)
			}
			var text strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			order, e := c.Compare(&a, &b)
			serviceMust(t, e)
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			want := -1
			if kind == "equal" {
				want = 0
			}
			if order != want {
				t.Fatal("comparison result", order, want)
			}
			inner := strings.Contains(text.String(), ":1738 ")
			expected := tags.ShouldTrace("algebraic") && (kind == "minimal" || kind == "nonminimal")
			if inner != expected {
				t.Fatal("inner trace crossed source early return", kind, text.String())
			}
			if expected {
				flag := "1"
				if !minimal {
					flag = "0"
				}
				record := "comparing\na: x^2 - 2\n(1, 2)\ncell_a->m_minimal: " + flag + "\nb: x^2 - 3\n(1, 2)\ncell_b->m_minimal: " + flag + "\n"
				if !strings.Contains(text.String(), record) {
					t.Fatal("inner comparison literal", text.String(), record)
				}
			}
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		})
	}
}

func TestAlgebraicCompareInnerDisplayStopPrecedesRefinement(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, false)
	w := &algebraicBinaryDisplayStopWriter{core: c, marker: ":1738 "}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	_, body := c.Compare(&a, &b)
	if tags.ShouldTrace("algebraic") {
		if !errors.Is(body, arena.ErrBounds) || w.ref == 0 || !c.rootTracePending() || c.compareRefine != 0 || c.compareSturm != 0 || c.comparePoly != 0 {
			t.Fatal("inner formatter stop crossed refinement/algorithm frontier", body)
		}
		if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
			t.Fatal("stopped comparison admitted input deletion", e)
		}
		c.config.Rationals.Integers().Store(w.ref, w.original)
	} else {
		serviceMust(t, body)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() {
		t.Fatal("owning comparison display acknowledgement pending")
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
}

type comparisonCharacterReadGuard struct {
	strings.Builder
	core    *AlgebraicRootCore
	guarded bool
}

func (w *comparisonCharacterReadGuard) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *comparisonCharacterReadGuard) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	r := &w.core.rootTrace.insertions[0]
	if !w.guarded && strings.Contains(w.String(), "src/util/mpn.cpp:395 ") && r.Active() && r.Character != nil && r.Character.Completed {
		// The actual formatter is complete, but its enclosing C-string insertion
		// has not read the produced character span yet. Guard that reached read.
		w.guarded = true
		r.Character.Buffer = r.Character.Buffer[:0]
	}
	return n, e
}
func TestAlgebraicCompareBasicTraceOwnsActualCharacterReadFailure(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/7"))
	var zero, a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	w := &comparisonCharacterReadGuard{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	tags.Enable("mpn_to_string")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	order, body := c.Compare(&zero, &a)
	if tags.ShouldTrace("algebraic") {
		r := &c.rootTrace.insertions[0]
		if !w.guarded || !errors.Is(body, arena.ErrBounds) || !c.rootTracePending() || !r.Active() || r.SourceCause() != nil || r.Character.Completed {
			t.Fatal("actual character read failure lost accepted formatter or source completion", body)
		}
	} else if body != nil || order != -1 || w.guarded {
		t.Fatal("erased comparison formatter", body, order)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.rootTracePending() {
		t.Fatal("comparison retained acknowledged formatter")
	}
	order, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if order != 0 {
		t.Fatal("formatter failure changed caller value", order)
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicCompareDisabledInnerTraceDoesNotRead(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	serviceMust(t, c.traceComparisonRoots(nil, nil))
	serviceMust(t, c.traceComparison(nil, nil))
}
