// Source: src/math/subpaving/subpaving_t_def.h:1441-1484; config_mpfx.
package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func fixedPolynomialPropagationTraceFixture(t *testing.T) (*FixedContext, uint32, *FixedNode) {
	t.Helper()
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	coefficient, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := coefficient.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := m.SetInt64(v, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(coefficient, 1); err != nil {
		t.Fatal(err)
	}
	defined, err := c.DefinePolynomial(v, []FixedPolynomialTerm{{Variable: 0, Coefficient: coefficient}})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(v, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, defined, v, true, false); err != nil {
		t.Fatal(err)
	}
	return c, defined, root
}

func TestFixedPolynomialPropagationSourceTraceOrder(t *testing.T) {
	c, defined, root := fixedPolynomialPropagationTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"propagate_polynomial", "propagate_polynomial_detail", "propagate_polynomial_bug"} {
		tags.Enable(tag)
	}
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagatePolynomial(defined, root); err != nil || root.lower[0] == nil {
		t.Fatal("source backward polynomial propagation", err)
	}
	if tags.ShouldTrace("propagate_polynomial") {
		got := output.String()
		for _, part := range []string{
			"subpaving_t_def.h:1467", "propagate_polynomial: x1\n",
			"subpaving_t_def.h:1468", "2 <= x1",
			"subpaving_t_def.h:1484", "unbounded_var: x0\n",
			"subpaving_t_def.h:1441", "a: 1\n",
			"subpaving_t_def.h:1444", "r before mul 1/a: ",
			"subpaving_t_def.h:1446", "r after mul 1/a:  ",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing polynomial TRACE %q in %q", part, got)
			}
		}
		if !(strings.Index(got, "subpaving_t_def.h:1467") < strings.Index(got, "subpaving_t_def.h:1468") &&
			strings.Index(got, "subpaving_t_def.h:1468") < strings.Index(got, "subpaving_t_def.h:1484") &&
			strings.Index(got, "subpaving_t_def.h:1484") < strings.Index(got, "subpaving_t_def.h:1441") &&
			strings.Index(got, "subpaving_t_def.h:1441") < strings.Index(got, "subpaving_t_def.h:1444") &&
			strings.Index(got, "subpaving_t_def.h:1444") < strings.Index(got, "subpaving_t_def.h:1446")) || output.flushes != 6 {
			t.Fatalf("polynomial TRACE order: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased polynomial TRACE emitted", output.String())
	}
}

func TestFixedPolynomialPropagationTraceRefusalAndPrinterPrefix(t *testing.T) {
	c, defined, root := fixedPolynomialPropagationTraceFixture(t)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("propagate_polynomial")
	failed.Enable("propagate_polynomial_detail")
	failed.Enable("propagate_polynomial_bug")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagatePolynomial(defined, root); err != nil || root.lower[0] == nil {
		t.Fatal("borrowed trace refusal changed polynomial propagation", err)
	}

	d, otherDefined, otherRoot := fixedPolynomialPropagationTraceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_polynomial")
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
	err := d.PropagatePolynomial(otherDefined, otherRoot)
	if tags.ShouldTrace("propagate_polynomial") {
		if !errors.Is(err, printerErr) || d.polynomials[otherDefined].visited != 0 || otherRoot.lower[0] != nil ||
			!strings.Contains(output.String(), "propagate_polynomial: candidate") {
			t.Fatal("polynomial TRACE stopped before visited update", err, d.polynomials[otherDefined].visited, output.String())
		}
	} else if err != nil || otherRoot.lower[0] == nil {
		t.Fatal("native gate changed polynomial propagation", err)
	}
}
