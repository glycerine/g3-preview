// Source: src/util/debug.h:60-65; src/util/permutation.cpp:54,66-75; src/util/vector.h:303-310.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
)

func permutationActionFixture(t *testing.T, text string) (*RuntimePermutation, *DebugContext, *DebugActionContext, *NumericCharInput, *MemoryStandardStreams) {
	t.Helper()
	p, d, _, b := permutationDiagnosticFixture(t, 2)
	inputFile, e := os.CreateTemp(".", "permutation-action-input-*")
	paramsMust(t, e)
	_, e = inputFile.WriteString(text)
	paramsMust(t, e)
	_, e = inputFile.Seek(0, 0)
	paramsMust(t, e)
	out, e := os.CreateTemp(".", "permutation-action-output-*")
	paramsMust(t, e)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	paramsMust(t, e)
	input, e := NewDebuggerCharInput(streams, inputFile)
	paramsMust(t, e)
	action, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: streams, Input: input})
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(action))
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, action.Close())
		paramsMust(t, input.Close())
		paramsMust(t, streams.Close())
		inputFile.Close()
		out.Close()
		os.Remove(inputFile.Name())
		os.Remove(out.Name())
	})
	return p, d, action, input, streams
}
func TestPermutationBoundFalseSizeContinueResumesSourceBody(t *testing.T) {
	p, _, _, in, s := permutationActionFixture(t, "C")
	paramsMust(t, p.forward.Reset())
	before := p.budget.Used()
	p.budget.FailAfter(0)
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal("source return true after continuing false SASSERT")
	}
	state, e := in.State()
	paramsMust(t, e)
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	want := ""
	position := uint64(0)
	if sourceDebug {
		want = "ASSERTION VIOLATION\nFile: src/util/permutation.cpp\nLine: 67\nm_p.size() == m_inv_p.size()\n" + DebugSourcePrompt
		position = 1
	}
	if string(data) != want || state.Position != position || p.budget.Used() != before {
		t.Fatal(string(data), state, p.budget.Used(), before)
	}
}
func TestPermutationBoundIndexedContinueReadsActualInitializedSpare(t *testing.T) {
	p, _, _, in, s := permutationActionFixture(t, "C")
	paramsMust(t, p.forward.Reset())
	value, e := p.At(1)
	paramsMust(t, e)
	if value != 1 {
		t.Fatal("source access after continuation", value)
	}
	state, e := in.State()
	paramsMust(t, e)
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	if sourceDebug {
		if state.Position != 1 || !strings.Contains(string(data), "File: src/util/vector.h\nLine: 309\nidx < size()\n") {
			t.Fatal(state, string(data))
		}
	} else if state.Position != 0 || len(data) != 0 {
		t.Fatal("release erased action", state, string(data))
	}
}
func TestPermutationBoundThrowAndDisabledUnreadClosedAction(t *testing.T) {
	p, d, a, in, _ := permutationActionFixture(t, "T")
	paramsMust(t, p.forward.Reset())
	_, e := p.CheckInvariant()
	if sourceDebug {
		var ex DefaultException
		if !errors.As(e, &ex) || ex.Message() != "assertion violation" {
			t.Fatal(e)
		}
	} else {
		paramsMust(t, e)
	}
	paramsMust(t, a.Close())
	paramsMust(t, d.EnableAssertions(false))
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal(yes)
	}
	state, e := in.State()
	paramsMust(t, e)
	position := uint64(0)
	if sourceDebug {
		position = 1
	}
	if state.Position != position {
		t.Fatal("disabled reached closed action", state)
	}
}
func TestPermutationBoundTrueAndBindingOwnerGuards(t *testing.T) {
	p, d, a, in, _ := permutationActionFixture(t, "C")
	paramsMust(t, a.Close())
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal(yes)
	}
	state, e := in.State()
	paramsMust(t, e)
	if state.Position != 0 {
		t.Fatal("true assertion read action", state)
	}
	foreign := arena.NewBudget(1 << 20)
	other, e := NewDebugActionContext(DebugActionConfig{Budget: foreign, Profile: DebugNoDebugger})
	paramsMust(t, e)
	defer other.Close()
	if !errors.Is(d.BindAssertionActions(other), arena.ErrConfig) {
		t.Fatal("foreign binding")
	}
	copied := *d
	if !errors.Is(copied.BindAssertionActions(a), ErrDebugContextOwner) {
		t.Fatal("copied debug owner")
	}
}

