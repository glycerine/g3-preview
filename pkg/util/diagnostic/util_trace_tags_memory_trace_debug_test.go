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

func TestTraceTagTableAllocationFollowsSourceGate(t *testing.T) {
	var out bytes.Buffer
	tags := NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	if out.Len() != 0 {
		t.Fatal("newly inserted memory tag traced its own table allocation")
	}
	tags.Finalize()
	tags.EnableAll(true)
	tags.Disable("other")
	want := "alloc " + fmt.Sprintf("%p", &tags.tags) +
		" src/util/trace.cpp:31 str_hashtable " + strconv.FormatUint(uint64(reflect.TypeOf(map[string]struct{}{}).Size()), 10) + "\n"
	if !strings.Contains(out.String(), want) || strings.Count(out.String(), "alloc ") != 1 {
		t.Fatal("source allocation trace", out.String(), want)
	}
	tags.Enable("other")
	if strings.Count(out.String(), "alloc ") != 1 {
		t.Fatal("existing table was allocated again")
	}
	tags.Finalize()
	if !strings.Contains(out.String(), "dealloc "+fmt.Sprintf("%p", &tags.tags)+" src/util/trace.cpp:37\n") ||
		strings.Count(out.String(), "dealloc ") != 1 || !tags.all {
		t.Fatal("source table finalization trace and all flag", out.String(), tags.all)
	}
	tags.Finalize()
	if strings.Count(out.String(), "dealloc ") != 1 {
		t.Fatal("null source table deallocated", out.String())
	}
}

func TestTraceTagTableFinalizationSuppressesMemoryDuringProcessFinalize(t *testing.T) {
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: arena.NewBudget(1024), Process: p})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var out bytes.Buffer
	tags := NewTraceTagsWithWriter(&out)
	if err := tags.BindMemoryProcess(p); err != nil {
		t.Fatal(err)
	}
	tags.EnableAll(true)
	tags.Enable("memory")
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatal("source initialization", first, err)
	}
	if err := r.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatal("source finalization", active, err)
	}
	tags.Finalize()
	if strings.Contains(out.String(), "dealloc ") || tags.tags != nil {
		t.Fatal("memory trace during source finalization", out.String())
	}
	if err := p.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
}
