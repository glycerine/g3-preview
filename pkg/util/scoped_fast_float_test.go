// Concrete owner/domain regressions.
package util

import (
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatVectorCheckedAllocationBounds(t *testing.T) {
	for _, tc := range []struct {
		count, stride, host uint64
		bad                 bool
	}{{2, 8, 2147483647, false}, {134217727, 16, 2147483647, false}, {134217728, 16, 2147483647, true}, {536870910, 8, ^uint64(0) >> 1, false}, {536870911, 8, ^uint64(0) >> 1, true}, {^uint64(0), 16, ^uint64(0), true}} {
		n, e := ffVectorAllocation(tc.count, tc.stride, tc.host)
		if tc.bad {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal(tc, n, e)
			}
		} else if e != nil || n != tc.count*tc.stride {
			t.Fatal(tc, n, e)
		}
	}
}

func TestFastFloatOwnerProvenanceAndVectorLifetime(t *testing.T) {
	m := fastManager(t, 3)
	other := fastManager(t, 3)
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	defer a.Close()
	b, e := NewScopedFastFloat(other)
	serviceMust(t, e)
	defer b.Close()
	serviceMust(t, a.SetInt32(-7))
	serviceMust(t, b.SetInt32(3))
	before := m.config.Storage.Descriptors.Budget.Used()
	if !errors.Is(a.Assign(b), ErrScopedOwner) {
		t.Fatal("foreign assignment accepted")
	}
	if _, e = a.Add(b); !errors.Is(e, ErrScopedOwner) {
		t.Fatal("foreign expression accepted", e)
	}
	if m.config.Storage.Descriptors.Budget.Used() != before {
		t.Fatal("provenance checked after numeric work")
	}
	shallow := *a
	if _, e = shallow.Get(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal("copied cold owner", e)
	}
	v, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer v.Close()
	for i := 0; i < 17; i++ {
		serviceMust(t, v.Push(&a.num))
	}
	if v.capacity != 18 {
		t.Fatal("source growth", v.capacity)
	}
	roots := []*FastFloat{&a.num}
	for i := range v.values {
		roots = append(roots, &v.values[i])
		if v.values[i].index() == a.num.index() {
			t.Fatal("shallow vector copy")
		}
	}
	fastAudit(t, m, roots...)
	// Resize/shrink release ascending lexical entries. Source recycle is LIFO,
	// so a new independent value receives the last destroyed element's ID.
	last := v.values[16].index()
	m.config.Storage.Descriptors.Budget.FailAfter(0)
	serviceMust(t, v.Shrink(0))
	serviceMust(t, v.Resize(1))
	serviceMust(t, m.SetInt32(&v.values[0], 2))
	if v.values[0].index() != last {
		t.Fatal("destructor order", last, v.values[0].index())
	}
	fastAudit(t, m, &a.num, &v.values[0])
	serviceMust(t, v.Close())
	serviceMust(t, v.Close())
	fastAudit(t, m, &a.num)
	if _, e = v.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestFastFloatNeighborAndThinRoutes(t *testing.T) {
	m := fastManager(t, 5)
	var a, b FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	serviceMust(t, m.SetInt32(&a, -7))
	serviceMust(t, m.Copy(&b, &a))
	serviceMust(t, m.Abs(&b))
	serviceMust(t, m.Swap(&a, &b))
	serviceMust(t, m.Neg(&a))
	ok, e := m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.Inc(&a))
	serviceMust(t, m.Dec(&a))
	ok, e = m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetInt32(&a, 2))
	serviceMust(t, m.Inv(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatal("inverse")
	}
	serviceMust(t, m.InvInPlace(&b))
	ok, e = m.Equal(&a, &b)
	fastEnsure(t, ok, e)
	serviceMust(t, m.SetFraction32(&a, -7, 3))
	serviceMust(t, m.Floor(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(-3, 1)) != 0 {
		t.Fatal("floor")
	}
	serviceMust(t, m.Ceil(&a, &b))
	if fastExact(m, &b).Cmp(big.NewRat(-2, 1)) != 0 {
		t.Fatal("ceil")
	}
	serviceMust(t, m.SetMax(&a))
	e = m.Next(&a)
	if !errors.Is(e, ErrFastFloatOverflow) || a.exponent != 2147483647 || !m.minSig(&a) {
		t.Fatal("source next overflow prefix", e, a)
	}
	fastAudit(t, m, &a, &b)
	serviceMust(t, m.SetInt32(&a, 0))
	serviceMust(t, m.SetExponent(&a, 123))
	if a.exponent != 0 {
		t.Fatal("zero exponent noop")
	}
	if e = m.ToInteger(&b, nil, nil); e == nil {
		t.Fatal("invalid integer owner")
	}
}

func TestFastFloatVectorOldNewPeakBudget(t *testing.T) {
	setup := func(limit uint64) (*FastFloatManager, *ScopedFastFloatVector, *arena.Budget) {
		b := arena.NewBudget(limit)
		c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
		c.InitialCapacity = 1024
		m, e := NewFastFloatManager(c)
		serviceMust(t, e)
		v, e := NewScopedFastFloatVector(m)
		serviceMust(t, e)
		serviceMust(t, v.Resize(140))
		serviceMust(t, m.SetInt32(&v.values[0], 3))
		serviceMust(t, m.SetInt32(&v.values[1], 7))
		return m, v, b
	}
	m, v, b := setup(1 << 28)
	baseline := b.Used()
	serviceMust(t, v.Close())
	m.Close()
	if b.Used() != 0 {
		t.Fatal("setup leak")
	}
	stride := uint64(reflect.TypeOf(FastFloat{}).Size())
	m, v, b = setup(baseline + 210*stride - 1)
	e := v.Resize(141)
	if !errors.Is(e, arena.ErrBudget) || v.capacity != 140 || len(v.values) != 140 || b.Used() != baseline {
		t.Fatal("new backing must coexist with old", e, v.capacity, len(v.values), b.Used(), baseline)
	}
	fastAudit(t, m, &v.values[0], &v.values[1])
	serviceMust(t, v.Close())
	m.Close()
	if b.Used() != 0 {
		t.Fatal("peak refusal leak")
	}
}

func TestFastFloatVectorSourceRetainedZeroShrink(t *testing.T) {
	m := fastManager(t, 2)
	v, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer v.Close()
	serviceMust(t, v.Resize(3))
	serviceMust(t, v.Shrink(1))
	used := m.config.Storage.Descriptors.Budget.Used()
	m.config.Storage.Descriptors.Budget.FailAfter(0)
	e = v.Shrink(2)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || len(v.values) != 1 {
			t.Fatal("source shrink assertion", e, len(v.values))
		}
	} else {
		serviceMust(t, e)
		if len(v.values) != 2 || v.values[1] != (FastFloat{}) {
			t.Fatal("release retained initialized zeros", v.values)
		}
	}
	if v.capacity != 3 || m.config.Storage.Descriptors.Budget.Used() != used {
		t.Fatal("retained shrink reserved storage")
	}
	fastAudit(t, m)
}
func TestFastFloatVectorSourceRetainedZeroAt(t *testing.T) {
	m := fastManager(t, 2)
	v, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer v.Close()
	serviceMust(t, v.Resize(3))
	serviceMust(t, v.Shrink(1))
	used := m.config.Storage.Descriptors.Budget.Used()
	m.config.Storage.Descriptors.Budget.FailAfter(0)
	a, e := v.At(2)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || a != nil {
			t.Fatal("source subscript assertion", e, a)
		}
	} else {
		serviceMust(t, e)
		if a == nil || *a != (FastFloat{}) {
			t.Fatal("release retained initialized zeros", a)
		}
	}
	if len(v.values) != 1 || v.capacity != 3 || m.config.Storage.Descriptors.Budget.Used() != used {
		t.Fatal("retained read changed owner")
	}
	fastAudit(t, m)
}

