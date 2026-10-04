// Source: src/math/polynomial/polynomial.cpp:7439-7476.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialGlobalConvertSelfBorrow(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{2, 0, 0}}, {-1, [3]uint32{0, 0, 0}}})
	before := m.core.integer.budget.Used()
	out, e := ConvertPolynomial(m, p, m, ^uint32(0), 0)
	serviceMust(t, e)
	if out != p || m.core.integer.budget.Used() != before {
		t.Fatal("same manager returns borrowed source", out, p)
	}
	serviceMust(t, m.CleanupConversion())
}
func TestPolynomialGlobalConvertDistinctOwnerFilter(t *testing.T) {
	sm, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{2, 0, 0}}, {-2, [3]uint32{1, 0, 0}}, {5, [3]uint32{0, 0, 0}}})
	tm, _, _ := polynomialEvalFixture(t, nil)
	before := sm.core.integer.budget.Used()
	out, e := ConvertPolynomial(sm, p, tm, 0, 1)
	serviceMust(t, e)
	serviceMust(t, tm.core.IncRef(out))
	defer func() { serviceMust(t, tm.core.DecRef(out)) }()
	if sm.core.integer.budget.Used() != before {
		t.Fatal("source owner modified")
	}
	for x := int32(-3); x <= 3; x++ {
		in, v := NewInt(x), Int{}
		serviceMust(t, tm.UnivariateEvaluate(out, 0, &in, &v))
		got, e := tm.core.integer.Int64(&v)
		serviceMust(t, e)
		if got != 5-2*int64(x) {
			t.Fatal("independent filtered value", x, got)
		}
		serviceMust(t, tm.core.integer.Del(&v))
	}
	serviceMust(t, tm.CleanupConversion())
}

func TestPolynomialGlobalConvertEveryActualPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			sm, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{2, 0, 0}}, {-2, [3]uint32{1, 0, 0}}, {5, [3]uint32{0, 0, 0}}})
			tm, _, _ := polynomialEvalFixture(t, nil)
			before := sm.core.integer.budget.Used()
			tm.core.integer.budget.FailAfter(ordinal)
			out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual source refusal", ordinal, e)
			}
			polynomialImpPhysicalAuditNamed(t, tm, []Polynomial{out}, nil, 0)
			polynomialImpPhysicalAuditValues(t, sm, []Polynomial{p})
			if sm.core.integer.budget.Used() != before {
				t.Fatal("source was mutated")
			}
			tm.core.integer.budget.FailAfter(-1)
			serviceMust(t, tm.CleanupConversion())
			if e == nil {
				success = true
				serviceMust(t, tm.core.IncRef(out))
				in, value := NewInt(2), Int{}
				serviceMust(t, tm.UnivariateEvaluate(out, 0, &in, &value))
				got, e := tm.core.integer.Int64(&value)
				serviceMust(t, e)
				if got != 13 {
					t.Fatal("independent unfiltered value", got)
				}
				serviceMust(t, tm.core.integer.Del(&value))
				serviceMust(t, tm.core.DecRef(out))
			}
		})
		if success {
			t.Log("first success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("actual prefix horizon exhausted")
	}
}

func TestPolynomialGlobalConvertOverflowingBuffersEveryPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			sm, _, _ := polynomialEvalFixture(t, nil)
			tm, _, _ := polynomialEvalFixture(t, nil)
			values := make([]Int, 130)
			for i := range values {
				values[i] = NewInt(1)
			}
			p, e := sm.ToPolynomial(130, values, 0)
			serviceMust(t, e)
			serviceMust(t, sm.core.IncRef(p))
			defer func() { serviceMust(t, sm.core.DecRef(p)) }()
			tm.core.integer.budget.FailAfter(ordinal)
			out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual overflow source prefix", ordinal, e)
			}
			polynomialImpPhysicalAuditValues(t, tm, []Polynomial{out})
			polynomialImpPhysicalAuditValues(t, sm, []Polynomial{p})
			tm.core.integer.budget.FailAfter(-1)
			serviceMust(t, tm.CleanupConversion())
			if e == nil {
				success = true
				serviceMust(t, tm.core.IncRef(out))
				in, value := NewInt(1), Int{}
				serviceMust(t, tm.UnivariateEvaluate(out, 0, &in, &value))
				got, e := tm.core.integer.Int64(&value)
				serviceMust(t, e)
				if got != 130 {
					t.Fatal("independent sum130", got)
				}
				serviceMust(t, tm.core.integer.Del(&value))
				serviceMust(t, tm.core.DecRef(out))
			}
		})
		if success {
			t.Log("first overflow success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("actual overflow horizon exhausted")
	}
}

