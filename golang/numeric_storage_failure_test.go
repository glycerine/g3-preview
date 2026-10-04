// Source: src/util/mpq.h; src/util/rational.h (ownership audit adaptation). Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

// Cold Go metadata reservations retain their arena category. An actual source
// malloc refusal goes through memory_manager.cpp's out_of_memory_error(101).
func numericStorageReservationRefusal(err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var physical memory.Failure
	return errors.As(err, &physical) && physical.Kind == memory.FailureMemout && physical.ErrorCode() == 101
}

// Enumerate named persistent scratch roots, so source free-before-allocation
// failures may remove a cache root without hiding an unowned temporary cell.
func numericServiceScratchRoots(t *testing.T, q *mpnum.RationalService) (uint32, uint32) {
	t.Helper()
	service := reflect.ValueOf(q).Elem()
	manager := service.FieldByName("manager").Elem()
	z := manager.FieldByName("z").Elem()
	failed := z.FieldByName("failedRef").Uint()
	refs := make(map[uint64]bool)
	var number func(reflect.Value)
	number = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		if c := v.FieldByName("cell"); c.IsValid() {
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
		a := z.FieldByName(name)
		for i := 0; i < a.Len(); i++ {
			r := a.Index(i).Uint()
			if r != 0 && r != failed {
				refs[r] = true
			}
		}
	}
	for _, name := range []string{"intMin", "two64"} {
		number(z.FieldByName(name))
	}
	for _, name := range []string{"nTmp", "addTmp1", "addTmp2", "addMulTmp", "ltTmp1", "ltTmp2"} {
		number(manager.FieldByName(name))
	}
	arithmetic := manager.FieldByName("rawArithmetic")
	for _, local := range []struct{ active, field string }{
		{"powerActive", "power"}, {"addActive", "addOne"}, {"addActive", "addTwo"},
		{"normActive", "norm"}, {"compareActive", "compareOne"}, {"compareActive", "compareTwo"},
		{"divActive", "div"},
	} {
		if arithmetic.FieldByName(local.active).Bool() {
			number(arithmetic.FieldByName(local.field))
		}
	}
	for _, name := range []string{"zero", "one", "minusOne", "infZero", "infOne", "infMinusOne", "infIntZero", "infIntOne", "infIntMinusOne"} {
		number(service.FieldByName(name))
	}
	cache := service.FieldByName("cache")
	for i := 0; i < cache.Len(); i++ {
		number(cache.Index(i))
	}
	cells := z.FieldByName("cells").Elem()
	pageSize := cells.FieldByName("config").FieldByName("PageSize").Uint()
	for ref := range refs {
		slot := uint64(uint32(ref))
		gen := ref >> 32
		if slot == 0 || slot > cells.FieldByName("slots").Uint() {
			t.Fatal("invalid named scratch slot", ref)
		}
		page := cells.FieldByName("pages").Index(int((slot - 1) / pageSize))
		meta := page.FieldByName("meta").Index(int((slot - 1) % pageSize))
		if !meta.FieldByName("live").Bool() || (z.FieldByName("failedRef").Type().Size() == 8 && meta.FieldByName("generation").Uint() != gen) {
			t.Fatal("invalid named scratch generation", ref)
		}
	}
	natural := z.FieldByName("natural").Elem()
	words := make(map[[2]uint64]bool)
	for _, name := range []string{"divScratch", "formatScratch"} {
		a := natural.FieldByName(name)
		for i := 0; i < a.Len(); i++ {
			r := a.Index(i)
			slot := r.FieldByName("Slot").Uint()
			if slot != 0 {
				words[[2]uint64{slot, r.FieldByName("Generation").Uint()}] = true
			}
		}
	}
	return uint32(len(refs)), uint32(len(words))
}
