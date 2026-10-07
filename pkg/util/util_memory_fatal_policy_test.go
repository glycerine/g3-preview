// Source: src/util/memory_manager.cpp:33-72; src/util/memory_manager.h:49-71.
// Source: src/util/error_codes.h; src/util/z3_exception.h.
package util

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMemoryFatalStandardStreamsColdGraph(t *testing.T) {
	f, e := os.CreateTemp(".", "fatal-stdout-*")
	smallObjectMust(t, e)
	defer os.Remove(f.Name())
	defer f.Close()
	b := arena.NewBudget(1 << 20)
	for fault := int64(0); fault <= 3; fault++ {
		b.FailAfter(fault)
		s, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: f, Stderr: f})
		if fault < 3 {
			if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != 0 {
				t.Fatal(fault, s, e, b.Used())
			}
			continue
		}
		smallObjectMust(t, e)
		expected := uint64(reflect.TypeOf(MemoryStandardStreams{}).Size()) + 2*uint64(reflect.TypeOf(NumericTextStream{}).Size())
		if b.Used() != expected || s.stdout.Buffer != nil || s.StderrField.Buffer != nil {
			t.Fatal("actual cold provider graph", b.Used(), expected)
		}
		b.FailAfter(-1)
		smallObjectMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	if _, e = f.Write([]byte("still borrowed")); e != nil {
		t.Fatal(e)
	}
}
func TestMemoryFatalNonExitExceptionCatch(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	smallObjectMust(t, e)
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: streams})
	smallObjectMust(t, e)
	r, e := NewRuntime(RuntimeConfig{Budget: b, FatalPolicy: c})
	smallObjectMust(t, e)
	used := b.Used()
	b.FailAfter(0)
	h, e := r.Allocate(1)
	var derived OutOfMemoryError
	var base Z3Error
	if !errors.As(e, &derived) || !errors.As(e, &base) || base.ErrorCode() != ErrorMemout || !r.State().OutOfMemory || h != (Block{}) || b.Used() != used {
		t.Fatal(h, e, r.State(), b.Used(), used)
	}
	b.FailAfter(-1)
	smallObjectMust(t, r.Close())
	smallObjectMust(t, c.Close())
	smallObjectMust(t, streams.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestMemoryFatalSubprocessChild(t *testing.T) {
	route := os.Getenv("G3_MEMORY_FATAL_CHILD")
	if route == "" {
		return
	}
	b := arena.NewBudget(1 << 20)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	smallObjectMust(t, e)
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: streams})
	smallObjectMust(t, e)
	r, e := NewRuntime(RuntimeConfig{Budget: b, FatalPolicy: c})
	smallObjectMust(t, e)
	defer os.Stdout.WriteString("UNWOUND")
	var region *arena.BytesRegion
	audit := func() {
		want := r.Stats().TotalBytes + c.header + streams.header + streams.stdout.Stats().OwnerBytes + streams.StderrField.Stats().OwnerBytes
		if region != nil {
			rs := region.Stats()
			ds := rs.Descriptors
			want += rs.OwnerBytes + rs.PayloadBytes + rs.PageHeaderBytes + rs.DirectoryBytes + rs.LogBytes + rs.ScopeBytes + ds.OwnerBytes + ds.RecordBytes + ds.MetadataBytes + ds.DirectoryBytes
		}
		if b.Used() != want || streams.stdout.Buffer != nil || streams.StderrField.Buffer != nil {
			t.Fatal("named cold/raw graph before refusal", b.Used(), want)
		}
	}
	switch route {
	case "count", "count-exit", "count-bad":
		smallObjectMust(t, c.ExitWhenOutOfMemory(route == "count-exit", nil, GlobalParamCString{}))
		smallObjectMust(t, r.SetMaxCount(1))
		_, e = r.Allocate(0)
		smallObjectMust(t, e)
		if route == "count-bad" {
			smallObjectMust(t, os.Stdout.Close())
		}
		audit()
		b.FailAfter(0)
		_, e = r.Allocate(0)
	case "oom", "oom-bad", "oom-custom":
		smallObjectMust(t, c.ExitWhenOutOfMemory(true, nil, GlobalParamCString{}))
		if route == "oom-custom" {
			region, e = arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
			smallObjectMust(t, e)
			msg := smallObjectID(t, region, "old")
			smallObjectMust(t, c.ExitWhenOutOfMemory(true, region, msg))
			smallObjectMust(t, region.Write(msg.Ref, 0, 'n'))
			smallObjectMust(t, region.Write(msg.Ref, 1, 'e'))
			smallObjectMust(t, region.Write(msg.Ref, 2, 'w'))
			smallObjectMust(t, c.ExitWhenOutOfMemory(false, nil, GlobalParamCString{}))
			smallObjectMust(t, c.ExitWhenOutOfMemory(true, nil, GlobalParamCString{}))
		}
		if route == "oom-bad" {
			smallObjectMust(t, os.Stderr.Close())
		}
		audit()
		b.FailAfter(0)
		_, e = r.Allocate(8)
	default:
		t.Fatal(route)
	}
	t.Fatal("fatal returned", e)
}
func TestMemoryFatalRealExitWithActiveReservationRefusal(t *testing.T) {
	exe, e := os.Executable()
	smallObjectMust(t, e)
	for _, route := range []string{"count", "count-exit", "count-bad", "oom", "oom-bad", "oom-custom"} {
		t.Run(route, func(t *testing.T) {
			cmd := exec.Command(exe, "-test.run=^TestMemoryFatalSubprocessChild$")
			cmd.Env = append(os.Environ(), "G3_MEMORY_FATAL_CHILD="+route)
			var out, errout bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &errout
			e := cmd.Run()
			var exit *exec.ExitError
			code := 101
			stdout := ""
			stderr := "ERROR: out of memory\n"
			if route == "count" || route == "count-exit" || route == "count-bad" {
				code = 113
				stdout = "Maximal allocation counts 1 have been exceeded\n"
				stderr = ""
			}
			if route == "count-bad" {
				stdout = ""
			}
			if route == "oom-custom" {
				stderr = "new\n"
			}
			if route == "oom-bad" {
				stderr = ""
			}
			if !errors.As(e, &exit) || exit.ExitCode() != code || out.String() != stdout || errout.String() != stderr {
				t.Fatal(route, e, out.String(), errout.String())
			}
		})
	}
}

