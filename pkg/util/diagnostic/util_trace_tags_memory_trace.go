// Source: src/util/trace.cpp:28-33; src/util/memory_manager.cpp:202-211.
package diagnostic

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// The source allocation is traced before g_enabled_trace_tags is assigned.
// Thus only enable_all_trace(true), not a tag in the new table, can enable it.
func (t *TraceTags) traceTagTableAllocated() {
	if !sourceDebug || !t.ShouldTrace("memory") {
		return
	}
	_ = t.EmitConditional("memory", "allocate", "src/util/memory_manager.cpp", 209,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{
				"alloc ", fmt.Sprintf("%p", &t.tags), " src/util/trace.cpp:31 str_hashtable ",
				strconv.FormatUint(uint64(reflect.TypeOf(map[string]struct{}{}).Size()), 10), "\n",
			} {
				_, _ = io.WriteString(w, part)
			}
			return nil
		})
}
