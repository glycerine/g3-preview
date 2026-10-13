package numeral

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerLCMRetainedBranchTraceAndAlias(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("lcm_bug")
	compiled := tags.ShouldTrace("lcm_bug")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	const header = "-------- [lcm_bug] lcm src/util/mpz.cpp:%d ---------\n"
	const footer = "------------------------------------------------\n"
	record := func(line int, body string) string { return fmt.Sprintf(header, line) + body + footer }
	var want string
	for _, tc := range []struct {
		a, b, result int32
		alias        bool
		trace        string
	}{
		{6, 1, 6, false, record(988, "1. lcm(6, 1) = 6\n")},
		{1, 6, 6, false, record(992, "2. lcm(1, 6) = 6\n")},
		{7, 7, 7, false, record(992, "2. lcm(7, 7) = 7\n")},
		{6, 12, 12, false, record(997, "gcd(6, 12) = 6\n") + record(1000, "3. lcm(6, 12) = 12\n")},
		{12, 6, 12, false, record(997, "gcd(12, 6) = 6\n") + record(1004, "4. lcm(12, 6) = 12\n")},
		{6, 10, 30, false, record(997, "gcd(6, 10) = 2\n") + record(1011, "5. lcm(6, 10) = 30\n")},
		{6, 10, 30, true, record(997, "gcd(6, 10) = 2\n") + record(1011, "5. lcm(30, 10) = 30\n")},
	} {
		a, b, c := NewInt(tc.a), NewInt(tc.b), NewInt(0)
		out := &c
		if tc.alias {
			out = &a
		}
		if err := m.LCM(&a, &b, out); err != nil {
			t.Fatal(tc, err)
		}
		got, err := m.String(out)
		if err != nil || got != fmt.Sprint(tc.result) {
			t.Fatal(tc, got, err)
		}
		want += tc.trace
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased LCM TRACE emitted", output.String())
		}
	} else if got := output.String(); got != want {
		t.Fatalf("source LCM TRACE\ngot: %q\nwant: %q", got, want)
	}
}
