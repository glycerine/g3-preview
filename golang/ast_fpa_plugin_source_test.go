// Source: src/ast/fpa_decl_plugin.cpp:23-49,182-231;
// src/ast/fpa_decl_plugin.h:109-122; src/ast/ast.cpp:1406-1414,1635-1638.
// Source author: Leonardo de Moura (2012-01-15).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
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
	if p.ids.next != 0 || len(p.ids.free) != 0 || p.ids.budget != m.budget || p.values.manager != p.numeric || p.values.raw.self != &p.values.raw || p.values.raw.runtime != p.numeric.RuntimeOwner() || p.values.raw.block != (memory.Block{}) {
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
			if v.kind != ASTParameterInt || v.integer != want {
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
func astFloatingPointConstructorRawGraph(t *testing.T, q *mpnum.RationalService, p *astFloatingPointPluginOwner, foreign ...*mpnum.OwnedRational) {
	astFloatingPointRawGraphWithBlocks(t, q, p, nil, foreign...)
}
func astFloatingPointRawGraphWithBlocks(t *testing.T, q *mpnum.RationalService, p *astFloatingPointPluginOwner, blocksOwned []memory.Block, foreign ...*mpnum.OwnedRational) {
	astFloatingPointRawGraphSourceOwners(t, q, p, blocksOwned, nil, nil, foreign...)
}
func astFloatingPointFormatRawGraph(t *testing.T, q *mpnum.RationalService, p *astFloatingPointPluginOwner, texts ...*ArbitraryFloatFormatText) {
	astFloatingPointRawGraphSourceOwners(t, q, p, nil, texts, nil)
}

// Only explicitly passed foreign Integer owners and their actual live values.
type astFloatBorrowedInteger struct {
	owner *mpnum.IntegerManager
	value *mpnum.Int
}

func astFloatingPointRawGraphSourceOwners(t *testing.T, q *mpnum.RationalService, p *astFloatingPointPluginOwner, blocksOwned []memory.Block, texts []*ArbitraryFloatFormatText, integers []astFloatBorrowedInteger, foreign ...*mpnum.OwnedRational) {
	t.Helper()
	r := reflect.ValueOf(q.RuntimeOwner()).Elem()
	roots := map[[3]uint64]bool{}
	raw := func(v reflect.Value) {
		k := [3]uint64{v.FieldByName("origin").Uint(), v.FieldByName("slot").Uint(), v.FieldByName("generation").Uint()}
		if k[1] == 0 {
			return
		}
		if k[0] != r.FieldByName("origin").Uint() {
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
		if z.FieldByName("self").Pointer() != z.Addr().Pointer() {
			t.Fatal("copied numeric context")
		}
		refs := map[uint64]bool{}
		add := func(n uint64) {
			if n != 0 && n != z.FieldByName("failedRef").Uint() && n != z.FieldByName("failedNullRef").Uint() {
				refs[n] = true
			}
		}

		for _, foreign := range integers {
			if reflect.ValueOf(foreign.owner).Pointer() == z.Addr().Pointer() {
				add(reflect.ValueOf(foreign.value).Elem().FieldByName("cell").Uint())
			}
		}

		if p != nil && p.numeric != nil && reflect.ValueOf(p.numeric).Elem().FieldByName("integer").Pointer() == z.Addr().Pointer() {
			parent := reflect.ValueOf(p.numeric).Elem()
			for w := parent.FieldByName("powerWrappers"); !w.IsNil(); w = w.Elem().FieldByName("next") {
				v := w.Elem()
				if v.FieldByName("self").Pointer() != w.Pointer() || v.FieldByName("manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign Power wrapper")
				}
				raw(v.FieldByName("block"))
				raw(v.FieldByName("roots").FieldByName("cell").FieldByName("raw"))
				raw(v.FieldByName("transferred").FieldByName("block"))

			}

			for i := uint32(0); i < p.values.initialized; i++ {
				value, e := p.numeric.CaptureRuntimeValue(p.values.raw.block, 8+32*uint64(i))
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
					add(reflect.ValueOf(&f.sig).Elem().FieldByName("cell").Uint())
				}
				if f.valueConstructed && !f.valueConsumed {
					words, e := p.numeric.ValueWords(&f.value)
					astTestMust(t, e)
					add(uint64(words[4]) | uint64(words[5])<<32)
				}
			}
			for call := parent.FieldByName("valueCalls"); !call.IsNil(); call = call.Elem().FieldByName("next") {
				v := call.Elem()
				if v.FieldByName("self").Pointer() != call.Pointer() || v.FieldByName("manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign MPF copy call")
				}
				frame := v.FieldByName("childFrame")
				raw(frame.FieldByName("roots").FieldByName("cell").FieldByName("raw"))
				raw(frame.FieldByName("transferred").FieldByName("block"))
			}

			for _, text := range texts {
				if text == nil || text.self != text {
					t.Fatal("foreign text owner")
				}
				f := reflect.ValueOf(&text.numeric).Elem()
				if f.FieldByName("manager").IsNil() {
					continue
				}
				if f.FieldByName("manager").Pointer() != parent.Addr().Pointer() || f.FieldByName("self").Pointer() != f.Addr().Pointer() {
					t.Fatal("foreign format invocation")
				}
				count := f.FieldByName("constructed").Uint()
				if count >= 1 {
					add(f.FieldByName("num").FieldByName("cell").Uint())
				}
				if count >= 2 {
					add(f.FieldByName("denom").FieldByName("cell").Uint())
				}
				if count >= 3 {
					for _, field := range []string{"num", "den"} {
						add(f.FieldByName("q").FieldByName(field).FieldByName("cell").Uint())
					}
				}
				for _, scope := range []reflect.Value{f, f.FieldByName("decimal")} {
					raw(scope.FieldByName("numeric").FieldByName("cell").FieldByName("raw"))
					allocations := scope.FieldByName("numeric").FieldByName("allocator")
					for i := 0; i < allocations.Len(); i++ {
						raw(allocations.Index(i))
					}
					raw(scope.FieldByName("transferred").FieldByName("block"))
				}
				d := f.FieldByName("decimal")
				if !d.FieldByName("parent").IsNil() {
					for i := 1; i < 4; i++ {
						add(d.FieldByName("values").Index(i).FieldByName("cell").Uint())
					}
					if d.FieldByName("count").Uint() >= 4 {
						add(d.FieldByName("values").Index(0).FieldByName("cell").Uint())
					}
				}
			}
		}

		// Literal RAW declarations of this explicitly supplied Integer owner.
		// Only count-reached locals and exact detached leaf backing are named.
		for _, name := range []string{"divisionRaw", "gcdRaw", "extendedRaw"} {
			scope := z.FieldByName(name)
			for i := uint64(0); i < scope.FieldByName("count").Uint(); i++ {
				add(scope.FieldByName("values").Index(int(i)).FieldByName("cell").Uint())
			}
			raw(scope.FieldByName("numeric").FieldByName("cell").FieldByName("raw"))
			allocations := scope.FieldByName("numeric").FieldByName("allocator")
			for i := 0; i < allocations.Len(); i++ {
				raw(allocations.Index(i))
			}
		}
		for _, f := range []string{"tmp", "arg"} {
			v := z.FieldByName(f)
			for i := 0; i < v.Len(); i++ {
				add(v.Index(i).Uint())
			}
		}
		for _, f := range []string{"intMin", "two64"} {
			add(z.FieldByName(f).FieldByName("cell").Uint())
		}
		if cache.IsValid() {
			service := reflect.ValueOf(q).Elem()
			for call := service.FieldByName("lexicalInvocations"); !call.IsNil(); call = call.Elem().FieldByName("next") {
				v := call.Elem()
				if v.FieldByName("self").Pointer() != call.Pointer() || v.FieldByName("service").Pointer() != service.Addr().Pointer() {
					t.Fatal("foreign lexical numerical caller")
				}
				scope := v.FieldByName("scope")
				for i := uint64(0); i < scope.FieldByName("count").Uint(); i++ {
					n := scope.FieldByName("roots").Index(int(i))
					add(n.FieldByName("num").FieldByName("cell").Uint())
					add(n.FieldByName("den").FieldByName("cell").Uint())
				}
				raw(v.FieldByName("numeric").FieldByName("cell").FieldByName("raw"))
				allocations := v.FieldByName("numeric").FieldByName("allocator")
				for i := 0; i < allocations.Len(); i++ {
					raw(allocations.Index(i))
				}
				raw(v.FieldByName("transferred").FieldByName("block"))
				characters := v.FieldByName("characters").FieldByName("buffer")
				raw(characters.FieldByName("raw"))
				raw(characters.FieldByName("pending"))
				locals := v.FieldByName("characterFormatting").FieldByName("locals")
				for i := 0; i < locals.Len(); i++ {
					raw(locals.Index(i).FieldByName("raw"))
					raw(locals.Index(i).FieldByName("pending"))
				}
			}
			for _, value := range foreign {
				v := reflect.ValueOf(value).Elem()
				if v.FieldByName("ownerID").Uint() != reflect.ValueOf(q).Elem().FieldByName("ownerID").Uint() {
					t.Fatal("foreign rational owner")
				}
				v = v.FieldByName("raw")
				add(v.FieldByName("num").FieldByName("cell").Uint())
				add(v.FieldByName("den").FieldByName("cell").Uint())
			}
			for i := 0; i < cache.Len(); i++ {
				v := cache.Index(i)
				add(v.FieldByName("num").FieldByName("cell").Uint())
				add(v.FieldByName("den").FieldByName("cell").Uint())
			}
		}
		s := z.FieldByName("cells")
		if !s.IsNil() {
			s = s.Elem()
			pages := s.FieldByName("pages")
			pageSize := s.FieldByName("config").FieldByName("PageSize").Uint()
			named := uint64(0)
			for i := uint64(0); i < s.FieldByName("slots").Uint(); i++ {
				page := pages.Index(int(i / pageSize))
				state := page.FieldByName("meta").Index(int(i % pageSize))
				if !state.FieldByName("live").Bool() {
					continue
				}
				cell := page.FieldByName("records").Index(int(i % pageSize))
				ref := state.FieldByName("generation").Uint()<<32 | (i + 1)
				if cell.FieldByName("unpublished").Bool() || cell.FieldByName("cleanupPending").Bool() {
					refs[ref] = true
				}
				if !refs[ref] && !(z.FieldByName("failedRef").Type().Size() == 4 && refs[i+1]) {
					t.Fatal("anonymous constructor numeric leaf", ref)
				}
				named++
				if !cell.FieldByName("released").Bool() {
					raw(cell.FieldByName("backing").FieldByName("raw"))
				}
			}
			if named != s.FieldByName("live").Uint() {
				t.Fatal("descriptor bijection", named, s.FieldByName("live").Uint())
			}
		}
		natural := z.FieldByName("natural")
		if !natural.IsNil() {
			n := natural.Elem()
			division := n.FieldByName("division")
			for i := 0; i < division.Len(); i++ {
				raw(division.Index(i).FieldByName("raw"))
				raw(division.Index(i).FieldByName("pending"))
			}
			verification := n.FieldByName("verification")
			raw(verification.FieldByName("raw"))
			raw(verification.FieldByName("pending"))
			characters := n.FieldByName("charFormatting").FieldByName("buffer")
			raw(characters.FieldByName("raw"))
			raw(characters.FieldByName("pending"))
			locals := n.FieldByName("formatting").FieldByName("locals")
			for i := 0; i < locals.Len(); i++ {
				raw(locals.Index(i).FieldByName("raw"))
				raw(locals.Index(i).FieldByName("pending"))
			}
		}
		raw(z.FieldByName("pendingCell").FieldByName("raw"))
		a := z.FieldByName("allocator")
		if !a.IsNil() {
			a = a.Elem()
			// Source-owned chunk chains survive pooled NUM destruction. Follow only
			// actual class head IDs and initialized next words, not locator enumeration.
			heads := a.FieldByName("chunks")
			locations := a.FieldByName("locations")
			width := a.FieldByName("config").FieldByName("Profile").FieldByName("PointerBits").Uint() / 8
			for class := 0; class < heads.Len(); class++ {
				for id := heads.Index(class).Uint(); id != 0; {
					if id > uint64(locations.Len()) {
						t.Fatal("invalid source chunk head")
					}
					chunk := locations.Index(int(id - 1))
					if !chunk.FieldByName("live").Bool() || chunk.FieldByName("kind").Uint() != 1 {
						t.Fatal("nonowned source chunk")
					}
					b := chunk.FieldByName("raw")
					raw(b)
					physical := r.FieldByName("blocks").Index(int(b.FieldByName("slot").Uint() - 1))
					if !physical.FieldByName("live").Bool() || physical.FieldByName("generation").Uint() != b.FieldByName("generation").Uint() {
						t.Fatal("source chunk lifetime")
					}
					next := uint64(0)
					for j := uint64(0); j < width; j++ {
						k := j + r.FieldByName("config").FieldByName("SizeTBits").Uint()/8
						if physical.FieldByName("initialized").Index(int(k/64)).Uint()&(uint64(1)<<(k%64)) == 0 {
							t.Fatal("source chunk link uninitialized")
						}
						next |= physical.FieldByName("data").Index(int(k)).Uint() << (8 * j)
					}
					id = next
				}
			}
			raw(a.FieldByName("pending"))
			v := a.FieldByName("pendingScratch")
			for i := 0; i < v.Len(); i++ {
				raw(v.Index(i))
			}
		}
	}
	s := reflect.ValueOf(q).Elem()
	addManager(s.FieldByName("manager").Elem().FieldByName("z"), s.FieldByName("cache"))
	if p != nil {
		if p.self != p || p.owner == nil {
			t.Fatal("foreign plugin owner")
		}
		if !p.values.raw.pendingReplaces {
			raw(reflect.ValueOf(p.values.raw.block))
		}
		raw(reflect.ValueOf(p.values.raw.pending))
		for a := p.values.history; a != nil; a = a.next {
			if a.self != a || a.owner != &p.values {
				t.Fatal("foreign value acquisition")
			}
			raw(reflect.ValueOf(a.active))
		}
		if p.numeric != nil {
			v := reflect.ValueOf(p.numeric).Elem()
			addManager(v.FieldByName("integer"), reflect.Value{})
		} else {
			var z *mpnum.IntegerManager
			if errors.As(p.failedNumeric, &z) {
				addManager(reflect.ValueOf(z), reflect.Value{})
			}
		}
	}
	for _, foreign := range integers {
		addManager(reflect.ValueOf(foreign.owner), reflect.Value{})
	}
	blocks := r.FieldByName("blocks")
	live := 0
	for i := 0; i < blocks.Len(); i++ {
		v := blocks.Index(i)
		if !v.FieldByName("live").Bool() {
			continue
		}
		k := [3]uint64{r.FieldByName("origin").Uint(), uint64(i + 1), v.FieldByName("generation").Uint()}
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
		r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
		astTestMust(t, e)
		q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		sy, e := NewSymbolManager(SymbolConfig{Budget: b})
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
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: b})
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
