package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

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
