package polynomial

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestMonomialConstructorBindsMemoryChunkTrace(t *testing.T) {
	budget := arena.NewBudget(1 << 22)
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := NewMonomialManager(MonomialManagerConfig{
		Budget:  budget,
		Memory:  runtime,
		Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true},
		Trace:   tags,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err = runtime.Close(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("retained constructor charge", budget.Used())
	}
	if tags.ShouldTrace("memory") {
		for _, want := range []string{"alloc 0x", "src/util/small_object_allocator.cpp:132 chunk 8192\n", "dealloc 0x", "src/util/small_object_allocator.cpp:44\n"} {
			if !strings.Contains(output.String(), want) {
				t.Fatal("monomial constructor trace", want, output.String())
			}
		}
	} else if output.Len() != 0 {
		t.Fatal("erased constructor trace", output.String())
	}
}
