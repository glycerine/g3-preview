// Source: src/util/timeit.cpp:25-58.
package stopwatch

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestTimeitReadsSharedProcessAllocationAtBothSnapshots(t *testing.T) {
	b := arena.NewBudget(2 << 20)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	timerMust(t, err)
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	timerMust(t, err)
	var output bytes.Buffer
	var scope Timeit
	timerMust(t, scope.Construct(true, TimeitConfig{Output: &output, Message: "shared", Process: p,
		Clock: func() (uint64, error) { return 0, nil }}))
	block, err := other.Allocate(1 << 20)
	timerMust(t, err)
	if got := p.AllocationSize(); got < 1<<20 {
		t.Fatal("other runtime did not publish process allocation", got)
	}
	timerMust(t, scope.Close())
	if got, want := output.String(), "(shared :time 0.00 :before-memory 0.00 :after-memory 1.00)\n"; got != want {
		t.Fatal(got, want)
	}
	timerMust(t, other.Deallocate(&block))
	timerMust(t, other.Close())
	timerMust(t, r.Close())
	if p.AllocationSize() != 0 || b.Used() != 0 {
		t.Fatal("shared process retained memory", p.AllocationSize(), b.Used())
	}
}

func TestTimeitOriginalMemoryCaptureAndFormatting(t *testing.T) {
	var output bytes.Buffer
	now, memory := uint64(100), uint64(0)
	clockReads, memoryReads := 0, 0
	var scope Timeit
	timerMust(t, scope.Construct(true, TimeitConfig{
		Output: &output, Message: "isolate\x00ignored",
		Clock:  func() (uint64, error) { clockReads++; return now, nil },
		Memory: func() (uint64, error) { memoryReads++; return memory, nil },
	}))
	if clockReads != 1 || memoryReads != 1 || output.Len() != 0 {
		t.Fatal("construction reads", clockReads, memoryReads, output.String())
	}
	now += 1_250_000_000
	memory = 1024 * 1024
	timerMust(t, scope.Close())
	timerMust(t, scope.Close())
	if got, want := output.String(), "(isolate :time 1.25 :before-memory 0.00 :after-memory 1.00)\n"; got != want {
		t.Fatal(got, want)
	}
	if clockReads != 2 || memoryReads != 2 {
		t.Fatal("report must stop without restarting", clockReads, memoryReads)
	}
}

func TestTimeitDisabledErasureAndDefaultOutput(t *testing.T) {
	var disabled Timeit
	timerMust(t, disabled.Construct(false, TimeitConfig{
		Clock:  func() (uint64, error) { t.Fatal("disabled clock"); return 0, nil },
		Memory: func() (uint64, error) { t.Fatal("disabled memory"); return 0, nil },
	}))
	timerMust(t, disabled.Close())
	timerMust(t, disabled.Cleanup())
	var scope Timeit
	timerMust(t, scope.Construct(true, TimeitConfig{Clock: func() (uint64, error) { return 100, nil }}))
	if scope.output != os.Stderr {
		t.Fatal("default output")
	}
	copy := scope
	if copy.Close() != ErrTimeitOwner || copy.Cleanup() != ErrTimeitOwner {
		t.Fatal("copied scope")
	}
	timerMust(t, scope.Cleanup())
	var absent *Timeit
	if absent.Construct(false, TimeitConfig{}) != ErrTimeitOwner || absent.Close() != ErrTimeitOwner {
		t.Fatal("nil scope")
	}
}

func TestTimeitClockAndMemoryFailurePrefixes(t *testing.T) {
	failure := errors.New("provider failure")
	for _, failedProvider := range []string{"clock", "memory"} {
		for _, failedRead := range []int{1, 2} {
			var output bytes.Buffer
			calls := 0
			provider := func() (uint64, error) {
				calls++
				if calls == failedRead {
					return 0, failure
				}
				return 100, nil
			}
			config := TimeitConfig{Output: &output, Clock: func() (uint64, error) { return 100, nil }}
			if failedProvider == "clock" {
				config.Clock = provider
			} else {
				config.Memory = provider
			}
			var scope Timeit
			e := scope.Construct(true, config)
			if failedRead == 2 {
				timerMust(t, e)
				e = scope.Close()
			}
			if e != failure || scope.Close() != failure || calls != failedRead || output.Len() != 0 {
				t.Fatal(failedProvider, failedRead, e, calls, output.String())
			}
			timerMust(t, scope.Cleanup())
			timerMust(t, scope.Close())
		}
	}
}