func TestPermutationBoundSwapCapturesIndexedLvaluesOnlyOnce(t *testing.T) {
	p, _, _, in, s := permutationActionFixture(t, "CCCCCCCC")
	paramsMust(t, p.forward.Reset())
	paramsMust(t, p.Swap(0, 1))
	state, e := in.State()
	paramsMust(t, e)
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	want := uint64(0)
	if sourceDebug {
		want = 4
	}
	if state.Position != want {
		t.Fatal("source two scalar reads then two acquired swap lvalues; assignments do not reevaluate operator[]", state.Position, want, string(data))
	}
	if sourceDebug && strings.Count(string(data), "Line: 304\nidx < size()\n") != 4 {
		t.Fatal("mutable source overload", string(data))
	}
	paramsMust(t, p.forward.Push(1))
	paramsMust(t, p.forward.Push(0))
	permutationExpect(t, p, []uint32{1, 0})
}

func permutationBindingPhysical(t *testing.T, p *RuntimePermutation, d *DebugContext, a *DebugActionContext, in *NumericCharInput, s *MemoryStandardStreams) {
	t.Helper()
	runtimeVectorAuditBlocks(t, &p.forward.Column.Raw, p.inverse.Column.Raw.Block)
	r := p.forward.Column.Raw.Runtime
	x := d.region.Stats()
	descriptors := x.Descriptors
	cold := p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse) + d.ownerBytes + x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + descriptors.OwnerBytes + descriptors.RecordBytes + descriptors.MetadataBytes + descriptors.DirectoryBytes
	cold += a.header + in.header + s.header + s.stdout.Stats().OwnerBytes + s.StderrField.Stats().OwnerBytes
	if p.budget.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("exact named raw/cold binding graph", p.budget.Used(), r.Stats().TotalBytes, cold)
	}
}
func TestPermutationBoundImmediateNamedGraphUnderActualRefusal(t *testing.T) {
	p, d, a, in, s := permutationActionFixture(t, "C")
	paramsMust(t, p.forward.Reset())
	before := p.budget.Used()
	p.budget.FailAfter(0)
	permutationBindingPhysical(t, p, d, a, in, s)
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal(yes)
	}
	permutationBindingPhysical(t, p, d, a, in, s)
	if p.budget.Used() != before {
		t.Fatal("binding dispatch made cold reservation")
	}
}
func TestPermutationBoundNestedConditionsContinueWithoutReplacingSourceResult(t *testing.T) {
	p, d, a, in, s := permutationActionFixture(t, "CCCC")
	paramsMust(t, p.forward.Set(0, 1))
	paramsMust(t, p.forward.Set(1, 0))
	permutationBindingPhysical(t, p, d, a, in, s)
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal("literal invariant returns true after source C")
	}
	state, e := in.State()
	paramsMust(t, e)
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	if sourceDebug {
		if state.Position != 4 || strings.Count(string(data), "m_p[m_inv_p[i]] == i") != 2 || strings.Count(string(data), "m_inv_p[m_p[i]] == i") != 2 {
			t.Fatal(state, string(data))
		}
	} else if state.Position != 0 || len(data) != 0 {
		t.Fatal("release conditions erased", state, string(data))
	}
	permutationBindingPhysical(t, p, d, a, in, s)
}

