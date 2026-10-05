// Source: src/smt/params/theory_datatype_params.h:22-35.
// Source author: Leonardo de Moura (2008-11-04).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"io"
	"reflect"
	"testing"
)

func TestSMTDatatypeParamsSourceConstructorAndUnsignedDisplay(t *testing.T) {
	p := NewSMTDatatypeParams()
	if p.LazySplits != 1 || reflect.TypeOf(p).Size() != 4 {
		t.Fatal("source scalar default/layout", p)
	}
	for _, tc := range []struct {
		value uint32
		text  string
	}{
		{0, "m_dt_lazy_splits=0\n"},
		{1, "m_dt_lazy_splits=1\n"},
		{2, "m_dt_lazy_splits=2\n"},
		{^uint32(0), "m_dt_lazy_splits=4294967295\n"},
	} {
		f, out, b := numericFileFixture(t)
		before := b.Used()
		p.LazySplits = tc.value
		copyValue := p
		b.FailAfter(0) // Scalar construction and file insertion need no allocation.
		if q := NewSMTDatatypeParams(); q.LazySplits != 1 {
			t.Fatal("constructor depends on previous value", q)
		}
		paramsMust(t, p.Display(out))
		if p != copyValue || b.Used() != before {
			t.Fatal("const display changed scalar/cold ownership", p, b.Used(), before)
		}
		b.FailAfter(-1)
		_, e := f.Seek(0, io.SeekStart)
		paramsMust(t, e)
		got, e := io.ReadAll(f)
		paramsMust(t, e)
		if string(got) != tc.text {
			t.Fatalf("source output: got %q, want %q", got, tc.text)
		}
		paramsMust(t, out.Close())
		if _, e = f.Write([]byte("caller")); e != nil {
			t.Fatal("display closed borrowed file", e)
		}
	}
}

func TestSMTDatatypeParamsSourceStreamFlagsAndClosedEntry(t *testing.T) {
	p := NewSMTDatatypeParams()
	p.LazySplits = ^uint32(0)
	f, out, _ := numericFileFixture(t)
	paramsMust(t, out.SetIntegerBase(16))
	paramsMust(t, out.SetUppercase(true))
	paramsMust(t, out.SetShowBase(true))
	paramsMust(t, out.SetWidth(19))
	paramsMust(t, out.SetFill('_'))
	paramsMust(t, p.Display(out))
	_, e := f.Seek(0, io.SeekStart)
	paramsMust(t, e)
	got, e := io.ReadAll(f)
	paramsMust(t, e)
	if string(got) != "__m_dt_lazy_splits=0XFFFFFFFF\n" || out.State().Width != 0 {
		t.Fatal("source insertion order/borrowed flags", string(got), out.State())
	}
	paramsMust(t, out.Close())
	before := p
	if e = p.Display(out); e == nil || p != before {
		t.Fatal("closed entry changed const value", e, p)
	}
	if e = p.Display(nil); e != ErrNumericStreamOwner || p != before {
		t.Fatal("invalid stream entry", e, p)
	}
}
