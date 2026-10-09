// Source: src/test/escaped.cpp:21-35; src/util/util.cpp:122-152.
package escaped

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestEscapedThirteenSourcePrintedFixtures(t *testing.T) {
	type testCase struct {
		in   *string
		trim bool
		want string
	}
	var cases []testCase
	add := func(s string, trim bool, want string) {
		cases = append(cases, testCase{in: &s, trim: trim, want: want})
	}
	add("\"hello\"\"world\"\n\n", false, "\\\"hello\\\"\\\"world\\\"\n\n")
	add("\"hello\"\nworld\"\n\n\n", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\nworld\"\n", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\nworld\"", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\n\"world\"\n\n", false, "\\\"hello\\\"\n\\\"world\\\"\n\n")
	add("\n\n\n", true, "")
	add("\n\n\n", false, "\n\n\n")
	add("\n", true, "")
	add("\n", false, "\n")
	add("", true, "")
	add("", false, "")
	cases = append(cases, testCase{trim: true}, testCase{})
	if len(cases) != 13 {
		t.Fatal(len(cases))
	}
	for i, c := range cases {
		got, err := String(c.in, c.trim, 0)
		if err != nil || got != c.want {
			t.Fatalf("case %d: %q, %v", i, got, err)
		}
	}
}

func TestEscapedIndependentByteIndentAndTrim(t *testing.T) {
	for _, trim := range []bool{false, true} {
		for indent := uint32(0); indent < 5; indent++ {
			for code := 0; code < 256; code++ {
				in := string([]byte{byte(code), '"', '\\', '\n', 'x', '\r', '\n', '\n'})
				plain := in
				if end := strings.IndexByte(plain, 0); end >= 0 {
					plain = plain[:end]
				}
				if trim {
					plain = strings.TrimRight(plain, "\n")
				}
				want := strings.ReplaceAll(plain, "\"", "\\\"")
				want = strings.ReplaceAll(want, "\n", "\n"+strings.Repeat(" ", int(indent)))
				got, err := String(&in, trim, indent)
				if err != nil || got != want {
					t.Fatalf("byte %d trim %v indent %d: %q/%q, %v", code, trim, indent, got, want, err)
				}
			}
		}
	}
}

type refusingWriter struct{ calls int }

func (w *refusingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 3 {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func TestEscapedWriterAndSizeBound(t *testing.T) {
	in := "a\"\nb"
	var out strings.Builder
	if err := Write(&out, &in, false, 2); err != nil || out.String() != "a\\\"\n  b" {
		t.Fatal(out.String(), err)
	}
	var refusing refusingWriter
	if err := Write(&refusing, &in, false, 2); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	newline := "\n"
	if _, err := String(&newline, false, ^uint32(0)); !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
	if err := Write(nil, nil, false, 0); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}
