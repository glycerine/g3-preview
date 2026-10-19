// Source: src/math/polynomial/polynomial.cpp:2528-2544,2665-2778.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialVariableConstructionFailurePrefixes(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			_, e = mm.MkVar()
			serviceMust(t, e)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unexpected constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("unexpected cooperate"); return nil }})
			serviceMust(t, e)
			b.FailAfter(fault)
			p, e := m.MkVariable(0, 3)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(p))
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation cause", e)
			}
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p}, nil)
			b.FailAfter(-1)
			if p != (Polynomial{}) {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, m.CloseChecked())
			// The failed source callee's actual pending construction is not
			// a completed MM destructor root. Explicit owning abort precedes
			// the final source reference release and its DEBUG empty check.
			serviceMust(t, mm.CleanupCreation())
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual whole graph teardown", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded real failure sweep did not reach success")
	}
}

func TestPolynomialMultiplicationFailurePrefixes(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			_, e = mm.MkVar()
			serviceMust(t, e)
			visits, cooperations := uint32(0), uint32(0)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { visits++; return visits < 1000 }, Cooperation: func() error { cooperations++; return nil }})
			serviceMust(t, e)
			x, e := m.MkVariable(0, 1)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(x))
			b.FailAfter(fault)
			p, e := m.Mul(x, x)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(p))
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation cause", e)
			}
			if visits == 0 || cooperations != visits {
				t.Fatal("actual checkpoint/cooperation pairing", visits, cooperations)
			}
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{x, p}, nil)
			b.FailAfter(-1)
			if p != (Polynomial{}) {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, m.DecRef(x))
			serviceMust(t, m.CloseChecked())
			// The failed source callee's actual pending construction is not
			// a completed MM destructor root. Explicit owning abort precedes
			// the final source reference release and its DEBUG empty check.
			serviceMust(t, mm.CleanupCreation())
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual whole graph teardown", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded real failure sweep did not reach success")
	}
}

func TestPolynomialLinearMixedGeneratedFailurePrefixes(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			for i := 0; i < 3; i++ {
				_, e = mm.MkVar()
				serviceMust(t, e)
			}
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unexpected constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("unexpected cooperate"); return nil }})
			serviceMust(t, e)
			b.FailAfter(fault)
			as := []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(3), unumeral.NewInt(5)}
			c := unumeral.NewInt(7)
			p, e := m.MkLinearTake(as, []uint32{0, 1, 0}, &c)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(p))
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation cause", e)
			}
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p}, nil)
			b.FailAfter(-1)
			if p != (Polynomial{}) {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, m.CloseChecked())
			// The failed source callee's actual pending construction is not
			// a completed MM destructor root. Explicit owning abort precedes
			// the final source reference release and its DEBUG empty check.
			serviceMust(t, mm.CleanupCreation())
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual whole graph teardown", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded real failure sweep did not reach success")
	}
}

// Test oracle: enumerate actual named source/private/caller fields. The locator
// directory is used only to validate the explicit token, never to find owners.
func polynomialOperationMonomialAudit(t *testing.T, m *MultivariatePolynomialCore, callers []Polynomial, callerMonomials []Monomial) {
	t.Helper()
	refs := map[Monomial]bool{}
	add := func(a Monomial) {
		if a == (Monomial{}) {
			return
		}
		l, e := m.monomials.location(a)
		serviceMust(t, e)
		if !l.published {
			t.Fatal("named unjoined monomial publication")
		}
		refs[a] = true
	}
	if owner := m.squareRootOwner; owner != nil {
		add(owner.reference.value)
		add(owner.reference.incoming)
		for _, a := range owner.orphans {
			add(a)
		}
		if m.matchesFactoryInvocation(&owner.factory) && owner.factory.frame.result != (Polynomial{}) {
			callers = append(callers, owner.factory.frame.result)
		}
	}
	add(m.monomials.unit)
	if m.translation.constructed {
		add(m.translation.xk.value)
		add(m.translation.xk.incoming)
	}
	add(m.exactDivision.monomial.value)
	add(m.exactDivision.monomial.incoming)
	add(m.pseudoDivision.monomial.value)
	add(m.pseudoDivision.monomial.incoming)
	for _, a := range callerMonomials {
		add(a)
	}
	if m.generatedActive {
		add(m.generatedMonomial)
	}
	if m.singletonReference {
		add(m.singletonMonomial)
	}
	for i, owned := range m.linearGenerated {
		if owned {
			add(m.linearMonomials[i])
		}
	}
	cells := append([]Polynomial{m.zero, m.unit, m.overloadResult, m.negativeResult, m.constantScope.result, m.exactDivision.returned, m.translation.freshResult}, callers...)
	for i := range m.pseudoDivision.refs {
		cells = append(cells, m.pseudoDivision.refs[i].value, m.pseudoDivision.refs[i].incoming)
	}
	if m.powerLocal.active {
		cells = append(cells, m.powerLocal.result.value, m.powerLocal.result.incoming)
	}
	for _, p := range cells {
		if p == (Polynomial{}) {
			continue
		}
		n, e := m.readWord(p, 8)
		serviceMust(t, e)
		for i := uint32(0); i < n; i++ {
			a, e := m.monomialCore(p, i)
			serviceMust(t, e)
			add(a)
		}
	}
	if m.pending.value != (Polynomial{}) {
		l, e := m.polynomialLocation(m.pending.value)
		serviceMust(t, e)
		for i := uint32(0); i < l.initializedMonomials; i++ {
			a, e := m.monomialCore(m.pending.value, i)
			serviceMust(t, e)
			add(a)
		}
	}
	for _, b := range [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		if b.hasPendingReference {
			add(b.pendingMonomial)
		}
		n, e := b.ms.size(m)
		serviceMust(t, e)
		for i := b.transferred; i < n; i++ {
			if b.borrowedAppend && i == b.borrowedAppendIndex {
				continue // INC did not complete; this slot is still borrowed.
			}
			a, e := b.ms.monomial(m, i)
			serviceMust(t, e)
			add(a)
		}
	}
	if c := m.monomials.creation; c.stage != 0 && c.slot != 0 {
		l := &m.monomials.locations[c.slot-1]
		if l.published {
			add(Monomial{m.monomials.origin, c.slot, l.generation})
		}
	}
	if uint32(len(refs)) != m.monomials.publishedObjects {
		t.Fatal("unnamed published source monomial", len(refs), m.monomials.publishedObjects)
	}
}

