// Source: src/util/trace.cpp:25-55; src/util/trace.h:40-46.
package diagnostic

import (
	"io"
	"os"
	"strconv"
	"strings"
)

// TraceTags is the caller-owned, single-goroutine trace-tag control state.
// The retained shell and API callers can bind one shared instance.
type TraceTags struct {
	all     bool
	tags    map[string]struct{}
	writer  io.Writer
	openErr error
}

func NewTraceTags() *TraceTags { return &TraceTags{} }

// The source tag table and trace stream are process globals. An explicit
// instance may borrow a writer for a concrete caller or a focused test.
var processTraceTags = newProcessTraceTags()

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

func (t *TraceTags) traceWriter() (io.Writer, error) {
	if t.openErr != nil {
		return nil, t.openErr
	}
	if t.writer == nil {
		f, err := os.OpenFile(".z3-trace", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return nil, err
		}
		t.writer = f
	}
	return t.writer, nil
}

func writeTracePart(w io.Writer, part string) error {
	n, err := io.WriteString(w, part)
	if err != nil {
		return err
	}
	if n != len(part) {
		return io.ErrShortWrite
	}
	return nil
}

func flushTrace(w io.Writer) error {
	if f, ok := w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// TRACE and CTRACE insert each header component separately before CODE runs.
func writeTraceHeader(w io.Writer, tag, function, file string, line int) error {
	for _, part := range [...]string{"-------- [", traceTagCString(tag), "] ", function, " ", file, ":", strconv.Itoa(line), " ---------\n"} {
		if err := writeTracePart(w, part); err != nil {
			return err
		}
	}
	return nil
}

// Emit writes one TRACE record to the source-order stream. The caller
// supplies its static source site and already-gated body. An explicitly
// constructed unbound stream opens .z3-trace on its first
// emitted record. The process stream opens it during package initialization.
func (t *TraceTags) Emit(tag, function, file string, line int, body string) error {
	if !t.ShouldTrace(tag) {
		return nil
	}
	w, err := t.traceWriter()
	if err != nil {
		return err
	}
	if err := writeTraceHeader(w, tag, function, file, line); err != nil {
		return err
	}
	if err := writeTracePart(w, body); err != nil {
		return err
	}
	if err := writeTracePart(w, "------------------------------------------------\n"); err != nil {
		return err
	}
	return flushTrace(w)
}

// EmitShort implements STRACE: no header or footer, but each reached body is
// flushed. The body is evaluated only after the compile and tag gates.
func (t *TraceTags) EmitShort(tag string, body func(io.Writer) error) error {
	if !t.ShouldTrace(tag) {
		return nil
	}
	w, err := t.traceWriter()
	if err != nil {
		return err
	}
	if body != nil {
		if err := body(w); err != nil {
			return err
		}
	}
	return flushTrace(w)
}

// EmitConditional implements CTRACE's compile, tag, then condition order.
// Neither the condition nor the body is evaluated when an earlier gate fails.
// The header is written before the body, just as the source macro expands.
func (t *TraceTags) EmitConditional(tag, function, file string, line int, condition func() bool, body func(io.Writer) error) error {
	if !t.ShouldTrace(tag) || condition == nil || !condition() {
		return nil
	}
	w, err := t.traceWriter()
	if err != nil {
		return err
	}
	if err := writeTraceHeader(w, tag, function, file, line); err != nil {
		return err
	}
	if body != nil {
		if err := body(w); err != nil {
			return err
		}
	}
	if err := writeTracePart(w, "------------------------------------------------\n"); err != nil {
		return err
	}
	return flushTrace(w)
}

// Finalize releases the tag table. Source finalize_trace leaves the all flag.
func (t *TraceTags) Finalize() { t.tags = nil }
