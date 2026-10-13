package bitset

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBitVectorResizeSourceTraceSites(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bit_vector")
	v := new(Vector)
	v.BindTrace(tags)
	v.Resize(35, true)
	const header1 = "-------- [bit_vector] resize src/util/bit_vector.cpp:43 ---------\n"
	const header2 = "-------- [bit_vector] resize src/util/bit_vector.cpp:67 ---------\n"
	const footer = "------------------------------------------------\n"
	if tags.ShouldTrace("bit_vector") {
		want := header1 + "expanding: 35 capacity: 0 num words: 2\n" + footer +
			header2 + "num_bits: 0\nbwidx:    0\newidx:    2\npos:      0\nmask:     0\ncval:     -1\n" + footer
		if got := output.String(); got != want {
			t.Fatalf("source resize TRACE records: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased resize TRACE emitted", output.String())
	}
	output.Reset()
	v.Resize(1, false)
	if output.Len() != 0 {
		t.Fatal("shrink emitted growth TRACE", output.String())
	}
	capacity := cap(v.words)
	v.Resize(34, false)
	if tags.ShouldTrace("bit_vector") {
		want := header1 + fmt.Sprintf("expanding: 34 capacity: %d num words: 2\n", capacity) + footer +
			header2 + "num_bits: 1\nbwidx:    0\newidx:    2\npos:      1\nmask:     1\ncval:     0\n" + footer
		if got := output.String(); got != want {
			t.Fatalf("source second resize TRACE records: got %q, want %q", got, want)
		}
	}
	if v.Size() != 34 || !v.Get(0) || v.Get(1) {
		t.Fatal("trace changed packed bit growth")
	}
}
