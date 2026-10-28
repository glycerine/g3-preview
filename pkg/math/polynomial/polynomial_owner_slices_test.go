package polynomial

// Source: src/util/scoped_ptr_vector.h;
// src/math/polynomial/polynomial.cpp:2159-2214,2914-2952,4071.

import (
	"reflect"
	"testing"
)

func TestPolynomialSkeletonNativePointerGrowthAndOwningHistory(t *testing.T) {
	m := polynomialCellFixture(t)
	a, e := NewPolynomialSkeleton(m, m.unit, 0)
	serviceMust(t, e)
	b, e := NewPolynomialSkeleton(m, m.unit, 0)
	serviceMust(t, e)
	v, e := NewPolynomialSkeletonVector(m)
	serviceMust(t, e)
	defer func() { m.integer.Budget().FailAfter(-1); serviceMust(t, v.CloseChecked()) }()
	stats, used := m.memory.Stats(), m.integer.Budget().Used()
	m.integer.Budget().FailAfter(0)
	serviceMust(t, v.Push(a))
	serviceMust(t, v.Resize(4097))
	serviceMust(t, v.Push(b))
	if m.memory.Stats() != stats || m.integer.Budget().Used() != used {
		t.Fatal("native skeleton pointer storage charged Runtime")
	}
	size, e := v.Size()
	serviceMust(t, e)
	if size != 4098 {
		t.Fatal("native pointer collection size", size)
	}
	for i := uint32(0); i < size; i++ {
		child, e := v.Get(i)
		serviceMust(t, e)
		var want *PolynomialSkeleton
		if i == 0 {
			want = a
		}
		if i == size-1 {
			want = b
		}
		if child != want {
			t.Fatal("native child association", i)
		}
	}
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.Resize(1))
	if !b.sourceFinalized || b.closed || a.sourceStarted {
		t.Fatal("shrink crossed retained child lifetime")
	}
	serviceMust(t, v.SourceFinalize())
	if v.data != nil || !a.sourceFinalized || a.closed || len(v.children) != 2 {
		t.Fatal("source destruction lost actual owning contexts")
	}
	polynomialSkeletonVectorPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	if !a.closed || !b.closed {
		t.Fatal("owning completion missed a published child")
	}
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMNativeChildrenHaveOnlyActualOwnerCharges(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	defer func() { m.integer.Budget().FailAfter(-1); serviceMust(t, v.CloseChecked()) }()
	serviceMust(t, v.SetOwner(m))
	stats, used := m.memory.Stats(), m.integer.Budget().Used()
	last, e := v.At(128)
	serviceMust(t, e)
	if len(v.children) != 129 || last != v.children[128] {
		t.Fatal("native lazy child growth")
	}
	childBytes := uint64(129) * uint64(reflect.TypeOf(PolynomialSOMBuffer{}).Size())
	if m.memory.Stats() != stats || m.integer.Budget().Used() != used+childBytes {
		t.Fatal("child collection charged redundant backing")
	}
	m.integer.Budget().FailAfter(0)
	again, e := v.At(128)
	serviceMust(t, e)
	if again != last {
		t.Fatal("native reuse replaced an owned child")
	}
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.SourceFinalize())
	for _, child := range v.children {
		if !child.sourceFinalized || child.closed {
			t.Fatal("source/owning child destruction boundary")
		}
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	header := v.header
	serviceMust(t, v.CloseChecked())
	if v.children != nil || !last.closed || m.integer.Budget().Used() != used-header {
		t.Fatal("native collection owning release")
	}
	polynomialPhysicalAudit(t, m)
}
