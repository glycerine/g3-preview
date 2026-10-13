package params

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type limitedDescriptorWriter struct {
	out   bytes.Buffer
	limit int
	calls int
	short bool
}

func (w *limitedDescriptorWriter) Write(p []byte) (int, error) {
	w.calls++
	room := w.limit - w.out.Len()
	if room < 0 {
		room = 0
	}
	if room > len(p) {
		room = len(p)
	}
	n, _ := w.out.Write(p[:room])
	if n != len(p) {
		if w.short {
			return n, nil
		}
		return n, io.ErrClosedPipe
	}
	return n, nil
}

func TestParamDescrsDisplayToStickyWriterPrefix(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	pdinsert(t, d, "B-key", ParamUInt, pdstr("second"), pdstr("2"), nil)
	pdinsert(t, d, "A-key", ParamBool, pdstr("first"), pdstr("false"), nil)
	want, err := d.Display(2, false, true)
	paramsMust(t, err)
	for _, short := range []bool{false, true} {
		for limit := 0; limit < len(want); limit++ {
			w := &limitedDescriptorWriter{limit: limit, short: short}
			err := d.DisplayTo(w, 2, false, true)
			kind := error(io.ErrClosedPipe)
			if short {
				kind = io.ErrShortWrite
			}
			if !errors.Is(err, kind) || w.out.String() != want[:limit] || w.calls == 0 {
				t.Fatal("source sticky output prefix", short, limit, err, w.out.String(), want[:limit], w.calls)
			}
		}
	}
	var out bytes.Buffer
	paramsMust(t, d.DisplayTo(&out, 2, false, true))
	if out.String() != want {
		t.Fatal("writer output differs from source buffer", out.String(), want)
	}
}
