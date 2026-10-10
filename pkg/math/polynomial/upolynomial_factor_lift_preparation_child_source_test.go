// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175,805-977.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// Qualified compiled checker profile uses the actual nested helper and captured
// tuple. It tests parent transport, not an ordinary allocation between moves.
func TestUPolynomialFactorLiftPreparationCapturedCheckerSourceStop(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	armed := false
	var external umemory.Block
	var c *upolyHenselCallerScope
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Cooperation: func() error { return nil }, Checkpoint: func() bool {
		if c != nil && c.checkSource == 2 && !armed {
			if !c.checkPowerConstructed || (c.checkPower).Cell == 0 {
				t.Fatal("actual Power publication before first checker MUL")
			}
			armed = true
			contextTLSNearMinimum(t, rt)
			x := rt.Deallocate(&external)
			if !errors.Is(x, arena.ErrBounds) || external != (umemory.Block{}) {
				t.Fatal("actual signed release guard", x)
			}
			return false
		}
		return true
	}})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(b)
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
	serviceMust(t, s.beginHenselCaller(parent))
	c = &parent.hensel
	zp, e := NewRelatedUPolynomialManager(u)
	serviceMust(t, e)
	zpe, e := NewRelatedUPolynomialManager(u)
	serviceMust(t, e)
	c.prime, c.lift = zp, zpe
	var prime, pe util.Int
	serviceMust(t, q.Integers().SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	serviceMust(t, q.Integers().Mul(&prime, &prime, &pe))
	serviceMust(t, zp.SetZP(&prime))
	serviceMust(t, zpe.SetZP(&pe))
	pf, e := zp.NewFactors()
	serviceMust(t, e)
	lf, e := zpe.NewFactors()
	serviceMust(t, e)
	var f, a upolyVectorState
	serviceMust(t, u.set(&f, []util.Int{util.NewInt(-1), util.NewInt(0), util.NewInt(1)}))
	for _, v := range [][]util.Int{{util.NewInt(-1), util.NewInt(1)}, {util.NewInt(1), util.NewInt(1)}} {
		serviceMust(t, u.set(&a, v))
		serviceMust(t, pf.PushCopy(u.stateView(&a), 1))
		serviceMust(t, lf.PushCopy(u.stateView(&a), 1))
	}

	out, e := u.NewFactors()
	serviceMust(t, e)
	parent.zp, parent.zpe, parent.best, parent.lifted = zp, zpe, pf, lf
	serviceMust(t, u.set(&parent.fpp, []util.Int{util.NewInt(-1), util.NewInt(0), util.NewInt(1)}))
	parent.constructed = 9
	prep, e := s.beginPrimePreparation(out)
	serviceMust(t, e)
	prep.hensel = c
	prep.henselInvocation = c.invocation
	prep.enterCall()
	c.outerConstructed = 4
	for i := range c.outer {
		serviceMust(t, u.set(&c.outer[i], []util.Int{util.NewInt(7)}))
	}
	c.finalLCInvConstructed = true
	serviceMust(t, q.Integers().Copy(&c.finalLCInv, &q.Integers().Two64))
	older, olderLC := c.outer, c.finalLCInv
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	c.enterCall()
	_, body := s.checkHenselFactorsWithCaller(u.stateView(&f), pf, lf, 2, c)
	c.leaveCall()
	c.executing = false
	prep.executing = false
	prep.leaveCall()
	returned := prep.finish(&prep.receipt, body)
	cause := c.sourceCause
	if !errors.Is(returned, c.sourceCause) || prep.sourceCause != c.sourceCause {
		t.Fatal("captured helper source history", returned, prep.sourceCause)
	}
	if !armed || !errors.Is(cause, arena.ErrBounds) || c.checkSource != 0 || c.checkPowerEnded || c.outer != older || c.finalLCInv != olderLC {
		t.Fatal("actual younger checker NUM DT stop", body, cause, c.checkPowerEnded)
	}
	if !errors.Is(body, cause) || !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("body+source returned cause graph", body)
	}
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf, out}, &prime, &pe)
	if c.finish() != cause || c.outer != older || c.finalLCInv != olderLC {
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
	for _, v := range []*upolyVectorState{&f, &a} {
		contextTLSFinish(t, rt, func() error { return u.reset(v) })
		u.freeBacking(v)
	}
	contextTLSFinish(t, rt, out.Close)
	contextTLSFinish(t, rt, func() error { return q.Integers().Del(&prime) })
	contextTLSFinish(t, rt, func() error { return q.Integers().Del(&pe) })
	x := rt.Deallocate(&recovery)
	if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal(x)
	}
}