func TestFastFloatVectorRetainedInitializationAndGrowthPrefix(t *testing.T) {
	m := fastManager(t, 2)
	v, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer v.Close()
	serviceMust(t, v.Resize(4))
	serviceMust(t, v.Shrink(1))
	a, e := v.At(4)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || a != nil {
			t.Fatal(e, a)
		}
	} else if !errors.Is(e, ErrFastFloatDomain) || a != nil {
		t.Fatal("never constructed slot is not a source zero", e, a)
	}
	w, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer w.Close()
	serviceMust(t, w.Resize(3))
	serviceMust(t, w.Shrink(1))
	// An initialized scalar record's exponent bytes are defined to read/copy;
	// no arithmetic or canonical-zero claim is made about this private fixture.
	w.values[:cap(w.values)][2].exponent = 17
	m.config.Storage.Descriptors.Budget.FailAfter(1)
	e = w.Resize(6)
	if !errors.Is(e, arena.ErrInjectedFailure) || w.capacity != 5 || len(w.values) != 1 {
		t.Fatal("completed source reallocation prefix", e, w.capacity, len(w.values))
	}
	if w.values[:cap(w.values)][2].exponent != 17 {
		t.Fatal("trivial source realloc must retain entire old capacity")
	}
	a, e = w.At(2)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || a != nil {
			t.Fatal(e, a)
		}
	} else if e != nil || a == nil || a.exponent != 17 {
		t.Fatal("initialized tail survives growth", e, a)
	}
	fastAudit(t, m)
}

func TestFastFloatVectorSourceTrivialCapacityCopy(t *testing.T) {
	m := fastManager(t, 2)
	v, e := NewScopedFastFloatVector(m)
	serviceMust(t, e)
	defer v.Close()
	serviceMust(t, v.Resize(3))
	serviceMust(t, v.Shrink(1))
	v.values[:cap(v.values)][2].exponent = 17
	m.config.Storage.Descriptors.Budget.FailAfter(1)
	e = v.Resize(6)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.capacity != 5 || len(v.values) != 1 {
		t.Fatal(e, v.capacity, len(v.values))
	}
	if v.values[:cap(v.values)][2].exponent != 17 {
		t.Fatal("source trivially-copyable reallocate lost initialized tail")
	}
	fastAudit(t, m)
}
