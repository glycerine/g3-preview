//go:build g3trace

package backtrack

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type unionFindTraceRefusal struct {
	output        bytes.Buffer
	calls, failAt int
}

func (w *unionFindTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errors.New("trace refused")
	}
	return w.output.Write(p)
}
func (w *unionFindTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestUnionFindMergeTraceSourceInsertionPrefixes(t *testing.T) {
	const header = "-------- [union_find] merge src/util/union_find.h:122 ---------\n"
	parts := [...]string{"merging ", "0", " ", "1", "\n"}
	for refused := range parts {
		writer := &unionFindTraceRefusal{failAt: 10 + refused}
		tags := diagnostic.NewTraceTagsWithWriter(writer)
		tags.Enable("union_find")
		u := NewDefaultUnionFind()
		u.MkVar()
		u.MkVar()
		u.BindTrace(tags)
		u.Merge(0, 1)
		want := header
		for _, part := range parts[:refused] {
			want += part
		}
		if got := writer.output.String(); got != want || u.Find(0) != u.Find(1) {
			t.Fatalf("refusal %d: got %q, want %q; roots %v", refused, got, want, u.find)
		}
	}
}
