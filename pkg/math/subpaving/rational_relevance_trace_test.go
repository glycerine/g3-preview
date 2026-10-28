package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1210-1311; config_mpq.

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func rationalRelevanceTraceFixture(t *testing.T) (*numeral.RationalManager, *RationalContext, *numeral.Rational, *RationalNode) {
	t.Helper()
	c, m, value, _, _ := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	return m, c, value, root
}

func TestRationalRelevanceTraceConflictThresholdAndImprovementBranches(t *testing.T) {
	m, c, v, root := rationalRelevanceTraceFixture(t)
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
	check(10_000_000_001, true, false, false, 1229, "exceeds :max-bound threshold")
	check(-10_000_000_001, false, false, false, 1279, "exceeds -:max-bound threshold")
	if err := c.updateEpsilon(0); err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		value int64
		lower bool
	}{{0, true}, {10, false}} {
		if err := m.SetInt64(v, bound.value); err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, 0, v, bound.lower, false); err != nil {
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

func TestRationalRelevanceTraceRefusalAndPrinterPrefix(t *testing.T) {
	m, c, v, root := rationalRelevanceTraceFixture(t)
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
		if !errors.Is(err, printerErr) || got || !strings.Contains(output.String(), "candidate") {
			t.Fatal("candidate TRACE stopped before comparisons", got, err, output.String())
		}
	} else if err != nil || !got {
		t.Fatal("native TRACE gate changed relevance", got, err)
	}
}

func TestRationalRelevanceTraceEpsilonGapRecords(t *testing.T) {
	m, c, value, root := rationalRelevanceTraceFixture(t)
	if err := c.updateEpsilon(20); err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		value int64
		lower bool
	}{{10, true}, {20, false}} {
		if err := m.SetInt64(value, bound.value); err != nil {
			t.Fatal(err)
		}
		if err := c.PublishBound(root, 0, value, bound.lower, false); err != nil {
			t.Fatal(err)
		}
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	tags.Enable("subpaving_relevant_bound_bug")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		numerator int32
		lower     bool
		line      string
	}{
		{41, true, "subpaving_t_def.h:1259"},
		{79, false, "subpaving_t_def.h:1304"},
	} {
		if err := m.SetFraction(value, int64(tc.numerator), 4); err != nil {
			t.Fatal(err)
		}
		prior, flushes := output.Len(), output.flushes
		relevant, err := c.RelevantNewBound(root, 0, value, tc.lower, false)
		if err != nil || relevant {
			t.Fatal("epsilon-gap candidate must be refused", tc, relevant, err)
		}
		if tags.ShouldTrace("subpaving_relevant_bound") {
			record := output.String()[prior:]
			if !strings.Contains(record, "subpaving_t_def.h:1210") || !strings.Contains(record, tc.line) ||
				!strings.Contains(record, "does not improve existing bound to at least ") {
				t.Fatal("epsilon-gap source record", tc, record)
			}
			wantFlushes := 2
			if tc.lower {
				wantFlushes = 3
				if !strings.Contains(record, "subpaving_t_def.h:1253") || !strings.Contains(record, "curr_lower: ") {
					t.Fatal("lower epsilon diagnostic", record)
				}
			}
			if output.flushes-flushes != wantFlushes {
				t.Fatal("epsilon-gap trace flushes", tc, output.flushes-flushes)
			}
		}
	}
}
