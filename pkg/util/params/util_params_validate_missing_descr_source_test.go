package params

import (
	"errors"
	"testing"
)

func TestParamsValidationLegalListMissingDescriptionSourceGuard(t *testing.T) {
	c, symbols, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	d, err := NewParamDescrs(symbols, ParamDescrsConfig{})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, d.Close()) })
	known := internSymbol(t, symbols, "known")
	paramsMust(t, d.Insert(known, ParamBool, nil, nil, nil))
	paramsMust(t, c.SetBool(&v, known, true))
	if err := c.Validate(&v, d); err != nil {
		t.Fatal("matching kind inspected unused description", err)
	}
	paramsMust(t, c.SetBool(&v, internSymbol(t, symbols, "missing"), true))
	err = c.Validate(&v, d)
	if paramDescrsStreamSourceDebug {
		if !errors.Is(err, ErrParamDescrsPrecondition) {
			t.Fatal("checked legal-list assertion", err)
		}
	} else {
		want := "unknown parameter 'missing'\nLegal parameters are:\n  known (bool)\n"
		if err == nil || err.Error() != want {
			t.Fatal("release legal list omitted description", err)
		}
	}
}
