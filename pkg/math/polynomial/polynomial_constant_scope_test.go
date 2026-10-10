// Source: src/util/rational.h:54; src/math/polynomial/polynomial.cpp:2530-2536,2696-2698.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialConstImplicitRationalUsesActualBorrowedOwner(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner()})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("constant route checkpoint"); return false }, Cooperation: func() error { t.Fatal("constant route cooperate"); return nil }})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()); serviceMust(t, mm.DecManagerRef()) })
	var a util.Int
	serviceMust(t, q.Integers().SetInt64(&a, 1<<40))
	t.Cleanup(func() { serviceMust(t, q.Integers().Del(&a)) })
	before := q.Integers().Cells.Live
	p, e := m.MulAdd(m.zero, Polynomial{}, &a)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	if q.Integers().Cells.Live != before+1 {
		t.Fatal("two const copies, both completed lexical temporaries destroyed", q.Integers().Cells.Live, before)
	}
	got, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	v, e := q.Integers().Int64(&got)
	serviceMust(t, e)
	if v != 1<<40 {
		t.Fatal(v)
	}
	if m.constantScope.active || m.constantScope.rational != (util.Rational{}) || m.constantScope.numeral != (util.Int{}) {
		t.Fatal("completed source local scope retained")
	}
}

// A refusal in the actual implicit rational constructor leaves that constructor
// incomplete: only an explicit owning abort may drain its retained fields.
func TestPolynomialConstFailedConstructorRetainedUntilOwningClose(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("constant checkpoint"); return false }, Cooperation: func() error { t.Fatal("constant cooperation"); return nil }})
	serviceMust(t, e)
	var a util.Int
	serviceMust(t, q.Integers().SetInt64(&a, 1<<40))
	b.FailAfter(0)
	p, e := m.MulAdd(m.zero, Polynomial{}, &a)
	if !numericPhysicalReservationRefusal(e) || p != (Polynomial{}) {
		t.Fatal("actual first-copy refusal", p, e)
	}
	if !m.constantScope.active || !m.constantScope.failedConstructor || m.constantScope.rationalCompleted {
		t.Fatal("failed source constructor incorrectly completed", m.constantScope)
	}
	// Capture the real source roots before changing injection or cleaning them.
	polynomialPhysicalAudit(t, m, &a)
	if got, e := q.Integers().Int64(&a); e != nil || got != 1<<40 {
		t.Fatal("borrowed const input changed", got, e)
	}
	b.FailAfter(-1)
	serviceMust(t, q.Integers().Del(&a))
	serviceMust(t, m.CloseChecked())
	if m.constantScope.active {
		t.Error("owning Close lost retained failed rational constructor")
	}
	serviceMust(t, mm.DecManagerRef())
	serviceMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("owning cleanup retained physical storage", b.Used())
	}
}

func TestPolynomialConstTwoCopyFailurePrefixes(t *testing.T) {
	first, second, success := false, false, false
	for fault := int64(0); fault < 60 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 22)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("constant checkpoint"); return false }, Cooperation: func() error { t.Fatal("constant cooperate"); return nil }})
			serviceMust(t, e)
			var a util.Int
			serviceMust(t, q.Integers().SetInt64(&a, 1<<40))
			count, e := m.memory.AllocationCount()
			serviceMust(t, e)
			before, e := m.monomialRefCount(mm.unit)
			serviceMust(t, e)
			b.FailAfter(fault)
			p, e := m.MulAdd(m.zero, Polynomial{}, &a)
			events, ce := m.memory.AllocationCount()
			serviceMust(t, ce)
			var coef util.Int
			if e == nil {
				success = true
				coef, e = m.Coefficient(p, 0)
				serviceMust(t, e)
				serviceMust(t, m.IncRef(p))
			} else {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("exact refusal transport", e)
				}
				if p != (Polynomial{}) {
					t.Fatal("caller published before completed return", p)
				}
				first = first || m.constantScope.failedConstructor
				second = second || (!m.constantScope.failedConstructor && events-count == 2 && m.pending.value == (Polynomial{}) && m.pending.raw == (umemory.ObjectRef{}))
			}
			polynomialPhysicalAudit(t, m, &a, &coef)
			got, e := q.Integers().Int64(&a)
			serviceMust(t, e)
			if got != 1<<40 {
				t.Fatal("const caller changed", got)
			}
			if success {
				got, e = q.Integers().Int64(&coef)
				serviceMust(t, e)
				if got != 1<<40 {
					t.Fatal(got)
				}
			}
			// All immediate roots/actions have been checked before disabling injection.
			b.FailAfter(-1)
			if success {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, q.Integers().Del(&a))
			serviceMust(t, m.CloseChecked())
			after, e := mm.headerCore(mm.unit)
			serviceMust(t, e)
			if after.RefCount != before-1 {
				t.Fatal("owning abort leaked actual mk_const unit reference", before, after.RefCount)
			}
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("whole actual graph cleanup", b.Used())
			}
		})
	}
	if !first || !second || !success {
		t.Fatal("two copies and first full success not reached", first, second, success)
	}
}
