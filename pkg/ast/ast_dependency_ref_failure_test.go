// Licensed under the MIT license.
package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTDependencyRefOwnersPointerFreeAndDistinctStorage(t *testing.T) {
	for _, ty := range []reflect.Type{reflect.TypeOf(ASTDependencyRef{}), reflect.TypeOf(ASTDependencyRefVector{}), reflect.TypeOf(ASTDependencyRefBuffer{}), reflect.TypeOf(ASTDependencyRefElement{}), reflect.TypeOf(ASTDependencyRefVectorIterator{}), reflect.TypeOf(ASTDependencyRefBufferBorrow{}), reflect.TypeOf(ASTDependencyRefBufferIterator{})} {
		astTestMust(t, arena.ValidateRecordType(ty))
	}
	m := astTestManager(t)
	base := m.allocator.Stats().Live
	r, e := m.NewDependencyRef(ASTDependency{})
	astTestMust(t, e)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	if m.allocator.Stats().Live != base+3 {
		t.Fatal("three actual cold records")
	}
	if n, e := m.allocator.Size(r.object); e != nil || n != 16 {
		t.Fatal(n, e)
	}
	if n, e := m.allocator.Size(b.buffer); e != nil || n != 160 {
		t.Fatal(n, e)
	}
	astTestMust(t, m.DelDependencyRef(&r))
	astTestMust(t, m.DelDependencyRefVector(&v))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	if m.allocator.Stats().Live != base {
		t.Fatal("cold owner graph before Close")
	}
}
func TestASTDependencyRefVectorAndBufferGrowthCopyDifferentTails(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorResize(v, 5))
	astTestMust(t, m.DependencyRefVectorShrink(v, 2))
	it, e := m.DependencyRefVectorBegin(v)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorReserve(v, 8))
	if _, _, e = m.DependencyRefVectorNext(&it); !errors.Is(e, ErrASTHandle) {
		t.Fatal("relocated pointer", e)
	}
	s, e := m.dependencyVectorState(v)
	astTestMust(t, e)
	if _, e = m.dependencyVectorPhysicalValue(s, 4); e != nil {
		t.Fatal("vector full-capacity initialized tail", e)
	}
	moved, e := m.MoveDependencyRefVector(v)
	astTestMust(t, e)
	it, e = m.DependencyRefVectorBegin(moved)
	astTestMust(t, e)
	astTestMust(t, m.SwapDependencyRefVectors(v, moved))
	if _, ok, e := m.DependencyRefVectorNext(&it); e != nil || !ok {
		t.Fatal("physical borrow survives transfer", ok, e)
	}
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferResize(b, 20))
	astTestMust(t, m.DependencyRefBufferShrink(b, 3))
	old, e := m.DependencyRefBufferElement(b, 2)
	astTestMust(t, e)
	bs, e := m.dependencyBufferState(b)
	astTestMust(t, e)
	astTestMust(t, m.dependencyGrowBuffer(b, &bs))
	if _, e = m.dependencyBufferPhysical(b, bs, 10); !errors.Is(e, ErrASTDomain) {
		t.Fatal("buffer logical-only expansion invented tail", e)
	}
	if _, e = m.DependencyRefBufferBorrowGet(old); !errors.Is(e, ErrASTHandle) {
		t.Fatal("freed dynamic borrow", e)
	}
	astTestMust(t, m.DelDependencyRefVector(&v))
	astTestMust(t, m.DelDependencyRefVector(&moved))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
}
func TestASTDependencyRefBufferCapturedDynamicSelfAppendPrefix(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7300))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	for i := 0; i < 20; i++ {
		astTestMust(t, m.DependencyRefBufferPush(b, d))
	}
	e = m.DependencyRefBufferAppend(b, b)
	if !errors.Is(e, ErrASTHandle) {
		t.Fatal("captured old dynamic pointer must be guarded", e)
	}
	// Entry12 is read while the old block lives, then Push grows and succeeds.
	// Entry13 is the first subsequent captured read into the freed old block.
	if n, e := m.DependencyRefBufferSize(b); e != nil || n != 33 {
		t.Fatal(n, e)
	}
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 34 {
		t.Fatal("completed source callback prefix", rc, e)
	}
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	astTestMust(t, m.DelDependencyRef(&pin))
}
func TestASTDependencyRefVectorVsBufferIncreasingShrinkSourceProfiles(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorResize(v, 3))
	astTestMust(t, m.DependencyRefVectorShrink(v, 1))
	astTestMust(t, m.DependencyRefBufferResize(b, 3))
	astTestMust(t, m.DependencyRefBufferShrink(b, 1))
	ev := m.DependencyRefVectorShrink(v, 2)
	eb := m.DependencyRefBufferShrink(b, 2)
	if ASTSourceDebug {
		if !errors.Is(ev, ErrASTDomain) || !errors.Is(eb, ErrASTDomain) {
			t.Fatal(ev, eb)
		}
	} else {
		astTestMust(t, ev)
		astTestMust(t, eb)
	}
	expected := uint32(2)
	if ASTSourceDebug {
		expected = 1
	}
	if n, e := m.DependencyRefVectorSize(v); e != nil || n != expected {
		t.Fatal(n, e)
	}
	if n, e := m.DependencyRefBufferSize(b); e != nil || n != expected {
		t.Fatal(n, e)
	}
	if !ASTSourceDebug {
		if v, e := m.DependencyRefBufferGet(b, 1); e != nil || !v.IsNull() {
			t.Fatal(v, e)
		}
	}
	astTestMust(t, m.DelDependencyRefVector(&v))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
}
func TestASTDependencyRefSourcePushRefusalRetainsIncrement(t *testing.T) {
	for _, buffer := range []bool{false, true} {
		m := astTestManager(t)
		d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7400))
		astTestMust(t, e)
		pin, e := m.NewDependencyRef(d)
		astTestMust(t, e)
		if buffer {
			b, e := m.NewDependencyRefBuffer()
			astTestMust(t, e)
			astTestMust(t, m.DependencyRefBufferResize(b, 16))
			m.BudgetField.FailAfter(0)
			e = m.DependencyRefBufferPush(b, d)
			m.BudgetField.FailAfter(-1)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if n, e := m.DependencyRefBufferSize(b); e != nil || n != 16 {
				t.Fatal(n, e)
			}
			astTestMust(t, m.DelDependencyRefBuffer(&b))
		} else {
			v, e := m.NewDependencyRefVector()
			astTestMust(t, e)
			m.BudgetField.FailAfter(0)
			e = m.DependencyRefVectorPush(v, d)
			m.BudgetField.FailAfter(-1)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if n, e := m.DependencyRefVectorSize(v); e != nil || n != 0 {
				t.Fatal(n, e)
			}
			astTestMust(t, m.DelDependencyRefVector(&v))
		}
		if rc, e := m.DependencyRefCount(d); e != nil || rc != 2 {
			t.Fatal("source retained incoming INC", rc, e)
		}
		astTestMust(t, m.DependencyDec(d))
		astTestMust(t, m.DelDependencyRef(&pin))
	}
}
func TestASTDependencyRefRawBufferBorrowDoesNotInventCallbacks(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7500))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferResize(b, 1))
	el, e := m.DependencyRefBufferElement(b, 0)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferBorrowWriteRaw(el, d))
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	// Restore null through the same raw T*& before ordinary owning destruction.
	astTestMust(t, m.DependencyRefBufferBorrowWriteRaw(el, ASTDependency{}))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	astTestMust(t, m.DelDependencyRef(&pin))
}
func TestASTDependencyRefBufferInlineBorrowSurvivesFinalizeAndGrowth(t *testing.T) {
	m := astTestManager(t)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferResize(b, 16))
	el, e := m.DependencyRefBufferElement(b, 15)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferResize(b, 17))
	if d, e := m.DependencyRefBufferBorrowGet(el); e != nil || !d.IsNull() {
		t.Fatal(d, e)
	}
	astTestMust(t, m.DependencyRefBufferFinalize(b))
	if d, e := m.DependencyRefBufferBorrowGet(el); e != nil || !d.IsNull() {
		t.Fatal(d, e)
	}
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	if _, e = m.DependencyRefBufferBorrowGet(el); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
}

