package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatLessRetainedShortTraceBranches(t *testing.T) {
	m := fastManager(t, 2)
	values := [5]FastFloat{}
	for i, v := range []int32{0, 1, -1, 2, -2} {
		if err := m.SetInt32(&values[i], v); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		a, b   int
		result bool
		record string
	}{
		{0, 0, false, "[mpff] (0 < 0) == (1 == 0)\n"},
		{0, 2, false, "[mpff] (0 < -1) == (1 == 0)\n"},
		{0, 1, true, "[mpff] (0 < 1) == (1 == 1)\n"},
		{2, 0, true, "[mpff] (-1 < 0) == (1 == 1)\n"},
		{1, 0, false, "[mpff] (1 < 0) == (1 == 0)\n"},
		{2, 1, true, "[mpff] (-1 < 1) == (1 == 1)\n"},
		{1, 2, false, "[mpff] (1 < -1) == (1 == 0)\n"},
		{4, 2, true, "[mpff] (-2 < -1) == (1 == 1)\n"},
		{2, 4, false, "[mpff] (-1 < -2) == (0 == 1)\n"},
		{1, 3, true, "[mpff] (1 < 2) == (1 == 1)\n"},
		{3, 1, false, "[mpff] (2 < 1) == (0 == 1)\n"},
	} {
		output.Reset()
		got, err := m.Less(&values[test.a], &values[test.b])
		if err != nil || got != test.result {
			t.Fatal("source less result", test, got, err)
		}
		want := test.record
		if !compiled {
			want = ""
		}
		if output.String() != want {
			t.Fatal("source less STRACE", test, output.String())
		}
	}
	for i := range values {
		if err := m.Del(&values[i]); err != nil {
			t.Fatal(err)
		}
	}
}