func TestPolynomialGlobalConvertBigCoefficientDistinctInteger(t *testing.T) {
	sm, _, _ := polynomialEvalFixture(t, nil)
	tm, _, _ := polynomialEvalFixture(t, nil)
	input := []Int{sm.core.integer.two64, NewInt(1)}
	p, e := sm.ToPolynomial(2, input, 0)
	serviceMust(t, e)
	serviceMust(t, sm.core.IncRef(p))
	defer func() { serviceMust(t, sm.core.DecRef(p)) }()
	out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
	serviceMust(t, e)
	serviceMust(t, tm.core.IncRef(out))
	defer func() { serviceMust(t, tm.core.DecRef(out)) }()
	polynomialImpPhysicalAuditValues(t, sm, []Polynomial{p})
	polynomialImpPhysicalAuditValues(t, tm, []Polynomial{out})
	in, value := NewInt(1), Int{}
	serviceMust(t, tm.UnivariateEvaluate(out, 0, &in, &value))
	text, e := tm.core.integer.String(&value)
	serviceMust(t, e)
	if text != "18446744073709551617" {
		t.Fatal("distinct actual BIG owner", text)
	}
	serviceMust(t, tm.core.integer.Del(&value))
	serviceMust(t, tm.CleanupConversion())
}

func TestPolynomialGlobalConvertExactNumericForeignReplacement(t *testing.T) {
	sm, _, _ := polynomialEvalFixture(t, nil)
	a := Int{}
	serviceMust(t, sm.core.integer.Copy(&a, &sm.core.integer.two64))
	p, e := sm.core.MkConstantTake(&a)
	serviceMust(t, e)
	serviceMust(t, sm.core.IncRef(p))
	defer func() { serviceMust(t, sm.core.DecRef(p)) }()
	budget := arena.NewBudget(1 << 24)
	z, e := NewIntegerManager(IntegerConfig{Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	tm, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: z, Checkpoint: func() bool { panic("source convert has no checkpoint") }, Cooperation: func() error { panic("source convert has no cooperation") }}})
	serviceMust(t, e)
	foreign := Int{}
	defer func() {
		serviceMust(t, tm.CleanupConversion())
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.Del(&foreign))
		serviceMust(t, tm.CloseChecked())
		serviceMust(t, z.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("whole target graph", budget.Used())
		}
	}()
	_, cause := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
	if !errors.Is(cause, arena.ErrSlotLimit) || z.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("actual coefficient payload before slot refusal", cause)
	}
	original := z.pendingCell
	polynomialImpPhysicalAuditNamed(t, tm, nil, []*Int{&foreign}, 0)
	serviceMust(t, z.CleanupPendingAllocation())
	e = z.Copy(&foreign, &z.two64)
	if !errors.Is(e, arena.ErrSlotLimit) || z.pendingCell.object == original.object {
		t.Fatal("actual later generation", e)
	}
	replacement := z.pendingCell
	polynomialImpPhysicalAuditNamed(t, tm, nil, []*Int{&foreign}, 0)
	serviceMust(t, tm.CleanupConversion())
	if z.pendingCell != replacement {
		t.Fatal("scope consumed foreign pending backing")
	}
	polynomialImpPhysicalAuditNamed(t, tm, nil, []*Int{&foreign}, 0)
}

