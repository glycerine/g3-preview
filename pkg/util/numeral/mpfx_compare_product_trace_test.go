package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFixedPointComparisonAndProductSourceTrace(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	b, _ := m.New()
	zero, _ := m.New()
	out, _ := m.New()
	t.Cleanup(func() {
		for _, v := range []*FixedPoint{a, b, zero, out} {
			if e := v.Close(); e != nil {
				t.Error(e)
			}
		}
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	if err := m.SetInt64(a, -3); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(b, 2); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Less(a, b); err != nil || output.Len() != 0 {
		t.Fatal("disabled comparison trace", err, output.String())
	}
	tags.Enable("mpfx_trace")
	check := func(name, want string) {
		t.Helper()
		if !tags.ShouldTrace("mpfx_trace") {
			want = ""
		}
		if got := output.String(); got != want {
			t.Fatalf("%s trace: got %q, want %q", name, got, want)
		}
		output.Reset()
	}
	less, err := m.Less(a, b)
	if err != nil || !less {
		t.Fatal("less", less, err)
	}
	check("less", "[mpfx] (-3 < 2) == (1 == 1)\n")
	greater, err := m.Greater(a, b)
	if err != nil || greater {
		t.Fatal("greater", greater, err)
	}
	check("greater reverses operands", "[mpfx] (2 < -3) == (0 == 1)\n")
	le, err := m.LessEqual(a, b)
	if err != nil || !le {
		t.Fatal("less equal", le, err)
	}
	check("less equal reverses operands", "[mpfx] (2 < -3) == (0 == 1)\n")
	ge, err := m.GreaterEqual(a, b)
	if err != nil || ge {
		t.Fatal("greater equal", ge, err)
	}
	check("greater equal", "[mpfx] (-3 < 2) == (1 == 1)\n")
	if err := m.SetRounding(true); err != nil {
		t.Fatal(err)
	}
	if err := m.Mul(a, b, out); err != nil {
		t.Fatal(err)
	}
	check("multiply up", "[mpfx] (-3) * (2) <= -6\n")
	if err := m.SetRounding(false); err != nil {
		t.Fatal(err)
	}
	if err := m.Div(a, b, out); err != nil {
		t.Fatal(err)
	}
	check("divide down", "[mpfx] (-3) / (2) >= -3/2\n")
	if err := m.Div(zero, b, out); err != nil {
		t.Fatal(err)
	}
	check("zero dividend", "[mpfx] (0) / (2) >= 0\n")
	assertFixedPointException(t, m.Div(a, zero, out), ErrFixedPointDivZero, ErrFixedPointOverflow, "multi-precision fixed point (mpfx) division by zero")
	check("zero divisor before trace", "")
}
