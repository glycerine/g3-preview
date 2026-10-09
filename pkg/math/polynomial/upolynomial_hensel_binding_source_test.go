package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func upolyHenselImportedFixture(t *testing.T) (*UPolynomialManager, *UPolynomialManager, *UPolynomialManager, *UPolynomialManager, *UPolynomialFactors, *UPolynomialFactors, *UPolynomialFactorizationService, UPolynomialView, UPolynomialView, *arena.Budget) {
	t.Helper()
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	config := UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }}
	cores := make([]*UPolynomialManager, 3)
	for i := range cores {
		var e error
		cores[i], e = NewUPolynomialManager(config)
		if e != nil {
			t.Fatal(e)
		}
		owner := cores[i]
		t.Cleanup(func() {
			if e := owner.Close(); e != nil {
				t.Error(e)
			}
		})
	}
	prime, lift, donor := cores[0], cores[1], cores[2]
	if e := prime.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	primes, e := primegen.NewGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	})
	service, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	})
	pf, e := prime.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := pf.Close(); e != nil {
			t.Error(e)
		}
	})
	lf, e := lift.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := lf.Close(); e != nil {
			t.Error(e)
		}
	})
	var captured UPolynomialView
	for i, cs := range [][]int32{{-1, 1}, {1, 1}, {2, 1}} {
		p, e := donor.NewPolynomial()
		if e != nil {
			t.Fatal(e)
		}
		values := []util.Int{util.NewInt(cs[0]), util.NewInt(cs[1])}
		if e = donor.SetInts(p, values); e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			captured = mustUPolyView(t, donor, p)
		}
		if e = pf.PushMoveFrom(donor, p, 1); e != nil {
			t.Fatal(e)
		}
		if e = donor.DelPolynomial(&p); e != nil {
			t.Fatal(e)
		}
	}
	f := upolySourceNew(t, m, []int32{-2, -1, 2, 1})
	return m, prime, lift, donor, pf, lf, service, mustUPolyView(t, m, f), captured, b
}

func TestUPolynomialHenselImportedBindingFailurePrefixes(t *testing.T) {
	// Source numerical actions start only after these three actual cold growth
	// reservations. Successful bindings are retained on subsequent refusal.
	expected := [][4]uint32{{1, 1, 1, 2}, {2, 2, 2, 2}, {3, 2, 2, 2}}
	for ordinal, want := range expected {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, captured, b := upolyHenselImportedFixture(t)
			cores := []*UPolynomialManager{m, prime, lift, donor}
			stores := make([]*UPolynomialCoefficientStore, 4)
			graph := func() uint64 {
				var n uint64
				for i, core := range cores {
					n += core.Stats().ExclusiveBytes
					ss, e := stores[i].Stats()
					if e != nil {
						t.Fatal(e)
					}
					n += ss.TotalBytes
				}
				return n
			}
			for i, core := range cores {
				var e error
				stores[i], e = core.CoefficientStore()
				if e != nil {
					t.Fatal(e)
				}
			}
			external := b.Used() - graph()
			beforePF, beforeLF := pf.Stats(), lf.Stats()
			beforeNumeric := m.Stats()
			b.FailAfter(int64(ordinal))
			err := service.HenselLiftFactors(f, pf, 3, lf)
			// Inspect actual owner graph before fault disable, formatting or cleanup.
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal(ordinal, err)
			}
			for i, store := range stores {
				ss, e := store.Stats()
				if e != nil || ss.Owners != want[i] {
					t.Fatal("retained cold prefix", ordinal, i, ss, e, want)
				}
			}
			if b.Used() != external+graph() {
				t.Fatal("once-only physical store graph", ordinal, b.Used(), external+graph())
			}
			if pf.Stats() != beforePF || lf.Stats() != beforeLF {
				t.Fatal("cold refusal changed factor ownership", pf.Stats(), lf.Stats())
			}
			immediate := m.Stats()
			if immediate.IntegerLiveCells != beforeNumeric.IntegerLiveCells || immediate.IntegerWordsRefs != beforeNumeric.IntegerWordsRefs {
				t.Fatal("cold prefix changed numeric roots", immediate, beforeNumeric)
			}
			v, e := pf.FactorView(0)
			if e != nil {
				t.Fatal(e)
			}
			if same, e := prime.EqualViews(captured, v); e != nil || !same {
				t.Fatal("actual imported backing identity", same, e)
			}
			b.FailAfter(-1)
		})
	}
}

