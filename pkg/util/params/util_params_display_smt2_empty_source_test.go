package params

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsReferenceDisplaySMT2EmptySkipsDescriptors(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	module := "m"
	if got, err := c.DisplaySMT2(&v, nil, nil); err != nil || got != "" {
		t.Fatalf("empty reference touched nil arguments: %q, %v", got, err)
	}
	if n, err := c.FormatSMT2Into(&v, nil, nil, nil); err != nil || n != 0 {
		t.Fatalf("empty reference formatted nil arguments: %d, %v", n, err)
	}
	d, err := NewParamDescrs(sy, ParamDescrsConfig{})
	paramsMust(t, err)
	paramsMust(t, d.Close())
	if got, err := c.DisplaySMT2(&v, &module, d); err != nil || got != "" {
		t.Fatalf("empty reference touched closed descriptors: %q, %v", got, err)
	}
	k := internSymbol(t, sy, "x")
	paramsMust(t, c.SetBool(&v, k, true))
	if _, err := c.DisplaySMT2(&v, &module, nil); !errors.Is(err, ErrParamsPrecondition) {
		t.Fatalf("populated reference skipped descriptor precondition: %v", err)
	}
	if _, err := c.DisplaySMT2(&v, &module, d); err == nil {
		t.Fatal("populated reference skipped closed descriptor guard")
	}
}

func TestParamsReferenceDisplayKeyEmptySkipsSymbolRead(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	invalid := usymbol.Symbol{Tag: 4}
	if got, err := c.DisplayKey(&v, invalid); err != nil || got != "default" {
		t.Fatalf("empty keyed display read symbol: %q, %v", got, err)
	}
	var buf [7]byte
	if n, err := c.FormatKeyInto(&v, invalid, buf[:]); err != nil || n != 7 || string(buf[:]) != "default" {
		t.Fatalf("empty keyed format read symbol: %d, %q, %v", n, buf, err)
	}
	k := internSymbol(t, sy, "x")
	paramsMust(t, c.SetBool(&v, k, true))
	if _, err := c.DisplayKey(&v, invalid); !errors.Is(err, usymbol.ErrInvalid) {
		t.Fatalf("populated keyed display skipped symbol guard: %v", err)
	}
}
