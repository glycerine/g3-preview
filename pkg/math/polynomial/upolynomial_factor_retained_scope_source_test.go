// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1299.
package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialFactorRetainedLexicalCleanup(t *testing.T) {
	// The allocation ordinal depends on the actual source storage profile. Find
	// the first genuine suspended lexical owner, rather than forge a dead bound.
	found := false
	directDebug := false
	for ordinal := int64(0); ordinal < 2048; ordinal++ {
		m, _, b := upolySourceOwner(t)
		profile := testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(reflect.ValueOf(m.IntegerManager()).Elem(), "allocator").Elem(), "config"), "Profile")
		directDebug = testFieldBySourceName(profile, "Debug").Bool() && !testFieldBySourceName(profile, "Windows").Bool()
		primes, e := primegen.NewGenerator(b)
		if e != nil {
			t.Fatal(e)
		}
		service, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
		if e != nil {
			t.Fatal(e)
		}
		fs, e := m.NewFactors()
		if e != nil {
			t.Fatal(e)
		}
		p := upolySourceNew(t, m, nil)
		input := make([]util.Int, 4)
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
		_, failure := service.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
		immediate := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}})
		stats := service.Stats()
		if stats.RetainedScopes != 0 {
			if stats.RetainedOwnerBytes == 0 {
				t.Fatal("uncharged lexical scope")
			}
			core, e := service.RetainedCleanupCore(0)
			pair := testFieldBySourceName(reflect.ValueOf(service).Elem(), "henselPairs")
			preRelatedPair := stats.HenselPending && !pair.IsNil() && testFieldBySourceName(pair.Elem(), "active").Bool() && testFieldBySourceName(pair.Elem(), "constructed").Uint() == 0 && testFieldBySourceName(pair.Elem(), "rm").IsNil()
			if errors.Is(e, arena.ErrBounds) && (stats.MignottePending || stats.PrimeTrialPending || preRelatedPair) {
				// Source completed older declarations, leaving an exact unjoined
				// Mignotte or Prime RAW child, or the exact pair constructor
				// frontier before the related manager is published.
				// This new case must not replace the original later core/set_z campaign.
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact retained child owning acknowledgment", e)
				}
				if preRelatedPair && (testFieldBySourceName(pair.Elem(), "active").Bool() || service.Stats().CachedOwnerBytes != stats.CachedOwnerBytes || service.Stats().CachedScopes != stats.CachedScopes || service.Stats().HenselPending) {
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
				enclosing := testFieldBySourceName(reflect.ValueOf(service).Elem(), "scope")
				step := testFieldBySourceName(enclosing.Elem(), "recombinationStep")
				receipt := testFieldBySourceName(step, "receipt")
				nonce := testFieldBySourceName(step, "invocation").Uint()
				bodyNil, sourceNil, ownNil := testFieldBySourceName(step, "bodyCause").IsNil(), testFieldBySourceName(step, "sourceCause").IsNil(), testFieldBySourceName(step, "ownCause").IsNil()
				if enclosing.IsNil() || testFieldBySourceName(enclosing.Elem(), "constructed").Uint() != 0 || !testFieldBySourceName(step, "active").Bool() || nonce == 0 || bodyNil || failure == nil || testFieldBySourceName(receipt, "owner").Pointer() != enclosing.Pointer() || testFieldBySourceName(receipt, "frame").Pointer() != step.Addr().Pointer() || testFieldBySourceName(receipt, "invocation").Uint() != nonce {
					t.Fatal("actual retired older fields/exact recombination tuple", ordinal, failure)
				}
				// These are already recorded real acquisition fields, not
				// inferred ownership from Stats or graph cardinality.
				actualRoots := !testFieldBySourceName(step, "numeric").IsZero()
				for _, name := range []string{"divisionRaw", "extendedRaw", "gcdRaw", "boolNumeral", "addMul"} {
					frame := testFieldBySourceName(testFieldBySourceName(step, name), "frame")
					if !frame.IsNil() && testFieldBySourceName(frame.Elem(), "active").Bool() {
						actualRoots = true
					}
				}
				if !actualRoots {
					t.Fatal("no exact retained recombination child root", ordinal)
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams()); !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("exact recombination refused new caller", e)
				}
				if after := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats || testFieldBySourceName(step, "invocation").Uint() != nonce {
					t.Fatal("rejected caller changed exact retained graph")
				}
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact recombination false/older/true acknowledgment", e)
				}
				if service.Stats().RetainedScopes != 0 || testFieldBySourceName(step, "active").Bool() || testFieldBySourceName(step, "invocation").Uint() != nonce || testFieldBySourceName(step, "bodyCause").IsNil() != bodyNil || testFieldBySourceName(step, "sourceCause").IsNil() != sourceNil || testFieldBySourceName(step, "ownCause").IsNil() != ownNil {
					t.Fatal("own acknowledgment changed original recombination history", ordinal)
				}
				upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}})
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
				parent := testFieldBySourceName(reflect.ValueOf(service).Elem(), "scope")
				caller := testFieldBySourceName(parent.Elem(), "hensel")
				prep := testFieldBySourceName(parent.Elem(), "primePreparation")
				nonce := testFieldBySourceName(caller, "invocation").Uint()
				message := failure.Error()
				bodyNil, sourceNil, ownNil := testFieldBySourceName(caller, "bodyCause").IsNil(), testFieldBySourceName(caller, "sourceCause").IsNil(), testFieldBySourceName(caller, "ownCause").IsNil()
				if testFieldBySourceName(parent.Elem(), "constructed").Uint() != 0 || !testFieldBySourceName(caller, "active").Bool() || nonce == 0 || bodyNil || !sourceNil || testFieldBySourceName(caller, "owner").Pointer() != parent.Pointer() || testFieldBySourceName(prep, "hensel").Pointer() != caller.Addr().Pointer() || testFieldBySourceName(prep, "henselInvocation").Uint() != nonce || testFieldBySourceName(caller, "outerSource").Uint() != 0 || testFieldBySourceName(caller, "quadraticSource").Uint() != 0 || testFieldBySourceName(caller, "iterationSource").Uint() != 0 {
					t.Fatal("exact completed-source retained Hensel caller", ordinal, failure)
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams()); !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("exact caller refused reuse", e)
				}
				if after := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats {
					t.Fatal("rejected caller changed completed-source ownership")
				}
				b.FailAfter(-1)
				if e = service.CleanupRetained(); e != nil {
					t.Fatal("exact completed-child false/older/true owning acknowledgment", ordinal, e)
				}
				if service.Stats().RetainedScopes != 0 || testFieldBySourceName(caller, "active").Bool() || testFieldBySourceName(caller, "invocation").Uint() != nonce || testFieldBySourceName(caller, "bodyCause").IsNil() != bodyNil || testFieldBySourceName(caller, "sourceCause").IsNil() != sourceNil || testFieldBySourceName(caller, "ownCause").IsNil() != ownNil || failure.Error() != message {
					t.Fatal("completed-source retirement changed original histories")
				}
				upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}})
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
			enclosing := testFieldBySourceName(reflect.ValueOf(service).Elem(), "scope")
			matchedCaller := !enclosing.IsNil() && testFieldBySourceName(testFieldBySourceName(enclosing.Elem(), "hensel"), "active").Bool()
			if matchedCaller {
				// Close is explicitly owning CleanupRetained, not a source retry.
				// This exact newly captured child can now be rescued. The original
				// source/caller-set_z assertions below remain for the old scope route.
				caller := testFieldBySourceName(enclosing.Elem(), "hensel")
				callerCauseNil := testFieldBySourceName(caller, "sourceCause").IsNil()
				var child reflect.Value
				var invocation, sourceRemaining, sourceIndex uint64
				causeNil, bodyNil := true, true
				if !pair.IsNil() {
					child = pair.Elem()
					invocation = testFieldBySourceName(child, "invocation").Uint()
					sourceRemaining = testFieldBySourceName(child, "sourceRemaining").Uint()
					sourceIndex = testFieldBySourceName(child, "sourceIndex").Uint()
					causeNil = testFieldBySourceName(child, "sourceCause").IsNil()
					bodyNil = testFieldBySourceName(child, "bodyCause").IsNil()
				}
				if _, e = service.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams()); !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("matched caller refused new operation", e)
				}

				var sourceCursor [3]uint64
				sourceClass := testFieldBySourceName(enclosing.Elem(), "primePreparation")
				sourceClassNonce := testFieldBySourceName(sourceClass, "invocation").Uint()
				sourceClassReached := errors.Is(failure, arena.ErrInvalidSlot)
				sourceMessage := failure.Error()
				if sourceClassReached {
					cursor := testFieldBySourceName(sourceClass, "factorSource").Index(1)
					sourceCursor = [3]uint64{testFieldBySourceName(cursor, "stage").Uint(), testFieldBySourceName(cursor, "row").Uint(), testFieldBySourceName(cursor, "column").Uint()}
					if testFieldBySourceName(enclosing.Elem(), "constructed").Uint() != 9 || testFieldBySourceName(sourceClass, "sourceCause").IsNil() || sourceClassNonce == 0 || testFieldBySourceName(sourceClass, "owner").Pointer() != enclosing.Pointer() || testFieldBySourceName(sourceClass, "lift").Pointer() != reflect.ValueOf(core).Pointer() || testFieldBySourceName(enclosing.Elem(), "zpe").Pointer() != reflect.ValueOf(core).Pointer() {
						t.Fatal("actual captured class source frontier", ordinal, failure)
					}
					if e = core.SetZ(); e != nil {
						t.Fatal("actual captured source core.SetZ", e)
					}
					if after := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}}); after != immediate || service.Stats() != stats || failure.Error() != sourceMessage || !errors.Is(failure, arena.ErrInvalidSlot) {
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
						cursor := testFieldBySourceName(sourceClass, "factorSource").Index(1)
						afterCursor := [3]uint64{testFieldBySourceName(cursor, "stage").Uint(), testFieldBySourceName(cursor, "row").Uint(), testFieldBySourceName(cursor, "column").Uint()}
						if afterCursor != sourceCursor || testFieldBySourceName(sourceClass, "sourceCause").IsNil() || testFieldBySourceName(sourceClass, "invocation").Uint() != sourceClassNonce || failure.Error() != sourceMessage || !errors.Is(failure, arena.ErrInvalidSlot) {
							t.Fatal("owning rescue replayed captured source class")
						}
					}
					if child.IsValid() && (testFieldBySourceName(child, "active").Bool() || testFieldBySourceName(child, "invocation").Uint() != invocation || testFieldBySourceName(child, "sourceRemaining").Uint() != sourceRemaining || testFieldBySourceName(child, "sourceIndex").Uint() != sourceIndex || testFieldBySourceName(child, "sourceCause").IsNil() != causeNil || testFieldBySourceName(child, "bodyCause").IsNil() != bodyNil) {
						t.Fatal("owning changed child source history")
					}
					if testFieldBySourceName(caller, "active").Bool() || testFieldBySourceName(caller, "sourceCause").IsNil() != callerCauseNil {
						t.Fatal("captured older caller not retired")
					}
					upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs})
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
			if _, e = service.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams()); !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("new arithmetic accepted", e)
			}
			if e = service.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("guarded lexical cleanup", e)
			}
			if service.Stats() != stats {
				t.Fatal("cleanup refusal discarded charged context")
			}
			if after := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}}); after != immediate {
				t.Fatal("refused teardown changed retained physical roots", immediate, after)
			}
			// Caller performs the actual source set_z operation, separately from
			// failed arithmetic/cleanup. No bound or published output is repaired.
			if e = core.SetZ(); e != nil {
				t.Fatal(e)
			}
			if after := upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{service}}); after != immediate {
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
