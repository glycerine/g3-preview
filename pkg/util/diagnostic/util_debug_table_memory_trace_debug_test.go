//go:build g3debug && g3trace

package diagnostic

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestDebugTableMemoryTraceFollowsLazyLifetime(t *testing.T) {
	d, _, _ := debugFixture(t)
	var out bytes.Buffer
	tags := NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	previous, err := d.BindMemoryTrace(tags)
	if err != nil || previous != nil {
		t.Fatal("debug trace binding", previous, err)
	}
	if err := d.DisableDebug("missing"); err != nil {
		t.Fatal(err)
	}
	allocation := "alloc " + fmt.Sprintf("%p", &d.Tags) +
		" src/util/debug.cpp:48 str_hashtable " + strconv.FormatUint(uint64(reflect.TypeOf(map[string]struct{}{}).Size()), 10) + "\n"
	if !strings.Contains(out.String(), allocation) || strings.Count(out.String(), "\nalloc ") != 1 {
		t.Fatal("lazy table allocation", out.String())
	}
	if err := d.EnableDebug("x"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "\nalloc ") != 1 {
		t.Fatal("existing debug table reallocated", out.String())
	}
	if err := d.FinalizeDebug(); err != nil {
		t.Fatal(err)
	}
	deallocation := "dealloc " + fmt.Sprintf("%p", &d.Tags) + " src/util/debug.cpp:53\n"
	if !strings.Contains(out.String(), deallocation) || strings.Count(out.String(), "\ndealloc ") != 1 {
		t.Fatal("debug table finalization", out.String())
	}
	if err := d.FinalizeDebug(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "\ndealloc ") != 1 {
		t.Fatal("null debug table deallocated", out.String())
	}
	if _, err := d.IsDebugEnabled("missing"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "\nalloc ") != 2 {
		t.Fatal("lazy table not reconstructed", out.String())
	}
}

func TestDebugTableDeallocationSuppressesMemoryDuringProcessFinalize(t *testing.T) {
	d, _, _ := debugFixture(t)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: arena.NewBudget(1024), Process: p})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var out bytes.Buffer
	tags := NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	if err := tags.BindMemoryProcess(p); err != nil {
		t.Fatal(err)
	}
	if _, err := d.BindMemoryTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := d.EnableDebug("x"); err != nil {
		t.Fatal(err)
	}
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatal("source initialization", first, err)
	}
	if err := r.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatal("source finalization", active, err)
	}
	if err := d.FinalizeDebug(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "dealloc ") || d.Tags != nil {
		t.Fatal("debug memory trace during source finalization", out.String())
	}
	if err := p.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
}
