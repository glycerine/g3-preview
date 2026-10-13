package mincut

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMinCutRetainedEdgeTraceAndCut(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m := New()
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a := m.NewNode()
	if err := m.AddEdge(0, a); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatal("disabled tag emitted", output.String())
	}
	tags.Enable("spacer.mincut")
	if err := m.AddEdgeCapacity(a, 1, 3); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("spacer.mincut") {
		const want = "-------- [spacer.mincut] add_edge src/util/min_cut.cpp:35 ---------\n" +
			"adding edge (2,1)\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("source add-edge TRACE: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased trace emitted", output.String())
	}
	cut, err := m.ComputeMinCut(nil)
	if err != nil || len(cut) != 1 || cut[0] != a {
		t.Fatal("trace changed min-cut graph", cut, err)
	}
}