type timeitWriter struct {
	bytes.Buffer
	writeError, flushError error
	writes, flushes        int
}

func (w *timeitWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writeError != nil {
		n, _ := w.Buffer.Write(p[:1])
		return n, w.writeError
	}
	return w.Buffer.Write(p)
}
func (w *timeitWriter) Flush() error { w.flushes++; return w.flushError }

func TestTimeitOutputFailureAndFlush(t *testing.T) {
	failure := errors.New("output failure")
	for _, mode := range []string{"write", "flush", "success"} {
		output := &timeitWriter{}
		if mode == "write" {
			output.writeError = failure
		}
		if mode == "flush" {
			output.flushError = failure
		}
		var scope Timeit
		timerMust(t, scope.Construct(true, TimeitConfig{Output: output, Message: "report", Clock: func() (uint64, error) { return 0, nil }}))
		cause := scope.Close()
		if mode == "success" {
			timerMust(t, cause)
		} else if cause != failure || scope.Close() != failure {
			t.Fatal("output error must be sticky", mode, cause)
		}
		if mode == "write" {
			if output.String() != "(" || output.flushes != 0 {
				t.Fatal(output)
			}
		} else if output.String() != "(report :time 0.00 :before-memory 0.00 :after-memory 0.00)\n" || output.flushes != 1 {
			t.Fatal(output)
		}
		timerMust(t, scope.Cleanup())
		wantWrites := 10
		if mode == "write" {
			wantWrites = 1
		}
		if output.writes != wantWrites {
			t.Fatal("cleanup replayed output", output.writes)
		}
	}
}

type timeitBoundaryWriter struct {
	bytes.Buffer
	call, failAt, partial, flushes int
	cause                          error
}

func (w *timeitBoundaryWriter) Write(p []byte) (int, error) {
	w.call++
	if w.call == w.failAt {
		if w.partial == 0 {
			return 0, w.cause
		}
		if len(p) > 1 {
			return w.Buffer.Write(p[:1])
		}
		return 0, nil
	}
	return w.Buffer.Write(p)
}
func (w *timeitBoundaryWriter) Flush() error { w.flushes++; return nil }

func TestTimeitEachSourceOutputBoundary(t *testing.T) {
	parts := []string{"(", "report", " :time ", "0.00", " :before-memory ", "0.00", " :after-memory ", "0.00", ")", "\n"}
	failure := errors.New("write failed")
	for _, partial := range []int{0, 1} {
		for at := 1; at <= len(parts); at++ {
			out := &timeitBoundaryWriter{failAt: at, partial: partial, cause: failure}
			var scope Timeit
			timerMust(t, scope.Construct(true, TimeitConfig{Output: out, Message: "report", Clock: func() (uint64, error) { return 0, nil }}))
			e := scope.Close()
			want := failure
			if partial != 0 {
				want = io.ErrShortWrite
			}
			prefix := strings.Join(parts[:at-1], "")
			if partial != 0 && len(parts[at-1]) > 1 {
				prefix += parts[at-1][:1]
			}
			if e != want || scope.Close() != want || out.String() != prefix || out.call != at || out.flushes != 0 {
				t.Fatal("timeit output prefix", partial, at, e, out.String(), prefix, out.call, out.flushes)
			}
			timerMust(t, scope.Cleanup())
		}
	}
}

func TestTimeitProviderReentryAndBorrowedOutput(t *testing.T) {
	var scope Timeit
	var output bytes.Buffer
	clockReads, memoryReads := 0, 0
	checkReentry := func() {
		if scope.Close() != ErrTimeitPending || scope.Cleanup() != ErrTimeitPending {
			t.Fatal("provider reentry")
		}
	}
	timerMust(t, scope.Construct(true, TimeitConfig{Output: &output, Message: "reentry",
		Clock:  func() (uint64, error) { checkReentry(); clockReads++; return uint64(clockReads) * 100, nil },
		Memory: func() (uint64, error) { checkReentry(); memoryReads++; return uint64(memoryReads) * 1024 * 1024, nil },
	}))
	timerMust(t, scope.Close())
	if clockReads != 2 || memoryReads != 2 || output.String() != "(reentry :time 0.00 :before-memory 1.00 :after-memory 2.00)\n" {
		t.Fatal(clockReads, memoryReads, output.String())
	}
	timerMust(t, scope.Cleanup())
	output.WriteString("caller")
	if !bytes.HasSuffix(output.Bytes(), []byte("caller")) {
		t.Fatal("borrowed output")
	}
}
