package util

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSymbolPackageBoundaryBorrowAndEncoding(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, e := usymbol.NewManager(usymbol.Config{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	s, e := m.InternBytes([]byte{'a', 'b', 0, 'c'})
	if e != nil {
		t.Fatal(e)
	}
	want := internSymbol(t, m, "ab")
	if s != want {
		t.Fatal("byte interner ignored CString terminator")
	}
	before := b.Used()
	view, e := SymbolCStringView(m, s)
	if e != nil {
		t.Fatal(e)
	}
	for i := uint32(0); i < 128; i++ {
		internSymbol(t, m, strconv.FormatUint(uint64(i), 10))
	}
	n, e := view.Length()
	if e != nil || n != 2 {
		t.Fatal(n, e)
	}
	for i, v := range []byte{'a', 'b', 0} {
		got, e := view.Byte(uint64(i))
		if e != nil || got != v {
			t.Fatal(i, got, e)
		}
	}
	decoded, e := m.DecodeSymbolWords(s.SymbolWords())
	if e != nil || decoded != s {
		t.Fatal(decoded, e)
	}
	if b.Used() != before {
		t.Fatal("borrow/encoding allocated source storage")
	}
	if _, e = SymbolCStringView(m, usymbol.Symbol{}); !errors.Is(e, usymbol.ErrInvalid) {
		t.Fatal(e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = view.Byte(0); e == nil {
		t.Fatal("borrow survived owner close")
	}
	if b.Used() != 0 {
		t.Fatal("symbol storage leaked", b.Used())
	}
}

func TestSymbolDictionaryPackageBoundaryGrowthAndCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, e := usymbol.NewManager(usymbol.Config{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	var d usymbol.DictionaryInt32
	if e = d.Init(m); e != nil {
		t.Fatal(e)
	}
	for i := uint32(0); i < 128; i++ {
		if e = d.Insert(usymbol.Numerical(i), int32(i*3)); e != nil {
			t.Fatal(e)
		}
	}
	for i := uint32(0); i < 128; i++ {
		v, ok, e := d.Find(usymbol.Numerical(i))
		if e != nil || !ok || v != int32(i*3) {
			t.Fatal(i, v, ok, e)
		}
	}
	if d.Size() != 128 {
		t.Fatal(d.Size())
	}
	if e = d.Remove(usymbol.Numerical(63)); e != nil {
		t.Fatal(e)
	}
	if _, ok, e := d.Find(usymbol.Numerical(63)); e != nil || ok {
		t.Fatal(ok, e)
	}
	d.Close()
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("dictionary storage leaked", b.Used())
	}
}
