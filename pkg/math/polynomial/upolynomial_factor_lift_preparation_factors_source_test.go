// Source: src/math/polynomial/upolynomial.cpp:33-54;
// src/math/polynomial/upolynomial_factorization.cpp:1153-1175.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

// Qualified compiled lexical-storage profile: a real completed caller owns
// the factors; a valid BIG codec/capacity reaches the actual source NUM release.
func TestUPolynomialFactorLiftPreparationFactorsSourceCursorAndOwnHistory(t *testing.T) {
	for _, site := range []string{"row", "reset", "own"} {
		t.Run(site, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			z, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
			serviceMust(t, e)
			primes, e := primegen.NewGenerator(b)
			serviceMust(t, e)
			s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
			serviceMust(t, e)
			out, e := m.NewFactors()
			serviceMust(t, e)
			defer func() {
				b.FailAfter(-1)
				contextTLSFinish(t, rt, s.CleanupRetained)
				contextTLSFinish(t, rt, out.Close)
				contextTLSFinish(t, rt, s.Close)
				serviceMust(t, primes.Close())
				contextTLSFinish(t, rt, m.Close)
				contextTLSFinish(t, rt, z.CloseChecked)
				contextTLSFinish(t, rt, rt.Close)
				if b.Used() != 0 {
					t.Fatal("actual class refunds", b.Used())
				}
			}()
			c, e := s.newSquareFreeScope()
			serviceMust(t, e)
			c.constructed = 1
			serviceMust(t, m.set(&c.fpp, []unumeral.Int{unumeral.NewInt(-6), unumeral.NewInt(1), unumeral.NewInt(1)}))
			serviceMust(t, s.preparePrimeSearch(out))
			serviceMust(t, m.modular.SetInt32(&c.p, 7))
			serviceMust(t, c.zp.SetZP(&c.p))
			trials := uint32(0)
			_, e = s.factorPrimeTrial(c.zp, &c.fpp, out, c.best, 1, &c.p, &c.bestP, &c.ds, &trials)
			serviceMust(t, e)
			serviceMust(t, s.prepareFactorLift(out))
			fs := c.lifted
			prep := &c.primePreparation
			target := &fs.constant
			if site != "reset" {
				target, e = c.zpe.read(c.zpe.stateView(&fs.factors[0]), 0)
				serviceMust(t, e)
			}
			serviceMust(t, z.Integers().Copy(target, &z.Integers().Two64))
			serviceMust(t, z.Integers().EnsureOwnedCapacity(target, 40000))
			olderCore, olderBest, olderPrime := c.zpe, c.best, c.bestP
			contextTLSNearMinimum(t, rt)
			before := rt.Stats().LiveBlocks
			first := c.closePreparedFactors(fs, 1, site != "own")
			if !errors.Is(first, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || c.constructed != 9 || c.zpe != olderCore || c.best != olderBest || c.bestP != olderPrime {
				t.Fatal("actual class first release/caller suffix", first, c.constructed)
			}
			if site == "own" {
				if prep.ownCause != first || prep.sourceCause != nil || prep.factorOwn[1].row != 0 || prep.factorOwn[1].column != 0 {
					t.Fatal("actual first own history", prep.ownCause)
				}
			} else {
				cursor := prep.factorSource[1]
				if prep.sourceCause != first || site == "row" && (cursor.stage != 0 || cursor.row != 0 || cursor.column != 0) || site == "reset" && cursor.stage != 2 {
					t.Fatal("actual source cursor", first, cursor)
				}
				if retry := c.closePreparedFactors(fs, 1, true); retry != first || prep.factorSource[1] != cursor || c.constructed != 9 {
					t.Fatal("source retry replayed stopped class suffix", retry, prep.factorSource[1], cursor)
				}
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{out})
			recovery, e := rt.Allocate(400000)
			serviceMust(t, e)
			contextTLSFinish(t, rt, func() error { return prep.own(&prep.receipt, false) })
			contextTLSFinish(t, rt, s.CleanupRetained)
			if site == "own" && prep.ownCause != first || site != "own" && prep.sourceCause != first {
				t.Fatal("own acknowledgment erased first class history")
			}
			primeTrialPhysicalAuditAll(t, z, s, []*UPolynomialFactors{out}, []umemory.Block{recovery})
			e = rt.Deallocate(&recovery)
			if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (umemory.Block{})) {
				t.Fatal(e)
			}
		})
	}
}
