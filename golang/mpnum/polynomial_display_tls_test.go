// Source: src/math/polynomial/polynomial.cpp:1558-1570; src/util/mpz.cpp;
// src/util/buffer.h; src/util/memory_manager.cpp.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialDisplayYoungerStoppedDestructor(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, p, _ := polynomialEvalFixtureWithRationals(t, q, nil)
	// Real ordinary source destruction warms ID free-vector payloads before the
	// signed-host-domain recipe, isolating the actual lexical char release.
	var warm [5]Polynomial
	for i := range warm {
		a := NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for _, v := range warm {
		serviceMust(t, m.core.DecRef(v))
	}
	x, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	term, e := m.core.monomials.MkVariable(x, 7)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(term))
	serviceMust(t, m.core.monomials.DecRef(term))
	outer, e := m.core.displayFrame(&polynomialDisplayTestSink{})
	serviceMust(t, e)
	outer.constructed = true
	serviceMust(t, outer.numericCall(0, &q.z.two64))
	var younger *polynomialDisplayFrame
	sink := &polynomialDisplayTestSink{decimalHook: func(*PolynomialDecimalView) error {
		var e error
		younger, e = m.core.displayFrame(&polynomialDisplayTestSink{})
		serviceMust(t, e)
		younger.constructed = true
		serviceMust(t, younger.numericCall(0, &q.z.two64))
		serviceMust(t, younger.characters.resize(rt, 200000))
		younger.characterCompleted = true
		younger.characterPending = true
		polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
		contextTLSNearMinimum(t, rt)
		e = younger.destroyCharacters(nil)
		if !errors.Is(e, arena.ErrBounds) || younger.characters.raw != (memory.Block{}) {
			t.Fatal("actual consumed postphysical leaf", e)
		}
		return younger.finish(e)
	}}
	outer.sink = sink
	// Exercise the exact consumer/outer destructor engine with actual BIG input.
	// The nested helper-domain char construction avoids claiming this signed
	// overflow is naturally reachable from valid mathematical display inputs.
	e = outer.displayInteger(&outer.numeral)
	e = outer.finish(e)
	if !errors.Is(e, arena.ErrBounds) || !outer.constructed || outer.numeral.cell == 0 || !outer.characterCompleted {
		t.Fatal("older NUM must survive younger source destructor stop", e, outer.constructed)
	}
	cause := outer.sourceCause
	state := rt.State()
	if e = outer.finish(nil); !errors.Is(e, arena.ErrBounds) || outer.sourceCause != cause || rt.State() != state {
		t.Fatal("sticky source stop without older destructor replay", e)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	contextTLSFinish(t, rt, m.CleanupDisplay)
	if outer.sourceCause != cause || younger.sourceCause != cause {
		t.Fatal("owning abort changed first source history")
	}
	contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
	contextTLSFinish(t, rt, m.CloseChecked)
	// The ordinary fixture closure sees already-closed m and does not replay DEC.
	contextTLSFinish(t, rt, q.CloseChecked)
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("whole real graph refund", budget.Used())
	}
}

func polynomialDisplayTLSOwner(t *testing.T) (*MultivariatePolynomialManager, *RationalManager, *memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = r
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { panic("display has no source checkpoint") }, Cooperation: func() error { panic("display has no source cooperation") }}})
	serviceMust(t, e)
	return m, q, r, b
}
func TestPolynomialDisplayExactNumericReceiptAndReplacement(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprint(replace), func(t *testing.T) {
			m, q, r, b := polynomialDisplayTLSOwner(t)
			z := q.z
			digits := make([]uint32, 40000)
			digits[39999] = 1
			a, foreign := Int{}, Int{}
			serviceMust(t, z.SetLimbs(&a, digits))
			defer func() {
				serviceMust(t, r.SetMaxSize(0))
				serviceMust(t, m.CleanupDisplay())
				serviceMust(t, z.allocator.CleanupPending())
				serviceMust(t, z.Del(&foreign))
				serviceMust(t, z.Del(&a))
				serviceMust(t, m.CloseChecked())
				serviceMust(t, q.CloseChecked())
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("whole exact receipt graph refund", b.Used())
				}
			}()
			f, e := m.core.displayFrame(&polynomialDisplayTestSink{})
			serviceMust(t, e)
			f.constructed = true
			serviceMust(t, r.SetMaxSize(1))
			cause := f.numericCall(0, &a)
			roots, e := z.allocator.PendingRoots()
			serviceMust(t, e)
			if !numericPhysicalReservationRefusal(cause) || roots[0] == (memory.Block{}) || f.numeric.allocator != roots {
				t.Fatal("actual invoked Copy raw receipt", cause, roots, f.numeric.allocator)
			}
			cause = f.finish(cause)
			polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a}, 0)
			old := roots
			if replace {
				serviceMust(t, r.SetMaxSize(0))
				serviceMust(t, z.allocator.CleanupPending())
				serviceMust(t, r.SetMaxSize(1))
				e = z.Copy(&foreign, &a)
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual foreign replacement Copy", e)
				}
				roots, e = z.allocator.PendingRoots()
				serviceMust(t, e)
				if roots[0] == (memory.Block{}) || roots == old {
					t.Fatal("replacement needs a distinct actual token", roots, old)
				}
				polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a, &foreign}, 0)
			}
			serviceMust(t, r.SetMaxSize(0))
			serviceMust(t, m.CleanupDisplay())
			now, e := z.allocator.PendingRoots()
			serviceMust(t, e)
			if replace {
				if now != roots {
					t.Fatal("owning display cleanup swept later foreign raw", now, roots)
				}
			} else if now != ([3]memory.Block{}) {
				t.Fatal("owning display cleanup omitted its own exact Copy raw", now)
			}
			if f.bodyCause == nil || !errors.Is(f.bodyCause, cause) {
				t.Fatal("original body cause lost during owning continuation")
			}
			polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a, &foreign}, 0)
		})
	}
}
func TestPolynomialDisplaySinkCreatesForeignPending(t *testing.T) {
	m, q, r, b := polynomialDisplayTLSOwner(t)
	z := q.z
	a, foreign := Int{}, Int{}
	serviceMust(t, z.Copy(&a, &z.two64))
	p, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, r.SetMaxSize(0))
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, z.allocator.CleanupPending())
		serviceMust(t, z.Del(&foreign))
		serviceMust(t, m.core.CleanupValue(&p))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	var owned [3]memory.Block
	sink := &polynomialDisplayTestSink{decimalHook: func(*PolynomialDecimalView) error {
		digits := make([]uint32, 40000)
		digits[39999] = 1
		serviceMust(t, r.SetMaxSize(1))
		cause := z.SetLimbs(&foreign, digits)
		if !numericPhysicalReservationRefusal(cause) {
			t.Fatal(cause)
		}
		owned, e = z.allocator.PendingRoots()
		serviceMust(t, e)
		if owned[0] == (memory.Block{}) {
			t.Fatal("no actual caller pending raw")
		}
		return cause
	}}
	cause := m.DisplaySMT2Into(p, sink)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal(cause)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&foreign}, 0)
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupDisplay())
	now, e := z.allocator.PendingRoots()
	serviceMust(t, e)
	if now != owned {
		t.Fatal("stream callback roots were attributed to parent display", now, owned)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&foreign}, 0)
}
