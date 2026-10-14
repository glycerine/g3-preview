package diagnostic

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
)

type flushedWarningWriter struct {
	bytes.Buffer
	flushes int
	err     error
}

func (w *flushedWarningWriter) Flush() error {
	w.flushes++
	return w.err
}

func TestWarningFlushesCompletedBorrowedAndDefaultStreams(t *testing.T) {
	var output bytes.Buffer
	buffered := bufio.NewWriter(&output)
	s := NewWarningService(buffered)
	if err := s.Warning("default"); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "WARNING: default\n" {
		t.Fatal("default buffered warning was not flushed", got)
	}

	flushed := &flushedWarningWriter{}
	if err := s.SetWarningStream(flushed); err != nil {
		t.Fatal(err)
	}
	if err := s.Warningf("selected %d", 7); err != nil {
		t.Fatal(err)
	}
	if flushed.String() != "WARNING: selected 7\n" || flushed.flushes != 1 {
		t.Fatal("selected stream flush", flushed.String(), flushed.flushes)
	}
	flushErr := errors.New("flush failed")
	flushed.err = flushErr
	if err := s.Warning("again"); err != nil || flushed.flushes != 2 {
		t.Fatal("completed warning flush failure", err, flushed.flushes)
	}
	if err := s.Enable(false); err != nil {
		t.Fatal(err)
	}
	if err := s.Warning("disabled"); err != nil || flushed.flushes != 2 {
		t.Fatal("disabled warning flushed", err, flushed.flushes)
	}
}

func TestWarningShortWriteStillFlushesSourceStream(t *testing.T) {
	w := &shortFlushingWarningWriter{shortWarningWriter: shortWarningWriter{limit: len("WARNING: x")}}
	s := NewWarningService(w)
	if err := s.Warning("xy"); err != nil || w.flushes != 1 || w.buf.String() != "WARNING: x" {
		t.Fatal("short write source flush and prefix", err, w.flushes, w.buf.String())
	}
	flushErr := errors.New("flush failed")
	w = &shortFlushingWarningWriter{shortWarningWriter: shortWarningWriter{limit: 0}, err: flushErr}
	s = NewWarningService(w)
	if err := s.Warning("xy"); err != nil || w.flushes != 1 {
		t.Fatal("first insertion error precedes later flush failure", err, w.flushes)
	}
}

func TestWarningBorrowedStreamFailurePersistsAcrossCalls(t *testing.T) {
	w := &shortFlushingWarningWriter{shortWarningWriter: shortWarningWriter{limit: len("WARNING: x")}}
	s := NewWarningService(&bytes.Buffer{})
	if e := s.SetWarningStream(w); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("xy"); e != nil {
		t.Fatal(e)
	}
	w.limit = 100
	if e := s.Warning("later"); e != nil {
		t.Fatal(e)
	}
	if got := w.buf.String(); got != "WARNING: x" || w.flushes != 2 {
		t.Fatalf("failed ostream accepted a later insertion: %q, flushes %d", got, w.flushes)
	}
	if e := s.SetWarningStream(w); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("same"); e != nil || w.buf.String() != "WARNING: x" {
		t.Fatalf("rebinding the same failed ostream cleared its state: %q, %v", w.buf.String(), e)
	}
	var next bytes.Buffer
	if e := s.SetWarningStream(&next); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("new"); e != nil || next.String() != "WARNING: new\n" {
		t.Fatalf("new ostream inherited the old failed state: %q, %v", next.String(), e)
	}
}

func TestWarningBorrowedFlushFailurePersistsAcrossCalls(t *testing.T) {
	flushErr := errors.New("flush refused")
	w := &flushedWarningWriter{err: flushErr}
	s := NewWarningService(&bytes.Buffer{})
	if e := s.SetWarningStream(w); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("first"); e != nil {
		t.Fatal(e)
	}
	w.err = nil
	if e := s.Warning("second"); e != nil {
		t.Fatal(e)
	}
	if got := w.String(); got != "WARNING: first\n" || w.flushes != 2 {
		t.Fatalf("failed flush allowed later insertion: %q, flushes %d", got, w.flushes)
	}
}

func TestWarningTypedNilStreamSelectsDefaultRoute(t *testing.T) {
	var defaultOutput bytes.Buffer
	s := NewWarningService(&defaultOutput)
	var absent *bytes.Buffer
	if e := s.SetWarningStream(absent); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("default"); e != nil {
		t.Fatal(e)
	}
	if got := defaultOutput.String(); got != "WARNING: default\n" {
		t.Fatalf("null stream pointer did not select the default route: %q", got)
	}
}

