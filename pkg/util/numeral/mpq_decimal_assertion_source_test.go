package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestRationalDecimalSourceAssertionAfterDigitDivision(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, _ := bindMPNDebugActions(t, "T")
	m, _, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(1), Den: NewInt(3)}
	var out utext.OwnedStringStream
	serviceMust(t, out.Construct(b))
	serviceMust(t, d.Close())
	err := m.rational.DisplayDecimalInto(&out, &a, 1, false)
	if !errors.Is(err, arena.ErrClosed) {
		t.Fatal("source assertion switch must read closed debug owner", err)
	}
	got, err := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
	serviceMust(t, err)
	if got != "0." || !m.rational.decimalStream.active {
		t.Fatal("assertion must precede digit publication", got, m.rational.decimalStream)
	}
	serviceMust(t, m.rational.CleanupDecimalStream())
	out.Buffer.OwnAbort()
}
