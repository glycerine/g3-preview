//go:build g3debug

// Source: src/util/sexpr.cpp; src/util/mpq.h (debug ownership audit adaptation). Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"strings"
	"testing"
)

func sxFractionalBVSourceProfile(t *testing.T, m *SexprManager, h SexprHandle) {
	// Source Rem asserts integrality before computing the first digit.
	sxRender(t, m, h, "#x", mpnum.ErrRationalNotInteger)
}

func TestSexprDebugBVInvariantAndShortCircuit(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	for _, tc := range []struct {
		c     string
		base  int64
		valid bool
	}{{"0", 16, true}, {"15", 16, true}, {"16", 16, false}, {"-1", 16, false}, {"0", 2, true}, {"1", 2, true}, {"2", 2, false}} {
		c := sxNumber(t, q, tc.c)
		e := m.sexprBVDiagnostic(&c, tc.base)
		if tc.valid {
			sxMust(t, e)
		} else if !errors.Is(e, ErrSexprBVInvariant) {
			t.Fatal(tc, e)
		}
		sxMust(t, q.Del(&c))
	}
	negative := sxNumber(t, q, "-1/1684996666696914987166688442938726917102321526408785780068975640579")
	before, e := q.Stats()
	sxMust(t, e)
	m.config.Budget.FailAfter(0)
	e = m.sexprBVDiagnostic(&negative, 16)
	if !errors.Is(e, ErrSexprBVInvariant) {
		t.Fatal("lower inequality must short circuit", e)
	}
	after, err := q.Stats()
	sxMust(t, err)
	if before.LiveCells != after.LiveCells || before.LivePayloads != after.LivePayloads {
		t.Fatal(before, after)
	}
	m.config.Budget.FailAfter(-1)
	sxMust(t, q.Del(&negative))
}
func TestSexprDebugBVComparisonFailureRoots(t *testing.T) {
	failures := 0
	success := false
	for prefix := int64(0); prefix < 256; prefix++ {
		b := arena.NewBudget(1 << 26)
		m, _, q := sxOwner(t, b)
		text := "1" + strings.Repeat("0", 400)
		den := "3" + strings.Repeat("0", 399) + "1"
		c := sxNumber(t, q, text+"/"+den)
		var filler [16]mpnum.OwnedRational
		for i := range filler {
			filler[i], _ = q.NewValue()
			sxMust(t, q.Copy(&filler[i], &c))
		}
		b.FailAfter(prefix)
		e := m.sexprBVDiagnostic(&c, 16)
		sexprAuditNumericRoots(t, q, &c, filler[:])
		if e != nil {
			if !numericStorageReservationRefusal(e) {
				t.Fatal(e)
			}
			failures++
		} else {
			success = true
		}
		b.FailAfter(-1)
		for i := range filler {
			sxMust(t, q.Del(&filler[i]))
		}
		sxMust(t, q.Del(&c))
		sxMust(t, m.Close())
		sxMust(t, q.Close())
		sxMust(t, m.config.Symbols.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal(success, failures)
	}
}

// Enumerate known source roots, not live allocations. Reflection reads only
// scalar test diagnostics; production does not expose or box numeric records.
func sexprAuditNumericRoots(t *testing.T, q *mpnum.RationalService, value *mpnum.OwnedRational, filler []mpnum.OwnedRational) {
	t.Helper()
	service := reflect.ValueOf(q).Elem()
	manager := service.FieldByName("manager").Elem()
	integer := manager.FieldByName("z").Elem()
	failed := integer.FieldByName("failedRef").Uint()
	cells := make(map[uint64]bool)
	add := func(ref uint64) {
		if ref != 0 && ref != failed {
			cells[ref] = true
		}
	}
	var numeral func(reflect.Value)
	numeral = func(v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		if r := v.FieldByName("cell"); r.IsValid() {
			add(r.Uint())
			return
		}
		for i := 0; i < v.NumField(); i++ {
			numeral(v.Field(i))
		}
	}
	for _, name := range []string{"tmp", "arg"} {
		v := integer.FieldByName(name)
		for i := 0; i < v.Len(); i++ {
			add(v.Index(i).Uint())
		}
	}
	for _, name := range []string{"intMin", "two64"} {
		numeral(integer.FieldByName(name))
	}
	for _, name := range []string{"nTmp", "addTmp1", "addTmp2", "addMulTmp", "ltTmp1", "ltTmp2"} {
		numeral(manager.FieldByName(name))
	}
	arithmetic := manager.FieldByName("rawArithmetic")
	for _, local := range []struct{ active, field string }{
		{"powerActive", "power"}, {"addActive", "addOne"}, {"addActive", "addTwo"},
		{"normActive", "norm"}, {"compareActive", "compareOne"}, {"compareActive", "compareTwo"},
		{"divActive", "div"},
	} {
		if arithmetic.FieldByName(local.active).Bool() {
			numeral(arithmetic.FieldByName(local.field))
		}
	}
	for _, name := range []string{"zero", "one", "minusOne", "infZero", "infOne", "infMinusOne", "infIntZero", "infIntOne", "infIntMinusOne"} {
		numeral(service.FieldByName(name))
	}
	cache := service.FieldByName("cache")
	for i := 0; i < cache.Len(); i++ {
		numeral(cache.Index(i))
	}
	public := func(v *mpnum.OwnedRational) {
		if _, e := q.IsNeg(v); e != nil {
			t.Fatal("invalid public comparison root", e)
		}
		numeral(reflect.ValueOf(v).Elem())
	}
	public(value)
	for i := range filler {
		public(&filler[i])
	}
	natural := integer.FieldByName("natural").Elem()
	payloads := uint32(len(cells))
	for _, name := range []string{"divScratch", "formatScratch"} {
		refs := natural.FieldByName(name)
		for i := 0; i < refs.Len(); i++ {
			if refs.Index(i).FieldByName("Slot").Uint() != 0 {
				payloads++
			}
		}
	}
	// Division overflow owns actual Runtime blocks rather than Words vectors.
	// Enumerate only the four source division buffers and active debug verifier.
	buffers := natural.FieldByName("division")
	raw := make(map[[3]uint64]bool)
	addRaw := func(buffer reflect.Value) {
		for _, name := range []string{"raw", "pending"} {
			block := buffer.FieldByName(name)
			key := [3]uint64{block.FieldByName("origin").Uint(), block.FieldByName("slot").Uint(), block.FieldByName("generation").Uint()}
			if key == [3]uint64{} {
				continue
			}
			if raw[key] {
				t.Fatal("duplicate Natural physical root", key)
			}
			raw[key] = true
		}
	}
	for i := 0; i < buffers.Len(); i++ {
		addRaw(buffers.Index(i))
	}
	if natural.FieldByName("verificationActive").Bool() {
		addRaw(natural.FieldByName("verification"))
	}
	payloads += uint32(len(raw))
	pending := integer.FieldByName("pendingCell").FieldByName("object")
	if pending.FieldByName("owner").Uint() != 0 {
		payloads++
	}
	stats, e := q.Stats()
	sxMust(t, e)
	if stats.LiveCells != uint32(len(cells)) || stats.LivePayloads != payloads {
		t.Fatal("unnamed debug comparison root", stats, len(cells), payloads)
	}
}
