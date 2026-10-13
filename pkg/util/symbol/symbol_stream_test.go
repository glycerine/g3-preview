package symbol

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type symbolRefusingWriter struct {
	bytes.Buffer
	writes int
	stop   int
	err    error
}

type symbolReentrantWriter struct {
	bytes.Buffer
	onWrite func() error
}

func (w *symbolReentrantWriter) Write(p []byte) (int, error) {
	if e := w.onWrite(); e != nil {
		return 0, e
	}
	return w.Buffer.Write(p)
}

func (w *symbolReentrantWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *symbolRefusingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.stop {
		return 0, w.err
	}
	return w.Buffer.Write(p)
}

func (w *symbolRefusingWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestSymbolSourceStreamInsertionAndWriteBoundary(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	spelling := internSymbol(t, m, "ab\x00ignored")
	for _, tc := range []struct {
		symbol Symbol
		want   string
		writes int
	}{
		{Symbol{}, "null", 1},
		{spelling, "ab", 1},
		{Numerical(0), "k!0", 2},
		{Numerical(^uint32(0)), "k!-1", 2},
	} {
		w := &symbolRefusingWriter{}
		if err := m.WriteDisplay(w, tc.symbol); err != nil || w.String() != tc.want || w.writes != tc.writes {
			t.Fatalf("symbol stream %v: %q, %d writes, %v", tc.symbol, w.String(), w.writes, err)
		}
	}
	refused := errors.New("refused symbol write")
	for _, stop := range []int{1, 2} {
		w := &symbolRefusingWriter{stop: stop, err: refused}
		if err := m.WriteDisplay(w, Numerical(7)); !errors.Is(err, refused) || w.writes != stop {
			t.Fatalf("refused write %d: %q, %d writes, %v", stop, w.String(), w.writes, err)
		}
		want := ""
		if stop == 2 {
			want = "k!"
		}
		if w.String() != want {
			t.Fatalf("refused prefix %d: %q", stop, w.String())
		}
	}
	short := &symbolRefusingWriter{stop: 2}
	if err := m.WriteDisplay(short, Numerical(9)); !errors.Is(err, io.ErrShortWrite) || short.String() != "k!" {
		t.Fatalf("short numeric write: %q, %v", short.String(), err)
	}
	marked, err := spelling.Mark()
	if err != nil {
		t.Fatal(err)
	}
	w := &symbolRefusingWriter{}
	if err := m.WriteDisplay(w, marked); !errors.Is(err, ErrInvalid) || w.writes != 0 {
		t.Fatalf("marked stream: %q, %v", w.String(), err)
	}
	if err := m.WriteDisplay(nil, spelling); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil stream", err)
	}
}

func TestSymbolStreamWriterCanReenterManager(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	spelling := internSymbol(t, m, "alpha")
	for _, tc := range []struct {
		symbol Symbol
		want   string
	}{
		{spelling, "alpha"},
		{Numerical(17), "k!17"},
		{Symbol{}, "null"},
	} {
		writes := 0
		w := &symbolReentrantWriter{onWrite: func() error {
			writes++
			if _, e := m.Display(spelling); e != nil {
				return e
			}
			_, e := m.Intern("during-output")
			return e
		}}
		if e := m.WriteDisplay(w, tc.symbol); e != nil || w.String() != tc.want || writes == 0 {
			t.Fatalf("symbol %v: output %q, writes %d, error %v", tc.symbol, w.String(), writes, e)
		}
	}
}
