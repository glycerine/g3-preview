// Source: src/math/polynomial/polynomial.cpp:6832-6841,2582-2594.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialToPolynomialZeroBeforeLocal(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	z := m.core.integer
	before := z.budget.Used()
	z.budget.FailAfter(0)
	p, e := m.ToPolynomial(0, nil, ^uint32(0))
	z.budget.FailAfter(-1)
	serviceMust(t, e)
	if p != m.core.zero || z.budget.Used() != before {
		t.Fatal("source zero before buffer or variable access", p, before, z.budget.Used())
	}
	serviceMust(t, m.CleanupToPolynomial())
}

func TestPolynomialToPolynomialIndependentAscendingCoefficients(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	z := m.core.integer
	coefficients := []Int{NewInt(3), NewInt(0), NewInt(-2), NewInt(0), NewInt(1)}
	before := append([]Int(nil), coefficients...)
	p, e := m.ToPolynomial(5, coefficients, 0)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	defer func() { serviceMust(t, m.core.DecRef(p)) }()
	for x := int32(-3); x <= 3; x++ {
		in, out := NewInt(x), Int{}
		serviceMust(t, m.UnivariateEvaluate(p, 0, &in, &out))
		got, e := z.Int64(&out)
		serviceMust(t, e)
		xx := int64(x)
		want := 3 - 2*xx*xx + xx*xx*xx*xx
		if got != want {
			t.Fatal("independent finite polynomial", x, got, want)
		}
		serviceMust(t, z.Del(&out))
	}
	for i := range coefficients {
		if coefficients[i] != before[i] {
			t.Fatal("const input was consumed", i)
		}
	}
	serviceMust(t, m.CleanupToPolynomial())
}

func TestPolynomialToPolynomialEveryActualPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _, _ := polynomialEvalFixture(t, nil)
			z := m.core.integer
			coefficients := make([]Int, 130)
			serviceMust(t, z.Copy(&coefficients[0], &z.two64))
			serviceMust(t, z.Copy(&coefficients[129], &z.two64))
			defer func() { serviceMust(t, z.Del(&coefficients[0])); serviceMust(t, z.Del(&coefficients[129])) }()
			before := [2]Int{coefficients[0], coefficients[129]}
			z.budget.FailAfter(ordinal)
			p, e := m.ToPolynomial(130, coefficients, 0)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual source reservation", ordinal, e)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&coefficients[0], &coefficients[129]}, 0)
			if coefficients[0] != before[0] || coefficients[129] != before[1] {
				t.Fatal("const input changed")
			}
			z.budget.FailAfter(-1)
			serviceMust(t, m.CleanupToPolynomial())
			if e == nil {
				success = true
				serviceMust(t, m.core.IncRef(p))
				one, out := NewInt(1), Int{}
				serviceMust(t, m.UnivariateEvaluate(p, 0, &one, &out))
				text, e := z.String(&out)
				serviceMust(t, e)
				if text != "36893488147419103232" {
					t.Fatal("independent two coefficients at one", text)
				}
				serviceMust(t, z.Del(&out))
				serviceMust(t, m.core.DecRef(p))
			}
		})
		if success {
			t.Log("first actual success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("actual reservation horizon exhausted")
	}
}

func TestPolynomialToPolynomialZeroKeepsOlderStoppedLocal(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	z := m.core.integer
	z.budget.FailAfter(0)
	_, cause := m.ToPolynomial(1, []Int{z.two64}, 0)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal("actual first copied coefficient refusal", cause)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	old := m.toPolynomial.bodyCause
	count := m.toPolynomial.buffer.count
	z.budget.FailAfter(-1)
	p, e := m.ToPolynomial(0, nil, ^uint32(0))
	serviceMust(t, e)
	if p != m.core.zero || !m.toPolynomial.active || m.toPolynomial.bodyCause != old || m.toPolynomial.buffer.count != count {
		t.Fatal("zero early route acquired or erased old lexical context")
	}
	serviceMust(t, m.CleanupToPolynomial())
}

