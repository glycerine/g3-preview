// Source: src/math/polynomial/polynomial.cpp:2217-2255,2280-2299.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialCheapMatchedRootRowRetirementPreservesForeign(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	var receipt polynomialCheapAppendInvocationReceipt
	first, second := unumeral.NewInt(7), unumeral.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&first, c.monomials.unit, false, &receipt))
	serviceMust(t, c.cheap.add(&second, foreign, false))
	beforeForeign, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	beforeUnit, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, false))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, true))
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	coefficient, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	afterForeign, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	afterUnit, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if n != 1 || as != 1 || term != foreign || coefficient != second || afterForeign != beforeForeign || afterUnit+1 != beforeUnit {
		t.Fatal("matched row retirement changed foreign source row/ownership", n, as, term, coefficient, beforeForeign, afterForeign, beforeUnit, afterUnit)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	// Source-owned remaining foreign row is released by its ordinary buffer reset.
	serviceMust(t, c.cheap.reset())
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialCheapActualFactoryMovesRetireMatchedTags(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	var receipt polynomialCheapAppendInvocationReceipt
	a, b := unumeral.NewInt(7), unumeral.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &receipt))
	serviceMust(t, c.cheap.add(&b, foreign, false))
	// The actual later unreceipted source factory moves every existing row;
	// its returned cell belongs to this caller, not the older append scope.
	result, e := c.cheap.mk(false)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(result))
	defer func() { serviceMust(t, c.DecRef(result)) }()
	before, e := c.readWord(result, 0)
	serviceMust(t, e)
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{result})
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, true))
	after, e := c.readWord(result, 0)
	serviceMust(t, e)
	if before != after {
		t.Fatal("older append abort changed actual returned cell", before, after)
	}
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{0, 0, 0}: 7, {1, 0, 0}: 11})
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{result})
}

func TestPolynomialCheapMatchedRowConsumedLastDropResumesExactLeaf(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	owned, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	var receipt polynomialCheapAppendInvocationReceipt
	a, b := unumeral.NewInt(7), unumeral.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, owned, false, &receipt))
	serviceMust(t, c.cheap.add(&b, c.monomials.unit, false))
	foreignRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	c.integer.Budget().FailAfter(0)
	cause := c.cheap.cleanupAppendInvocation(&receipt, true)
	if !numericPhysicalReservationRefusal(cause) || !c.cheap.appendScope.dropConsumed || !c.cheap.appendScope.dropLast {
		t.Fatal("actual last DEC before ID recycle refusal", cause, c.cheap.appendScope)
	}
	rc, e := c.monomialRefCount(owned)
	serviceMust(t, e)
	if rc != 0 {
		t.Fatal("actual consumed RC write", rc)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, true))
	now, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if now != foreignRC {
		t.Fatal("own retry changed foreign reference", now, foreignRC)
	}
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	coefficient, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	if n != 1 || coefficient != b {
		t.Fatal("resumed matched row did not preserve foreign payload", n, coefficient)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.reset())
}

func TestPolynomialCheapOwnerCloseResumesConsumedMatchedDrop(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	owned, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	var receipt polynomialCheapAppendInvocationReceipt
	a, b := unumeral.NewInt(7), unumeral.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, owned, false, &receipt))
	serviceMust(t, c.cheap.add(&b, c.monomials.unit, false))
	foreignRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	c.integer.Budget().FailAfter(0)
	cause := c.cheap.cleanupAppendInvocation(&receipt, true)
	if !numericPhysicalReservationRefusal(cause) || !c.cheap.appendScope.dropConsumed || !c.cheap.appendScope.dropLast {
		t.Fatal("actual last DEC before ID recycle refusal", cause, c.cheap.appendScope)
	}
	rc, e := c.monomialRefCount(owned)
	serviceMust(t, e)
	if rc != 0 {
		t.Fatal("actual consumed RC write", rc)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cheap.close())
	after, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if after+1 != foreignRC {
		t.Fatal("whole owning close did not consume its remaining foreign member once", foreignRC, after)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialCheapSourceResetConsumesOldTagsBeforeForeignReuse(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(foreign))
	defer func() { serviceMust(t, c.monomials.DecRef(foreign)) }()
	var r polynomialCheapAppendInvocationReceipt
	a, b := unumeral.NewInt(7), unumeral.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	serviceMust(t, c.cheap.reset())
	serviceMust(t, c.cheap.add(&b, foreign, false))
	rc, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	beforeBudget := c.integer.Budget().Used()
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
	now, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	value, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	if now != rc || term != foreign || value != b || c.integer.Budget().Used() != beforeBudget {
		t.Fatal("old reset-consumed receipt changed foreign row", rc, now, term, value)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.reset())
}

