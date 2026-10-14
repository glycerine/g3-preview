package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMyMPZSqrtRetainedShortTraceBranches(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		input int32
		want  string
		first string
		mid   string
		last  string
	}{
		{1, "8", "", "", ""},
		{64, "8", "SIG = 64 LOWER = 8 UPPER = 512\n", "MID = 260 PROD = 67600\n", "choosing lower\n"},
		{65, "9", "SIG = 65 LOWER = 8 UPPER = 520\n", "MID = 264 PROD = 69696\n", "choosing upper\n"},
	} {
		output.Reset()
		in, out := NewInt(test.input), NewInt(0)
		if err := MyMPZSqrt(m, 2, false, &in, &out); err != nil || integerString(t, m, &out) != test.want {
			t.Fatal("source root branch", test.input, err)
		}
		if err := m.Del(&out); err != nil {
			t.Fatal(err)
		}
		if !compiled || test.first == "" {
			if output.Len() != 0 {
				t.Fatal("erased or equal-bounds trace", test.input, output.String())
			}
			continue
		}
		got := output.String()
		if !strings.HasPrefix(got, test.first) || !strings.Contains(got, test.mid) || !strings.HasSuffix(got, test.last) || strings.Count(got, "choosing ") != 1 {
			t.Fatal("source bounds/mid/choice trace order", test.input, got)
		}
	}
}

func TestMyMPZSqrtReachedEndlFlushes(t *testing.T) {
	m, _ := integerManager(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	for _, input := range []int32{64, 65} {
		w.events = nil
		w.flushes = 0
		in, out := NewInt(input), NewInt(0)
		serviceMust(t, MyMPZSqrt(m, 2, false, &in, &out))
		serviceMust(t, m.Del(&out))
		if !tags.ShouldTrace("mpf_dbg") {
			if len(w.events) != 0 {
				t.Fatalf("erased sqrt trace for %d: %q", input, w.events)
			}
			continue
		}
		joined := strings.Join(w.events, "|")
		if !strings.Contains(joined, "write:SIG = |") || !strings.Contains(joined, "write:MID = |") || !strings.Contains(joined, "write:choosing ") {
			t.Fatalf("missing reached sqrt branch for %d: %q", input, w.events)
		}
		newlines := 0
		for i, event := range w.events {
			if event != "write:\n" {
				continue
			}
			newlines++
			if i+1 == len(w.events) || w.events[i+1] != "flush" {
				t.Fatalf("sqrt std::endl for %d did not flush: %q", input, w.events)
			}
		}
		if newlines < 3 || w.flushes != newlines {
			t.Fatalf("sqrt std::endl count for %d: %d newlines, %d flushes", input, newlines, w.flushes)
		}
	}
}