func TestUPolynomialHenselImportedOriginAndCapturedView(t *testing.T) {
	m, prime, lift, _, pf, lf, service, f, captured, _ := upolyHenselImportedFixture(t)
	if e := service.HenselLiftFactors(f, pf, 3, lf); e != nil {
		t.Fatal(e)
	}
	v, e := pf.FactorView(0)
	if e != nil {
		t.Fatal(e)
	}
	if same, e := m.EqualViews(captured, v); e != nil || !same {
		t.Fatal("main adopted actual nonprimary factor origin", same, e)
	}
	if same, e := lift.EqualViews(captured, v); e != nil || !same {
		t.Fatal("lift adopted actual nonprimary factor origin", same, e)
	}
	if same, e := prime.EqualViews(captured, v); e != nil || !same {
		t.Fatal("unchanged borrowed physical view", same, e)
	}
	out := upolySourceNew(t, lift, nil)
	if e = lf.Multiply(out); e != nil {
		t.Fatal(e)
	}
	if same, e := lift.EqualViews(f, mustUPolyView(t, lift, out)); e != nil || !same {
		t.Fatal("three actual lifted factors reconstruct original", same, e)
	}
	if n, e := lf.DistinctFactors(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
}

func TestUPolynomialHenselImportedAlgorithmImmediateFailureRoots(t *testing.T) {
	failures, successes, late := 0, 0, 0
	firstPrimeRetain := int64(-1)
	var previousPrimeOwners uint32
	var previousLiftExclusive uint64
	previousLiftHadPrime := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		succeeded := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, prime, lift, donor, pf, lf, service, f, _, b := upolyHenselImportedFixture(t)
			cores := []*UPolynomialManager{m, prime, lift, donor}
			b.FailAfter(ordinal)
			err := service.HenselLiftFactors(f, pf, 3, lf)
			upolyHenselAuditPhysical(t, cores, []*UPolynomialFactors{pf, lf})
			if err != nil && !upolyHenselAllocationRefusal(m.IntegerManager(), err) {
				t.Fatal(ordinal, err)
			}
			if err != nil {
				failures++
				mainStore, e := m.CoefficientStore()
				if e != nil {
					t.Fatal(e)
				}
				ss, e := mainStore.Stats()
				if e != nil {
					t.Fatal(e)
				}
				if ss.Owners == 3 && ordinal > 2 {
					late++
				}
			}
			primeStore, e := prime.CoefficientStore()
			if e != nil {
				t.Fatal(e)
			}
			ps, e := primeStore.Stats()
			if e != nil {
				t.Fatal(e)
			}
			liftHasPrime := false
			liftBindings := testFieldBySourceName(reflect.ValueOf(lift).Elem(), "bindings")
			primeIdentity := testFieldBySourceName(reflect.ValueOf(prime).Elem(), "store").Pointer()
			for i := 0; i < liftBindings.Len(); i++ {
				if testFieldBySourceName(liftBindings.Index(i), "store").Pointer() == primeIdentity {
					liftHasPrime = true
				}
			}
			if firstPrimeRetain < 0 && liftHasPrime {
				firstPrimeRetain = ordinal
				field, ok := testTypeFieldBySourceName(reflect.TypeOf(prime).Elem(), "bindings")
				if !ok {
					t.Fatal("actual cold directory")
				}
				bindingBytes := uint64(field.Type.Elem().Size())
				if previousLiftHadPrime || ps.Owners < 3 || lift.Stats().ExclusiveBytes != previousLiftExclusive+2*bindingBytes {
					t.Fatal("actual lifted PRIME binding growth transition", ordinal, previousPrimeOwners, ps, previousLiftExclusive, lift.Stats())
				}
			}
			previousPrimeOwners = ps.Owners
			previousLiftHadPrime = liftHasPrime
			previousLiftExclusive = lift.Stats().ExclusiveBytes
			b.FailAfter(-1)
			if err == nil {
				successes++
				succeeded = true
				out := upolySourceNew(t, lift, nil)
				if e := lf.Multiply(out); e != nil {
					t.Fatal(e)
				}
				if same, e := lift.EqualViews(f, mustUPolyView(t, lift, out)); e != nil || !same {
					t.Fatal(same, e)
				}
			}
		})
		if succeeded {
			break
		}
	}
	if failures < 4 || late == 0 || successes != 1 || firstPrimeRetain < 1 {
		t.Fatal("must cover cold prefixes, numerical refusals and success", failures, late, successes)
	}
	t.Log("actual imported Hensel refusal endpoints", failures, "after cold phase", late, "successes", successes, "actual-read binding refusal ordinal", firstPrimeRetain-1)
}