func TestPermutationBoundSubprocessHelper(t *testing.T) {
	route := os.Getenv("G3_PERMUTATION_BOUND_CHILD")
	if route == "" {
		return
	}
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	d, e := NewDebugContext(region, b)
	paramsMust(t, e)
	p, e := NewRuntimePermutationWithAssertions(r, b, 0, d)
	paramsMust(t, e)
	paramsMust(t, p.inverse.Push(0))
	s, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	paramsMust(t, e)
	in, e := NewDebuggerCharInput(s, os.Stdin)
	paramsMust(t, e)
	cfg := DebugActionConfig{Budget: b, Streams: s, Input: in}
	if route == "no-debugger" {
		cfg.Profile = DebugNoDebugger
	}
	a, e := NewDebugActionContext(cfg)
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(s))
	paramsMust(t, d.BindAssertionActions(a))
	b.FailAfter(0)
	yes, e := p.CheckInvariant()
	if e != nil || !yes {
		os.Exit(96)
	}
	out, _ := s.Stdout()
	_ = out.AppendCString("continued\n")
	b.FailAfter(-1)
	paramsMust(t, p.Close())
	paramsMust(t, a.Close())
	paramsMust(t, in.Close())
	paramsMust(t, s.Close())
	paramsMust(t, d.Close())
	paramsMust(t, region.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		os.Exit(97)
	}
	os.Exit(0)
}
func TestPermutationBoundActualAbortEOFAndNoDebuggerActions(t *testing.T) {
	for _, test := range []struct {
		name, text string
		code       int
		prompt     bool
	}{{"abort", "A", 1, true}, {"eof", "", 110, true}, {"no-debugger", "", 110, false}, {"continue", "C", 0, true}} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPermutationBoundSubprocessHelper$")
			f, e := os.CreateTemp(".", "permutation-bound-child-*")
			paramsMust(t, e)
			defer func() { f.Close(); os.Remove(f.Name()) }()
			_, e = f.WriteString(test.text)
			paramsMust(t, e)
			_, e = f.Seek(0, 0)
			paramsMust(t, e)
			cmd.Stdin = f
			cmd.Env = append(os.Environ(), "G3_PERMUTATION_BOUND_CHILD="+test.name)
			var out, errout strings.Builder
			cmd.Stdout = &out
			cmd.Stderr = &errout
			e = cmd.Run()
			if ctx.Err() != nil {
				t.Fatal("bounded source exit child", ctx.Err())
			}
			code := 0
			if e != nil {
				var exit *exec.ExitError
				if !errors.As(e, &exit) {
					t.Fatal(e)
				}
				code = exit.ExitCode()
			}
			wantCode := 0
			wantOut := "continued\n"
			wantErr := ""
			if sourceDebug {
				wantCode = test.code
				if test.code != 0 {
					wantOut = ""
				}
				wantErr = "ASSERTION VIOLATION\nFile: src/util/permutation.cpp\nLine: 67\nm_p.size() == m_inv_p.size()\n"
				if test.prompt {
					wantErr += DebugSourcePrompt
				}
			}
			if code != wantCode || out.String() != wantOut || errout.String() != wantErr {
				t.Fatal(code, out.String(), errout.String(), wantCode, wantOut, wantErr)
			}
		})
	}
}

// This explicit pointer-representation fixture keeps both actual backing roots
// live while exchanging source m_data fields. No source allocation failure or
// debugger mutation is fabricated; views must preserve the acquired address.
func TestPermutationSwapWordViewsKeepActualCapturedLiveBacking(t *testing.T) {
	p, d, r, b := permutationDiagnosticFixture(t, 2)
	other, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, other.Close()) })
	paramsMust(t, other.Push(17))
	paramsMust(t, other.Push(19))
	left, e := p.acquireMutable(&p.forward, 0)
	paramsMust(t, e)
	right, e := p.acquireMutable(&p.forward, 1)
	paramsMust(t, e)
	old := p.forward.Column.Raw.Block
	next := other.Column.Raw.Block
	p.forward.Column.Raw.Block, other.Column.Raw.Block = next, old
	permutationWordViewPhysical(t, p, d, other)
	x, e := left.read()
	paramsMust(t, e)
	y, e := right.read()
	paramsMust(t, e)
	if x != 0 || y != 1 {
		t.Fatal("captured loads reevaluated current m_data", x, y)
	}
	paramsMust(t, left.write(y))
	paramsMust(t, right.write(x))
	actual, e := other.At(0)
	paramsMust(t, e)
	if actual != 1 {
		t.Fatal("captured store", actual)
	}
	actual, e = p.forward.At(0)
	paramsMust(t, e)
	if actual != 17 {
		t.Fatal("new source field untouched", actual)
	}
	permutationWordViewPhysical(t, p, d, other)
	// Restore actual source pointer ownership before ordinary class checks.
	p.forward.Column.Raw.Block, other.Column.Raw.Block = old, next
}
func TestPermutationAcquiredWordGuardAfterActualBackingDestruction(t *testing.T) {
	p, _, _, _ := permutationDiagnosticFixture(t, 2)
	view, e := p.acquireMutable(&p.forward, 0)
	paramsMust(t, e)
	paramsMust(t, p.forward.SourceFinalize())
	if _, e = view.read(); e == nil {
		t.Fatal("actual consumed captured block read succeeded")
	}
	if e = view.write(9); e == nil {
		t.Fatal("actual consumed captured block write succeeded")
	}
}

