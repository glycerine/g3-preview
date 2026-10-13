package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatNumeralSourceRationalString(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var x ArbitraryFloat
	for _, c := range []struct {
		n, d int32
		want string
	}{{1, 4, "1/4"}, {-3, 2, "-3/2"}, {2, 1, "2"}} {
		if err := m.SetFractionSigned(&x, c.n, c.d); err != nil {
			t.Fatal(err)
		}
		got, err := m.ToRationalString(&x)
		if err != nil || got != c.want {
			t.Fatalf("%d/%d: %q, %v", c.n, c.d, got, err)
		}
	}
	if err := core.Del(&x); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ToRationalString(&x); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed string", err)
	}
}

type mpfNumeralShortWriter struct{ buf bytes.Buffer }

func (w *mpfNumeralShortWriter) String() string { return w.buf.String() }

func (w *mpfNumeralShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return w.buf.Write(p[:1])
}

func TestArbitraryFloatNumeralSourceStringAndDisplay(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var x ArbitraryFloat
	if err := m.SetInt32(&x, 1); err != nil {
		t.Fatal(err)
	}
	got, err := m.ToString(&x)
	if err != nil || got != "1.0 0" {
		t.Fatal("source format", got, err)
	}
	again, err := m.ToString(&x)
	if err != nil || again != got {
		t.Fatal("repeated source format", again, err)
	}
	var out bytes.Buffer
	if err := m.DisplayTo(&out, &x); err != nil || out.String() != got {
		t.Fatal("display", out.String(), err)
	}
	var short mpfNumeralShortWriter
	if err := m.DisplayTo(&short, &x); !errors.Is(err, io.ErrShortWrite) || short.String() != got[:1] {
		t.Fatal("short writer", short.String(), err)
	}
	if err := core.Del(&x); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}
