package filepath

// Source: src/util/file_path.h:23-35.

import (
	"strings"
	"testing"
)

func TestExtension(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extension string
		found     bool
	}{
		{"", "", false},
		{"file", "", false},
		{"file.smt2", "smt2", true},
		{"file.tar.gz", "gz", true},
		{".profile", "profile", true},
		{".", "", true},
		{"..", "", true},
		{"file.", "", true},
		{"dir.name/file", "name/file", true},
		{"dir.name/file.ext", "ext", true},
		{"dir.name\\file", "name\\file", true},
		{"a\x00.b", "", false},
		{"a.b\x00.c", "b", true},
		{"a.\x00b.c", "", true},
		{"\x00.b", "", false},
		{"\xff.\xfe", "\xfe", true},
		{"a.日本語", "日本語", true},
	} {
		got, found := Extension(tc.name)
		if got != tc.extension || found != tc.found {
			t.Fatalf("Extension(%q) = %q, %v; want %q, %v", tc.name, got, found, tc.extension, tc.found)
		}
	}
}

func TestExtensionExhaustive(t *testing.T) {
	alphabet := []byte{'a', '.', '/', 0}
	for length := 0; length <= 7; length++ {
		count := 1 << (2 * length)
		for code := 0; code < count; code++ {
			input := make([]byte, length)
			for i := range input {
				input[i] = alphabet[(code>>(2*i))&3]
			}
			visible := string(input)
			if end := strings.IndexByte(visible, 0); end >= 0 {
				visible = visible[:end]
			}
			dot := strings.LastIndexByte(visible, '.')
			got, found := Extension(string(input))
			if found != (dot >= 0) {
				t.Fatalf("Extension(%q) found = %v", input, found)
			}
			if dot >= 0 && got != visible[dot+1:] {
				t.Fatalf("Extension(%q) = %q", input, got)
			}
		}
	}
}
