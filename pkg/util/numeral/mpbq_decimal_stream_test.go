package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryRationalDecimalSourceStreamAndRawScope(t *testing.T) {
	m, _ := binaryManager(t)
	refusal := errors.New("refused decimal insertion")
	for _, tc := range []struct {
		name     string
		value    BinaryRational
		prec     uint32
		want     string
		prefixes []string
	}{
		{"integer", NewBinaryInt(-7), 4, "-7", []string{""}},
		{"exact", NewBinaryIntPower(-3, 3), 4, "-0.375", []string{"", "-", "-0", "-0.", "-0.3", "-0.37"}},
		{"uncertain", NewBinaryIntPower(1, 6), 3, "0.015?", []string{"", "0", "0.", "0.0", "0.01", "0.015"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &binaryRationalRefusingWriter{}
			serviceMust(t, m.DisplayDecimalTo(out, &tc.value, tc.prec))
			if out.String() != tc.want || out.writes != len(tc.prefixes) || m.DisplayState().Pending {
				t.Fatal("source spelling, writes or raw scope", out.String(), out.writes, m.DisplayState())
			}
			for i, prefix := range tc.prefixes {
				out := &binaryRationalRefusingWriter{stop: i + 1, err: refusal}
				if err := m.DisplayDecimalTo(out, &tc.value, tc.prec); !errors.Is(err, refusal) || out.String() != prefix || out.writes != i+1 {
					t.Fatalf("write %d: %q, %d writes, %v", i+1, out.String(), out.writes, err)
				}
				if tc.value.K() != 0 && !m.DisplayState().Pending {
					t.Fatal("refused raw local scope not retained")
				}
				serviceMust(t, m.CleanupDisplay())
			}
			short := &binaryRationalRefusingWriter{stop: len(tc.prefixes)}
			if err := m.DisplayDecimalTo(short, &tc.value, tc.prec); !errors.Is(err, io.ErrShortWrite) || short.String() != tc.prefixes[len(tc.prefixes)-1] {
				t.Fatal("short final insertion", short.String(), err)
			}
			serviceMust(t, m.CleanupDisplay())
		})
	}
	a := NewBinaryInt(1)
	if err := m.DisplayDecimalTo(nil, &a, 2); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := m.DisplayDecimalTo(&bytes.Buffer{}, nil, 2); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil numeral", err)
	}
}