func permutationWordViewPhysical(t *testing.T, p *RuntimePermutation, d *DebugContext, other *RuntimeUnsignedVector) {
	t.Helper()
	runtimeVectorAuditBlocks(t, &p.forward.Column.Raw, p.inverse.Column.Raw.Block, other.Column.Raw.Block)
	x := d.region.Stats()
	z := x.Descriptors
	r := p.forward.Column.Raw.Runtime
	cold := p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse) + other.header + unsignedCopyCold(t, other) + d.ownerBytes + x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + z.OwnerBytes + z.RecordBytes + z.MetadataBytes + z.DirectoryBytes
	if p.budget.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("named captured-address owners raw/cold graph", p.budget.Used(), r.Stats().TotalBytes, cold)
	}
}

func TestDebugBindingReciprocalPublicSetterKeepsOneActualStreamService(t *testing.T) {
	p, d, a, input, s := permutationActionFixture(t, "C")
	out, e := os.CreateTemp(".", "debug-binding-other-stream-*")
	paramsMust(t, e)
	other, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: d.BudgetField, Stdout: out, Stderr: out})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, other.Close()); out.Close(); os.Remove(out.Name()) })
	paramsMust(t, s.StderrField.SetIntegerBase(16))
	paramsMust(t, other.StderrField.SetIntegerBase(10))
	before := d.BudgetField.Used()
	d.BudgetField.FailAfter(0)
	if e = d.BindAssertionStreams(other); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("different actual std::cerr binding must be rejected", e)
	}
	if d.assertionStreams != s || d.assertionActions != a || d.BudgetField.Used() != before {
		t.Fatal("rejected binding mutated pointers/cold charge")
	}
	paramsMust(t, d.NotifyAssertionViolation("binding.cpp", 12, "condition"))
	actual, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	unused, e := os.ReadFile(out.Name())
	paramsMust(t, e)
	if string(actual) != "ASSERTION VIOLATION\nFile: binding.cpp\nLine: c\ncondition\n" || len(unused) != 0 {
		t.Fatal("actual selected output identity", string(actual), string(unused))
	}
	paramsMust(t, p.forward.Reset())
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal(yes)
	}
	state, e := input.State()
	paramsMust(t, e)
	actual, e = os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	unused, e = os.ReadFile(out.Name())
	paramsMust(t, e)
	if sourceDebug && (state.Position != 1 || !strings.HasSuffix(string(actual), "Line: 43\nm_p.size() == m_inv_p.size()\n"+DebugSourcePrompt)) {
		t.Fatal("notification and action retained same formatted stderr", state, string(actual))
	}
	if len(unused) != 0 {
		t.Fatal("rejected output was used", string(unused))
	}
	// The identity check does not read the different service's closed lifetime.
	paramsMust(t, other.Close())
	if e = d.BindAssertionStreams(other); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("identity before pointed lifetime", e)
	}
}
func TestDebugBindingReverseOrderNullAndNoDebuggerConfiguration(t *testing.T) {
	_, d, a, _, s := permutationActionFixture(t, "C")
	out, e := os.CreateTemp(".", "debug-binding-reverse-*")
	paramsMust(t, e)
	other, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: d.BudgetField, Stdout: out, Stderr: out})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, other.Close()); out.Close(); os.Remove(out.Name()) })
	paramsMust(t, d.BindAssertionActions(nil))
	paramsMust(t, d.BindAssertionStreams(other))
	if e = d.BindAssertionActions(a); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("reverse setter order identity", e)
	}
	paramsMust(t, d.BindAssertionStreams(nil))
	paramsMust(t, d.BindAssertionActions(a))
	if e = d.NotifyAssertionViolation("unbound.cpp", 1, "condition"); !errors.Is(e, ErrDebugNotificationBinding) {
		t.Fatal("nil notification binding remains explicit", e)
	}
	paramsMust(t, d.BindAssertionStreams(s))
	paramsMust(t, a.Close())
	paramsMust(t, d.BindAssertionStreams(s)) // same closed action identity remains unread.
	paramsMust(t, d.BindAssertionActions(nil))
	n, e := NewDebugActionContext(DebugActionConfig{Budget: d.BudgetField, Profile: DebugNoDebugger})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, n.Close()) })
	paramsMust(t, d.BindAssertionActions(n))
	paramsMust(t, d.BindAssertionStreams(other)) // NO_DEBUGGER erases action streams.
}
