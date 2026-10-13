package numeral

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpffLessTraceWriter struct {
	parts     []string
	failDigit string
	armed     bool
	flushes   int
}

func (w *mpffLessTraceWriter) Write(p []byte) (int, error) {
	part := string(p)
	if w.armed && part == w.failDigit && w.failDigit != "" {
		return 0, io.ErrClosedPipe
	}
	w.parts = append(w.parts, part)
	if part == "(" {
		w.armed = true
	}
	return len(p), nil
}

func (w *mpffLessTraceWriter) Flush() error { w.flushes++; return nil }

func TestFastFloatLessComputedTraceSourceInsertions(t *testing.T) {
	m := fastManager(t, 2)
	one, two := NewFastFloat(), NewFastFloat()
	if err := m.SetInt32(&one, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&two, 2); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		a, b *FastFloat
		want bool
		fail bool
	}{{&one, &two, true, false}, {&two, &one, false, false}, {&two, &one, false, true}} {
		w := &mpffLessTraceWriter{}
		if tc.fail {
			w.failDigit = "0"
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpff_trace")
		if !tags.ShouldTrace("mpff_trace") {
			break
		}
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		got, err := m.Less(tc.a, tc.b)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
		at := -1
		for i, part := range w.parts {
			if part == "(" {
				at = i
				break
			}
		}
		if at < 0 || w.flushes != 2 {
			t.Fatalf("computed result or source flush missing: %q, %d", w.parts, w.flushes)
		}
		if tc.fail {
			if len(w.parts) != at+1 || !strings.HasSuffix(strings.Join(w.parts, ""), "(") {
				t.Fatalf("refused boolean wrote suffix: %q", w.parts)
			}
		} else {
			want := traceBoolDigit(tc.want)
			if len(w.parts) < at+3 || w.parts[at+1] != want || w.parts[at+2] != " == 1)\n" {
				t.Fatalf("computed source insertions: %q", w.parts)
			}
		}
	}
	if err := m.Del(&one); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&two); err != nil {
		t.Fatal(err)
	}
}

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