func TestPolynomialToPolynomialModularSourceSets(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	c := m.core
	z := c.integer
	serviceMust(t, c.modular.SetZPUint64(7))
	coefficients := []Int{NewInt(-11), NewInt(14), NewInt(3)}
	p, e := m.ToPolynomial(3, coefficients, 0)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	for x := int32(0); x < 7; x++ {
		in, out := NewInt(x), Int{}
		serviceMust(t, c.modular.PNormalize(&in))
		serviceMust(t, m.UnivariateEvaluate(p, 0, &in, &out))
		got, e := z.Int64(&out)
		serviceMust(t, e)
		want := (3 + 3*int64(x)*int64(x)) % 7
		// Source mpzzp uses balanced representatives [-3,3].
		if want > 3 {
			want -= 7
		}
		if got != want {
			t.Fatal("actual modular SET normalization", x, got, want)
		}
		serviceMust(t, z.Del(&out))
	}
	if coefficients[0] != NewInt(-11) || coefficients[1] != NewInt(14) {
		t.Fatal("const input normalized in place")
	}
}

func TestPolynomialToPolynomialExactNumericForeignReplacement(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	z, e := NewIntegerManager(IntegerConfig{Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: z, Checkpoint: func() bool { panic("constructor has no checkpoint") }, Cooperation: func() error { panic("constructor has no cooperation") }}})
	serviceMust(t, e)
	foreign := NewInt(19)
	defer func() {
		serviceMust(t, m.CleanupToPolynomial())
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.Del(&foreign))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, z.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("full actual owner graph", budget.Used())
		}
	}()
	_, cause := m.ToPolynomial(1, []Int{z.two64}, 0)
	if !errors.Is(cause, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("actual copied coefficient descriptor refusal", cause)
	}
	original := z.pendingCell
	polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&foreign}, 0)
	serviceMust(t, z.CleanupPendingAllocation())
	e = z.Copy(&foreign, &z.two64)
	if !errors.Is(e, arena.ErrSlotLimit) || z.pendingCell.object == original.object {
		t.Fatal("actual later caller continuation", e)
	}
	replacement := z.pendingCell
	polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&foreign}, 0)
	serviceMust(t, m.CleanupToPolynomial())
	if z.pendingCell != replacement {
		t.Fatal("operation abort consumed foreign pending backing")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&foreign}, 0)
}

// Actual completed buffer and mk_univariate return expression followed by a
// public signed-host-domain raw destructor stop. This owner-primitive proof
// does not claim naturally reached mathematical allocation exhaustion.
func TestPolynomialToPolynomialReturnedResultBeforeRawDestructorStop(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: z, Checkpoint: func() bool { panic("constructor no checkpoint") }, Cooperation: func() error { panic("constructor no cooperation") }}})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, m.CleanupToPolynomial)
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, z.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("whole named graph refund", budget.Used())
		}
	}()
	// Real ordinary ID creation/INC/DEC establishes teardown freelist capacity
	// before the guarded profile; no counter or source allocation repair.
	var warm [5]Polynomial
	for i := range warm {
		a := NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for _, p := range warm {
		serviceMust(t, m.core.DecRef(p))
	}
	x, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	term, e := m.core.monomials.MkVariable(x, 7)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(term))
	serviceMust(t, m.core.monomials.DecRef(term))
	serviceMust(t, m.beginToPolynomial())
	s := &m.toPolynomial
	serviceMust(t, s.buffer.resize(16384))
	one := NewInt(1)
	serviceMust(t, s.buffer.set(0, &one))
	view, e := s.buffer.view()
	serviceMust(t, e)
	p, e := m.core.mkUnivariateSource(x, 16383, polynomialUnivariateCoefficients{array: view, indexed: true})
	serviceMust(t, e)
	if !s.ownsResult || s.result != p {
		t.Fatal("actual callee result not retained before source local destructor")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	contextTLSNearMinimum(t, rt)
	cause := m.finishToPolynomial(nil)
	if !errors.Is(cause, arena.ErrBounds) || s.buffer.raw != (memory.Block{}) || !s.ownsResult || s.result != p || s.buffer.count != 0 {
		t.Fatal("actual raw release before result publication", cause)
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	state := rt.State()
	stats := rt.Stats()
	if again := m.finishToPolynomial(nil); again != cause || rt.State() != state || rt.Stats() != stats {
		t.Fatal("source retry replayed release or changed cause", again)
	}
	saved := s.sourceCause
	contextTLSFinish(t, rt, m.CleanupToPolynomial)
	if s.result != (Polynomial{}) || s.sourceCause != saved || s.buffer.sourceCause != cause {
		t.Fatal("owning abort erased history or retained result")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}
