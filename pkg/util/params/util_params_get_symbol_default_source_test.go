package params

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsGetSymbolMissReturnsDefaultWithoutReadingIt(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	fallback, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() {
		paramsMust(t, c.Del(&v))
		paramsMust(t, c.Del(&fallback))
	})
	invalid := usymbol.Symbol{Tag: 4}
	got, err := c.GetSymbol(&v, invalid, invalid)
	if err != nil || got != invalid {
		t.Fatalf("null-backed get_sym read key or default: %v, %v", got, err)
	}
	key := internSymbol(t, sy, "present")
	missing := internSymbol(t, sy, "missing")
	paramsMust(t, c.SetBool(&v, key, true))
	got, err = c.GetSymbol(&v, missing, invalid)
	if err != nil || got != invalid {
		t.Fatalf("populated missing get_sym read default: %v, %v", got, err)
	}
	name := "missing"
	got, err = c.GetSymbolFallbackName(&v, &name, &fallback, invalid)
	if err != nil || got != invalid {
		t.Fatalf("empty fallback get_sym read default: %v, %v", got, err)
	}
}
