package lp

import (
	"bytes"
	"errors"
	"io"
	"math/rand/v2"
	"testing"
)

type larConstraintPrefixWriter struct {
	output    bytes.Buffer
	remaining int
	err       error
}

func (w *larConstraintPrefixWriter) Write(p []byte) (int, error) {
	if len(p) <= w.remaining {
		w.remaining -= len(p)
		return w.output.Write(p)
	}
	n, _ := w.output.Write(p[:w.remaining])
	w.remaining = 0
	return n, w.err
}

func TestLARBoundWitnessSourceValueSemantics(t *testing.T) {
	r := rand.New(rand.NewPCG(661, 993))
	for _, row := range []uint32{0, 1, 71, InvalidLARIndex - 1, InvalidLARIndex} {
		w := NewLARBoundWitnesses(row)
		if w.LowBoundWitness != InvalidLARIndex || w.UpperBoundWitness != InvalidLARIndex || w.Row != row {
			t.Fatal(w)
		}
		for i := 0; i < 100; i++ {
			w.LowBoundWitness = r.Uint32()
			w.UpperBoundWitness = r.Uint32()
			copy := w
			if copy != w {
				t.Fatal("copy")
			}
			copy.LowBoundWitness++
			if copy == w {
				t.Fatal("lower witness omitted")
			}
			copy = w
			copy.UpperBoundWitness++
			if copy == w {
				t.Fatal("upper witness omitted")
			}
			copy = w
			copy.Row++
			if copy == w {
				t.Fatal("row omitted")
			}
		}
	}
}

func TestLARBoundWitnessKindStreamAndFailures(t *testing.T) {
	failure := errors.New("constraint stream refused")
	for _, tc := range []struct {
		kind LConstraintKind
		text string
	}{
		{LConstraintLE, "<="}, {LConstraintLT, "<"}, {LConstraintGE, ">="}, {LConstraintGT, ">"}, {LConstraintEQ, "="}, {-3, "??"}, {3, "??"}, {2147483647, "??"},
	} {
		var output bytes.Buffer
		if err := WriteLConstraintKind(&output, tc.kind); err != nil || output.String() != tc.text {
			t.Fatal(output.String(), err)
		}
		for accepted := 0; accepted < len(tc.text); accepted++ {
			w := &larConstraintPrefixWriter{remaining: accepted, err: failure}
			if err := WriteLConstraintKind(w, tc.kind); !errors.Is(err, failure) || w.output.String() != tc.text[:accepted] {
				t.Fatal(w.output.String(), err)
			}
			short := &larConstraintPrefixWriter{remaining: accepted}
			if err := WriteLConstraintKind(short, tc.kind); !errors.Is(err, io.ErrShortWrite) || short.output.String() != tc.text[:accepted] {
				t.Fatal("short kind stream", short.output.String(), err)
			}
		}
	}
}