// The published buffers and returned factory value are genuine. The public
// signed-host counter recipe proves lexical stop order, not ordinary valid OOM.
func TestPolynomialGlobalConvertReturnedResultSourceStop(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: z, Checkpoint: func() bool { panic("local helper no checkpoint") }, Cooperation: func() error { panic("local helper no cooperation") }}})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, m.CleanupConversion)
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, z.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("whole named graph", budget.Used())
		}
	}()
	// Ordinary allocations establish genuine ID teardown capacities beforehand.
	var warm [5]Polynomial
	for i := range warm {
		a := NewInt(int32(i + 9))
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
	serviceMust(t, m.beginConversion())
	s := &m.conversion
	unit, e := m.core.monomials.Unit()
	serviceMust(t, e)
	zero := Int{}
	for i := uint32(0); i < 16384; i++ {
		serviceMust(t, s.ms.push(m.core, unit))
		serviceMust(t, s.as.push(&zero))
	}
	one := NewInt(1)
	serviceMust(t, s.as.set(0, &one))
	av, e := s.as.view()
	serviceMust(t, e)
	mv := monomialArrayView{runtime: rt, raw: s.ms.raw, stride: 16, count: s.ms.size}
	serviceMust(t, m.core.som.reset())
	p, e := m.core.mkTermsSource(s.as.count, polynomialUnivariateCoefficients{array: av, indexed: true}, mv)
	serviceMust(t, e)
	if !s.ownsResult || s.result != p {
		t.Fatal("actual source return expression ownership")
	}
	older := s.ms.raw
	polynomialImpPhysicalAudit(t, m)
	contextTLSNearMinimum(t, rt)
	cause := m.finishConversion(nil)
	if !errors.Is(cause, arena.ErrBounds) || s.as.raw != (memory.Block{}) || s.ms.raw != older || !s.ownsResult {
		t.Fatal("younger consumed raw stop preserves older pointer buffer/result", cause)
	}
	polynomialImpPhysicalAudit(t, m)
	state, stats := rt.State(), rt.Stats()
	if again := m.finishConversion(nil); again != cause || rt.State() != state || rt.Stats() != stats {
		t.Fatal("literal sticky source stop", again)
	}
	contextTLSFinish(t, rt, m.CleanupConversion)
	if s.result != (Polynomial{}) || s.sourceCause != cause || s.as.sourceCause != cause {
		t.Fatal("own abort/history")
	}
	polynomialImpPhysicalAudit(t, m)
}

func TestPolynomialGlobalConvertRingZeroNewVersusBorrowedMonomial(t *testing.T) {
	for _, borrowed := range []bool{false, true} {
		t.Run(fmt.Sprint(borrowed), func(t *testing.T) {
			sm, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{7, [3]uint32{2, 0, 0}}})
			tm, _, _ := polynomialEvalFixture(t, nil)
			serviceMust(t, tm.core.SetZPUint64(7))
			var prior Monomial
			if borrowed {
				var e error
				prior, e = tm.core.monomials.MkVariable(0, 2)
				serviceMust(t, e)
			}
			out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
			serviceMust(t, e)
			n, e := tm.core.readWord(out, 8)
			serviceMust(t, e)
			if n != 0 {
				t.Fatal("actual target SET zero size", n)
			}
			serviceMust(t, tm.core.IncRef(out))
			defer func() { serviceMust(t, tm.core.DecRef(out)) }()
			if tm.conversionHasOrphans() == borrowed {
				t.Fatal("created outcome authority distinct from borrowed RC0", borrowed)
			}
			polynomialImpPhysicalAuditValues(t, tm, []Polynomial{out})
			serviceMust(t, tm.CleanupConversion())
			if borrowed {
				if _, e := tm.core.monomials.location(prior); e != nil {
					t.Fatal("own abort swept borrowed existing zero-reference result", e)
				}
				serviceMust(t, tm.core.monomials.CleanupValue(&prior))
			}
			polynomialImpPhysicalAudit(t, tm)
		})
	}
}

