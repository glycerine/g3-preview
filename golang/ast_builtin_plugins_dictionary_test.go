// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestASTUserDictionaryResetShrinkRefusalKeepsCoherentClearedBacking(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	astTestMust(t, e)
	defer s.Close()
	var d astUserDictionary
	astTestMust(t, d.init(s, b, nil))
	defer d.close()
	astTestMust(t, d.replace(32, false))
	astTestMust(t, d.insert(NumericalSymbol(1), 9))
	before := b.Used()
	b.FailAfter(0)
	e = d.reset()
	b.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || len(d.entries) != 32 || b.Used() != before || d.size != 0 || d.deleted != 0 {
		t.Fatal("safe cleared-table continuation", e, len(d.entries), b.Used(), before, d.size, d.deleted)
	}
	for _, entry := range d.entries {
		if entry.state != 0 {
			t.Fatal("source completed FREE marking")
		}
	}
	astTestMust(t, d.insert(NumericalSymbol(2), 10))
	if d.size != 1 {
		t.Fatal("retry over cleared backing", d.size)
	}
	_, found, e := d.find(NumericalSymbol(1))
	astTestMust(t, e)
	if found {
		t.Fatal("old membership restored")
	}
}

func TestASTUserDictionaryActualMapNullAndCachedEntries(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var d astUserDictionary
	if e = d.init(s, b, nil); e != nil {
		t.Fatal(e)
	}
	defer d.close()
	named, e := s.Intern("name")
	if e != nil {
		t.Fatal(e)
	}
	keys := []Symbol{{}, NumericalSymbol(0), NumericalSymbol(8), named}
	for i, k := range keys {
		if e = d.insert(k, int32(i)); e != nil {
			t.Fatal(e)
		}
	}
	for i, k := range keys {
		v, ok, e := d.find(k)
		if e != nil || !ok || v != int32(i) {
			t.Fatal("actual raw key identity", i, v, ok, e)
		}
	}
	if e = d.insert(Symbol{}, 77); e != nil {
		t.Fatal(e)
	}
	v, ok, e := d.find(Symbol{})
	if e != nil || !ok || v != 77 || d.size != 4 {
		t.Fatal("null overwrite", v, ok, e, d.size)
	}
	if d.bytes != uint64(cap(d.entries))*uint64(reflect.TypeOf(astUserDictionaryEntry{}).Size()) {
		t.Fatal("actual target layout")
	}
}

func TestASTUserDictionaryLastDeletedAndRetainedScalarPayload(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var d astUserDictionary
	if e = d.init(s, b, nil); e != nil {
		t.Fatal(e)
	}
	defer d.close()
	for i := uint32(0); i < 3; i++ {
		if e = d.insert(NumericalSymbol(i*8), int32(i+1)); e != nil {
			t.Fatal(e)
		}
	}
	if e = d.remove(NumericalSymbol(0)); e != nil {
		t.Fatal(e)
	}
	if e = d.remove(NumericalSymbol(8)); e != nil {
		t.Fatal(e)
	}
	if d.deleted != 2 || d.entries[0].value != 1 || d.entries[1].value != 2 {
		t.Fatal("inactive payload retention", d)
	}
	if e = d.insert(NumericalSymbol(24), 9); e != nil {
		t.Fatal(e)
	}
	if d.entries[1].key != NumericalSymbol(24) || d.entries[0].state != 1 || d.deleted != 1 {
		t.Fatal("source uses last encountered deleted slot", d)
	}
	if e = d.remove(NumericalSymbol(16)); e != nil {
		t.Fatal(e)
	}
	if d.entries[2].state != 0 || d.entries[2].key != NumericalSymbol(16) || d.entries[2].value != 3 {
		t.Fatal("tail erase marks free without destroying scalar payload", d)
	}
}

func TestASTUserDictionaryPrepareBeforeOverwriteFailure(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var d astUserDictionary
	if e = d.init(s, b, nil); e != nil {
		t.Fatal(e)
	}
	defer d.close()
	for i := uint32(0); i < 7; i++ {
		if e = d.insert(NumericalSymbol(i), int32(i)); e != nil {
			t.Fatal(e)
		}
	}
	used := b.Used()
	b.FailAfter(0)
	if e = d.insert(NumericalSymbol(0), 99); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("source strict 75 percent grows before overwrite", e)
	}
	v, ok, e := d.find(NumericalSymbol(0))
	if e != nil || !ok || v != 0 || len(d.entries) != 8 || b.Used() != used {
		t.Fatal("reserve failure prefix", v, ok, e)
	}
	b.FailAfter(-1)
	if e = d.insert(NumericalSymbol(0), 99); e != nil {
		t.Fatal(e)
	}
	if len(d.entries) != 16 || d.size != 7 {
		t.Fatal("growth", d)
	}
}

func TestASTUserDictionaryResetRetainsAndRehashUsesCachedHash(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var d astUserDictionary
	if e = d.init(s, b, nil); e != nil {
		t.Fatal(e)
	}
	defer d.close()
	key, e := s.Intern("borrowed")
	if e != nil {
		t.Fatal(e)
	}
	if e = d.insert(key, 5); e != nil {
		t.Fatal(e)
	}
	// Table movement uses cached hashes and scalar assignment; it must not reread
	// a borrowed spelling, even after its owner is closed. Queries still reject it.
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = d.replace(16, true); e != nil {
		t.Fatal(e)
	}
	var occupied int
	for _, v := range d.entries {
		if v.state == 2 {
			occupied++
			if v.key != key || v.value != 5 {
				t.Fatal("cached move")
			}
		}
	}
	if occupied != 1 {
		t.Fatal(occupied)
	}
	if _, _, e = d.find(key); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("borrowed lifetime guard", e)
	}
	if e = d.reset(); e != nil {
		t.Fatal(e)
	}
	var retained bool
	for _, v := range d.entries {
		if v.key == key && v.value == 5 && v.state == 0 {
			retained = true
		}
	}
	if !retained || d.size != 0 || d.deleted != 0 {
		t.Fatal("reset scalar retention")
	}
	d.close()
	if b.Used() != 0 {
		t.Fatal("immediate complete owner teardown", b.Used())
	}
}
