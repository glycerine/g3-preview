package numeral

import (
	utext "github.com/glycerine/g3/pkg/util/text"
	"testing"
)

// mpbq::display formats the numerator into a default local string, then
// inserts the unsigned exponent with the receiving stream's actual flags.
func TestBinaryDisplaySourceNumeratorStringAndExponentFlags(t *testing.T) {
	m, budget := binaryManager(t)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, out.SetIntegerBase(16))
	serviceMust(t, out.SetShowBase(true))
	a := NewBinaryIntPower(-31, 31)
	var receipt NumeralInsertionReceipt
	serviceMust(t, m.DisplaySourceTracked(out, &a, &receipt))
	text, e := out.String()
	serviceMust(t, e)
	if text != "-31/2^0x1f" {
		t.Fatal("numerator string/exponent insertion distinction", text)
	}
	serviceMust(t, m.CleanupSourceString(&receipt))
	if receipt.Active() {
		t.Fatal("completed source display retained child")
	}
}
