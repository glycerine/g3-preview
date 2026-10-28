package polynomial

// Source: src/math/polynomial/linear_eq_solver.h:79-161.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestLinearNativeRHSBeforeReachedInput(t *testing.T) {
	m, _, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(2))
	rhs := unumeral.NewInt(7)
	e = s.Add(0, []unumeral.Int{unumeral.NewInt(3)}, &rhs)
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
	if a != unumeral.NewInt(3) {
		t.Fatal("first coefficient prefix", a)
	}
	a, e = s.load(r, 1)
	serviceMust(t, e)
	if a != unumeral.NewInt(0) {
		t.Fatal("unreached second coefficient", a)
	}
}
func TestLinearNativeOutputsAndRHSAlias(t *testing.T) {
	for _, alias := range []bool{false, true} {
		m, _, _ := modularManager(t, 97)
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(2))
		b := unumeral.NewInt(5)
		serviceMust(t, s.Add(0, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}, &b))
		b = unumeral.NewInt(3)
		serviceMust(t, s.Add(1, []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(1)}, &b))
		out := make([]unumeral.Int, 2)
		if alias {
			out = s.b
		}
		ok, e := s.Solve(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("invertible")
		}
		for i, want := range []int32{2, 3} {
			if out[i] != unumeral.NewInt(want) {
				t.Fatal("independent solution", alias, i, out[i])
			}
		}
		serviceMust(t, s.Close())
	}
}
func TestLinearNativeEmptySolve(t *testing.T) {
	m, _, _ := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	ok, e := s.Solve(nil)
	serviceMust(t, e)
	if !ok {
		t.Fatal("empty source solve")
	}
}

func linearNativeNamedAudit(t *testing.T, s *ModularLinearSolver, values ...*unumeral.Int) {
	t.Helper()
	m := s.manager
	roots := []*unumeral.Int{m.BorrowModulus(), &m.Lower, &m.Upper, &m.InvTmp1, &m.InvTmp2, &m.InvTmp3, &m.DivTmp}
	roots = append(roots, values...)
	for i := range s.b {
		roots = append(roots, &s.b[i])
	}
	for i := range s.A {
		for j := range s.A[i] {
			roots = append(roots, &s.A[i][j])
		}
	}
	integerPhysicalAuditWithRaw(t, m.Integers(), nil, roots...)
}
func TestLinearNativeOutputCopyEveryRefusalPrefix(t *testing.T) {
	sawPublished := false
	for fail := int64(0); fail < 80; fail++ {
		z, budget := mpzPublicationDirectOwner(t)
		m, e := unumeral.NewIntegerModularManager(z)
		serviceMust(t, e)
		serviceMust(t, m.SetZPUint64(18446744073709551557))
		t.Cleanup(func() { serviceMust(t, m.Close()) })
		s, e := NewModularLinearSolver(m)
		serviceMust(t, e)
		serviceMust(t, s.Resize(1))
		rhs := unumeral.Int{}
		serviceMust(t, z.SetUint64(&rhs, 1<<40))
		serviceMust(t, s.Add(0, []unumeral.Int{unumeral.NewInt(1)}, &rhs))
		out := make([]unumeral.Int, 1)
		budget.FailAfter(fail)
		ok, e := s.Solve(out)
		a := out[0]
		linearNativeNamedAudit(t, s, &rhs, &out[0])
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
		serviceMust(t, z.Del(&out[0]))
		serviceMust(t, s.Close())
		serviceMust(t, z.Del(&rhs))
		serviceMust(t, z.CleanupPendingAllocation())
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

func TestLinearModularNativeGrowthIndependentTriangularSystem(t *testing.T) {
	m, z, budget := modularManager(t, 97)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { budget.FailAfter(-1); serviceMust(t, s.Close()) }()
	beforeBytes, beforeStats := budget.Used(), z.Memory.Stats()
	budget.FailAfter(0)
	for _, n := range []uint32{129, 3, 257, 0, 17} {
		serviceMust(t, s.Resize(n))
		if budget.Used() != beforeBytes || z.Memory.Stats() != beforeStats {
			t.Fatal("native growth charged Runtime")
		}
	}
	// Build A*x independently with small signed integers. All diagonal entries
	// are one, so no pivot swap is needed and the chosen x is the unique solution.
	const n = 17
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < n; i++ {
			row := make([]unumeral.Int, n)
			sum := 0
			for j := i; j < n; j++ {
				a := (i+j)%5 - 2
				if i == j {
					a = 1
				}
				row[j] = unumeral.NewInt(int32(a))
				sum += a * (j%7 - 3)
			}
			sum = (sum%97 + 97) % 97
			if sum > 48 {
				sum -= 97
			}
			rhs := unumeral.NewInt(int32(sum))
			serviceMust(t, s.Add(uint32(i), row, &rhs))
		}
		out := make([]unumeral.Int, n)
		ok, e := s.Solve(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("unit triangular system is invertible")
		}
		for i, a := range out {
			if a != unumeral.NewInt(int32(i%7-3)) {
				t.Fatal("independent solution", pass, i, a)
			}
		}
		serviceMust(t, s.Reset())
	}
	if budget.Used() != beforeBytes || z.Memory.Stats() != beforeStats {
		t.Fatal("small native solver allocated Runtime storage")
	}
}

func TestLinearNativeBigSelfCopyAliases(t *testing.T) {
	q, _ := rationalPublicationDirectOwner(t, false)
	r, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, r.Resize(1))
	rhs := unumeral.NewRational()
	serviceMust(t, q.SetString(&rhs, "79228162514264337593543950337"))
	one := unumeral.NewRational()
	serviceMust(t, q.SetInt64(&one, 1))
	serviceMust(t, r.Add(0, []unumeral.Rational{one}, &rhs))
	serviceMust(t, r.Add(0, r.A[0], &r.b[0]))
	ok, e := r.Solve(r.b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("rational identity")
	}
	equal, e := q.Equal(&r.b[0], &rhs)
	serviceMust(t, e)
	if !equal {
		t.Fatal("rational self-copy lost BIG owner")
	}
	serviceMust(t, r.Close())
	serviceMust(t, q.Del(&rhs))
	serviceMust(t, q.Del(&one))
	m, z, _ := modularManager(t, 18446744073709551557)
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	b := unumeral.Int{}
	serviceMust(t, z.SetUint64(&b, 1<<40))
	serviceMust(t, s.Add(0, []unumeral.Int{unumeral.NewInt(1)}, &b))
	serviceMust(t, s.Add(0, s.A[0], &s.b[0]))
	ok, e = s.Solve(s.b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("modular identity")
	}
	equal, e = z.Equal(&s.b[0], &b)
	serviceMust(t, e)
	if !equal {
		t.Fatal("modular self-copy lost BIG owner")
	}
	serviceMust(t, s.Close())
	serviceMust(t, z.Del(&b))
}
