package numeral

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpffRationalTraceWriter struct {
	parts        []string
	failExponent string
	flushes      int
}

func (w *mpffRationalTraceWriter) Write(p []byte) (int, error) {
	part := string(p)
	if part == w.failExponent && w.failExponent != "" {
		return 0, io.ErrClosedPipe
	}
	w.parts = append(w.parts, part)
	return len(p), nil
}

func (w *mpffRationalTraceWriter) Flush() error {
	w.flushes++
	return nil
}

func TestFastFloatToRationalTraceSourceExponentInsertions(t *testing.T) {
	m := fastManager(t, 2)
	s, err := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var a FastFloat
	if err := m.SetFraction32(&a, 3, 2); err != nil {
		t.Fatal(err)
	}
	defer m.Del(&a)
	for _, fail := range []bool{false, true} {
		w := &mpffRationalTraceWriter{}
		exponent := strconv.FormatInt(int64(a.exponent), 10)
		if fail {
			w.failExponent = exponent
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpff_to_mpq")
		if !tags.ShouldTrace("mpff_to_mpq") {
			return
		}
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		out := NewRational()
		if err := m.ToRational(&a, s.Manager, &out); err != nil {
			t.Fatal(err)
		}
		if err := s.Manager.Del(&out); err != nil {
			t.Fatal(err)
		}
		at := -1
		for i, part := range w.parts {
			if part == "\nexp: " {
				at = i
				break
			}
		}
		if at < 0 || w.flushes != 1 {
			t.Fatalf("source exponent prefix or flush missing: %q, %d", w.parts, w.flushes)
		}
		if fail {
			if at != len(w.parts)-1 || !strings.HasSuffix(strings.Join(w.parts, ""), "\nexp: ") {
				t.Fatalf("refused exponent wrote later insertions: %q", w.parts)
			}
		} else if at+2 >= len(w.parts) || w.parts[at+1] != exponent || w.parts[at+2] != "\n" {
			t.Fatalf("source exponent insertions merged: %q", w.parts)
		}
	}
}

func TestFastFloatToRationalRetainedTraceExponentBranches(t *testing.T) {
	m := fastManager(t, 2)
	s, err := NewRationalService(serviceConfig(m.config.Storage.Descriptors.Budget))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_to_mpq")
	compiled := tags.ShouldTrace("mpff_to_mpq")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		sign int
		set  func(*FastFloat) error
	}{
		{"negative", -1, func(a *FastFloat) error { return m.SetFraction32(a, 3, 2) }},
		{"zero", 0, func(a *FastFloat) error { return m.SetUint64(a, 1<<63) }},
		{"positive", 1, func(a *FastFloat) error {
			if err := m.SetUint64(a, 1<<63); err != nil {
				return err
			}
			return m.Mul(a, a, a)
		}},
	} {
		var a FastFloat
		out := NewRational()
		if err := tc.set(&a); err != nil {
			t.Fatal(tc.name, err)
		}
		if got := signOfExponent(a.exponent); got != tc.sign {
			t.Fatal(tc.name, "exponent sign", got, a.exponent)
		}
		formatted, err := m.String(&a)
		if err != nil {
			t.Fatal(tc.name, err)
		}
		output.Reset()
		if err := m.ToRational(&a, s.Manager, &out); err != nil {
			t.Fatal(tc.name, err)
		}
		want := ""
		if compiled {
			want = "-------- [mpff_to_mpq] to_mpq_core src/util/mpff.cpp:1119 ---------\n" +
				"to_mpq: " + formatted + "\nexp: " + strconv.FormatInt(int64(a.exponent), 10) + "\n" +
				"------------------------------------------------\n"
		}
		if got := output.String(); got != want {
			t.Fatalf("%s: trace got %q, want %q", tc.name, got, want)
		}
		if err := s.Manager.Del(&out); err != nil {
			t.Fatal(err)
		}
		if err := m.Del(&a); err != nil {
			t.Fatal(err)
		}
	}
}

func signOfExponent(x int32) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