func TestASTDependencyRefVectorDebugCounterRefusalKeepsOwnedBacking(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7600))
	astTestMust(t, e)
	c, e := m.dependencies.node(d.record)
	astTestMust(t, e)
	c.rc = astDependencyMaxRC
	astTestMust(t, m.dependencies.store(d.record, c))
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorPush(v, d))
	live := m.vectorAllocator.Stats().Live
	e = m.DependencyRefVectorFinalize(v)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	if n, e := m.DependencyRefVectorSize(v); e != nil || n != 1 {
		t.Fatal("source assertion cannot finalize still-owned references", n, e)
	}
	if got, e := m.DependencyRefVectorGet(v, 0); e != nil || got != d {
		t.Fatal(got, e)
	}
	if m.vectorAllocator.Stats().Live != live {
		t.Fatal("refused DEC lost physical backing")
	}
	c.rc = 1
	astTestMust(t, m.dependencies.store(d.record, c))
	astTestMust(t, m.DelDependencyRefVector(&v))
}
func TestASTDependencyRefVectorConsumedCallbackErrorReclaimsLexicalOwner(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	baseNodes := m.liveNodes
	var args [32]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(7700+i), sort)
		astTestMust(t, e)
	}
	h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
	astTestMust(t, e)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorPush(v, d))
	m.BudgetField.FailAfter(0)
	e = m.DelDependencyRefVector(&v)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if v != (ASTDependencyRefVector{}) || m.dependencies.live != 0 || m.liveNodes != baseNodes {
		t.Fatal("consumed callback lost lexical owner", v, m.dependencies.live, m.liveNodes, baseNodes)
	}
}

