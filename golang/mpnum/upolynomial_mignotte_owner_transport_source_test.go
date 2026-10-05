// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/memory_manager.cpp:274-285. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// ErrorTransport is an actual supported Go callback profile; this does not
// claim a native callback at the C++ allocation exception.
func TestUPolynomialMignotteTransportCannotRetireExecutingOwner(t *testing.T) {
	for _, mode := range []string{"service", "upm", "source"} {
		t.Run(mode, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			var s *UPolynomialFactorizationService
			var u *UPolynomialManager
			var foreign Int
			armed := false
			calls := 0
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
				if armed {
					calls++
					// This actual nil-context Copy publishes into an independently caller-owned
					// BIG destination with sufficient existing capacity; it allocates no storage.
					serviceMust(t, u.config.Integer.Copy(&foreign, &u.config.Integer.two64))
					before := rtMignotteOwnerSnapshot(u, s)
					var stop error
					if mode == "service" {
						stop = s.CleanupRetained()
					} else if mode == "source" {
						stop = u.SourceFinalize()
					} else {
						stop = u.Close()
					}
					if stop != ErrUPolynomialCleanupPending || rtMignotteOwnerSnapshot(u, s) != before {
						t.Errorf("executing owner changed in transport: mode=%s error=%v before=%+v after=%+v", mode, stop, before, rtMignotteOwnerSnapshot(u, s))
					}
				}
				return f
			}})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := NewRationalManager(RationalConfig{Integers: cfg})
			serviceMust(t, e)
			u, e = NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
			serviceMust(t, e)
			primes, e := NewPrimeGenerator(b)
			serviceMust(t, e)
			s, e = NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes})
			serviceMust(t, e)
			defer func() {
				armed = false
				b.FailAfter(-1)
				serviceMust(t, rt.SetMaxSize(0))
				serviceMust(t, s.Close())
				serviceMust(t, primes.Close())
				if !u.closed {
					serviceMust(t, u.Close())
				}
				serviceMust(t, q.CloseChecked())
				serviceMust(t, rt.Close())
				if b.Used() != 0 {
					t.Error("named refund", b.Used())
				}
			}()
			var big Int
			serviceMust(t, q.z.SetLimbs(&big, []uint32{1, 0, 1}))
			words := make([]uint32, 80)
			words[79] = 1
			serviceMust(t, q.z.SetLimbs(&foreign, words))
			defer func() { serviceMust(t, q.z.Del(&foreign)) }()
			defer func() { serviceMust(t, q.z.Del(&big)) }()
			_, e = s.newSquareFreeScope()
			serviceMust(t, e)
			r, e := s.beginMignotteInvocation()
			serviceMust(t, e)
			r.frame.constructed = 3
			pad := int64(99996) - rt.State().LocalSize - 12
			if pad <= 0 {
				t.Fatal("real threshold", rt.State())
			}
			held, e := rt.Allocate(uint64(pad))
			serviceMust(t, e)
			defer func() { serviceMust(t, rt.Deallocate(&held)) }()
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := s.mignotteNumericCall(&r, 4, &big, nil, &r.frame.tmp, 0)
			armed = false
			r.frame.executing = false
			if !numericPhysicalReservationRefusal(body) || calls != 1 {
				t.Fatal("actual allocation transport absent", body, calls, rt.State())
			}
			mignotteScopePhysicalAuditExtra(t, q, s, []memory.Block{held}, &big, &foreign)
			if !polynomialNegNumericPending(&r.frame.numeric) {
				t.Fatal("real returning allocation lost exact pending receipt")
			}
			_ = s.finishMignotteSource(&r, body)
			serviceMust(t, s.cleanupMignotteInvocation(&r, false))
			serviceMust(t, s.cleanupMignotteInvocation(&r, true))
			mignotteScopePhysicalAuditExtra(t, q, s, []memory.Block{held}, &big, &foreign)
			same, e := q.z.Equal(&foreign, &q.z.two64)
			serviceMust(t, e)
			if !same {
				t.Fatal("owning consumed callback foreign numeral")
			}
			serviceMust(t, rt.SetMaxSize(0))
		})
	}
}

type rtMignotteOwnerState struct {
	closing, closed                   bool
	sourceFinalizing, sourceFinalized bool
	sourceStage                       uint8
	scope                             *upolyFactorSquareFreeScope
	active, executing                 bool
	invocation                        uint64
}

func rtMignotteOwnerSnapshot(u *UPolynomialManager, s *UPolynomialFactorizationService) rtMignotteOwnerState {
	v := rtMignotteOwnerState{closing: u.closing, closed: u.closed, scope: s.scope, sourceFinalizing: u.sourceFinalizing, sourceFinalized: u.sourceFinalized, sourceStage: u.sourceStage}
	if s.scope != nil {
		v.active = s.scope.mignotte.active
		v.executing = s.scope.mignotte.executing
		v.invocation = s.scope.mignotte.invocation
	}
	return v
}
