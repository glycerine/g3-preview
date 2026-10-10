// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175,805-977.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// Valid compiled GF7 candidate and BIG primitive square-free input reach the
// actual prepareFactorLift capture and debug checker, without seeding child
// declarations. Signed TLS accounting is the qualified reached release domain.
func TestUPolynomialFactorSquareFreeActualCheckerSourceStop(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source conditional checker erased in release profile")
	}
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
	var s *UPolynomialFactorizationService
	var inputState upolyVectorState
	var receipt upolyFactorSquareFreeInvocationReceipt
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Cooperation: func() error { return nil }, Checkpoint: func() bool {
		if s != nil && s.scope != nil {
			c = &s.scope.hensel
		}
		if c != nil && c.checkSource == 2 && (c.checkPower).Cell != 0 && !armed {
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
	s, e = NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes, FactorizationDebugTag: true})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, func() error { return u.reset(&inputState) })
		u.freeBacking(&inputState)
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, u.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual owner refunds", b.Used())
		}
	}()
	out, e := u.NewFactors()
	serviceMust(t, e)
	inputs := []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(1)}
	serviceMust(t, q.Integers().SetString(&inputs[0], "-387381625547900583942"))
	serviceMust(t, q.Integers().SetString(&inputs[1], "-129127208515966861311"))
	serviceMust(t, u.set(&inputState, inputs))
	for i := range inputs {
		serviceMust(t, q.Integers().Del(&inputs[i]))
	}
	external, e = rt.Allocate(200000)
	serviceMust(t, e)

	_, returned := s.factorSquareFreeWithReceipt(u.stateView(&inputState), out, 1, DefaultUPolynomialFactorParams(), &receipt)
	parent := receipt.frame
	if parent == nil {
		t.Fatal("actual caller scope never constructed", returned)
	}
	c = &parent.hensel
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
	values := make([]unumeral.Int, inputState.size)
	roots := make([]*unumeral.Int, len(values))
	for i := range values {
		v, x := u.read(u.stateView(&inputState), uint32(i))
		serviceMust(t, x)
		values[i] = *v
		roots[i] = &values[i]
	}
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{out}, roots...)
	if c.finish() != cause {
		t.Fatal("source retry drained older fields")
	}

	savedLift, savedCore := parent.lifted, parent.zpe
	if e := s.finishSquareFreeScopeSource(); !errors.Is(e, cause) || parent.constructed != 9 || parent.lifted != savedLift || parent.zpe != savedCore {
		t.Fatal("captured younger source stop consumed older lifted fields", e, parent.constructed)
	}
	firstOwn := s.cleanupFactorSquareFreeInvocation(&receipt, false)
	if firstOwn == nil || parent.ownCause != firstOwn {
		t.Fatal("actual matched owning release stop missing", firstOwn, parent.ownCause)
	}
	if s.factorSquareFreeInvocationSourceCause(&receipt) != cause || !errors.Is(body, cause) {
		t.Fatal("owning error replaced immutable source/body history")
	}
	primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{out}, roots...)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return s.cleanupFactorSquareFreeInvocation(&receipt, false) })
	if parent.constructed != 0 {
		t.Fatal("callee lexical suffix remains before older parent locals", parent.constructed)
	}
	// Older multivariate adapter locals would destruct at this point.
	contextTLSFinish(t, rt, func() error { return s.cleanupFactorSquareFreeInvocation(&receipt, true) })
	if s.factorSquareFreeInvocationSourceCause(&receipt) != cause || !parent.retired || parent.ownCause != firstOwn {
		t.Fatal("exact callee history/own retirement")
	}
	if c.sourceCause != cause || !errors.Is(body, cause) {
		t.Fatal("owning erased original source history")
	}
	contextTLSFinish(t, rt, out.Close)
	x := rt.Deallocate(&recovery)
	if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal(x)
	}
}
