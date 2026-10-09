// Source: src/ast/fpa_decl_plugin.cpp:23-49,182-231;
// src/ast/fpa_decl_plugin.h:109-122; src/ast/ast.cpp:1406-1414,1635-1638.
package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTFloatingPointActualConstructorRegistrationAndSorts(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	family, e := m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	if p == nil || !p.published || !p.completed || p.numeric == nil || p.table.capacity != 10 || p.real.slot == 0 || p.integer.slot == 0 || p.bitvector != m.bitVectorPlugin {
		t.Fatal("actual completed members/publication/set_manager", p)
	}
	if p.ids.Range() != 0 || p.ids.FreeLen() != 0 || p.ids.Budget() != m.BudgetField || p.values.manager != p.numeric || p.values.raw.Self != &p.values.raw || p.values.raw.Runtime != p.numeric.RuntimeOwner() || p.values.raw.Block != (util.Block{}) {
		t.Fatal("actual initially-empty ID generator and scoped MPF vector members")
	}
	for _, tc := range []struct {
		k    int32
		e, s int32
	}{{2, 5, 11}, {3, 8, 24}, {4, 11, 53}, {5, 15, 113}} {
		h, e := m.MkFamilySort(family, tc.k, nil)
		astTestMust(t, e)
		r, e := m.internal(h)
		astTestMust(t, e)
		info, e := m.sortInfo(m.sortAt(r.slot).info)
		astTestMust(t, e)
		if info.family != family || info.kind != 0 || info.count != 2 || !info.size.IsVeryBig() {
			t.Fatal("literal floating predefined sort", tc, *info)
		}
		for i, want := range []int32{tc.e, tc.s} {
			value, e := m.infoParameter(info, uint32(i))
			astTestMust(t, e)
			v, e := m.parameter(value)
			astTestMust(t, e)
			if v.Kind != ASTParameterInt || v.integer != want {
				t.Fatal("actual integer parameters", tc, i, *v)
			}
		}
	}
	h, e := m.MkFamilySort(family, 1, nil)
	astTestMust(t, e)
	r, e := m.internal(h)
	astTestMust(t, e)
	info, e := m.sortInfo(m.sortAt(r.slot).info)
	astTestMust(t, e)
	if info.kind != 1 || !info.size.IsInfinite() || info.count != 0 {
		t.Fatal("literal rounding mode sort", *info)
	}
}

// Read-only exact provider census for explicitly passed constructor contexts.
// Default debug allocator objects are direct; no pooled-list or directory scan
// supplies ownership. Numeric roots come from reached member fields/cache, and
// explicitly stamped unfinished placement entries of these exact contexts.
func astFloatingPointConstructorRawGraph(t *testing.T, q *util.RationalService, p *astFloatingPointPluginOwner, foreign ...*util.OwnedRational) {
	astFloatingPointRawGraphWithBlocks(t, q, p, nil, foreign...)
}
func astFloatingPointRawGraphWithBlocks(t *testing.T, q *util.RationalService, p *astFloatingPointPluginOwner, blocksOwned []util.Block, foreign ...*util.OwnedRational) {
	astFloatingPointRawGraphSourceOwners(t, q, p, blocksOwned, nil, nil, foreign...)
}
func astFloatingPointFormatRawGraph(t *testing.T, q *util.RationalService, p *astFloatingPointPluginOwner, texts ...*util.ArbitraryFloatFormatText) {
	astFloatingPointRawGraphSourceOwners(t, q, p, nil, texts, nil)
}

// Only explicitly passed foreign Integer owners and their actual live values.
type astFloatBorrowedInteger struct {
	owner *util.IntegerManager
	value *util.Int
}

