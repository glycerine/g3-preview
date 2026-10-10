// Source: src/util/mpf.cpp:61-67,1519-1569; src/util/mpq.h:25-49; src/util/small_object_allocator.cpp:28-53.
// Test-only graph projection from actual named numeric owners, without AST construction.
package util

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func mpfFormatNumericRawGraph(t *testing.T, q *RationalService, m *ArbitraryFloatManager, blocksOwned []umemory.Block, texts []*ArbitraryFloatFormatText, inputs ...*ArbitraryFloat) {
	t.Helper()
	r := reflect.ValueOf(q.RuntimeOwner()).Elem()
	roots := map[[3]uint64]bool{}
	raw := func(v reflect.Value) {
		k := [3]uint64{testFieldBySourceName(v, "origin").Uint(), testFieldBySourceName(v, "slot").Uint(), testFieldBySourceName(v, "generation").Uint()}
		if k[1] == 0 {
			return
		}
		if k[0] != testFieldBySourceName(r, "origin").Uint() {
			t.Fatal("foreign raw provider", k)
		}
		roots[k] = true
	}
	for _, block := range blocksOwned {
		raw(reflect.ValueOf(block))
	}
	addManager := func(z reflect.Value, cache reflect.Value) {
		if !z.IsValid() || z.IsNil() {
			return
		}
		z = z.Elem()
		if testFieldBySourceName(z, "self").Pointer() != z.Addr().Pointer() {
			t.Fatal("copied numeric context")
		}
		refs := map[uint64]bool{}
		add := func(n uint64) {
			if n != 0 && n != testFieldBySourceName(z, "failedRef").Uint() && n != testFieldBySourceName(z, "failedNullRef").Uint() {
				refs[n] = true
			}
		}

		if m != nil && testFieldBySourceName(reflect.ValueOf(m).Elem(), "integer").Pointer() == z.Addr().Pointer() {
			parent := reflect.ValueOf(m).Elem()
			for _, input := range inputs {
				if input == nil {
					t.Fatal("nil explicitly supplied MPF input")
				}
				v := reflect.ValueOf(input).Elem()
				add(testFieldBySourceName(testFieldBySourceName(v, "significand"), "cell").Uint())
			}
			for w := testFieldBySourceName(parent, "powerWrappers"); !w.IsNil(); w = testFieldBySourceName(w.Elem(), "next") {
				v := w.Elem()
				if testFieldBySourceName(v, "self").Pointer() != w.Pointer() || testFieldBySourceName(v, "manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign Power wrapper")
				}
				raw(testFieldBySourceName(v, "block"))
				raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(v, "roots"), "cell"), "raw"))
				raw(testFieldBySourceName(testFieldBySourceName(v, "transferred"), "block"))

			}
			for _, text := range texts {
				if text == nil || text.Self != text {
					t.Fatal("foreign text owner")
				}
				f := reflect.ValueOf(&text.Numeric).Elem()
				if testFieldBySourceName(f, "manager").IsNil() {
					continue
				}
				if testFieldBySourceName(f, "manager").Pointer() != parent.Addr().Pointer() || testFieldBySourceName(f, "self").Pointer() != f.Addr().Pointer() {
					t.Fatal("foreign format invocation")
				}
				count := testFieldBySourceName(f, "constructed").Uint()
				if count >= 1 {
					add(testFieldBySourceName(testFieldBySourceName(f, "num"), "cell").Uint())
				}
				if count >= 2 {
					add(testFieldBySourceName(testFieldBySourceName(f, "denom"), "cell").Uint())
				}
				if count >= 3 {
					for _, field := range []string{"num", "den"} {
						add(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(f, "q"), field), "cell").Uint())
					}
				}
				for _, scope := range []reflect.Value{f, testFieldBySourceName(f, "decimal")} {
					raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(scope, "numeric"), "cell"), "raw"))
					allocations := testFieldBySourceName(testFieldBySourceName(scope, "numeric"), "allocator")
					for i := 0; i < allocations.Len(); i++ {
						raw(allocations.Index(i))
					}
					raw(testFieldBySourceName(testFieldBySourceName(scope, "transferred"), "block"))
				}
				d := testFieldBySourceName(f, "decimal")
				if !testFieldBySourceName(d, "parent").IsNil() {
					for i := 1; i < 4; i++ {
						add(testFieldBySourceName(testFieldBySourceName(d, "values").Index(i), "cell").Uint())
					}
					if testFieldBySourceName(d, "count").Uint() >= 4 {
						add(testFieldBySourceName(testFieldBySourceName(d, "values").Index(0), "cell").Uint())
					}
				}
			}
		}

		// Literal RAW declarations of this explicitly supplied Integer owner.
		// Only count-reached locals and exact detached leaf backing are named.
		for _, name := range []string{"divisionRaw", "gcdRaw", "extendedRaw"} {
			scope := testFieldBySourceName(z, name)
			for i := uint64(0); i < testFieldBySourceName(scope, "count").Uint(); i++ {
				add(testFieldBySourceName(testFieldBySourceName(scope, "values").Index(int(i)), "cell").Uint())
			}
			raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(scope, "numeric"), "cell"), "raw"))
			allocations := testFieldBySourceName(testFieldBySourceName(scope, "numeric"), "allocator")
			for i := 0; i < allocations.Len(); i++ {
				raw(allocations.Index(i))
			}
		}
		for _, f := range []string{"tmp", "arg"} {
			v := testFieldBySourceName(z, f)
			for i := 0; i < v.Len(); i++ {
				add(v.Index(i).Uint())
			}
		}
		for _, f := range []string{"intMin", "two64"} {
			add(testFieldBySourceName(testFieldBySourceName(z, f), "cell").Uint())
		}
		if cache.IsValid() {
			for i := 0; i < cache.Len(); i++ {
				v := cache.Index(i)
				add(testFieldBySourceName(testFieldBySourceName(v, "num"), "cell").Uint())
				add(testFieldBySourceName(testFieldBySourceName(v, "den"), "cell").Uint())
			}
		}
		s := testFieldBySourceName(z, "cells")
		if !s.IsNil() {
			s = s.Elem()
			pages := testFieldBySourceName(s, "pages")
			pageSize := testFieldBySourceName(testFieldBySourceName(s, "config"), "PageSize").Uint()
			named := uint64(0)
			for i := uint64(0); i < testFieldBySourceName(s, "slots").Uint(); i++ {
				page := pages.Index(int(i / pageSize))
				state := testFieldBySourceName(page, "meta").Index(int(i % pageSize))
				if !testFieldBySourceName(state, "live").Bool() {
					continue
				}
				cell := testFieldBySourceName(page, "records").Index(int(i % pageSize))
				ref := testFieldBySourceName(state, "generation").Uint()<<32 | (i + 1)
				if testFieldBySourceName(cell, "unpublished").Bool() || testFieldBySourceName(cell, "cleanupPending").Bool() {
					refs[ref] = true
				}
				if !refs[ref] && !(testFieldBySourceName(z, "failedRef").Type().Size() == 4 && refs[i+1]) {
					t.Fatal("anonymous constructor numeric leaf", ref)
				}
				named++
				if !testFieldBySourceName(cell, "released").Bool() {
					raw(testFieldBySourceName(testFieldBySourceName(cell, "backing"), "raw"))
				}
			}
			if named != testFieldBySourceName(s, "live").Uint() {
				t.Fatal("descriptor bijection", named, testFieldBySourceName(s, "live").Uint())
			}
		}
		raw(testFieldBySourceName(testFieldBySourceName(z, "pendingCell"), "raw"))
		a := testFieldBySourceName(z, "allocator")
		if !a.IsNil() {
			a = a.Elem()
			// Source-owned chunk chains survive pooled NUM destruction. Follow only
			// actual class head IDs and initialized next words, not locator enumeration.
			heads := testFieldBySourceName(a, "chunks")
			locations := testFieldBySourceName(a, "locations")
			width := testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(a, "config"), "Profile"), "PointerBits").Uint() / 8
			for class := 0; class < heads.Len(); class++ {
				for id := heads.Index(class).Uint(); id != 0; {
					if id > uint64(locations.Len()) {
						t.Fatal("invalid source chunk head")
					}
					chunk := locations.Index(int(id - 1))
					if !testFieldBySourceName(chunk, "live").Bool() || testFieldBySourceName(chunk, "kind").Uint() != 1 {
						t.Fatal("nonowned source chunk")
					}
					b := testFieldBySourceName(chunk, "raw")
					raw(b)
					physical := testFieldBySourceName(r, "blocks").Index(int(testFieldBySourceName(b, "slot").Uint() - 1))
					if !testFieldBySourceName(physical, "live").Bool() || testFieldBySourceName(physical, "generation").Uint() != testFieldBySourceName(b, "generation").Uint() {
						t.Fatal("source chunk lifetime")
					}
					next := uint64(0)
					for j := uint64(0); j < width; j++ {
						k := j + testFieldBySourceName(testFieldBySourceName(r, "config"), "SizeTBits").Uint()/8
						if testFieldBySourceName(physical, "initialized").Index(int(k/64)).Uint()&(uint64(1)<<(k%64)) == 0 {
							t.Fatal("source chunk link uninitialized")
						}
						next |= testFieldBySourceName(physical, "data").Index(int(k)).Uint() << (8 * j)
					}
					id = next
				}
			}
			raw(testFieldBySourceName(a, "pending"))
		}
	}
	s := reflect.ValueOf(q).Elem()
	addManager(testFieldBySourceName(testFieldBySourceName(s, "manager").Elem(), "z"), testFieldBySourceName(s, "cache"))
	if m == nil {
		t.Fatal("nil explicit numeric owner")
	}
	parent := reflect.ValueOf(m).Elem()
	if testFieldBySourceName(parent, "self").Pointer() != reflect.ValueOf(m).Pointer() {
		t.Fatal("copied MPF owner")
	}
	addManager(testFieldBySourceName(parent, "integer"), reflect.Value{})

	blocks := testFieldBySourceName(r, "blocks")
	live := 0
	for i := 0; i < blocks.Len(); i++ {
		v := blocks.Index(i)
		if !testFieldBySourceName(v, "live").Bool() {
			continue
		}
		k := [3]uint64{testFieldBySourceName(r, "origin").Uint(), uint64(i + 1), testFieldBySourceName(v, "generation").Uint()}
		if !roots[k] {
			t.Fatal("anonymous real numeric raw block", k)
		}
		live++
	}
	if live != len(roots) {
		t.Fatal("raw bijection", live, len(roots))
	}
}

func mpfFormatNumericOwner(t *testing.T) (*arena.Budget, *umemory.Runtime, *RationalService, *ArbitraryFloatManager) {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	q, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	m, e := q.NewArbitraryFloatManager()
	serviceMust(t, e)
	return b, rt, q, m
}
