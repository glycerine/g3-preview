// Source: src/math/polynomial/upolynomial_factorization.cpp:677-700,805-871.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialHenselActualConditionalCheckerDeclarations(t *testing.T) {
	_, s := henselDirectFixtureWithConfig(t, func() bool { return true }, true)
	m := s.manager
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	zp, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	zpe, e := NewRelatedUPolynomialManager(m)
	serviceMust(t, e)
	c.prime, c.lift = zp, zpe
	five, twentyFive := unumeral.NewInt(5), unumeral.NewInt(25)
	serviceMust(t, zp.SetZP(&five))
	serviceMust(t, zpe.SetZP(&twentyFive))
	pf, e := zp.NewFactors()
	serviceMust(t, e)
	lf, e := zpe.NewFactors()
	serviceMust(t, e)
	var f, a, al upolyVectorState
	serviceMust(t, m.set(&f, []unumeral.Int{unumeral.NewInt(-6), unumeral.NewInt(0), unumeral.NewInt(1)}))
	for _, v := range [][]unumeral.Int{{unumeral.NewInt(-1), unumeral.NewInt(1)}, {unumeral.NewInt(1), unumeral.NewInt(1)}} {
		serviceMust(t, m.set(&a, v))
		serviceMust(t, pf.PushCopy(m.stateView(&a), 1))
	}
	for _, v := range [][]unumeral.Int{{unumeral.NewInt(9), unumeral.NewInt(1)}, {unumeral.NewInt(-9), unumeral.NewInt(1)}} {
		serviceMust(t, m.set(&al, v))
		serviceMust(t, lf.PushCopy(m.stateView(&al), 1))
	}
	ok, e := s.checkIndividualLiftWithCaller(zp, zp.stateView(&pf.factors[0]), zpe, zpe.stateView(&lf.factors[0]), c)
	serviceMust(t, e)
	if !ok {
		t.Fatal("independent individual reduction")
	}
	ok, e = s.checkHenselFactorsWithCaller(m.stateView(&f), pf, lf, 2, c)
	serviceMust(t, e)
	if !ok {
		t.Fatal("independent actual full checker")
	}
	if (c.checkPower).Cell != 0 || c.checkSource != 0 || c.individualSource != 0 {
		t.Fatal("actual conditional lexical suffix")
	}
	c.executing = false
	serviceMust(t, s.CleanupRetained())
	for _, v := range []*upolyVectorState{&f, &a, &al} {
		serviceMust(t, m.reset(v))
		m.freeBacking(v)
	}
	serviceMust(t, pf.Close())
	serviceMust(t, lf.Close())
	serviceMust(t, zp.Close())
	serviceMust(t, zpe.Close())
}

func TestUPolynomialHenselActualCheckerEveryBIGReservation(t *testing.T) {
	sawDirectLeaf := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		q, s := henselDirectFixtureWithConfig(t, func() bool { return true }, true)
		m := s.manager
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		serviceMust(t, s.beginHenselCaller(parent))
		c := &parent.hensel
		zp, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		zpe, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.prime, c.lift = zp, zpe
		var prime, pe unumeral.Int
		serviceMust(t, q.Integers().SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
		serviceMust(t, q.Integers().Mul(&prime, &prime, &pe))
		serviceMust(t, zp.SetZP(&prime))
		serviceMust(t, zpe.SetZP(&pe))
		pf, e := zp.NewFactors()
		serviceMust(t, e)
		lf, e := zpe.NewFactors()
		serviceMust(t, e)
		var f, a upolyVectorState
		serviceMust(t, m.set(&f, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), unumeral.NewInt(1)}))
		for _, v := range [][]unumeral.Int{{unumeral.NewInt(-1), unumeral.NewInt(1)}, {unumeral.NewInt(1), unumeral.NewInt(1)}} {
			serviceMust(t, m.set(&a, v))
			serviceMust(t, pf.PushCopy(m.stateView(&a), 1))
			serviceMust(t, lf.PushCopy(m.stateView(&a), 1))
		}
		q.Budget().FailAfter(ordinal)
		c.enterCall()
		ok, body := s.checkHenselFactorsWithCaller(m.stateView(&f), pf, lf, 2, c)
		c.leaveCall()
		c.executing = false
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf}, &prime, &pe)
		if polynomialNegNumericPending(&c.numeric) {
			sawDirectLeaf = true
			held := c.numeric
			serviceMust(t, c.own(false))
			if (c.numeric).NumericAllocationRoots != (held).NumericAllocationRoots {
				t.Fatal("false ACK consumed direct unjoined Power/NUM allocation")
			}
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf}, &prime, &pe)
		}
		q.Budget().FailAfter(-1)
		serviceMust(t, c.own(false))
		serviceMust(t, s.CleanupRetained())
		for _, v := range []*upolyVectorState{&f, &a} {
			serviceMust(t, m.reset(v))
			m.freeBacking(v)
		}
		serviceMust(t, pf.Close())
		serviceMust(t, lf.Close())
		serviceMust(t, zp.Close())
		serviceMust(t, zpe.Close())
		serviceMust(t, q.Integers().Del(&prime))
		serviceMust(t, q.Integers().Del(&pe))
		if body == nil {
			if !ok {
				t.Fatal("independent BIG checker equality")
			}
			if !sawDirectLeaf {
				t.Fatal("BIG checker reached no actual direct leaf receipt")
			}
			t.Log("first actual BIG checker completion", ordinal)
			return
		}
	}
	t.Fatal("actual BIG checker never completed")
}

