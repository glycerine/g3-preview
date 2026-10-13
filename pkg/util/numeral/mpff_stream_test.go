package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type fastDisplayWrites struct {
	bytes.Buffer
	parts   []string
	at      int
	refusal error
}

func (w *fastDisplayWrites) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	if len(w.parts) == w.at {
		if w.refusal != nil {
			return 0, w.refusal
		}
		return w.Buffer.Write(p[:len(p)-1])
	}
	return w.Buffer.Write(p)
}

func (w *fastDisplayWrites) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestFastFloatDisplayToSourceInsertionsAndShortWrites(t *testing.T) {
	m := fastManager(t, 2)
	owner, err := NewScopedFastFloat(m)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, owner.Close()) })
	a, err := owner.Get()
	serviceMust(t, err)
	for _, tc := range []struct {
		name       string
		value, exp int32
		parts      []string
	}{
		{"positive short exponent", 3, 1, []string{"*", "2"}},
		{"negative large positive exponent", -3, 65, []string{"-", "*2", "^", "65"}},
		{"negative large divisor", -3, -150, []string{"-", "/2", "^", "88"}},
		{"negative no exponent", -3, 0, []string{"-"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serviceMust(t, m.SetInt32(a, tc.value))
			serviceMust(t, m.SetExponent(a, tc.exp))
			want, err := m.String(a)
			serviceMust(t, err)
			full := &fastDisplayWrites{}
			serviceMust(t, m.DisplayTo(full, a))
			if got := full.String(); got != want {
				t.Fatalf("display = %q, string = %q", got, want)
			}
			if len(full.parts) != len(tc.parts)+1 {
				t.Fatalf("insertions = %q", full.parts)
			}
			digitsIndex := 0
			suffix := tc.parts
			if tc.value < 0 {
				if full.parts[0] != "-" {
					t.Fatalf("sign insertion = %q", full.parts)
				}
				digitsIndex = 1
				suffix = suffix[1:]
			}
			if full.parts[digitsIndex] == "" {
				t.Fatalf("empty significand insertion = %q", full.parts)
			}
			for _, c := range full.parts[digitsIndex] {
				if c < '0' || c > '9' {
					t.Fatalf("nondecimal significand insertion = %q", full.parts)
				}
			}
			for i, part := range suffix {
				if full.parts[digitsIndex+1+i] != part {
					t.Fatalf("insertions = %q, suffix = %q", full.parts, suffix)
				}
			}
			for at := range full.parts {
				w := &fastDisplayWrites{at: at + 1}
				err := m.DisplayTo(w, a)
				var before string
				for _, part := range full.parts[:at] {
					before += part
				}
				prefix := before + full.parts[at][:len(full.parts[at])-1]
				if !errors.Is(err, io.ErrShortWrite) || w.String() != prefix || len(w.parts) != at+1 {
					t.Fatalf("short insertion %d = (%q, %d, %v), want %q", at+1, w.String(), len(w.parts), err, prefix)
				}
				refusal := errors.New("refused insertion")
				refused := &fastDisplayWrites{at: at + 1, refusal: refusal}
				err = m.DisplayTo(refused, a)
				if !errors.Is(err, refusal) || refused.String() != before || len(refused.parts) != at+1 {
					t.Fatalf("refused insertion %d = (%q, %d, %v)", at+1, refused.String(), len(refused.parts), err)
				}
			}
		})
	}
	if err := m.DisplayTo(nil, a); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
}
