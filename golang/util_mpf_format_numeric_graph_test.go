// Source: src/util/mpf.cpp:61-67,1519-1569; src/util/mpq.h:25-49; src/util/small_object_allocator.cpp:28-53.
// Test-only graph projection from actual named numeric owners, without AST construction.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func mpfFormatNumericRawGraph(t *testing.T, q *mpnum.RationalService, m *mpnum.ArbitraryFloatManager, blocksOwned []memory.Block, texts []*ArbitraryFloatFormatText, inputs ...*mpnum.ArbitraryFloat) {
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

		if m != nil && reflect.ValueOf(m).Elem().FieldByName("integer").Pointer() == z.Addr().Pointer() {
			parent := reflect.ValueOf(m).Elem()
			for _, input := range inputs {
				if input == nil {
					t.Fatal("nil explicitly supplied MPF input")
				}
				v := reflect.ValueOf(input).Elem()
				add(v.FieldByName("significand").FieldByName("cell").Uint())
			}
			for w := parent.FieldByName("powerWrappers"); !w.IsNil(); w = w.Elem().FieldByName("next") {
				v := w.Elem()
				if v.FieldByName("self").Pointer() != w.Pointer() || v.FieldByName("manager").Pointer() != parent.Addr().Pointer() {
					t.Fatal("foreign Power wrapper")
				}
				raw(v.FieldByName("block"))
				raw(v.FieldByName("roots").FieldByName("cell").FieldByName("raw"))
				raw(v.FieldByName("transferred").FieldByName("block"))

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
	if m == nil {
		t.Fatal("nil explicit numeric owner")
	}
	parent := reflect.ValueOf(m).Elem()
	if parent.FieldByName("self").Pointer() != reflect.ValueOf(m).Pointer() {
		t.Fatal("copied MPF owner")
	}
	addManager(parent.FieldByName("integer"), reflect.Value{})

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

func mpfFormatNumericOwner(t *testing.T) (*arena.Budget, *memory.Runtime, *mpnum.RationalService, *mpnum.ArbitraryFloatManager) {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	astTestMust(t, e)
	m, e := q.NewArbitraryFloatManager()
	astTestMust(t, e)
	return b, rt, q, m
}