var astDependencyRefEscapedVectorIterator ASTDependencyRefVectorIterator
var astDependencyRefEscapedBufferIterator ASTDependencyRefBufferIterator

func TestASTDependencyRefWarmCallbacksAndEscapingBorrowValuesAllocateZero(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7800))
	astTestMust(t, e)
	r, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorPush(v, d))
	astTestMust(t, m.DependencyRefVectorPop(v))
	m.BudgetField.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := m.DependencyRefAssign(r, r); e != nil {
			panic(e)
		}
		if e := m.DependencyRefVectorPush(v, d); e != nil {
			panic(e)
		}
		var e error
		astDependencyRefEscapedVectorIterator, e = m.DependencyRefVectorBegin(v)
		if e != nil {
			panic(e)
		}
		if e = m.DependencyRefVectorPop(v); e != nil {
			panic(e)
		}
		if e = m.DependencyRefBufferPush(b, d); e != nil {
			panic(e)
		}
		astDependencyRefEscapedBufferIterator, e = m.DependencyRefBufferBegin(b)
		if e != nil {
			panic(e)
		}
		if e = m.DependencyRefBufferPop(b); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.DelDependencyRefVector(&v))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	astTestMust(t, m.DelDependencyRef(&r))
}
func TestASTDependencyRefFiniteIndependentContainerCounterOracle(t *testing.T) {
	for _, buffer := range []bool{false, true} {
		m := astTestManager(t)
		d, e := m.DependencyLeaf(astDependencyExpr(t, m, 7900))
		astTestMust(t, e)
		pin, e := m.NewDependencyRef(d)
		astTestMust(t, e)
		var v ASTDependencyRefVector
		var b ASTDependencyRefBuffer
		if buffer {
			b, e = m.NewDependencyRefBuffer()
		} else {
			v, e = m.NewDependencyRefVector()
		}
		astTestMust(t, e)
		var data [32]bool
		size := uint32(0)
		seed := uint32(17)
		for step := 0; step < 500; step++ {
			seed = seed*1664525 + 1013904223
			switch seed % 4 {
			case 0:
				if size < 24 {
					value := ASTDependency{}
					present := seed&(1<<12) != 0
					if present {
						value = d
					}
					if buffer {
						e = m.DependencyRefBufferPush(b, value)
					} else {
						e = m.DependencyRefVectorPush(v, value)
					}
					astTestMust(t, e)
					data[size] = present
					size++
				}
			case 1:
				if size > 0 {
					if buffer {
						e = m.DependencyRefBufferPop(b)
					} else {
						e = m.DependencyRefVectorPop(v)
					}
					astTestMust(t, e)
					size--
				}
			case 2:
				if size > 0 {
					i := (seed >> 8) % size
					value := ASTDependency{}
					present := seed&(1<<15) != 0
					if present {
						value = d
					}
					if buffer {
						e = m.DependencyRefBufferSet(b, i, value)
					} else {
						e = m.DependencyRefVectorSet(v, i, value)
					}
					astTestMust(t, e)
					data[i] = present
				}
			case 3:
				n := (seed >> 8) % 25
				if buffer {
					e = m.DependencyRefBufferResize(b, n)
				} else {
					e = m.DependencyRefVectorResize(v, n)
				}
				astTestMust(t, e)
				for i := size; i < n; i++ {
					data[i] = false
				}
				size = n
			}
			count := uint32(1)
			for i := uint32(0); i < size; i++ {
				var got ASTDependency
				if buffer {
					got, e = m.DependencyRefBufferGet(b, i)
				} else {
					got, e = m.DependencyRefVectorGet(v, i)
				}
				astTestMust(t, e)
				want := ASTDependency{}
				if data[i] {
					want = d
					count++
				}
				if got != want {
					t.Fatal(buffer, step, i, got, want)
				}
			}
			if rc, e := m.DependencyRefCount(d); e != nil || rc != count {
				t.Fatal("independent owning multiplicity", buffer, step, rc, count, e)
			}
		}
		if buffer {
			astTestMust(t, m.DelDependencyRefBuffer(&b))
		} else {
			astTestMust(t, m.DelDependencyRefVector(&v))
		}
		if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
			t.Fatal(rc, e)
		}
		astTestMust(t, m.DelDependencyRef(&pin))
	}
}
func TestASTDependencyRefVectorConstructorFaultSweepImmediateOwnership(t *testing.T) {
	var values [80]ASTDependency
	sawFailure, sawSuccess := false, false
	for fail := int64(0); fail < 100; fail++ {
		m := astTestManager(t)
		d, e := m.DependencyLeaf(astDependencyExpr(t, m, 8000))
		astTestMust(t, e)
		pin, e := m.NewDependencyRef(d)
		astTestMust(t, e)
		for i := range values {
			values[i] = d
		}
		cells := m.allocator.Stats().Live
		payload := m.vectorAllocator.Stats().Live
		m.BudgetField.FailAfter(fail)
		v, e := m.NewDependencyRefVectorFrom(values[:])
		m.BudgetField.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(fail, e)
			}
			sawFailure = true
			if v != (ASTDependencyRefVector{}) {
				t.Fatal(v)
			}
			rc, e := m.DependencyRefCount(d)
			astTestMust(t, e)
			if rc != 1 && rc != 2 {
				t.Fatal("only actual failed-push INC may remain", fail, rc)
			}
			if rc == 2 {
				astTestMust(t, m.DependencyDec(d))
			}
		} else {
			sawSuccess = true
			astTestMust(t, m.DelDependencyRefVector(&v))
		}
		if m.allocator.Stats().Live != cells || m.vectorAllocator.Stats().Live != payload {
			t.Fatal("lost constructor roots before Close", fail, m.allocator.Stats().Live, cells, m.vectorAllocator.Stats().Live, payload)
		}
		if rc, e := m.DependencyRefCount(d); e != nil || rc != 1 {
			t.Fatal(fail, rc, e)
		}
		astTestMust(t, m.DelDependencyRef(&pin))
		if sawSuccess {
			break
		}
	}
	if !sawSuccess || !sawFailure {
		t.Fatal(sawSuccess, sawFailure)
	}
}
func TestASTDependencyRefBufferScalarCapacityBoundaries(t *testing.T) {
	if _, _, e := astDependencyBufferCapacity(16, 256); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual header must fit simulated host limit", e)
	}
	if n, bytes, e := astDependencyBufferCapacity(16, ^uint64(0)); e != nil || n != 32 || bytes != 268 {
		t.Fatal(n, bytes, e)
	}
	for _, old := range []uint64{1 << 31, 1 << 29, ^uint64(0)} {
		if _, _, e := astDependencyBufferCapacity(old, ^uint64(0)); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(old, e)
		}
	}
	if _, _, e := astDependencyBufferCapacity(1<<28, (1<<31)-1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual simulated32bit host bound", e)
	}
}

