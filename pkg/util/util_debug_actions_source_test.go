// Source: src/util/debug.cpp:73-117; src/util/debug.h:42-56.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
)

const DebugSourcePrompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"

func TestDebugActionSubprocessHelper(t *testing.T) {
	route := os.Getenv("G3_DEBUG_ACTION_CHILD")
	if route == "" {
		return
	}
	b := arena.NewBudget(1 << 20)
	if route == "no-debugger" {
		action, e := NewDebugActionContext(DebugActionConfig{Budget: b, Profile: DebugNoDebugger})
		if e != nil {
			os.Exit(90)
		}
		b.FailAfter(0)
		_ = action.InvokeDebugger()
		os.Exit(91)
	}
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	if e != nil {
		os.Exit(92)
	}
	input, e := NewDebuggerCharInput(streams, os.Stdin)
	if e != nil {
		os.Exit(93)
	}
	action, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: streams, Input: input})
	if e != nil {
		os.Exit(94)
	}
	b.FailAfter(0)
	e = action.InvokeDebugger()
	if route == "throw" {
		var ex DefaultException
		var base Z3Exception
		if !errors.As(e, &ex) || !errors.As(e, &base) || ex.Message() != "assertion violation" || base.HasErrorCode() {
			os.Exit(95)
		}
		out, _ := streams.Stdout()
		_ = out.AppendCString("caught: assertion violation\n")
	} else if e != nil {
		os.Exit(96)
	} else {
		out, _ := streams.Stdout()
		_ = out.AppendCString("returned\n")
	}
	b.FailAfter(-1)
	_ = action.Close()
	_ = input.Close()
	_ = streams.Close()
	if b.Used() != 0 {
		os.Exit(97)
	}
	os.Exit(0)
}
func debugActionChild(t *testing.T, route string, data []byte, pathEmpty bool) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDebugActionSubprocessHelper$")
	file, e := os.CreateTemp(".", "debug-action-input-*")
	paramsMust(t, e)
	defer func() { file.Close(); os.Remove(file.Name()) }()
	_, e = file.Write(data)
	paramsMust(t, e)
	_, e = file.Seek(0, 0)
	paramsMust(t, e)
	cmd.Stdin = file
	cmd.Env = append(os.Environ(), "G3_DEBUG_ACTION_CHILD="+route, "GOTRACEBACK=none")
	if pathEmpty {
		cmd.Env = append(cmd.Env, "PATH=")
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	e = cmd.Run()
	if ctx.Err() != nil {
		t.Fatal("bounded child timed out", route, ctx.Err(), stderr.String())
	}
	code := 0
	if e != nil {
		var ex *exec.ExitError
		if !errors.As(e, &ex) {
			t.Fatal(e)
		}
		code = ex.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}
func TestDebugActionLiteralContinueAbortThrowEOFFlow(t *testing.T) {
	cases := []struct {
		name, input, out       string
		code, prompts, invalid int
	}{{"continue", "C", "returned\n", 0, 1, 0}, {"lowercase", "c", "returned\n", 0, 1, 0}, {"abort", " A", "", 1, 1, 0}, {"abort-lower", "a", "", 1, 1, 0}, {"throw", "T", "caught: assertion violation\n", 0, 1, 0}, {"throw-lower", "t", "caught: assertion violation\n", 0, 1, 0}, {"eof", " \t\n", "", 110, 1, 0}, {"invalid", "?C", "returned\n", 0, 2, 1}, {"classic-byte", "\x85C", "returned\n", 0, 2, 1}, {"no-debugger", "", "", 110, 0, 0}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			route := c.name
			if c.name == "throw-lower" {
				route = "throw"
			}
			out, err, code := debugActionChild(t, route, []byte(c.input), false)
			want := DebugSourcePrompt
			if c.prompts == 0 {
				want = ""
			}
			if c.invalid != 0 {
				want += "INVALID COMMAND\n" + DebugSourcePrompt
			}
			if out != c.out || err != want || code != c.code {
				t.Fatal(out, err, code, c)
			}
		})
	}
}
func TestDebugActionActualHostStopAndSystemFailure(t *testing.T) {
	if !debugHostLinux {
		t.Skip("actual Linux OS replacement proof only")
	}
	for _, c := range []struct {
		name, input string
		system      bool
	}{{"stop", "S", false}, {"stop-lower", "s", false}, {"system-failure", "G", true}, {"system-failure-lower", "g", true}} {
		t.Run(c.name, func(t *testing.T) {
			out, err, code := debugActionChild(t, c.name, []byte(c.input), c.system)
			if out != "" || code != 2 || !strings.HasPrefix(err, DebugSourcePrompt) {
				t.Fatal(out, err, code)
			}
			if !strings.Contains(err, "SIGSEGV: segmentation violation") {
				t.Fatal("real Go signal diagnostic", err)
			}
			if c.system && (!strings.Contains(err, "invoking GDB...\n") || !strings.Contains(err, "error starting GDB...\n") || strings.Contains(err, "continuing the execution...")) {
				t.Fatal("actual nonzero shell branch", err)
			}
		})
	}
}
func TestDebugActionLiteralCommandUsesActualEmbeddedStorage(t *testing.T) {
	input, streams, b, _ := numericCharInputFixture(t, []byte("C"))
	a, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: streams, Input: input})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, a.Close()) })
	before := b.Used()
	b.FailAfter(0)
	cmd, e := a.hostCommand()
	paramsMust(t, e)
	want := "gdb -nw /proc/" + strconv.Itoa(os.Getpid()) + "/exe " + strconv.Itoa(os.Getpid())
	if cmd != want || string(a.command[:len(cmd)]) != want || a.command[len(cmd)] != 0 || b.Used() != before {
		t.Fatal("source sprintf storage/actual PID/zero source event", cmd, want)
	}
}

