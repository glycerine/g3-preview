// Source: src/util/stats.h:25-35.
package util

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestPrintStatPositiveGateAndCStringMessage(t *testing.T) {
	var out bytes.Buffer
	serviceMust(t, PrintStat(&out, "zero:", 0))
	serviceMust(t, PrintStat(&out, "count:", 17))
	serviceMust(t, PrintStat(&out, "nul:\x00ignored", math.MaxUint32))
	if got, want := out.String(), "count:17\nnul:"+strconv.FormatUint(math.MaxUint32, 10)+"\n"; got != want {
		t.Fatal([]byte(got), []byte(want))
	}
}

func TestPrintStatFloatGoFormattingAndPositiveGate(t *testing.T) {
	var out bytes.Buffer
	for _, value := range []float32{0, float32(math.Copysign(0, -1)), -1, float32(math.NaN()), float32(math.Inf(-1))} {
		serviceMust(t, PrintStatFloat(&out, "skip:", value))
	}
	for _, value := range []float32{1, 1.2345678, math.SmallestNonzeroFloat32, math.MaxFloat32, float32(math.Inf(1))} {
		serviceMust(t, PrintStatFloat(&out, "v:", value))
		want := "v:" + strconv.FormatFloat(float64(value), 'g', 6, 32) + "\n"
		if got := out.String(); got != want {
			t.Fatal(value, got, want)
		}
		out.Reset()
	}
}

type statFailWriter struct {
	calls int
	err   error
}

func (w *statFailWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, w.err
}

func TestPrintStatWriterErrorsOnlyAfterPositiveGate(t *testing.T) {
	boom := errors.New("stat output failed")
	out := statFailWriter{err: boom}
	serviceMust(t, PrintStat(&out, "u:", 0))
	serviceMust(t, PrintStatFloat(&out, "f:", -1))
	if out.calls != 0 {
		t.Fatal("nonpositive statistic reached writer")
	}
	if err := PrintStat(&out, "u:", 1); !errors.Is(err, boom) || out.calls != 1 {
		t.Fatal("unsigned write error", err, out.calls)
	}
	if err := PrintStatFloat(&out, "f:", 1); !errors.Is(err, boom) || out.calls != 2 {
		t.Fatal("float write error", err, out.calls)
	}
}
