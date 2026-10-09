// Source: src/math/polynomial/linear_eq_solver.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func linearDefaultPhysicalCensus(t *testing.T, q *util.RationalManager, s *RationalLinearSolver, baseline uint32) {
	t.Helper()
	if q.Integers().Memory.Stats().LiveBlocks != baseline {
		t.Fatal("default native matrix allocated Runtime blocks")
	}
	want := q.HeaderBytes + integerActualOwnedBytes(q.Integers()) + s.header
	if q.Budget().Used() != want {
		t.Fatal("actual default owner census", q.Budget().Used(), want)
	}
}

func TestLinearRationalNativeResizeAndReuse(t *testing.T) {
	q, budget := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	baseline := q.Integers().Memory.Stats().LiveBlocks
	budget.FailAfter(0)
	for _, n := range []uint32{129, 3, 257, 0, 17} {
		serviceMust(t, s.Resize(n))
		linearDefaultPhysicalCensus(t, q, s, baseline)
		// Rational defaults must be 0/1, including all newly grown matrix cells.
		for i := range s.A {
			for _, a := range s.A[i] {
				if *a.BorrowNumerator() != util.NewInt(0) || *a.BorrowDenominator() != util.NewInt(1) {
					t.Fatal("default rational", i, a)
				}
			}
			if *s.b[i].BorrowNumerator() != util.NewInt(0) || *s.b[i].BorrowDenominator() != util.NewInt(1) {
				t.Fatal("default RHS", i, s.b[i])
			}
		}
		serviceMust(t, s.Reset())
	}
	serviceMust(t, s.Close())
	linearDefaultPhysicalCensus(t, q, s, baseline)
	budget.FailAfter(-1)
}

func TestLinearRationalIndependentFiniteTwoByTwo(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	rows := [2][2]util.Rational{{util.NewRational(), util.NewRational()}, {util.NewRational(), util.NewRational()}}
	rhs := [2]util.Rational{util.NewRational(), util.NewRational()}
	xs := [2]util.Rational{util.NewRational(), util.NewRational()}
	for a := int32(-2); a <= 2; a++ {
		if a == 0 {
			continue
		}
		for b := int32(-2); b <= 2; b++ {
			for c := int32(-2); c <= 2; c++ {
				for d := int32(-2); d <= 2; d++ {
					det := a*d - b*c
					if det == 0 {
						continue
					}
					for i, row := range [2][2]int32{{a, b}, {c, d}} {
						for j, v := range row {
							serviceMust(t, q.SetInt32(&rows[i][j], v))
						}
					}
					serviceMust(t, q.SetInt32(&rhs[0], -3))
					serviceMust(t, q.SetInt32(&rhs[1], 5))
					serviceMust(t, s.Add(0, rows[0][:], &rhs[0]))
					serviceMust(t, s.Add(1, rows[1][:], &rhs[1]))
					ok, e := s.Solve(xs[:])
					serviceMust(t, e)
					if !ok {
						t.Fatal("nonsingular independent determinant")
					}
					for i, num := range [2]int32{-3*d - 5*b, 5*a + 3*c} {
						want := util.NewRational()
						serviceMust(t, q.SetFractionSigned(&want, num, det))
						eq, e := q.Equal(&xs[i], &want)
						serviceMust(t, e)
						if !eq {
							t.Fatal("independent Cramer oracle", a, b, c, d, i)
						}
						serviceMust(t, q.Del(&want))
					}
				}
			}
		}
	}
	for i := range xs {
		serviceMust(t, q.Del(&xs[i]))
		serviceMust(t, q.Del(&rhs[i]))
		for j := range rows[i] {
			serviceMust(t, q.Del(&rows[i][j]))
		}
	}
	serviceMust(t, s.Close())
}

func TestLinearRationalSourceBackSubstitutionOutputGuard(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	one := util.NewRational()
	serviceMust(t, q.SetInt64(&one, 1))
	zero := util.NewRational()
	b0, b1 := util.NewRational(), util.NewRational()
	serviceMust(t, q.SetInt64(&b0, 5))
	serviceMust(t, q.SetInt64(&b1, 2))
	serviceMust(t, s.Add(0, []util.Rational{one, one}, &b0))
	serviceMust(t, s.Add(1, []util.Rational{zero, one}, &b1))
	xs := []util.Rational{util.NewRational()}
	serviceMust(t, q.SetInt64(&xs[0], 77))
	ok, e := s.Solve(xs)
	if ok || !errors.Is(e, arena.ErrBounds) {
		t.Fatal("descending reached result guard", ok, e)
	}
	want := util.NewRational()
	serviceMust(t, q.SetInt64(&want, 77))
	eq, e := q.Equal(&xs[0], &want)
	serviceMust(t, e)
	if !eq {
		t.Fatal("unreached lower output changed")
	}
	actual, e := s.RightHandSide(0)
	serviceMust(t, e)
	eq, e = q.Equal(&actual, &b0)
	serviceMust(t, e)
	if !eq {
		t.Fatal("back substitution ran before output publication")
	}
	for _, a := range []*util.Rational{&one, &zero, &b0, &b1, &xs[0], &want} {
		serviceMust(t, q.Del(a))
	}
	serviceMust(t, s.Close())
}

