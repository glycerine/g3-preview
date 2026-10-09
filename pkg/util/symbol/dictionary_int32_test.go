package symbol

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSymbolDictionaryInt32LookupMutationAndReset(t *testing.T) {
	s, e := NewManager(Config{})
	dictionaryMust(t, e)
	defer s.Close()

	var d DictionaryInt32
	dictionaryMust(t, d.Init(s))
	defer d.Close()
	named, e := s.Intern("name")
	dictionaryMust(t, e)
	keys := []Symbol{{}, Numerical(0), Numerical(8), named}
	for i, k := range keys {
		dictionaryMust(t, d.Insert(k, int32(i)))
	}
	dictionaryMust(t, d.Insert(Symbol{}, 77))
	if v, ok, e := d.Find(Symbol{}); e != nil || !ok || v != 77 || d.Size() != 4 {
		t.Fatal(v, ok, e, d.Size())
	}
	dictionaryMust(t, d.Remove(Numerical(8)))
	if _, ok, e := d.Find(Numerical(8)); e != nil || ok {
		t.Fatal(ok, e)
	}
	dictionaryMust(t, d.Reset())
	if d.Size() != 0 {
		t.Fatal(d.Size())
	}
}

func TestSymbolDictionaryInt32RejectsClosedSymbolOwner(t *testing.T) {
	s, e := NewManager(Config{})
	dictionaryMust(t, e)
	key, e := s.Intern("borrowed")
	dictionaryMust(t, e)
	var d DictionaryInt32
	dictionaryMust(t, d.Init(s))
	dictionaryMust(t, d.Insert(key, 5))
	dictionaryMust(t, s.Close())
	if _, _, e = d.Find(key); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	d.Close()
	if e = d.Insert(Symbol{}, 1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func dictionaryMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
