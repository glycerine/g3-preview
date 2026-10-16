// Source: src/sat/sat_clause.cpp:203,209.
package sat

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satClauseStorageTraceRecord(function, line, body string) string {
	return "-------- [sat] " + function + " src/sat/sat_clause.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATClauseAllocatorSourceAllocationAndDeletionTraces(t *testing.T) {
	for _, learned := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			a := satStore(t, nil)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			if enabled {
				tags.Enable("sat")
			}
			a.traceTags = tags
			c, err := a.MkClause([]Literal{0, 2, 4}, learned)
			satOK(t, err)
			id, err := a.ID(c)
			satOK(t, err)
			clause, err := satTraceClause(a, c)
			satOK(t, err)
			satOK(t, a.Delete(c))
			want := ""
			if tags.ShouldTrace("sat") {
				kind := "a"
				if learned {
					kind = "l"
				}
				idText := strconv.FormatUint(uint64(id), 10)
				want = satClauseStorageTraceRecord("mk_clause", "203", "alloc: "+idText+" "+clause+" "+kind+"\n") +
					satClauseStorageTraceRecord("del_clause", "209", "delete: "+idText+" "+clause+"\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("learned=%v enabled=%v: got %q, want %q", learned, enabled, got, want)
			}
		}
	}
}
