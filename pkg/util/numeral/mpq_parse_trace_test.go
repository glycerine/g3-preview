package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalSetStringTraceInsertionAndEndlBoundaries(t *testing.T) {
	s, _ := serviceNew(t)
	m := s.Manager
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpq_set")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a := NewRational()
	if err := m.SetString(&a, "1.25e-2"); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&a); err != nil {
		t.Fatal(err)
	}
	if !tags.ShouldTrace("mpq_set") {
		if len(w.events) != 0 {
			t.Fatal("erased source trace wrote", w.events)
		}
		return
	}
	// The first record has a literal newline. The next two use std::endl,
	// which flushes before TRACE writes its footer and flushes again.
	for _, want := range [][]string{
		{"write:[before] a: ", "write:1", "write:\n"},
		{"write:[exp]: ", "write:2", "write:, str[0]: ", "write:2", "write:\n", "flush"},
		{"write:a: ", "write:5/4", "write:, exp_sign:", "write:1", "write:, exp: ", "write:2", "write: ", "write:100", "write:\n", "flush"},
	} {
		found := false
		for i := 0; i+len(want) <= len(w.events); i++ {
			match := true
			for j, event := range want {
				if w.events[i+j] != event {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing source insertion sequence %q in %q", want, w.events)
		}
	}
	if w.flushes != 5 {
		t.Fatal("three TRACE footer flushes plus two std::endl flushes", w.flushes, w.events)
	}
}

func TestRationalSetStringTraceEndlRefusalKeepsNumericResult(t *testing.T) {
	s, _ := serviceNew(t)
	m := s.Manager
	refusal := errors.New("trace body flush refused")
	w := &mpfEndlWriter{failAt: 2, refusal: refusal}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpq_set")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a := NewRational()
	if err := m.SetString(&a, "1.25e-2"); err != nil {
		t.Fatal(err)
	}
	got, err := m.String(&a)
	if err != nil || got != "1/80" {
		t.Fatal("source parser changed result after TRACE stream failure", got, err)
	}
	if !tags.ShouldTrace("mpq_set") {
		if len(w.events) != 0 {
			t.Fatal("erased source trace wrote", w.events)
		}
	} else {
		if w.flushes != 4 {
			t.Fatal("source endl and reached TRACE footer flush calls", w.events)
		}
		bodyEnd := -1
		for i, event := range w.events {
			if event == "write:\n" {
				bodyEnd = i
			}
		}
		if bodyEnd < 0 || bodyEnd+1 >= len(w.events) {
			t.Fatal("source std::endl missing", w.events)
		}
		for _, event := range w.events[bodyEnd+1:] {
			if event != "flush" {
				t.Fatal("failed source trace stream wrote after endl refusal", w.events)
			}
		}
		before := len(w.events)
		if err := m.SetString(&a, "2e3"); err != nil {
			t.Fatal(err)
		}
		for _, event := range w.events[before:] {
			if event != "flush" {
				t.Fatal("failed source trace stream wrote on later parse", w.events[before:])
			}
		}
	}
	if err := m.Del(&a); err != nil {
		t.Fatal(err)
	}
}

func TestRationalSetStringRetainedTraceBranches(t *testing.T) {
	s, _ := serviceNew(t)
	m := s.Manager
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpq_set")
	compiled := tags.ShouldTrace("mpq_set")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	header := func(line string) string {
		return "-------- [mpq_set] set src/util/mpq.cpp:" + line + " ---------\n"
	}
	footer := "------------------------------------------------\n"
	for _, tc := range []struct {
		name, input, before, extra, value string
		rawZero                           bool
		wantErr                           error
	}{
		{"integer", "12", "12", "", "12", false, nil},
		{"fraction", "3/4", "3", "", "3/4", false, nil},
		{"scientific", "1.25e-2", "1", header("241") + "[exp]: 2, str[0]: 2\n" + footer + header("260") + "a: 5/4, exp_sign:1, exp: 2 100\n" + footer, "1/80", false, nil},
		{"nondigit-exponent", "2e+3x", "2", header("241") + "[exp]: 0, str[0]: -5\n" + footer + header("241") + "[exp]: 3, str[0]: 3\n" + footer + header("241") + "[exp]: 3, str[0]: 72\n" + footer + header("260") + "a: 2, exp_sign:0, exp: 3 1000\n" + footer, "2000", false, nil},
		{"mixed-refusal", "5/2e3", "5", "", "", false, ErrMixedNotation},
		{"raw-zero-denominator", "7/0", "7/0", "", "", true, ErrDivideByZero},
	} {
		a := NewRational()
		if tc.rawZero {
			a.Den = NewInt(0)
		}
		output.Reset()
		err := m.SetString(&a, tc.input)
		if tc.wantErr == nil && err != nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
			t.Fatal(tc.name, err)
		}
		if tc.wantErr == nil {
			got, err := m.String(&a)
			if err != nil || got != tc.value {
				t.Fatal(tc.name, got, err)
			}
		}
		want := ""
		if compiled {
			want = header("203") + "[before] a: " + tc.before + "\n" + footer + tc.extra
		}
		if got := output.String(); got != want {
			t.Fatalf("%s: trace got %q, want %q", tc.name, got, want)
		}
		if err := m.Del(&a); err != nil {
			t.Fatal(tc.name, err)
		}
	}
}