func TestPolynomialCheapMatchedInvocationStartsAfterForeignSourceRow(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	a, b := unumeral.NewInt(11), unumeral.NewInt(7)
	serviceMust(t, c.cheap.add(&a, foreign, false))
	rc, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&b, c.monomials.unit, false, &r))
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
	now, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	value, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	if now != rc || term != foreign || value != a {
		t.Fatal("new matched invocation consumed earlier foreign row", rc, now, term, value)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.reset())
}

func TestPolynomialCheapKnownPartialCopyPreservesEarlierForeignRow(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 32 && !found; ordinal++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		foreign, e := c.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		a := unumeral.NewInt(11)
		large := unumeral.NewInt(0)
		two := unumeral.NewInt(2)
		serviceMust(t, c.integer.Power(&two, 15000, &large))
		serviceMust(t, c.cheap.add(&a, foreign, false))
		rc, e := c.monomialRefCount(foreign)
		serviceMust(t, e)
		var r polynomialCheapAppendInvocationReceipt
		c.integer.Budget().FailAfter(ordinal)
		cause := c.cheap.addWithAppendReceipt(&large, c.monomials.unit, false, &r)
		as, e := c.cheap.as.size(c)
		serviceMust(t, e)
		ms, e := c.cheap.ms.size(c)
		serviceMust(t, e)
		polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
		partial := cause != nil && as == 2 && ms == 1 && c.cheap.appendScope.coefficientClaims == 1 && c.cheap.appendScope.monomialClaims == 0
		c.integer.Budget().FailAfter(-1)
		if partial {
			found = true
		}
		if e = c.cheap.cleanupAppendInvocation(&r, true); e != nil {
			t.Fatalf("ordinal %d as %d ms %d claims %d/%d partial %v: %v", ordinal, as, ms, c.cheap.appendScope.coefficientClaims, c.cheap.appendScope.monomialClaims, partial, e)
		}
		now, e := c.monomialRefCount(foreign)
		serviceMust(t, e)
		term, e := c.cheap.ms.monomial(c, 0)
		serviceMust(t, e)
		value, e := c.cheap.as.coefficient(c, 0)
		serviceMust(t, e)
		if now != rc || term != foreign || value != a {
			t.Fatal("partial owned copy consumed earlier foreign row", ordinal, rc, now, term, value)
		}
		polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
		serviceMust(t, c.cheap.reset())
		serviceMust(t, c.integer.Del(&large))
	}
	if !found {
		t.Fatal("no actual coefficient-copy refusal before monomial INC/push")
	}
}

