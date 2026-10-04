// Licensed under the MIT license.
package g3

import (
	"errors"
	"testing"
)

func TestASTDependencyArrayPersistentCallbacksAndSourceTrails(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1800)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	var a, b ASTDependencyArray
	astTestMust(t, m.DependencyArrayMk(&a))
	for i := 0; i < 40; i++ {
		astTestMust(t, m.DependencyArrayPush(&a, d))
	}
	astTestMust(t, m.DependencyArrayCopy(&a, &b))
	for i := uint32(0); i < 40; i++ {
		astTestMust(t, m.DependencyArraySet(&a, i, ASTDependency{}))
	}
	for i := uint32(0); i < 40; i++ {
		x, e := m.DependencyArrayGet(&a, i)
		astTestMust(t, e)
		if !x.IsNull() {
			t.Fatal("updated version", i, x)
		}
		x, e = m.DependencyArrayGet(&b, i)
		astTestMust(t, e)
		if x != d {
			t.Fatal("persistent source version", i, x)
		}
	}
	astTestMust(t, m.DependencyArrayUnfold(&a))
	astTestMust(t, m.DependencyArrayUnshare(&b))
	astTestMust(t, m.DependencyArrayReroot(&b))
	astTestMust(t, m.DependencyArrayDel(&a))
	astTestMust(t, m.DependencyArrayDel(&b))
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal("array real dependency destruction", rc, e)
	}
}

func TestASTDependencyArrayRetainedSourceSlots(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1801)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d)) // keep the retained scalar alive independently.
	var a ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, d))
	astTestMust(t, m.DependencyArrayPush(&a, ASTDependency{}))
	astTestMust(t, m.DependencyArrayPop(&a))
	x, e := m.DependencyArrayGet(&a, 1)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
		if !x.IsNull() {
			t.Fatal(x)
		}
	}
	astTestMust(t, m.DependencyArrayPop(&a))
	x, e = m.DependencyArrayGet(&a, 0)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
		if x != d {
			t.Fatal(x)
		}
	}
	astTestMust(t, m.DependencyArrayDel(&a))
	astTestMust(t, m.DependencyDec(d))
}

func TestASTDependencyArrayFromOutputAliases(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 1802)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	var a, b ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, d))
	astTestMust(t, m.DependencyArrayPushFrom(&a, ASTDependency{}, &b))
	astTestMust(t, m.DependencyArraySetFrom(&b, 0, ASTDependency{}, &b))
	astTestMust(t, m.DependencyArrayPopFrom(&b, &b))
	if n, e := m.DependencyArraySize(&a); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if n, e := m.DependencyArraySize(&b); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	astTestMust(t, m.DependencyArrayDel(&b))
	astTestMust(t, m.DependencyArrayDel(&a))
	if rc, e := m.GetRefCount(h); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
}

