package polynomial

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUPolynomialHenselConcreteVectorOwnerCensus(t *testing.T) {
	m, prime, lift, donor, pf, lf, service, f, _, _ := upolyHenselPhysicalFixture(t)
	cores := []*UPolynomialManager{m, prime, lift, donor}
	// The donor has no live polynomials: its buffers are actually owned by pf.
	// A store's live backing registry alone must not supply their missing owners.
	if e := upolyHenselCheckVectorOwners(cores, []*UPolynomialFactors{lf}); e == nil {
		t.Fatal("omitted owning pf accepted as store ownership")
	}
	if e := upolyHenselCheckVectorOwners(cores, []*UPolynomialFactors{pf, lf}); e != nil {
		t.Fatal(e)
	}
	if e := service.HenselLiftFactors(f, pf, 16, lf); e != nil {
		t.Fatal(e)
	}
	if e := upolyHenselCheckVectorOwners(cores, []*UPolynomialFactors{pf}); e == nil {
		t.Fatal("omitted actual lifted output owner accepted")
	}
	if e := upolyHenselCheckVectorOwners(cores, []*UPolynomialFactors{pf, lf}); e != nil {
		t.Fatal(e)
	}
	upolyHenselAuditPhysical(t, cores, []*UPolynomialFactors{pf, lf})
}

// Every live coefficient buffer must be reachable through a concrete source
// vector state; the storage registry is physical metadata, not a vector owner.
func upolyHenselCheckVectorOwners(cores []*UPolynomialManager, factors []*UPolynomialFactors, extras ...upolyPhysicalExtraOwners) error {
	var e error
	cores, factors, e = upolyPhysicalRetainedOwners(cores, factors, extras)
	if e != nil {
		return e
	}
	stores := map[uint64]reflect.Value{}
	for _, core := range cores {
		v := reflect.ValueOf(core).Elem()
		store := testFieldBySourceName(v, "store").Elem()
		stores[testFieldBySourceName(store, "origin").Uint()] = store
		bindings := testFieldBySourceName(v, "bindings")
		for i := 0; i < bindings.Len(); i++ {
			store = testFieldBySourceName(bindings.Index(i), "store").Elem()
			stores[testFieldBySourceName(store, "origin").Uint()] = store
		}
	}
	owners := map[[3]uint64]string{}
	add := func(core reflect.Value, state reflect.Value, label string) error {
		backing := testFieldBySourceName(state, "backing").Uint()
		if backing == 0 {
			return nil
		}
		origin := testFieldBySourceName(state, "origin").Uint()
		if origin == 0 {
			origin = testFieldBySourceName(testFieldBySourceName(core, "store").Elem(), "origin").Uint()
		}
		key := [3]uint64{origin, backing, testFieldBySourceName(state, "generation").Uint()}
		store, ok := stores[origin]
		if !ok {
			return fmt.Errorf("%s vector has no bound physical origin %v", label, key)
		}
		backings := testFieldBySourceName(store, "backings")
		if backing > uint64(backings.Len()) {
			return fmt.Errorf("%s vector outside physical directory %v", label, key)
		}
		b := backings.Index(int(backing - 1))
		if !testFieldBySourceName(b, "live").Bool() || testFieldBySourceName(b, "generation").Uint() != key[2] {
			return fmt.Errorf("%s vector physical lifetime mismatch %v", label, key)
		}
		if previous, ok := owners[key]; ok {
			return fmt.Errorf("one buffer has two scalar vector owners %v: %s and %s", key, previous, label)
		}
		owners[key] = label
		if testFieldBySourceName(state, "size").Uint() > uint64(testFieldBySourceName(b, "values").Len()) {
			return fmt.Errorf("%s vector native constructed extent mismatch %v", label, key)
		}
		return nil
	}
	for index, owner := range cores {
		core := reflect.ValueOf(owner).Elem()
		for _, field := range []string{"basic", "div1", "div2", "exactDiv", "gcd1", "gcd2", "cra", "sqf1", "sqf2", "pw", "db", "dbab1", "dbab2", "tr", "push"} {
			if e := add(core, testFieldBySourceName(core, field), fmt.Sprintf("core%d.%s", index, field)); e != nil {
				return e
			}
		}
		mgcd := testFieldBySourceName(core, "mgcd")
		for i := 0; i < mgcd.Len(); i++ {
			if e := add(core, mgcd.Index(i), fmt.Sprintf("core%d.mgcd%d", index, i)); e != nil {
				return e
			}
		}
		pages := testFieldBySourceName(core, "pages")
		for i := 0; i < pages.Len(); i++ {
			rows := testFieldBySourceName(pages.Index(i).Elem(), "rows")
			for j := 0; j < rows.Len(); j++ {
				row := rows.Index(j)
				if testFieldBySourceName(row, "live").Bool() {
					if e := add(core, testFieldBySourceName(row, "state"), fmt.Sprintf("core%d.polynomial%d", index, i*rows.Len()+j)); e != nil {
						return e
					}
				}
			}
		}
	}
	for index, owner := range factors {
		fs := reflect.ValueOf(owner).Elem()
		core := testFieldBySourceName(fs, "manager").Elem()
		states := testFieldBySourceName(fs, "factors")
		for i := uint64(0); i < testFieldBySourceName(fs, "count").Uint(); i++ {
			if e := add(core, states.Index(int(i)), fmt.Sprintf("factors%d.element%d", index, i)); e != nil {
				return e
			}
		}
	}
	if e := upolyPhysicalExtraVectors(extras, add); e != nil {
		return e
	}
	for _, extra := range extras {
		for _, service := range extra.Services {
			front := testFieldBySourceName(reflect.ValueOf(service).Elem(), "frontend")
			root := testFieldBySourceName(reflect.ValueOf(service).Elem(), "manager").Elem()
			for _, name := range []string{"pp", "prime", "A", "B", "D"} {
				if e := add(root, testFieldBySourceName(front, name), "actual factorFrontend."+name); e != nil {
					return e
				}
			}
			for _, name := range []string{"f1", "f2", "product"} {
				if e := add(root, testFieldBySourceName(testFieldBySourceName(front, "quadratic"), name), "actual factorFrontend.quadratic."+name); e != nil {
					return e
				}
			}
			for pair := testFieldBySourceName(reflect.ValueOf(service).Elem(), "henselPairs"); !pair.IsNil(); pair = testFieldBySourceName(pair.Elem(), "next") {
				h := pair.Elem()
				root := testFieldBySourceName(testFieldBySourceName(h, "service").Elem(), "manager").Elem()
				for _, field := range []string{"f", "Vf", "t", "S", "T", "tmp"} {
					owner := root
					if field != "f" && !testFieldBySourceName(h, "rm").IsNil() {
						owner = testFieldBySourceName(h, "rm").Elem()
					}
					if e := add(owner, testFieldBySourceName(h, field), "retained henselPair."+field); e != nil {
						return e
					}
				}
				check := testFieldBySourceName(h, "check")
				for _, field := range []string{"test1", "test2a", "test2b", "test3a", "test3b"} {
					if e := add(root, testFieldBySourceName(check, field), "retained henselCheck."+field); e != nil {
						return e
					}
				}
			}
			scope := testFieldBySourceName(reflect.ValueOf(service).Elem(), "scope")
			if scope.IsNil() {
				continue
			}
			scope = scope.Elem()
			caller := testFieldBySourceName(scope, "hensel")
			for _, name := range []string{"outer", "quadratic", "iteration"} {
				group := testFieldBySourceName(caller, name)
				for i := 0; i < group.Len(); i++ {
					if e := add(testFieldBySourceName(scope, "manager").Elem(), group.Index(i), "retained henselCaller."+name); e != nil {
						return e
					}
				}
			}
			prime := testFieldBySourceName(scope, "primeTrial")
			if !testFieldBySourceName(prime, "zp").IsNil() {
				if e := add(testFieldBySourceName(prime, "zp").Elem(), testFieldBySourceName(prime, "fzp"), "retained primeTrial.f_pp_zp"); e != nil {
					return e
				}
			}
			core := testFieldBySourceName(scope, "manager").Elem()
			for _, field := range []string{"fpp", "trial", "quotient"} {
				if e := add(core, testFieldBySourceName(scope, field), "retained factorSquareFree."+field); e != nil {
					return e
				}
			}
		}
	}
	for origin, store := range stores {
		backings := testFieldBySourceName(store, "backings")
		for i := 0; i < backings.Len(); i++ {
			b := backings.Index(i)
			if testFieldBySourceName(b, "live").Bool() {
				key := [3]uint64{origin, uint64(i + 1), testFieldBySourceName(b, "generation").Uint()}
				if _, ok := owners[key]; !ok {
					return fmt.Errorf("anonymous live coefficient vector %v", key)
				}
			}
		}
	}
	return nil
}

