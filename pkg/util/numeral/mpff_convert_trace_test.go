package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatSetIntegerRetainedTraceBranches(t *testing.T) {
	m := fastManager(t, 2)
	s, err := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	z := s.Manager.Z
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff")
	compiled := tags.ShouldTrace("mpff")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	header := func(line string) string {
		return "-------- [mpff] set_core src/util/mpff.cpp:" + line + " ---------\n"
	}
	footer := "------------------------------------------------\n"
	for _, tc := range []struct {
		name, input, branchLine, branchBody string
		up                                  bool
	}{
		{"zero", "0", "331", "is_int64 0\n", true},
		{"signed", "-3", "331", "is_int64 -3\n", false},
		{"unsigned", "18446744073709551615", "335", "is_uint64\n", true},
		{"big-up", "1208925819614629174706177", "345", "w words: 1 0 65536 \n", true},
		{"big-down", "-1208925819614629174706177", "345", "w words: 1 0 65536 \n", false},
	} {
		var integer Int
		var value FastFloat
		if err := z.SetString(&integer, tc.input); err != nil {
			t.Fatal(tc.name, err)
		}
		if err := m.SetRounding(tc.up); err != nil {
			t.Fatal(tc.name, err)
		}
		output.Reset()
		if err := m.SetInteger(&value, z, &integer); err != nil {
			t.Fatal(tc.name, err)
		}
		raw, err := m.RawString(&value)
		if err != nil {
			t.Fatal(tc.name, err)
		}
		want := ""
		if compiled {
			want = header("329") + "mpz->mpff\n" + tc.input + "\n" + footer +
				header(tc.branchLine) + tc.branchBody + footer +
				header("371") + "mpz->mpff result:\n" + raw + "\n" + footer
		}
		if got := output.String(); got != want {
			t.Fatalf("%s: trace got %q, want %q", tc.name, got, want)
		}
		if tc.branchLine == "345" && !strings.Contains(raw, "*2^") {
			t.Fatal("multiword result raw exponent", tc.name, raw)
		}
		if err := m.Del(&value); err != nil {
			t.Fatal(err)
		}
		if err := z.Del(&integer); err != nil {
			t.Fatal(err)
		}
	}
}
