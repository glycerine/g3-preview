// Source: src/util/debug.h:63-92; src/util/heap.h:57-66,79-239,258-265.
// Source: src/util/vector.h:303-344,389-403.
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

func TestDebugMacroUnreachableIgnoresAssertionSwitch(t *testing.T) {
	_, d, _, in, s := permutationActionFixture(t, "T")
	paramsMust(t, d.EnableAssertions(false))
	e := d.UnreachableSource("macro_fixture.cpp", 41)
	if sourceDebug {
		var ex DefaultException
		if !errors.As(e, &ex) {
			t.Fatal(e)
		}
		state, e := in.State()
		paramsMust(t, e)
		if state.Position != 1 {
			t.Fatal(state)
		}
		data, e := os.ReadFile(s.StderrField.File.Name())
		paramsMust(t, e)
		want := "ASSERTION VIOLATION\nFile: macro_fixture.cpp\nLine: 41\nUNREACHABLE CODE WAS REACHED.\n" + DebugSourcePrompt
		if string(data) != want {
			t.Fatal(string(data))
		}
	} else if !errors.Is(e, ErrDebugNativeUnreachable) {
		t.Fatal("GNU source undefined-domain guard", e)
	}
}
func TestDebugMacroVerifyFalsePrefixAndContinue(t *testing.T) {
	_, d, _, in, s := permutationActionFixture(t, "C")
	paramsMust(t, d.EnableAssertions(false))
	e := d.VerifyFailure("macro_fixture.cpp", 52, "value == expected")
	data, re := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, re)
	want := "Failed to verify: value == expected\n"
	if sourceDebug {
		paramsMust(t, e)
		want += "ASSERTION VIOLATION\nFile: macro_fixture.cpp\nLine: 52\nUNREACHABLE CODE WAS REACHED.\n" + DebugSourcePrompt
		state, e := in.State()
		paramsMust(t, e)
		if state.Position != 1 {
			t.Fatal(state)
		}
	} else if !errors.Is(e, ErrDebugNativeUnreachable) {
		t.Fatal(e)
	}
	if string(data) != want {
		t.Fatal(string(data), want)
	}
}
func TestDebugMacroXAssertionLiteralExtraBeforeAction(t *testing.T) {
	_, d, _, in, s := permutationActionFixture(t, "T")
	var reached uint32
	// Exact definition-only expansion: gate, then condition, notify, actual EXTRA,
	// debugger. No upstream XASSERT invocation exists and no callback is used.
	on, e := PermutationAssertions(d)
	paramsMust(t, e)
	if on {
		reached++
		condition := false
		if !condition {
			paramsMust(t, d.NotifyAssertionViolation("macro_fixture.cpp", 65, "condition"))
			out, e := s.Stderr()
			paramsMust(t, e)
			paramsMust(t, out.AppendCString("EXTRA\n"))
			// EXTRA can change the switch; invoke_gdb itself does not read it.
			paramsMust(t, d.EnableAssertions(false))
			e = d.InvokeAssertionDebugger()
			var ex DefaultException
			if !errors.As(e, &ex) {
				t.Fatal(e)
			}
		}
	}
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	state, e := in.State()
	paramsMust(t, e)
	if sourceDebug {
		want := "ASSERTION VIOLATION\nFile: macro_fixture.cpp\nLine: 65\ncondition\nEXTRA\n" + DebugSourcePrompt
		if reached != 1 || state.Position != 1 || string(data) != want {
			t.Fatal(reached, state, string(data))
		}
	} else if reached != 0 || state.Position != 0 || len(data) != 0 {
		t.Fatal("release erased condition/EXTRA/actions", reached, state, string(data))
	}
}
func TestDebugMacroHeapTaggedDuplicateInsertContinueAllNestedAssertions(t *testing.T) {
	p, d, a, in, s := permutationActionFixture(t, "CCC")
	r := p.forward.Column.Raw.Runtime
	b := p.budget
	paramsMust(t, d.EnableDebug(debugTag(t, d.region, "heap")))
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.Close()) })
	paramsMust(t, h.Insert(1))
	macroHeapPhysical(t, p, d, a, in, s, h)
	paramsMust(t, h.Insert(1))
	macroHeapPhysical(t, p, d, a, in, s, h)
	state, e := in.State()
	paramsMust(t, e)
	data, e := os.ReadFile(s.StderrField.File.Name())
	paramsMust(t, e)
	want := uint64(0)
	if sourceDebug {
		want = 3
		if strings.Count(string(data), "Line: 232\n!contains(val)\n") != 1 || strings.Count(string(data), "Line: 63\nm_value2indices[m_values[idx]] == idx\n") != 2 {
			t.Fatal(string(data))
		}
	}
	if state.Position != want {
		t.Fatal("literal duplicate insert/invariant continued prefixes", state, want)
	}
	size, e := h.runtimeQueue.sizeChecked()
	paramsMust(t, e)
	if size != 2 {
		t.Fatal("continued source publication", size)
	}
}

