// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func upolyHenselPhysicalFixture(t *testing.T) (*mpnum.UPolynomialManager, *mpnum.UPolynomialManager, *mpnum.UPolynomialManager, *mpnum.UPolynomialManager, *mpnum.UPolynomialFactors, *mpnum.UPolynomialFactors, *mpnum.UPolynomialFactorizationService, mpnum.UPolynomialView, mpnum.UPolynomialView, *arena.Budget) {
	t.Helper()
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	config := mpnum.UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }}
	cores := make([]*mpnum.UPolynomialManager, 3)
	for i := range cores {
		var e error
		cores[i], e = mpnum.NewUPolynomialManager(config)
		if e != nil {
			t.Fatal(e)
		}
		owner := cores[i]
		t.Cleanup(func() {
			if e := owner.Close(); e != nil && !errors.Is(e, arena.ErrClosed) {
				t.Error(e)
			}
		})
	}
	prime, lift, donor := cores[0], cores[1], cores[2]
	if e := prime.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	primes, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	})
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	})
	pf, e := prime.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := pf.Close(); e != nil {
			t.Error(e)
		}
	})
	lf, e := lift.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := lf.Close(); e != nil && !errors.Is(e, arena.ErrClosed) {
			t.Error(e)
		}
	})
	var captured mpnum.UPolynomialView
	for i, cs := range [][]int32{{-1, 1}, {1, 1}, {2, 1}} {
		p, e := donor.NewPolynomial()
		if e != nil {
			t.Fatal(e)
		}
		values := []mpnum.Int{mpnum.NewInt(cs[0]), mpnum.NewInt(cs[1])}
		if e = donor.SetInts(p, values); e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			captured = mustUPolyView(t, donor, p)
		}
		if e = pf.PushMoveFrom(donor, p, 1); e != nil {
			t.Fatal(e)
		}
		if e = donor.DelPolynomial(&p); e != nil {
			t.Fatal(e)
		}
	}
	f := upolySourceNew(t, m, []int32{-2, -1, 2, 1})
	return m, prime, lift, donor, pf, lf, service, mustUPolyView(t, m, f), captured, b
}

type upolyHenselPhysicalGraph struct{ Named, Unpublished, Cleanup, Pending, Objects, Words uint32 }