func TestLinearModularIndependentFiniteTwoByTwo(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := util.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	serviceMust(t, m.SetZPUint64(7))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	// An independent bounded modular inverse oracle, without calling field
	// division or reproducing Gaussian elimination.
	for a := int64(1); a < 7; a++ {
		for b := int64(0); b < 7; b++ {
			for c := int64(0); c < 7; c++ {
				for d := int64(0); d < 7; d++ {
					det := ((a*d-b*c)%7 + 7) % 7
					if det == 0 {
						continue
					}
					inverse := int64(0)
					for k := int64(1); k < 7; k++ {
						if det*k%7 == 1 {
							inverse = k
							break
						}
					}
					rows := [2][2]util.Int{}
					rhs := [2]util.Int{}
					xs := [2]util.Int{}
					for i, row := range [2][2]int64{{a, b}, {c, d}} {
						for j, v := range row {
							serviceMust(t, m.SetInt64(&rows[i][j], v))
						}
					}
					serviceMust(t, m.SetInt64(&rhs[0], 3))
					serviceMust(t, m.SetInt64(&rhs[1], 5))
					serviceMust(t, s.Add(0, rows[0][:], &rhs[0]))
					serviceMust(t, s.Add(1, rows[1][:], &rhs[1]))
					ok, e := s.Solve(xs[:])
					serviceMust(t, e)
					if !ok {
						t.Fatal("independent invertible matrix", a, b, c, d)
					}
					for i, n := range [2]int64{3*d - 5*b, 5*a - 3*c} {
						want := util.Int{}
						serviceMust(t, m.SetInt64(&want, ((n*inverse)%7+7)%7))
						eq, e := m.Equal(&xs[i], &want)
						serviceMust(t, e)
						if !eq {
							t.Fatal("independent modular inverse oracle", a, b, c, d, i)
						}
						serviceMust(t, m.Del(&want))
						serviceMust(t, m.Del(&xs[i]))
						serviceMust(t, m.Del(&rhs[i]))
						for j := range rows[i] {
							serviceMust(t, m.Del(&rows[i][j]))
						}
					}
				}
			}
		}
	}
	serviceMust(t, s.Close())
	serviceMust(t, m.Close())
}

func linearArithmeticPhysicalCensus(t *testing.T, q *util.RationalManager, s *RationalLinearSolver, callers ...*util.Rational) {
	t.Helper()
	values := []util.Int{q.NTmp, q.AddTmp1, q.AddTmp2, q.AddMulTmp.Num, q.AddMulTmp.Den, q.LtTmp1.Num, q.LtTmp1.Den, q.LtTmp2.Num, q.LtTmp2.Den}
	for _, a := range callers {
		values = append(values, (*a.BorrowNumerator()), (*a.BorrowDenominator()))
	}
	for _, a := range s.b {
		values = append(values, *a.BorrowNumerator(), *a.BorrowDenominator())
	}
	for _, row := range s.A {
		for _, a := range row {
			values = append(values, *a.BorrowNumerator(), *a.BorrowDenominator())
		}
	}
	refs := make([]*util.Int, len(values))
	for i := range values {
		refs[i] = &values[i]
	}
	integerPhysicalAuditWithRaw(t, q.Integers(), nil, refs...)
	want := q.HeaderBytes + integerActualOwnedBytes(q.Integers()) + s.header
	if q.Budget().Used() != want {
		t.Fatal("complete arithmetic physical/cold census", q.Budget().Used(), want)
	}
}

func TestLinearRationalArithmeticEveryReservationPrefix(t *testing.T) {
	for _, solve := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 512; ordinal++ {
			q, b := rationalPublicationDirectOwner(t, false)
			s, e := NewRationalLinearSolver(q)
			serviceMust(t, e)
			serviceMust(t, s.Resize(2))
			input := [6]util.Rational{util.NewRational(), util.NewRational(), util.NewRational(), util.NewRational(), util.NewRational(), util.NewRational()}
			for i, text := range []string{"79228162514264337593543950337", "1", "1", "79228162514264337593543950338", "158456325028528675187087900677", "237684487542793012780631851016"} {
				serviceMust(t, q.SetString(&input[i], text))
			}
			outputs := [2]util.Rational{util.NewRational(), util.NewRational()}
			if solve {
				serviceMust(t, s.Add(0, input[:2], &input[4]))
				serviceMust(t, s.Add(1, input[2:4], &input[5]))
			}
			b.FailAfter(ordinal)
			if solve {
				var ok bool
				ok, e = s.Solve(outputs[:])
				if e == nil && !ok {
					t.Fatal("source nonzero pivots", ordinal)
				}
			} else {
				e = s.Add(0, input[:2], &input[4])
			}
			roots := []*util.Rational{&input[0], &input[1], &input[2], &input[3], &input[4], &input[5], &outputs[0], &outputs[1]}
			linearArithmeticPhysicalCensus(t, q, s, roots...)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual arithmetic reservation cause", solve, ordinal, e)
				}
				failures++
			} else {
				success = true
				if solve {
					for i, v := range []int64{2, 3} {
						want := util.NewRational()
						serviceMust(t, q.SetInt64(&want, v))
						equal, e := q.Equal(&outputs[i], &want)
						serviceMust(t, e)
						if !equal {
							t.Fatal("independent chosen solution", ordinal, i)
						}
						serviceMust(t, q.Del(&want))
					}
				}
			}
			b.FailAfter(-1)
			if closeErr := s.Close(); closeErr != nil {
				serviceMust(t, s.Close())
			}
			for _, a := range roots {
				serviceMust(t, q.Del(a))
			}
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("complete arithmetic owning refund", solve, ordinal, b.Used())
			}
			if success {
				t.Log("every actual refusal before success", solve, failures, ordinal)
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("bounded complete real arithmetic campaign", solve, success, failures)
		}
	}
}