func TestDebugMacroExitSubprocessHelper(t *testing.T) {
	route := os.Getenv("G3_DEBUG_MACRO_CHILD")
	if route == "" {
		return
	}
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	d, e := NewDebugContext(region, b)
	paramsMust(t, e)
	cfg := MemoryStandardStreamsConfig{Budget: b}
	if route == "ensure-bad-file" {
		cfg.Stderr = os.NewFile(3, "borrowed-parent-descriptor")
	}
	s, e := NewMemoryStandardStreams(cfg)
	paramsMust(t, e)
	input, e := NewDebuggerCharInput(s, os.Stdin)
	paramsMust(t, e)
	a, e := NewDebugActionContext(DebugActionConfig{Budget: b, Streams: s, Input: input})
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(s))
	paramsMust(t, d.BindAssertionActions(a))
	b.FailAfter(0)
	if route == "ensure-bad-file" {
		paramsMust(t, cfg.Stderr.Close())
	}
	e = nil
	if route == "ensure" || route == "ensure-bad-file" {
		e = d.EnsureFailure("condition")
	} else {
		e = d.NotImplementedYet("macro_fixture.cpp", 80)
	}
	if errors.Is(e, ErrDebugNativeUnreachable) {
		out, _ := s.Stdout()
		_ = out.AppendCString("undefined-release-guard\n")
		paramsMust(t, a.Close())
		paramsMust(t, input.Close())
		paramsMust(t, s.Close())
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		if b.Used() != 0 {
			os.Exit(97)
		}
		os.Exit(0)
	}
	os.Exit(96)
}
func TestDebugMacroActualEnsureAndNotImplementedExits(t *testing.T) {
	for _, route := range []string{"ensure", "ensure-bad-file", "not-implemented"} {
		t.Run(route, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDebugMacroExitSubprocessHelper$")
			cmd.Env = append(os.Environ(), "G3_DEBUG_MACRO_CHILD="+route)
			f, e := os.CreateTemp(".", "debug-macro-exit-input-*")
			paramsMust(t, e)
			defer func() { f.Close(); os.Remove(f.Name()) }()
			_, e = f.WriteString("C")
			paramsMust(t, e)
			_, e = f.Seek(0, 0)
			paramsMust(t, e)
			cmd.Stdin = f
			if route == "ensure-bad-file" {
				cmd.ExtraFiles = []*os.File{f}
			}
			var out, errout strings.Builder
			cmd.Stdout = &out
			cmd.Stderr = &errout
			e = cmd.Run()
			if ctx.Err() != nil {
				t.Fatal("bounded exit child", ctx.Err())
			}
			code := 0
			if e != nil {
				var exit *exec.ExitError
				if !errors.As(e, &exit) {
					t.Fatal(e)
				}
				code = exit.ExitCode()
			}
			wantOut := ""
			wantErr := "Failed to verify: condition\n"
			wantCode := 255
			if route == "ensure-bad-file" {
				wantErr = ""
			}
			if route == "not-implemented" {
				wantErr = "NOT IMPLEMENTED YET!\n"
				if sourceDebug {
					wantCode = 107
					wantErr += "ASSERTION VIOLATION\nFile: macro_fixture.cpp\nLine: 80\nUNREACHABLE CODE WAS REACHED.\n" + DebugSourcePrompt
				} else {
					wantCode = 0
					wantOut = "undefined-release-guard\n"
				}
			}
			if code != wantCode || out.String() != wantOut || errout.String() != wantErr {
				t.Fatal(code, out.String(), errout.String(), wantCode, wantOut, wantErr)
			}
		})
	}
}

