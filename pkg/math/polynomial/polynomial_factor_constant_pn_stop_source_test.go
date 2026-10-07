// Source: src/math/polynomial/polynomial.cpp:6577-6590;
// src/util/mpzzp.h:58-68; src/util/mpn.cpp:206-217.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// A real checked verification NUM destructor stop in the reached first SET/PN child.
// The public signed-host TLS guard domain is distinct from ordinary math OOM.
func TestPolynomialFactorConstantYoungerPNProtectsActualHelperNUM(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 27)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	var modulus, value util.Int
	digits := make([]uint32, 17)
	for i := range digits {
		digits[i] = ^uint32(0)
	}
	digits[16] = 511
	serviceMust(t, c.integer.SetLimbs(&modulus, digits))
	digits = make([]uint32, 17)
	digits[0], digits[16] = 7, 1024
	serviceMust(t, c.integer.SetLimbs(&value, digits))
	// Literal SET has no normalized-input assertion. Preserve the source-valid
	// Z constant across the mode switch; the first reached helper Copy performs
	// field normalization before NEG or the final constant SET.
	serviceMust(t, fs.SetConstant(&value))
	serviceMust(t, c.modular.SetZP(&modulus))
	var warm util.Int
	serviceMust(t, c.modular.Copy(&warm, &value))
	if warm != util.NewInt(9) {
		t.Fatal("ordinary actual SET/PN warm", warm)
	}
	serviceMust(t, c.integer.Del(&warm))
	prior := fs.constant
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	p, e := c.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, fs.Push(p, 1))
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	var recovery util.Block
	defer func() {
		extra, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, fs.CloseChecked)
		contextTLSFinish(t, rt, func() error { return c.DecRef(p) })
		for _, a := range []*util.Int{&modulus, &value} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, full.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		for _, raw := range []*util.Block{&external, &recovery, &extra} {
			if *raw != (util.Block{}) {
				x := rt.Deallocate(raw)
				if x != nil && (!errors.Is(x, arena.ErrBounds) || *raw != (util.Block{})) {
					t.Fatal(x)
				}
			}
		}
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual named owner retained", b.Used())
		}
	}()
	contextTLSNearMinimum(t, rt)
	if x := rt.Deallocate(&external); !errors.Is(x, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual arm release", x)
	}
	var receipt polynomialFactorConstantInvocationReceipt
	returned := full.flipFactorSignWithReceipt(fs, &receipt)
	s := &fs.constantInvocation
	cause := s.sourceCause
	if returned == nil || cause == nil || !errors.Is(returned, cause) || s.phase != 0 || !s.localConstructed || !fs.localActive || fs.constant != prior || c.modular.PNormalizeInvocationSourceCause(&s.normalize[0]) != cause {
		t.Fatal("youngest actual PN stop did not protect helper NUM/older SET", returned, cause, s.phase, fs.local, fs.constant)
	}
	polynomialFactorsAudit(t, fs, []Polynomial{p}, []*util.Int{&modulus, &value})
	classState := rtStateForFactors(fs)
	rows, e := fs.factors.size(c)
	serviceMust(t, e)
	reset := fs.resetIndex
	runtimeState := rt.State()
	againSource := fs.SourceFinalize()
	afterRows, x := fs.factors.size(c)
	serviceMust(t, x)
	if againSource != cause || rtStateForFactors(fs) != classState || afterRows != rows || fs.resetIndex != reset || rt.State() != runtimeState {
		t.Fatal("younger helper source stop must protect enclosing Factors suffix", againSource)
	}
	polynomialFactorsAudit(t, fs, []Polynomial{p}, []*util.Int{&modulus, &value})
	before := *s
	state := rt.State()
	var refused polynomialFactorConstantInvocationReceipt
	again := full.flipFactorSignWithReceipt(fs, &refused)
	if !errors.Is(again, ErrPolynomialPending) || refused.invocation != 0 || !testNumericScopeEqual(*s, before) || rt.State() != state {
		t.Fatal("retained helper mutated at rejected begin", again)
	}
	text := returned.Error()
	pn := s.normalize[0]
	recovery, e = rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return fs.cleanupFactorConstantInvocation(&receipt, false) })
	if s.sourceCause != cause || c.modular.PNormalizeInvocationSourceCause(&pn) != cause || s.localConstructed || fs.localActive {
		t.Fatal("false ACK lost younger cause/helper NUM suffix")
	}
	contextTLSFinish(t, rt, func() error { return c.integer.Del(&fs.constant) })
	contextTLSFinish(t, rt, func() error { return fs.cleanupFactorConstantInvocation(&receipt, true) })
	one := util.NewInt(1)
	serviceMust(t, fs.SetConstant(&one))
	old, ok := fs.factorConstantDiagnostic(&receipt)
	if !ok || old.sourceCause != cause || returned.Error() != text || !errors.Is(returned, cause) {
		t.Fatal("actual later invocation erased immutable error/history")
	}
	fresh := fs.constantInvocation
	state = rt.State()
	serviceMust(t, fs.cleanupFactorConstantInvocation(&receipt, true))
	if !testNumericScopeEqual(fs.constantInvocation, fresh) || state != rt.State() {
		t.Fatal("stale old ACK touched later caller")
	}
}