// Source tst1<true>, tst2<true>, tst3<true> action/condition ports adapted from
// int/ref_count=false/max_trail8 to actual dependency/ref_count=true/trail16.
// This is not a port of the original int specialization or its false mode.
func TestASTDependencyArrayAdaptedRegisteredParrayAssertions(t *testing.T) {
	m := astTestManager(t)
	values := map[int]ASTDependency{}
	val := func(i int) ASTDependency {
		if d, ok := values[i]; ok {
			return d
		}
		h := astDependencyExpr(t, m, uint32(3000+i))
		d, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		astTestMust(t, m.DependencyInc(d))
		values[i] = d
		return d
	}
	count := 0
	ensure := func(b bool) {
		t.Helper()
		count++
		if !b {
			t.Fatal("adapted source assertion", count)
		}
	}
	size := func(a *ASTDependencyArray) uint32 { n, e := m.DependencyArraySize(a); astTestMust(t, e); return n }
	get := func(a *ASTDependencyArray, i uint32) ASTDependency {
		d, e := m.DependencyArrayGet(a, i)
		astTestMust(t, e)
		return d
	}
	var a1, a2, a3, a4 ASTDependencyArray
	astTestMust(t, m.DependencyArrayMk(&a1))
	ensure(size(&a1) == 0)
	astTestMust(t, m.DependencyArrayPushFrom(&a1, val(10), &a2))
	ensure(size(&a1) == 0)
	ensure(size(&a2) == 1)
	astTestMust(t, m.DependencyArrayPushFrom(&a1, val(20), &a1))
	astTestMust(t, m.DependencyArrayPushFrom(&a1, val(30), &a1))
	ensure(get(&a1, 0) == val(20))
	ensure(get(&a1, 1) == val(30))
	ensure(get(&a2, 0) == val(10))
	ensure(size(&a1) == 2)
	ensure(size(&a2) == 1)
	ensure(size(&a3) == 0)
	astTestMust(t, m.DependencyArrayPushFrom(&a2, val(100), &a3))
	ensure(size(&a3) == 2)
	ensure(get(&a3, 0) == val(10))
	ensure(get(&a3, 1) == val(100))
	astTestMust(t, m.DependencyArrayPush(&a2, val(50)))
	ensure(get(&a2, 0) == val(10))
	ensure(get(&a2, 1) == val(50))
	ensure(size(&a2) == 2)
	for _, a := range []*ASTDependencyArray{&a1, &a2, &a3} {
		astTestMust(t, m.DependencyArrayDel(a))
	}
	ensureCount1 := count
	for i := 0; i < 100; i++ {
		astTestMust(t, m.DependencyArrayPush(&a1, val(i)))
	}
	ensure(size(&a1) == 100)
	astTestMust(t, m.DependencyArrayPushFrom(&a1, val(100), &a2))
	for i := 0; i < 10; i++ {
		astTestMust(t, m.DependencyArrayPush(&a2, val(i+101)))
	}
	ensure(get(&a1, 0) == val(0))
	for i := uint32(0); i < size(&a1); i++ {
		ensure(get(&a1, i) == val(int(i)))
	}
	for i := uint32(0); i < size(&a2); i++ {
		ensure(get(&a2, i) == val(int(i)))
	}
	astTestMust(t, m.DependencyArrayUnshare(&a1))
	astTestMust(t, m.DependencyArrayDel(&a1))
	astTestMust(t, m.DependencyArrayDel(&a2))
	ensureCount2 := count - ensureCount1
	for i := 0; i < 20; i++ {
		astTestMust(t, m.DependencyArrayPush(&a1, val(i)))
	}
	ensure(size(&a1) == 20)
	astTestMust(t, m.DependencyArraySetFrom(&a1, 0, val(1), &a2))
	for i := 1; i < 20; i++ {
		if i == 6 {
			astTestMust(t, m.DependencyArrayCopy(&a2, &a3))
			astTestMust(t, m.DependencyArrayPop(&a3))
			astTestMust(t, m.DependencyArrayPop(&a3))
			astTestMust(t, m.DependencyArrayPush(&a3, val(40)))
		}
		astTestMust(t, m.DependencyArraySet(&a2, uint32(i), val(i+1)))
	}
	astTestMust(t, m.DependencyArrayPopFrom(&a2, &a4))
	astTestMust(t, m.DependencyArrayPop(&a4))
	astTestMust(t, m.DependencyArrayPush(&a4, val(30)))
	for i := uint32(0); i < 20; i++ {
		ensure(get(&a2, i) == val(int(i)+1))
	}
	ensure(get(&a1, 10) == val(10))
	ensure(size(&a1) == 20)
	ensure(size(&a2) == 20)
	ensure(size(&a3) == 19)
	ensure(size(&a4) == 19)
	for i := uint32(0); i < 20; i++ {
		ensure(get(&a1, i) == val(int(i)))
		ensure(get(&a2, i) == val(int(i)+1))
		ensure(i >= 18 || get(&a4, i) == val(int(i)+1))
		ensure(i >= 6 || get(&a3, i) == val(int(i)+1))
		ensure(!(6 <= i && i <= 17) || get(&a3, i) == val(int(i)))
	}
	ensure(get(&a4, 18) == val(30))
	ensure(get(&a3, 18) == val(40))
	astTestMust(t, m.DependencyArrayUnshare(&a1))
	astTestMust(t, m.DependencyArrayReroot(&a4))
	astTestMust(t, m.DependencyArrayUnshare(&a2))
	for _, a := range []*ASTDependencyArray{&a1, &a2, &a3, &a4} {
		astTestMust(t, m.DependencyArrayDel(a))
	}
	if ensureCount1 != 15 || ensureCount2 != 213 || count-ensureCount1-ensureCount2 != 128 {
		t.Fatal("source action assertion inventory", ensureCount1, ensureCount2, count-ensureCount1-ensureCount2)
	}
	for _, d := range values {
		astTestMust(t, m.DependencyDec(d))
	}
	if m.dependencies.live != 0 || m.dependencyArrays.liveCells != 0 || m.dependencyArrays.liveBlocks != 0 {
		t.Fatal("adapted callback teardown", m.dependencies.live, m.dependencyArrays.liveCells, m.dependencyArrays.liveBlocks)
	}
}

func TestASTDependencyArrayReleaseEmptyNonrootPopFinitePrefix(t *testing.T) {
	m := astTestManager(t)
	h := astDependencyExpr(t, m, 2500)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	var a, b ASTDependencyArray
	astTestMust(t, m.DependencyArrayPush(&a, d))
	astTestMust(t, m.DependencyArrayCopy(&a, &b))
	astTestMust(t, m.DependencyArrayPop(&a)) // retain shared new empty ROOT.
	astTestMust(t, m.DependencyArrayPop(&b)) // nonroot POP(size=1) => empty.
	e = m.DependencyArrayPop(&b)
	if astSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source empty assertion", e)
		}
	} else {
		astTestMust(t, e)
		if n, e := m.DependencyArraySize(&b); e != nil || n != ^uint32(0) {
			t.Fatal("finite nonroot unsigned source underflow", n, e)
		}
	}
	astTestMust(t, m.DependencyArrayDel(&a))
	astTestMust(t, m.DependencyArrayDel(&b))
}