func macroHeapPhysical(t *testing.T, p *RuntimePermutation, d *DebugContext, a *DebugActionContext, in *NumericCharInput, s *MemoryStandardStreams, h *IntHeap) {
	t.Helper()
	runtimeVectorAuditBlocks(t, &h.runtimeQueue.runtimeValues.Raw, h.runtimeQueue.runtimeIndices.Raw.Block, p.forward.Column.Raw.Block, p.inverse.Column.Raw.Block)
	x := d.region.Stats()
	z := x.Descriptors
	r := p.forward.Column.Raw.Runtime
	cold := p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse) + d.ownerBytes + x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + z.OwnerBytes + z.RecordBytes + z.MetadataBytes + z.DirectoryBytes + intHeapOwnerBytes + a.header + in.header + s.header + s.stdout.Stats().OwnerBytes + s.StderrField.Stats().OwnerBytes
	if p.budget.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("actual named macro heap/borrowed caller raw and cold graph", p.budget.Used(), r.Stats().TotalBytes, cold)
	}
}
func TestDebugMacroHeapDuplicateInsertEveryActualGrowthRefusal(t *testing.T) {
	success := int64(-1)
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		ok := false
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			p, d, a, in, s := permutationActionFixture(t, strings.Repeat("C", 16))
			b := p.budget
			r := p.forward.Column.Raw.Runtime
			paramsMust(t, d.EnableDebug(debugTag(t, d.region, "heap")))
			h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.Close()) })
			paramsMust(t, h.Insert(1))
			b.FailAfter(ordinal)
			e = h.Insert(1)
			macroHeapPhysical(t, p, d, a, in, s, h) // before any fault lifting/owning cleanup.
			if e == nil {
				ok = true
				return
			}
			var failure Failure
			if !errors.As(e, &failure) {
				t.Fatal("real Runtime growth failure", e)
			}
			state, e := in.State()
			paramsMust(t, e)
			want := uint64(0)
			if sourceDebug {
				want = 1
			}
			if state.Position != want {
				t.Fatal("only reached pre-push CASSERT before allocation failure", state, want)
			}
			b.FailAfter(-1)
		})
		if ok {
			success = ordinal
			break
		}
	}
	if success < 0 {
		t.Fatal("did not reach first actual growth success")
	}
	t.Log("actual growth reservation horizon", success)
}
func TestDebugMacroCassertLiteralTagShortCircuitAndUninitializedCondition(t *testing.T) {
	p, d, _, _, _ := permutationActionFixture(t, "C")
	r := p.forward.Column.Raw.Runtime
	word, e := r.Allocate(1)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, r.Deallocate(&word)) })
	paramsMust(t, d.EnableAssertions(false))
	on := false
	// Release erases this whole gate; assertion-off erases tag and payload reads.
	if sourceDebug {
		on, e = d.ConditionalAssertionEnabled("heap")
		paramsMust(t, e)
	}
	if on {
		_, e = r.Read(word, 0)
		t.Fatal("erased uninitialized condition was read", e)
	}
	paramsMust(t, d.EnableAssertions(true))
	if sourceDebug {
		on, e = d.ConditionalAssertionEnabled("heap")
		paramsMust(t, e)
	}
	if on {
		t.Fatal("disabled source tag read condition")
	}
	paramsMust(t, d.EnableDebug(debugTag(t, d.region, "heap")))
	if sourceDebug {
		on, e = d.ConditionalAssertionEnabled("heap")
		paramsMust(t, e)
	}
	if on {
		_, e = r.Read(word, 0)
		if !errors.Is(e, ErrRuntimeUninitialized) {
			t.Fatal("actual reached condition word guard", e)
		}
	}
}

