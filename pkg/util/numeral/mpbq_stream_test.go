package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type binaryRationalRefusingWriter struct {
	bytes.Buffer
	writes int
	stop   int
	err    error
}

func (w *binaryRationalRefusingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.stop {
		return 0, w.err
	}
	return w.Buffer.Write(p)
}

func (w *binaryRationalRefusingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestBinaryRationalSourceStreamInsertionsAndRefusalPrefixes(t *testing.T) {
	m, _ := binaryManager(t)
	refusal := errors.New("refused binary rational insertion")
	for _, tc := range []struct {
		name     string
		value    BinaryRational
		pretty   bool
		want     string
		prefixes []string
	}{
		{"integer", NewBinaryInt(-7), false, "-7", []string{""}},
		{"half", NewBinaryIntPower(-3, 1), false, "-3/2", []string{"", "-3"}},
		{"power", NewBinaryIntPower(-3, 4), false, "-3/2^4", []string{"", "-3", "-3/2", "-3/2^"}},
		{"pretty", NewBinaryIntPower(-3, 4), true, "-3/2<sup>4</sup>", []string{"", "-3", "-3/2", "-3/2<sup>", "-3/2<sup>4"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			show := m.DisplayTo
			if tc.pretty {
				show = m.DisplayPPTo
			}
			out := &binaryRationalRefusingWriter{}
			serviceMust(t, show(out, &tc.value))
			if out.String() != tc.want || out.writes != len(tc.prefixes) {
				t.Fatal("source spelling and insertion count", out.String(), out.writes)
			}
			for i, prefix := range tc.prefixes {
				out := &binaryRationalRefusingWriter{stop: i + 1, err: refusal}
				if err := show(out, &tc.value); !errors.Is(err, refusal) || out.String() != prefix || out.writes != i+1 {
					t.Fatalf("write %d: %q, %d writes, %v", i+1, out.String(), out.writes, err)
				}
			}
			short := &binaryRationalRefusingWriter{stop: len(tc.prefixes)}
			if err := show(short, &tc.value); !errors.Is(err, io.ErrShortWrite) || short.String() != tc.prefixes[len(tc.prefixes)-1] {
				t.Fatal("short final insertion", short.String(), err)
			}
		})
	}
	a := NewBinaryInt(1)
	if err := m.DisplayTo(nil, &a); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := m.DisplayTo(&bytes.Buffer{}, nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil numeral", err)
	}
}
