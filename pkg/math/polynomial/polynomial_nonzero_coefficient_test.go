package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialNonzeroConstantCoefficientIndependentCases(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, -7, 5, 2}, [][]MonomialPower{{{0, 1}, {1, 3}}, {{1, 1}}, {{1, 2}}, nil})
	for _, row := range []struct {
		x, k uint32
		want bool
	}{{1, 3, false}, {1, 2, true}, {1, 1, true}, {1, 0, true}, {1, 9, false}, {0, 1, false}, {0, 0, false}} {
		got, e := m.NonzeroConstantCoefficient(p, row.x, row.k)
		serviceMust(t, e)
		if got != row.want || m.nonzeroConstant.active || m.nonzeroConstant.numeralConstructed {
			t.Fatal("independent constant/nonzero query or scoped destructor", row, got)
		}
	}
	// The concrete coefficient owner is the modular manager, not an ad-hoc
	// integer read: five reduces to zero, while minus seven is nonzero mod five.
	serviceMust(t, m.modular.SetZPUint64(5))
	for _, row := range []struct {
		k    uint32
		want bool
	}{{2, false}, {1, true}, {0, true}, {3, false}, {7, false}} {
		got, e := m.NonzeroConstantCoefficient(p, 1, row.k)
		serviceMust(t, e)
		if got != row.want {
			t.Fatal("finite-field coefficient normalization", row, got)
		}
	}
	serviceMust(t, m.modular.SetZ())
}

func nonzeroCoefficientTLSFixture(t *testing.T) (*MultivariatePolynomialManager, *unumeral.RationalManager, *umemory.Runtime, *arena.Budget, Polynomial, *unumeral.Int) {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := new(unumeral.Int)
	limbs := make([]uint32, 32768)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, c.integer.SetLimbs(large, limbs))
	p, e := c.MkConstantTakeCopy(large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	return m, q, rt, b, p, large
}
func nonzeroCoefficientTLSClose(t *testing.T, m *MultivariatePolynomialManager, q *unumeral.RationalManager, rt *umemory.Runtime, b *arena.Budget, p Polynomial, large *unumeral.Int) {
	t.Helper()
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.core.CleanupNonzeroConstantCoefficient())
	serviceMust(t, m.core.DecRef(p))
	serviceMust(t, m.core.integer.Del(large))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("predicate fixture retained actual ownership", b.Used())
	}
}
func TestPolynomialNonzeroConstantCoefficientActualTLSFailureAndRetry(t *testing.T) {
	m, q, rt, b, p, large := nonzeroCoefficientTLSFixture(t)
	defer nonzeroCoefficientTLSClose(t, m, q, rt, b, p, large)
	c := m.core
	serviceMust(t, rt.SetMaxSize(1))
	_, e := c.NonzeroConstantCoefficient(p, 0, 0)
	s := &c.nonzeroConstant
	if e == nil || !rt.State().OutOfMemory || !s.active || !c.matchesConstantCoefficientInvocation(&s.constant) || !c.integer.MatchesCopyAllocationInvocation(&s.constant.frame.copy) {
		t.Fatal("actual child SET prefix not retained", e)
	}
	if s.numeralConstructed {
		t.Fatal("source scoped numeral destructor did not finish before owning allocation abort")
	}
	before, child := *s, *s.constant.frame
	used, state, stats := b.Used(), rt.State(), rt.Stats()
	for _, f := range []func() error{c.CloseChecked, c.SourceFinalizePrefix, m.CloseChecked, m.SourceFinalize} {
		if e := f(); !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("parent progressed during pending predicate", e)
		}
	}
	_, e = c.NonzeroConstantCoefficient(p, 0, 0)
	if !errors.Is(e, ErrPolynomialPending) || *s != before || *s.constant.frame != child || b.Used() != used || rt.State() != state || rt.Stats() != stats {
		t.Fatal("rejected predicate changed its earlier accepted prefix", e)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.CleanupNonzeroConstantCoefficient())
	if s.active || c.constantCoefficientInvocation.active {
		t.Fatal("accepted predicate/child remained pending")
	}
	result, e := c.NonzeroConstantCoefficient(p, 0, 0)
	serviceMust(t, e)
	if !result || s.active {
		t.Fatal("actual retry failed", result)
	}
}
func TestPolynomialNonzeroConstantCoefficientPreservesOlderCoefficientScope(t *testing.T) {
	m, q, rt, b, p, large := nonzeroCoefficientTLSFixture(t)
	defer nonzeroCoefficientTLSClose(t, m, q, rt, b, p, large)
	c := m.core
	out := unumeral.NewInt(19)
	var older polynomialConstantCoefficientInvocationReceipt
	serviceMust(t, rt.SetMaxSize(1))
	_, e := c.constantCoefficientWithReceipt(p, 0, 0, &out, &older)
	if e == nil || !c.matchesConstantCoefficientInvocation(&older) {
		t.Fatal("older owner not qualified", e)
	}
	before, copyBefore := *older.frame, *older.frame.copy.Frame
	used, state, stats := b.Used(), rt.State(), rt.Stats()
	_, e = c.NonzeroConstantCoefficient(p, 0, 0)
	if e == nil || c.nonzeroConstant.constant.manager != nil || c.nonzeroConstant.numeralConstructed {
		t.Fatal("predicate adopted a refused child call or retained its zero numeral", e)
	}
	serviceMust(t, c.CleanupNonzeroConstantCoefficient())
	if *older.frame != before || *older.frame.copy.Frame != copyBefore || b.Used() != used || rt.State() != state || rt.Stats() != stats {
		t.Fatal("predicate cleanup changed independent older ownership")
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&older, true))
}
func TestPolynomialNonzeroConstantCoefficientSourceGuardAndPendingClose(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	result, e := m.NonzeroConstantCoefficient(p, 99, 1)
	if polynomialSourceDebug {
		if !linearThrownAssertion(e) {
			t.Fatal("source coefficient assertion action", e)
		}
	} else {
		serviceMust(t, e)
		if result {
			t.Fatal("erased source assertion invented a nonzero coefficient")
		}
	}
	if m.nonzeroConstant.active {
		serviceMust(t, m.CleanupNonzeroConstantCoefficient())
	}
	got, e := m.NonzeroConstantCoefficient(p, 0, 1)
	serviceMust(t, e)
	if !got {
		t.Fatal("guard cleanup corrupted later coefficient query")
	}
}

