package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSymbolDictionaryInt32LookupMutationAndReset(t *testing.T) {
	s, e := NewSymbolManager(SymbolConfig{})
	paramsMust(t, e)
	defer s.Close()

	var d SymbolDictionaryInt32
	paramsMust(t, d.Init(s))
	defer d.Close()
	named, e := s.Intern("name")
	paramsMust(t, e)
	keys := []Symbol{{}, NumericalSymbol(0), NumericalSymbol(8), named}
	for i, k := range keys {
		paramsMust(t, d.Insert(k, int32(i)))
	}
	paramsMust(t, d.Insert(Symbol{}, 77))
	if v, ok, e := d.Find(Symbol{}); e != nil || !ok || v != 77 || d.Size() != 4 {
		t.Fatal(v, ok, e, d.Size())
	}
	paramsMust(t, d.Remove(NumericalSymbol(8)))
	if _, ok, e := d.Find(NumericalSymbol(8)); e != nil || ok {
		t.Fatal(ok, e)
	}
	paramsMust(t, d.Reset())
	if d.Size() != 0 {
		t.Fatal(d.Size())
	}
}

func TestSymbolDictionaryInt32RejectsClosedSymbolOwner(t *testing.T) {
	s, e := NewSymbolManager(SymbolConfig{})
	paramsMust(t, e)
	key, e := s.Intern("borrowed")
	paramsMust(t, e)
	var d SymbolDictionaryInt32
	paramsMust(t, d.Init(s))
	paramsMust(t, d.Insert(key, 5))
	paramsMust(t, s.Close())
	if _, _, e = d.Find(key); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	d.Close()
	if e = d.Insert(Symbol{}, 1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
