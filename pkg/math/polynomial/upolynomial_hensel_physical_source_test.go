// Source: src/math/polynomial/upolynomial.cpp.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func upolyHenselPhysicalFixture(t *testing.T) (*UPolynomialManager, *UPolynomialManager, *UPolynomialManager, *UPolynomialManager, *UPolynomialFactors, *UPolynomialFactors, *UPolynomialFactorizationService, UPolynomialView, UPolynomialView, *arena.Budget) {
	t.Helper()
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	config := UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }}
	cores := make([]*UPolynomialManager, 3)
	for i := range cores {
		var e error
		cores[i], e = NewUPolynomialManager(config)
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
	primes, e := primegen.NewGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	})
	service, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
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
	var captured UPolynomialView
	for i, cs := range [][]int32{{-1, 1}, {1, 1}, {2, 1}} {
		p, e := donor.NewPolynomial()
		if e != nil {
			t.Fatal(e)
		}
		values := []unumeral.Int{unumeral.NewInt(cs[0]), unumeral.NewInt(cs[1])}
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

type upolyHenselPhysicalGraph struct{ Named, Unpublished, Cleanup, Pending, Objects uint32 }

// Read-only independent physical ownership audit. Reflection reads scalar
// metadata; it neither borrows mutable payloads nor repairs failed numerals.
func upolyHenselAuditPhysical(t *testing.T, cores []*UPolynomialManager, factors []*UPolynomialFactors, extras ...upolyPhysicalExtraOwners) upolyHenselPhysicalGraph {
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
	stores := map[uintptr]reflect.Value{}
	for _, owner := range cores {
		core := reflect.ValueOf(owner).Elem()
		store := testFieldBySourceName(core, "store")
		stores[store.Pointer()] = store.Elem()
		bindings := testFieldBySourceName(core, "bindings")
		for i := 0; i < bindings.Len(); i++ {
			store = testFieldBySourceName(bindings.Index(i), "store")
			stores[store.Pointer()] = store.Elem()
		}
		modular := testFieldBySourceName(core, "modular").Elem()
		for _, field := range []string{"p", "lower", "upper", "invTmp1", "invTmp2", "invTmp3", "divTmp"} {
			addInt(testFieldBySourceName(modular, field))
		}
	}
	// All initialized coefficient storage belongs to actual live buffers, even
	// when a failed source copy has not yet published its logical vector size.
	for _, store := range stores {
		backings := testFieldBySourceName(store, "backings")
		for i := 0; i < backings.Len(); i++ {
			b := backings.Index(i)
			if !testFieldBySourceName(b, "live").Bool() {
				continue
			}
			values := testFieldBySourceName(b, "values")
			n := uint64(values.Len())
			for j := uint64(0); j < n; j++ {
				addInt(values.Index(int(j)))
			}
		}
	}
	for _, fs := range factors {
		addInt(testFieldBySourceName(reflect.ValueOf(fs).Elem(), "constant"))
	}
	upolyPhysicalExtraNumerals(extras, addInt)
	for _, extra := range extras {
		for _, service := range extra.Services {
			front := testFieldBySourceName(reflect.ValueOf(service).Elem(), "frontend")
			addInt(testFieldBySourceName(front, "content"))
			for _, x := range []struct{ scope, value string }{{"vectorNormalize", "g"}, {"vectorNormalize", "old"}, {"vectorNormalize", "tmp"}, {"modularPower", "power"}, {"flip", "c"}, {"image", "old"}, {"image", "c"}, {"remainder", "am"}, {"monic", "lc"}, {"monic", "inv"}, {"derivative", "d"}, {"divisibility", "ratio"}, {"primitive", "cont"}} {
				addInt(testFieldBySourceName(testFieldBySourceName(front, x.scope), x.value))
			}
			for _, name := range []string{"quadratic", "modularGCD", "cra"} {
				values := testFieldBySourceName(testFieldBySourceName(front, name), "values")
				for i := 0; i < values.Len(); i++ {
					addInt(values.Index(i))
				}
			}
			for r := testFieldBySourceName(testFieldBySourceName(front, "derivative"), "retired"); !r.IsNil(); r = testFieldBySourceName(r.Elem(), "next") {
				addInt(testFieldBySourceName(r.Elem(), "value"))
			}
			for pair := testFieldBySourceName(reflect.ValueOf(service).Elem(), "henselPairs"); !pair.IsNil(); pair = testFieldBySourceName(pair.Elem(), "next") {
				h := pair.Elem()
				if testFieldBySourceName(h, "amConstructed").Bool() {
					addInt(testFieldBySourceName(h, "am"))
				}
				check := testFieldBySourceName(h, "check")
				if testFieldBySourceName(check, "constructed").Uint() >= 1 {
					addInt(testFieldBySourceName(check, "br"))
				}
				if testFieldBySourceName(check, "transportBound").Bool() {
					addInt(testFieldBySourceName(check, "transport"))
				}
			}
			scope := testFieldBySourceName(reflect.ValueOf(service).Elem(), "scope")
			if scope.IsNil() {
				continue
			}
			scope = scope.Elem()
			caller := testFieldBySourceName(scope, "hensel")
			for _, n := range []struct{ flag, value string }{{"amConstructed", "am"}, {"initialLCConstructed", "initialLC"}, {"finalLCInvConstructed", "finalLCInv"}, {"checkPowerConstructed", "checkPower"}, {"transportBound", "transport"}} {
				if testFieldBySourceName(caller, n.flag).Bool() {
					addInt(testFieldBySourceName(caller, n.value))
				}
			}
			if testFieldBySourceName(caller, "extendedRawConstructed").Bool() {
				for i := 0; i < 2; i++ {
					addInt(testFieldBySourceName(caller, "monicRaw").Index(i))
				}
			}
			for i := 0; i < int(testFieldBySourceName(caller, "extendedScopedConstructed").Uint()); i++ {
				addInt(testFieldBySourceName(caller, "extendedScoped").Index(i))
			}
			prime := testFieldBySourceName(scope, "primeTrial")
			if testFieldBySourceName(prime, "transportBound").Bool() {
				addInt(testFieldBySourceName(prime, "transport"))
			}
			for _, field := range []string{"p", "bestP", "gcdTmp", "leading", "trialContent"} {
				addInt(testFieldBySourceName(scope, field))
			}
		}
	}
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

func TestUPolynomialHenselImportedBigModulusPhysicalFailureRoots(t *testing.T) {
	failures, successes, big, pending, unpublished, cleanup := 0, 0, 0, 0, 0, 0
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		succeeded := false
		passed := t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, _, budget := upolyHenselPhysicalFixture(t)
			cores := []*UPolynomialManager{m, prime, lift, donor}
			owners := []*UPolynomialFactors{pf, lf}
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
				var failure umemory.Failure
				if !errors.As(err, &failure) || failure.Kind != umemory.FailureMemout || !m.IntegerManager().RuntimeOwner().State().OutOfMemory {
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
			var modulus unumeral.Int
			if e := lift.Modulus(&modulus); e != nil {
				t.Fatal(e)
			}
			if modulus.IsSmall() {
				t.Fatal("5^16 must have physical numeric payload")
			}
			var expected unumeral.Int
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
func upolyHenselCallerSetZForDeadBounds(t *testing.T, cores []*UPolynomialManager, owners []*UPolynomialFactors, extras ...upolyPhysicalExtraOwners) uint32 {
	var e error
	cores, owners, e = upolyPhysicalRetainedOwners(cores, owners, extras)
	if e != nil {
		t.Fatal(e)
	}
	t.Helper()
	integer := reflect.ValueOf(cores[0].IntegerManager()).Elem()
	failed := testFieldBySourceName(integer, "failedRef").Uint()
	failedNull := testFieldBySourceName(integer, "failedNullRef").Uint()
	transitions := uint32(0)
	for _, core := range cores {
		modular := testFieldBySourceName(reflect.ValueOf(core).Elem(), "modular").Elem()
		beforeValues := [3][2]uint64{}
		dead := false
		for i, field := range []string{"p", "lower", "upper"} {
			v := testFieldBySourceName(modular, field)
			beforeValues[i] = [2]uint64{uint64(testFieldBySourceName(v, "val").Int()), testFieldBySourceName(v, "cell").Uint()}
			dead = dead || beforeValues[i][1] == failed || beforeValues[i][1] == failedNull
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
			v := testFieldBySourceName(modular, field)
			after := [2]uint64{uint64(testFieldBySourceName(v, "val").Int()), testFieldBySourceName(v, "cell").Uint()}
			if after != beforeValues[i] {
				t.Fatal("SetZ repaired/mutated source bound", field, beforeValues[i], after)
			}
		}
		transitions++
	}
	return transitions
}

func TestUPolynomialHenselDeadLowerGuardedCloseAndCallerSetZ(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("exact ordinal159 belongs to source debug direct allocator profile")
	}
	m, prime, lift, donor, pf, lf, service, f, _, b := upolyHenselPhysicalFixture(t)
	allocator := testFieldBySourceName(reflect.ValueOf(m.IntegerManager()).Elem(), "allocator").Elem()
	profile := testFieldBySourceName(testFieldBySourceName(allocator, "config"), "Profile")
	if !testFieldBySourceName(profile, "Debug").Bool() || testFieldBySourceName(profile, "Windows").Bool() {
		t.Skip("ordinal159 requires actual non-Windows debug direct allocator profile")
	}
	locations := testFieldBySourceName(allocator, "locations")
	chunks, direct := 0, 0
	for i := 0; i < locations.Len(); i++ {
		loc := locations.Index(i)
		if testFieldBySourceName(loc, "live").Bool() {
			switch testFieldBySourceName(loc, "kind").Uint() {
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
	cores := []*UPolynomialManager{m, prime, lift, donor}
	owners := []*UPolynomialFactors{pf, lf}
	// The cached pair frame and public caller adapter are real cold owners.
	// Their two reservations precede this unchanged setup_p lower Copy;
	// the independent exact physical frontier probe reaches it at159.
	b.FailAfter(159)
	err := service.HenselLiftFactors(f, pf, 16, lf)
	var failure umemory.Failure
	if !errors.As(err, &failure) || failure.Kind != umemory.FailureMemout || !m.IntegerManager().RuntimeOwner().State().OutOfMemory {
		t.Fatal("frozen debug raw-allocation refusal prefix", err)
	}
	upolyHenselAuditPhysical(t, cores, owners)
	z := reflect.ValueOf(m.IntegerManager()).Elem()
	failed := testFieldBySourceName(z, "failedNullRef").Uint()
	modular := testFieldBySourceName(reflect.ValueOf(lift).Elem(), "modular").Elem()
	if testFieldBySourceName(testFieldBySourceName(modular, "lower"), "cell").Uint() != failed || testFieldBySourceName(testFieldBySourceName(modular, "p"), "cell").Uint() == failed || testFieldBySourceName(testFieldBySourceName(modular, "upper"), "cell").Uint() == failed {
		t.Fatal("exact setup_p lower-copy failure")
	}
	b.FailAfter(-1)
	if e := lf.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("source normalize continuation must remain guarded", e)
	}
	if lf.Stats().OwnerBytes == 0 || testFieldBySourceName(reflect.ValueOf(lift).Elem(), "liveFactors").Uint() != 1 || !testFieldBySourceName(reflect.ValueOf(lf).Elem(), "closing").Bool() {
		t.Fatal("interrupted closing owner was dropped")
	}
	upolyHenselAuditPhysical(t, cores, owners)
	if n := upolyHenselCallerSetZForDeadBounds(t, cores, owners); n != 1 {
		t.Fatal("exact one source mode transition", n)
	}
	if e := lf.Close(); e != nil {
		t.Fatal("retry after caller mode transition", e)
	}
	if lf.Stats().TotalBytes != 0 || testFieldBySourceName(reflect.ValueOf(lift).Elem(), "liveFactors").Uint() != 0 {
		t.Fatal("retry retained factor header")
	}
	upolyHenselAuditPhysical(t, cores, owners)
}

// Raw source allocation uses its actual memout transport, while typed Go
// bookkeeping preserves arena refusal. No other error category is accepted.
func upolyHenselAllocationRefusal(z *unumeral.IntegerManager, err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var failure umemory.Failure
	return errors.As(err, &failure) && failure.Kind == umemory.FailureMemout && z.RuntimeOwner().State().OutOfMemory
}

// Source: src/util/mpz.cpp:730-817.
// Retained Lehmer declarations belong to the external shared Integer owner,
// not UPolynomial cached/owned stats. Name only actual source frame fields.
func upolyIntegerGCDRawOwnedCells(t *testing.T, zowner *unumeral.IntegerManager) uint32 {
	t.Helper()
	z := reflect.ValueOf(zowner).Elem()
	if testFieldBySourceName(z, "self").Pointer() != z.Addr().Pointer() {
		t.Fatal("invalid Integer census owner")
	}
	raw := testFieldBySourceName(z, "gcdRaw")
	if testFieldBySourceName(raw, "invocation").Uint() == 0 {
		return 0
	}
	if testFieldBySourceName(raw, "owner").Pointer() != z.Addr().Pointer() {
		t.Fatal("invalid actual GCD frame owner")
	}
	cells := testFieldBySourceName(z, "cells").Elem()
	pages := testFieldBySourceName(cells, "pages")
	pageSize := testFieldBySourceName(testFieldBySourceName(cells, "config"), "PageSize").Uint()
	named := map[uint64]bool{}
	add := func(v reflect.Value) {
		ref := testFieldBySourceName(v, "cell").Uint()
		if ref == 0 || ref == testFieldBySourceName(z, "failedRef").Uint() || ref == testFieldBySourceName(z, "failedNullRef").Uint() {
			return
		}
		slot := uint64(uint32(ref))
		if slot == 0 || slot > testFieldBySourceName(cells, "slots").Uint() {
			t.Fatal("named GCD source token outside directory", ref)
		}
		state := testFieldBySourceName(pages.Index(int((slot-1)/pageSize)), "meta").Index(int((slot - 1) % pageSize))
		if !testFieldBySourceName(state, "live").Bool() || testFieldBySourceName(z, "failedRef").Type().Size() == 8 && testFieldBySourceName(state, "generation").Uint() != ref>>32 {
			t.Fatal("named GCD token stale", ref)
		}
		named[testFieldBySourceName(state, "generation").Uint()<<32|slot] = true
	}
	values := testFieldBySourceName(raw, "values")
	count := testFieldBySourceName(raw, "count").Uint()
	if count != 5 {
		t.Fatal("invalid actual GCD declaration count", count)
	}
	for i := uint64(0); i < count; i++ {
		add(values.Index(int(i)))
	}
	child := testFieldBySourceName(raw, "addMul")
	scope := testFieldBySourceName(z, "addMulInvocation")
	parent := testFieldBySourceName(scope, "parentContext")
	rawReceipt := testFieldBySourceName(parent, "raw")
	if testFieldBySourceName(child, "manager").Pointer() == z.Addr().Pointer() && testFieldBySourceName(child, "frame").Pointer() == scope.Addr().Pointer() && testFieldBySourceName(child, "invocation").Uint() != 0 && testFieldBySourceName(child, "invocation").Uint() == testFieldBySourceName(scope, "invocation").Uint() && testFieldBySourceName(scope, "active").Bool() {
		if rawReceipt.IsNil() || !testFieldBySourceName(raw, "active").Bool() || testFieldBySourceName(rawReceipt.Elem(), "manager").Pointer() != z.Addr().Pointer() || testFieldBySourceName(rawReceipt.Elem(), "invocation").Uint() != testFieldBySourceName(raw, "invocation").Uint() || testFieldBySourceName(rawReceipt.Elem(), "frame").Pointer() != raw.Addr().Pointer() || testFieldBySourceName(parent, "rawInvocation").Uint() != testFieldBySourceName(raw, "invocation").Uint() {
			t.Fatal("invalid matched GCD child parent context")
		}
		add(testFieldBySourceName(scope, "tmp"))
	}
	return uint32(len(named))
}

// Source: src/util/mpz.h:393-426. Public AddMul/SubMul retain the actual
// reached RAW temporary separately from UPolynomial members and cached cells.
func upolyIntegerRawOwnedCells(t *testing.T, owner *unumeral.IntegerManager) uint32 {
	t.Helper()
	count := upolyIntegerGCDRawOwnedCells(t, owner)
	z := reflect.ValueOf(owner).Elem()
	scope := testFieldBySourceName(z, "addMulInvocation")
	if !testFieldBySourceName(scope, "active").Bool() {
		return count
	}
	// A GCD child is already named by its exact parent receipt above.
	if !testFieldBySourceName(testFieldBySourceName(scope, "parentContext"), "raw").IsNil() {
		return count
	}
	receipt := testFieldBySourceName(scope, "receipt")
	if testFieldBySourceName(receipt, "manager").Pointer() != z.Addr().Pointer() || testFieldBySourceName(receipt, "frame").Pointer() != scope.Addr().Pointer() || testFieldBySourceName(receipt, "invocation").Uint() == 0 || testFieldBySourceName(receipt, "invocation").Uint() != testFieldBySourceName(scope, "invocation").Uint() {
		t.Fatal("invalid actual public AddMul/SubMul receipt")
	}
	ref := testFieldBySourceName(testFieldBySourceName(scope, "tmp"), "cell").Uint()
	if ref == 0 || ref == testFieldBySourceName(z, "failedRef").Uint() || ref == testFieldBySourceName(z, "failedNullRef").Uint() {
		return count
	}
	cells := testFieldBySourceName(z, "cells").Elem()
	slot := uint64(uint32(ref))
	if slot == 0 || slot > testFieldBySourceName(cells, "slots").Uint() {
		t.Fatal("public RAW token outside directory", ref)
	}
	pageSize := testFieldBySourceName(testFieldBySourceName(cells, "config"), "PageSize").Uint()
	state := testFieldBySourceName(testFieldBySourceName(cells, "pages").Index(int((slot-1)/pageSize)), "meta").Index(int((slot - 1) % pageSize))
	if !testFieldBySourceName(state, "live").Bool() || testFieldBySourceName(z, "failedRef").Type().Size() == 8 && testFieldBySourceName(state, "generation").Uint() != ref>>32 {
		t.Fatal("public RAW token stale", ref)
	}
	return count + 1
}
