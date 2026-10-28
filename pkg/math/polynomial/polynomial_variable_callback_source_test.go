package polynomial

// Source: src/math/polynomial/polynomial.cpp:2538-2544;
// src/util/memory_manager.cpp:274-285.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// This is the returning Go ErrorTransport owner boundary, not a native source callback.
func TestPolynomialVariableActualTransportCannotRetireExecutingOwner(t *testing.T) {
	for _, operation := range []string{"receipt", "manager", "source"} {
		t.Run(operation, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			var full *MultivariatePolynomialManager
			var m *MultivariatePolynomialCore
			var receipt polynomialVariableInvocationReceipt
			active, seen := false, false
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
				if active {
					seen = true
					state := m.memory.State()
					used := b.Used()
					closing, stage := full.closing, full.cleanupStage
					started, sourceStage := m.sourceStarted, m.sourceStage
					var err error
					switch operation {
					case "receipt":
						err = m.cleanupVariableInvocation(&receipt, true)
					case "manager":
						err = full.CloseChecked()
					case "source":
						err = m.SourceFinalizePrefix()
					}
					if err != ErrPolynomialPending || state != m.memory.State() || used != b.Used() || closing != full.closing || stage != full.cleanupStage || started != m.sourceStarted || sourceStage != m.sourceStage {
						t.Errorf("executing owner callback mutated suffix: %v closing=%v stage=%d source=%v/%d", err, full.closing, full.cleanupStage, m.sourceStarted, m.sourceStage)
					}
				}
				return f
			}})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			m = full.core
			_, e = mm.MkVar()
			serviceMust(t, e)
			pad := int64(99996) - rt.State().LocalSize - 8
			if pad <= 0 {
				t.Fatal("actual sync frontier", rt.State())
			}
			held, e := rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			active = true
			p, cause := m.mkVariableWithReceipt(0, 7, &receipt)
			active = false
			if !seen || !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual returning transport absent", seen, cause)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p}, nil, 0, []umemory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
			serviceMust(t, m.cleanupVariableInvocation(&receipt, true))
			serviceMust(t, full.CloseChecked())
			serviceMust(t, q.CloseChecked())
			serviceMust(t, mm.CloseChecked())
			serviceMust(t, rt.Deallocate(&held))
			serviceMust(t, rt.Close())
			if b.Used() != 0 {
				t.Fatal("actual named owners retained", b.Used())
			}
		})
	}
}
