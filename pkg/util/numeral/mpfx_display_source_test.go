package numeral

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type fixedSourceDisplayWrites struct {
	bytes.Buffer
	parts   []string
	at      int
	refusal error
}

func (w *fixedSourceDisplayWrites) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	if len(w.parts) == w.at {
		if w.refusal != nil {
			return 0, w.refusal
		}
		return w.Buffer.Write(p[:len(p)-1])
	}
	return w.Buffer.Write(p)
}

func (w *fixedSourceDisplayWrites) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestFixedPointDisplaySourceInsertionsAndShortWrites(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	serviceMust(t, err)
	a, err := m.New()
	serviceMust(t, err)
	t.Cleanup(func() {
		serviceMust(t, a.Close())
		serviceMust(t, m.Close())
	})
	for _, tc := range []struct {
		name  string
		num   int64
		den   uint64
		parts []string
	}{
		{"zero", 0, 1, []string{"0"}},
		{"positive integer", 7, 1, []string{"7"}},
		{"negative half", -1, 2, []string{"-", "1", "/2"}},
		{"negative eighths", -3, 8, []string{"-", "3", "/2", "^", "3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serviceMust(t, m.SetRatio(a, tc.num, tc.den))
			want, err := m.String(a)
			serviceMust(t, err)
			full := &fixedSourceDisplayWrites{}
			serviceMust(t, m.Display(full, a))
			if full.String() != want || !reflect.DeepEqual(full.parts, tc.parts) {
				t.Fatalf("display = (%q, %q), want (%q, %q)", full.String(), full.parts, want, tc.parts)
			}
			for at := range tc.parts {
				var before string
				for _, part := range tc.parts[:at] {
					before += part
				}
				short := &fixedSourceDisplayWrites{at: at + 1}
				err := m.Display(short, a)
				prefix := before + tc.parts[at][:len(tc.parts[at])-1]
				if !errors.Is(err, io.ErrShortWrite) || short.String() != prefix || len(short.parts) != at+1 {
					t.Fatalf("short insertion %d = (%q, %d, %v)", at+1, short.String(), len(short.parts), err)
				}
				refusal := errors.New("refused insertion")
				refused := &fixedSourceDisplayWrites{at: at + 1, refusal: refusal}
				err = m.Display(refused, a)
				if !errors.Is(err, refusal) || refused.String() != before || len(refused.parts) != at+1 {
					t.Fatalf("refused insertion %d = (%q, %d, %v)", at+1, refused.String(), len(refused.parts), err)
				}
			}
		})
	}
	if err := m.Display(nil, a); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
}
