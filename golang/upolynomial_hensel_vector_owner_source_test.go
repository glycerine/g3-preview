// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func TestUPolynomialHenselConcreteVectorOwnerCensus(t *testing.T) {
	m, prime, lift, donor, pf, lf, service, f, _, _ := upolyHenselPhysicalFixture(t)
	cores := []*mpnum.UPolynomialManager{m, prime, lift, donor}
	// The donor has no live polynomials: its buffers are actually owned by pf.
	// A store's live backing registry alone must not supply their missing owners.
	if e := upolyHenselCheckVectorOwners(cores, []*mpnum.UPolynomialFactors{lf}); e == nil {
		t.Fatal("omitted owning pf accepted as store ownership")
	}
	if e := upolyHenselCheckVectorOwners(cores, []*mpnum.UPolynomialFactors{pf, lf}); e != nil {
		t.Fatal(e)
	}
	if e := service.HenselLiftFactors(f, pf, 16, lf); e != nil {
		t.Fatal(e)
	}
	if e := upolyHenselCheckVectorOwners(cores, []*mpnum.UPolynomialFactors{pf}); e == nil {
		t.Fatal("omitted actual lifted output owner accepted")
	}
	if e := upolyHenselCheckVectorOwners(cores, []*mpnum.UPolynomialFactors{pf, lf}); e != nil {
		t.Fatal(e)
	}
	upolyHenselAuditPhysical(t, cores, []*mpnum.UPolynomialFactors{pf, lf})
}

// Every live coefficient buffer must be reachable through a concrete source
// vector state; the storage registry is physical metadata, not a vector owner.
func upolyHenselCheckVectorOwners(cores []*mpnum.UPolynomialManager, factors []*mpnum.UPolynomialFactors, extras ...upolyPhysicalExtraOwners) error {
	var e error
	cores, factors, e = upolyPhysicalRetainedOwners(cores, factors, extras)
	if e != nil {
		return e
	}
	stores := map[uint64]reflect.Value{}
	for _, core := range cores {
		v := reflect.ValueOf(core).Elem()
		store := v.FieldByName("store").Elem()
		stores[store.FieldByName("origin").Uint()] = store
		bindings := v.FieldByName("bindings")
		for i := 0; i < bindings.Len(); i++ {
			store = bindings.Index(i).FieldByName("store").Elem()
			stores[store.FieldByName("origin").Uint()] = store
		}
	}
	owners := map[[3]uint64]string{}
	add := func(core reflect.Value, state reflect.Value, label string) error {
		backing := state.FieldByName("backing").Uint()
		if backing == 0 {
			return nil
		}
		origin := state.FieldByName("origin").Uint()
		if origin == 0 {
			origin = core.FieldByName("store").Elem().FieldByName("origin").Uint()
		}
		key := [3]uint64{origin, backing, state.FieldByName("generation").Uint()}
		store, ok := stores[origin]
		if !ok {
			return fmt.Errorf("%s vector has no bound physical origin %v", label, key)
		}
		backings := store.FieldByName("backings")
		if backing > uint64(backings.Len()) {
			return fmt.Errorf("%s vector outside physical directory %v", label, key)
		}
		b := backings.Index(int(backing - 1))
		if !b.FieldByName("live").Bool() || b.FieldByName("generation").Uint() != key[2] {
			return fmt.Errorf("%s vector physical lifetime mismatch %v", label, key)
		}
		if previous, ok := owners[key]; ok {
			return fmt.Errorf("one buffer has two scalar vector owners %v: %s and %s", key, previous, label)
		}
		owners[key] = label
		if state.FieldByName("capacity").Uint() != uint64(b.FieldByName("values").Len()) || state.FieldByName("size").Uint() > state.FieldByName("capacity").Uint() {
			return fmt.Errorf("%s vector physical capacity mismatch %v", label, key)
		}
		return nil
	}
	for index, owner := range cores {
		core := reflect.ValueOf(owner).Elem()
		for _, field := range []string{"basic", "div1", "div2", "exactDiv", "gcd1", "gcd2", "cra", "sqf1", "sqf2", "pw", "db", "dbab1", "dbab2", "tr", "push"} {
			if e := add(core, core.FieldByName(field), fmt.Sprintf("core%d.%s", index, field)); e != nil {
				return e
			}
		}
		mgcd := core.FieldByName("mgcd")
		for i := 0; i < mgcd.Len(); i++ {
			if e := add(core, mgcd.Index(i), fmt.Sprintf("core%d.mgcd%d", index, i)); e != nil {
				return e
			}
		}
		pages := core.FieldByName("pages")
		for i := 0; i < pages.Len(); i++ {
			rows := pages.Index(i).Elem().FieldByName("rows")
			for j := 0; j < rows.Len(); j++ {
				row := rows.Index(j)
				if row.FieldByName("live").Bool() {
					if e := add(core, row.FieldByName("state"), fmt.Sprintf("core%d.polynomial%d", index, i*rows.Len()+j)); e != nil {
						return e
					}
				}
			}
		}
	}
	for index, owner := range factors {
		fs := reflect.ValueOf(owner).Elem()
		core := fs.FieldByName("manager").Elem()
		states := fs.FieldByName("factors")
		for i := uint64(0); i < fs.FieldByName("count").Uint(); i++ {
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
			scope := reflect.ValueOf(service).Elem().FieldByName("scope")
			if scope.IsNil() {
				continue
			}
			scope = scope.Elem()
			core := scope.FieldByName("manager").Elem()
			for _, field := range []string{"fpp", "trial", "quotient"} {
				if e := add(core, scope.FieldByName(field), "retained factorSquareFree."+field); e != nil {
					return e
				}
			}
		}
	}
	for origin, store := range stores {
		backings := store.FieldByName("backings")
		for i := 0; i < backings.Len(); i++ {
			b := backings.Index(i)
			if b.FieldByName("live").Bool() {
				key := [3]uint64{origin, uint64(i + 1), b.FieldByName("generation").Uint()}
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
func upolyPhysicalRetainedOwners(cores []*mpnum.UPolynomialManager, factors []*mpnum.UPolynomialFactors, extras []upolyPhysicalExtraOwners) ([]*mpnum.UPolynomialManager, []*mpnum.UPolynomialFactors, error) {
	cs := append([]*mpnum.UPolynomialManager(nil), cores...)
	fs := append([]*mpnum.UPolynomialFactors(nil), factors...)
	for _, extra := range extras {
		for _, service := range extra.Services {
			for i := uint32(0); i < 2; i++ {
				c, e := service.RetainedCleanupCore(i)
				if errors.Is(e, arena.ErrBounds) {
					break
				}
				if e != nil {
					return nil, nil, e
				}
				found := false
				for _, old := range cs {
					if old == c {
						found = true
					}
				}
				if !found {
					cs = append(cs, c)
				}
			}
			for i := uint32(0); i < 2; i++ {
				f, e := service.RetainedCleanupFactors(i)
				if errors.Is(e, arena.ErrBounds) {
					break
				}
				if e != nil {
					return nil, nil, e
				}
				found := false
				for _, old := range fs {
					if old == f {
						found = true
					}
				}
				if !found {
					fs = append(fs, f)
				}
			}
		}
	}
	return cs, fs, nil
}
