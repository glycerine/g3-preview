package params

import (
	"bytes"
	"errors"
	"io"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestParamsReferenceDirectWriterCompletedPrefixes(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	k := internSymbol(t, sy, "alpha")
	paramsMust(t, c.SetBool(&v, k, true))
	text := "payload"
	paramsMust(t, c.SetString(&v, internSymbol(t, sy, "beta"), &text))
	want, err := c.Display(&v)
	paramsMust(t, err)
	for _, short := range []bool{false, true} {
		for limit := 0; limit < len(want); limit++ {
			w := &limitedDescriptorWriter{limit: limit, short: short}
			err := c.DisplayTo(w, &v)
			kind := error(io.ErrClosedPipe)
			if short {
				kind = io.ErrShortWrite
			}
			if !errors.Is(err, kind) || w.out.String() != want[:limit] {
				t.Fatalf("direct prefix short=%v limit=%d: got %q, %v; want %q", short, limit, w.out.String(), err, want[:limit])
			}
		}
	}
	var out bytes.Buffer
	paramsMust(t, c.DisplayTo(&out, &v))
	if out.String() != want {
		t.Fatalf("direct display = %q, want %q", out.String(), want)
	}
	if got, err := c.Display(&v); err != nil || got != want {
		t.Fatalf("buffered display after refusal = %q, %v", got, err)
	}
}

func TestParamsReferenceDirectWriterEmptyKeyAndSMT2Guards(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	invalid := usymbol.Symbol{Tag: 4}
	var out bytes.Buffer
	paramsMust(t, c.DisplayKeyTo(&out, &v, invalid))
	if out.String() != "default" {
		t.Fatalf("empty key output = %q", out.String())
	}
	out.Reset()
	paramsMust(t, c.DisplaySMT2To(&out, &v, nil, nil))
	if out.Len() != 0 {
		t.Fatalf("empty SMT2 output = %q", out.String())
	}
	k := internSymbol(t, sy, "alpha")
	paramsMust(t, c.SetBool(&v, k, true))
	if err := c.DisplaySMT2To(&out, &v, nil, nil); !errors.Is(err, ErrParamsPrecondition) {
		t.Fatalf("populated SMT2 with nil descriptors: %v", err)
	}
	if err := c.DisplayKeyTo(&out, &v, invalid); !errors.Is(err, usymbol.ErrInvalid) {
		t.Fatalf("populated key with invalid symbol: %v", err)
	}
	d, err := NewParamDescrs(sy, ParamDescrsConfig{})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, d.Close()) })
	description := "boolean"
	paramsMust(t, d.Insert(k, ParamBool, &description, nil, nil))
	module := "mod"
	want, err := c.DisplaySMT2(&v, &module, d)
	paramsMust(t, err)
	out.Reset()
	paramsMust(t, c.DisplaySMT2To(&out, &v, &module, d))
	if out.String() != want {
		t.Fatalf("direct SMT2 output = %q, want %q", out.String(), want)
	}
	textKey := internSymbol(t, sy, "text")
	paramsMust(t, c.SetString(&v, textKey, nil))
	want, err = c.Display(&v)
	if !errors.Is(err, utext.ErrNullCString) {
		t.Fatalf("buffered null string: %v", err)
	}
	out.Reset()
	if err := c.DisplayTo(&out, &v); !errors.Is(err, utext.ErrNullCString) || out.String() != want {
		t.Fatalf("direct null string prefix = %q, %v; want %q", out.String(), err, want)
	}
	refused := &limitedDescriptorWriter{limit: len(want) - 1}
	if err := c.DisplayTo(refused, &v); !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, utext.ErrNullCString) || refused.out.String() != want[:len(want)-1] {
		t.Fatalf("refused prefix and later null string = %q, %v", refused.out.String(), err)
	}
}