func TestASTDependencyRefVectorPointerComparisonsDoNotDereferenceAbsentDeadValue(t *testing.T) {
	m := astTestManager(t)
	dead, e := m.DependencyLeaf(astDependencyExpr(t, m, 8100))
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(dead))
	astTestMust(t, m.DependencyDec(dead))
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorResize(v, 1))
	if yes, e := m.DependencyRefVectorContains(v, dead); e != nil || yes {
		t.Fatal("source pointer comparison is not INC/dereference", yes, e)
	}
	astTestMust(t, m.DependencyRefVectorEraseValue(v, dead))
	astTestMust(t, m.DelDependencyRefVector(&v))
}

func TestASTDependencyRefAssignmentDebugRefusalRetainsSourceIncomingIncrement(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	old, e := m.DependencyLeaf(astDependencyExpr(t, m, 8200))
	astTestMust(t, e)
	incoming, e := m.DependencyLeaf(astDependencyExpr(t, m, 8201))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(incoming)
	astTestMust(t, e)
	c, e := m.dependencies.node(old.record)
	astTestMust(t, e)
	c.rc = astDependencyMaxRC
	astTestMust(t, m.dependencies.store(old.record, c))
	r, e := m.NewDependencyRef(old)
	astTestMust(t, e)
	e = m.DependencyRefSet(r, incoming)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	if got, e := m.DependencyRefGet(r); e != nil || got != old {
		t.Fatal(got, e)
	}
	if rc, e := m.DependencyRefCount(incoming); e != nil || rc != 2 {
		t.Fatal("completed incoming INC cannot roll back", rc, e)
	}
	astTestMust(t, m.DependencyDec(incoming))
	c.rc = 1
	astTestMust(t, m.dependencies.store(old.record, c))
	astTestMust(t, m.DelDependencyRef(&r))
	astTestMust(t, m.DelDependencyRef(&pin))
}
func TestASTDependencyRefRetainedCopyOnlyGetAndPhysicalBorrowOrdering(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 8300))
	astTestMust(t, e)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorPush(v, d))
	el, e := m.DependencyRefVectorElement(v, 0)
	astTestMust(t, e)
	it, e := m.DependencyRefVectorBegin(v)
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorPop(v))
	got, e := m.DependencyRefVectorGet(v, 0)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else {
		astTestMust(t, e)
		if got != d {
			t.Fatal(got)
		}
	}
	got, e = m.DependencyRefElementGet(el)
	astTestMust(t, e)
	if got != d {
		t.Fatal("source element borrow has no new logical assertion", got)
	}
	got, ok, e := m.DependencyRefVectorNext(&it)
	astTestMust(t, e)
	if !ok || got != d {
		t.Fatal(got, ok)
	}
	if _, e = m.DependencyIsLeaf(got); !errors.Is(e, ErrASTHandle) {
		t.Fatal("actual dereference validates lifetime", e)
	}
	astTestMust(t, m.DelDependencyRefVector(&v))
}
func TestASTDependencyRefBufferSetXWrappedResizeBeforeIncrement(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 8400))
	astTestMust(t, e)
	pin, e := m.NewDependencyRef(d)
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferPush(b, d))
	e = m.DependencyRefBufferSetX(b, ^uint32(0), d)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if n, e := m.DependencyRefBufferSize(b); e != nil || n != 0 {
		t.Fatal("uint32 wrap clears logical range before Set", n, e)
	}
	if rc, e := m.DependencyRefCount(d); e != nil || rc != 2 {
		t.Fatal("Set INC precedes its failing source subscript", rc, e)
	}
	astTestMust(t, m.DependencyDec(d))
	astTestMust(t, m.DelDependencyRefBuffer(&b))
	astTestMust(t, m.DelDependencyRef(&pin))
}

