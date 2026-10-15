package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func fixedMonomialPropagationTraceFixture(t *testing.T) (*FixedContext, uint32, *FixedNode) {
	t.Helper()
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	x, err := c.DefineMonomial([]FixedPower{{Variable: 0, Degree: 3}})
	if err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		value int64
		lower bool
	}{{8, true}, {27, false}} {
		if err := m.SetInt64(v, bound.value); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, x, v, bound.lower, false); err != nil {
			t.Fatal(err)
		}
	}
	return c, x, root
}

func TestFixedMonomialPropagationSourceTrace(t *testing.T) {
	c, x, root := fixedMonomialPropagationTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_monomial")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateMonomial(x, root); err != nil || root.lower[0] == nil || root.upper[0] == nil {
		t.Fatal("downward monomial propagation", err)
	}
	if tags.ShouldTrace("propagate_monomial") {
		got := output.String()
		for _, part := range []string{
			"subpaving_t_def.h:1501", "propagate_monomial: x1\n",
			"subpaving_t_def.h:1525", "found_zero: 0, found_unbounded: 1\n",
			"subpaving_t_def.h:1603", "propagate_monomial_downward: x1, j: 0\nx0^3\n",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing monomial TRACE %q in %q", part, got)
			}
		}
		if !(strings.Index(got, "subpaving_t_def.h:1501") < strings.Index(got, "subpaving_t_def.h:1525") &&
			strings.Index(got, "subpaving_t_def.h:1525") < strings.Index(got, "subpaving_t_def.h:1603")) || output.flushes != 3 {
			t.Fatalf("monomial TRACE order: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased monomial TRACE emitted", output.String())
	}
}

func TestFixedMonomialPropagationTraceRefusalAndPrinterPrefix(t *testing.T) {
	c, x, root := fixedMonomialPropagationTraceFixture(t)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("propagate_monomial")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateMonomial(x, root); err != nil || root.lower[0] == nil {
		t.Fatal("borrowed trace refusal changed monomial propagation", err)
	}

	d, otherX, otherRoot := fixedMonomialPropagationTraceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_monomial")
	if err := d.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := d.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err := d.PropagateMonomial(otherX, otherRoot)
	if tags.ShouldTrace("propagate_monomial") {
		if !errors.Is(err, printerErr) || d.monomials[otherX].visited != 0 || otherRoot.lower[0] != nil ||
			!strings.Contains(output.String(), "propagate_monomial: candidate") {
			t.Fatal("monomial TRACE stopped before visited update", err, d.monomials[otherX].visited, output.String())
		}
	} else if err != nil || otherRoot.lower[0] == nil {
		t.Fatal("native gate changed monomial propagation", err)
	}
}
