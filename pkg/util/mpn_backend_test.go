// Licensed under the MIT license.
// Real mpn.h member sbuffers and mpn.cpp division storage action ordering.
package util

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestNaturalMandatoryInlineConstructorAndRuntime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	runtime := n.RuntimeOwner()
	if runtime == nil || runtime.Stats().LiveBlocks != 0 || n.words.Stats().Live != 0 {
		t.Fatal("inline construction allocated division payload")
	}
	for i := range n.Division {
		s := &n.Division[i]
		if s.capacity() != 16 || s.size() != 0 || s.Raw != (Block{}) || s.Pending != (Block{}) || n.divScratch[i] != (arena.VectorRef{}) {
			t.Fatal("source inline constructor", i, s)
		}
	}
	if e = arena.ValidateRecordType(reflect.TypeOf(n.Division[0])); e != nil {
		t.Fatal(e)
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if runtime.Stats().TotalBytes != 0 || b.Used() != 0 {
		t.Fatal("actual owned runtime/header leak")
	}
}

func TestNaturalMandatoryIntegerBorrowsActualRuntime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	z, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	if z.Natural.RuntimeOwner() != z.RuntimeOwner() {
		t.Fatal("disconnected numeric Natural runtime")
	}
	if z.Words.Stats().Live != 0 {
		t.Fatal("Natural inline storage still uses Words records")
	}
	if e = z.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("shared child ownership leak")
	}
}

func TestNaturalMandatoryActualOverflowAndShrink(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	r := n.RuntimeOwner()
	if e = n.ResizeDiv(0, 17); e != nil {
		t.Fatal(e)
	}
	s := &n.Division[0]
	if s.capacity() != 32 || s.size() != 17 || r.Stats().LiveBlocks != 1 || r.Stats().PhysicalBytes != 128+uint64(r.SizeTBits()/8) {
		t.Fatal("source 16->32 actual backing", s, r.Stats())
	}
	for i := uint32(0); i < 17; i++ {
		if v, e := s.read(r, i); e != nil || v != 0 {
			t.Fatal("source resize initializes only pushed elements", i, v, e)
		}
	}
	raw := s.Raw
	before := r.Stats()
	charge := b.Used()
	if e = n.ResizeDiv(0, 4); e != nil {
		t.Fatal(e)
	}
	if s.Raw != raw || s.capacity() != 32 || s.size() != 4 || r.Stats() != before || b.Used() != charge {
		t.Fatal("shrink changed actual source backing")
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("overflow destructor leak")
	}
}
