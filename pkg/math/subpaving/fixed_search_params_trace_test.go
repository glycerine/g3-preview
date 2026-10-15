package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFixedSearchSourceParameterDisplayAndTrace(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, nil, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	want := "max_nodes  8192\nmax_depth  128\nepsilon    0\nmax_bound  100\nmax_memory " + strconv.FormatUint(c.maxMemory, 10) + "\n"
	var direct bytes.Buffer
	if err := c.DisplayParams(&direct); err != nil || direct.String() != want {
		t.Fatalf("source parameter display: %q, %v", direct.String(), err)
	}
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
	if tags.ShouldTrace("subpaving_main") {
		if got := output.String(); strings.Count(got, "subpaving_t_def.h:1810") != 1 || !strings.Contains(got, want) || output.flushes != 6 ||
			strings.Count(got, "subpaving_t_def.h:1809") != 1 || strings.Count(got, "subpaving_t_def.h:1844") != 1 ||
			strings.Count(got, "statistics:\n") != 2 ||
			!strings.Contains(got, "selected node: #0, depth: 0\n") ||
			!strings.Contains(got, "node #0 after propagation\n") ||
			!strings.Contains(got, "splitting variable: [null]\n") {
			t.Fatalf("source search parameter TRACE: %q, flushes=%d", got, output.flushes)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased search parameter TRACE", output.String())
	}
	var short fixedShortWriter
	if err := c.DisplayParams(&short); !errors.Is(err, io.ErrShortWrite) || short.String() != "max" {
		t.Fatal("display first failed prefix", short.String(), err)
	}
	tags = diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal("borrowed trace refusal changed completed search", err)
	}
}

func TestFixedSearchBranchTraceIDsAndDepthLimit(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	c.maxDepth = 1
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_main")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if c.root.id != 0 || len(c.searchOwned) != 2 || c.searchOwned[0].id != 1 || c.searchOwned[1].id != 2 {
		t.Fatal("source node ID generation", c.root.id, len(c.searchOwned))
	}
	first, err := c.NewChild(c.root)
	if err != nil || first.id != 3 {
		t.Fatal("next source node ID", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reused, err := c.NewChild(c.root)
	if err != nil || reused.id != 3 {
		t.Fatal("LIFO recycled node ID", err)
	}
	if err := reused.Close(); err != nil {
		t.Fatal(err)
	}
	if !tags.ShouldTrace("subpaving_main") {
		if output.Len() != 0 {
			t.Fatal("erased branch trace", output.String())
		}
		return
	}
	got := output.String()
	for _, record := range []string{
		"selected node: #0, depth: 0\n", "splitting variable: x0\n",
		"selected node: #1, depth: 1\n", "maximum depth reached, skipping node #1\n",
		"selected node: #2, depth: 1\n", "maximum depth reached, skipping node #2\n",
	} {
		if !strings.Contains(got, record) {
			t.Fatalf("missing MPFX search TRACE %q in %q", record, got)
		}
	}
	if strings.Index(got, "selected node: #0") > strings.Index(got, "selected node: #1") ||
		strings.Index(got, "selected node: #1") > strings.Index(got, "selected node: #2") ||
		strings.Count(got, "subpaving_t_def.h:1819") != 3 || strings.Count(got, "subpaving_t_def.h:1825") != 3 ||
		strings.Count(got, "subpaving_t_def.h:1833") != 2 || output.flushes != 10 {
		t.Fatalf("MPFX breadth-first TRACE order: %q, flushes=%d", got, output.flushes)
	}
}

type fixedShortWriter struct{ buffer bytes.Buffer }

func (w *fixedShortWriter) String() string { return w.buffer.String() }

func (w *fixedShortWriter) Write(p []byte) (int, error) {
	if w.buffer.Len() != 0 {
		return 0, io.ErrClosedPipe
	}
	_, _ = w.buffer.Write(p[:3])
	return 3, nil
}
