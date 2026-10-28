package ast

// Source: src/ast/ast_smt2_pp.cpp:165-291; src/util/rational.cpp:64-83.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

// Accounting for the explicitly borrowed global numeral service only. Actual
// payload ownership is proved independently by the strict physical collector.
func astFormatterRationalColdBytes(t *testing.T, q *unumeral.RationalService) uint64 {
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
	return n
}
