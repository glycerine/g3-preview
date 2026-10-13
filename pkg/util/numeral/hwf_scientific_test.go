package numeral

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

type hwfShortWriter struct{ prefix []byte }

func (w *hwfShortWriter) Write(p []byte) (int, error) {
	n := len(p) - 1
	w.prefix = append(w.prefix, p[:n]...)
	return n, nil
}

func TestHardwareFloatSourceScientificTraceDisplay(t *testing.T) {
	m, e := NewHardwareFloatNumeralManager(NewHardwareFloatManager(), 11, 53)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range []struct {
		value float64
		want  string
	}{
		{1, "1.000000e+00"},
		{-1.25, "-1.250000e+00"},
		{math.Copysign(0, -1), "-0.000000e+00"},
		{0.001, "1.000000e-03"},
		{math.Inf(1), "inf"},
	} {
		x := HardwareFloat{row.value}
		s, e := m.ToString(&x)
		if e != nil || s != row.want {
			t.Fatalf("value %v: %q, want %q: %v", row.value, s, row.want, e)
		}
		var out bytes.Buffer
		if e := m.DisplayTo(&out, &x); e != nil || out.String() != row.want {
			t.Fatalf("display value %v: %q, want %q: %v", row.value, out.String(), row.want, e)
		}
	}
	x := HardwareFloat{1}
	short := &hwfShortWriter{}
	if e := m.DisplayTo(short, &x); !errors.Is(e, io.ErrShortWrite) || string(short.prefix) != "1.000000e+0" {
		t.Fatal("short display prefix", string(short.prefix), e)
	}
}
