// Source: src/math/subpaving/subpaving_t_def.h:1210-1311; config_mpf.
package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryRelevanceTraceBranchesAndEpsilonGap(t *testing.T) {
	f := newArbitraryClauseFixtureWithPolicy(t, 2, false)
	c := f.context
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	tags.Enable("subpaving_relevant_bound_bug")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	check := func(value *numeral.ArbitraryFloat, lower, want bool, source, reason string, flushes int) {
		t.Helper()
		start, priorFlushes := output.Len(), output.flushes
		got, err := c.RelevantNewBound(root, f.x, value, lower, false)
		if err != nil || got != want {
			t.Fatalf("relevance lower=%t: %t, %v; want %t", lower, got, err, want)
		}
		if tags.ShouldTrace("subpaving_relevant_bound") {
			record := output.String()[start:]
			if !strings.Contains(record, "subpaving_t_def.h:1210") || !strings.Contains(record, "existing bounds:\n") ||
				!strings.Contains(record, source) || !strings.Contains(record, reason) || output.flushes-priorFlushes != flushes {
				t.Fatalf("source TRACE %q; flushes=%d", record, output.flushes-priorFlushes)
			}
		} else if output.Len() != start || output.flushes != priorFlushes {
			t.Fatal("erased TRACE emitted")
		}
	}
	check(&f.five, true, true, "subpaving_t_def.h:1310", "new bound is relevant", 2)
	var beyond, below, ten, seven, eight numeral.ArbitraryFloat
	for _, pair := range []struct {
		out   *numeral.ArbitraryFloat
		value int32
	}{{&beyond, 101}, {&below, -101}, {&ten, 10}, {&seven, 7}, {&eight, 8}} {
		if err := f.numbers.SetInt32(pair.out, pair.value); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, v := range []*numeral.ArbitraryFloat{&beyond, &below, &ten, &seven, &eight} {
			if err := f.numbers.Del(v); err != nil {
				t.Error(err)
			}
		}
	})
	check(&beyond, true, false, "subpaving_t_def.h:1229", "exceeds :max-bound threshold", 2)
	check(&below, false, false, "subpaving_t_def.h:1279", "exceeds -:max-bound threshold", 2)
	if err := c.PublishBound(root, f.x, &f.four, true, false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &ten, false, false, true); err != nil {
		t.Fatal(err)
	}
	check(&f.five, true, false, "subpaving_t_def.h:1259", "does not improve existing bound to at least", 3)
	check(&eight, false, false, "subpaving_t_def.h:1304", "does not improve existing bound to at least", 2)
	check(&seven, true, true, "subpaving_t_def.h:1310", "new bound is relevant", 3)
	check(&below, false, true, "subpaving_t_def.h:1268", "triggers conflict", 2)
	check(&beyond, true, true, "subpaving_t_def.h:1218", "triggers conflict", 2)
}

func TestArbitraryRelevanceTraceRefusalAndPrinterPrefix(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	root, err := f.context.Root()
	if err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_relevant_bound")
	if err := f.context.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if got, err := f.context.RelevantNewBound(root, f.x, &f.four, true, false); err != nil || !got {
		t.Fatal("borrowed trace refusal changed relevance", got, err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	if err := f.context.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("variable printer stopped")
	if err := f.context.SetVariableDisplay(func(w io.Writer, _ uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	got, err := f.context.RelevantNewBound(root, f.x, &f.four, true, false)
	if tags.ShouldTrace("subpaving_relevant_bound") {
		if got || !errors.Is(err, printerErr) || !strings.Contains(output.String(), "candidate") {
			t.Fatal("candidate TRACE failure prefix", got, err, output.String())
		}
	} else if err != nil || !got {
		t.Fatal("erased TRACE changed relevance", got, err)
	}
}
