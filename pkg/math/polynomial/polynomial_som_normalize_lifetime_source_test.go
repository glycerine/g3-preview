// Source: src/math/polynomial/polynomial.cpp:1863-1876;
// src/util/mpz.cpp:1186-1205.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Guarded helper-domain fixture: the real Copy replacement failure yields a
// dangling source field. This does not claim ordinary mathematical GCD grows
// its result to a larger magnitude than its input coefficients.
func TestPolynomialSOMNormalizeCompletedNumeralSourceGuard(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	b := &c.cheap
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	serviceMust(t, b.beginNormalizeNumeral(nil))
	t.Cleanup(func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, b.finishNormalizeNumeral(true))
	})
	serviceMust(t, z.Copy(&b.gcd, &z.Two64))
	z.Budget().FailAfter(0)
	body := z.Copy(&b.gcd, &large)
	if !numericPhysicalReservationRefusal(body) || (b.gcd).Cell != z.FailedRef {
		t.Fatal("actual freed-old replacement refusal", body, b.gcd)
	}
	b.gcdBodyCause = body
	before := b.gcd
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	e := b.finishNormalizeNumeral(false)
	if !errors.Is(e, arena.ErrInvalidSlot) || b.gcd != before || !b.gcdActive {
		t.Fatal("source destructor acknowledged actual dangling field", e, before, b.gcd)
	}
	if again := b.finishNormalizeNumeral(false); again != e || b.gcd != before {
		t.Fatal("source destruction replay changed original frontier", e, again)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, b.finishNormalizeNumeral(true))
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}

func TestPolynomialNormalizeYoungerSourceGuardRetainsOlderNumeral(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	serviceMust(t, full.beginNormalization())
	s := &full.normalization
	// Source's modular-copy branch has only a, then normalize_numerals g;
	// the later integer-branch g has not been constructed in this graph.
	s.aConstructed = true
	serviceMust(t, z.Copy(&s.a, &z.Two64))
	older := s.a
	b := &c.cheap
	serviceMust(t, b.beginNormalizeNumeral(s))
	serviceMust(t, z.Copy(&b.gcd, &z.Two64))
	z.Budget().FailAfter(0)
	body := z.Copy(&b.gcd, &large)
	if !numericPhysicalReservationRefusal(body) || (b.gcd).Cell != z.FailedRef {
		t.Fatal("actual helper dangling state", body, b.gcd)
	}
	b.gcdBodyCause = body
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	guard := b.finishNormalizeNumeral(false)
	if !errors.Is(guard, arena.ErrInvalidSlot) {
		t.Fatal("actual younger source guard", guard)
	}
	if e := full.finishNormalizationSource(); e != guard || s.a != older || s.cursor != 0 {
		t.Fatal("younger stopped destructor consumed older source local", e, s.a, older, s.cursor)
	}
	if e := full.finishNormalizationSource(); e != guard || s.a != older {
		t.Fatal("source stop/cause replay", e)
	}
	c.pscLeave()
	if e := full.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("owning close must preserve actual retained local before children", e)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupNormalization())
	if s.cursor != 0 || b.gcdEnded || full.normalizeInvocationSourceCause(&s.directReceipt) != guard || b.gcdCleanupCause != guard {
		t.Error("own acknowledgment advanced stopped source history")
	}
	if s.active || b.gcdActive {
		t.Fatal("explicit exact child-first owning cleanup left roots")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}

// Actual completed constructor scopes plus the real Copy-produced guarded
// source field: no fabricated destructor error or mathematical reachability.
func TestPolynomialSubstitutionYoungerSOMGuardRetainsOlderNumerals(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	serviceMust(t, c.beginSubstitution(2))
	s := &c.substitution.scope
	s.newAConstructed, s.tmpConstructed = true, true
	b := &c.som
	t.Cleanup(func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, b.finishNormalizeNumeral(true))
		serviceMust(t, c.CleanupSubstitution())
	})
	serviceMust(t, z.Copy(&s.newA, &z.Two64))
	serviceMust(t, z.Copy(&s.tmp, &z.Two64))
	olderA, olderTmp := s.newA, s.tmp
	serviceMust(t, b.beginNormalizeNumeral(nil))
	serviceMust(t, z.Copy(&b.gcd, &z.Two64))
	z.Budget().FailAfter(0)
	body := z.Copy(&b.gcd, &large)
	if !numericPhysicalReservationRefusal(body) || (b.gcd).Cell != z.FailedRef {
		t.Fatal("actual helper replacement state", body, b.gcd)
	}
	b.gcdBodyCause = body
	guard := b.finishNormalizeNumeral(false)
	if !errors.Is(guard, arena.ErrInvalidSlot) {
		t.Fatal("actual stopped inner destructor", guard)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	e := c.finishSubstitution(body)
	if !errors.Is(e, guard) || s.newA != olderA || s.tmp != olderTmp || s.stage != 0 {
		t.Fatal("younger SOM source stop consumed older completed locals", e, s.newA, s.tmp, s.stage)
	}
	if again := c.substitutionCleanup(false); again != guard || s.newA != olderA || s.tmp != olderTmp {
		t.Fatal("source stop replay", again)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, c.CleanupSubstitution())
	if b.gcdActive || s.active {
		t.Fatal("standalone owning cleanup did not drain exact child before older locals")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}

func TestPolynomialSubstitutionExpiredSOMReceiptPreservesReplacement(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	serviceMust(t, c.beginSubstitution(2))
	s := &c.substitution.scope
	s.newAConstructed, s.tmpConstructed = true, true
	b := &c.som
	t.Cleanup(func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, b.finishNormalizeNumeral(true))
		serviceMust(t, c.CleanupSubstitution())
	})
	serviceMust(t, z.Copy(&s.newA, &z.Two64))
	serviceMust(t, z.Copy(&s.tmp, &z.Two64))
	serviceMust(t, b.beginNormalizeNumeral(nil))
	serviceMust(t, z.Copy(&b.gcd, &z.Two64))
	oldInvocation := b.gcdInvocation
	z.Budget().FailAfter(0)
	body := z.Copy(&b.gcd, &large)
	if !numericPhysicalReservationRefusal(body) || (b.gcd).Cell != z.FailedRef {
		t.Fatal("actual dangling state", body, b.gcd)
	}
	guard := b.finishNormalizeNumeral(false)
	if !errors.Is(guard, arena.ErrInvalidSlot) {
		t.Fatal(guard)
	}
	if e := c.finishSubstitution(body); !errors.Is(e, guard) {
		t.Fatal(e)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	z.Budget().FailAfter(-1)
	// Explicit owning acknowledgement of this reached callee, followed by a
	// genuinely new independent constructor. The old outer receipt survives.
	serviceMust(t, b.finishNormalizeNumeral(true))
	serviceMust(t, b.beginNormalizeNumeral(nil))
	serviceMust(t, z.Copy(&b.gcd, &z.Two64))
	foreign := b.gcd
	if b.gcdInvocation == oldInvocation || b.gcdSubstitution != nil {
		t.Fatal("replacement inherited stopped caller", b.gcdInvocation)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	serviceMust(t, c.CleanupSubstitution())
	if !b.gcdActive || b.gcd != foreign {
		t.Fatal("expired receipt swept replacement", foreign, b.gcd)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	// Only the actual independent local's own source destruction consumes it.
	serviceMust(t, b.finishNormalizeNumeral(false))
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}
