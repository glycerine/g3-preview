//go:build g3trace

package mincut

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type minCutTraceRefusal struct {
	output        bytes.Buffer
	calls, failAt int
}

func (w *minCutTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errors.New("trace refused")
	}
	return w.output.Write(p)
}
func (w *minCutTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestMinCutTraceSourceInsertionPrefixes(t *testing.T) {
	const header = "-------- [spacer.mincut] add_edge src/util/min_cut.cpp:35 ---------\n"
	parts := [...]string{"adding edge (", "2", ",", "1", ")\n"}
	for refused := range parts {
		writer := &minCutTraceRefusal{failAt: 10 + refused}
		tags := diagnostic.NewTraceTagsWithWriter(writer)
		tags.Enable("spacer.mincut")
		m := New()
		node := m.NewNode()
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		if err := m.AddEdge(node, 1); err != nil {
			t.Fatal(err)
		}
		want := header
		for _, part := range parts[:refused] {
			want += part
		}
		if got := writer.output.String(); got != want || len(m.edges[node]) != 1 {
			t.Fatalf("refusal %d: got %q, want %q; edges %v", refused, got, want, m.edges[node])
		}
	}
}