func TestPolynomialNonzeroConstantCoefficientActualNormalizeScratch(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large unumeral.Int
	digits := make([]uint32, 128)
	digits[0] = 17
	digits[127] = 1
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	pad := int64(100000-700-8) - rt.State().LocalSize
	if pad <= 0 {
		t.Fatal("source lane passed cold normalization prefix")
	}
	external, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.CleanupNonzeroConstantCoefficient())
		serviceMust(t, rt.Deallocate(&external))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&large))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("normalize-scratch owner retained bytes", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, e = c.NonzeroConstantCoefficient(p, 0, 0)
	s := &c.nonzeroConstant
	roots, rootErr := c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, rootErr)
	if e == nil || !rt.State().OutOfMemory || !s.active || s.numeralConstructed || !s.value.IsSmall() || roots[0] == (umemory.Block{}) || !c.matchesConstantCoefficientInvocation(&s.constant) {
		t.Fatal("source scoped NUM did not die before unjoined Rem scratch abort", e, s.numeralConstructed, roots)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.CleanupNonzeroConstantCoefficient())
	roots, e = c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if roots[0] != (umemory.Block{}) || s.active {
		t.Fatal("exact normalize scratch remains after owning acknowledgment", roots)
	}
	if c.integer.Tmp[0] != 0 {
		t.Fatal("owning cleanup repaired the source scratch hole")
	}
	// Source ensure_tmp freed the old scratch before its replacement refused.
	// Owning cleanup drains acquired storage; it does not recreate that source
	// member. The next actual call must still guard the non-readable hole.
	_, e = c.NonzeroConstantCoefficient(p, 0, 0)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("failed scratch lifetime was made readable by cleanup", e)
	}
	serviceMust(t, c.CleanupNonzeroConstantCoefficient())
	if c.integer.Tmp[0] != 0 {
		t.Fatal("rejected retry repaired source scratch")
	}

}
