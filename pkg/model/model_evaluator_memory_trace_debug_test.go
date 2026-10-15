//go:build g3debug && g3trace

package model_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelEvaluatorImpMemoryTraceConstructionCleanupAndClose(t *testing.T) {
	_, md, _, _, p, config := evaluatorFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	config.Trace = tags
	ev, err := model.NewModelEvaluator(md.Core(), config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, ev.Close()) })
	astTestMust(t, ev.Cleanup(p))
	astTestMust(t, ev.Close())

	id := fmt.Sprintf("%p", ev)
	size := strconv.FormatUint(uint64(reflect.TypeOf(*ev).Size()), 10)
	record := func(operation string, wrapperLine, siteLine int, body string) string {
		return "-------- [memory] " + operation + " src/util/memory_manager.cpp:" + strconv.Itoa(wrapperLine) + " ---------\n" +
			body + strconv.Itoa(siteLine) + "\n------------------------------------------------\n"
	}
	// The allocated Go owner/header extent is printed after the source site.
	want := "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" +
		"alloc " + id + "/0 src/model/model_evaluator.cpp:523 imp " + size + "\n------------------------------------------------\n" +
		record("deallocate", 204, 556, "dealloc "+id+"/0 src/model/model_evaluator.cpp:") +
		"-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" +
		"alloc " + id + "/1 src/model/model_evaluator.cpp:557 imp " + size + "\n------------------------------------------------\n" +
		record("deallocate", 204, 531, "dealloc "+id+"/1 src/model/model_evaluator.cpp:")
	if got := output.String(); got != want {
		t.Fatalf("source evaluator imp records:\n got %q\nwant %q", got, want)
	}
}
