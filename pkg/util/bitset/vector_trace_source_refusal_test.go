//go:build g3trace

package bitset

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type resizeTraceRefusalWriter struct {
	output bytes.Buffer
	calls  int
	failAt int
	cause  error
}

func (w *resizeTraceRefusalWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, w.cause
	}
	return w.output.Write(p)
}
func (w *resizeTraceRefusalWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestBitVectorResizeTraceBodySourceInsertionPrefixes(t *testing.T) {
	const header1 = "-------- [bit_vector] resize src/util/bit_vector.cpp:43 ---------\n"
	const header2 = "-------- [bit_vector] resize src/util/bit_vector.cpp:67 ---------\n"
	const footer = "------------------------------------------------\n"
	start := []string{"expanding: ", "35", " capacity: ", "0", " num words: ", "2", "\n"}
	fields := []string{
		"num_bits: ", "0", "\n", "bwidx:    ", "0", "\n",
		"ewidx:    ", "2", "\n", "pos:      ", "0", "\n",
		"mask:     ", "0", "\n", "cval:     ", "-1", "\n",
	}
	for _, site := range []struct {
		parts  []string
		base   int
		prefix string
	}{
		{start, 10, header1},
		{fields, 27, header1 + "expanding: 35 capacity: 0 num words: 2\n" + footer + header2},
	} {
		for refused := range site.parts {
			w := &resizeTraceRefusalWriter{failAt: site.base + refused, cause: errors.New("trace refused")}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("bit_vector")
			v := new(Vector)
			v.BindTrace(tags)
			v.Resize(35, true)
			want := site.prefix
			for _, part := range site.parts[:refused] {
				want += part
			}
			if got := w.output.String(); got != want || v.Size() != 35 || !v.Get(34) {
				t.Fatal("source refused insertion", site.base, refused, got, want, v.Size())
			}
		}
	}
}
