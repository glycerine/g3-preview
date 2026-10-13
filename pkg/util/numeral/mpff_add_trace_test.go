package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatAddSubOuterAndInnerTraceOrder(t *testing.T) {
	m := fastManager(t, 2)
	zero, one, two, out := NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat()
	for _, assignment := range []struct {
		value *FastFloat
		word  int32
	}{{&zero, 0}, {&one, 1}, {&two, 2}} {
		if err := m.SetInt32(assignment.value, assignment.word); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	tags.Enable("mpff")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	header := func(line int) string {
		if line == 646 {
			return "-------- [mpff] add_sub src/util/mpff.cpp:646 ---------\n"
		}
		return "-------- [mpff] add_sub src/util/mpff.cpp:765 ---------\n"
	}
	footer := "------------------------------------------------\n"
	for _, test := range []struct {
		name           string
		a, b           *FastFloat
		sub, roundPlus bool
		want           string
	}{
		{"add", &one, &two, false, true, "[mpff] 1 + 2 <= " + header(646) + "add(1, 2)\n" + footer + header(765) + "result: 3\n" + footer + "3\n"},
		{"sub", &one, &two, true, true, "[mpff] 1 - 2 <= " + header(646) + "sub(1, 2)\n" + footer + header(765) + "result: -1\n" + footer + "-1\n"},
		{"zero-left", &zero, &two, false, true, "[mpff] 0 + 2 <= 2\n"},
		{"zero-result", &one, &one, true, true, "[mpff] 1 - 1 <= " + header(646) + "sub(1, 1)\n" + footer + header(765) + "result: 0\n" + footer + "0\n"},
		{"round-minus", &one, &two, false, false, "[mpff] 1 + 2 >= " + header(646) + "add(1, 2)\n" + footer + header(765) + "result: 3\n" + footer + "3\n"},
	} {
		output.Reset()
		if err := m.SetRounding(test.roundPlus); err != nil {
			t.Fatal(err)
		}
		var err error
		if test.sub {
			err = m.Sub(test.a, test.b, &out)
		} else {
			err = m.Add(test.a, test.b, &out)
		}
		if err != nil {
			t.Fatal(test.name, err)
		}
		want := test.want
		if !compiled {
			want = ""
		}
		if output.String() != want {
			t.Fatal("source add/sub trace", test.name, output.String())
		}
	}
	for _, value := range []*FastFloat{&zero, &one, &two, &out} {
		if err := m.Del(value); err != nil {
			t.Fatal(err)
		}
	}
}