func TestASTDependencyRefVectorGrowingSelfAppendGuardAndEmptyFastPath(t *testing.T) {
	m := astTestManager(t)
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorAppend(v, v))
	astTestMust(t, m.DependencyRefVectorResize(v, 1))
	before := m.vectorAllocator.Stats().Live
	e = m.DependencyRefVectorAppend(v, v)
	if !errors.Is(e, ErrASTDependencySelfAppend) {
		t.Fatal(e)
	}
	if n, e := m.DependencyRefVectorSize(v); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if m.vectorAllocator.Stats().Live != before {
		t.Fatal("guard must precede growing traversal")
	}
	astTestMust(t, m.DelDependencyRefVector(&v))
}

func TestASTDependencyRefBufferRawPointerCreationHasNoLogicalAssertion(t *testing.T) {
	m := astTestManager(t)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefBufferResize(b, 3))
	astTestMust(t, m.DependencyRefBufferShrink(b, 1))
	el, e := m.DependencyRefBufferElement(b, 2)
	astTestMust(t, e)
	if got, e := m.DependencyRefBufferBorrowGet(el); e != nil || !got.IsNull() {
		t.Fatal(got, e)
	}
	astTestMust(t, m.DelDependencyRefBuffer(&b))
}

func TestASTDependencyRefRangeRefusalResumesAfterConsumedPrefix(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	for _, buffer := range []bool{false, true} {
		m := astTestManager(t)
		first, e := m.DependencyLeaf(astDependencyExpr(t, m, 8500))
		astTestMust(t, e)
		second, e := m.DependencyLeaf(astDependencyExpr(t, m, 8501))
		astTestMust(t, e)
		c, e := m.dependencies.node(second.record)
		astTestMust(t, e)
		c.rc = astDependencyMaxRC
		astTestMust(t, m.dependencies.store(second.record, c))
		var v ASTDependencyRefVector
		var b ASTDependencyRefBuffer
		if buffer {
			b, e = m.NewDependencyRefBuffer()
			astTestMust(t, e)
			astTestMust(t, m.DependencyRefBufferPush(b, first))
			astTestMust(t, m.DependencyRefBufferPush(b, second))
		} else {
			v, e = m.NewDependencyRefVector()
			astTestMust(t, e)
			astTestMust(t, m.DependencyRefVectorPush(v, first))
			astTestMust(t, m.DependencyRefVectorPush(v, second))
		}
		cells := m.allocator.Stats().Live
		if buffer {
			e = m.DependencyRefBufferReset(b)
		} else {
			e = m.DependencyRefVectorReset(v)
		}
		if !errors.Is(e, ErrASTDomain) || m.dependencies.live != 1 || m.allocator.Stats().Live != cells-1 {
			t.Fatal("actual consumed prefix before refused DEC", buffer, e, m.dependencies.live, m.allocator.Stats().Live, cells)
		}
		// The source raw header/slots remain visible; only callback ownership advances.
		if buffer {
			if n, e := m.DependencyRefBufferSize(b); e != nil || n != 2 {
				t.Fatal(n, e)
			}
		} else {
			if n, e := m.DependencyRefVectorSize(v); e != nil || n != 2 {
				t.Fatal(n, e)
			}
		}
		c, e = m.dependencies.node(second.record)
		astTestMust(t, e)
		c.rc = 1
		astTestMust(t, m.dependencies.store(second.record, c))
		if buffer {
			astTestMust(t, m.DependencyRefBufferReset(b))
			astTestMust(t, m.DelDependencyRefBuffer(&b))
		} else {
			astTestMust(t, m.DependencyRefVectorReset(v))
			astTestMust(t, m.DelDependencyRefVector(&v))
		}
		if m.dependencies.live != 0 || m.allocator.Stats().Live != cells-3 {
			t.Fatal("retry must not repeat freed-prefix DEC or lose suffix", buffer, m.dependencies.live, m.allocator.Stats().Live, cells)
		}
	}
}
func TestASTDependencyRefVectorResizeFillUnusedDeadValueIsNoAction(t *testing.T) {
	m := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 8600))
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(d))
	astTestMust(t, m.DependencyDec(d))
	v, e := m.NewDependencyRefVector()
	astTestMust(t, e)
	astTestMust(t, m.DependencyRefVectorResizeFill(v, 0, d))
	astTestMust(t, m.DelDependencyRefVector(&v))
}

