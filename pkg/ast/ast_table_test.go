package ast

import (
	"testing"
)

func TestASTArrayHashPermanentActualSourceVectors(t *testing.T) {
	// Actual ast.cpp::ast_array_hash source text compiled unchanged with hash.h.
	// H::hash is a read-only scalar getter; inputs 0x80000001+i*1234567, seed17.
	// Source capture .cache/ast_hash_reference.cpp; the test has no C++ dependency.
	expected := [11]uint32{17, 2164966943, 414713083, 3486530243, 1763846523, 656409297, 19394886, 2448893099, 994928918, 2540800055, 1037887258}
	for n, want := range expected {
		got := astHashArray(uint32(n), 17, func(i uint32) uint32 { return 0x80000001 + i*1234567 })
		if got != want {
			t.Fatal("actual source forward small/backward mixed triples", n, got, want)
		}
	}
}
