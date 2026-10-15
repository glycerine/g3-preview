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

func TestFastInequalityReferenceTraceClauseOrder(t *testing.T) {
	m, c, v, _ := fastMonomialFixture(t, []bool{false, false})
	newAtom := func(variable uint32, value int32, lower bool) *FastInequality {
		t.Helper()
		if err := m.SetInt32(v, value); err != nil {
			t.Fatal(err)
		}
		a, err := c.NewInequality(variable, v, lower, false)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	first := newAtom(0, 5, true)
	second := newAtom(1, 7, false)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_ref_count")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*FastInequality{second, first}, false, true)
	if err != nil || first.refs != 1 || second.refs != 1 {
		t.Fatal("clause increments", err)
	}
	if err := c.releaseClause(clause); err != nil || first.refs != 0 || second.refs != 0 {
		t.Fatal("clause decrements", err)
	}
	if tags.ShouldTrace("subpaving_ref_count") {
		got := output.String()
		want := []string{
			fmt.Sprintf("inc-ref: %p 0\n", second), fmt.Sprintf("inc-ref: %p 0\n", first),
			fmt.Sprintf("dec-ref: %p 1\n5 <= x0\n", first), fmt.Sprintf("dec-ref: %p 1\nx1 <= 7\n", second),
		}
		last := -1
		for _, part := range want {
			at := strings.Index(got, part)
			if at <= last {
				t.Fatalf("reference TRACE order missing %q in %q", part, got)
			}
			last = at
		}
		if strings.Count(got, "subpaving_t_def.h:786") != 2 || strings.Count(got, "subpaving_t_def.h:794") != 2 || output.flushes != 4 {
			t.Fatalf("reference TRACE headers/flushes: %q, %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased reference TRACE emitted", output.String())
	}
}

func TestFastInequalityReferenceTraceRefusalAndStoppedRelease(t *testing.T) {
	m, c, v, _ := fastMonomialFixture(t, []bool{false})
	if err := m.SetInt32(v, 9); err != nil {
		t.Fatal(err)
	}
	a, err := c.NewInequality(0, v, false, false)
	if err != nil {
		t.Fatal(err)
	}
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
	err = c.releaseUnit(0)
	if tags.ShouldTrace("subpaving_ref_count") {
		if !errors.Is(err, printerErr) || a.refs != 1 || c.unitClauses[0].released || !strings.Contains(output.String(), "candidate") {
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