func TestASTDependencyRefCleanupMetadataBoundAndProgressTransfer(t *testing.T) {
	limit := uint32((^uint32(0) - 12) / 8)
	for _, target := range []uint32{0, 16, limit} {
		w, e := astDependencyRefCleanupWord(target, true)
		astTestMust(t, e)
		if w&(1<<31) == 0 || w&^(uint32(1)<<31) != target {
			t.Fatal(w, target)
		}
	}
	if _, e := astDependencyRefCleanupWord(1<<29, true); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	first, e := m.DependencyLeaf(astDependencyExpr(t, m, 8700))
	astTestMust(t, e)
	second, e := m.DependencyLeaf(astDependencyExpr(t, m, 8701))
	astTestMust(t, e)
	c, e := m.dependencies.node(second.record)
	astTestMust(t, e)
	c.rc = astDependencyMaxRC
	astTestMust(t, m.dependencies.store(second.record, c))
	v, e := m.NewDependencyRefVectorFrom([]ASTDependency{first, second})
	astTestMust(t, e)
	e = m.DependencyRefVectorReset(v)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	if e = m.DependencyRefVectorPush(v, ASTDependency{}); !errors.Is(e, ErrASTDependencyCleanupPending) {
		t.Fatal("pending mutation guard", e)
	}
	moved, e := m.MoveDependencyRefVector(v)
	astTestMust(t, e)
	astTestMust(t, m.SwapDependencyRefVectors(v, moved))
	state, e := m.dependencyVectorState(v)
	astTestMust(t, e)
	if !state.decPending || state.decNext != 1 || state.decTarget != 0 {
		t.Fatal(state)
	}
	if bytes, e := m.allocator.Size(v.record); e != nil || bytes != 24 {
		t.Fatal("real charged progress width", bytes, e)
	}
	c, e = m.dependencies.node(second.record)
	astTestMust(t, e)
	c.rc = 1
	astTestMust(t, m.dependencies.store(second.record, c))
	astTestMust(t, m.DelDependencyRefVector(&v))
	astTestMust(t, m.DelDependencyRefVector(&moved))
	if m.dependencies.live != 0 {
		t.Fatal(m.dependencies.live)
	}
}
func TestASTDependencyRefPendingForceManagerCloseReclaimsPhysicalGraph(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkVar(8800, sort)
	astTestMust(t, e)
	d, e := m.DependencyLeaf(h)
	astTestMust(t, e)
	b, e := m.NewDependencyRefBuffer()
	astTestMust(t, e)
	if ASTSourceDebug {
		c, e := m.dependencies.node(d.record)
		astTestMust(t, e)
		c.rc = astDependencyMaxRC
		astTestMust(t, m.dependencies.store(d.record, c))
	}
	astTestMust(t, m.DependencyRefBufferPush(b, d))
	if ASTSourceDebug {
		e = m.DependencyRefBufferReset(b)
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
		if e = m.DependencyRefBufferPush(b, ASTDependency{}); !errors.Is(e, ErrASTDependencyCleanupPending) {
			t.Fatal(e)
		}
	}
	allocator, vectorAllocator := m.allocator, m.vectorAllocator
	before := m.BudgetField.Used()
	owned := m.Stats().AllocatedBytes
	astTestMust(t, m.Close())
	if allocator.Stats().Live != 0 || vectorAllocator.Stats().Live != 0 || m.dependencies.live != 0 {
		t.Fatal("force close lost encoded pending roots")
	}
	if m.BudgetField.Used() != before-owned {
		t.Fatal("actual force-close graph refund", m.BudgetField.Used(), before, owned)
	}
}

