//go:build g3debug && g3trace

package stopwatch

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestTimeitImpMemoryTraceDisabledFailureAndCompletedReport(t *testing.T) {
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("memory")
	var disabled Timeit
	if err := disabled.Construct(false, TimeitConfig{Trace: tags}); err != nil {
		t.Fatal(err)
	}
	if err := disabled.Close(); err != nil || trace.Len() != 0 {
		t.Fatal("disabled source imp", err, trace.String())
	}
	var failed Timeit
	if err := failed.Construct(true, TimeitConfig{Trace: tags}); err != ErrTimeitMemorySource {
		t.Fatal(err)
	}
	failedAlloc := "alloc " + fmt.Sprintf("%p", &failed) + " src/util/timeit.cpp:50 imp " + strconv.FormatUint(uint64(unsafe.Sizeof(failed)), 10) + "\n"
	failedDealloc := "dealloc " + fmt.Sprintf("%p", &failed) + " src/util/timeit.cpp:57\n"
	if strings.Count(trace.String(), failedAlloc) != 1 || strings.Contains(trace.String(), failedDealloc) {
		t.Fatal("failed constructor retained imp until owning cleanup", trace.String())
	}
	if err := failed.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(trace.String(), failedDealloc) != 1 {
		t.Fatal("failed imp cleanup", trace.String())
	}
	var report bytes.Buffer
	var completed Timeit
	if err := completed.Construct(true, TimeitConfig{
		Trace: tags, Output: &report, Message: "timed",
		Memory: func() (uint64, error) { return 0, nil },
		Clock: func() (uint64, error) {
			if !strings.Contains(trace.String(), "alloc "+fmt.Sprintf("%p", &completed)+" src/util/timeit.cpp:50 imp ") {
				t.Fatal("allocation did not precede source watch start", trace.String())
			}
			return 0, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := completed.Close(); err != nil {
		t.Fatal(err)
	}
	if report.String() != "(timed :time 0.00 :before-memory 0.00 :after-memory 0.00)\n" ||
		strings.Count(trace.String(), "dealloc "+fmt.Sprintf("%p", &completed)+" src/util/timeit.cpp:57\n") != 1 {
		t.Fatal("completed report and imp deletion", report.String(), trace.String())
	}
}

func TestTimeitImpMemoryTraceProcessFinalizeGuard(t *testing.T) {
	process := memory.NewProcessAccounting()
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: arena.NewBudget(1024), Process: process})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if first, err := runtime.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatal(first, err)
	}
	if err := runtime.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	if err := tags.BindMemoryProcess(process); err != nil {
		t.Fatal(err)
	}
	tags.Enable("memory")
	var scope Timeit
	if err := scope.Construct(true, TimeitConfig{Trace: tags, Output: &bytes.Buffer{},
		Process: process, Clock: func() (uint64, error) { return 0, nil }}); err != nil {
		t.Fatal(err)
	}
	if active, err := process.BeginFinalize(); err != nil || !active {
		t.Fatal(active, err)
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(trace.String(), "dealloc "+fmt.Sprintf("%p", &scope)+" src/util/timeit.cpp:57\n") {
		t.Fatal("deallocation traced during process finalization", trace.String())
	}
	if err := process.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
}
