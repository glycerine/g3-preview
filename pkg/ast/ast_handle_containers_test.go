// Source: src/util/buffer.h and vector.h.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestPointerBufferSourceSpillRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 16)
	var v ASTHandleBuffer
	for i := uint32(0); i < 16; i++ {
		astTestMust(t, v.Push(b, ASTHandle{slot: i}))
	}
	if b.Used() != 0 {
		t.Fatal("inline storage charged", b.Used())
	}
	b.FailAfter(0)
	if e := v.Push(b, ASTHandle{slot: 16}); e != arena.ErrInjectedFailure {
		t.Fatal("spill refusal", e)
	}
	if len(v.Slice()) != 16 || b.Used() != 0 {
		t.Fatal("failed spill changed storage")
	}
	b.FailAfter(-1)
	astTestMust(t, v.Push(b, ASTHandle{slot: 16}))
	if cap(v.Slice()) != 32 || b.Used() != 512 {
		t.Fatal("source doubled capacity", cap(v.Slice()), b.Used())
	}
	for i, n := range v.Slice() {
		if n != (ASTHandle{slot: uint32(i)}) {
			t.Fatal("lost value", i, n)
		}
	}
	v.Close(b)
	if b.Used() != 0 || len(v.Slice()) != 0 {
		t.Fatal("source destructor retained spill")
	}
}
func TestPointerVectorSourceGrowthAndErase(t *testing.T) {
	b := arena.NewBudget(1 << 16)
	var v ASTHandleVector
	for i := 0; i < 6; i++ {
		astTestMust(t, v.Push(b, ASTHandle{slot: uint32(i)}))
		if cap(v.Slice()) < len(v.Slice()) || b.Used() != 0 {
			t.Fatal("native slice capacity", i, cap(v.Slice()), b.Used())
		}
	}
	v.Erase(ASTHandle{slot: 2})
	expected := []uint32{0, 1, 3, 4, 5}
	for i, n := range v.Slice() {
		if n != (ASTHandle{slot: expected[i]}) {
			t.Fatal("source erase", i, n)
		}
	}
	v.Close(b)
	if b.Used() != 0 || len(v.Slice()) != 0 {
		t.Fatal("source destructor retained vector")
	}
}

func TestASTHandleVectorNativeGrowthAndEraseClearing(t *testing.T) {
	b := arena.NewBudget(1)
	var v ASTHandleVector
	defer v.Close(b)
	b.FailAfter(0)
	first, second := ASTHandle{slot: 1}, ASTHandle{slot: 2}
	astTestMust(t, v.Push(b, first))
	astTestMust(t, v.Push(b, second))
	astTestMust(t, v.Push(b, ASTHandle{slot: 3}))
	if len(v.Slice()) != 3 || v.Slice()[0] != first || v.Slice()[1] != second || b.Used() != 0 {
		t.Fatal("native growth changed values or numeric budget")
	}
	b.FailAfter(-1)
	v.Erase(first)
	if len(v.Slice()) != 2 || v.Slice()[0] != second || v.Slice()[:cap(v.Slice())][2] != (ASTHandle{}) {
		t.Fatal("erase must preserve order and clear the vacated slot")
	}
	v.Close(b)
	if b.Used() != 0 {
		t.Fatal("close did not release the static element charge", b.Used())
	}
}
