//go:build g3debug && g3trace

package shell

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

func TestShellScopedTimerMemoryTraceSourceLifetime(t *testing.T) {
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: arena.NewBudget(4096), Process: p})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatal(first, err)
	}
	if err := r.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	s := NewTimeoutService(nil)
	defer s.Close()
	if err := s.BindMemoryTrace(tags, p); err != nil {
		t.Fatal(err)
	}
	for _, disabled := range []int64{0, int64(math.MaxUint32)} {
		if err := s.SetTimeout(disabled); err != nil {
			t.Fatal(err)
		}
	}
	if out.Len() != 0 {
		t.Fatal("disabled timer allocated imp", out.String())
	}
	if err := s.SetTimeout(1_000_000); err != nil {
		t.Fatal(err)
	}
	firstTimer := s.timer
	firstAlloc := "alloc " + fmt.Sprintf("%p", firstTimer) + " src/util/scoped_timer.cpp:257 imp " + strconv.FormatUint(uint64(unsafe.Sizeof(*firstTimer)), 10) + "\n"
	if strings.Count(out.String(), firstAlloc) != 1 {
		t.Fatal("source allocation", out.String(), firstAlloc)
	}
	if err := s.SetTimeout(1_000_000); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dealloc "+fmt.Sprintf("%p", firstTimer)+" src/util/scoped_timer.cpp:264\n") {
		t.Fatal("replacement did not release first imp", out.String())
	}
	secondTimer := s.timer
	if secondTimer == firstTimer || strings.Count(out.String(), "[memory] allocate ") != 2 {
		t.Fatal("replacement allocation", out.String())
	}
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatal(active, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "dealloc "+fmt.Sprintf("%p", secondTimer)+" src/util/scoped_timer.cpp:264\n") {
		t.Fatal("source finalization emitted deallocation", out.String())
	}
	if err := p.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
}
