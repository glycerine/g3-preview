// Source: src/math/polynomial/polynomial.cpp:3461-3465;
// src/util/mpzzp.h:58-68,111-114,229-235; src/util/mpn.cpp:206-217.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The source DEBUG verification local exists only in the checked profile.
// Signed-host TLS isolates its literal DT; it is not mathematical OOM.
func TestPolynomialICCPConstantYoungerPNSourceStopAndImmutableHistory(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	var modulus, value, i util.Int
	words := make([]uint32, 17)
	for j := range words {
		words[j] = ^uint32(0)
	}
	words[16] = 511
	serviceMust(t, c.integer.SetLimbs(&modulus, words))
	words = make([]uint32, 17)
	words[0], words[16] = 7, 1024
	serviceMust(t, c.integer.SetLimbs(&value, words))
	p, e := c.MkConstantTakeCopy(&value)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZP(&modulus))
	var content, primitive polynomialReference
	var child polynomialICCPInvocationReceipt
	serviceMust(t, full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &child))
	serviceMust(t, full.cleanupICCPInvocation(&child, false))
	serviceMust(t, full.cleanupICCPInvocation(&child, true))
	if i != util.NewInt(9) {
		t.Fatal("independent field remainder", i)
	}
	// Distinct older caller values prove SET throws before c/pp assignments.
	serviceMust(t, content.assign(c, p))
	serviceMust(t, primitive.assign(c, p))
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	t.Cleanup(func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, full.CleanupContentPrimitiveParts)
		contextTLSFinish(t, rt, func() error { return content.close(c) })
		contextTLSFinish(t, rt, func() error { return primitive.close(c) })
		contextTLSFinish(t, rt, func() error { return c.DecRef(p) })
		for _, n := range []*util.Int{&modulus, &value, &i} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(n) })
		}
		contextTLSFinish(t, rt, full.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		if external != (util.Block{}) {
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&external) })
		}
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (util.Block{})) {
			t.Fatal(x)
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("actual named owner budget", budget.Used())
		}
	})
	// No checkpoint or invented local is inserted into the constant branch.
	// Its warmed ordinary SET/Rem precedes the reached DEBUG temp destructor.
	contextTLSNearMinimum(t, rt)
	release := rt.Deallocate(&external)
	if !errors.Is(release, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual physical accounting setup", release)
	}
	body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &child)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*util.Int{&modulus, &value, &i}, 0)
	s := &full.iccpConstant
	cause := s.sourceCause
	pn := s.normalize
	if cause == nil || !errors.Is(body, cause) || c.modular.PNormalizeInvocationSourceCause(&pn) != cause || !full.hasICCPConstantHistory(&child.shortcutConstant) || len(full.iccpFrames) != 0 {
		t.Fatal("actual constant PN source-stop missing", body, cause, pn)
	}
	first := body
	originalContent, originalPrimitive := content, primitive
	before := *s
	oldContext := integerAllocationInvocationContext{iccpConstant: &s.directReceipt, iccpConstantInvocation: s.invocation}
	if (oldContext.roots(c.integer)) != nil {
		t.Fatal("completed allocation body still grants new allocation authority")
	}
	var rejected polynomialICCPInvocationReceipt
	retry := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &rejected)
	if !errors.Is(retry, ErrPolynomialPending) || rejected != (polynomialICCPInvocationReceipt{}) || !testNumericScopeEqual(*s, before) || content != originalContent || primitive != originalPrimitive {
		t.Fatal("stopped constant retry changed source prefix or acquired a new frame", retry)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return full.cleanupICCPInvocation(&child, false) })
	if full.iccpInvocationSourceCause(&child) != cause || s.bodyCause == nil {
		t.Fatal("false ACK erased first actual history")
	}
	contextTLSFinish(t, rt, func() error { return c.integer.Del(&i) })
	contextTLSFinish(t, rt, func() error { return full.cleanupICCPInvocation(&child, true) })
	if full.iccpInvocationSourceCause(&child) != cause || c.modular.PNormalizeInvocationSourceCause(&pn) != cause || !errors.Is(first, cause) {
		t.Fatal("true ACK erased child/parent cause")
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, content.value, primitive.value}, []*util.Int{&modulus, &value, &i}, 0, []util.Block{recovery})
	var later polynomialICCPInvocationReceipt
	serviceMust(t, full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &later))
	if oldContext.iccpConstant.invocation != later.shortcutConstant.invocation || (oldContext.roots(c.integer)) != nil {
		t.Fatal("saved mutable receipt pointer retargeted captured allocation nonce")
	}
	if later.shortcutConstant.invocation == child.shortcutConstant.invocation || full.iccpInvocationSourceCause(&child) != nil || !errors.Is(first, cause) || i != util.NewInt(9) {
		t.Fatal("later actual SET retargeted old returned cause", first, cause, i)
	}
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal(e)
	}
}
