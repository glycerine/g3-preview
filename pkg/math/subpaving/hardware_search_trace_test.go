package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1809-1844; config_hwf.

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

func hardwareSearchTraceFixture(t *testing.T, variables []bool) (*numeral.HardwareFloatNumeralManager, *HardwareContext, *numeral.HardwareFloat, *HardwareNode) {
	t.Helper()
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, integer := range variables {
		if _, err := c.NewVariable(integer); err != nil {
			t.Fatal(err)
		}
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	return m, c, &numeral.HardwareFloat{}, root
}

func TestHardwareSearchTraceSelectionSplitDepthAndStatistics(t *testing.T) {
	_, c, _, _ := hardwareSearchTraceFixture(t, []bool{false})
	c.maxDepth = 1
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	tags.Enable("subpaving_stats")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
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
				t.Fatalf("missing source search TRACE %q in %q", part, got)
			}
		}
		if !regexp.MustCompile(`:nodes\s+3`).MatchString(got) || !regexp.MustCompile(`:splits\s+1`).MatchString(got) ||
			strings.Count(got, "subpaving_t_def.h:1819") != 3 || strings.Count(got, "subpaving_t_def.h:1825") != 3 ||
			strings.Count(got, "subpaving_t_def.h:1833") != 2 || output.flushes != 12 {
			t.Fatalf("search source record counts: %q; flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased search TRACE emitted", output.String())
	}
}

func TestHardwareSearchTraceNullVariableAndBorrowedRefusal(t *testing.T) {
	_, c, _, _ := hardwareSearchTraceFixture(t, nil)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("subpaving_main") {
		if !strings.Contains(output.String(), "splitting variable: [null]\n") {
			t.Fatal("source null-variable search", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased null-variable TRACE emitted")
	}
	_, d, _, _ := hardwareSearchTraceFixture(t, []bool{false})
	d.maxDepth = 1
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_main")
	failed.Enable("subpaving_stats")
	if err := d.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := d.Search(); err != nil || len(d.nodes) != 3 || d.numSplits != 1 {
		t.Fatal("borrowed writer refusal changed search", err, len(d.nodes), d.numSplits)
	}
}

func TestHardwareSearchTraceNodeLimitStillEmitsExitStatistics(t *testing.T) {
	_, c, _, _ := hardwareSearchTraceFixture(t, []bool{false})
	c.maxNodes = 0
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_stats")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil || len(c.nodes) != 1 {
		t.Fatal("source node-limit stop", err, len(c.nodes))
	}
	if tags.ShouldTrace("subpaving_stats") {
		if strings.Count(output.String(), "statistics:\n(:nodes 1)\n") != 2 ||
			!strings.Contains(output.String(), "subpaving_t_def.h:1844") || output.flushes != 2 {
			t.Fatal("missing exit statistics after node limit", output.String(), output.flushes)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased statistics TRACE emitted")
	}
}

func TestHardwareSearchTraceChildConflict(t *testing.T) {
	m, c, v, _ := hardwareSearchTraceFixture(t, []bool{false, false})
	c.maxDepth = 1
	if err := m.SetInt32(v, 0); err != nil {
		t.Fatal(err)
	}
	xNegative, err := c.NewInequality(0, v, false, true)
	if err != nil {
		t.Fatal(err)
	}
	yAtMostZero, err := c.NewInequality(1, v, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(v, 1); err != nil {
		t.Fatal(err)
	}
	yAtLeastOne, err := c.NewInequality(1, v, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, consequent := range []*HardwareInequality{yAtLeastOne, yAtMostZero} {
		if _, err := c.AddClause([]*HardwareInequality{xNegative, consequent}, false, true); err != nil {
			t.Fatal(err)
		}
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if len(c.nodes) != 3 || c.nodes[2].conflictVar == math.MaxUint32 || c.numConflicts != 1 {
		t.Fatal("right child did not conflict", len(c.nodes), c.numConflicts)
	}
	if tags.ShouldTrace("subpaving_main") {
		got := output.String()
		if !strings.Contains(got, "node #2 after propagation\n") || !strings.Contains(got, "node #2 is inconsistent.\n") ||
			strings.Count(got, "subpaving_t_def.h:1828") != 1 {
			t.Fatal("source conflict record", got)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased conflict TRACE emitted")
	}
}

func TestHardwareSearchTraceStoppedBoundsPrinterPreservesSelectedNode(t *testing.T) {
	m, c, v, root := hardwareSearchTraceFixture(t, []bool{false})
	c.maxDepth = 0
	if err := m.SetInt32(v, 1); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(root, 0, v, true, false); err != nil {
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
	err := c.Search()
	if tags.ShouldTrace("subpaving_main") {
		if !errors.Is(err, printerErr) || c.searchNode != root || c.searchPhase != 2 || !strings.Contains(output.String(), "node #0 after propagation\n") {
			t.Fatal("selected node lost at display stop", err, c.searchPhase, output.String())
		}
		if err := c.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		c.maxDepth = 0
		if err := c.Search(); err != nil || c.searchNode != nil {
			t.Fatal("selected node did not resume", err)
		}
	} else if err != nil {
		t.Fatal("erased TRACE changed search", err)
	}
}
