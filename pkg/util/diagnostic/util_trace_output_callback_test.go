package diagnostic

import (
	"io"
	"testing"
)

type traceOutputCallbackProbe struct {
	t                        *testing.T
	tags                     *TraceTags
	writes, strings, flushes int
	panicOnWrite             bool
}

func (w *traceOutputCallbackProbe) check() {
	if !w.tags.OutputCallbackActive() {
		w.t.Error("host trace callback lacks its executing scope")
	}
	if w.panicOnWrite {
		panic("host output")
	}
}
func (w *traceOutputCallbackProbe) Write(p []byte) (int, error) {
	w.check()
	w.writes++
	return len(p), nil
}
func (w *traceOutputCallbackProbe) WriteString(s string) (int, error) {
	w.check()
	w.strings++
	return len(s), nil
}
func (w *traceOutputCallbackProbe) Flush() error { w.check(); w.flushes++; return nil }

func TestTraceOutputCallbacksExcludeBodyAndRestoreAfterPanic(t *testing.T) {
	w := traceOutputCallbackProbe{t: t}
	tags := NewTraceTagsWithWriter(&w)
	w.tags = tags
	tags.Enable("callback_test")
	if !tags.ShouldTrace("callback_test") {
		t.Skip("source trace macro erased")
	}
	e := tags.EmitConditional("callback_test", "probe", "probe.cpp", 1, func() bool { return true }, func(out io.Writer) error {
		if tags.OutputCallbackActive() {
			t.Error("ordinary trace body classified as host callback")
		}
		if _, e := out.Write([]byte("bytes")); e != nil {
			return e
		}
		if _, e := io.WriteString(out, "string"); e != nil {
			return e
		}
		return WriteTraceEndl(out)
	})
	if e != nil || tags.OutputCallbackActive() || w.writes == 0 || w.strings == 0 || w.flushes == 0 {
		t.Fatal("trace callback lifetime", e, w.writes, w.strings, w.flushes)
	}
	w.panicOnWrite = true
	func() {
		defer func() {
			if recover() == nil {
				t.Error("host panic not propagated")
			}
		}()
		_ = tags.Emit("callback_test", "probe", "probe.cpp", 2, "panic")
	}()
	if tags.OutputCallbackActive() {
		t.Fatal("panic retained trace callback scope")
	}
}
