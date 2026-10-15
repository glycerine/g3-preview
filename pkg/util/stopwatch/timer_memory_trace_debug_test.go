//go:build g3debug && g3trace

package stopwatch

import (
	"bytes"
	"errors"
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

func TestTimerWatchMemoryTraceAllocationBeforeStartAndClose(t *testing.T) {
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	clockFailure := errors.New("clock refused")
	var p *Timer
	p, err := NewTimerWithMemoryTrace(func() (uint64, error) {
		if p != nil {
			t.Fatal("constructor published before first clock read")
		}
		if !strings.Contains(out.String(), "src/util/timer.cpp:25 stopwatch ") {
			t.Fatal("source allocation must precede start", out.String())
		}
		return 0, clockFailure
	}, tags)
	if !errors.Is(err, clockFailure) || p == nil {
		t.Fatal("constructor failure prefix", p, err)
	}
	alloc := "alloc " + fmt.Sprintf("%p", &p.watch) + " src/util/timer.cpp:25 stopwatch " + strconv.FormatUint(uint64(unsafe.Sizeof(p.watch)), 10) + "\n"
	dealloc := "dealloc " + fmt.Sprintf("%p", &p.watch) + " src/util/timer.cpp:30\n"
	if strings.Count(out.String(), alloc) != 1 || strings.Contains(out.String(), dealloc) {
		t.Fatal("watch allocation lifetime", out.String())
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), dealloc) != 1 {
		t.Fatal("watch released more or less than once", out.String())
	}
}

func TestTimerWatchMemoryTraceProcessFinalizeGuard(t *testing.T) {
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
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	if err := tags.BindMemoryProcess(process); err != nil {
		t.Fatal(err)
	}
	tags.Enable("memory")
	p, err := NewTimerWithMemoryTrace(func() (uint64, error) { return 1, nil }, tags)
	if err != nil {
		t.Fatal(err)
	}
	if active, err := process.BeginFinalize(); err != nil || !active {
		t.Fatal(active, err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "dealloc "+fmt.Sprintf("%p", &p.watch)+" src/util/timer.cpp:30\n") {
		t.Fatal("deallocation trace during source process finalization", out.String())
	}
	if err := process.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
}
