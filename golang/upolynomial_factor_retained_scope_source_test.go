// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1299.
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
			pair := reflect.ValueOf(service).Elem().FieldByName("henselPairs")
			preRelatedPair := stats.HenselPending && !pair.IsNil() && pair.Elem().FieldByName("active").Bool() && pair.Elem().FieldByName("constructed").Uint() == 0 && pair.Elem().FieldByName("rm").IsNil()
			if errors.Is(e, arena.ErrBounds) && (stats.MignottePending || stats.PrimeTrialPending || preRelatedPair) {
				// Source completed older declarations, leaving an exact unjoined
				// Mignotte or Prime RAW child, or the exact pair constructor
				// frontier before the related manager is published.
				// This new case must not replace the original later core/set_z campaign.
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact retained child owning acknowledgment", e)
				}
				if preRelatedPair && (pair.Elem().FieldByName("active").Bool() || service.Stats().CachedOwnerBytes != stats.CachedOwnerBytes || service.Stats().CachedScopes != stats.CachedScopes || service.Stats().HenselPending) {
					t.Fatal("exact pre-related invocation not retired into charged idle cache", service.Stats())
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
			if errors.Is(e, arena.ErrBounds) && stats.RecombinationPending {
				enclosing := reflect.ValueOf(service).Elem().FieldByName("scope")
				step := enclosing.Elem().FieldByName("recombinationStep")
				receipt := step.FieldByName("receipt")
				nonce := step.FieldByName("invocation").Uint()
				bodyNil, sourceNil, ownNil := step.FieldByName("bodyCause").IsNil(), step.FieldByName("sourceCause").IsNil(), step.FieldByName("ownCause").IsNil()
				if enclosing.IsNil() || enclosing.Elem().FieldByName("constructed").Uint() != 0 || !step.FieldByName("active").Bool() || nonce == 0 || bodyNil || failure == nil || receipt.FieldByName("owner").Pointer() != enclosing.Pointer() || receipt.FieldByName("frame").Pointer() != step.Addr().Pointer() || receipt.FieldByName("invocation").Uint() != nonce {
					t.Fatal("actual retired older fields/exact recombination tuple", ordinal, failure)
				}
				// These are already recorded real acquisition fields, not
				// inferred ownership from Stats or graph cardinality.
				actualRoots := !step.FieldByName("numeric").IsZero()
				for _, name := range []string{"divisionRaw", "extendedRaw", "gcdRaw", "boolNumeral", "addMul", "verification"} {
					frame := step.FieldByName(name).FieldByName("frame")
					if !frame.IsNil() && frame.Elem().FieldByName("active").Bool() {
						actualRoots = true
					}
				}
				if !actualRoots {
					t.Fatal("no exact retained recombination child root", ordinal)
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams()); !errors.Is(e, mpnum.ErrUPolynomialCleanupPending) {
					t.Fatal("exact recombination refused new caller", e)
				}
				if after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats || step.FieldByName("invocation").Uint() != nonce {
					t.Fatal("rejected caller changed exact retained graph")
				}
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact recombination false/older/true acknowledgment", e)
				}
				if service.Stats().RetainedScopes != 0 || step.FieldByName("active").Bool() || step.FieldByName("invocation").Uint() != nonce || step.FieldByName("bodyCause").IsNil() != bodyNil || step.FieldByName("sourceCause").IsNil() != sourceNil || step.FieldByName("ownCause").IsNil() != ownNil {
					t.Fatal("own acknowledgment changed original recombination history", ordinal)
				}
				upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}})
				if e = fs.Close(); e != nil {
					t.Fatal(e)
				}
				if e = service.Close(); e != nil {
					t.Fatal(e)
				}
				if e = primes.Close(); e != nil {
					t.Fatal(e)
				}
				continue // Preserve the original later core.SetZ/found search.
			}

			if errors.Is(e, arena.ErrBounds) && stats.HenselPending {
				parent := reflect.ValueOf(service).Elem().FieldByName("scope")
				caller := parent.Elem().FieldByName("hensel")
				prep := parent.Elem().FieldByName("primePreparation")
				nonce := caller.FieldByName("invocation").Uint()
				message := failure.Error()
				bodyNil, sourceNil, ownNil := caller.FieldByName("bodyCause").IsNil(), caller.FieldByName("sourceCause").IsNil(), caller.FieldByName("ownCause").IsNil()
				if parent.Elem().FieldByName("constructed").Uint() != 0 || !caller.FieldByName("active").Bool() || nonce == 0 || bodyNil || !sourceNil || caller.FieldByName("owner").Pointer() != parent.Pointer() || prep.FieldByName("hensel").Pointer() != caller.Addr().Pointer() || prep.FieldByName("henselInvocation").Uint() != nonce || caller.FieldByName("outerSource").Uint() != 0 || caller.FieldByName("quadraticSource").Uint() != 0 || caller.FieldByName("iterationSource").Uint() != 0 {
					t.Fatal("exact completed-source retained Hensel caller", ordinal, failure)
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams()); !errors.Is(e, mpnum.ErrUPolynomialCleanupPending) {
					t.Fatal("exact caller refused reuse", e)
				}
				if after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats {
					t.Fatal("rejected caller changed completed-source ownership")
				}
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact completed-child false/older/true owning acknowledgment", ordinal, e)
				}
				if service.Stats().RetainedScopes != 0 || caller.FieldByName("active").Bool() || caller.FieldByName("invocation").Uint() != nonce || caller.FieldByName("bodyCause").IsNil() != bodyNil || caller.FieldByName("sourceCause").IsNil() != sourceNil || caller.FieldByName("ownCause").IsNil() != ownNil || failure.Error() != message {
					t.Fatal("completed-source retirement changed original histories")
				}
				upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}})
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
			enclosing := reflect.ValueOf(service).Elem().FieldByName("scope")
			matchedCaller := !enclosing.IsNil() && enclosing.Elem().FieldByName("hensel").FieldByName("active").Bool()
			if matchedCaller {
				// Close is explicitly owning CleanupRetained, not a source retry.
				// This exact newly captured child can now be rescued. The original
				// source/caller-set_z assertions below remain for the old scope route.
				caller := enclosing.Elem().FieldByName("hensel")
				callerCauseNil := caller.FieldByName("sourceCause").IsNil()
				var child reflect.Value
				var invocation, sourceRemaining, sourceIndex uint64
				causeNil, bodyNil := true, true
				if !pair.IsNil() {
					child = pair.Elem()
					invocation = child.FieldByName("invocation").Uint()
					sourceRemaining = child.FieldByName("sourceRemaining").Uint()
					sourceIndex = child.FieldByName("sourceIndex").Uint()
					causeNil = child.FieldByName("sourceCause").IsNil()
					bodyNil = child.FieldByName("bodyCause").IsNil()
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams()); !errors.Is(e, mpnum.ErrUPolynomialCleanupPending) {
					t.Fatal("matched caller refused new operation", e)
				}

				var sourceCursor [3]uint64
				sourceClass := enclosing.Elem().FieldByName("primePreparation")
				sourceClassNonce := sourceClass.FieldByName("invocation").Uint()
				sourceClassReached := errors.Is(failure, arena.ErrInvalidSlot)
				sourceMessage := failure.Error()
				if sourceClassReached {
					cursor := sourceClass.FieldByName("factorSource").Index(1)
					sourceCursor = [3]uint64{cursor.FieldByName("stage").Uint(), cursor.FieldByName("row").Uint(), cursor.FieldByName("column").Uint()}
					if enclosing.Elem().FieldByName("constructed").Uint() != 9 || sourceClass.FieldByName("sourceCause").IsNil() || sourceClassNonce == 0 || sourceClass.FieldByName("owner").Pointer() != enclosing.Pointer() || sourceClass.FieldByName("lift").Pointer() != reflect.ValueOf(core).Pointer() || enclosing.Elem().FieldByName("zpe").Pointer() != reflect.ValueOf(core).Pointer() {
						t.Fatal("actual captured class source frontier", ordinal, failure)
					}
					if e = core.SetZ(); e != nil {
						t.Fatal("actual captured source core.SetZ", e)
					}
					if after := upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*mpnum.UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats || failure.Error() != sourceMessage || !errors.Is(failure, arena.ErrInvalidSlot) {
						t.Fatal("SetZ changed actual stopped source graph/cause")
					}
					found = true // Only the proved source guard + original actual SetZ.
				}
				b.FailAfter(-1)
				e = service.Close()
				if e != nil && !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("actual matched owning result", ordinal, e)
				}
				if e == nil {
					if sourceClassReached {
						cursor := sourceClass.FieldByName("factorSource").Index(1)
						afterCursor := [3]uint64{cursor.FieldByName("stage").Uint(), cursor.FieldByName("row").Uint(), cursor.FieldByName("column").Uint()}
						if afterCursor != sourceCursor || sourceClass.FieldByName("sourceCause").IsNil() || sourceClass.FieldByName("invocation").Uint() != sourceClassNonce || failure.Error() != sourceMessage || !errors.Is(failure, arena.ErrInvalidSlot) {
							t.Fatal("owning rescue replayed captured source class")
						}
					}
					if child.IsValid() && (child.FieldByName("active").Bool() || child.FieldByName("invocation").Uint() != invocation || child.FieldByName("sourceRemaining").Uint() != sourceRemaining || child.FieldByName("sourceIndex").Uint() != sourceIndex || child.FieldByName("sourceCause").IsNil() != causeNil || child.FieldByName("bodyCause").IsNil() != bodyNil) {
						t.Fatal("owning changed child source history")
					}
					if caller.FieldByName("active").Bool() || caller.FieldByName("sourceCause").IsNil() != callerCauseNil {
						t.Fatal("captured older caller not retired")
					}
					upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, []*mpnum.UPolynomialFactors{fs})
					if e = service.Close(); !errors.Is(e, arena.ErrClosed) {
						t.Fatal("owning repeated consumption", e)
					}
					if e = fs.Close(); e != nil {
						t.Fatal(e)
					}
					if e = primes.Close(); e != nil {
						t.Fatal(e)
					}
					continue
				}
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
