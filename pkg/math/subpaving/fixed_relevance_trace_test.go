package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1210-1311; config_mpfx.

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedRelevanceTraceConflictThresholdAndImprovementBranches(t *testing.T) {
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	check := func(value int64, lower, open, want bool, line int, reason string) {
		t.Helper()
		if err := m.SetInt64(v, value); err != nil {
			t.Fatal(err)
		}
		prior, flushes := output.Len(), output.flushes
		got, err := c.RelevantNewBound(root, 0, v, lower, open)
		if err != nil || got != want {
			t.Fatalf("relevance %d lower=%t open=%t: %t, %v; want %t", value, lower, open, got, err, want)
		}
		if tags.ShouldTrace("subpaving_relevant_bound") {
			record := output.String()[prior:]
			if !strings.Contains(record, "subpaving_t_def.h:1210") || !strings.Contains(record, "existing bounds:\n") ||
				!strings.Contains(record, "subpaving_t_def.h:"+strconv.Itoa(line)) || !strings.Contains(record, reason) || output.flushes-flushes != 2 {
				t.Fatalf("relevance branch TRACE: %q, flushes=%d", record, output.flushes-flushes)
			}
		} else if output.Len() != prior || output.flushes != flushes {
			t.Fatal("erased relevance TRACE emitted", output.String())
		}
	}
	check(1, true, false, true, 1310, "new bound is relevant")
	check(101, true, false, false, 1229, "exceeds :max-bound threshold")
	check(-101, false, false, false, 1279, "exceeds -:max-bound threshold")
	for _, bound := range []struct {
		value int64
		lower bool
	}{{0, true}, {10, false}} {
		if err := m.SetInt64(v, bound.value); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, 0, v, bound.lower, false); err != nil {
			t.Fatal(err)
		}
	}
	check(0, true, false, false, 1224, "does not improve existing bound")
	check(10, false, false, false, 1274, "does not improve existing bound")
	check(10, true, true, true, 1218, "triggers conflict")
	check(0, false, true, true, 1268, "triggers conflict")
	check(1, true, false, true, 1310, "new bound is relevant")
	check(9, false, false, true, 1310, "new bound is relevant")
}

func TestFixedRelevanceTraceRefusalAndPrinterPrefix(t *testing.T) {
	m, c, v, root := fixedClauseFixture(t, []bool{false})
	if err := m.SetInt64(v, 1); err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_relevant_bound")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if got, err := c.RelevantNewBound(root, 0, v, true, false); err != nil || !got {
		t.Fatal("borrowed trace refusal changed relevance", got, err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	got, err := c.RelevantNewBound(root, 0, v, true, false)
	if tags.ShouldTrace("subpaving_relevant_bound") {
		if !errors.Is(err, printerErr) || got || c.arithmeticFailed || !strings.Contains(output.String(), "candidate") {
			t.Fatal("candidate TRACE stopped before comparisons", got, err, output.String())
		}
	} else if err != nil || !got {
		t.Fatal("native TRACE gate changed relevance", got, err)
	}
}
