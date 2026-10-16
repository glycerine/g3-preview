package params

import (
	"errors"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func TestParamsValidationLegalListMissingDescriptionSourceGuard(t *testing.T) {
	if paramDescrsStreamSourceDebug {
		bindProcessAssertionRefusal(t)
	}
	c, symbols, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	d, err := NewParamDescrs(symbols, ParamDescrsConfig{})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, d.Close()) })
	known := internSymbol(t, symbols, "known")
	first := internSymbol(t, symbols, "first")
	descr := "first description"
	paramsMust(t, d.Insert(first, ParamUInt, &descr, nil, nil))
	paramsMust(t, d.Insert(known, ParamBool, nil, nil, nil))
	paramsMust(t, c.SetBool(&v, known, true))
	if err := c.Validate(&v, d); err != nil {
		t.Fatal("matching kind inspected unused description", err)
	}
	paramsMust(t, c.SetBool(&v, internSymbol(t, symbols, "missing"), true))
	err = c.Validate(&v, d)
	if paramDescrsStreamSourceDebug {
		if !errors.Is(err, udiag.ErrDebugAssertion) {
			t.Fatal("checked legal-list assertion", err)
		}
		wantPrefix := "unknown parameter 'missing'\nLegal parameters are:\n  first (unsigned int)\n  known"
		if got := string(c.workspace); got != wantPrefix {
			t.Fatal("checked assertion lost earlier source insertions", got)
		}
	} else {
		want := "unknown parameter 'missing'\nLegal parameters are:\n  first (unsigned int)\n  known (bool)\n"
		if err == nil || err.Error() != want {
			t.Fatal("release legal list omitted description", err)
		}
	}
}
