package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedPointAddSubSourceTraceOrderAndShortcuts(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	zero, _ := m.New()
	one, _ := m.New()
	two, _ := m.New()
	out, _ := m.New()
	t.Cleanup(func() {
		for _, v := range []*FixedPoint{zero, one, two, out} {
			if e := v.Close(); e != nil {
				t.Error(e)
			}
		}
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	if err := m.SetInt64(one, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(two, 2); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.Add(one, two, out); err != nil || output.Len() != 0 {
		t.Fatal("disabled trace", err, output.String())
	}
	tags.Enable("mpfx_trace")
	tags.Enable("mpfx")
	const head = "-------- [mpfx] add_sub src/util/mpfx.cpp:"
	const foot = "------------------------------------------------\n"
	for _, tc := range []struct {
		name    string
		a, b    *FixedPoint
		sub     bool
		want    string
		wantNum string
	}{
		{"add", one, two, false, "[mpfx] 1 + 2 == " + head + "368 ---------\nadd(1, 2)\n" + foot + head + "403 ---------\nresult: 3\n" + foot + "3\n", "3"},
		{"sub", one, two, true, "[mpfx] 1 - 2 == " + head + "368 ---------\nsub(1, 2)\n" + foot + head + "403 ---------\nresult: -1\n" + foot + "-1\n", "-1"},
		{"zero-left", zero, two, false, "[mpfx] 0 + 2 == 2\n", "2"},
		{"zero-result", one, one, true, "[mpfx] 1 - 1 == " + head + "368 ---------\nsub(1, 1)\n" + foot + head + "403 ---------\nresult: 0\n" + foot + "0\n", "0"},
	} {
		output.Reset()
		if tc.sub {
			err = m.Sub(tc.a, tc.b, out)
		} else {
			err = m.Add(tc.a, tc.b, out)
		}
		if err != nil {
			t.Fatal(tc.name, err)
		}
		gotNum, err := m.String(out)
		if err != nil || gotNum != tc.wantNum {
			t.Fatal(tc.name, gotNum, err)
		}
		want := tc.want
		if !tags.ShouldTrace("mpfx_trace") {
			want = ""
		}
		if got := output.String(); got != want {
			t.Fatalf("%s trace: got %q, want %q", tc.name, got, want)
		}
	}
}
