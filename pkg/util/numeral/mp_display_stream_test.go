package numeral

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type numeralDisplayWrites struct {
	bytes.Buffer
	parts   []string
	at      int
	refusal error
}

func (w *numeralDisplayWrites) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	if len(w.parts) == w.at {
		if w.refusal != nil {
			return 0, w.refusal
		}
		return w.Buffer.Write(p[:len(p)-1])
	}
	return w.Buffer.Write(p)
}

func (w *numeralDisplayWrites) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func testNumeralDisplayParts(t *testing.T, display func(io.Writer) error, want string, parts []string) {
	t.Helper()
	full := &numeralDisplayWrites{}
	serviceMust(t, display(full))
	if full.String() != want || !reflect.DeepEqual(full.parts, parts) {
		t.Fatalf("display = (%q, %q), want (%q, %q)", full.String(), full.parts, want, parts)
	}
	for at := range parts {
		var before string
		for _, part := range parts[:at] {
			before += part
		}
		short := &numeralDisplayWrites{at: at + 1}
		err := display(short)
		prefix := before + parts[at][:len(parts[at])-1]
		if !errors.Is(err, io.ErrShortWrite) || short.String() != prefix || len(short.parts) != at+1 {
			t.Fatalf("short insertion %d = (%q, %d, %v)", at+1, short.String(), len(short.parts), err)
		}
		refusal := errors.New("refused insertion")
		refused := &numeralDisplayWrites{at: at + 1, refusal: refusal}
		err = display(refused)
		if !errors.Is(err, refusal) || refused.String() != before || len(refused.parts) != at+1 {
			t.Fatalf("refused insertion %d = (%q, %d, %v)", at+1, refused.String(), len(refused.parts), err)
		}
	}
}

func TestIntegerAndRationalDisplaySourceInsertionOrder(t *testing.T) {
	m, _ := rationalManager(t, false)
	value := NewRational()
	t.Cleanup(func() { serviceMust(t, m.Del(&value)) })
	for _, tc := range []struct {
		name, value string
		integer     []string
		rational    []string
	}{
		{"small negative fraction", "-3/2", []string{"-3"}, []string{"-3", "/", "2"}},
		{"large negative fraction", "-4294967296/7", []string{"-", "4294967296"}, []string{"-", "4294967296", "/", "7"}},
		{"large denominator", "1/4294967296", []string{"1"}, []string{"1", "/", "4294967296"}},
		{"integer", "7", []string{"7"}, []string{"7"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serviceMust(t, m.SetString(&value, tc.value))
			num, err := m.Z.String(&value.Num)
			serviceMust(t, err)
			testNumeralDisplayParts(t, func(w io.Writer) error { return m.Z.DisplayTo(w, &value.Num) }, num, tc.integer)
			whole, err := m.String(&value)
			serviceMust(t, err)
			testNumeralDisplayParts(t, func(w io.Writer) error { return m.DisplayTo(w, &value) }, whole, tc.rational)
		})
	}
	if err := m.Z.DisplayTo(nil, &value.Num); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil integer writer", err)
	}
	if err := m.DisplayTo(nil, &value); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil rational writer", err)
	}
}
