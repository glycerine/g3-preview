//go:build g3debug && g3trace

package api

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAPIConfigSourceMemoryWrapperOrder(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	c, err := mkConfigWithTrace(tags)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, engine.FinalizeDefaultProcess()) })
	ptr := fmt.Sprintf("%p", c.params)
	size := reflect.TypeOf(smtcore.ContextParams{}).Size()
	satOK(t, c.Close())
	want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/api/api_config_params.cpp:73 context_params %d\n------------------------------------------------\n", ptr, size) +
		fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/api/api_config_params.cpp:79\n------------------------------------------------\n", ptr)
	if got := output.String(); got != want {
		t.Fatalf("config memory wrapper records:\n got %q\nwant %q", got, want)
	}
	if err := c.Close(); err != ErrConfigOwner || output.String() != want {
		t.Fatal("closed config repeated deallocation trace", err, output.String())
	}
}
