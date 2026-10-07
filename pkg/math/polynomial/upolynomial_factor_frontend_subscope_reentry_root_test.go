// Source: src/math/polynomial/upolynomial.cpp:3042-3118,2960-3040.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorFrontendReachedSubscopeReentry(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	var s *UPolynomialFactorizationService
	reached := false
	var refusal, sourceRefusal, imageRefusal, modularRefusal error
	subscopeUnchanged := false
	sourceUnchanged := false
	constructorFrontier := uint8(0)
	imageClassConstructed := false
	var before, after uint64
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Cooperation: func() error { return nil }, Checkpoint: func() bool {
		if s != nil && !reached {
			reached = true
			constructorFrontier = s.frontend.modularGCD.constructed
			imageClassConstructed = s.frontend.image.savedConstructed && s.frontend.image.classConstructed
			prior := s.frontend
			priorImage, priorModular := s.frontend.image, s.frontend.modularGCD
			imageRefusal = s.frontend.image.finish(nil)
			modularRefusal = s.frontend.modularGCD.finish(nil)
			subscopeUnchanged = reflect.DeepEqual(priorImage, s.frontend.image) && reflect.DeepEqual(priorModular, s.frontend.modularGCD)
			sourceRefusal = s.frontend.finish(nil)
			sourceUnchanged = reflect.DeepEqual(prior, s.frontend)
			before = b.Used()
			refusal = s.Close()
			after = b.Used()
			return false
		}
		return true
	}})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e = NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	out, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{util.NewInt(1), util.NewInt(1), util.NewInt(0), util.NewInt(1)}))
	defer func() {
		b.FailAfter(-1)
		if !s.closed {
			serviceMust(t, s.Close())
		}
		serviceMust(t, out.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, primes.Close())
		serviceMust(t, m.Close())
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Error("actual owner refunds", b.Used())
		}
	}()
	_, body := s.Factor(m.stateView(&input), out, DefaultUPolynomialFactorParams())
	if !reached || body == nil {
		t.Fatal("actual GCD checkpoint body refusal not reached", reached, body)
	}
	if !imageClassConstructed {
		t.Fatal("actual Euclid checkpoint lost scoped_set_zp saved NUM/class frontier")
	}
	if constructorFrontier != 6 {
		t.Fatal("actual modular GCD checkpoint did not bind six reached source NUMs", constructorFrontier)
	}
	if imageRefusal != ErrUPolynomialCleanupPending || modularRefusal != ErrUPolynomialCleanupPending || !subscopeUnchanged {
		t.Fatal("executing child scope was finalized", imageRefusal, modularRefusal, subscopeUnchanged)
	}
	if sourceRefusal != ErrUPolynomialCleanupPending || !sourceUnchanged {
		t.Fatal("actual source finalize changed executing caller", sourceRefusal, sourceUnchanged)
	}
	if refusal != ErrUPolynomialCleanupPending || s.closed || before != after {
		t.Fatal("frontend owner retired while actual source NUM/vectors live", refusal, s.closed, before, after)
	}
}

func TestUPolynomialFactorFrontendReachedSubscopeOwningReentry(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	var s *UPolynomialFactorizationService
	reached := false
	var refusal, sourceRefusal, imageRefusal, modularRefusal error
	subscopeUnchanged := false
	sourceUnchanged := false
	constructorFrontier := uint8(0)
	imageClassConstructed := false
	var before, after uint64
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Cooperation: func() error { return nil }, Checkpoint: func() bool {
		if s != nil && !reached {
			reached = true
			constructorFrontier = s.frontend.modularGCD.constructed
			imageClassConstructed = s.frontend.image.savedConstructed && s.frontend.image.classConstructed
			prior := s.frontend
			priorImage, priorModular := s.frontend.image, s.frontend.modularGCD
			imageRefusal = s.frontend.image.own()
			modularRefusal = s.frontend.modularGCD.own()
			subscopeUnchanged = reflect.DeepEqual(priorImage, s.frontend.image) && reflect.DeepEqual(priorModular, s.frontend.modularGCD)
			sourceRefusal = s.frontend.finish(nil)
			sourceUnchanged = reflect.DeepEqual(prior, s.frontend)
			before = b.Used()
			refusal = s.Close()
			after = b.Used()
			return false
		}
		return true
	}})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e = NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	out, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	serviceMust(t, m.set(&input, []util.Int{util.NewInt(1), util.NewInt(1), util.NewInt(0), util.NewInt(1)}))
	defer func() {
		b.FailAfter(-1)
		if !s.closed {
			serviceMust(t, s.Close())
		}
		serviceMust(t, out.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, primes.Close())
		serviceMust(t, m.Close())
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Error("actual owner refunds", b.Used())
		}
	}()
	_, body := s.Factor(m.stateView(&input), out, DefaultUPolynomialFactorParams())
	if !reached || body == nil {
		t.Fatal("actual GCD checkpoint body refusal not reached", reached, body)
	}
	if !imageClassConstructed {
		t.Fatal("actual Euclid checkpoint lost scoped_set_zp saved NUM/class frontier")
	}
	if constructorFrontier != 6 {
		t.Fatal("actual modular GCD checkpoint did not bind six reached source NUMs", constructorFrontier)
	}
	if imageRefusal != ErrUPolynomialCleanupPending || modularRefusal != ErrUPolynomialCleanupPending || !subscopeUnchanged {
		t.Fatal("executing child scope accepted owning cleanup", imageRefusal, modularRefusal, subscopeUnchanged)
	}
	if sourceRefusal != ErrUPolynomialCleanupPending || !sourceUnchanged {
		t.Fatal("actual source finalize changed executing caller", sourceRefusal, sourceUnchanged)
	}
	if refusal != ErrUPolynomialCleanupPending || s.closed || before != after {
		t.Fatal("frontend owner retired while actual source NUM/vectors live", refusal, s.closed, before, after)
	}
}
