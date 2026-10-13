package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type binaryIntervalRefusingWriter struct {
	bytes.Buffer
	writes int
	stop   int
	err    error
}

func (w *binaryIntervalRefusingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.stop {
		return 0, w.err
	}
	return w.Buffer.Write(p)
}

func (w *binaryIntervalRefusingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestBinaryIntervalSourceStreamOrderAndRefusalPrefixes(t *testing.T) {
	m := intervalManager(t)
	var a BinaryInterval
	defer m.Del(&a)
	lo, hi := NewBinaryIntPower(-3, 1), NewBinaryIntPower(5, 2)
	serviceMust(t, m.SetBounds(&a, &lo, &hi))
	if text, err := m.String(&a); err != nil || text != "(-3/2, 5/2^2)" {
		t.Fatal("staged interval spelling", text, err)
	}
	w := &binaryIntervalRefusingWriter{}
	serviceMust(t, m.Display(w, &a))
	if w.String() != "(-3/2, 5/2^2)" || w.writes != 5 {
		t.Fatal("source stream insertion", w.String(), w.writes)
	}
	refusal := errors.New("refused interval insertion")
	for i, prefix := range []string{"", "(", "(-3/2", "(-3/2, ", "(-3/2, 5/2^2"} {
		w := &binaryIntervalRefusingWriter{stop: i + 1, err: refusal}
		if err := m.Display(w, &a); !errors.Is(err, refusal) || w.String() != prefix || w.writes != i+1 {
			t.Fatalf("write %d: %q, %d writes, %v", i+1, w.String(), w.writes, err)
		}
	}
	short := &binaryIntervalRefusingWriter{stop: 4}
	if err := m.Display(short, &a); !errors.Is(err, io.ErrShortWrite) || short.String() != "(-3/2, " {
		t.Fatal("short upper insertion", short.String(), err)
	}
	if err := m.Display(nil, &a); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := m.Display(&bytes.Buffer{}, nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil interval", err)
	}
}