func debugActionPhysical(t *testing.T, a *DebugActionContext, in *NumericCharInput, s *MemoryStandardStreams, b *arena.Budget) {
	t.Helper()
	want := a.header + in.header + s.header + s.stdout.Stats().OwnerBytes + s.StderrField.Stats().OwnerBytes
	if b.Used() != want {
		t.Fatal("exact action/input/standard-stream cold graph", b.Used(), want)
	}
}
func TestDebugActionConstructorEveryReservationAndBorrowedClose(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			in, s, b, _ := numericCharInputFixture(t, []byte("C"))
			before := b.Used()
			b.FailAfter(ordinal)
			a, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: in})
			if ordinal == 0 {
				if e == nil || a != nil || b.Used() != before {
					t.Fatal("sole actual header refusal", a, e, b.Used(), before)
				}
				return
			}
			paramsMust(t, e)
			debugActionPhysical(t, a, in, s, b)
			b.FailAfter(0)
			paramsMust(t, a.InvokeDebugger())
			debugActionPhysical(t, a, in, s, b)
			paramsMust(t, a.Close())
			if b.Used() != before {
				t.Fatal("action header refund", b.Used(), before)
			}
			state, e := in.State()
			paramsMust(t, e)
			if state.Position != 1 {
				t.Fatal(state)
			}
			_, e = s.Stderr()
			paramsMust(t, e)
		})
	}
}
func TestDebugActionForeignBudgetCopyClosedAndUnreadProviders(t *testing.T) {
	in, s, b, _ := numericCharInputFixture(t, []byte("C"))
	foreign := arena.NewBudget(1 << 20)
	foreign.FailAfter(0)
	a, e := NewDebugActionContext(DebugActionConfig{Budget: foreign, Streams: s, Input: in})
	if !errors.Is(e, arena.ErrConfig) || a != nil || foreign.Used() != 0 {
		t.Fatal(a, e, foreign.Used())
	}
	a, e = NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: in})
	paramsMust(t, e)
	copy := *a
	if !errors.Is(copy.InvokeDebugger(), ErrDebugActionOwner) || !errors.Is(copy.Close(), ErrDebugActionOwner) {
		t.Fatal("shallow copy guard")
	}
	paramsMust(t, a.Close())
	if !errors.Is(a.InvokeDebugger(), arena.ErrClosed) {
		t.Fatal("closed own guard")
	}
	// Configuration borrows identity only; closed pointed lifetimes remain unread.
	paramsMust(t, in.Close())
	paramsMust(t, s.Close())
	a, e = NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: in})
	paramsMust(t, e)
	if !errors.Is(a.InvokeDebugger(), arena.ErrClosed) {
		t.Fatal("actual prompt stream guard")
	}
	paramsMust(t, a.Close())
}
func TestDebugActionClosedOutputBadbitStillReachesInputAndThrow(t *testing.T) {
	in, s, b, _ := numericCharInputFixture(t, []byte("T"))
	a, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: in})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, a.Close()) })
	paramsMust(t, s.StderrField.File.Close())
	before := b.Used()
	b.FailAfter(0)
	e = a.InvokeDebugger()
	var ex DefaultException
	if !errors.As(e, &ex) || ex.Message() != "assertion violation" {
		t.Fatal(e)
	}
	state, e := in.State()
	paramsMust(t, e)
	if state.Position != 1 || !s.StderrField.FileBad {
		t.Fatal("badbit did not preserve reached input", state, s.StderrField.State())
	}
	if b.Used() != before {
		t.Fatal("action allocated")
	}
	debugActionPhysical(t, a, in, s, b)
}
func TestDebugActionPromptBeforeClosedTieRead(t *testing.T) {
	in, s, b, _ := numericCharInputFixture(t, []byte("C"))
	a, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: in})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, a.Close()) })
	paramsMust(t, s.stdout.Close())
	e = a.InvokeDebugger()
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	state, e := in.State()
	paramsMust(t, e)
	if state.Position != 0 {
		t.Fatal("closed tie read input", state)
	}
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	if string(data) != DebugSourcePrompt {
		t.Fatal("prompt prefix", string(data))
	}
}
