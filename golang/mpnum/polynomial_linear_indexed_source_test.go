// Source: src/math/polynomial/linear_eq_solver.h:79-161.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012-01-22.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestLinearIndexedRHSBeforeReachedInput(t *testing.T) {
	m, _, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(2))
	rhs := NewInt(7)
	e = s.AddIndexed(0, integerArrayView{values: []Int{NewInt(3)}, count: 1}, &rhs)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached short input", e)
	}
	b, e := s.RightHandSide(0)
	serviceMust(t, e)
	if b != rhs {
		t.Fatal("RHS publication", b)
	}
	r, e := s.row(0)
	serviceMust(t, e)
	a, e := s.load(r, 0)
	serviceMust(t, e)
	if a != NewInt(3) {
		t.Fatal("first coefficient prefix", a)
	}
	a, e = s.load(r, 1)
	serviceMust(t, e)
	if a != NewInt(0) {
		t.Fatal("unreached second coefficient", a)
	}
}
func TestLinearIndexedRuntimeOutputsAndRHSAlias(t *testing.T) {
	for alias := false; ; alias = true {
		m, z, _ := modularManager(t, 97)
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(2))
		raw, e := z.memory.Allocate(24)
		serviceMust(t, e)
		for i, w := range []uint32{1, 0, 0, 1, 0, 0} {
			serviceMust(t, z.memory.WriteUint32(raw, uint64(i)*4, w))
		}
		v := integerArrayView{runtime: z.memory, raw: raw, count: 2, stride: 12}
		b := NewInt(5)
		serviceMust(t, s.AddIndexed(0, v, &b))
		serviceMust(t, z.memory.WriteUint32(raw, 0, 0))
		b = NewInt(3)
		serviceMust(t, s.AddIndexed(1, v, &b))
		out := v
		if alias {
			out = integerArrayView{runtime: z.memory, raw: s.b.raw, offset: 8, count: 2, stride: 12}
		}
		ok, e := s.SolveIndexed(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("invertible")
		}
		for i, want := range []int32{2, 3} {
			a, e := out.at(uint32(i))
			serviceMust(t, e)
			if a != NewInt(want) {
				t.Fatal("independent solution", alias, i, a)
			}
		}
		serviceMust(t, s.Close())
		serviceMust(t, z.memory.Deallocate(&raw))
		if alias {
			break
		}
	}
}
func TestLinearIndexedEmptyLeavesViewUnread(t *testing.T) {
	m, _, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	v := integerArrayView{runtime: &memory.Runtime{}, count: 17}
	ok, e := s.SolveIndexed(v)
	serviceMust(t, e)
	if !ok {
		t.Fatal("empty source solve")
	}

}

func TestLinearIndexedReachedForeignAndObjectViews(t *testing.T) {
	m, z, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(1))
	b := NewInt(11)
	foreign := integerArrayView{runtime: &memory.Runtime{}, count: 1}
	e = s.AddIndexed(0, foreign, &b)
	if !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("reached cohort", e)
	}
	rhs, e := s.RightHandSide(0)
	serviceMust(t, e)
	if rhs != b {
		t.Fatal("cohort preflight reordered RHS", rhs)
	}
	object, e := z.allocator.Allocate(12)
	serviceMust(t, e)
	for i, w := range []uint32{1, 0, 0} {
		serviceMust(t, z.allocator.WriteUint32(object, uint64(i)*4, w))
	}
	v := integerArrayView{allocator: z.allocator, object: object, count: 1, stride: 12}
	serviceMust(t, s.AddIndexed(0, v, &b))
	ok, e := s.SolveIndexed(v)
	serviceMust(t, e)
	if !ok {
		t.Fatal("object solution")
	}
	a, e := v.at(0)
	serviceMust(t, e)
	if a != b {
		t.Fatal("actual object write", a)
	}
	serviceMust(t, z.allocator.Deallocate(12, object))
}
func linearIndexedNamedAudit(t *testing.T, s *ModularLinearSolver, caller memory.Block, values ...*Int) {
	t.Helper()
	m := s.manager
	roots := []*Int{&m.p, &m.lower, &m.upper, &m.invTmp1, &m.invTmp2, &m.invTmp3, &m.divTmp}
	roots = append(roots, values...)
	raw := []memory.Block{caller, s.A.raw, s.A.pending, s.A.old, s.b.raw, s.b.pending, s.b.old}
	decoded := []Int{}
	for i := uint32(0); i < s.b.initialized; i++ {
		a, e := s.b.intAt(s.runtime, i)
		serviceMust(t, e)
		decoded = append(decoded, a)
	}
	for i := range s.rows {
		r := &s.rows[i]
		raw = append(raw, r.raw, r.pending, r.old)
		for j := uint32(0); j < r.initialized; j++ {
			a, e := r.intAt(s.runtime, j)
			serviceMust(t, e)
			decoded = append(decoded, a)
		}
	}
	for i := range decoded {
		roots = append(roots, &decoded[i])
	}
	filtered := raw[:0]
	for _, r := range raw {
		if r != (memory.Block{}) {
			filtered = append(filtered, r)
		}
	}
	integerPhysicalAuditWithRaw(t, m.manager, filtered, roots...)
}
func TestLinearIndexedOutputCopyEveryRefusalPrefix(t *testing.T) {
	sawPublished := false
	for fail := int64(0); fail < 80; fail++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := NewIntegerModularManager(z)
		serviceMust(t, e)
		serviceMust(t, m.SetZPUint64(18446744073709551557))
		t.Cleanup(func() { serviceMust(t, m.Close()) })
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(1))
		rhs := Int{}
		serviceMust(t, z.SetUint64(&rhs, 1<<40))
		serviceMust(t, s.Add(0, []Int{NewInt(1)}, &rhs))
		raw, e := z.memory.Allocate(12)
		serviceMust(t, e)
		for i := uint64(0); i < 12; i += 4 {
			serviceMust(t, z.memory.WriteUint32(raw, i, 0))
		}
		out := integerArrayView{runtime: z.memory, raw: raw, count: 1, stride: 12}
		budget.FailAfter(fail)
		ok, e := s.SolveIndexed(out)
		a, readErr := out.at(0)
		serviceMust(t, readErr)
		linearIndexedNamedAudit(t, s, raw, &rhs, &a)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual source refusal", fail, e)
		}
		if e != nil && z.isFailureRef(a.cell) {
			if a.val != 1 || a.cell != z.failedNullRef {
				t.Fatal("fresh output Copy prefix", a)
			}
			sawPublished = true
		}
		if e == nil {
			if !ok {
				t.Fatal("nonsingular")
			}
			equal, x := z.Equal(&a, &rhs)
			serviceMust(t, x)
			if !equal {
				t.Fatal("independent single-row output")
			}
		}
		budget.FailAfter(-1)
		serviceMust(t, z.Del(&a))
		serviceMust(t, out.store(0, a))
		serviceMust(t, s.Close())
		serviceMust(t, z.Del(&rhs))
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.memory.Deallocate(&raw))
		if e == nil {
			if !sawPublished {
				t.Fatal("no actual output Copy refusal reached")
			}
			t.Log("refusals before success", fail)
			return
		}
	}
	t.Fatal("no complete refusal endpoint")
}
