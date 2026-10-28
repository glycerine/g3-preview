package params

// Source: src/util/mpz.cpp; src/util/mpq.cpp; src/util/params.cpp (ownership audit adaptation).

import (
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

type paramsPhysicalValueOwner struct {
	Context *ParamsContext
	Value   *ParamsRef
}

type paramsPhysicalRationalOwner struct {
	Service *unumeral.RationalService
	Value   *unumeral.OwnedRational
}

type numericPhysicalExtraOwners struct {
	CallerBlocks         []umemory.Block // actual directly returned caller allocations only
	DebugContexts        []*udiag.DebugContext
	ParamsRationalValues []paramsPhysicalRationalOwner
	ParamsValues         []paramsPhysicalValueOwner
	RationalServices     []*unumeral.RationalService
	Integers             []*unumeral.Int
	Rationals            []*unumeral.Rational
	RationalManagers     []*unumeral.RationalManager
	BinaryValues         []*unumeral.BinaryRational
	BinaryManagers       []*unumeral.BinaryRationalManager
}

func numericPhysicalExtraNumerals(extras []numericPhysicalExtraOwners, addInt func(reflect.Value)) {
	for _, x := range extras {

		addRat := func(v reflect.Value) {
			addInt(testFieldBySourceName(v, "num"))
			addInt(testFieldBySourceName(v, "den"))
		}
		addManager := func(v reflect.Value) {
			for _, f := range []string{"nTmp", "addTmp1", "addTmp2"} {
				addInt(testFieldBySourceName(v, f))
			}
			for _, f := range []string{"addMulTmp", "ltTmp1", "ltTmp2"} {
				addRat(testFieldBySourceName(v, f))
			}
			scope := testFieldBySourceName(v, "lexical")
			for i := 0; i < int(testFieldBySourceName(scope, "count").Uint()); i++ {
				addRat(testFieldBySourceName(scope, "roots").Index(i))
			}
			raw := testFieldBySourceName(v, "rawArithmetic")
			if testFieldBySourceName(raw, "divActive").Bool() {
				addInt(testFieldBySourceName(raw, "div"))
			}
			if testFieldBySourceName(raw, "powerActive").Bool() {
				addRat(testFieldBySourceName(raw, "power"))
			}
			if testFieldBySourceName(raw, "addActive").Bool() {
				addInt(testFieldBySourceName(raw, "addOne"))
				addInt(testFieldBySourceName(raw, "addTwo"))
			}
			if testFieldBySourceName(raw, "compareActive").Bool() {
				addRat(testFieldBySourceName(raw, "compareOne"))
				addRat(testFieldBySourceName(raw, "compareTwo"))
			}
			if testFieldBySourceName(raw, "normActive").Bool() {
				addInt(testFieldBySourceName(raw, "norm"))
			}
		}
		// Only the explicitly supplied owner supplies roots. Its initialized
		// placement records and acquired lexical value are concrete acquisition
		// fields, never discoveries from the numeric allocator directory.

		for _, owner := range x.ParamsRationalValues {
			if owner.Service == nil || owner.Value == nil {
				panic(unumeral.ErrRationalServiceOwner)
			}
			service := reflect.ValueOf(owner.Service).Elem()
			value := reflect.ValueOf(owner.Value).Elem()
			if testFieldBySourceName(service, "self").Pointer() != reflect.ValueOf(owner.Service).Pointer() || testFieldBySourceName(value, "ownerID").Uint() == 0 || testFieldBySourceName(value, "ownerID").Uint() != testFieldBySourceName(service, "ownerID").Uint() {
				panic(unumeral.ErrRationalServiceOwner)
			}
			addRat(testFieldBySourceName(value, "raw"))
		}
		paramsContexts := make(map[*ParamsContext]bool)
		for _, owner := range x.ParamsValues {
			if owner.Context == nil || owner.Value == nil {
				panic(ErrParamsOwner)
			}
			c, v := owner.Context, owner.Value
			if !paramsContexts[c] {
				paramsContexts[c] = true
				for i, frame := range c.SetFrames {
					if !frame.Active {
						continue
					}
					if !frame.Acquired || frame.Invocation == 0 || i >= len(c.Blocks) || frame.Generation != c.Blocks[i].Generation || frame.Next.Kind != ParamNumeral {
						panic(ErrParamsOwner)
					}
					raw := reflect.ValueOf(&frame.Next.Rat).Elem()
					if testFieldBySourceName(raw, "ownerID").Uint() != testFieldBySourceName(reflect.ValueOf(c.NumeralsField).Elem(), "ownerID").Uint() {
						panic(unumeral.ErrRationalServiceOwner)
					}
					addRat(testFieldBySourceName(raw, "raw"))
				}
				for _, frame := range c.SourceFrames {
					if !frame.Active {
						continue
					}
					if frame.Invocation == 0 || frame.HeldSlot == 0 || int(frame.HeldSlot) > len(c.Blocks) {
						panic(ErrParamsOwner)
					}
					held := &c.Blocks[frame.HeldSlot-1]
					if !held.Live || held.Generation != frame.HeldGeneration {
						panic(ErrParamsOwner)
					}
					for i := uint32(0); i < held.Size; i++ {
						entry := c.Entry(held, i)
						if entry.Kind == ParamNumeral {
							addRat(testFieldBySourceName(reflect.ValueOf(&entry.Rat).Elem(), "raw"))
						}
					}
				}
			}
			if c.Self != c || c.Closed || v.Owner != c.Owner {
				panic(ErrParamsOwner)
			}
			if v.Slot == 0 {
				continue
			}
			if int(v.Slot) > len(c.Blocks) {
				panic(ErrParamsOwner)
			}
			b := &c.Blocks[v.Slot-1]
			if !b.Live || b.Generation != v.Generation {
				panic(ErrParamsOwner)
			}
			// refs==0 is allowed only as a reached stopped destructor owner. The
			// explicitly supplied owning value, not the count, names its actual block.
			for i := uint32(0); i < b.Size; i++ {
				e := c.Entry(b, i)
				if e.Kind == ParamNumeral {
					addRat(testFieldBySourceName(reflect.ValueOf(&e.Rat).Elem(), "raw"))
				}
			}
		}
		for _, s := range x.RationalServices {
			v := reflect.ValueOf(s).Elem()
			addManager(testFieldBySourceName(v, "manager").Elem())
			for _, f := range []string{"zero", "one", "minusOne"} {
				addRat(testFieldBySourceName(v, f))
			}
			cache := testFieldBySourceName(v, "cache")
			for i := 0; i < cache.Len(); i++ {
				addRat(cache.Index(i))
			}
			scope := testFieldBySourceName(v, "lexical")
			for i := 0; i < int(testFieldBySourceName(scope, "count").Uint()); i++ {
				addRat(testFieldBySourceName(scope, "roots").Index(i))
			}
		}

		for _, n := range x.Integers {
			addInt(reflect.ValueOf(n).Elem())
		}
		for _, r := range x.Rationals {
			v := reflect.ValueOf(r).Elem()
			addInt(testFieldBySourceName(v, "num"))
			addInt(testFieldBySourceName(v, "den"))
		}
		for _, m := range x.RationalManagers {
			addManager(reflect.ValueOf(m).Elem())
		}
		for _, b := range x.BinaryValues {
			addInt(testFieldBySourceName(reflect.ValueOf(b).Elem(), "num"))
		}
		for _, m := range x.BinaryManagers {
			v := reflect.ValueOf(m).Elem()
			for _, f := range []string{"tmp", "tmp2", "selectIntTmp1", "selectIntTmp2", "selectSmallTmp", "divTmp1", "divTmp2", "divTmp3"} {
				addInt(testFieldBySourceName(v, f))
			}
			for _, f := range []string{"addMulTmp", "selectSmallTmp1", "selectSmallTmp2"} {
				addInt(testFieldBySourceName(testFieldBySourceName(v, f), "num"))
			}
		}

	}
}

type numericPhysicalGraph struct{ Named, Unpublished, Cleanup, Pending, Objects uint32 }

// Read-only independent physical ownership audit. Reflection reads scalar
// metadata; it neither borrows mutable payloads nor repairs failed numerals.
func numericAuditPhysical(t *testing.T, manager *unumeral.IntegerManager, extras ...numericPhysicalExtraOwners) numericPhysicalGraph {
	t.Helper()
	var graph numericPhysicalGraph
	z := reflect.ValueOf(manager).Elem()
	cells := testFieldBySourceName(z, "cells").Elem()
	pages := testFieldBySourceName(cells, "pages")
	pageSize := testFieldBySourceName(testFieldBySourceName(cells, "config"), "PageSize").Uint()
	slots := testFieldBySourceName(cells, "slots").Uint()
	failed := testFieldBySourceName(z, "failedRef").Uint()
	failedNull := testFieldBySourceName(z, "failedNullRef").Uint()
	named := map[uint64]bool{}
	add := func(ref uint64) {
		if ref == 0 || ref == failed || ref == failedNull {
			return
		}
		slot, gen := uint64(uint32(ref)), ref>>32
		if slot == 0 || slot > slots {
			t.Fatal("owning scalar ref outside numeric directory", ref)
		}
		state := testFieldBySourceName(pages.Index(int((slot-1)/pageSize)), "meta").Index(int((slot - 1) % pageSize))
		if !testFieldBySourceName(state, "live").Bool() || (testFieldBySourceName(z, "failedRef").Type().Size() == 8 && testFieldBySourceName(state, "generation").Uint() != gen) {
			t.Fatal("owning scalar ref stale", ref)
		}
		// Release source pointers carry only slots; checked tokens also carry
		// generation. Canonical census identities use the actual live ledger.
		named[testFieldBySourceName(state, "generation").Uint()<<32|slot] = true
	}
	addInt := func(v reflect.Value) { add(testFieldBySourceName(v, "cell").Uint()) }
	for _, field := range []string{"tmp", "arg"} {
		v := testFieldBySourceName(z, field)
		for i := 0; i < v.Len(); i++ {
			add(v.Index(i).Uint())
		}
	}
	// Actual reached RAW declarations are owned by the shared Integer manager.
	// Read their named count/fields; descriptor liveness never supplies roots.
	for _, field := range []string{"divisionRaw", "extendedRaw", "gcdRaw"} {
		scope := testFieldBySourceName(z, field)
		values := testFieldBySourceName(scope, "values")
		for i := uint64(0); i < testFieldBySourceName(scope, "count").Uint(); i++ {
			addInt(values.Index(int(i)))
		}
	}
	addMul := testFieldBySourceName(z, "addMulInvocation")
	for _, child := range []reflect.Value{testFieldBySourceName(testFieldBySourceName(z, "gcdRaw"), "addMul"), testFieldBySourceName(addMul, "receipt")} {
		if testFieldBySourceName(child, "manager").Pointer() == z.Addr().Pointer() && testFieldBySourceName(child, "frame").Pointer() == addMul.Addr().Pointer() && testFieldBySourceName(child, "invocation").Uint() != 0 && testFieldBySourceName(child, "invocation").Uint() == testFieldBySourceName(addMul, "invocation").Uint() && testFieldBySourceName(addMul, "active").Bool() {
			addInt(testFieldBySourceName(addMul, "tmp"))
		}
	}
	addInt(testFieldBySourceName(z, "intMin"))
	addInt(testFieldBySourceName(z, "two64"))
	rootRAW := testFieldBySourceName(testFieldBySourceName(z, "rootRaw"), "values")
	for i := 0; i < rootRAW.Len(); i++ {
		addInt(rootRAW.Index(i))
	}
	addInt(testFieldBySourceName(testFieldBySourceName(z, "powerRaw"), "value"))
	numericPhysicalExtraNumerals(extras, addInt)

	runtime := testFieldBySourceName(z, "memory").Elem()
	blocks := testFieldBySourceName(runtime, "blocks")
	allocator := testFieldBySourceName(z, "allocator").Elem()
	locations := testFieldBySourceName(allocator, "locations")
	objects := map[[4]uint64]bool{}
	rawRoots := map[[3]uint64]bool{}
	rawKey := func(v reflect.Value) [3]uint64 {
		return [3]uint64{testFieldBySourceName(v, "origin").Uint(), testFieldBySourceName(v, "slot").Uint(), testFieldBySourceName(v, "generation").Uint()}
	}
	addBacking := func(backing reflect.Value, complete bool) {
		object := testFieldBySourceName(backing, "object")
		key := [4]uint64{testFieldBySourceName(object, "owner").Uint(), testFieldBySourceName(object, "address").Uint(), testFieldBySourceName(object, "slot").Uint(), testFieldBySourceName(object, "generation").Uint()}
		if key == [4]uint64{} {
			return
		}
		if key[0] != testFieldBySourceName(allocator, "origin").Uint() || key[2] == 0 || key[2] > uint64(locations.Len()) {
			t.Fatal("physical object owner/slot", key)
		}
		loc := locations.Index(int(key[2] - 1))
		if !testFieldBySourceName(loc, "live").Bool() || testFieldBySourceName(loc, "free").Bool() || testFieldBySourceName(loc, "generation").Uint() != key[3] || testFieldBySourceName(loc, "address").Uint() != key[1] {
			t.Fatal("physical owned object lifetime", key, loc)
		}
		if objects[key] {
			t.Fatal("two numeric owners of one physical object", key)
		}
		objects[key] = true
		raw := rawKey(testFieldBySourceName(loc, "raw"))
		if raw[0] != testFieldBySourceName(runtime, "origin").Uint() || raw[1] == 0 || raw[1] > uint64(blocks.Len()) {
			t.Fatal("owned cell raw block", raw)
		}
		rawBlock := blocks.Index(int(raw[1] - 1))
		if !testFieldBySourceName(rawBlock, "live").Bool() || testFieldBySourceName(rawBlock, "generation").Uint() != raw[2] {
			t.Fatal("owned cell raw lifetime", raw)
		}
		extent := testFieldBySourceName(loc, "extent").Uint()
		if testFieldBySourceName(loc, "kind").Uint() == 3 {
			width := testFieldBySourceName(testFieldBySourceName(runtime, "config"), "SizeTBits").Uint() / 8
			data := testFieldBySourceName(rawBlock, "data")
			initialized := testFieldBySourceName(rawBlock, "initialized")
			extent = 0
			for i := uint64(0); i < width; i++ {
				if initialized.Index(int(i/64)).Uint()&(uint64(1)<<(i%64)) == 0 {
					t.Fatal("direct runtime header not initialized")
				}
				extent |= data.Index(int(i)).Uint() << (8 * i)
			}
			extent -= width
		}
		if complete && (rawKey(testFieldBySourceName(loc, "raw")) != rawKey(testFieldBySourceName(backing, "raw")) || testFieldBySourceName(loc, "offset").Uint() != testFieldBySourceName(backing, "offset").Uint() || extent != testFieldBySourceName(backing, "extent").Uint() || testFieldBySourceName(loc, "requested").Uint() != testFieldBySourceName(backing, "requested").Uint()) {
			t.Fatal("physical cell bounds/location", key)
		}
	}
	live := uint32(0)
	for slot := uint64(1); slot <= slots; slot++ {
		page := pages.Index(int((slot - 1) / pageSize))
		index := int((slot - 1) % pageSize)
		state := testFieldBySourceName(page, "meta").Index(index)
		if !testFieldBySourceName(state, "live").Bool() {
			continue
		}
		live++
		ref := testFieldBySourceName(state, "generation").Uint()<<32 | slot
		record := testFieldBySourceName(page, "records").Index(index)
		if named[ref] {
			graph.Named++
		} else if testFieldBySourceName(record, "unpublished").Bool() {
			graph.Unpublished++
		} else if testFieldBySourceName(record, "cleanupPending").Bool() {
			graph.Cleanup++
		} else {
			t.Fatal("anonymous published numeric descriptor", ref)
		}
		if !testFieldBySourceName(record, "released").Bool() {
			addBacking(testFieldBySourceName(record, "backing"), true)
		}
	}
	if live != graph.Named+graph.Unpublished+graph.Cleanup || uint32(len(named)) != graph.Named || uint64(live) != testFieldBySourceName(cells, "live").Uint() {
		t.Fatal("exact descriptor root census", live, graph, len(named))
	}
	pending := testFieldBySourceName(z, "pendingCell")
	if testFieldBySourceName(testFieldBySourceName(pending, "object"), "slot").Uint() != 0 {
		graph.Pending++
		addBacking(pending, false)
	}
	graph.Objects = uint32(len(objects))
	for i := 0; i < locations.Len(); i++ {
		loc := locations.Index(i)
		if !testFieldBySourceName(loc, "live").Bool() {
			continue
		}
		if !testFieldBySourceName(loc, "free").Bool() && testFieldBySourceName(loc, "kind").Uint() != 1 {
			key := [4]uint64{testFieldBySourceName(allocator, "origin").Uint(), testFieldBySourceName(loc, "address").Uint(), uint64(i + 1), testFieldBySourceName(loc, "generation").Uint()}
			if !objects[key] {
				t.Fatal("anonymous live allocator object", key)
			}
		}
		rawRoots[rawKey(testFieldBySourceName(loc, "raw"))] = true
	}
	key := rawKey(testFieldBySourceName(allocator, "pending"))
	if key[1] != 0 {
		rawRoots[key] = true
	}
	// These owners are explicitly supplied by the caller. No Runtime directory
	// supplies authority for a missing debug root.
	for _, extra := range extras {
		for _, b := range extra.CallerBlocks {
			key := [3]uint64{b.OwnerID(), uint64(b.Slot()), uint64(b.Generation())}
			if key[0] != testFieldBySourceName(runtime, "origin").Uint() || key[1] == 0 || key[1] > uint64(blocks.Len()) || rawRoots[key] {
				t.Fatal("invalid or duplicate caller raw acquisition", key)
			}
			block := blocks.Index(int(key[1] - 1))
			if !testFieldBySourceName(block, "live").Bool() || testFieldBySourceName(block, "generation").Uint() != key[2] {
				t.Fatal("ended caller raw acquisition", key)
			}
			rawRoots[key] = true
		}
		for _, d := range extra.DebugContexts {
			if d == nil || d.Self != d {
				t.Fatal("invalid named DebugContext")
			}
		}
	}
	// The explicitly passed manager owns these concrete lexical formatting
	// fields. Enumerate only their actual returned acquisition tokens, never
	// infer missing roots from Runtime live blocks.
	formatOwner := testFieldBySourceName(z, "natural").Elem()
	addFormattingRaw := func(v reflect.Value) {
		for _, field := range []string{"raw", "pending"} {
			key := rawKey(testFieldBySourceName(v, field))
			if key[1] != 0 {
				if key[0] != testFieldBySourceName(runtime, "origin").Uint() || rawRoots[key] {
					t.Fatal("foreign or duplicate lexical formatting root", key)
				}
				rawRoots[key] = true
			}
		}
	}
	formatting := testFieldBySourceName(formatOwner, "formatting")
	if testFieldBySourceName(formatting, "active").Bool() || testFieldBySourceName(formatting, "pending").Bool() || testFieldBySourceName(formatting, "completed").Uint() != 0 {
		locals := testFieldBySourceName(formatting, "locals")
		for i := 0; i < locals.Len(); i++ {
			addFormattingRaw(locals.Index(i))
		}
	}
	for i := 0; i < blocks.Len(); i++ {
		block := blocks.Index(i)
		if testFieldBySourceName(block, "live").Bool() {
			key := [3]uint64{testFieldBySourceName(runtime, "origin").Uint(), uint64(i + 1), testFieldBySourceName(block, "generation").Uint()}
			if !rawRoots[key] {
				t.Fatal("anonymous raw numeric allocation", key)
			}
		}
	}
	// Numeral and Natural storage no longer publish Words descriptors.
	return graph
}
