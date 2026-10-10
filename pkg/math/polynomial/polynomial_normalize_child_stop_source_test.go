// Source: src/math/polynomial/polynomial.cpp:5904-5940; src/util/mpz.cpp:730-817.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The signed-host TLS guard is an owner-primitive test profile, not mathematical OOM.
func TestPolynomialNormalizeReachedGCDSourceStop(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c, z := full.core, q.Integers()
	for n := 0; n < 2; n++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	var a, b, i, g unumeral.Int
	words := make([]uint32, 15000)
	words[0] = 7
	words[len(words)-1] = 1
	serviceMust(t, z.SetLimbs(&a, words))
	words[0] = 8
	serviceMust(t, z.SetLimbs(&b, words))
	serviceMust(t, z.GCD(&a, &b, &g))
	serviceMust(t, z.Copy(&i, &a))
	x, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	y, e := mm.MkVariable(1, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{a, b}, []Monomial{x, y})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var receipt polynomialNormalizeInvocationReceipt
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-110000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed allocation guard", e)
	}
	_, returned := full.normalizeWithReceipt(p, &receipt)
	s := &full.normalization
	if !errors.Is(returned, arena.ErrBounds) || !z.GcdRaw.BodySucceeded || !s.gConstructed || s.aConstructed || s.cursor != 0 || s.g != unumeral.NewInt(1) || s.gcdRaw.Frame != &z.GcdRaw {
		t.Fatal("actual child end DEL protects older g", returned, z.GcdRaw.BodySucceeded, s.gConstructed, s.aConstructed, s.g)
	}
	captured := returned.Error()
	cause := full.normalizeInvocationSourceCause(&receipt)
	if cause == nil || cause != z.RawInvocationSourceCause(&s.gcdRaw) {
		t.Fatal("actual child source stop missing parent", cause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&a, &b, &i, &g}, 0, []umemory.Block{lane})
	own := full.cleanupNormalizeInvocation(&receipt, false)
	if own != nil && !errors.Is(own, arena.ErrBounds) {
		t.Fatal("actual false owning stop", own)
	}
	if own != nil && s.cleanupCause != own {
		t.Fatal("first owning cause missing", own, s.cleanupCause)
	}
	if full.normalizeInvocationSourceCause(&receipt) != cause || s.aConstructed || s.cursor != 0 {
		t.Fatal("false own phase advanced source cursor")
	}
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, full.cleanupNormalizeInvocation(&receipt, false))
	serviceMust(t, full.cleanupNormalizeInvocation(&receipt, true))
	if own != nil && s.cleanupCause != own {
		t.Fatal("own retry erased first own cause")
	}
	if full.normalizeInvocationSourceCause(&receipt) != cause || z.RawInvocationSourceCause(&receipt.frame.gcdRaw) != cause || returned.Error() != captured {
		t.Fatal("own acknowledgement changed source history")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&a, &b, &i, &g}, 0, []umemory.Block{lane, recovery})
	var later polynomialNormalizeInvocationReceipt
	laterResult, e := full.normalizeWithReceipt(p, &later)
	serviceMust(t, e)
	if laterResult != p {
		t.Fatal("gcd-one source borrowed return changed")
	}
	if later.invocation == receipt.invocation || returned.Error() != captured || !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("later source invocation changed old error")
	}
	serviceMust(t, c.DecRef(p))
	for _, v := range []*unumeral.Int{&a, &b, &i, &g} {
		serviceMust(t, z.Del(v))
	}
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal("actual recovery consumption", e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("explicit complete owner graph remains", budget.Used())
	}
}

func TestPolynomialSOMNormalizeFirstOwnCauseHistory(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c, z := full.core, q.Integers()
	for n := 0; n < 2; n++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	var a, b unumeral.Int
	words := make([]uint32, 15000)
	words[0] = 7
	words[len(words)-1] = 1
	serviceMust(t, z.SetLimbs(&a, words))
	words[0] = 8
	serviceMust(t, z.SetLimbs(&b, words))
	buf := &c.cheap
	serviceMust(t, buf.beginNormalizeNumeral(nil))
	serviceMust(t, z.Copy(&buf.gcd, &a))
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-110000) - rt.State().LocalSize
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	returned := buf.finishNormalizeNumeralPhase(true, false)
	if !errors.Is(returned, arena.ErrBounds) {
		t.Fatal("actual own NUM postphysical guard", returned)
	}
	captured := returned.Error()
	if buf.normalizeScope.ownCause != returned {
		t.Error("actual first owning cause was not retained", returned, buf.normalizeScope.ownCause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, nil, []*unumeral.Int{&a, &b}, 0, []umemory.Block{lane})
	recovery, e := rt.Allocate(2000000)
	serviceMust(t, e)
	serviceMust(t, buf.finishNormalizeNumeralPhase(true, true))
	if buf.normalizeScope.ownCause != returned || returned.Error() != captured || buf.normalizeScope.sourceCause != nil {
		t.Error("own acknowledgment changed diagnostic history")
	}
	serviceMust(t, z.Del(&a))
	serviceMust(t, z.Del(&b))
	serviceMust(t, rt.Deallocate(&lane))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	e = rt.Deallocate(&recovery)
	if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal(e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if budget.Used() != 0 {
		t.Fatal("cold owner graph", budget.Used())
	}
}
