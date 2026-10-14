package params

import "testing"

// src/util/params.cpp:350-375 accepts an unsigned value for a numeral
// descriptor, but not a numeral value for an unsigned descriptor. A successful
// module lookup rewrites the stored key before a later validation error.
func TestParamsValidationNumeralKindAsymmetryAndEarlierKeyRewrite(t *testing.T) {
	c, symbols, numerals := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	d, err := NewParamDescrs(symbols, ParamDescrsConfig{})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, d.Close()) })

	n, u := internSymbol(t, symbols, "n"), internSymbol(t, symbols, "u")
	module, description := "m", "kind check"
	paramsMust(t, d.Insert(n, ParamNumeral, &description, nil, &module))
	paramsMust(t, d.Insert(u, ParamUInt, &description, nil, nil))
	paramsMust(t, c.SetUInt(&v, internSymbol(t, symbols, "m.n"), 7))
	value, err := numerals.NewValue()
	paramsMust(t, err)
	paramsMust(t, numerals.SetFractionSigned(&value, 3, 2))
	paramsMust(t, c.SetRat(&v, u, &value))
	paramsMust(t, numerals.Del(&value))

	err = c.Validate(&v, d)
	if err == nil || err.Error() != "Parameter u was given argument of type rational, expected unsigned int" {
		t.Fatal("asymmetric source kind check", err)
	}
	if got, err := c.GetUInt(&v, n, 0); err != nil || got != 7 {
		t.Fatal("earlier qualified key rewrite was rolled back", got, err)
	}
	paramsMust(t, c.ResetKey(&v, u))
	if err := c.Validate(&v, d); err != nil {
		t.Fatal("unsigned value should satisfy numeral descriptor", err)
	}
}
