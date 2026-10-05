// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/mpz.h:393-407. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// This uses real public signed-host TLS operations and the actually called
// AddMul end DEL. It is a guarded owner-primitive profile, not ordinary OOM.
func TestUPolynomialMignotteActualAddMulEndStopsOlderNUM(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	var a, warm Int
	defer func() {
		recovery, e := rt.Allocate(400000)
		serviceMust(t, e)
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, func() error { return q.z.Del(&a) })
		contextTLSFinish(t, rt, func() error { return q.z.Del(&warm) })
		contextTLSFinish(t, rt, u.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		e = rt.Deallocate(&recovery)
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal(e)
		}
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("named graph refund", b.Used())
		}
	}()
	_, e = s.newSquareFreeScope()
	serviceMust(t, e)
	r, e := s.beginMignotteInvocation()
	serviceMust(t, e)
	r.frame.constructed = 1
	digits := make([]uint32, 64)
	digits[0] = 17
	digits[63] = 1
	serviceMust(t, q.z.SetLimbs(&a, digits))
	product := make([]uint32, 64)
	product[0] = 34
	product[63] = 2
	serviceMust(t, q.z.SetLimbs(&r.frame.norm, product))
	serviceMust(t, q.z.Neg(&r.frame.norm))
	prime := NewInt(251)
	serviceMust(t, q.z.Rem(&a, &prime, &warm))
	serviceMust(t, q.z.ensureTmp(0, 40000))
	contextTLSNearMinimum(t, rt)
	two := NewInt(2)
	body := s.mignotteNumericCall(&r, 0, &two, &a, &r.frame.norm, 0)
	r.frame.executing = false
	mignotteScopePhysicalAudit(t, q, s, &a, &warm)
	if !errors.Is(body, arena.ErrBounds) || r.frame.addMul.frame == nil || r.frame.addMul.frame.stage != 3 || r.frame.norm != NewInt(0) {
		t.Fatal("actual AddMul output before explicit DEL", body, r.frame.norm)
	}
	stop := s.finishMignotteSource(&r, body)
	if !errors.Is(stop, body) || r.frame.sourceRemaining != 1 {
		t.Fatal("younger AddMul stop drained older NUM", stop, r.frame)
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return s.cleanupMignotteInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupMignotteInvocation(&r, true) })
	if s.mignotteInvocationSourceCause(&r) == nil || !errors.Is(stop, body) {
		t.Fatal("owning erased source cause")
	}
	mignotteScopePhysicalAuditExtra(t, q, s, []memory.Block{recovery}, &a, &warm)
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal(e)
	}
}
