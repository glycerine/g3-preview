// Source: src/math/polynomial/polynomial.cpp:6978-6980;
// src/nlsat/nlsat_solver.cpp:2467-2485; src/util/mpn.cpp:206-217.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// DEBUG verification has an actual scoped temporary only in checked builds.
// Signed-host TLS qualifies the reached physical destructor stop, not math OOM.
func TestPolynomialVariableRecognitionYoungerPNStopAndCallerNUMHistory(t *testing.T) {
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
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var modulus, value, n util.Int
	digits := make([]uint32, 17)
	for i := range digits {
		digits[i] = ^uint32(0)
	}
	digits[16] = 511
	serviceMust(t, c.integer.SetLimbs(&modulus, digits))
	digits = make([]uint32, 17)
	digits[0], digits[16] = 7, 1024
	serviceMust(t, c.integer.SetLimbs(&value, digits))
	term, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{util.NewInt(1), value}, []Monomial{term, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZP(&modulus))
	v := uint32(77)
	yes, e := full.IsVariableNumeral(p, &v, &n)
	serviceMust(t, e)
	if !yes || v != 0 || n != util.NewInt(9) {
		t.Fatal("actual ordinary SET warm", yes, v, n)
	}
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	var recovery util.Block
	defer func() {
		extra, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, full.CleanupVariableNumeral)
		contextTLSFinish(t, rt, func() error { return c.DecRef(p) })
		for _, a := range []*util.Int{&modulus, &value, &n} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, full.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		for _, b := range []*util.Block{&external, &recovery, &extra} {
			if *b != (util.Block{}) {
				x := rt.Deallocate(b)
				if x != nil && (!errors.Is(x, arena.ErrBounds) || *b != (util.Block{})) {
					t.Fatal(x)
				}
			}
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("actual named owner residue", budget.Used())
		}
	}()
	contextTLSNearMinimum(t, rt)
	if x := rt.Deallocate(&external); !errors.Is(x, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual arm release", x)
	}
	var receipt polynomialVariableNumeralInvocationReceipt
	yes, returned := c.isVariableNumeralWithReceipt(p, &v, &n, &receipt)
	s := &c.variableNumeral
	cause := s.sourceCause
	if yes || returned == nil || cause == nil || !errors.Is(returned, cause) || v != 0 || n == (util.Int{}) || c.modular.PNormalizeInvocationSourceCause(&s.normalize) != cause {
		t.Fatal("actual younger PN DT did not preserve reached variable/caller NUM", yes, returned, v, n, cause)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&modulus, &value, &n}, 0)
	before := *s
	state := rt.State()
	var refused polynomialVariableNumeralInvocationReceipt
	_, again := c.isVariableNumeralWithReceipt(p, &v, &n, &refused)
	if !errors.Is(again, ErrPolynomialPending) || refused != (polynomialVariableNumeralInvocationReceipt{}) || *s != before || rt.State() != state {
		t.Fatal("retained invocation acquired/mutated new SET", again)
	}
	text := returned.Error()
	pn := s.normalize
	recovery, e = rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return c.cleanupVariableNumeralInvocation(&receipt, false) })
	if s.sourceCause != cause || c.modular.PNormalizeInvocationSourceCause(&pn) != cause {
		t.Fatal("false ACK lost younger child/source history")
	}
	// n is the external caller's actual scoped NUM. It is not a helper local.
	contextTLSFinish(t, rt, func() error { return c.integer.Del(&n) })
	contextTLSFinish(t, rt, func() error { return c.cleanupVariableNumeralInvocation(&receipt, true) })
	old, ok := c.variableNumeralDiagnostic(&receipt)
	if !ok || old.sourceCause != cause {
		t.Fatal("own ACK erased actual cause")
	}
	var later polynomialVariableNumeralInvocationReceipt
	yes, e = c.isVariableNumeralWithReceipt(p, &v, &n, &later)
	serviceMust(t, e)
	after, ok := c.variableNumeralDiagnostic(&receipt)
	if !yes || n != util.NewInt(9) || !ok || after != old || later.invocation == receipt.invocation || returned.Error() != text || !errors.Is(returned, cause) {
		t.Fatal("later call changed earlier immutable error/tuple", yes, e, n)
	}
	beforeLater := *later.frame
	serviceMust(t, c.cleanupVariableNumeralInvocation(&receipt, true))
	if *later.frame != beforeLater {
		t.Fatal("stale own ACK changed later invocation")
	}
}
