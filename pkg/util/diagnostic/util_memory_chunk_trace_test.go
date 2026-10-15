package diagnostic

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestMemoryChunkTraceSourceSitesAndFinalizeGuard(t *testing.T) {
	budget := arena.NewBudget(1 << 22)
	process := umemory.NewProcessAccounting()
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Process: process, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	a, err := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{
		Runtime:    runtime,
		Profile:    umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true},
		ChunkTrace: NewMemoryChunkTrace(tags, runtime),
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := a.Allocate(8)
	if err != nil {
		t.Fatal(err)
	}
	address := r.LogicalAddress() - 16 // two pointer-width chunk header fields
	if err = a.Reset(); err != nil {
		t.Fatal(err)
	}
	if traceCompiled {
		alloc := fmt.Sprintf("alloc 0x%x src/util/small_object_allocator.cpp:132 chunk 8192\n", address)
		dealloc := fmt.Sprintf("dealloc 0x%x src/util/small_object_allocator.cpp:60\n", address)
		if !strings.Contains(output.String(), alloc) || !strings.Contains(output.String(), dealloc) ||
			!strings.Contains(output.String(), "allocate src/util/memory_manager.cpp:209") ||
			!strings.Contains(output.String(), "deallocate src/util/memory_manager.cpp:204") {
			t.Fatal("missing retained source records", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased trace wrote output", output.String())
	}
	r, err = a.Allocate(8)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Deallocate(8, r); err != nil {
		t.Fatal(err)
	}
	active, err := runtime.BeginSourceInitialize(uint64(^uint32(0)))
	if err != nil || !active {
		t.Fatal("begin process", active, err)
	}
	if err = runtime.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
	active, err = process.BeginFinalize()
	if err != nil || !active {
		t.Fatal("begin finalization", active, err)
	}
	before := output.Len()
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	if output.Len() != before {
		t.Fatal("finalizing deallocation traced", output.String()[before:])
	}
	if err = process.CompleteFinalize(); err != nil {
		t.Fatal(err)
	}
	consolidating, err := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{
		Runtime:    runtime,
		Profile:    umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true},
		ChunkTrace: NewMemoryChunkTrace(tags, runtime),
	})
	if err != nil {
		t.Fatal(err)
	}
	var objects []umemory.ObjectRef
	for i := 0; i < 340; i++ { // 8176-byte payload holds exactly 340 24-byte objects
		obj, e := consolidating.Allocate(24)
		if e != nil {
			t.Fatal(e)
		}
		objects = append(objects, obj)
	}
	consolidatedAddress := objects[0].LogicalAddress() - 16
	for _, obj := range objects {
		if err = consolidating.Deallocate(24, obj); err != nil {
			t.Fatal(err)
		}
	}
	if err = consolidating.Consolidate(); err != nil {
		t.Fatal(err)
	}
	if err = consolidating.Close(); err != nil {
		t.Fatal(err)
	}
	closing, err := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{
		Runtime:    runtime,
		Profile:    umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true},
		ChunkTrace: NewMemoryChunkTrace(tags, runtime),
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err = closing.Allocate(8)
	if err != nil {
		t.Fatal(err)
	}
	closingAddress := r.LogicalAddress() - 16
	if err = closing.Deallocate(8, r); err != nil {
		t.Fatal(err)
	}
	if err = closing.Close(); err != nil {
		t.Fatal(err)
	}
	if traceCompiled {
		for _, want := range []string{
			fmt.Sprintf("dealloc 0x%x src/util/small_object_allocator.cpp:221\n", consolidatedAddress),
			fmt.Sprintf("dealloc 0x%x src/util/small_object_allocator.cpp:44\n", closingAddress),
		} {
			if !strings.Contains(output.String(), want) {
				t.Fatal("missing retained chunk release", want, output.String())
			}
		}
	}
	if err = runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("retained allocation", budget.Used())
	}
}
