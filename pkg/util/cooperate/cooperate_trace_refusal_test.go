//go:build g3trace

package cooperate

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type cooperationTraceRefusal struct {
	output        bytes.Buffer
	calls, failAt int
}

func (w *cooperationTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errors.New("trace refused")
	}
	return w.output.Write(p)
}
func (w *cooperationTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestCooperationCheckpointTraceSourceInsertionPrefixes(t *testing.T) {
	const detailHeader = "-------- [cooperate_detail] checkpoint src/util/cooperate.cpp:56 ---------\n"
	const footer = "------------------------------------------------\n"
	const taskHeader = "-------- [cooperate] checkpoint src/util/cooperate.cpp:57 ---------\n"
	detail := [...]string{"sls", ", tid: ", "0", "\n"}
	task := [...]string{"moving to task: ", "sls", "\n"}
	for _, site := range []struct {
		parts  []string
		base   int
		prefix string
	}{
		{detail[:], 10, detailHeader},
		{task[:], 24, detailHeader + "sls, tid: 0\n" + footer + taskHeader},
	} {
		for refused := range site.parts {
			c, _ := cooperationTestOwner(t)
			writer := &cooperationTraceRefusal{failAt: site.base + refused}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("cooperate_detail")
			tags.Enable("cooperate")
			if err := c.BindTrace(tags); err != nil {
				t.Fatal(err)
			}
			if err := c.BeginSection(); err != nil {
				t.Fatal(err)
			}
			if err := c.Checkpoint("sls"); err != nil {
				t.Fatal(err)
			}
			if err := c.EndSection(); err != nil {
				t.Fatal(err)
			}
			want := site.prefix
			for _, part := range site.parts[:refused] {
				want += part
			}
			if got := writer.output.String(); got != want || c.OwnerThread != 0 {
				t.Fatalf("refusal %d: got %q, want %q; owner %d", refused, got, want, c.OwnerThread)
			}
		}
	}
}
