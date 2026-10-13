package interval

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type shortExtendedWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *shortExtendedWriter) Write(p []byte) (int, error) {
	available := w.limit - w.buf.Len()
	if available < len(p) {
		if available < 0 {
			available = 0
		}
		return w.buf.Write(p[:available])
	}
	return w.buf.Write(p)
}

func assertExtendedShortWrites(t *testing.T, want string, display func(io.Writer) error) {
	t.Helper()
	for limit := 0; limit <= len(want); limit++ {
		w := &shortExtendedWriter{limit: limit}
		err := display(w)
		if got := w.buf.String(); got != want[:limit] {
			t.Fatalf("limit %d: got prefix %q, want %q", limit, got, want[:limit])
		}
		if limit == len(want) {
			if err != nil {
				t.Fatalf("complete output: %v", err)
			}
		} else if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("limit %d: got %v, want short write", limit, err)
		}
	}
}

func TestExtendedInfinityDisplaysShortWrites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		want    string
		display func(io.Writer) error
	}{
		{"mpq minus", "-oo", func(w io.Writer) error { return (RationalExtended{}).Display(w, nil, MinusInfinity) }},
		{"mpff plus", "+oo", func(w io.Writer) error { return (FastExtended{}).Display(w, nil, PlusInfinity) }},
		{"mpfx minus", "-oo", func(w io.Writer) error { return (FixedExtended{}).Display(w, nil, MinusInfinity) }},
		{"mpbq pretty plus", "+&infin;", func(w io.Writer) error { return (BinaryExtended{}).Display(w, nil, PlusInfinity, true) }},
	} {
		t.Run(tc.name, func(t *testing.T) { assertExtendedShortWrites(t, tc.want, tc.display) })
	}
}
