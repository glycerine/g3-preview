// Source: src/math/polynomial/polynomial.cpp:66-82; src/math/polynomial/polynomial.h:62-69.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialVariableDegreesActualNumericStreamFlags(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	d, e := NewPolynomialVariableDegrees(r)
	if e != nil {
		t.Fatal(e)
	}
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := out.Close(); e != nil {
			t.Error(e)
		}
		if e := d.CloseChecked(); e != nil {
			t.Error(e)
		}
		if e := r.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("actual stream/map ownership remains", b.Used())
		}
	})
	for _, e := range []error{d.SetDegree(10, 171), out.SetIntegerBase(16), out.SetShowBase(true), out.SetUppercase(true), out.SetShowPos(true), out.SetFill('_'), out.SetWidth(4)} {
		if e != nil {
			t.Fatal(e)
		}
	}
	before := r.Stats()
	if e := d.DisplayInto(out); e != nil {
		t.Fatal(e)
	}
	got, e := out.String()
	want := "___[x0^0,x0X1^0,x0X2^0,x0X3^0,x0X4^0,x0X5^0,x0X6^0,x0X7^0,x0X8^0,x0X9^0,x0XA^0XAB]"
	if e != nil || got != want {
		t.Fatal(got, want, e)
	}
	state := out.State()
	if state.Width != 0 || state.IntegerBase != 16 || !state.ShowBase || !state.Uppercase || !state.ShowPos || state.Fill != '_' {
		t.Fatal("source stream flags/one-shot width", state)
	}
	if before != r.Stats() {
		t.Fatal("display changed borrowed map storage", before, r.Stats())
	}
}
