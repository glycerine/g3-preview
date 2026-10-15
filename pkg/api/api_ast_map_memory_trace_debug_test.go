//go:build g3debug && g3trace

package api

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAPIASTMapSourceMemoryWrapperReleaseSites(t *testing.T) {
	for _, tc := range []struct {
		name string
		line int
	}{
		{"zero-reference", 54},
		{"context-destructor", 116},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := astTestManager(t)
			c, err := newAPIContextResults(m, false)
			astTestMust(t, err)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("memory")
			c.trace = tags
			a, err := newAPIASTMapObject(c)
			astTestMust(t, err)
			if tc.line == 54 {
				astTestMust(t, c.saveObject(&a.object))
				astTestMust(t, a.object.incRef())
				astTestMust(t, c.assignLastObject(nil))
				astTestMust(t, a.object.decRef())
			}
			astTestMust(t, c.Close())
			ptr := fmt.Sprintf("%p", a)
			size := reflect.TypeOf(apiASTMapObject{}).Size()
			want := fmt.Sprintf("-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\nalloc %s src/api/api_ast_map.cpp:37 Z3_ast_map_ref %d\n------------------------------------------------\n", ptr, size) +
				fmt.Sprintf("-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\ndealloc %s src/api/api_context.cpp:%d\n------------------------------------------------\n", ptr, tc.line)
			if got := output.String(); got != want {
				t.Fatalf("source AST-map memory records:\n got %q\nwant %q", got, want)
			}
		})
	}
}
