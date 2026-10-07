// Source: src/math/polynomial/linear_eq_solver.h:79-161.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestLinearIndexedRHSBeforeReachedInput(t *testing.T) {
	m, _, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(2))
	rhs := util.NewInt(7)
	e = s.AddIndexed(0, util.IntegerArrayView{Values: []util.Int{util.NewInt(3)}, Count: 1}, &rhs)
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
	if a != util.NewInt(3) {
		t.Fatal("first coefficient prefix", a)
	}
	a, e = s.load(r, 1)
	serviceMust(t, e)
	if a != util.NewInt(0) {
		t.Fatal("unreached second coefficient", a)
	}
}
func TestLinearIndexedRuntimeOutputsAndRHSAlias(t *testing.T) {
	for alias := false; ; alias = true {
		m, z, _ := modularManager(t, 97)
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(2))
		raw, e := z.RuntimeOwner().Allocate(24)
		serviceMust(t, e)
		for i, w := range []uint32{1, 0, 0, 1, 0, 0} {
			serviceMust(t, z.RuntimeOwner().WriteUint32(raw, uint64(i)*4, w))
		}
		v := util.IntegerArrayView{Runtime: z.RuntimeOwner(), Raw: raw, Count: 2, Stride: 12}
		b := util.NewInt(5)
		serviceMust(t, s.AddIndexed(0, v, &b))
		serviceMust(t, z.RuntimeOwner().WriteUint32(raw, 0, 0))
		b = util.NewInt(3)
		serviceMust(t, s.AddIndexed(1, v, &b))
		out := v
		if alias {
			out = util.IntegerArrayView{Runtime: z.RuntimeOwner(), Raw: s.b.raw, Offset: 8, Count: 2, Stride: 12}
		}
		ok, e := s.SolveIndexed(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("invertible")
		}
		for i, want := range []int32{2, 3} {
			a, e := out.At(uint32(i))
			serviceMust(t, e)
			if a != util.NewInt(want) {
				t.Fatal("independent solution", alias, i, a)
			}
		}
		serviceMust(t, s.Close())
		serviceMust(t, z.RuntimeOwner().Deallocate(&raw))
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
	v := util.IntegerArrayView{Runtime: &util.Runtime{}, Count: 17}
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
	b := util.NewInt(11)
	foreign := util.IntegerArrayView{Runtime: &util.Runtime{}, Count: 1}
	e = s.AddIndexed(0, foreign, &b)
	if !errors.Is(e, util.ErrRuntimeOwner) {
		t.Fatal("reached cohort", e)
	}
	rhs, e := s.RightHandSide(0)
	serviceMust(t, e)
	if rhs != b {
		t.Fatal("cohort preflight reordered RHS", rhs)
	}
	object, e := z.Allocator.Allocate(12)
	serviceMust(t, e)
	for i, w := range []uint32{1, 0, 0} {
		serviceMust(t, z.Allocator.WriteUint32(object, uint64(i)*4, w))
	}
	v := util.IntegerArrayView{Allocator: z.Allocator, Object: object, Count: 1, Stride: 12}
	serviceMust(t, s.AddIndexed(0, v, &b))
	ok, e := s.SolveIndexed(v)
	serviceMust(t, e)
	if !ok {
		t.Fatal("object solution")
	}
	a, e := v.At(0)
	serviceMust(t, e)
	if a != b {
		t.Fatal("actual object write", a)
	}
	serviceMust(t, z.Allocator.Deallocate(12, object))
}
func linearIndexedNamedAudit(t *testing.T, s *ModularLinearSolver, caller util.Block, values ...*util.Int) {
	t.Helper()
	m := s.manager
	roots := []*util.Int{&(*m.BorrowModulus()), &m.Lower, &m.Upper, &m.InvTmp1, &m.InvTmp2, &m.InvTmp3, &m.DivTmp}
	roots = append(roots, values...)
	raw := []util.Block{caller, s.A.raw, s.A.pending, s.A.old, s.b.raw, s.b.pending, s.b.old}
	decoded := []util.Int{}
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
		if r != (util.Block{}) {
			filtered = append(filtered, r)
		}
	}
	integerPhysicalAuditWithRaw(t, m.Integers(), filtered, roots...)
}
func TestLinearIndexedOutputCopyEveryRefusalPrefix(t *testing.T) {
	sawPublished := false
	for fail := int64(0); fail < 80; fail++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := util.NewIntegerModularManager(z)
		serviceMust(t, e)
		serviceMust(t, m.SetZPUint64(18446744073709551557))
		t.Cleanup(func() { serviceMust(t, m.Close()) })
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(1))
		rhs := util.Int{}
		serviceMust(t, z.SetUint64(&rhs, 1<<40))
		serviceMust(t, s.Add(0, []util.Int{util.NewInt(1)}, &rhs))
		raw, e := z.Memory.Allocate(12)
		serviceMust(t, e)
		for i := uint64(0); i < 12; i += 4 {
			serviceMust(t, z.Memory.WriteUint32(raw, i, 0))
		}
		out := util.IntegerArrayView{Runtime: z.Memory, Raw: raw, Count: 1, Stride: 12}
		budget.FailAfter(fail)
		ok, e := s.SolveIndexed(out)
		a, readErr := out.At(0)
		serviceMust(t, readErr)
		linearIndexedNamedAudit(t, s, raw, &rhs, &a)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual source refusal", fail, e)
		}
		if e != nil && z.IsFailureRef((a).Cell) {
			if a.NativeScalar() != 1 || (a).Cell != z.FailedNullRef {
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
		serviceMust(t, out.Store(0, a))
		serviceMust(t, s.Close())
		serviceMust(t, z.Del(&rhs))
		serviceMust(t, z.CleanupPendingAllocation())
		serviceMust(t, z.Memory.Deallocate(&raw))
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
