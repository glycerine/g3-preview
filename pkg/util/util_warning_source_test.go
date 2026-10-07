// Source: src/util/warning.cpp:63-89,109-169; src/util/warning.h:24-57.
// Source author: Leonardo de Moura (2006-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func warningFixture(t *testing.T) (*WarningContext, *Runtime, *MemoryStandardStreams, *os.File, *arena.Budget) {
	t.Helper()
	f, e := os.CreateTemp(".", "warning-source-*")
	serviceMust(t, e)
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	serviceMust(t, e)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: f, Stderr: f})
	serviceMust(t, e)
	w, e := NewWarningContext(WarningContextConfig{Budget: b, Runtime: r, Streams: streams})
	serviceMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		serviceMust(t, w.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, r.Close())
		f.Close()
		os.Remove(f.Name())
		if b.Used() != 0 {
			t.Fatal("cold leak", b.Used())
		}
	})
	return w, r, streams, f, b
}
func TestWarningSourceStdioUntruncatedAndOstreamAMD64Truncation(t *testing.T) {
	w, _, _, f, b := warningFixture(t)
	msg := strings.Repeat("x", 1100) + "\x00unread"
	serviceMust(t, w.WarnLiteral(msg))
	_, e := f.Seek(0, io.SeekStart)
	serviceMust(t, e)
	v, e := io.ReadAll(f)
	serviceMust(t, e)
	if string(v) != "WARNING: "+strings.Repeat("x", 1100)+"\n" {
		t.Fatal("stdio truncated", len(v))
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	serviceMust(t, e)
	defer out.Close()
	serviceMust(t, out.SetFill('_'))
	serviceMust(t, out.SetWidth(12))
	serviceMust(t, w.SetWarningStream(out))
	serviceMust(t, w.WarnLiteral(msg))
	got, e := out.String()
	serviceMust(t, e)
	if got != "___WARNING: "+strings.Repeat("x", 1023)+"\n" {
		t.Fatal("ostream prefix/format truncation", len(got))
	}
}
func TestWarningSourceDisabledErasesBorrowAndClosedFileKeepsArgumentReads(t *testing.T) {
	w, _, streams, _, b := warningFixture(t)
	serviceMust(t, w.EnableWarningMessages(false))
	b.FailAfter(0)
	serviceMust(t, w.WarnCString(nil, GlobalParamCString{}))
	b.FailAfter(-1)
	out, e := streams.Stderr()
	serviceMust(t, e)
	serviceMust(t, w.SetWarningStream(out))
	serviceMust(t, w.EnableWarningMessages(true))
	serviceMust(t, out.File.Close())
	if e = w.WarnCString(nil, GlobalParamCString{}); e == nil {
		t.Fatal("printf borrowed argument erased by badbit")
	}
	bad, e := out.Bad()
	serviceMust(t, e)
	if !bad {
		t.Fatal("prefix not reached")
	}
	serviceMust(t, w.WarnLiteral("safe"))
}
func TestWarningSourceEveryFormattingReservationAndExactOwnCleanup(t *testing.T) {
	first := -1
	for ordinal := 0; ordinal < 100; ordinal++ {
		w, r, _, _, b := warningFixture(t)
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		serviceMust(t, e)
		serviceMust(t, w.SetWarningStream(out))
		b.FailAfter(int64(ordinal))
		e = w.WarnLiteral("message")
		st := out.Stats()
		warningRawGraph(t, w, r, b, w.streams.header+w.streams.stdout.Stats().OwnerBytes+w.streams.StderrField.Stats().OwnerBytes+st.OwnerBytes+st.ScratchBytes+st.OutputOwnerBytes+st.OutputOverflowBytes)
		if b.Used() != r.Stats().TotalBytes+w.Stats().OwnerBytes+w.Stats().FrameBytes+uint64(w.streams.header)+w.streams.stdout.Stats().OwnerBytes+w.streams.StderrField.Stats().OwnerBytes+st.OwnerBytes+st.ScratchBytes+st.OutputOwnerBytes+st.OutputOverflowBytes {
			t.Fatal("immediate cold graph", ordinal, b.Used(), w.Stats(), r.Stats(), st)
		}
		if e == nil {
			first = ordinal
		} else if !errors.Is(e, arena.ErrInjectedFailure) && !errors.Is(e, Failure{Kind: FailureMemout}) {
			t.Fatal(ordinal, e)
		}
		b.FailAfter(-1)
		serviceMust(t, w.Close())
		serviceMust(t, out.Close())
		if first >= 0 {
			break
		}
	}
	t.Log("actual first-success formatting horizon", first)
	if first < 0 {
		t.Fatal("no success horizon")
	}
}

func warningRawGraph(t *testing.T, w *WarningContext, r *Runtime, b *arena.Budget, extra uint64, anchors ...Block) {
	t.Helper()
	named := map[Block]bool{}
	add := func(h Block) {
		if h == (Block{}) {
			return
		}
		if named[h] {
			t.Fatal("duplicate actual formatting block", h)
		}
		named[h] = true
		_, e := r.Size(h)
		serviceMust(t, e)
	}
	for _, h := range anchors {
		add(h)
	}
	for f := w.frames; f != nil; f = f.next {
		if !f.column.PendingReplaces {
			add(f.column.Block)
		} else if _, e := r.Size(f.column.Block); e == nil {
			t.Fatal("old realloc pointer unexpectedly live")
		}
		add(f.column.Pending)
	}
	if r.Stats().LiveBlocks != uint32(len(named)) || b.Used() != r.Stats().TotalBytes+w.Stats().OwnerBytes+w.Stats().FrameBytes+extra {
		t.Fatal("named raw/cold graph", r.Stats(), len(named), b.Used(), w.Stats(), extra)
	}
}
func TestWarningSourceBorrowedCStringFullLengthAndStreamSelectionLifetime(t *testing.T) {
	w, _, streams, f, b := warningFixture(t)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	defer region.Close()
	p := debugNotificationCString(t, region, strings.Repeat("p", 1100)+"%literal")
	serviceMust(t, w.SendWarningsToStdout(true))
	serviceMust(t, w.WarnCString(region, p))
	_, e = f.Seek(0, io.SeekStart)
	serviceMust(t, e)
	v, e := io.ReadAll(f)
	serviceMust(t, e)
	if string(v) != "WARNING: "+strings.Repeat("p", 1100)+"%literal\n" {
		t.Fatal("actual CString stdio", len(v))
	}
	out, e := streams.Stderr()
	serviceMust(t, e)
	serviceMust(t, w.SetWarningStream(out))
	serviceMust(t, out.Close())
	serviceMust(t, w.SetWarningStream(out)) // setter borrows, never queries pointed lifetime.
	if e = w.WarnCString(region, p); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("reached prefix lifetime", e)
	}
	serviceMust(t, w.EnableWarningMessages(false))
	serviceMust(t, w.WarnCString(nil, GlobalParamCString{}))
}
func TestWarningSourcePostphysicalFailedFrameSurvivesValidRetryAndCallback(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	var w *WarningContext
	var callbackErr error
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, Transport: func(f Failure) error { callbackErr = w.Close(); return f }})
	serviceMust(t, e)
	file, e := os.CreateTemp(".", "warning-oom-*")
	serviceMust(t, e)
	defer os.Remove(file.Name())
	defer file.Close()
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: file, Stderr: file})
	serviceMust(t, e)
	out, e := streams.Stderr()
	serviceMust(t, e)
	w, e = NewWarningContext(WarningContextConfig{Budget: b, Runtime: r, Streams: streams})
	serviceMust(t, e)
	serviceMust(t, w.SetWarningStream(out))
	anchor, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	e = w.WarnLiteral("first")
	if !errors.Is(e, Failure{Kind: FailureMemout}) || !errors.Is(callbackErr, ErrWarningExecuting) {
		t.Fatal("actual transport prefix", e, callbackErr)
	}
	old := w.frames
	body := old.bodyCause
	pending := old.column.Pending
	if pending == (Block{}) || old.column.Block != (Block{}) || !old.sourceDone {
		t.Fatal("unfinished acquire is not source m_data", old)
	}
	extra := streams.header + streams.stdout.Stats().OwnerBytes + streams.StderrField.Stats().OwnerBytes
	warningRawGraph(t, w, r, b, extra, anchor)
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, w.WarnLiteral("second"))
	warningRawGraph(t, w, r, b, extra, anchor)
	if w.frames != old || old.column.Pending != pending || old.bodyCause != body || w.bodyCause != body {
		t.Fatal("ordinary retry overwrote earlier acquired frame")
	}
	serviceMust(t, w.Close())
	if old.bodyCause != body {
		t.Fatal("own cleanup erased cause")
	}
	serviceMust(t, streams.Close())
	if _, e = file.Write([]byte("caller")); e != nil {
		t.Fatal("borrowed fd closed", e)
	}
	serviceMust(t, r.Deallocate(&anchor))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestWarningSourceHeaderPrefixAndBorrowedOwnerGuards(t *testing.T) {
	w, r, streams, _, b := warningFixture(t)
	before := b.Used()
	b.FailAfter(0)
	if x, e := NewWarningContext(WarningContextConfig{Budget: b, Runtime: r, Streams: streams}); x != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("actual header prefix", x, e, b.Used(), before)
	}
	copied := *w
	if e := copied.EnableWarningMessages(false); !errors.Is(e, ErrWarningOwner) {
		t.Fatal(e)
	}
	if e := copied.Close(); !errors.Is(e, ErrWarningOwner) {
		t.Fatal(e)
	}
	if b.Used() != before || !w.enabled {
		t.Fatal("copy mutated real owner")
	}
	b.FailAfter(-1)
	foreign := arena.NewBudget(1 << 20)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: foreign})
	serviceMust(t, e)
	defer out.Close()
	if e = w.SetWarningStream(out); !errors.Is(e, arena.ErrConfig) || w.warningStream != nil {
		t.Fatal("foreign stream binding", e)
	}
	serviceMust(t, streams.StderrField.SetWidth(50))
	b.FailAfter(0)
	serviceMust(t, w.WarnLiteral("file"))
	if streams.StderrField.State().Width != 50 || b.Used() != before {
		t.Fatal("stdio incorrectly uses ostream formatting/storage", streams.StderrField.State(), b.Used())
	}
	b.FailAfter(-1)
}
func TestWarningSourceSamePrintfParserBorrowedPrecisionAndRawErasedWrites(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	formatter, e := NewPrintfFormatter(PrintfConfig{Budget: b})
	serviceMust(t, e)
	got, e := formatter.Format("%.*s", []PrintfArg{PrintfInt32(0), PrintfCStringView(ParamsStringView{})})
	serviceMust(t, e)
	if got != "" {
		t.Fatal(got)
	}
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	p := debugNotificationCString(t, region, "real%borrow")
	v := ParamsStringView{region: region, regionRef: p.Ref, offset: p.Offset, PresentField: true, Mode: 3}
	got, e = formatter.Format("%13.4s", []PrintfArg{PrintfCStringView(v)})
	serviceMust(t, e)
	if got != "         real" {
		t.Fatal(got)
	}
	serviceMust(t, region.Close())
	if _, e = formatter.Format("%s", []PrintfArg{PrintfCStringView(v)}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("stale actual borrowed reader", e)
	}
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	serviceMust(t, e)
	serviceMust(t, r.Close())
	count, _, e := pfRenderSource(pfSink{runtime: r, extent: 0}, PrintfNonWindowsAMD64, "literal", nil)
	serviceMust(t, e)
	if count != 7 {
		t.Fatal("size0 erased provider writes", count)
	}
	serviceMust(t, formatter.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
