package polynomial

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicIntervalRefusingWriter struct {
	bytes.Buffer
	writes int
	stop   int
	err    error
}

func (w *algebraicIntervalRefusingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.stop {
		return 0, w.err
	}
	return w.Buffer.Write(p)
}

func (w *algebraicIntervalRefusingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestAlgebraicDisplayIntervalSourceBranchesAndWritePrefixes(t *testing.T) {
	c, p := algebraicFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	var basic Algebraic
	serviceMust(t, c.SetRational(&basic, &q))
	t.Cleanup(func() {
		serviceMust(t, c.Del(&basic))
		serviceMust(t, c.config.Rationals.Del(&q))
	})

	poly := algebraicPolynomial(t, p, -2, 0, 1)
	view, err := p.wholeView(poly)
	serviceMust(t, err)
	lo, hi := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, view, &lo, &hi, false))
	t.Cleanup(func() { serviceMust(t, c.Del(&root)) })

	refusal := errors.New("refused algebraic interval insertion")
	for _, tc := range []struct {
		name     string
		value    *Algebraic
		want     string
		prefixes []string
	}{
		{"zero", &Algebraic{}, "[0, 0]", []string{"", "[", "[0", "[0, ", "[0, 0"}},
		{"basic", &basic, "[3/2, 3/2]", []string{"", "[", "[3", "[3/", "[3/2", "[3/2, ", "[3/2, 3", "[3/2, 3/", "[3/2, 3/2"}},
		{"root", &root, "(1, 2)", []string{"", "(", "(1", "(1, ", "(1, 2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &algebraicIntervalRefusingWriter{}
			serviceMust(t, c.DisplayInterval(out, tc.value))
			if out.String() != tc.want || out.writes != len(tc.prefixes) {
				t.Fatal("source spelling and insertion count", out.String(), out.writes)
			}
			for i, prefix := range tc.prefixes {
				out := &algebraicIntervalRefusingWriter{stop: i + 1, err: refusal}
				if err := c.DisplayInterval(out, tc.value); !errors.Is(err, refusal) || out.String() != prefix || out.writes != i+1 {
					t.Fatalf("write %d: %q, %d writes, %v", i+1, out.String(), out.writes, err)
				}
			}
			short := &algebraicIntervalRefusingWriter{stop: 4}
			if err := c.DisplayInterval(short, tc.value); !errors.Is(err, io.ErrShortWrite) || short.String() != tc.prefixes[3] {
				t.Fatal("short interval insertion", short.String(), err)
			}
		})
	}
	if err := c.DisplayInterval(nil, &basic); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := c.DisplayInterval(&bytes.Buffer{}, nil); !errors.Is(err, ErrAlgebraicOwner) {
		t.Fatal("nil value", err)
	}
}