type shortFlushingWarningWriter struct {
	shortWarningWriter
	flushes int
	err     error
}

func (w *shortFlushingWarningWriter) Flush() error {
	w.flushes++
	return w.err
}

type shortWarningWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *shortWarningWriter) Write(p []byte) (int, error) {
	available := w.limit - w.buf.Len()
	if available < len(p) {
		if available < 0 {
			available = 0
		}
		return w.buf.Write(p[:available])
	}
	return w.buf.Write(p)
}

func TestWarningShortWritePreservesSourceInsertionPrefix(t *testing.T) {
	const output = "WARNING: xy\n"
	for limit := 0; limit <= len(output); limit++ {
		w := &shortWarningWriter{limit: limit}
		s := NewWarningService(&w.buf)
		if err := s.SetWarningStream(w); err != nil {
			t.Fatal(err)
		}
		err := s.Warning("xy")
		if got := w.buf.String(); got != output[:limit] {
			t.Fatalf("limit %d: prefix %q", limit, got)
		}
		if err != nil {
			t.Fatalf("limit %d: warning stream failure escaped: %v", limit, err)
		}
	}
}

func TestWarningSourceDefaultRoutingAndDisabledState(t *testing.T) {
	var stderr bytes.Buffer
	s := NewWarningService(&stderr)
	if !s.Enabled() {
		t.Fatal("source warning default")
	}
	if e := s.Warning("first\x00ignored"); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: first\n" {
		t.Fatal(got)
	}
	if e := s.Enable(false); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("suppressed"); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: first\n" {
		t.Fatal(got)
	}
}

func TestWarningSourceBorrowedStreamTruncationAndFallback(t *testing.T) {
	var stderr, diagnostic bytes.Buffer
	s := NewWarningService(&stderr)
	if e := s.SetWarningStream(&diagnostic); e != nil {
		t.Fatal(e)
	}
	long := strings.Repeat("x", 1100)
	if e := s.Warning(long); e != nil {
		t.Fatal(e)
	}
	if got := diagnostic.String(); got != "WARNING: "+long[:1023]+"\n" {
		t.Fatal("source 1024-byte ostream buffer", len(got))
	}
	if e := s.SetWarningStream(nil); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning(long); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: "+long+"\n" {
		t.Fatal("default FILE route must not truncate", len(got))
	}
}

func TestWarningFormattedStringAndIntegerRoutes(t *testing.T) {
	var stderr, diagnostic bytes.Buffer
	s := NewWarningService(&stderr)
	args := []any{"left\x00ignored", 17}
	if e := s.Warningf("value %s at %d%%\x00ignored", args...); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: value left at 17%\n" || args[0] != "left\x00ignored" {
		t.Fatal(got, args)
	}
	if e := s.SetWarningStream(&diagnostic); e != nil {
		t.Fatal(e)
	}
	long := strings.Repeat("x", 1100)
	if e := s.Warningf("%s-%d", long, 3); e != nil {
		t.Fatal(e)
	}
	if got := diagnostic.String(); got != "WARNING: "+long[:1023]+"\n" {
		t.Fatal("formatted stream truncation", len(got))
	}
	if e := s.Enable(false); e != nil {
		t.Fatal(e)
	}
	if e := s.Warningf("suppressed %d", 2); e != nil || stderr.String() != "WARNING: value left at 17%\n" {
		t.Fatal(e, stderr.String())
	}
}

func TestWarningRetainedDatalogUnsignedSymbol(t *testing.T) {
	var stderr, stream bytes.Buffer
	s := NewWarningService(&stderr)
	format := "symbol number %I64u on line %d in file %s does not belong to sort %s"
	if e := s.Warningf(format, ^uint64(0), 7, "facts.rel", "Color"); e != nil {
		t.Fatal(e)
	}
	want := "WARNING: symbol number 18446744073709551615 on line 7 in file facts.rel does not belong to sort Color\n"
	if got := stderr.String(); got != want {
		t.Fatalf("default stream = %q, want %q", got, want)
	}
	if e := s.SetWarningStream(&stream); e != nil {
		t.Fatal(e)
	}
	if e := s.Warningf("%%I64u: %I64u", uint64(42)); e != nil {
		t.Fatal(e)
	}
	if got, want := stream.String(), "WARNING: %I64u: 42\n"; got != want {
		t.Fatalf("borrowed stream = %q, want %q", got, want)
	}
}