func TestASTDependencyRefResumedConsumedErrorStillFinishesRequestedDestruction(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	for _, buffer := range []bool{false, true} {
		m := astTestManager(t)
		a, e := m.DependencyLeaf(astDependencyExpr(t, m, 8900))
		astTestMust(t, e)
		sort, e := m.MkBoolSort()
		astTestMust(t, e)
		var args [32]ASTHandle
		for i := range args {
			args[i], e = m.MkVar(uint32(8901+i), sort)
			astTestMust(t, e)
		}
		h, e := m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
		astTestMust(t, e)
		d, e := m.DependencyLeaf(h)
		astTestMust(t, e)
		c, e := m.dependencies.node(d.record)
		astTestMust(t, e)
		c.rc = astDependencyMaxRC
		astTestMust(t, m.dependencies.store(d.record, c))
		var v ASTDependencyRefVector
		var b ASTDependencyRefBuffer
		if buffer {
			b, e = m.NewDependencyRefBuffer()
			astTestMust(t, e)
			astTestMust(t, m.DependencyRefBufferPush(b, a))
			astTestMust(t, m.DependencyRefBufferPush(b, d))
			e = m.DependencyRefBufferShrink(b, 1)
		} else {
			v, e = m.NewDependencyRefVector()
			astTestMust(t, e)
			astTestMust(t, m.DependencyRefVectorPush(v, a))
			astTestMust(t, m.DependencyRefVectorPush(v, d))
			e = m.DependencyRefVectorShrink(v, 1)
		}
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal(e)
		}
		c.rc = 1
		astTestMust(t, m.dependencies.store(d.record, c))
		m.BudgetField.FailAfter(0)
		if buffer {
			e = m.DelDependencyRefBuffer(&b)
		} else {
			e = m.DelDependencyRefVector(&v)
		}
		m.BudgetField.FailAfter(-1)
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("actual consumed AST worklist refusal", e)
		}
		if m.dependencies.live != 0 || v != (ASTDependencyRefVector{}) || b != (ASTDependencyRefBuffer{}) {
			t.Fatal("completed resumed phase must not abandon still-owned prefix", buffer, e, m.dependencies.live, v, b)
		}
	}
}