// This uses the public signed-host TLS guard profile to isolate an owning
// pending-raw release after row compaction; ordinary math/OOM is not claimed.
func TestPolynomialCheapCompactedRowsRetryAfterPendingPhysicalRelease(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	var r polynomialCheapAppendInvocationReceipt
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, func() error { return c.cheap.cleanupAppendInvocation(&r, true) })
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
	}()
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[39999] = 1
	large := unumeral.Int{}
	serviceMust(t, q.Integers().SetLimbs(&large, digits))
	defer func() { contextTLSFinish(t, rt, func() error { return q.Integers().Del(&large) }) }()
	a := unumeral.NewInt(11)
	serviceMust(t, c.cheap.add(&a, foreign, false))
	// Warm only cold tag metadata, so the target is the source numeric COPY.
	serviceMust(t, c.cheap.prepareAppendTags(2))
	serviceMust(t, rt.SetMaxSize(1))
	cause := c.cheap.addWithAppendReceipt(&large, c.monomials.unit, false, &r)
	if !numericPhysicalReservationRefusal(cause) || c.cheap.appendScope.numeric.Allocator == ([3]umemory.Block{}) {
		t.Fatal("actual pending COPY raw", cause, c.cheap.appendScope.numeric)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSNearMinimum(t, rt)
	cause = c.cheap.cleanupAppendInvocation(&r, true)
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	ms, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	if !errors.Is(cause, arena.ErrBounds) || as != 1 || ms != 1 {
		t.Fatal("postcompaction actual physical release", cause, as, ms, c.cheap.appendScope)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
	later := unumeral.NewInt(13)
	serviceMust(t, c.cheap.add(&later, c.monomials.unit, false))
	unitRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
	contextTLSFinish(t, rt, func() error { return c.cheap.cleanupAppendInvocation(&r, true) })
	nowUnit, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	newValue, e := c.cheap.as.coefficient(c, 1)
	serviceMust(t, e)
	if nowUnit != unitRC || n != 2 || newValue != later {
		t.Fatal("final numeric-only retry changed later source append", nowUnit, unitRC, n, newValue)
	}
	value, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	if value != a {
		t.Fatal("retry changed foreign payload", value)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&large}, 0)
}

func TestPolynomialCheapLiveMatchedInvocationContinuesAfterForeignAppend(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	last, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(last))
	defer func() { serviceMust(t, c.monomials.DecRef(last)) }()
	a, b, d := unumeral.NewInt(7), unumeral.NewInt(11), unumeral.NewInt(13)
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	serviceMust(t, c.cheap.add(&b, foreign, false))
	rc, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	serviceMust(t, c.cheap.addWithAppendReceipt(&d, last, false, &r))
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
	now, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	value, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	lastRC, e := c.monomialRefCount(last)
	serviceMust(t, e)
	if now != rc || term != foreign || value != b || lastRC != 1 {
		t.Fatal("continued matched rows changed foreign/caller", now, rc, term, value, lastRC)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.reset())
}

func TestPolynomialCheapStoppedPrecompactionPreservesLaterForeignAppend(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	owned, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	later, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(later))
	defer func() { serviceMust(t, c.monomials.DecRef(later)) }()
	a, b, d := unumeral.NewInt(7), unumeral.NewInt(11), unumeral.NewInt(13)
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, owned, false, &r))
	serviceMust(t, c.cheap.add(&b, c.monomials.unit, false))
	c.integer.Budget().FailAfter(0)
	cause := c.cheap.cleanupAppendInvocation(&r, true)
	if !numericPhysicalReservationRefusal(cause) || !c.cheap.appendScope.dropConsumed {
		t.Fatal("actual DEC/write before recycle refusal", cause, c.cheap.appendScope)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cheap.add(&d, later, false))
	rc, e := c.monomialRefCount(later)
	serviceMust(t, e)
	unit, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	now, e := c.monomialRefCount(later)
	serviceMust(t, e)
	ur, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if n != 2 || now != rc || ur != unit {
		t.Fatal("old captured bounds truncated later source row", n, now, rc, ur, unit)
	}
	value, e := c.cheap.as.coefficient(c, 1)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 1)
	serviceMust(t, e)
	if value != d || term != later {
		t.Fatal("later foreign source row changed", value, term)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.reset())
}