func TestUPolynomialHenselActualTaggedEnclosingEveryReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		q, s := henselDirectFixtureWithConfig(t, func() bool { return true }, true)
		m := s.manager
		parent, e := s.newSquareFreeScope()
		serviceMust(t, e)
		serviceMust(t, s.beginHenselCaller(parent))
		c := &parent.hensel
		zp, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		zpe, e := NewRelatedUPolynomialManager(m)
		serviceMust(t, e)
		c.prime, c.lift = zp, zpe
		five := unumeral.NewInt(5)
		serviceMust(t, zp.SetZP(&five))
		pf, e := zp.NewFactors()
		serviceMust(t, e)
		lf, e := zpe.NewFactors()
		serviceMust(t, e)
		var f, a upolyVectorState
		serviceMust(t, m.set(&f, []unumeral.Int{unumeral.NewInt(-6), unumeral.NewInt(0), unumeral.NewInt(1)}))
		for _, v := range [][]unumeral.Int{{unumeral.NewInt(-1), unumeral.NewInt(1)}, {unumeral.NewInt(1), unumeral.NewInt(1)}} {
			serviceMust(t, m.set(&a, v))
			serviceMust(t, pf.PushCopy(m.stateView(&a), 1))
		}
		q.Budget().FailAfter(ordinal)
		body := s.henselFactorsWithCaller(m.stateView(&f), pf, 2, lf, c)
		primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf})
		if body == nil {
			if lf.count != 2 {
				t.Fatal("literal successful two-factor publication")
			}
			for i, want := range []unumeral.Int{unumeral.NewInt(9), unumeral.NewInt(-9)} {
				if lf.factors[i].size != 2 || m.values(&lf.factors[i])[0] != want || m.values(&lf.factors[i])[1] != unumeral.NewInt(1) {
					t.Fatal("independent enclosing lifted factors", ordinal, i)
				}
			}
			if polynomialSourceDebug && !c.checkPowerConstructed {
				t.Fatal("tag-true caller skipped actual final checker")
			}
			if c.active || c.outerSource != 0 || c.individualSource != 0 || c.checkSource != 0 {
				t.Fatal("enclosing completed source suffix")
			}
		}
		q.Budget().FailAfter(-1)
		c.executing = false
		serviceMust(t, c.own(false))
		serviceMust(t, s.CleanupRetained())
		for _, v := range []*upolyVectorState{&f, &a} {
			serviceMust(t, m.reset(v))
			m.freeBacking(v)
		}
		serviceMust(t, pf.Close())
		serviceMust(t, lf.Close())
		serviceMust(t, zp.Close())
		serviceMust(t, zpe.Close())
		if body == nil {
			t.Log("first actual tag-true enclosing completion", ordinal)
			return
		}
	}
	t.Fatal("actual tagged enclosing caller never completed")
}

