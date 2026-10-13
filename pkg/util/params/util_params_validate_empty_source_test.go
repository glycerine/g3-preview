package params

import (
	"errors"
	"testing"
)

func TestParamsReferenceValidateEmptySkipsDescriptors(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	if err := c.Validate(&v, nil); err != nil {
		t.Fatalf("empty source reference read nil descriptors: %v", err)
	}
	d, err := NewParamDescrs(sy, ParamDescrsConfig{})
	paramsMust(t, err)
	paramsMust(t, d.Close())
	if err := c.Validate(&v, d); err != nil {
		t.Fatalf("empty source reference read closed descriptors: %v", err)
	}
	k := internSymbol(t, sy, "x")
	paramsMust(t, c.SetBool(&v, k, true))
	if err := c.Validate(&v, nil); !errors.Is(err, ErrParamsOwner) {
		t.Fatalf("populated reference skipped descriptor owner: %v", err)
	}
	if err := c.Validate(&v, d); err == nil {
		t.Fatal("populated reference skipped closed descriptor guard")
	}
}
