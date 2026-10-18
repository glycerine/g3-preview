// Source: src/math/polynomial/linear_eq_solver.h:79,82,127.
package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestLinearModularCopyRefusalOwnsExactNumericRoot(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "owned"
		if replacement {
			name = "replacement"
		}
		t.Run(name, func(t *testing.T) {
			budget := arena.NewBudget(1 << 24)
			z, e := unumeral.NewIntegerManager(unumeral.IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			t.Cleanup(func() {
				z.Close()
				if budget.Used() != 0 {
					t.Errorf("retained bytes %d", budget.Used())
				}
			})
			m, e := unumeral.NewIntegerModularManager(z)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.Close()) })
			serviceMust(t, m.SetZPUint64(97))
			s, e := NewModularLinearSolver(m)
			serviceMust(t, e)
			t.Cleanup(func() {
				if err := s.Close(); err != nil {
					serviceMust(t, s.Close())
				}
			})
			serviceMust(t, s.Resize(1))
			body := s.Add(0, []unumeral.Int{unumeral.NewInt(1)}, &z.Two64)
			if !errors.Is(body, arena.ErrSlotLimit) || z.PendingCell.Object == (umemory.ObjectRef{}) {
				t.Fatal("actual copy descriptor refusal not reached", body)
			}
			root := z.PendingCell
			if s.numericRoots.Cell != root {
				t.Fatal("copy context lost exact allocation receipt")
			}
			before, oldRHS := budget.Used(), s.b[0]
			if err := s.Add(0, nil, nil); !errors.Is(err, ErrPolynomialPending) {
				t.Fatal("pending copy accepted new row", err)
			}
			if before != budget.Used() || oldRHS != s.b[0] || z.PendingCell != root {
				t.Fatal("refused row changed pending copy")
			}
			if replacement {
				serviceMust(t, z.CleanupPendingAllocation())
				foreign := unumeral.NewInt(19)
				e = z.Copy(&foreign, &z.Two64)
				if !errors.Is(e, arena.ErrSlotLimit) || z.PendingCell.Object == root.Object {
					t.Fatal("actual replacement allocation missing", e)
				}
				newer := z.PendingCell
				if e := s.Close(); e != nil {
					serviceMust(t, s.Close())
				}
				if z.PendingCell != newer {
					t.Fatal("linear owning cleanup consumed replacement allocation")
				}
				serviceMust(t, z.CleanupPendingAllocation())
				serviceMust(t, z.Del(&foreign))
			} else {
				if e := s.Close(); e != nil {
					serviceMust(t, s.Close())
				}
				if z.PendingAllocationObject() != (umemory.ObjectRef{}) {
					t.Error("linear owning cleanup lost its numeric allocation", root)
					serviceMust(t, z.CleanupPendingAllocation())
				}
			}
		})
	}
}

func TestLinearModularCopyContextInactiveDuringSolveTrace(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := unumeral.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.BindTrace(nil)); serviceMust(t, m.Close()) })
	serviceMust(t, m.SetZPUint64(97))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	serviceMust(t, s.Resize(1))
	rhs := unumeral.NewInt(6)
	serviceMust(t, s.Add(0, []unumeral.Int{unumeral.NewInt(2)}, &rhs))
	previous := unumeral.NumericAllocationContext{}.WithCaller(&s.numericHooks)
	sentinel, value := unumeral.NewInt(99), unumeral.NewInt(5)
	writer := &linearOperationProbeWriter{probe: func() {
		before := q.Budget().Used()
		err := m.WithAllocationContext(previous).Copy(&sentinel, &value)
		if !errors.Is(err, unumeral.ErrNumericOwner) || sentinel != unumeral.NewInt(99) || before != q.Budget().Used() {
			t.Error("inactive copy context acquired numerical authority", err, sentinel)
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("linear_eq_solver")
	serviceMust(t, m.BindTrace(tags))
	result := []unumeral.Int{{}}
	ok, e := s.Solve(result)
	serviceMust(t, e)
	if !ok || result[0] != unumeral.NewInt(3) || writer.called != tags.ShouldTrace("linear_eq_solver") {
		t.Fatal("trace copy guard changed source solution")
	}
	serviceMust(t, q.Integers().Del(&result[0]))
}

func TestLinearModularArithmeticOwnsEveryReservationPrefix(t *testing.T) {
	success, failures := false, 0
	sawDivision, sawExtended, sawAddMul := false, false, false
	for prefix := int64(0); prefix < 1024 && !success; prefix++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := unumeral.NewIntegerModularManager(z)
		serviceMust(t, e)
		serviceMust(t, m.SetZPUint64(18446744073709551557))
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(2))
		rhs0, rhs1 := unumeral.NewInt(7), unumeral.NewInt(11)
		serviceMust(t, s.Add(0, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(1)}, &rhs0))
		serviceMust(t, s.Add(1, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(3)}, &rhs1))
		result := []unumeral.Int{{}, {}}
		budget.FailAfter(prefix)
		ok, body := s.Solve(result)
		budget.FailAfter(-1)
		linearNativeNamedAudit(t, s, &result[0], &result[1])
		for i := range s.rawChildren {
			r := &s.rawChildren[i]
			sawDivision = sawDivision || (r.Invocation != 0 && r.Kind() == unumeral.IntegerRawDivision)
			sawExtended = sawExtended || (r.Invocation != 0 && r.Kind() == unumeral.IntegerRawExtendedGCD)
		}
		sawAddMul = sawAddMul || s.addMulChild.Frame != nil
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected arithmetic refusal", prefix, body)
		}
		if body == nil {
			success = true
			t.Log("all actual arithmetic prefixes before success", prefix)
			if !ok || result[0] != unumeral.NewInt(2) || result[1] != unumeral.NewInt(3) {
				t.Fatal("independent chosen solution", prefix, result)
			}
		} else {
			failures++
		}
		if e := s.Close(); e != nil {
			serviceMust(t, s.Close())
		}
		pending := z.PendingAllocationObject() != (umemory.ObjectRef{}) || z.DivisionRaw.Active || z.ExtendedRaw.Active || z.AddMulInvocation.Active
		for i := range result {
			serviceMust(t, z.Del(&result[i]))
		}
		serviceMust(t, m.Close())
		if pending {
			t.Fatalf("solver cleanup left actual arithmetic allocation/child at prefix %d", prefix)
		}
	}
	if !success || failures == 0 || !sawExtended || !sawAddMul {
		t.Fatal("arithmetic campaign missed reached callers or endpoint", failures, success, sawDivision, sawExtended, sawAddMul)
	}
}
