package numeral

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpffIntegerTraceWriter struct {
	parts                  []string
	flushes                int
	refuseWordSpace, words bool
	failed                 bool
}

func (w *mpffIntegerTraceWriter) Write(p []byte) (int, error) {
	part := string(p)
	if w.failed || w.refuseWordSpace && w.words && part == " " {
		w.failed = true
		return 0, io.ErrClosedPipe
	}
	w.parts = append(w.parts, part)
	if part == "w words: " {
		w.words = true
	}
	return len(p), nil
}

func (w *mpffIntegerTraceWriter) Flush() error {
	w.flushes++
	return nil
}

func TestFastFloatSetIntegerTraceSourceInsertionBoundaries(t *testing.T) {
	m := fastManager(t, 2)
	s, err := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	z := s.Manager.Z
	for _, tc := range []struct {
		input, label string
		want         []string
		refuseSpace  bool
	}{
		{"-3", "is_int64 ", []string{"is_int64 ", "-3", "\n"}, false},
		{"1208925819614629174706177", "w words: ", []string{"w words: ", "1", " ", "0", " ", "65536", " ", "\n"}, false},
		{"1208925819614629174706177", "w words: ", []string{"w words: ", "1"}, true},
	} {
		w := &mpffIntegerTraceWriter{refuseWordSpace: tc.refuseSpace}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpff")
		if !tags.ShouldTrace("mpff") {
			return
		}
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		var value Int
		var out FastFloat
		if err := z.SetString(&value, tc.input); err != nil {
			t.Fatal(err)
		}
		if err := m.SetInteger(&out, z, &value); err != nil {
			t.Fatal(err)
		}
		at := -1
		for i, part := range w.parts {
			if part == tc.label {
				at = i
				break
			}
		}
		if at < 0 || at+len(tc.want) > len(w.parts) {
			t.Fatalf("missing source insertions %q in %q", tc.want, w.parts)
		}
		for i, want := range tc.want {
			if got := w.parts[at+i]; got != want {
				t.Fatalf("source insertion %d = %q, want %q; all %q", i, got, want, w.parts)
			}
		}
		if tc.refuseSpace && (!w.failed || !strings.HasSuffix(strings.Join(w.parts, ""), "w words: 1")) {
			t.Fatalf("refused word separator wrote later bytes: %q", w.parts)
		}
		if w.flushes != 3 {
			t.Fatalf("source three trace flushes = %d", w.flushes)
		}
		if err := m.Del(&out); err != nil {
			t.Fatal(err)
		}
		if err := z.Del(&value); err != nil {
			t.Fatal(err)
		}
	}
}

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