func TestPolynomialUnivariateGeneratedFailurePrefixes(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			serviceMust(t, mm.IncManagerRef())
			_, e = mm.MkVar()
			serviceMust(t, e)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unexpected constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("unexpected cooperate"); return nil }})
			serviceMust(t, e)
			b.FailAfter(fault)
			as := []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(0), unumeral.NewInt(2)}
			p, e := m.MkUnivariateTake(0, 2, as)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(p))
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation cause", e)
			}
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p}, nil)
			b.FailAfter(-1)
			if p != (Polynomial{}) {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, m.CloseChecked())
			// The failed source callee's actual pending construction is not
			// a completed MM destructor root. Explicit owning abort precedes
			// the final source reference release and its DEBUG empty check.
			serviceMust(t, mm.CleanupCreation())
			serviceMust(t, mm.DecManagerRef())
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual whole graph teardown", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded real failure sweep did not reach success")
	}
}

func TestPolynomialSoleRetainCalleeConstructionCleanup(t *testing.T) {
	success := false
	for fault := int64(0); fault < 100 && !success; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			profile := umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: profile}})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().RuntimeOwner(), Profile: profile})
			serviceMust(t, e)
			_, e = mm.MkVar()
			serviceMust(t, e)
			m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unexpected constructor checkpoint"); return false }, Cooperation: func() error { t.Fatal("unexpected cooperate"); return nil }})
			serviceMust(t, e)
			b.FailAfter(fault)
			p, e := m.MkVariable(0, 3)
			if e == nil {
				success = true
				serviceMust(t, m.IncRef(p))
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual reservation cause", e)
			}
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p}, nil)
			b.FailAfter(-1)
			if p != (Polynomial{}) {
				serviceMust(t, m.DecRef(p))
			}
			serviceMust(t, m.CloseChecked())
			// The failed source callee's actual pending construction is not
			// a completed MM destructor root. Explicit owning abort precedes
			// the final source reference release and its DEBUG empty check.
			if !mm.closed {
				t.Fatal("last actual core retain did not close MM")
			}
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("actual whole graph teardown", b.Used())
			}
		})
	}
	if !success {
		t.Fatal("bounded real failure sweep did not reach success")
	}
}

func TestPolynomialPreexistingCalleeCreationRemainsBorrowed(t *testing.T) {
	m := polynomialCellFixture(t)
	m.integer.Budget().FailAfter(0)
	_, e := m.monomials.MkVariable(1, 4)
	if !numericPhysicalReservationRefusal(e) || m.monomials.creation.stage == 0 {
		t.Fatal("real preexisting caller creation", e, m.monomials.creation)
	}
	before := m.monomials.creation
	tableSize := m.monomials.table.entries.Len()
	count, e := m.memory.AllocationCount()
	serviceMust(t, e)
	_, e = m.MkVariable(2, 3)
	if e != ErrMonomialPending {
		t.Fatal("actual source pending guard", e)
	}
	if m.generatedActive || m.generatedCalleePending {
		t.Fatal("foreign call authority was fabricated")
	}
	polynomialPhysicalAudit(t, m)
	polynomialOperationMonomialAudit(t, m, nil, nil)
	serviceMust(t, m.CleanupConstruction())
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if m.monomials.creation != before || m.monomials.table.entries.Len() != tableSize || after != count {
		t.Fatal("borrowed prior creation changed")
	}
	polynomialPhysicalAudit(t, m)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, m.monomials.CleanupCreation())
}
