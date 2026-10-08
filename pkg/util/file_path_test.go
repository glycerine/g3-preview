// Source: src/util/file_path.h:23-35.
package util

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetExtensionSourcePathsAndCStringBoundary(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
		found bool
	}{
		{"", "", false}, {"file", "", false}, {"file.smt2", "smt2", true},
		{"file.tar.gz", "gz", true}, {".profile", "profile", true}, {".", "", true},
		{"..", "", true}, {"file.", "", true}, {"dir.name/file", "name/file", true},
		{"dir.name/file.ext", "ext", true}, {"dir.name\\file", "name\\file", true},
		{"a\x00.b", "", false}, {"a.b\x00.c", "b", true}, {"a.\x00b.c", "", true},
		{"\x00.b", "", false}, {"\xff.\xfe", "\xfe", true}, {"a.日本語", "日本語", true},
	} {
		input := []byte(tc.input)
		saved := bytes.Clone(input)
		got := GetExtension(input)
		if (got != nil) != tc.found || string(got) != tc.want || !bytes.Equal(input, saved) {
			t.Fatalf("%q: got %q (found %v), want %q (found %v)", tc.input, got, got != nil, tc.want, tc.found)
		}
	}
	if GetExtension(nil) != nil {
		t.Fatal("null input")
	}
}

func TestGetExtensionExhaustiveAndBorrowedSuffix(t *testing.T) {
	alphabet := []byte{'a', '.', '/', 0}
	for length := 0; length <= 7; length++ {
		count := 1 << (2 * length)
		for code := 0; code < count; code++ {
			input := make([]byte, length)
			for i := range input {
				input[i] = alphabet[(code>>(2*i))&3]
			}
			text := string(input)
			if end := strings.IndexByte(text, 0); end >= 0 {
				text = text[:end]
			}
			last := strings.LastIndexByte(text, '.')
			got := GetExtension(input)
			if last < 0 {
				if got != nil {
					t.Fatalf("%q: expected no extension", input)
				}
				continue
			}
			if got == nil || string(got) != text[last+1:] {
				t.Fatalf("%q: suffix %q", input, got)
			}
			if len(got) != 0 {
				input[last+1] = 'z'
				if got[0] != 'z' {
					t.Fatal("suffix was copied")
				}
			}
		}
	}
	input := []byte("dir.name/file.ext\x00ignored.more")
	if allocs := testing.AllocsPerRun(1000, func() {
		if got := GetExtension(input); string(got) != "ext" {
			panic("extension")
		}
	}); allocs != 0 {
		t.Fatal("borrowed lookup allocated", allocs)
	}
}
