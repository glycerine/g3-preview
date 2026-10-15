package subpaving

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedInequalityReferenceTraceSourceOrder(t *testing.T) {
	m, c, v, _ := fixedClauseFixture(t, []bool{false})
	a := fixedClauseAtom(t, m, c, v, 0, 5, true, false)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_ref_count")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(a, true); err != nil || a.refs != 1 {
		t.Fatal("unit increment", err)
	}
	if err := c.releaseUnit(0); err != nil || a.refs != 0 {
		t.Fatal("unit decrement", err)
	}
	if tags.ShouldTrace("subpaving_ref_count") {
		got := output.String()
		inc := fmt.Sprintf("inc-ref: %p 0\n", a)
		dec := fmt.Sprintf("dec-ref: %p 1\n5 <= x0\n", a)
		if !strings.Contains(got, "subpaving_t_def.h:786") || !strings.Contains(got, "subpaving_t_def.h:794") ||
			!strings.Contains(got, inc) || !strings.Contains(got, dec) ||
			strings.Index(got, inc) > strings.Index(got, dec) || output.flushes != 2 {
			t.Fatalf("reference TRACE order: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased reference TRACE emitted", output.String())
	}
}

func TestFixedInequalityReferenceTraceRefusalAndStoppedRelease(t *testing.T) {
	m, c, v, _ := fixedClauseFixture(t, []bool{false})
	a := fixedClauseAtom(t, m, c, v, 0, 7, false, false)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_ref_count")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(a, true); err != nil || a.refs != 1 {
		t.Fatal("borrowed stream refusal changed increment", err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_ref_count")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, _ uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err := c.releaseUnit(0)
	if tags.ShouldTrace("subpaving_ref_count") {
		if !errors.Is(err, printerErr) || a.refs != 1 || c.unitClauses[0].released || !strings.Contains(output.String(), "dec-ref: ") || !strings.Contains(output.String(), "candidate") {
			t.Fatal("reference TRACE stopped before decrement", err, a.refs, output.String())
		}
	} else if err != nil || a.refs != 0 {
		t.Fatal("native gate changed release", err)
	}
	if err := c.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
	if err := c.releaseUnit(0); err != nil || a.refs != 0 {
		t.Fatal("retry release", err)
	}
}
