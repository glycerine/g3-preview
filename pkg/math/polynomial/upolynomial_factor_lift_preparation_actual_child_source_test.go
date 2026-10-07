// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175,805-977.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Dejan (t-dejanj).
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Valid compiled GF7 candidate and BIG primitive square-free input reach the
// actual prepareFactorLift capture and debug checker, without seeding child
// declarations. Signed TLS accounting is the qualified reached release domain.
func TestUPolynomialFactorLiftPreparationActualHelperCheckerSourceStop(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source conditional checker erased in release profile")
	}
	b := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	armed := false
	var external util.Block
	var c *upolyHenselCallerScope
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Cooperation: func() error { return nil }, Checkpoint: func() bool {
		if c != nil && c.checkSource == 2 && (c.checkPower).Cell != 0 && !armed {
			if !c.checkPowerConstructed || (c.checkPower).Cell == 0 {
				t.Fatal("actual Power publication before first checker MUL")
			}
			armed = true
			contextTLSNearMinimum(t, rt)
			x := rt.Deallocate(&external)
			if !errors.Is(x, arena.ErrBounds) || external != (util.Block{}) {
				t.Fatal("actual signed release guard", x)
			}
			return false
		}
		return true
	}})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes, FactorizationDebugTag: true})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, u.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual owner refunds", b.Used())
		}
	}()
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)

	out, e := u.NewFactors()
	serviceMust(t, e)
	parent.constructed = 1
	inputs := []util.Int{util.NewInt(0), util.NewInt(0), util.NewInt(1)}
	serviceMust(t, q.Integers().SetString(&inputs[0], "-387381625547900583942"))
	serviceMust(t, q.Integers().SetString(&inputs[1], "-129127208515966861311"))
	serviceMust(t, u.set(&parent.fpp, inputs))
	for i := range inputs {
		serviceMust(t, q.Integers().Del(&inputs[i]))
	}
	serviceMust(t, s.preparePrimeSearch(out))
	serviceMust(t, u.modular.SetInt32(&parent.p, 7))
	serviceMust(t, parent.zp.SetZP(&parent.p))
	trials := uint32(0)
	route, e := s.factorPrimeTrial(parent.zp, &parent.fpp, out, parent.best, 1, &parent.p, &parent.bestP, &parent.ds, &trials)
	serviceMust(t, e)
	if route != 1 || parent.bestP != util.NewInt(7) || parent.best.count != 2 {
		t.Fatal("actual eligible uint32 candidate", route, parent.bestP)
	}
	c = &parent.hensel
	external, e = rt.Allocate(200000)
	serviceMust(t, e)

	returned := s.prepareFactorLift(out)
	prep := &parent.primePreparation
	cause := c.sourceCause
	body := returned
	if prep.hensel != c || prep.henselInvocation != c.invocation || parent.constructed != 9 || !prep.matchesHensel() {
		t.Fatal("actual reached helper capture", parent.constructed)
	}
	if !errors.Is(returned, c.sourceCause) || prep.sourceCause != c.sourceCause {
		t.Fatal("captured helper source history", returned, prep.sourceCause)
	}
	if !armed || !errors.Is(cause, arena.ErrBounds) || c.checkSource != 0 || c.checkPowerEnded {
		t.Fatal("actual younger checker NUM DT stop", body, cause, c.checkPowerEnded)
	}
	if !errors.Is(body, cause) || !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("body+source returned cause graph", body)
	}
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{out})
	if c.finish() != cause {
		t.Fatal("source retry drained older fields")
	}

	savedLift, savedCore := parent.lifted, parent.zpe
	if e := s.finishSquareFreeScopeSource(); !errors.Is(e, cause) || parent.constructed != 9 || parent.lifted != savedLift || parent.zpe != savedCore {
		t.Fatal("captured younger source stop consumed older lifted fields", e, parent.constructed)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return prep.own(&prep.receipt, false) })
	contextTLSFinish(t, rt, s.CleanupRetained)
	if c.sourceCause != cause || !errors.Is(body, cause) {
		t.Fatal("owning erased original source history")
	}
	contextTLSFinish(t, rt, out.Close)
	x := rt.Deallocate(&recovery)
	if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal(x)
	}
}