// Read-only independent physical ownership audit. Reflection reads scalar
// metadata; it neither borrows mutable payloads nor repairs failed numerals.
func upolyHenselAuditPhysical(t *testing.T, cores []*mpnum.UPolynomialManager, factors []*mpnum.UPolynomialFactors, extras ...upolyPhysicalExtraOwners) upolyHenselPhysicalGraph {
	var e error
	cores, factors, e = upolyPhysicalRetainedOwners(cores, factors, extras)
	if e != nil {
		t.Fatal(e)
	}
	t.Helper()
	if e := upolyHenselCheckVectorOwners(cores, factors, extras...); e != nil {
		t.Fatal(e)
	}
	var graph upolyHenselPhysicalGraph
	z := reflect.ValueOf(cores[0].IntegerManager()).Elem()
	cells := z.FieldByName("cells").Elem()
	pages := cells.FieldByName("pages")
	pageSize := cells.FieldByName("config").FieldByName("PageSize").Uint()
	slots := cells.FieldByName("slots").Uint()
	failed := z.FieldByName("failedRef").Uint()
	named := map[uint64]bool{}
	add := func(ref uint64) {
		if ref == 0 || ref == failed {
			return
		}
		slot, gen := uint64(uint32(ref)), ref>>32
		if slot == 0 || slot > slots {
			t.Fatal("owning scalar ref outside numeric directory", ref)
		}
		state := pages.Index(int((slot - 1) / pageSize)).FieldByName("meta").Index(int((slot - 1) % pageSize))
		if !state.FieldByName("live").Bool() || (z.FieldByName("failedRef").Type().Size() == 8 && state.FieldByName("generation").Uint() != gen) {
			t.Fatal("owning scalar ref stale", ref)
		}
		// Release source pointers carry only slots; checked tokens also carry
		// generation. Canonical census identities use the actual live ledger.
		named[state.FieldByName("generation").Uint()<<32|slot] = true
	}
	addInt := func(v reflect.Value) { add(v.FieldByName("cell").Uint()) }
	for _, field := range []string{"tmp", "arg"} {
		v := z.FieldByName(field)
		for i := 0; i < v.Len(); i++ {
			add(v.Index(i).Uint())
		}
	}
	addInt(z.FieldByName("intMin"))
	addInt(z.FieldByName("two64"))
	stores := map[uintptr]reflect.Value{}
	for _, owner := range cores {
		core := reflect.ValueOf(owner).Elem()
		store := core.FieldByName("store")
		stores[store.Pointer()] = store.Elem()
		bindings := core.FieldByName("bindings")
		for i := 0; i < bindings.Len(); i++ {
			store = bindings.Index(i).FieldByName("store")
			stores[store.Pointer()] = store.Elem()
		}
		modular := core.FieldByName("modular").Elem()
		for _, field := range []string{"p", "lower", "upper", "invTmp1", "invTmp2", "invTmp3", "divTmp"} {
			addInt(modular.FieldByName(field))
		}
	}
	// All initialized coefficient storage belongs to actual live buffers, even
	// when a failed source copy has not yet published its logical vector size.
	for _, store := range stores {
		backings := store.FieldByName("backings")
		for i := 0; i < backings.Len(); i++ {
			b := backings.Index(i)
			if !b.FieldByName("live").Bool() {
				continue
			}
			values := b.FieldByName("values")
			n := b.FieldByName("initialized").Uint()
			if n > uint64(values.Len()) {
				t.Fatal("coefficient physical extent", n, values.Len())
			}
			for j := uint64(0); j < n; j++ {
				addInt(values.Index(int(j)))
			}
		}
	}
	for _, fs := range factors {
		addInt(reflect.ValueOf(fs).Elem().FieldByName("constant"))
	}
	upolyPhysicalExtraNumerals(extras, addInt)
	for _, extra := range extras {
		for _, service := range extra.Services {
			scope := reflect.ValueOf(service).Elem().FieldByName("scope")
			if scope.IsNil() {
				continue
			}
			scope = scope.Elem()
			for _, field := range []string{"p", "bestP", "gcdTmp", "leading", "trialContent"} {
				addInt(scope.FieldByName(field))
			}
		}
	}
	runtime := z.FieldByName("memory").Elem()
	blocks := runtime.FieldByName("blocks")
	allocator := z.FieldByName("allocator").Elem()
	locations := allocator.FieldByName("locations")
	objects := map[[4]uint64]bool{}
	rawRoots := map[[3]uint64]bool{}
	rawKey := func(v reflect.Value) [3]uint64 {
		return [3]uint64{v.FieldByName("origin").Uint(), v.FieldByName("slot").Uint(), v.FieldByName("generation").Uint()}
	}
	addBacking := func(backing reflect.Value, complete bool) {
		object := backing.FieldByName("object")
		key := [4]uint64{object.FieldByName("owner").Uint(), object.FieldByName("address").Uint(), object.FieldByName("slot").Uint(), object.FieldByName("generation").Uint()}
		if key == [4]uint64{} {
			return
		}
		if key[0] != allocator.FieldByName("origin").Uint() || key[2] == 0 || key[2] > uint64(locations.Len()) {
			t.Fatal("physical object owner/slot", key)
		}
		loc := locations.Index(int(key[2] - 1))
		if !loc.FieldByName("live").Bool() || loc.FieldByName("free").Bool() || loc.FieldByName("generation").Uint() != key[3] || loc.FieldByName("address").Uint() != key[1] {
			t.Fatal("physical owned object lifetime", key, loc)
		}
		if objects[key] {
			t.Fatal("two numeric owners of one physical object", key)
		}
		objects[key] = true
		raw := rawKey(loc.FieldByName("raw"))
		if raw[0] != runtime.FieldByName("origin").Uint() || raw[1] == 0 || raw[1] > uint64(blocks.Len()) {
			t.Fatal("owned cell raw block", raw)
		}
		rawBlock := blocks.Index(int(raw[1] - 1))
		if !rawBlock.FieldByName("live").Bool() || rawBlock.FieldByName("generation").Uint() != raw[2] {
			t.Fatal("owned cell raw lifetime", raw)
		}
		extent := loc.FieldByName("extent").Uint()
		if loc.FieldByName("kind").Uint() == 3 {
			width := runtime.FieldByName("config").FieldByName("SizeTBits").Uint() / 8
			data := rawBlock.FieldByName("data")
			initialized := rawBlock.FieldByName("initialized")
			extent = 0
			for i := uint64(0); i < width; i++ {
				if initialized.Index(int(i/64)).Uint()&(uint64(1)<<(i%64)) == 0 {
					t.Fatal("direct runtime header not initialized")
				}
				extent |= data.Index(int(i)).Uint() << (8 * i)
			}
			extent -= width
		}
		if complete && (rawKey(loc.FieldByName("raw")) != rawKey(backing.FieldByName("raw")) || loc.FieldByName("offset").Uint() != backing.FieldByName("offset").Uint() || extent != backing.FieldByName("extent").Uint() || loc.FieldByName("requested").Uint() != backing.FieldByName("requested").Uint()) {
			t.Fatal("physical cell bounds/location", key)
		}
	}
	live := uint32(0)
	for slot := uint64(1); slot <= slots; slot++ {
		page := pages.Index(int((slot - 1) / pageSize))
		index := int((slot - 1) % pageSize)
		state := page.FieldByName("meta").Index(index)
		if !state.FieldByName("live").Bool() {
			continue
		}
		live++
		ref := state.FieldByName("generation").Uint()<<32 | slot
		record := page.FieldByName("records").Index(index)
		if named[ref] {
			graph.Named++
		} else if record.FieldByName("unpublished").Bool() {
			graph.Unpublished++
		} else if record.FieldByName("cleanupPending").Bool() {
			graph.Cleanup++
		} else {
			t.Fatal("anonymous published numeric descriptor", ref)
		}
		if !record.FieldByName("released").Bool() {
			addBacking(record.FieldByName("backing"), true)
		}
	}
	if live != graph.Named+graph.Unpublished+graph.Cleanup || uint32(len(named)) != graph.Named || uint64(live) != cells.FieldByName("live").Uint() {
		t.Fatal("exact descriptor root census", live, graph, len(named))
	}
	pending := z.FieldByName("pendingCell")
	if pending.FieldByName("object").FieldByName("slot").Uint() != 0 {
		graph.Pending++
		addBacking(pending, false)
	}
	graph.Objects = uint32(len(objects))
	for i := 0; i < locations.Len(); i++ {
		loc := locations.Index(i)
		if !loc.FieldByName("live").Bool() {
			continue
		}
		if !loc.FieldByName("free").Bool() && loc.FieldByName("kind").Uint() != 1 {
			key := [4]uint64{allocator.FieldByName("origin").Uint(), loc.FieldByName("address").Uint(), uint64(i + 1), loc.FieldByName("generation").Uint()}
			if !objects[key] {
				t.Fatal("anonymous live allocator object", key)
			}
		}
		rawRoots[rawKey(loc.FieldByName("raw"))] = true
	}
	for _, field := range []string{"pending", "pendingScratch"} {
		v := allocator.FieldByName(field)
		if field == "pending" {
			key := rawKey(v)
			if key[1] != 0 {
				rawRoots[key] = true
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				key := rawKey(v.Index(i))
				if key[1] != 0 {
					rawRoots[key] = true
				}
			}
		}
	}
	for i := 0; i < blocks.Len(); i++ {
		block := blocks.Index(i)
		if block.FieldByName("live").Bool() {
			key := [3]uint64{runtime.FieldByName("origin").Uint(), uint64(i + 1), block.FieldByName("generation").Uint()}
			if !rawRoots[key] {
				t.Fatal("anonymous raw numeric allocation", key)
			}
		}
	}
	// Words belongs to Natural scratch only. Check each scalar generation against
	// its actual descriptor, rather than inferring one Words root per numeral.
	words := z.FieldByName("words").Elem().FieldByName("descriptors").Elem()
	wordPages := words.FieldByName("pages")
	wordPageSize := words.FieldByName("config").FieldByName("PageSize").Uint()
	wordRoots := map[[2]uint64]bool{}
	natural := z.FieldByName("natural").Elem()
	for _, field := range []string{"divScratch", "formatScratch"} {
		refs := natural.FieldByName(field)
		for i := 0; i < refs.Len(); i++ {
			r := refs.Index(i)
			slot, gen := r.FieldByName("Slot").Uint(), r.FieldByName("Generation").Uint()
			if slot == 0 {
				continue
			}
			state := wordPages.Index(int((slot - 1) / wordPageSize)).FieldByName("meta").Index(int((slot - 1) % wordPageSize))
			if !state.FieldByName("live").Bool() || state.FieldByName("generation").Uint() != gen {
				t.Fatal("Natural scratch word lifetime", slot, gen)
			}
			wordRoots[[2]uint64{slot, gen}] = true
		}
	}
	graph.Words = uint32(len(wordRoots))
	if uint64(graph.Words) != words.FieldByName("live").Uint() {
		t.Fatal("anonymous Natural word descriptor", graph.Words, words.FieldByName("live").Uint())
	}
	return graph
}