func TestDebugMacroVerifyConditionIsNotErasedByAssertionSwitch(t *testing.T) {
	p, d, _, _, _ := permutationActionFixture(t, "C")
	paramsMust(t, d.EnableAssertions(false))
	r := p.forward.Column.Raw.Runtime
	word, e := r.Allocate(4)
	paramsMust(t, e)
	t.Cleanup(func() {
		if word != (Block{}) {
			paramsMust(t, r.Deallocate(&word))
		}
	})
	paramsMust(t, WriteRuntimeWord(r, word, 0, 7))
	// VERIFY evaluates this reached operand in release and with the assertion
	// switch false. A true condition performs no notification or input action.
	value, e := ReadRuntimeWord(r, word, 0)
	paramsMust(t, e)
	if !(value == 7) {
		paramsMust(t, d.VerifyFailure("macro_fixture.cpp", 121, "value == 7"))
	}
	paramsMust(t, r.Deallocate(&word))
	// This actual guarded read proves the same operand is still reached; VERIFY
	// must not erase it behind the source assertions switch in either profile.
	_, e = ReadRuntimeWord(r, word, 0)
	if e == nil {
		t.Fatal("destroyed operand unexpectedly readable")
	}
}

func TestDebugMacroHeapFalseTagThrowPrecedesValuePublication(t *testing.T) {
	p, d, a, input, streams := permutationActionFixture(t, "T")
	r, b := p.forward.Column.Raw.Runtime, p.budget
	paramsMust(t, d.EnableDebug(debugTag(t, d.region, "heap")))
	h, e := NewIntHeap(8, IntHeapConfig{Budget: b, Runtime: r, DebugContext: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.Close()) })
	paramsMust(t, h.Insert(1))
	e = h.Insert(1)
	if sourceDebug {
		var ex DefaultException
		if !errors.As(e, &ex) {
			t.Fatal(e)
		}
		n, e := h.runtimeQueue.sizeChecked()
		paramsMust(t, e)
		if n != 1 {
			t.Fatal("throw at heap.h:232 must precede append", n)
		}
	} else {
		paramsMust(t, e)
	}
	macroHeapPhysical(t, p, d, a, input, streams, h)
}

func TestDebugMacroHeapNonInsertFalseMinReachesSourceNotification(t *testing.T) {
	p, d, _, input, streams := permutationActionFixture(t, "T")
	h, e := NewIntHeap(8, IntHeapConfig{Budget: p.budget, Runtime: p.forward.Column.Raw.Runtime, DebugContext: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.Close()) })
	_, e = h.Min()
	if sourceDebug {
		var ex DefaultException
		if !errors.As(e, &ex) {
			t.Fatal("actual heap.h:169 false site must invoke bound T", e)
		}
		data, e := os.ReadFile(streams.StderrField.File.Name())
		paramsMust(t, e)
		if !strings.Contains(string(data), "Line: 169\n!empty()\n") || strings.Contains(string(data), "Line: 0\n") {
			t.Fatal(string(data))
		}
		state, e := input.State()
		paramsMust(t, e)
		if state.Position != 1 {
			t.Fatal(state)
		}
	} else if e == nil {
		t.Fatal("release reached actual physical bounds")
	}
}

