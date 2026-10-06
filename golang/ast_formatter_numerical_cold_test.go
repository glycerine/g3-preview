// Source: src/ast/ast_smt2_pp.cpp:165-291; src/util/rational.cpp:64-83.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

// Accounting for the explicitly borrowed global numeral service only. Actual
// payload ownership is proved independently by the strict physical collector.
func astFormatterRationalColdBytes(t *testing.T, q *mpnum.RationalService) uint64 {
	t.Helper()
	s := reflect.ValueOf(q).Elem()
	if s.FieldByName("self").Pointer() != reflect.ValueOf(q).Pointer() {
		t.Fatal("foreign numerical service")
	}
	n := s.FieldByName("headerBytes").Uint() + s.FieldByName("cacheBytes").Uint()
	m := s.FieldByName("manager").Elem()
	n += m.FieldByName("headerBytes").Uint()
	z := m.FieldByName("z").Elem()
	n += z.FieldByName("headerBytes").Uint()
	slab := func(v reflect.Value) uint64 {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
		out := v.FieldByName("directoryBytes").Uint()
		if !v.FieldByName("closed").Bool() {
			out += uint64(v.Type().Size())
		}
		pages := v.FieldByName("pages")
		for i := 0; i < pages.Len(); i++ {
			for _, key := range []string{"records", "meta"} {
				a := pages.Index(i).FieldByName(key)
				out += uint64(a.Len()) * uint64(a.Type().Elem().Size())
			}
		}
		return out
	}
	a := z.FieldByName("allocator").Elem()
	loc := a.FieldByName("locations")
	n += a.FieldByName("header").Uint() + uint64(loc.Cap())*uint64(loc.Type().Elem().Size())
	n += slab(z.FieldByName("cells"))
	w := z.FieldByName("words").Elem()
	if !w.FieldByName("closed").Bool() {
		n += uint64(w.Type().Size())
	}
	n += w.FieldByName("directoryBytes").Uint() + slab(w.FieldByName("descriptors"))
	pages := w.FieldByName("pages")
	for i := 0; i < pages.Len(); i++ {
		p := pages.Index(i)
		n += uint64(p.FieldByName("data").Len())*4 + uint64(p.FieldByName("next").Len())*4
	}
	nat := z.FieldByName("natural").Elem()
	n += nat.FieldByName("headerBytes").Uint()
	if text := nat.FieldByName("text"); !text.IsNil() {
		text = text.Elem()
		if !text.FieldByName("closed").Bool() {
			n += uint64(text.Type().Size())
		}
		n += text.FieldByName("directoryBytes").Uint() + slab(text.FieldByName("descriptors"))
		pages := text.FieldByName("pages")
		for i := 0; i < pages.Len(); i++ {
			p := pages.Index(i)
			n += uint64(p.FieldByName("data").Len()) + uint64(p.FieldByName("next").Len())*4
		}
	}
	return n
}