func TestUPolynomialHenselImportedBigModulusPhysicalFailureRoots(t *testing.T) {
	failures, successes, big, pending, unpublished, cleanup := 0, 0, 0, 0, 0, 0
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		succeeded := false
		passed := t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, _, budget := upolyHenselPhysicalFixture(t)
			cores := []*mpnum.UPolynomialManager{m, prime, lift, donor}
			owners := []*mpnum.UPolynomialFactors{pf, lf}
			baseline := upolyHenselAuditPhysical(t, cores, owners)
			budget.FailAfter(ordinal)
			err := service.HenselLiftFactors(f, pf, 16, lf)
			// No disable, formatting, retries or owner cleanup before this census.
			immediate := upolyHenselAuditPhysical(t, cores, owners)
			if immediate.Pending > 0 {
				pending++
			}
			if immediate.Unpublished > 0 {
				unpublished++
			}
			if immediate.Cleanup > 0 {
				cleanup++
			}
			if immediate.Named > baseline.Named {
				big++
			}
			if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
				// Actual raw allocation transports budget refusal through source memout.
				// Require both the concrete failure kind and source runtime OOM state.
				var failure memory.Failure
				if !errors.As(err, &failure) || failure.Kind != memory.FailureMemout || !m.IntegerManager().RuntimeOwner().State().OutOfMemory {
					t.Fatal(ordinal, err)
				}
			}

			budget.FailAfter(-1)
			if err != nil {
				upolyHenselCallerSetZForDeadBounds(t, cores, owners)
			}
			if err != nil {
				failures++
				return
			}
			successes++
			succeeded = true
			var modulus mpnum.Int
			if e := lift.Modulus(&modulus); e != nil {
				t.Fatal(e)
			}
			if modulus.IsSmall() {
				t.Fatal("5^16 must have physical numeric payload")
			}
			var expected mpnum.Int
			if e := m.IntegerManager().SetString(&expected, "152587890625"); e != nil {
				t.Fatal(e)
			}
			if same, e := m.IntegerManager().Equal(&modulus, &expected); e != nil || !same {
				t.Fatal("actual source modulus", same, e)
			}
			if e := m.IntegerManager().Del(&modulus); e != nil {
				t.Fatal(e)
			}
			if e := m.IntegerManager().Del(&expected); e != nil {
				t.Fatal(e)
			}
			out := upolySourceNew(t, lift, nil)
			if e := lf.Multiply(out); e != nil {
				t.Fatal(e)
			}
			if same, e := lift.EqualViews(f, mustUPolyView(t, lift, out)); e != nil || !same {
				t.Fatal("actual sixteen-exponent reconstruction", same, e)
			}
		})
		if !passed {
			return
		}
		if succeeded {
			break
		}
	}
	if failures == 0 || successes != 1 || big == 0 || pending == 0 {
		t.Fatal("actual big-root failure and success coverage", failures, successes, big)
	}
	t.Log("physical exponent16 refusal endpoints", failures, "success", successes, "big-root prefixes", big, "pending physical prefixes", pending, "unpublished descriptors", unpublished, "cleanup descriptors", cleanup)
}

