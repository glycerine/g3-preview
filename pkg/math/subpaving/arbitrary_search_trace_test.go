// Source: src/math/subpaving/subpaving_t_def.h:1809-1844; config_mpf.
package subpaving

import (
	"bytes"
	"errors"
	"io"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func arbitraryTraceBoundedRoot(t *testing.T, f *arbitraryClauseFixture) *ArbitraryNode {
	t.Helper()
	root, err := f.context.Root()
	if err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		value *numeral.ArbitraryFloat
		lower bool
	}{{&f.four, true}, {&f.five, false}} {
		if err := f.context.PublishBound(root, f.x, bound.value, bound.lower, false, true); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestArbitrarySearchTraceSelectionSplitDepthAndStatistics(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	c.maxDepth = 1
	arbitraryTraceBoundedRoot(t, f)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	tags.Enable("subpaving_stats")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err, "nodes", len(c.nodes), "phase", c.searchPhase, "selected", c.searchNode, "trace", output.String())
	}
	if len(c.nodes) != 3 || c.numSplits != 1 || c.nodes[0].id != 0 || c.nodes[1].id != 1 || c.nodes[2].id != 2 {
		t.Fatal("source node identity and breadth-first split", len(c.nodes), c.numSplits)
	}
	if tags.ShouldTrace("subpaving_main") {
		got := output.String()
		for _, part := range []string{
			"subpaving_t_def.h:1809", "subpaving_t_def.h:1810", "max_nodes  8192\nmax_depth  1\n",
			"selected node: #0, depth: 0\n", "node #0 after propagation\n", "splitting variable: x0\n",
			"selected node: #1, depth: 1\n", "selected node: #2, depth: 1\n",
			"maximum depth reached, skipping node #1\n", "maximum depth reached, skipping node #2\n",
			"subpaving_t_def.h:1844",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing search TRACE %q in %q", part, got)
			}
		}
		if !regexp.MustCompile(`:nodes\s+3`).MatchString(got) || !regexp.MustCompile(`:splits\s+1`).MatchString(got) ||
			strings.Count(got, "subpaving_t_def.h:1819") != 3 || strings.Count(got, "subpaving_t_def.h:1825") != 3 ||
			strings.Count(got, "subpaving_t_def.h:1833") != 2 || output.flushes != 12 {
			t.Fatalf("search record counts: %q; flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased TRACE emitted", output.String())
	}
}

func TestArbitrarySearchTraceNullVariableAndNodeLimit(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c, err := NewArbitraryContext(f.numbers, &control.Reslimit{Limit: 100}, 0, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	tags.Enable("subpaving_stats")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil || len(c.nodes) != 1 {
		t.Fatal("null-variable search", err, len(c.nodes))
	}
	if tags.ShouldTrace("subpaving_main") {
		if !strings.Contains(output.String(), "splitting variable: [null]\n") ||
			strings.Count(output.String(), "statistics:\n(:nodes 1)\n") != 2 {
			t.Fatal("source null-variable and exit statistics", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased null-variable TRACE emitted")
	}
	other := newArbitraryClauseFixture(t)
	other.context.maxNodes = 0
	var limitOutput polynomialTraceFlushingBuffer
	limitTags := diagnostic.NewTraceTagsWithWriter(&limitOutput)
	limitTags.Enable("subpaving_stats")
	if err := other.context.BindTrace(limitTags); err != nil {
		t.Fatal(err)
	}
	if err := other.context.Search(); err != nil || len(other.context.nodes) != 1 {
		t.Fatal("node-limit stop", err)
	}
	if limitTags.ShouldTrace("subpaving_stats") &&
		(strings.Count(limitOutput.String(), "statistics:\n(:nodes 1)\n") != 2 || limitOutput.flushes != 2) {
		t.Fatal("node-limit exit statistics", limitOutput.String())
	}
}

func TestArbitrarySearchTraceRefusalAndStoppedPrinter(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	f.context.maxDepth = 1
	arbitraryTraceBoundedRoot(t, f)
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_main")
	failed.Enable("subpaving_stats")
	if err := f.context.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := f.context.Search(); err != nil || len(f.context.nodes) != 3 {
		t.Fatal("borrowed writer refusal changed search", err, len(f.context.nodes))
	}
	other := newArbitraryClauseFixture(t)
	c := other.context
	c.maxDepth = 0
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, other.x, &other.four, true, false, true); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("bounds printer stopped")
	if err := c.SetVariableDisplay(func(w io.Writer, _ uint32) error {
		_, _ = io.WriteString(w, "candidate")
		return printerErr
	}); err != nil {
		t.Fatal(err)
	}
	err = c.Search()
	if tags.ShouldTrace("subpaving_main") {
		if !errors.Is(err, printerErr) || c.searchNode != root || c.searchPhase != 2 || !strings.Contains(output.String(), "node #0 after propagation\n") {
			t.Fatal("selected node lost at display stop", err, c.searchPhase, output.String())
		}
		if err := c.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		if err := c.Search(); err != nil || c.searchNode != nil {
			t.Fatal("selected node did not resume", err)
		}
	} else if err != nil {
		t.Fatal("erased TRACE changed search", err)
	}
}

func TestArbitrarySearchTraceChildConflict(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c, m := f.context, f.numbers
	c.maxDepth = 1
	var one, two numeral.ArbitraryFloat
	if err := m.SetInt32(&one, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&two, 2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Del(&two); err != nil {
			t.Error(err)
		}
		if err := m.Del(&one); err != nil {
			t.Error(err)
		}
	})
	xBelowOne, err := c.NewInequality(f.x, &one, false, true)
	if err != nil {
		t.Fatal(err)
	}
	yAtMostOne, err := c.NewInequality(f.y, &one, false, false)
	if err != nil {
		t.Fatal(err)
	}
	yAtLeastTwo, err := c.NewInequality(f.y, &two, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, consequent := range []*ArbitraryInequality{yAtLeastTwo, yAtMostOne} {
		if _, err := c.AddClause([]*ArbitraryInequality{xBelowOne, consequent}); err != nil {
			t.Fatal(err)
		}
	}
	var minusOne, three numeral.ArbitraryFloat
	if err := m.SetInt32(&minusOne, -1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&three, 3); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Del(&minusOne); err != nil {
			t.Error(err)
		}
		if err := m.Del(&three); err != nil {
			t.Error(err)
		}
	})
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &minusOne, true, false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.x, &three, false, false, true); err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err, "nodes", len(c.nodes), "phase", c.searchPhase, "selected", c.searchNode, "trace", output.String())
	}
	if len(c.nodes) != 3 || c.nodes[2].conflictVar == math.MaxUint32 || c.numConflicts != 1 {
		t.Fatal("right child did not conflict", len(c.nodes), c.numConflicts)
	}
	if tags.ShouldTrace("subpaving_main") {
		if !strings.Contains(output.String(), "node #2 after propagation\n") ||
			!strings.Contains(output.String(), "node #2 is inconsistent.\n") || strings.Count(output.String(), "subpaving_t_def.h:1828") != 1 {
			t.Fatal("source conflict record", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased conflict TRACE emitted")
	}
}
