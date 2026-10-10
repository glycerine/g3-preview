// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/mpz.cpp:1942-2021.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// Signed-host accounting is a guarded owner-primitive profile, not ordinary
// mathematical failure reachability. Root itself is actually called and stamps
// the parent's child receipt at its guarded begin; no receipt is reconstructed.
func TestUPolynomialMignotteActualRootEndStopsOlderNUM(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, u.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual refund", b.Used())
		}
	}()
	_, e = s.newSquareFreeScope()
	serviceMust(t, e)
	r, e := s.beginMignotteInvocation()
	serviceMust(t, e)
	f := r.frame
	f.constructed = 1
	serviceMust(t, q.Integers().Copy(&f.norm, &q.Integers().Two64))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&f.norm, 40000))
	contextTLSNearMinimum(t, rt)
	body := s.mignotteNumericCall(&r, 1, nil, nil, &f.norm, 2)
	f.executing = false
	mignotteScopePhysicalAudit(t, q, s)
	if !errors.Is(body, arena.ErrBounds) || !q.Integers().RootRaw.EndEntered || q.Integers().RootInvocationSourceCause(&f.root) == nil {
		t.Fatal("actual reached Root end DEL required", body, q.Integers().RootRaw.EndEntered, q.Integers().RootRaw.SourceIndex, rt.State())
	}
	older := f.norm
	stop := s.finishMignotteSource(&r, body)
	if !errors.Is(stop, body) || f.sourceRemaining != 1 || f.norm != older {
		t.Fatal("younger actual Root source stop drained older norm", stop, f)
	}
	contextTLSFinish(t, rt, func() error { return s.cleanupMignotteInvocation(&r, false) })
	contextTLSFinish(t, rt, func() error { return s.cleanupMignotteInvocation(&r, true) })
	if s.mignotteInvocationSourceCause(&r) == nil || !errors.Is(stop, body) {
		t.Fatal("owning changed immutable child history")
	}
	mignotteScopePhysicalAudit(t, q, s)
}
