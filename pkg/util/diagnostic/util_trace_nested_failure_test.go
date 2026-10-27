package diagnostic

import (
	"io"
	"testing"
)

type nestedTraceFailureWriter struct {
	failAt string
	failed bool
	after  int
}

func (w *nestedTraceFailureWriter) Write(p []byte) (int, error) {
	if w.failed {
		w.after++
		return 0, io.ErrClosedPipe
	}
	if string(p) == w.failAt {
		w.failed = true
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func TestNestedTraceRecordsShareImmediateStreamFailure(t *testing.T) {
	for _, failAt := range []string{"outer", "inner"} {
		t.Run(failAt, func(t *testing.T) {
			out := &nestedTraceFailureWriter{failAt: failAt}
			tags := NewTraceTagsWithWriter(out)
			tags.Enable("nested")
			outer, inner, later := 0, 0, 0
			err := tags.EmitConditional("nested", "outer", "source.cpp", 1, func() bool { return true }, func(w io.Writer) error {
				outer++
				_ = tags.EmitConditional("nested", "inner", "source.cpp", 2, func() bool { return true }, func(w io.Writer) error { inner++; _, _ = io.WriteString(w, "inner operand"); return nil })
				_, _ = io.WriteString(w, "outer operand")
				return nil
			})
			_ = tags.EmitConditional("nested", "later", "source.cpp", 3, func() bool { return true }, func(w io.Writer) error { later++; _, _ = io.WriteString(w, "later operand"); return nil })
			if tags.ShouldTrace("nested") {
				if !tags.IsStreamFailure(err) || outer != 1 || inner != 1 || later != 1 || !out.failed || out.after != 0 {
					t.Fatal("nested ostream state was not shared while operand execution continued", err, outer, inner, later, out.after)
				}
			} else if err != nil || outer != 0 || inner != 0 || later != 0 || out.failed {
				t.Fatal("erased trace evaluated nested operands")
			}
		})
	}
}
