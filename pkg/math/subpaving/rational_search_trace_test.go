// Source: src/math/subpaving/subpaving_t_def.h:1806-1843; config_mpq.
package subpaving

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalSearchTraceSelectionPropagationSplitAndDepth(t *testing.T) {
	c, _, _, _, _ := rationalClauseFixture(t)
	c.maxDepth = 1
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if len(c.nodes) != 3 || c.numSplits != 1 || c.searchNode != nil || c.nodes[0].id != 0 || c.nodes[1].id != 1 || c.nodes[2].id != 2 {
		t.Fatal("source search node identity or split", len(c.nodes), c.numSplits)
	}
	if tags.ShouldTrace("subpaving_main") {
		got := output.String()
		for _, part := range []string{
			"subpaving_t_def.h:1810", "max_nodes  8192\nmax_depth  1\nepsilon    1/20\nmax_bound  10000000000\nmax_memory 18446744073709551615\n",
			"selected node: #0, depth: 0\n", "node #0 after propagation\n", "splitting variable: x0\n",
			"selected node: #1, depth: 1\n", "selected node: #2, depth: 1\n",
			"maximum depth reached, skipping node #1\n", "maximum depth reached, skipping node #2\n",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing source search trace %q in %q", part, got)
			}
		}
		if strings.Count(got, "subpaving_t_def.h:1819") != 3 || strings.Count(got, "subpaving_t_def.h:1825") != 3 || strings.Count(got, "subpaving_t_def.h:1833") != 2 || output.flushes != 10 {
			t.Fatalf("source search site counts: %q; flushes %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased source search trace emitted", output.String(), output.flushes)
	}
}

func TestRationalSearchTraceNullVariable(t *testing.T) {
	_, m, _, _, _ := rationalClauseFixture(t)
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 100})
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
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil || c.searchNode != nil || len(c.nodes) != 1 {
		t.Fatal("source null-variable search", err, len(c.nodes))
	}
	if tags.ShouldTrace("subpaving_main") {
		if !strings.Contains(output.String(), "splitting variable: [null]\n") || strings.Count(output.String(), "subpaving_t_def.h:1837") != 1 {
			t.Fatal("source null variable display", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased source null trace emitted", output.String())
	}
}

func TestRationalSearchTraceRefusalAndPrinterResume(t *testing.T) {
	c, _, _, _, _ := rationalClauseFixture(t)
	c.maxDepth = 1
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_main")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil || len(c.nodes) != 3 {
		t.Fatal("borrowed stream refusal changed search", err, len(c.nodes))
	}

	d, _, _, _, _ := rationalClauseFixture(t)
	d.maxDepth = 1
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
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
	err := d.Search()
	if tags.ShouldTrace("subpaving_main") {
		if !errors.Is(err, printerErr) || d.searchNode != d.root || d.searchPhase != 3 || len(d.nodes) != 1 || d.leafHead != nil || !strings.Contains(output.String(), "splitting variable: candidate") {
			t.Fatal("source stopped at split trace after root selection", err, d.searchPhase, len(d.nodes), output.String())
		}
		before := output.String()
		if err := d.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		if err := d.Search(); err != nil || len(d.nodes) != 3 || strings.Count(output.String(), "selected node: #0") != 1 || strings.Count(output.String(), "node #0 after propagation") != 1 || output.String() == before {
			t.Fatal("search resume replayed completed root trace", err, len(d.nodes), output.String())
		}
	} else if err != nil || len(d.nodes) != 3 {
		t.Fatal("native trace gate changed search", err, len(d.nodes))
	}
}
