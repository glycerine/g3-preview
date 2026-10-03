// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Active src/test/ast.cpp assertions and src/test/parray.cpp::tst5.
package g3

import (
	"github.com/glycerine/g3/golang/utilhash"
	"testing"
)

func TestASTSourceNineAssertions(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, b)
	a, e := m.MkConst(astTestSymbol(t, m, "a"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	c, e := m.MkConst(astTestSymbol(t, m, "c"), b)
	astTestMust(t, e)
	astTestPin(t, m, c)
	i1, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpAnd, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i1)
	i2, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpAnd, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i2)
	i3, e := m.MkFamilyApp(m.BasicFamilyID(), ASTOpOr, nil, []ASTHandle{a, c})
	astTestMust(t, e)
	astTestPin(t, m, i3)
	if i1 != i2 {
		t.Fatal("source tst1: repeated AND must intern")
	}
	if i1 == i3 {
		t.Fatal("source tst1: AND and OR differ")
	}
	a1, e := m.MkConst(astTestSymbol(t, m, "a1"), b)
	astTestMust(t, e)
	astTestPin(t, m, a1)
	a2, e := m.MkConst(astTestSymbol(t, m, "a2"), b)
	astTestMust(t, e)
	astTestPin(t, m, a2)
	var arr1, arr2, arr3 ASTExprArray
	astTestMust(t, m.ExprArrayPush(&arr1, a1))
	astTestMust(t, m.ExprArrayPush(&arr1, a2))
	astTestMust(t, m.ExprArrayPopFrom(&arr1, &arr2))
	astTestMust(t, m.ExprArraySetFrom(&arr2, 0, a2, &arr3))
	n, e := m.ExprArraySize(&arr1)
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("source arr1 size", n)
	}
	n, e = m.ExprArraySize(&arr2)
	astTestMust(t, e)
	if n != 1 {
		t.Fatal("source arr2 size", n)
	}
	n, e = m.ExprArraySize(&arr3)
	astTestMust(t, e)
	if n != 1 {
		t.Fatal("source arr3 size", n)
	}
	h, e := m.ExprArrayGet(&arr1, 0)
	astTestMust(t, e)
	if h != a1 {
		t.Fatal("source arr1[0]")
	}
	h, e = m.ExprArrayGet(&arr1, 1)
	astTestMust(t, e)
	if h != a2 {
		t.Fatal("source arr1[1]")
	}
	h, e = m.ExprArrayGet(&arr2, 0)
	astTestMust(t, e)
	if h != a1 {
		t.Fatal("source arr2[0]")
	}
	h, e = m.ExprArrayGet(&arr3, 0)
	astTestMust(t, e)
	if h != a2 {
		t.Fatal("source arr3[0]")
	}
	astTestMust(t, m.ExprArrayDel(&arr1))
	astTestMust(t, m.ExprArrayDel(&arr2))
	astTestMust(t, m.ExprArrayDel(&arr3))
}

func TestASTSourcePArrayMillionUpdates(t *testing.T) {
	// Active parray.cpp::tst5 has zero ENSURE sites. Preserve its exact count
	// and AST factories; freeze a platform-independent test stream instead of
	// claiming libc rand's implementation. Independent checks add real roots.
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	var a1, a2 ASTExprArray
	astTestMust(t, m.ExprArrayMk(&a1))
	var original [100]ASTHandle
	for i := uint32(0); i < 100; i++ {
		h, e := m.MkVar(i, b)
		astTestMust(t, e)
		original[i] = h
		astTestMust(t, m.ExprArrayPush(&a1, h))
	}
	astTestMust(t, m.ExprArrayCopy(&a1, &a2))
	var expected [100]uint32
	var changed [100]bool
	rng := utilhash.NewRandom(1) // Explicit portable replacement for external libc rand input only.
	var plateau ASTManagerStats
	for i := uint32(0); i < 1000000; i++ {
		v := rng.Bounded(100)
		h, e := m.MkVar(v, b)
		astTestMust(t, e)
		if i >= 100 && expected[i%100] != v {
			changed[i%100] = true
		}
		astTestMust(t, m.ExprArraySet(&a1, i%100, h))
		expected[i%100] = v
		if i == 9999 {
			plateau = m.Stats()
		}
	}
	// preserve_roots admits update counters0..size before the strict >size
	// unshare test: at most size+1 inverse trails plus two array roots.
	beforeQueries := m.Stats()
	if beforeQueries.LiveNodes != plateau.LiveNodes || beforeQueries.AllocatedBytes != plateau.AllocatedBytes || beforeQueries.ArrayLiveCells > uint32(len(original))+3 {
		t.Fatalf("bounded million update owner graph grew: %#v -> %#v", plateau, beforeQueries)
	}
	for i := uint32(0); i < 100; i++ {
		if !changed[i] {
			t.Fatal("frozen workload never changed repeated site", i)
		}
		old, e := m.ExprArrayGet(&a2, i)
		astTestMust(t, e)
		if old != original[i] {
			t.Fatal("saved root mutated", i)
		}
		h, e := m.ExprArrayGet(&a1, i)
		astTestMust(t, e)
		n, e := m.GetVarIndex(h)
		astTestMust(t, e)
		if n != expected[i] {
			t.Fatal("updated root differs", i, n, expected[i])
		}
	}
	// Verifying the untouched saved version performs its first source reroot,
	// whose scratch growth is separate from the bounded update workload.
	astTestMust(t, m.ExprArrayDel(&a2))
	astTestMust(t, m.ExprArrayDel(&a1))
}
