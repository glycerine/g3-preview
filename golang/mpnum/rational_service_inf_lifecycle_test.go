// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/rational.cpp infinitesimal initialization/finalization.
package mpnum

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestRationalServiceInfinitesimalHeaderOwnership(t *testing.T) {
	s, _ := serviceNew(t)
	header := reflect.TypeOf(s).Elem()
	for _, name := range []string{"infZero", "infOne", "infMinusOne", "infIntZero", "infIntOne", "infIntMinusOne"} {
		field, ok := header.FieldByName(name)
		if !ok {
			t.Fatalf("missing owner-local source constant %s", name)
		}
		if err := arena.ValidateRecordType(field.Type); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	stats, err := s.Stats()
	serviceMust(t, err)
	if stats.HeaderBytes != uint64(header.Size()) {
		t.Fatalf("header charge %d, actual %d", stats.HeaderBytes, header.Size())
	}
}

func TestRationalServiceInfinitesimalConstructorFailureRefunds(t *testing.T) {
	var successes int
	for prefix := int64(0); prefix < 80; prefix++ {
		b := arena.NewBudget(1 << 24)
		b.FailAfter(prefix)
		s, err := NewRationalService(serviceConfig(b))
		if err != nil {
			if !numericPhysicalReservationRefusal(err) {
				t.Fatalf("prefix %d: %v", prefix, err)
			}
			constructionFailureOwningCleanup(t, err, b)
			if s != nil || b.Used() != 0 {
				t.Fatalf("prefix %d: failed constructor retained owner=%p bytes=%d", prefix, s, b.Used())
			}
			continue
		}
		successes++
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("prefix %d: close retained %d", prefix, b.Used())
		}
	}
	if successes == 0 {
		t.Fatal("no successful initialization prefix")
	}
}

func TestRationalServiceInfinitesimalFinalizerTraversal(t *testing.T) {
	s, _ := serviceNew(t)
	power, err := s.PowerOfTwo(40)
	serviceMust(t, err)
	serviceMust(t, s.Del(&power))
	cacheLen, cacheRef := len(s.cache), s.cache[40].num.cell
	// Populate constants with separately owned big pairs so release order is
	// observable in the manager's real free-slot stack before owner destruction.
	components := []*Rational{
		&s.infZero.second, &s.infZero.first,
		&s.infOne.second, &s.infOne.first,
		&s.infMinusOne.second, &s.infMinusOne.first,
		&s.infIntZero.first, &s.infIntOne.first, &s.infIntMinusOne.first,
	}
	var refs []numericCellRef
	for _, component := range components {
		serviceMust(t, s.manager.SetString(component, "123456789012345678901234567891/99887766554433221101"))
		refs = append(refs, component.num.cell, component.den.cell)
	}
	cells, words := s.manager.z.cells.Stats().Live, s.manager.z.words.Stats().Live
	s.mu.Lock()
	s.destroyInfinitesimalConstants()
	s.mu.Unlock()
	// Audit before any formatting, caller cleanup, or manager Close.
	if got := s.manager.z.cells.Stats().Live; got != cells-uint32(len(refs)) {
		t.Fatalf("constant cells: %d, want %d", got, cells-uint32(len(refs)))
	}
	if got := s.manager.z.words.Stats().Live; got != words {
		t.Fatalf("retained actual formatter payloads: %d, want %d", got, words)
	}
	serviceAuditRationalRoots(t, s)
	if len(s.cache) != cacheLen || s.cache[40].num.cell != cacheRef {
		t.Fatal("infinitesimal hooks destroyed the later powers cache")
	}
	var replacements []OwnedRational
	for i := len(refs) - 1; i >= 0; i-- {
		value, err := s.NewValue()
		serviceMust(t, err)
		serviceMust(t, s.SetUint64(&value, 4294967295))
		if numericSlot(value.raw.num.cell) != numericSlot(refs[i]) {
			t.Fatalf("source hook traversal at %d: reused %v, want %v", i, value.raw.num.cell, refs[i])
		}
		replacements = append(replacements, value)
	}
	for i := range replacements {
		serviceMust(t, s.Del(&replacements[i]))
	}
}
