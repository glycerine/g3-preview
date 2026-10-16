// Source: src/sat/sat_mus.cpp:40,61,71,93,120.
package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSATActualMUSSourceTraceTransitions(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, _, _ := satSolverCheckFixture(t)
		for i := 0; i < 4; i++ {
			_, err := s.MkVar(true, true)
			satOK(t, err)
		}
		satOK(t, s.MkClause([]Literal{1, 3, 5}))
		result, err := s.Check([]Literal{0, 2, 4, 6})
		satOK(t, err)
		if result != lbool.False {
			t.Fatal("source conflict", result)
		}
		satOK(t, s.conflict.core.Shrink(0))
		for _, v := range []uint32{0, 2, 4, 6} {
			satOK(t, s.conflict.core.Push(v))
		}
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("sat")
		}
		s.traceTags = tags
		result, err = s.mus.run()
		satOK(t, err)
		if result != lbool.True {
			t.Fatal("source MUS", result)
		}
		got := output.String()
		if !tags.ShouldTrace("sat") {
			if got != "" {
				t.Fatalf("disabled source MUS trace: %q", got)
			}
			continue
		}
		old := "-------- [sat] mus1 src/sat/sat_mus.cpp:61 ---------\n" +
			"old core: 0 1 2 3\n------------------------------------------------\n"
		loop := "-------- [sat] mus1 src/sat/sat_mus.cpp:71 ---------\n" +
			"core: 0 1 2 3\nmus:  \n------------------------------------------------\n"
		if !strings.Contains(got, old+loop) {
			t.Fatalf("source old-core/loop order: %q", got)
		}
		for _, record := range []string{
			"-------- [sat] mus1 src/sat/sat_mus.cpp:93 ---------\nmus: 0 1 2 -3\n------------------------------------------------\n",
			"-------- [sat] mus1 src/sat/sat_mus.cpp:120 ---------\ncore: 2 1 0\n mus: \n------------------------------------------------\n",
			"-------- [sat] set_core src/sat/sat_mus.cpp:40 ---------\nnew core: 0 1 2\n------------------------------------------------\n",
		} {
			if !strings.Contains(got, record) {
				t.Fatalf("missing reached MUS trace %q in %q", record, got)
			}
		}
	}
}
