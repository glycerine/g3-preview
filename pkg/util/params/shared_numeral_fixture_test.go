package params

// Source: src/util/mpq.h; src/util/rational.h (ownership audit adaptation).

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Cold Go metadata reservations retain their arena category. An actual source
// malloc refusal goes through memory_manager.cpp's out_of_memory_error(101).
func numericStorageReservationRefusal(err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var physical umemory.Failure
	return errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101
}

// Enumerate named persistent scratch roots, so source free-before-allocation
// failures may remove a cache root without hiding an unowned temporary cell.
func numericServiceScratchRoots(t *testing.T, q *unumeral.RationalService) (uint32, uint32) {
	t.Helper()
	service := reflect.ValueOf(q).Elem()
	manager := testFieldBySourceName(service, "manager").Elem()
	z := testFieldBySourceName(manager, "z").Elem()
	failed := testFieldBySourceName(z, "failedRef").Uint()
	refs := make(map[uint64]bool)
	var number func(reflect.Value)
	number = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		if c := testFieldBySourceName(v, "cell"); c.IsValid() {
			r := c.Uint()
			if r != 0 && r != failed {
				refs[r] = true
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			number(v.Field(i))
		}
	}
	for _, name := range []string{"tmp", "arg"} {
		a := testFieldBySourceName(z, name)
		for i := 0; i < a.Len(); i++ {
			r := a.Index(i).Uint()
			if r != 0 && r != failed {
				refs[r] = true
			}
		}
	}
	for _, name := range []string{"intMin", "two64"} {
		number(testFieldBySourceName(z, name))
	}
	for _, name := range []string{"nTmp", "addTmp1", "addTmp2", "addMulTmp", "ltTmp1", "ltTmp2"} {
		number(testFieldBySourceName(manager, name))
	}
	arithmetic := testFieldBySourceName(manager, "rawArithmetic")
	for _, local := range []struct{ active, field string }{
		{"powerActive", "power"}, {"addActive", "addOne"}, {"addActive", "addTwo"},
		{"normActive", "norm"}, {"compareActive", "compareOne"}, {"compareActive", "compareTwo"},
		{"divActive", "div"},
	} {
		if testFieldBySourceName(arithmetic, local.active).Bool() {
			number(testFieldBySourceName(arithmetic, local.field))
		}
	}
	for _, name := range []string{"zero", "one", "minusOne", "infZero", "infOne", "infMinusOne", "infIntZero", "infIntOne", "infIntMinusOne"} {
		number(testFieldBySourceName(service, name))
	}
	cache := testFieldBySourceName(service, "cache")
	for i := 0; i < cache.Len(); i++ {
		number(cache.Index(i))
	}
	cells := testFieldBySourceName(z, "cells").Elem()
	pageSize := testFieldBySourceName(testFieldBySourceName(cells, "config"), "PageSize").Uint()
	for ref := range refs {
		slot := uint64(uint32(ref))
		gen := ref >> 32
		if slot == 0 || slot > testFieldBySourceName(cells, "slots").Uint() {
			t.Fatal("invalid named scratch slot", ref)
		}
		page := testFieldBySourceName(cells, "pages").Index(int((slot - 1) / pageSize))
		meta := testFieldBySourceName(page, "meta").Index(int((slot - 1) % pageSize))
		if !testFieldBySourceName(meta, "live").Bool() || (testFieldBySourceName(z, "failedRef").Type().Size() == 8 && testFieldBySourceName(meta, "generation").Uint() != gen) {
			t.Fatal("invalid named scratch generation", ref)
		}
	}
	// No retained numeric scratch owns a Words descriptor.
	return uint32(len(refs)), 0
}
func serviceMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
