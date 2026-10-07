// Source: src/math/polynomial/upolynomial_factorization.cpp:1153-1175;
// src/util/mpzzp.h:122-123; src/util/mpz.h:698-707.
package polynomial

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorLiftPreparationSavedModulusCallerTransport(t *testing.T) {
	for _, mode := range []string{"service", "enclosing-source", "close", "source", "mode", "related-close", "related-source", "related-mode"} {
		t.Run(mode, func(t *testing.T) { liftPreparationSavedModulusTransport(t, mode) })
	}
}
func liftPreparationSavedModulusTransport(t *testing.T, mode string) {
	b := arena.NewBudget(1 << 26)
	var s *UPolynomialFactorizationService
	armed := false
	calls := 0
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f util.Failure) error {
		if armed {
			calls++
			prior := s.scope
			before := *prior
			var stop error
			switch mode {
			case "service":
				stop = s.CleanupRetained()
			case "enclosing-source":
				stop = s.finishSquareFreeScopeSource()
			case "related-close":
				stop = s.scope.zp.Close()
			case "related-source":
				stop = s.scope.zp.SourceFinalize()
			case "related-mode":
				stop = s.scope.zp.SetZ()
			case "close":
				stop = s.manager.Close()
			case "source":
				stop = s.manager.SourceFinalize()
			case "mode":
				stop = s.manager.SetZ()
			}
			if stop != ErrUPolynomialCleanupPending || s.scope != prior || !reflect.DeepEqual(*prior, before) {
				t.Errorf("actual saved-modulus restore changed older caller before refusal: %v", stop)
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := util.NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e = NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	var held util.Block
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var value util.Int
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return q.Integers().Del(&value) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual saved-modulus owner refunds", b.Used())
		}
	}()
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	parent.constructed = 1
	serviceMust(t, m.set(&parent.fpp, []util.Int{util.NewInt(1), util.NewInt(0), util.NewInt(1)}))
	serviceMust(t, s.preparePrimeSearch(fs))
	// Valid uint32 prime; compiled post-loop operand profile avoids an
	// impractical iterator search. Extra capacity does not change its value.
	serviceMust(t, q.Integers().SetUint64(&parent.bestP, 2147483647))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&parent.bestP, 80))
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual source threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := s.restoreSavedPrime(fs)
	armed = false
	if !numericPhysicalReservationRefusal(body) || calls != 1 {
		t.Fatal("actual saved-modulus source SET refusal", body, calls)
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []util.Block{held}, &value)
}

// The actual post-loop caller uses the literal saved-prime/Mignotte/related
// constructor/factors/Hensel sequence. Each refusal names the graph before
// any fault lift; successful lifted factors independently multiply to f_pp.
func TestUPolynomialFactorLiftPreparationEveryActualPrefix(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		done := false
		okay := t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			out, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { z.Budget().FailAfter(-1); serviceMust(t, s.CleanupRetained()); serviceMust(t, out.Close()) }()
			c, e := s.newSquareFreeScope()
			serviceMust(t, e)
			c.constructed = 1
			serviceMust(t, m.set(&c.fpp, []util.Int{util.NewInt(-6), util.NewInt(1), util.NewInt(1)}))
			serviceMust(t, s.preparePrimeSearch(out))
			// Compiled eligible post-loop prime profile, using the real trial engine.
			serviceMust(t, m.modular.SetInt32(&c.p, 7))
			serviceMust(t, c.zp.SetZP(&c.p))
			trials := uint32(0)
			route, e := s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
			serviceMust(t, e)
			if route != 1 || c.best.count != 2 || c.bestP != util.NewInt(7) {
				t.Fatal("actual saved-factor setup", route, c.best.count, c.bestP)
			}
			z.Budget().FailAfter(ordinal)
			body := s.prepareFactorLift(out)
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
			receipt := c.primePreparation.receipt
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual helper source refusal", body)
				}
				if c.constructed < 7 || c.constructed > 9 || c.primePreparation.bodyCause == nil {
					t.Fatal("actual declaration/cause frontier", c.constructed, body)
				}
			} else {
				if c.constructed != 9 || c.lifted == nil || c.lifted.count != 2 {
					t.Fatal("actual lifted declaration/result", c.constructed, c.lifted.Stats())
				}
				constants := [2]int64{}
				for i := uint32(0); i < 2; i++ {
					if c.lifted.factors[i].size != 2 {
						t.Fatal("independent linear factors")
					}
					coefficient, e := c.zpe.read(c.zpe.stateView(&c.lifted.factors[i]), 0)
					serviceMust(t, e)
					constants[i], e = c.zpe.modular.Int64(coefficient)
					serviceMust(t, e)
					leading, e := c.zpe.read(c.zpe.stateView(&c.lifted.factors[i]), 1)
					serviceMust(t, e)
					if *leading != util.NewInt(1) {
						t.Fatal("independent monic factor", *leading)
					}
				}
				if constants[0]+constants[1] != 1 || constants[0]*constants[1] != -6 {
					t.Fatal("independent lifted product", constants)
				}
				done = true
			}
			serviceMust(t, c.primePreparation.own(&receipt, false))
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
			z.Budget().FailAfter(-1)
			serviceMust(t, s.CleanupRetained())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
		})
		if !okay {
			return
		}
		if done {
			t.Log("first completed literal helper prefix", ordinal)
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("no completed literal lift preparation")
	}
}
