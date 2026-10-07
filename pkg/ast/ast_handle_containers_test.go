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
	for i, capacity := range []int{2, 2, 3, 5, 5, 8} {
		astTestMust(t, v.Push(b, ASTHandle{slot: uint32(i)}))
		if cap(v.Slice()) != capacity || b.Used() != uint64(capacity)*16 {
			t.Fatal("source capacity", i, cap(v.Slice()), b.Used())
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

func TestASTHandleVectorGrowthPeakAndFailurePublication(t *testing.T) {
	// Capacity 2 charges 32 bytes. Growing to 3 needs 48 additional bytes
	// while the old 32-byte backing remains live.
	b := arena.NewBudget(64)
	var v ASTHandleVector
	defer v.Close(b)
	first, second := ASTHandle{slot: 1}, ASTHandle{slot: 2}
	astTestMust(t, v.Push(b, first))
	astTestMust(t, v.Push(b, second))
	if err := v.Push(b, ASTHandle{slot: 3}); err != arena.ErrBudget {
		t.Fatal("growth must charge the old and new backing peak", err)
	}
	if cap(v.Slice()) != 2 || len(v.Slice()) != 2 || v.Slice()[0] != first || v.Slice()[1] != second || b.Used() != 32 {
		t.Fatal("failed growth changed values, capacity, or charge")
	}
	v.Erase(first)
	if len(v.Slice()) != 1 || v.Slice()[0] != second || v.Slice()[:cap(v.Slice())][1] != (ASTHandle{}) {
		t.Fatal("erase must preserve order and clear the vacated slot")
	}
	v.Close(b)
	if b.Used() != 0 {
		t.Fatal("close did not release the static element charge", b.Used())
	}
}