func astFloatingPointRawGraphSourceOwners(t *testing.T, q *util.RationalService, p *astFloatingPointPluginOwner, blocksOwned []util.Block, texts []*util.ArbitraryFloatFormatText, integers []astFloatBorrowedInteger, foreign ...*util.OwnedRational) {
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

		for _, foreign := range integers {
			if reflect.ValueOf(foreign.owner).Pointer() == z.Addr().Pointer() {
				add(testFieldBySourceName(reflect.ValueOf(foreign.value).Elem(), "cell").Uint())
			}
		}

		if p != nil && p.numeric != nil && testFieldBySourceName(reflect.ValueOf(p.numeric).Elem(), "integer").Pointer() == z.Addr().Pointer() {
			parent := reflect.ValueOf(p.numeric).Elem()
			for w := testFieldBySourceName(parent, "powerWrappers"); !w.IsNil(); w = testFieldBySourceName(w.Elem(), "next") {
				v := w.Elem()
				if testFieldBySourceName(v, "self").Pointer() != w.Pointer() || testFieldBySourceName(v, "manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign Power wrapper")
				}
				raw(testFieldBySourceName(v, "block"))
				raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(v, "roots"), "cell"), "raw"))
				raw(testFieldBySourceName(testFieldBySourceName(v, "transferred"), "block"))

			}

			for i := uint32(0); i < p.values.initialized; i++ {
				value, e := p.numeric.CaptureRuntimeValue(p.values.raw.Block, 8+32*uint64(i))
				astTestMust(t, e)
				words, e := p.numeric.ValueWords(&value)
				astTestMust(t, e)
				add(uint64(words[4]) | uint64(words[5])<<32)
			}
			for f := p.literalFrames; f != nil; f = f.next {
				if f.self != f || f.owner != p {
					t.Fatal("foreign literal frame")
				}
				if f.sigConstructed && !f.sigConsumed {
					add(testFieldBySourceName(reflect.ValueOf(&f.sig).Elem(), "cell").Uint())
				}
				if f.valueConstructed && !f.valueConsumed {
					words, e := p.numeric.ValueWords(&f.value)
					astTestMust(t, e)
					add(uint64(words[4]) | uint64(words[5])<<32)
				}
			}
			for call := testFieldBySourceName(parent, "valueCalls"); !call.IsNil(); call = testFieldBySourceName(call.Elem(), "next") {
				v := call.Elem()
				if testFieldBySourceName(v, "self").Pointer() != call.Pointer() || testFieldBySourceName(v, "manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign MPF copy call")
				}
				frame := testFieldBySourceName(v, "childFrame")
				raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(frame, "roots"), "cell"), "raw"))
				raw(testFieldBySourceName(testFieldBySourceName(frame, "transferred"), "block"))
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
			service := reflect.ValueOf(q).Elem()
			for call := testFieldBySourceName(service, "lexicalInvocations"); !call.IsNil(); call = testFieldBySourceName(call.Elem(), "next") {
				v := call.Elem()
				if testFieldBySourceName(v, "self").Pointer() != call.Pointer() || testFieldBySourceName(v, "service").Pointer() != service.Addr().Pointer() {
					t.Fatal("foreign lexical numerical caller")
				}
				scope := testFieldBySourceName(v, "scope")
				for i := uint64(0); i < testFieldBySourceName(scope, "count").Uint(); i++ {
					n := testFieldBySourceName(scope, "roots").Index(int(i))
					add(testFieldBySourceName(testFieldBySourceName(n, "num"), "cell").Uint())
					add(testFieldBySourceName(testFieldBySourceName(n, "den"), "cell").Uint())
				}
				raw(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(v, "numeric"), "cell"), "raw"))
				allocations := testFieldBySourceName(testFieldBySourceName(v, "numeric"), "allocator")
				for i := 0; i < allocations.Len(); i++ {
					raw(allocations.Index(i))
				}
				raw(testFieldBySourceName(testFieldBySourceName(v, "transferred"), "block"))
				locals := testFieldBySourceName(testFieldBySourceName(v, "characterFormatting"), "locals")
				for i := 0; i < locals.Len(); i++ {
					raw(testFieldBySourceName(locals.Index(i), "raw"))
					raw(testFieldBySourceName(locals.Index(i), "pending"))
				}
			}
			for _, value := range foreign {
				v := reflect.ValueOf(value).Elem()
				if testFieldBySourceName(v, "ownerID").Uint() != testFieldBySourceName(reflect.ValueOf(q).Elem(), "ownerID").Uint() {
					t.Fatal("foreign rational owner")
				}
				v = testFieldBySourceName(v, "raw")
				add(testFieldBySourceName(testFieldBySourceName(v, "num"), "cell").Uint())
				add(testFieldBySourceName(testFieldBySourceName(v, "den"), "cell").Uint())
			}
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
		natural := testFieldBySourceName(z, "natural")
		if !natural.IsNil() {
			n := natural.Elem()
			division := testFieldBySourceName(n, "division")
			for i := 0; i < division.Len(); i++ {
				raw(testFieldBySourceName(division.Index(i), "raw"))
				raw(testFieldBySourceName(division.Index(i), "pending"))
			}
			verification := testFieldBySourceName(n, "verification")
			raw(testFieldBySourceName(verification, "raw"))
			raw(testFieldBySourceName(verification, "pending"))
			locals := testFieldBySourceName(testFieldBySourceName(n, "formatting"), "locals")
			for i := 0; i < locals.Len(); i++ {
				raw(testFieldBySourceName(locals.Index(i), "raw"))
				raw(testFieldBySourceName(locals.Index(i), "pending"))
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
	if p != nil {
		if p.self != p || p.owner == nil {
			t.Fatal("foreign plugin owner")
		}
		if !p.values.raw.PendingReplaces {
			raw(reflect.ValueOf(p.values.raw.Block))
		}
		raw(reflect.ValueOf(p.values.raw.Pending))
		for a := p.values.history; a != nil; a = a.next {
			if a.self != a || a.owner != &p.values {
				t.Fatal("foreign value acquisition")
			}
			raw(reflect.ValueOf(a.active))
		}
		if p.numeric != nil {
			v := reflect.ValueOf(p.numeric).Elem()
			addManager(testFieldBySourceName(v, "integer"), reflect.Value{})
		} else {
			var z *util.IntegerManager
			if errors.As(p.failedNumeric, &z) {
				addManager(reflect.ValueOf(z), reflect.Value{})
			}
		}
	}
	for _, foreign := range integers {
		addManager(reflect.ValueOf(foreign.owner), reflect.Value{})
	}
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

func TestASTFloatingPointEveryActualConstructorAndPublicationPrefix(t *testing.T) {
	success := false
	failed := 0
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 28)
		r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
		astTestMust(t, e)
		q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		astFloatingPointConstructorRawGraph(t, q, nil)
		symbols := func() uint64 { return astTestSymbolGraph(sy) }
		// The original service's validated, unchanged cold namespace is a fixed
		// accounting term, never authority to adopt any newly returned acquisition.
		oldCold := b.Used() - m.Stats().AllocatedBytes - symbols() - r.Stats().TotalBytes
		cacheBefore, e := q.Stats()
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterFloatingPointPlugin()
		p := m.floatingPointPlugin
		if p == nil {
			p = m.floatingPointPluginFailed
		}
		astFloatingPointConstructorRawGraph(t, q, p)
		if b.Used() != oldCold+m.Stats().AllocatedBytes+symbols()+r.Stats().TotalBytes {
			t.Fatal("before lift exact reached cold graph", ordinal, b.Used(), oldCold, m.Stats(), r.Stats())
		}
		cacheAfter, e := q.Stats()
		astTestMust(t, e)
		if cacheAfter != cacheBefore {
			t.Fatal("borrowed service modified", ordinal, cacheBefore, cacheAfter)
		}
		if call == nil {
			success = true
		} else {
			failed++
			if p != nil && p.bodyCause == nil {
				t.Fatal("lost actual constructor body cause", ordinal, call)
			}
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, q.Close())
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("actual enclosing owner ends", ordinal, b.Used())
		}
		if success {
			t.Log("actual FPA first-success reservation", ordinal)
			break
		}
	}
	if !success || failed == 0 {
		t.Fatal("actual constructor horizon", failed, success)
	}
}

func TestASTFloatingPointSourceFinalizeAndOwnContextEndPreserveBorrowedNumerals(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	foreign, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetUint64(&foreign, 1<<63))
	original := foreign
	p := m.floatingPointPlugin
	astFloatingPointConstructorRawGraph(t, q, p, &foreign)
	astTestMust(t, m.SourceFinalize())
	if !p.sourceEnded || p.header == 0 || m.floatingPointPlugin != p || p.numeric.OwnedStorageBytes() == 0 || foreign != original {
		t.Fatal("source member end retains exact enclosing history/foreign token", p)
	}
	astFloatingPointConstructorRawGraph(t, q, p, &foreign)
	astTestMust(t, m.Close())
	if p.header != 0 || p.numeric.OwnedStorageBytes() != 0 || foreign != original {
		t.Fatal("exact owning context end", p, foreign, original)
	}
	value, e := q.Uint64(&foreign)
	astTestMust(t, e)
	if value != 1<<63 {
		t.Fatal("borrowed numeral source remains live", value)
	}
	astTestMust(t, q.Del(&foreign))
	astTestMust(t, sy.Close())
	astTestMust(t, q.Close())
	astTestMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("all actual owned contexts ended", b.Used())
	}
}
