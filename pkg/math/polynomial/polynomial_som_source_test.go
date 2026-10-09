// Source: src/math/polynomial/polynomial.cpp:2232-2287.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialCheapDuplicateAssertionBeforeZero(t *testing.T) {
	m := polynomialCellFixture(t)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	a := util.NewInt(3)
	serviceMust(t, m.cheap.add(&a, u, false))
	zero := util.NewInt(0)
	e = m.cheap.add(&zero, u, false)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("source duplicate assertion precedes zero exclusion", e)
		}
	} else {
		serviceMust(t, e)
	}
}
func TestPolynomialCheapResetUsesDel(t *testing.T) {
	m := polynomialCellFixture(t)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	a := util.NewInt(-3)
	serviceMust(t, m.cheap.add(&a, u, false))
	serviceMust(t, m.cheap.reset())
	got, e := m.cheap.as.coefficient(m, 0)
	serviceMust(t, e)
	if got != util.NewInt(-3) {
		t.Fatal("source Del retains inline m_val, unlike Reset", got)
	}
}

// Genuine public Runtime signed-overflow guard evidence: the second actual
// monomial free completes physically, then its TLS synchronization stops. This
// is not valid allocation exhaustion and uses no private counter mutation.
func TestPolynomialSOMResetStoppedAfterPhysicalFreeOwningCursor(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: r, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("reset checkpoint"); return false }, Cooperation: func() error { t.Fatal("reset cooperate"); return nil }})
	serviceMust(t, e)
	// Prepare genuine recycled-ID vector capacity before entering the source
	// signed-overflow domain; later cleanup must not need a new allocation whose
	// allocation-side accounting would independently stop before physical work.
	warm := [3]Monomial{}
	for i := range warm {
		x, e := mm.MkMonomial([]MonomialPower{{100 + uint32(i), 1}})
		serviceMust(t, e)
		serviceMust(t, mm.IncRef(x))
		warm[i] = x
	}
	for _, x := range warm {
		serviceMust(t, mm.DecRef(x))
	}
	for i := 0; i < 3; i++ {
		_ = mm.ids.Mk()
	}
	tmp := util.NewInt(2)
	p, e := m.MkConstantTake(&tmp)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	_ = m.ids.Mk()
	small, e := mm.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	powers := make([]MonomialPower, 20000)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), 1}
	}
	large, e := mm.MkMonomial(powers)
	serviceMust(t, e)
	a := util.NewInt(3)
	serviceMust(t, m.som.add(&a, small, false))
	a = util.NewInt(5)
	serviceMust(t, m.som.add(&a, large, false))
	contextTLSNearMinimum(t, r)
	before := r.Stats().LiveBlocks
	if capacity := mm.ids.FreeCapacity(); capacity < 3 {
		t.Fatal("actual ID preparation", capacity)
	}
	e = m.som.reset()
	if !errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks+2 != before {
		t.Fatal("two completed source physical frees with actual warmed ID vector", e, before, r.Stats())
	}
	if _, e = mm.location(small); !errors.Is(e, ErrMonomialOwner) {
		t.Fatal("completed first source deletion", e)
	}
	l, e := mm.location(large)
	serviceMust(t, e)
	if l.raw != (util.ObjectRef{}) || l.stage == 0 {
		t.Fatal("completed second physical action", l)
	}
	polynomialPhysicalAudit(t, m)
	// Owning cleanup must skip the already completed first source DEC and
	// acknowledge the stopped second deletion without re-reading its freed raw.
	contextTLSFinish(t, r, func() error { return m.som.close() })
	contextTLSFinish(t, r, m.CloseChecked)
	before = r.Stats().LiveBlocks
	e = mm.DecManagerRef()
	if e != nil && (!errors.Is(e, arena.ErrBounds) || r.Stats().LiveBlocks >= before) {
		t.Fatal("one actual manager DEC/source-destructor prefix", e)
	}
	// Do not repeat manager DEC after its source lifetime has begun. Explicit
	// owning abort retains/drains the known source-unit and member cursors.
	contextTLSFinish(t, r, mm.CloseChecked)
	contextTLSFinish(t, r, q.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("actual retained owner graph", b.Used())
	}
}

func TestPolynomialStoppedMonomialUnitOwningGuardDependency(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	_, e = mm.Unit()
	serviceMust(t, e)
	x, e := mm.MkMonomial([]MonomialPower{{3, 1}})
	serviceMust(t, e)
	serviceMust(t, mm.IncRef(x))
	serviceMust(t, mm.DecRef(x))
	caller, e := r.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, r)
	if e = r.Deallocate(&caller); !errors.Is(e, arena.ErrBounds) || caller != (util.Block{}) {
		t.Fatal("actual caller physical free before TLS guard", e, caller)
	}
	before := r.Stats().LiveBlocks
	e = mm.DecManagerRef()
	if !errors.Is(e, arena.ErrBounds) || mm.sourceStage != 0 || !mm.sourceFinalizing || r.Stats().LiveBlocks+1 != before {
		t.Fatal("literal stopped source-unit deletion", e, mm.sourceStage, r.Stats())
	}
	l, e := mm.location(mm.unit)
	serviceMust(t, e)
	if l.raw != (util.ObjectRef{}) || l.stage == 0 {
		t.Fatal("real consumed unit raw with charged deletion state", l)
	}
	monomialPhysicalAudit(t, mm)
	if _, e = mm.NumVars(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary operations during source destruction", e)
	}
	// No repeat DEC, capacity read, source-unit repair, or source destructor
	// completion is claimed. This is explicit owning cleanup of stopped fields.
	contextTLSFinish(t, r, mm.CloseChecked)
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("actual owning graph", b.Used())
	}
}

// A physically initialized duplicate slot exercises the literal source DEBUG
// compaction assertion. Owning cleanup must not re-DEC the consumed zero slot.
func TestPolynomialSOMCompactionDuplicateAssertionAndOwnedIntervals(t *testing.T) {
	m := polynomialCellFixture(t)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	before, e := m.monomialRefCount(u)
	serviceMust(t, e)
	for _, a := range []util.Int{util.NewInt(0), util.NewInt(5)} {
		serviceMust(t, m.monomials.IncRef(u))
		serviceMust(t, m.som.as.appendCoefficient(m, a))
		serviceMust(t, m.som.ms.appendMonomial(m, u))
	}
	serviceMust(t, m.som.positions.Set(u, 1))
	e = m.som.removeZeros(false)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("source destination assertion before copy", e)
		}
	} else {
		serviceMust(t, e)
	}
	polynomialPhysicalAudit(t, m)
	serviceMust(t, m.som.close())
	after, e := m.monomialRefCount(u)
	serviceMust(t, e)
	if after != before {
		t.Fatal("actual zero DEC and only remaining owning slot DEC", before, after)
	}
}