func TestPolynomialGlobalConvertSharedMonomialManager(t *testing.T) {
	sm, p, control := polynomialEvalFixture(t, []polynomialEvalTerm{{4, [3]uint32{3, 0, 0}}, {-3, [3]uint32{0, 0, 0}}})
	tm, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: sm.core.integer, Rationals: sm.core.config.Rationals, Monomials: sm.core.monomials, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	defer func() { serviceMust(t, tm.CleanupConversion()); serviceMust(t, tm.CloseChecked()) }()
	vars, e := sm.core.monomials.NumVars()
	serviceMust(t, e)
	out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
	serviceMust(t, e)
	serviceMust(t, tm.core.IncRef(out))
	defer func() { serviceMust(t, tm.core.DecRef(out)) }()
	after, e := sm.core.monomials.NumVars()
	serviceMust(t, e)
	if after != vars || cap(tm.conversion.orphans) != 0 || tm.conversion.calleeGeneration != 0 {
		t.Fatal("shared monomial path must not invoke conversion factory")
	}
	for x := int32(-2); x <= 2; x++ {
		in, v := NewInt(x), Int{}
		serviceMust(t, tm.UnivariateEvaluate(out, 0, &in, &v))
		got, e := tm.core.integer.Int64(&v)
		serviceMust(t, e)
		xx := int64(x)
		if got != 4*xx*xx*xx-3 {
			t.Fatal("independent shared-MM value", x, got)
		}
		serviceMust(t, tm.core.integer.Del(&v))
	}
}

func TestPolynomialGlobalConvertFactoryForeignReplacement(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			sm, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{2, 0, 0}}, {5, [3]uint32{0, 0, 0}}})
			tm, _, _ := polynomialEvalFixture(t, nil)
			defer func() { tm.core.integer.budget.FailAfter(-1); serviceMust(t, tm.CleanupConversion()) }()
			tm.core.integer.budget.FailAfter(ordinal)
			out, e := ConvertPolynomial(sm, p, tm, ^uint32(0), 0)
			polynomialImpPhysicalAuditValues(t, tm, []Polynomial{out})
			tm.core.integer.budget.FailAfter(-1)
			if e == nil {
				serviceMust(t, tm.core.IncRef(out))
				serviceMust(t, tm.core.DecRef(out))
				return
			}
			s := &tm.conversion
			if s.factoryGeneration == 0 || tm.core.pending.generation != s.factoryGeneration {
				return
			}
			found = true
			old := s.factoryGeneration
			// Explicit caller recovery of the original concrete callee precedes a
			// separately invoked large placement constructor, not implicit scope repair.
			serviceMust(t, tm.core.cleanupPendingCreation())
			coefficients := make([]Int, 1024)
			monomials := make([]Monomial, 1024)
			unit, e := tm.core.monomials.Unit()
			serviceMust(t, e)
			for i := range coefficients {
				coefficients[i] = NewInt(1)
				monomials[i] = unit
			}
			tm.core.integer.budget.FailAfter(0)
			_, cause := tm.core.mkCell(1024, coefficients, monomials, nil, nil)
			if !numericPhysicalReservationRefusal(cause) || tm.core.pending.generation == 0 || tm.core.pending.generation == old || tm.core.pending.conversionParent != nil {
				t.Fatal("actual foreign constructor continuation", cause, tm.core.pending.generation, old)
			}
			replacement := tm.core.pending
			polynomialImpPhysicalAudit(t, tm)
			tm.core.integer.budget.FailAfter(-1)
			serviceMust(t, tm.CleanupConversion())
			if tm.core.pending.generation != replacement.generation || tm.core.pending.raw != replacement.raw {
				t.Fatal("scope aborted foreign factory")
			}
			serviceMust(t, tm.core.cleanupPendingCreation())
		})
		if found {
			break
		}
	}
	if !found {
		t.Fatal("actual final factory refusal not exercised")
	}
}