// Explicit caller recovery after a failed source setup_p. This only toggles
// source m_z; it does not repair p/bounds, numerals, scratch or Hensel outputs.
func upolyHenselCallerSetZForDeadBounds(t *testing.T, cores []*mpnum.UPolynomialManager, owners []*mpnum.UPolynomialFactors, extras ...upolyPhysicalExtraOwners) uint32 {
	var e error
	cores, owners, e = upolyPhysicalRetainedOwners(cores, owners, extras)
	if e != nil {
		t.Fatal(e)
	}
	t.Helper()
	failed := reflect.ValueOf(cores[0].IntegerManager()).Elem().FieldByName("failedRef").Uint()
	transitions := uint32(0)
	for _, core := range cores {
		modular := reflect.ValueOf(core).Elem().FieldByName("modular").Elem()
		beforeValues := [3][2]uint64{}
		dead := false
		for i, field := range []string{"p", "lower", "upper"} {
			v := modular.FieldByName(field)
			beforeValues[i] = [2]uint64{uint64(v.FieldByName("val").Int()), v.FieldByName("cell").Uint()}
			dead = dead || beforeValues[i][1] == failed
		}
		if !dead {
			continue
		}
		before := upolyHenselAuditPhysical(t, cores, owners, extras...)
		if e := core.SetZ(); e != nil {
			t.Fatal("explicit caller source mode transition", e)
		}
		if after := upolyHenselAuditPhysical(t, cores, owners, extras...); after != before {
			t.Fatal("SetZ changed physical roots", before, after)
		}
		for i, field := range []string{"p", "lower", "upper"} {
			v := modular.FieldByName(field)
			after := [2]uint64{uint64(v.FieldByName("val").Int()), v.FieldByName("cell").Uint()}
			if after != beforeValues[i] {
				t.Fatal("SetZ repaired/mutated source bound", field, beforeValues[i], after)
			}
		}
		transitions++
	}
	return transitions
}

