// Source: src/util/trace.cpp:25-55; src/util/trace.h:40-46.
package diagnostic

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// TraceTags is the caller-owned, single-goroutine trace-tag control state.
// The retained shell and API callers can bind one shared instance.
type TraceTags struct {
	all    bool
	tags   map[string]struct{}
	writer io.Writer
}

func NewTraceTags() *TraceTags { return &TraceTags{} }

// The source tag table and trace stream are process globals. An explicit
// instance may borrow a writer for a concrete caller or a focused test.
var processTraceTags = NewTraceTags()

func BorrowProcessTraceTags() *TraceTags { return processTraceTags }

func NewTraceTagsWithWriter(w io.Writer) *TraceTags { return &TraceTags{writer: w} }

func traceTagCString(tag string) string {
	if i := strings.IndexByte(tag, 0); i >= 0 {
		return tag[:i]
	}
	return tag
}

func (t *TraceTags) Enable(tag string) {
	if t.tags == nil {
		t.tags = make(map[string]struct{})
	}
	t.tags[traceTagCString(tag)] = struct{}{}
}

func (t *TraceTags) Disable(tag string) {
	if t.tags == nil {
		t.tags = make(map[string]struct{})
	}
	delete(t.tags, traceTagCString(tag))
}

func (t *TraceTags) EnableAll(flag bool) { t.all = flag }

func (t *TraceTags) IsEnabled(tag string) bool {
	if t.all {
		return true
	}
	_, found := t.tags[traceTagCString(tag)]
	return found
}

// ShouldTrace includes the source _TRACE compile gate before the tag test.
// Callers use it before constructing the trace body, matching TRACE_CODE.
func (t *TraceTags) ShouldTrace(tag string) bool {
	return traceCompiled && t != nil && t.IsEnabled(tag)
}

// Emit writes one TRACE/CTRACE record to the source-order stream. The caller
// supplies its static source site and already-gated body. The default stream
// opens .z3-trace on first emitted record in a trace-enabled build.
func (t *TraceTags) Emit(tag, function, file string, line int, body string) error {
	if !t.ShouldTrace(tag) {
		return nil
	}
	if t.writer == nil {
		f, err := os.OpenFile(".z3-trace", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		t.writer = f
	}
	tag = traceTagCString(tag)
	header := fmt.Sprintf("-------- [%s] %s %s:%d ---------\n", tag, function, file, line)
	for _, part := range []string{header, body, "------------------------------------------------\n"} {
		n, err := io.WriteString(t.writer, part)
		if err != nil {
			return err
		}
		if n != len(part) {
			return io.ErrShortWrite
		}
	}
	if f, ok := t.writer.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// Finalize releases the tag table. Source finalize_trace leaves the all flag.
func (t *TraceTags) Finalize() { t.tags = nil }