func TestDebugMacroHeapMinContinueReachesConstIndexedSpare(t *testing.T) {
	p, d, a, input, streams := permutationActionFixture(t, "CC")
	h, e := NewIntHeap(8, IntHeapConfig{Budget: p.budget, Runtime: p.forward.Column.Raw.Runtime, DebugContext: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.Close()) })
	paramsMust(t, h.Insert(1))
	paramsMust(t, h.Erase(1))
	// Guarded invalid logical-index domain: actual source C continuation reads
	// an initialized physical spare, with separate !empty and operator[] sites.
	v, e := h.Min()
	paramsMust(t, e)
	if v != 1 {
		t.Fatal(v)
	}
	state, e := input.State()
	paramsMust(t, e)
	data, e := os.ReadFile(streams.StderrField.File.Name())
	paramsMust(t, e)
	if sourceDebug {
		if state.Position != 2 || !strings.Contains(string(data), "Line: 169\n!empty()\n") || !strings.Contains(string(data), "Line: 309\nidx < size()\n") {
			t.Fatal(state, string(data))
		}
	} else if state.Position != 0 || len(data) != 0 {
		t.Fatal(state, string(data))
	}
	macroHeapPhysical(t, p, d, a, input, streams, h)
}

func TestDebugMacroHeapRemainingFalseLeavesThrowBeforeBody(t *testing.T) {
	for _, name := range []string{"erase", "erase_min", "decreased", "increased"} {
		t.Run(name, func(t *testing.T) {
			p, d, a, input, streams := permutationActionFixture(t, "T")
			h, e := NewIntHeap(8, IntHeapConfig{Budget: p.budget, Runtime: p.forward.Column.Raw.Runtime, DebugContext: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.Close()) })
			line, cond := int32(0), ""
			switch name {
			case "erase":
				line, cond = 197, "contains(val)"
				e = h.Erase(1)
			case "erase_min":
				line, cond = 175, "!empty()"
				_, e = h.EraseMin()
			case "decreased":
				line, cond = 221, "contains(val)"
				e = h.Decreased(1)
			case "increased":
				line, cond = 226, "contains(val)"
				e = h.Increased(1)
			}
			if sourceDebug {
				var ex DefaultException
				if !errors.As(e, &ex) {
					t.Fatal(e)
				}
				data, re := os.ReadFile(streams.StderrField.File.Name())
				paramsMust(t, re)
				if !strings.Contains(string(data), "Line: "+strconv.Itoa(int(line))+"\n"+cond+"\n") {
					t.Fatal(string(data))
				}
				n, re := h.runtimeQueue.runtimeValues.Size()
				paramsMust(t, re)
				if n != 1 {
					t.Fatal("throw changed value SIZE", n)
				}
			} else if name == "erase" {
				// Erased contains assertion permits the literal index-zero branch:
				// clear inverse then pop the sentinel. Preserve that actual prefix.
				paramsMust(t, e)
				n, re := h.runtimeQueue.runtimeValues.Size()
				paramsMust(t, re)
				if n != 0 {
					t.Fatal(n)
				}
			} else if e == nil {
				t.Fatal("release must reach actual physical/null reverse-index guard")
			}
			macroHeapPhysical(t, p, d, a, input, streams, h)
		})
	}
}

func TestDebugMacroHeapNestedVectorShrinkContinuePublishesAfterAssertion(t *testing.T) {
	p, d, a, input, streams := permutationActionFixture(t, "C")
	h, e := NewIntHeap(8, IntHeapConfig{Budget: p.budget, Runtime: p.forward.Column.Raw.Runtime, DebugContext: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.Close()) })
	paramsMust(t, h.Insert(1))
	paramsMust(t, h.Erase(1))
	// Literal vector::shrink failure is a continued guarded domain. Its source
	// SIZE store follows the action; do not return immediately after C.
	paramsMust(t, h.shrinkUintRuntime(&h.runtimeQueue.runtimeValues, 2))
	n, e := h.runtimeQueue.runtimeValues.Size()
	paramsMust(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	state, e := input.State()
	paramsMust(t, e)
	data, e := os.ReadFile(streams.StderrField.File.Name())
	paramsMust(t, e)
	if sourceDebug {
		if state.Position != 1 || !strings.Contains(string(data), "Line: 391\ns <= reinterpret_cast<SZ *>(m_data)[SIZE_IDX]\n") {
			t.Fatal(state, string(data))
		}
	} else if state.Position != 0 || len(data) != 0 {
		t.Fatal(state, string(data))
	}
	macroHeapPhysical(t, p, d, a, input, streams, h)
}
