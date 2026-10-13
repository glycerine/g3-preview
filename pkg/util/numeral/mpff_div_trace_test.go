package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatDivTraceSourceShortcutsAndQuotient(t *testing.T) {
	m := fastManager(t, 2)
	zero, two, three, six, out := NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat()
	for _, assignment := range []struct {
		value *FastFloat
		word  int32
	}{{&zero, 0}, {&two, 2}, {&three, 3}, {&six, 6}} {
		if err := m.SetInt32(assignment.value, assignment.word); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	tags.Enable("mpff_div")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.SetRounding(true); err != nil {
		t.Fatal(err)
	}
	if err := m.Div(&six, &zero, &out); !errors.Is(err, ErrFastFloatDivisionByZero) || output.Len() != 0 {
		t.Fatal("divisor check must precede trace", err, output.String())
	}
	for _, test := range []struct {
		name     string
		a, b     *FastFloat
		wantText string
		want     string
	}{
		{"zero-numerator", &zero, &three, "0", "[mpff] (0) / (3) <= 0\n"},
		{"divisor-two", &six, &two, "3", "[mpff] (6) / (2) <= 3\n"},
		{"general", &six, &three, "2",
			"[mpff] (6) / (3) <= " +
				"-------- [mpff_div] div src/util/mpff.cpp:871 ---------\n" +
				"00000001 00000000 00000000 \n" +
				"------------------------------------------------\n" +
				"2\n"},
	} {
		output.Reset()
		if err := m.Div(test.a, test.b, &out); err != nil {
			t.Fatal(test.name, err)
		}
		if got, err := m.String(&out); err != nil || got != test.wantText {
			t.Fatal("quotient", test.name, got, err)
		}
		want := test.want
		if !compiled {
			want = ""
		}
		if output.String() != want {
			t.Fatal("source division trace", test.name, output.String())
		}
	}
	for _, value := range []*FastFloat{&zero, &two, &three, &six, &out} {
		if err := m.Del(value); err != nil {
			t.Fatal(err)
		}
	}
}
