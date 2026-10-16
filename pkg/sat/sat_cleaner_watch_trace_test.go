package sat

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATCleanerWatchSourceTracesKeepAndDrop(t *testing.T) {
	c, l := satCleanerFixture(t, 2)
	first, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	second, err := l.propagation.variables.watches.Get(3)
	satOK(t, err)
	w, err := newSATBinaryWatch(2, false)
	satOK(t, err)
	satOK(t, first.Push(w))
	w, err = newSATBinaryWatch(0, false)
	satOK(t, err)
	satOK(t, second.Push(w))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	l.traceTags = tags
	satOK(t, c.cleanupWatches())
	if output.Len() != 0 {
		t.Fatalf("disabled cleaner trace emitted %q", output.String())
	}
	tags.Enable("cleanup_bug")
	satOK(t, c.cleanupWatches())
	record := func(line int, body string) string {
		return "-------- [cleanup_bug] cleanup_watches src/sat/sat_cleaner.cpp:" +
			strconv.Itoa(line) + " ---------\n" + body +
			"------------------------------------------------\n"
	}
	want := ""
	if tags.ShouldTrace("cleanup_bug") {
		want = record(48, "processing wlist of 0\n") +
			record(48, "processing wlist of -0\n") +
			record(61, "keeping: 0 1\n") +
			record(48, "processing wlist of 1\n") +
			record(48, "processing wlist of -1\n") +
			record(61, "keeping: 1 0\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("unassigned watch traces: got %q, want %q", got, want)
	}
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.EnableAssertions(false))
	satOK(t, l.propagation.assign(0, satJustification{}))
	output.Reset()
	satOK(t, c.cleanupWatches())
	want = ""
	if tags.ShouldTrace("cleanup_bug") {
		want = record(48, "processing wlist of 1\n") +
			record(48, "processing wlist of -1\n") +
			record(61, "keeping: 1 0\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("dropped binary still traces source keeping: got %q, want %q", got, want)
	}
	size, err := second.Size()
	satOK(t, err)
	if size != 0 {
		t.Fatalf("satisfied binary watch was retained: %d", size)
	}
}
