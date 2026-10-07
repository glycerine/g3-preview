// Source: src/math/polynomial/polynomial.cpp:3415-3428; src/util/mpz.cpp:1186-1205.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This guarded-domain helper test uses the real setter's dangling-pointer
// continuation after old physical storage was freed and replacement refused.
// It does not claim that ordinary mathematical IC inputs cause that domain.
func TestPolynomialIntegerContentCompletedLocalActualDanglingGuard(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	z := m.integer
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&large)) })
	m.contentLocal = polynomialContentScope{invocation: 1, active: true, aiConstructed: true}
	t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, m.CleanupIntegerContent()) })
	serviceMust(t, z.Copy(&m.contentLocal.ai, &z.Two64))
	z.Budget().FailAfter(0)
	body := z.Copy(&m.contentLocal.ai, &large)
	if !numericPhysicalReservationRefusal(body) || (m.contentLocal.ai).Cell != z.FailedRef {
		t.Fatal("actual freed-old replacement refusal", body, m.contentLocal.ai)
	}
	original := m.contentLocal.ai
	m.contentLocal.bodyCause = body
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	e := m.finishIntegerContent(false)
	if !errors.Is(e, arena.ErrInvalidSlot) || m.contentLocal.ai != original || !m.contentLocal.active {
		t.Fatal("source destructor must retain actual dangling guard", e, original, m.contentLocal)
	}
	if again := m.finishIntegerContent(false); again != e || m.contentLocal.ai != original {
		t.Fatal("source destructor repeated owning acknowledgement", e, again)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, m.CleanupPSC())
	if m.contentLocal.ai != original || !m.contentLocal.active {
		t.Fatal("generic PSC cleanup acquired unrelated IC local", m.contentLocal)
	}
	if e := m.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("core owning close must retain IC local before private destruction", e)
	}
	if e := full.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("full owning close must retain IC local before child destruction", e)
	}
	// The same actual stopped destructor belongs to this explicit nested
	// invocation only; a different generation must not poison older locals.
	serviceMust(t, m.pscEnter())
	m.contentLocal.generation = m.pscExecutingOwner()
	if stopped := m.pscSourceStopped(); stopped != e {
		t.Fatal("matching IC source stop did not block older destruction", stopped, e)
	}
	m.contentLocal.generation++
	if stopped := m.pscSourceStopped(); stopped != nil {
		t.Fatal("unrelated IC generation blocked source destruction", stopped)
	}
	m.pscLeave()
	receipt := polynomialContentInvocationReceipt{m, &m.contentLocal, m.contentLocal.invocation}
	captured := e.Error()
	serviceMust(t, m.cleanupContentInvocation(&receipt, false))
	if m.contentInvocationSourceCause(&receipt) != e {
		t.Fatal("false owning phase erased source cause")
	}
	serviceMust(t, m.cleanupContentInvocation(&receipt, true))
	if m.contentInvocationSourceCause(&receipt) != e {
		t.Fatal("true owning phase erased source cause")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	p := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	var i util.Int
	var primitive polynomialReference
	var current polynomialContentInvocationReceipt
	serviceMust(t, m.integerContentWithReceipt(p, &i, &primitive, &current))
	if current.invocation == receipt.invocation || e.Error() != captured || !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("later invocation changed original source error", e)
	}
	serviceMust(t, m.cleanupContentInvocation(&receipt, false))
	serviceMust(t, m.cleanupContentInvocation(&receipt, true))
	if i != util.NewInt(2) {
		t.Fatal("later mathematical content", i)
	}
	polynomialQueryExpect(t, m, primitive.value, map[[3]uint32]int64{{1, 0, 0}: 1, {0, 1, 0}: 2})
	serviceMust(t, primitive.close(m))
	serviceMust(t, m.integer.Del(&i))
}

// A named cold-owner census fixture, not a claim that a mathematical IC call
// simultaneously constructs the monic scope. Each retained field owns a real
// independently copied BIG Integer; callers supply no duplicate local roots.
func TestPolynomialContentMonicCollectorActualRetainedLocals(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	m.contentLocal.active = true
	m.contentLocal.aiConstructed = true
	full.monic.active = true
	t.Cleanup(func() {
		serviceMust(t, full.CleanupGlexMonic())
		serviceMust(t, m.CleanupIntegerContent())
	})
	serviceMust(t, m.integer.Copy(&m.contentLocal.ai, &m.integer.Two64))
	serviceMust(t, m.integer.Copy(&full.monic.inv, &m.integer.Two64))
	serviceMust(t, m.integer.Copy(&full.monic.newA, &m.integer.Two64))
	if (m.contentLocal.ai).Cell == 0 || (full.monic.inv).Cell == 0 || (full.monic.newA).Cell == 0 {
		t.Fatal("fixture requires three actual retained BIG roots")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
}
