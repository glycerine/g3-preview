//go:build g3debug

package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBinaryRetainedDebugArithmeticTraceSites(t *testing.T) {
	m, _ := binaryManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpbq_bug")
	compiled := tags.ShouldTrace("mpbq_bug")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a := NewBinaryIntPower(3, 2)
	one := NewInt(1)
	var result BinaryRational
	output.Reset()
	if err := m.AddInteger(&a, &one, &result); err != nil {
		t.Fatal(err)
	}
	binaryText(t, m, &result, "7/2^2")
	want := ""
	if compiled {
		want = "-------- [mpbq_bug] add src/util/mpbq.cpp:167 ---------\n" +
			"add a: 3/4, b: 1, r: 7/4, expected: 7/4\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("integer add trace got %q, want %q", got, want)
	}
	if err := m.Del(&result); err != nil {
		t.Fatal(err)
	}
	b := NewBinaryIntPower(1, 1)
	output.Reset()
	if err := m.Sub(&a, &b, &a); err != nil {
		t.Fatal(err)
	}
	binaryText(t, m, &a, "1/2^2")
	want = ""
	if compiled {
		want = "-------- [mpbq_bug] sub src/util/mpbq.cpp:195 ---------\n" +
			"sub a: 3/4, b: 1/2, r: 1/4, expected: 1/4\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("binary sub alias trace got %q, want %q", got, want)
	}
	for _, v := range []*BinaryRational{&a, &b} {
		if err := m.Del(v); err != nil {
			t.Fatal(err)
		}
	}
}