func TestUPolynomialHenselDeadLowerGuardedCloseAndCallerSetZ(t *testing.T) {
	if !cooperationSourceDebug {
		t.Skip("exact ordinal157 belongs to source debug direct allocator profile")
	}
	m, prime, lift, donor, pf, lf, service, f, _, b := upolyHenselPhysicalFixture(t)
	allocator := reflect.ValueOf(m.IntegerManager()).Elem().FieldByName("allocator").Elem()
	profile := allocator.FieldByName("config").FieldByName("Profile")
	if !profile.FieldByName("Debug").Bool() || profile.FieldByName("Windows").Bool() {
		t.Skip("ordinal157 requires actual non-Windows debug direct allocator profile")
	}
	locations := allocator.FieldByName("locations")
	chunks, direct := 0, 0
	for i := 0; i < locations.Len(); i++ {
		loc := locations.Index(i)
		if loc.FieldByName("live").Bool() {
			switch loc.FieldByName("kind").Uint() {
			case 1:
				chunks++
			case 3:
				direct++
			}
		}
	}
	if chunks != 0 || direct == 0 {
		t.Fatal("actual direct profile topology", chunks, direct)
	}
	cores := []*mpnum.UPolynomialManager{m, prime, lift, donor}
	owners := []*mpnum.UPolynomialFactors{pf, lf}
	b.FailAfter(157)
	err := service.HenselLiftFactors(f, pf, 16, lf)
	var failure memory.Failure
	if !errors.As(err, &failure) || failure.Kind != memory.FailureMemout || !m.IntegerManager().RuntimeOwner().State().OutOfMemory {
		t.Fatal("frozen debug raw-allocation refusal prefix", err)
	}
	upolyHenselAuditPhysical(t, cores, owners)
	z := reflect.ValueOf(m.IntegerManager()).Elem()
	failed := z.FieldByName("failedRef").Uint()
	modular := reflect.ValueOf(lift).Elem().FieldByName("modular").Elem()
	if modular.FieldByName("lower").FieldByName("cell").Uint() != failed || modular.FieldByName("p").FieldByName("cell").Uint() == failed || modular.FieldByName("upper").FieldByName("cell").Uint() == failed {
		t.Fatal("exact setup_p lower-copy failure")
	}
	b.FailAfter(-1)
	if e := lf.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("source normalize continuation must remain guarded", e)
	}
	if lf.Stats().OwnerBytes == 0 || reflect.ValueOf(lift).Elem().FieldByName("liveFactors").Uint() != 1 || !reflect.ValueOf(lf).Elem().FieldByName("closing").Bool() {
		t.Fatal("interrupted closing owner was dropped")
	}
	upolyHenselAuditPhysical(t, cores, owners)
	if n := upolyHenselCallerSetZForDeadBounds(t, cores, owners); n != 1 {
		t.Fatal("exact one source mode transition", n)
	}
	if e := lf.Close(); e != nil {
		t.Fatal("retry after caller mode transition", e)
	}
	if lf.Stats().TotalBytes != 0 || reflect.ValueOf(lift).Elem().FieldByName("liveFactors").Uint() != 0 {
		t.Fatal("retry retained factor header")
	}
	upolyHenselAuditPhysical(t, cores, owners)
}

// Raw source allocation uses its actual memout transport, while typed Go
// bookkeeping preserves arena refusal. No other error category is accepted.
func upolyHenselAllocationRefusal(z *mpnum.IntegerManager, err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var failure memory.Failure
	return errors.As(err, &failure) && failure.Kind == memory.FailureMemout && z.RuntimeOwner().State().OutOfMemory
}
