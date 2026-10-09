// Source: src/ast/ast_smt2_pp.cpp:165-291; src/util/rational.cpp:64-83.
package ast

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Accounting for the explicitly borrowed global numeral service only. Actual
// payload ownership is proved independently by the strict physical collector.
func astFormatterRationalColdBytes(t *testing.T, q *util.RationalService) uint64 {
	t.Helper()
	s := reflect.ValueOf(q).Elem()
	if testFieldBySourceName(s, "self").Pointer() != reflect.ValueOf(q).Pointer() {
		t.Fatal("foreign numerical service")
	}
	n := testFieldBySourceName(s, "headerBytes").Uint()
	m := testFieldBySourceName(s, "manager").Elem()
	n += testFieldBySourceName(m, "headerBytes").Uint()
	z := testFieldBySourceName(m, "z").Elem()
	n += testFieldBySourceName(z, "headerBytes").Uint()
	slab := func(v reflect.Value) uint64 {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
		out := testFieldBySourceName(v, "directoryBytes").Uint()
		if !testFieldBySourceName(v, "closed").Bool() {
			out += uint64(v.Type().Size())
		}
		pages := testFieldBySourceName(v, "pages")
		for i := 0; i < pages.Len(); i++ {
			for _, key := range []string{"records", "meta"} {
				a := testFieldBySourceName(pages.Index(i), key)
				out += uint64(a.Len()) * uint64(a.Type().Elem().Size())
			}
		}
		return out
	}
	a := testFieldBySourceName(z, "allocator").Elem()
	loc := testFieldBySourceName(a, "locations")
	n += testFieldBySourceName(a, "header").Uint() + uint64(loc.Cap())*uint64(loc.Type().Elem().Size())
	n += slab(testFieldBySourceName(z, "cells"))
	nat := testFieldBySourceName(z, "natural").Elem()
	n += testFieldBySourceName(nat, "headerBytes").Uint()
	if text := testFieldBySourceName(nat, "text"); !text.IsNil() {
		text = text.Elem()
		if !testFieldBySourceName(text, "closed").Bool() {
			n += uint64(text.Type().Size())
		}
		n += testFieldBySourceName(text, "directoryBytes").Uint() + slab(testFieldBySourceName(text, "descriptors"))
		pages := testFieldBySourceName(text, "pages")
		for i := 0; i < pages.Len(); i++ {
			p := pages.Index(i)
			n += uint64(testFieldBySourceName(p, "data").Len()) + uint64(testFieldBySourceName(p, "next").Len())*4
		}
	}
	return n
}