// Source constructor SASSERT(ms[i]->ref_count()>0) at polynomial.cpp:1415.
// This deliberately invalid zero-reference input is a checked source guard
// fixture, not a source-valid mathematical/OOM path. Fixture payload is restored
// only after the immediate stopped graph; the actual initialized old INC is
// then retired through the matched owning receipt.
func TestPolynomialCheapFactoryCoefficientMoveBeforeZeroRefSourceGuard(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source constructor SASSERT profile")
	}
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	bad, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	defer func() { serviceMust(t, c.monomials.CleanupValue(&bad)) }()
	two := unumeral.NewInt(2)
	a := unumeral.Int{}
	serviceMust(t, c.integer.Power(&two, 70, &a))
	defer func() { serviceMust(t, c.integer.Del(&a)) }()
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	saved, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	unitRC, e := c.monomialRefCount(saved)
	serviceMust(t, e)
	serviceMust(t, c.cheap.ms.putMonomial(c, 0, bad))
	var f polynomialFactoryInvocationReceipt
	_, cause := c.cheap.mkWithFactoryReceipt(false, &f)
	if !errors.Is(cause, ErrPolynomialAssertion) || c.pending.value == (Polynomial{}) {
		t.Fatal("actual constructor coefficient swap then refcount guard", cause, c.pending)
	}
	l, e := c.polynomialLocation(c.pending.value)
	serviceMust(t, e)
	old, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	if l.initializedCoefficients != 1 || l.initializedMonomials != 0 || old != (unumeral.Int{}) || c.cheap.appendScope.coefficientClaims != 0 || c.cheap.appendScope.monomialClaims != 1 {
		t.Fatal("source asymmetric publication prefix", l, old, c.cheap.appendScope)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&a}, 0)
	// Restore this fixture's displaced, already initialized original source
	// pointer. No unread field or source output is filled/repaired.
	serviceMust(t, c.cheap.ms.putMonomial(c, 0, saved))
	if e = c.cheap.cleanupAppendInvocation(&r, true); e != nil {
		t.Fatal("known coefficient transfer/remaining monomial refused", e)
	}
	now, e := c.monomialRefCount(saved)
	serviceMust(t, e)
	if now+1 != unitRC {
		t.Fatal("remaining actual buffer INC not consumed once", unitRC, now)
	}
	serviceMust(t, c.cleanupFactoryInvocation(&f, false))
	serviceMust(t, c.cleanupFactoryInvocation(&f, true))
	polynomialImpPhysicalAuditNamed(t, m, nil, []*unumeral.Int{&a}, 0)
}

// Invalid Go owner/frame copies must never acquire physical root authority.
func TestPolynomialCheapCopiedBufferFrameCannotConsumeActualRows(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a := unumeral.NewInt(7)
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	copied := c.cheap
	forged := r
	forged.frame = &copied.appendScope
	rc, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	state := c.memory.State()
	used := c.integer.Budget().Used()
	serviceMust(t, copied.cleanupAppendInvocation(&forged, true))
	now, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	if now != rc || n != 1 || c.memory.State() != state || c.integer.Budget().Used() != used {
		t.Fatal("copied frame acquired actual root authority", rc, now, n)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
}

func TestPolynomialCheapOwnerClosePreservesAlreadyConsumedEarlierOwnedRow(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	owned, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	foreign, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(foreign))
	defer func() { serviceMust(t, c.monomials.DecRef(foreign)) }()
	a, b, d := unumeral.NewInt(7), unumeral.NewInt(11), unumeral.NewInt(13)
	var r polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	serviceMust(t, c.cheap.addWithAppendReceipt(&b, owned, false, &r))
	serviceMust(t, c.cheap.add(&d, foreign, false))
	before, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	c.integer.Budget().FailAfter(0)
	cause := c.cheap.cleanupAppendInvocation(&r, true)
	if !numericPhysicalReservationRefusal(cause) || c.cheap.appendScope.monomialCursor != 1 || !c.cheap.appendScope.dropConsumed {
		t.Fatal("earlier completed DEC then later recycle refusal", cause, c.cheap.appendScope)
	}
	mid, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if mid+1 != before {
		t.Fatal("earlier actual consumed reference", mid, before)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cheap.close())
	now, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	fr, e := c.monomialRefCount(foreign)
	serviceMust(t, e)
	if now != mid || fr != 1 {
		t.Fatal("whole owner replayed earlier consumed owning action", now, mid, fr)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}