// Public signed-accounting operations reach a real postphysical source NUM DT.
// This is a guarded owner-primitive profile, not ordinary mathematical OOM.
func TestUPolynomialHenselActualCheckerSourceStopProtectsOlderDeclarations(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
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
	var prime, pe unumeral.Int
	serviceMust(t, q.Integers().SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	serviceMust(t, q.Integers().Mul(&prime, &prime, &pe))
	serviceMust(t, zp.SetZP(&prime))
	serviceMust(t, zpe.SetZP(&pe))
	pf, e := zp.NewFactors()
	serviceMust(t, e)
	lf, e := zpe.NewFactors()
	serviceMust(t, e)
	var f, a upolyVectorState
	serviceMust(t, u.set(&f, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), unumeral.NewInt(1)}))
	for _, v := range [][]unumeral.Int{{unumeral.NewInt(-1), unumeral.NewInt(1)}, {unumeral.NewInt(1), unumeral.NewInt(1)}} {
		serviceMust(t, u.set(&a, v))
		serviceMust(t, pf.PushCopy(u.stateView(&a), 1))
		serviceMust(t, lf.PushCopy(u.stateView(&a), 1))
	}
	c.outerConstructed = 4
	for i := range c.outer {
		serviceMust(t, u.set(&c.outer[i], []unumeral.Int{unumeral.NewInt(7)}))
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
	cause := c.sourceCause
	if !armed || !errors.Is(cause, arena.ErrBounds) || c.checkSource != 0 || c.checkPowerEnded || c.outer != older || c.finalLCInv != olderLC {
		t.Fatal("actual younger checker NUM DT stop", body, cause, c.checkPowerEnded)
	}
	if !errors.Is(body, cause) || !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("body+source returned cause graph", body)
	}
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{pf, lf}, &prime, &pe)
	if c.finish() != cause || c.outer != older || c.finalLCInv != olderLC {
		t.Fatal("source retry drained older fields")
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.own(false) })
	contextTLSFinish(t, rt, s.CleanupRetained)
	if c.sourceCause != cause || !errors.Is(body, cause) {
		t.Fatal("owning erased original source history")
	}
	for _, v := range []*upolyVectorState{&f, &a} {
		contextTLSFinish(t, rt, func() error { return u.reset(v) })
		u.freeBacking(v)
	}
	contextTLSFinish(t, rt, pf.Close)
	contextTLSFinish(t, rt, lf.Close)
	contextTLSFinish(t, rt, zp.Close)
	contextTLSFinish(t, rt, zpe.Close)
	contextTLSFinish(t, rt, func() error { return q.Integers().Del(&prime) })
	contextTLSFinish(t, rt, func() error { return q.Integers().Del(&pe) })
	x := rt.Deallocate(&recovery)
	if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal(x)
	}
}

// Invalid Go frame/copy/stale contexts are guard-domain proofs, not new source
// allocation frontiers. A nil foreign route remains independently caller-owned.
func TestUPolynomialHenselCheckerPowerContextRejectsCopiedStaleForeign(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	m := s.manager
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	context := numericIntegerModularManager_henselCallerAllocationContext(m.modular, c)
	two, out := unumeral.NewInt(2), unumeral.NewInt(7)
	copied := *c
	bad := context
	bad.henselCaller = &copied
	used := q.Budget().Used()
	var receipt unumeral.IntegerPowerInvocationReceipt
	if e := q.Integers().WithAllocationContext((bad).utility(q.Integers())).PowerTracked(&two, 33, &out, &receipt); e != unumeral.ErrNumericOwner || out != unumeral.NewInt(7) || q.Budget().Used() != used || receipt != (unumeral.IntegerPowerInvocationReceipt{}) {
		t.Fatal("copied caller acquired Power leaf", e)
	}
	m.callbackDepth++
	if e := q.Integers().WithAllocationContext((context).utility(q.Integers())).PowerTracked(&two, 33, &out, &receipt); e != unumeral.ErrNumericOwner || out != unumeral.NewInt(7) || q.Budget().Used() != used {
		t.Fatal("callback acquired old caller Power", e)
	}
	m.callbackDepth--
	c.executing = false
	serviceMust(t, c.own(false))
	serviceMust(t, c.own(true))
	serviceMust(t, s.beginHenselCaller(parent))
	newInvocation := c.invocation
	if e := q.Integers().WithAllocationContext((context).utility(q.Integers())).PowerTracked(&two, 33, &out, &receipt); e != unumeral.ErrNumericOwner || out != unumeral.NewInt(7) || receipt != (unumeral.IntegerPowerInvocationReceipt{}) || c.invocation != newInvocation {
		t.Fatal("old actual context retargeted new Power", e)
	}
	// Actual nil Power is a foreign caller's quick allocation and is not captured.
	serviceMust(t, q.Integers().Power(&two, 33, &out))
	saved := out
	c.executing = false
	serviceMust(t, c.own(false))
	serviceMust(t, s.CleanupRetained())
	if out != saved {
		t.Fatal("parent own cleanup consumed foreign caller output")
	}
	serviceMust(t, q.Integers().Del(&out))
	primeTrialPhysicalAudit(t, q, s)
}
