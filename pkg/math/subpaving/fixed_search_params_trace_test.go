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
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("subpaving_main") {
		if got := output.String(); strings.Count(got, "subpaving_t_def.h:1810") != 1 || !strings.Contains(got, want) || output.flushes != 1 {
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

type fixedShortWriter struct{ buffer bytes.Buffer }

func (w *fixedShortWriter) String() string { return w.buffer.String() }

func (w *fixedShortWriter) Write(p []byte) (int, error) {
	if w.buffer.Len() != 0 {
		return 0, io.ErrClosedPipe
	}
	_, _ = w.buffer.Write(p[:3])
	return 3, nil
}