func TestMemoryFatalBorrowedPointerRetentionAndReachedSentry(t *testing.T) {
	f, s, b := numericFileFixture(t)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
	smallObjectMust(t, e)
	defer region.Close()
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: f, Stderr: f})
	smallObjectMust(t, e)
	defer streams.Close()
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: streams})
	smallObjectMust(t, e)
	defer c.Close()
	text := smallObjectID(t, region, "borrowed")
	smallObjectMust(t, c.ExitWhenOutOfMemory(true, region, text))
	smallObjectMust(t, c.ExitWhenOutOfMemory(false, nil, GlobalParamCString{}))
	if c.exitOOM || c.region != region || c.message != text || !c.borrowed {
		t.Fatal("false changed old pointer")
	}
	smallObjectMust(t, c.ExitWhenOutOfMemory(true, nil, GlobalParamCString{}))
	if !c.exitOOM || c.region != region || c.message != text {
		t.Fatal("null changed old pointer")
	}
	// Closed file sets nonthrowing badbit. CString strlen still precedes sentry.
	smallObjectMust(t, f.Close())
	smallObjectMust(t, s.AppendByte('x'))
	smallObjectMust(t, s.SetWidth(9))
	smallObjectMust(t, region.Reset())
	before := b.Used()
	if e = smallObjectCStringInto(s, region, text); e == nil {
		t.Fatal("badbit skipped reached stale C-string length")
	}
	if s.State().Width != 9 || b.Used() != before {
		t.Fatal("failed length advanced sentry", s.State(), b.Used(), before)
	}
	// Setter stores a stale pointer without any eager pointee read.
	smallObjectMust(t, c.ExitWhenOutOfMemory(true, region, text))
	copy := *c
	if e = copy.ExitWhenOutOfMemory(false, nil, GlobalParamCString{}); !errors.Is(e, ErrNumericStreamOwner) {
		t.Fatal(e)
	}
	smallObjectMust(t, c.Close())
	if e = c.ExitWhenOutOfMemory(false, nil, GlobalParamCString{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestMemoryFatalReachedMaxCountReadAfterOutput(t *testing.T) {
	f, _, b := numericFileFixture(t)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: f, Stderr: f})
	smallObjectMust(t, e)
	defer streams.Close()
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: streams})
	smallObjectMust(t, e)
	defer c.Close()
	r, e := NewRuntime(RuntimeConfig{Budget: b, FatalPolicy: c})
	smallObjectMust(t, e)
	smallObjectMust(t, r.Close())
	// Checked invalid-owner continuation: source count getter is reached only
	// AFTER the first insertion. This is not a valid allocation-limit execution.
	before := b.Used()
	e = c.HandleMemoryFailure(r, Failure{Kind: FailureAllocExceeded})
	if !errors.Is(e, arena.ErrClosed) || b.Used() != before {
		t.Fatal(e, b.Used(), before)
	}
	_, e = f.Seek(0, 0)
	smallObjectMust(t, e)
	got, e := os.ReadFile(f.Name())
	smallObjectMust(t, e)
	if string(got) != "Maximal allocation counts " {
		t.Fatal("eager count read", string(got))
	}
}
func TestMemoryFatalContextActualColdRefusalAndCohort(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	smallObjectMust(t, e)
	defer s.Close()
	before := b.Used()
	b.FailAfter(0)
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: s})
	if !errors.Is(e, arena.ErrInjectedFailure) || c != nil || b.Used() != before {
		t.Fatal(c, e, b.Used(), before)
	}
	b.FailAfter(-1)
	c, e = NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: s})
	smallObjectMust(t, e)
	defer c.Close()
	want := before + uint64(reflect.TypeOf(MemoryFatalContext{}).Size())
	if b.Used() != want {
		t.Fatal(b.Used(), want)
	}
	foreign := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: foreign, FatalPolicy: c})
	if !errors.Is(e, arena.ErrConfig) || r != nil || foreign.Used() != 0 {
		t.Fatal(r, e, foreign.Used())
	}
	r, e = NewRuntime(RuntimeConfig{Budget: b, FatalPolicy: c, Transport: func(f Failure) error { return f }})
	if !errors.Is(e, arena.ErrConfig) || r != nil || b.Used() != want {
		t.Fatal("two failure transports", r, e, b.Used(), want)
	}
}
func TestMemoryFatalMainRuntimeBinding(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	smallObjectMust(t, e)
	defer s.Close()
	c, e := NewMemoryFatalContext(MemoryFatalContextConfig{Budget: b, Streams: s})
	smallObjectMust(t, e)
	defer c.Close()
	r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, FatalPolicy: c})
	smallObjectMust(t, e)
	defer r.Close()
	before := b.Used()
	b.FailAfter(0)
	_, e = r.Allocate(8)
	var out OutOfMemoryError
	if !errors.As(e, &out) || !r.State().OutOfMemory || b.Used() != before {
		t.Fatal(e, r.State(), b.Used(), before)
	}
	b.FailAfter(-1)
}