func TestASTDependencyRefConstructorRefusedCleanupPublishesOwner(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	m := astTestManager(t)
	foreign := astTestManager(t)
	d, e := m.DependencyLeaf(astDependencyExpr(t, m, 9000))
	astTestMust(t, e)
	other, e := foreign.DependencyLeaf(astDependencyExpr(t, foreign, 9001))
	astTestMust(t, e)
	astTestMust(t, foreign.DependencyInc(other))
	defer func() { astTestMust(t, foreign.DependencyDec(other)) }()
	c, e := m.dependencies.node(d.record)
	astTestMust(t, e)
	c.rc = astDependencyMaxRC
	astTestMust(t, m.dependencies.store(d.record, c))
	v, e := m.NewDependencyRefVectorFrom([]ASTDependency{d, other})
	if !errors.Is(e, ErrASTHandle) || v.origin == 0 {
		t.Fatal("refused unwind must publish owned continuation", v, e)
	}
	s, e := m.dependencyVectorState(v)
	astTestMust(t, e)
	if !s.decPending || s.decNext != 0 || s.size != 1 {
		t.Fatal(s)
	}
	c, e = m.dependencies.node(d.record)
	astTestMust(t, e)
	c.rc = 1
	astTestMust(t, m.dependencies.store(d.record, c))
	astTestMust(t, m.DelDependencyRefVector(&v))
	if m.dependencies.live != 0 {
		t.Fatal(m.dependencies.live)
	}
}

func TestASTDependencyRefCopyConstructorRefusedCleanupPublishesOwner(t *testing.T) {
	if !ASTSourceDebug {
		return
	}
	found := false
	for fail := int64(0); fail < 40; fail++ {
		m := astTestManager(t)
		d, e := m.DependencyLeaf(astDependencyExpr(t, m, uint32(9100+fail)))
		astTestMust(t, e)
		src, e := m.NewDependencyRefVectorFrom([]ASTDependency{d, {}, {}})
		astTestMust(t, e)
		c, e := m.dependencies.node(d.record)
		astTestMust(t, e)
		c.rc = astDependencyMaxRC
		astTestMust(t, m.dependencies.store(d.record, c))
		m.BudgetField.FailAfter(fail)
		dst, err := m.CopyDependencyRefVector(src)
		m.BudgetField.FailAfter(-1)
		if err != nil && dst.origin != 0 {
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal(err)
			}
			s, e := m.dependencyVectorState(dst)
			astTestMust(t, e)
			if !s.decPending || s.decNext != 0 {
				t.Fatal(s)
			}
			found = true
		}
		c, e = m.dependencies.node(d.record)
		astTestMust(t, e)
		c.rc = 1
		if dst.origin != 0 {
			c.rc++
		}
		astTestMust(t, m.dependencies.store(d.record, c))
		if dst.origin != 0 {
			astTestMust(t, m.DelDependencyRefVector(&dst))
		}
		astTestMust(t, m.DelDependencyRefVector(&src))
		if found {
			break
		}
	}
	if !found {
		t.Fatal("missing partial owner at actual copy cleanup refusal")
	}
}
