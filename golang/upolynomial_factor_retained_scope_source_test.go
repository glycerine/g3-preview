// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func TestUPolynomialFactorRetainedLexicalCleanup(t *testing.T) {
	// The allocation ordinal depends on the actual source storage profile. Find
	// the first genuine suspended lexical owner, rather than forge a dead bound.
	found := false
	directDebug := false
	for ordinal := int64(0); ordinal < 2048; ordinal++ {
		m, _, b := upolySourceOwner(t)
		profile := reflect.ValueOf(m.IntegerManager()).Elem().FieldByName("allocator").Elem().FieldByName("config").FieldByName("Profile")
		directDebug = profile.FieldByName("Debug").Bool() && !profile.FieldByName("Windows").Bool()
		primes, e := mpnum.NewPrimeGenerator(b)
		if e != nil {
			t.Fatal(e)
		}
		service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
		if e != nil {
			t.Fatal(e)
		}
		fs, e := m.NewFactors()
		if e != nil {
			t.Fatal(e)
		}
		p := upolySourceNew(t, m, nil)
		input := make([]mpnum.Int, 4)
		for i, text := range []string{"-24691357802469135780", "-37037036703703703668", "-12345678901234567887", "1"} {
			if e = m.IntegerManager().SetString(&input[i], text); e != nil {
				t.Fatal(e)
			}
		}
		if e = m.SetInts(p, input); e != nil {
			t.Fatal(e)
		}
		for i := range input {
			if e = m.IntegerManager().Del(&input[i]); e != nil {
				t.Fatal(e)
			}
		}
		b.FailAfter(ordinal)
		_, failure := service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams())
		immediate := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}})
		stats := service.Stats()
		if stats.RetainedScopes != 0 {
			if stats.RetainedOwnerBytes == 0 {
				t.Fatal("uncharged lexical scope")
			}
			core, e := service.RetainedCleanupCore(0)
			if errors.Is(e, arena.ErrBounds) && (stats.MignottePending || stats.PrimeTrialPending) {
				// Source completed older declarations, leaving an exact unjoined
				// Mignotte or Prime RAW child. A related core is no longer a universal scope invariant.
				// This new case must not replace the original later core/set_z campaign.
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact retained child owning acknowledgment", e)
				}
				if service.Stats().RetainedScopes != 0 {
					t.Fatal("unjoined retained child scope not consumed")
				}
				if e = fs.Close(); e != nil {
					t.Fatal(e)
				}
				if e = service.Close(); e != nil {
					t.Fatal(e)
				}
				if e = primes.Close(); e != nil {
					t.Fatal(e)
				}
				continue
			}
			if e != nil {
				t.Fatal(e)
			}
			found = true
			if _, e = service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams()); !errors.Is(e, mpnum.ErrUPolynomialCleanupPending) {
				t.Fatal("new arithmetic accepted", e)
			}
			if e = service.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("guarded lexical cleanup", e)
			}
			if service.Stats() != stats {
				t.Fatal("cleanup refusal discarded charged context")
			}
			if after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}}); after != immediate {
				t.Fatal("refused teardown changed retained physical roots", immediate, after)
			}
			// Caller performs the actual source set_z operation, separately from
			// failed arithmetic/cleanup. No bound or published output is repaired.
			if e = core.SetZ(); e != nil {
				t.Fatal(e)
			}
			if after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}}); after != immediate {
				t.Fatal("set_z repaired or replaced retained roots", immediate, after)
			}
			if e = service.CleanupRetained(); e != nil {
				t.Fatal(e)
			}
			if service.Stats().RetainedScopes != 0 {
				t.Fatal("scope not consumed")
			}
		}
		b.FailAfter(-1)
		if e = fs.Close(); e != nil {
			t.Fatal(e)
		}
		if e = service.Close(); e != nil {
			t.Fatal(e)
		}
		if e = primes.Close(); e != nil {
			t.Fatal(e)
		}
		if found || failure == nil {
			break
		}
	}
	if directDebug && !found {
		t.Fatal("debug direct profile did not exercise retained cleanup")
	}
}
