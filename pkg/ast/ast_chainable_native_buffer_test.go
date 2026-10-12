// Source: src/ast/ast.cpp:2194-2201; src/util/buffer.h.
package ast

import (
	"fmt"
	"testing"
)

func TestASTChainableNativePairBufferOrderAndGrowth(t *testing.T) {
	m := astTestManager(t)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, boolSort)
	args := make([]ASTHandle, 20)
	for i := range args {
		args[i], e = m.MkConst(astTestSymbol(t, m, fmt.Sprintf("chain-%d", i)), boolSort)
		astTestMust(t, e)
		astTestPin(t, m, args[i])
	}
	pair, e := m.MkIff(args[0], args[1])
	astTestMust(t, e)
	astTestPin(t, m, pair)
	decl, e := m.GetDecl(pair)
	astTestMust(t, e)
	for _, count := range []int{3, 17, 20} {
		result, err := m.MkApp(decl, args[:count])
		astTestMust(t, err)
		astTestPin(t, m, result)
		gotCount, err := m.GetNumArgs(result)
		astTestMust(t, err)
		if gotCount != uint32(count-1) {
			t.Fatalf("%d arguments: got %d chain links", count, gotCount)
		}
		for i := 0; i < count-1; i++ {
			want, err := m.MkIff(args[i], args[i+1])
			astTestMust(t, err)
			got, err := m.GetArg(result, uint32(i))
			astTestMust(t, err)
			if got != want {
				t.Fatalf("%d arguments: pair %d = %v, want %v", count, i, got, want)
			}
		}
	}
}
