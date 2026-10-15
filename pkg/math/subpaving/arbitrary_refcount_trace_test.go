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

func TestArbitraryInequalityReferenceTraceSourceOrder(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	first, err := c.NewInequality(f.x, &f.five, true, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.NewInequality(f.y, &f.five, false, false)
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_ref_count")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddClause([]*ArbitraryInequality{second, first}); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil || first.refs != 0 || second.refs != 0 {
		t.Fatal("MPF clause release", err)
	}
	if tags.ShouldTrace("subpaving_ref_count") {
		got := output.String()
		want := []string{
			fmt.Sprintf("inc-ref: %p 0\n", second), fmt.Sprintf("inc-ref: %p 0\n", first),
			fmt.Sprintf("dec-ref: %p 1\n5 <= x0\n", first), fmt.Sprintf("dec-ref: %p 1\nx1 <= 5\n", second),
		}
		last := -1
		for _, part := range want {
			at := strings.Index(got, part)
			if at <= last {
				t.Fatalf("MPF reference TRACE order missing %q in %q", part, got)
			}
			last = at
		}
		if strings.Count(got, "subpaving_t_def.h:786") != 2 || strings.Count(got, "subpaving_t_def.h:794") != 2 || output.flushes != 4 {
			t.Fatalf("reference TRACE headers/flushes: %q, %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased MPF reference TRACE emitted", output.String())
	}
}

func TestArbitraryInequalityReferenceTraceRefusalAndStoppedClause(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	first, err := c.NewInequality(f.x, &f.four, false, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.NewInequality(f.y, &f.four, false, false)
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_ref_count")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*ArbitraryInequality{first, second})
	if err != nil || first.refs != 1 || second.refs != 1 {
		t.Fatal("borrowed stream refusal changed increments", err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_ref_count")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("second atom printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		if variable == f.y {
			_, _ = io.WriteString(w, "candidate")
			return printerErr
		}
		_, err := io.WriteString(w, "x0")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	err = c.Close()
	if tags.ShouldTrace("subpaving_ref_count") {
		if !errors.Is(err, printerErr) || first.refs != 0 || second.refs != 1 || clause.releaseAt != 1 || len(c.clauses) != 1 {
			t.Fatal("stopped MPF clause suffix", err, first.refs, second.refs, clause.releaseAt)
		}
	} else if err != nil || !c.closed {
		t.Fatal("native gate changed MPF close", err)
	}
	if !c.closed {
		if err := c.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		if err := c.Close(); err != nil || second.refs != 0 {
			t.Fatal("resume MPF clause suffix", err)
		}
		if strings.Count(output.String(), fmt.Sprintf("dec-ref: %p 1\n", first)) != 1 {
			t.Fatal("completed MPF decrement replayed", output.String())
		}
	}
}