// Only actual declared retained lexical owner fields are roots. Accessors borrow
// the charged owners; no live-descriptor registry can manufacture ownership.
// Read actual acquisition fields independently of public borrow APIs: a pending
// factor display deliberately refuses those APIs even for the root manager.
func upolyPhysicalRetainedOwners(cores []*UPolynomialManager, factors []*UPolynomialFactors, extras []upolyPhysicalExtraOwners) ([]*UPolynomialManager, []*UPolynomialFactors, error) {
	cs := append([]*UPolynomialManager(nil), cores...)
	fs := append([]*UPolynomialFactors(nil), factors...)
	addCore := func(m *UPolynomialManager) {
		if m == nil {
			return
		}
		for _, old := range cs {
			if old == m {
				return
			}
		}
		cs = append(cs, m)
	}
	addFactors := func(f *UPolynomialFactors) {
		if f == nil {
			return
		}
		for _, old := range fs {
			if old == f {
				return
			}
		}
		fs = append(fs, f)
	}
	for _, extra := range extras {
		for _, service := range extra.Services {
			if service == nil || service.self != service || service.closed || service.manager == nil || service.manager.self != service.manager {
				return nil, nil, ErrUPolynomialOwner
			}
			if c := service.scope; c != nil {
				if c.self != c || c.service != service {
					return nil, nil, ErrUPolynomialOwner
				}
				for _, m := range [3]*UPolynomialManager{c.zpe, c.zp, c.hensel.related} {
					addCore(m)
				}
				for _, f := range [3]*UPolynomialFactors{c.primeTrial.current, c.lifted, c.best} {
					addFactors(f)
				}
			}
			for h := service.henselPairs; h != nil; h = h.next {
				for _, m := range [3]*UPolynomialManager{h.check.bCore, h.check.brCore, h.rm} {
					if m != nil && !m.closed {
						addCore(m)
					}
				}
			}
		}
	}
	return cs, fs, nil
}
