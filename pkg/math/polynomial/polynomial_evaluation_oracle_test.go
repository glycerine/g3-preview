// Source: src/math/polynomial/polynomial.cpp:6436-6550.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialEvaluationIndependentRationalGrid(t *testing.T) {
	terms := []polynomialEvalTerm{{3, [3]uint32{3, 1, 0}}, {-2, [3]uint32{1, 2, 1}}, {5, [3]uint32{0, 0, 2}}, {-7, [3]uint32{}}}
	m, p, _ := polynomialEvalFixture(t, terms)
	q := m.core.config.Rationals
	v := polynomialEvalRationalMap{manager: q}
	for i := range v.values {
		v.values[i] = util.NewRational()
	}
	out := util.NewRational()
	for x := int64(-2); x <= 2; x++ {
		for y := int64(-2); y <= 2; y++ {
			for z := int64(-2); z <= 2; z++ {
				values := [3]int64{x, y, z}
				for i, a := range values {
					serviceMust(t, q.SetFraction(&v.values[i], a, 2))
				}
				// Independent direct term summation, without source Horner grouping/sort.
				want := new(big.Rat)
				for _, term := range terms {
					value := new(big.Rat).SetInt64(int64(term.coefficient))
					for i, d := range term.degrees {
						for j := uint32(0); j < d; j++ {
							value.Mul(value, new(big.Rat).SetFrac(big.NewInt(values[i]), big.NewInt(2)))
						}
					}
					want.Add(want, value)
				}
				serviceMust(t, m.EvaluateRational(p, &v, &out))
				actual := rationalString(t, q, &out)
				if actual != want.RatString() {
					t.Fatal(values, actual, want.RatString())
				}
			}
		}
	}
	polynomialEvaluationAudit(t, m, p, &v, &out)
	for i := range v.values {
		serviceMust(t, q.Del(&v.values[i]))
	}
	serviceMust(t, q.Del(&out))
}
func TestPolynomialEvaluationIndependentModularGrid(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{5, 0, 0}}, {-2, [3]uint32{3, 0, 0}}, {4, [3]uint32{1, 0, 0}}, {-7, [3]uint32{}}})
	vm := m.core.modular
	serviceMust(t, vm.SetZPUint64(7))
	n, e := m.core.readWord(p, 8)
	serviceMust(t, e)
	for i := uint32(0); i < n; i++ {
		a, e := m.core.coefficientCore(p, i)
		serviceMust(t, e)
		serviceMust(t, vm.PNormalize(&a))
		serviceMust(t, m.core.writeCoefficient(p, i, a))
	}
	out := util.Int{}
	for x := int32(-3); x <= 3; x++ {
		value := util.Int{}
		serviceMust(t, vm.SetInt32(&value, x))
		serviceMust(t, m.UnivariateEvaluate(p, 0, &value, &out))
		actual, e := m.core.integer.Int64(&out)
		serviceMust(t, e)
		xx := int64(x)
		want := (3*xx*xx*xx*xx*xx - 2*xx*xx*xx + 4*xx - 7) % 7
		if want > 3 {
			want -= 7
		}
		if want < -3 {
			want += 7
		}
		if actual != want {
			t.Fatal(x, actual, want)
		}
	}
	serviceMust(t, vm.SetZ())
	serviceMust(t, m.core.integer.Del(&out))
}

func TestPolynomialEvaluationModularEveryReservation(t *testing.T) {
	failed, success := 0, false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{5, 0, 0}}, {3, [3]uint32{1, 0, 0}}, {7, [3]uint32{}}})
			vm := m.core.modular
			const modulus uint64 = 18446744073709551557
			serviceMust(t, vm.SetZPUint64(modulus))
			value, out := util.Int{}, util.Int{}
			serviceMust(t, vm.SetInt64(&value, (1<<60)+17))
			m.originalBudget.FailAfter(ordinal)
			e := m.UnivariateEvaluate(p, 0, &value, &out)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&value, &out}, 0)
			if e == nil {
				success = true
			} else {
				failed++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual modular evaluation source reservation", e)
				}
			}
			m.originalBudget.FailAfter(-1)
			serviceMust(t, m.CleanupEvaluation())
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&value, &out}, 0)
			if e == nil {
				x := big.NewInt((1 << 60) + 17)
				prime := new(big.Int).SetUint64(modulus)
				want := new(big.Int).Exp(x, big.NewInt(5), prime)
				want.Add(want, new(big.Int).Mul(x, big.NewInt(3)))
				want.Add(want, big.NewInt(7))
				want.Mod(want, prime)
				half := new(big.Int).Rsh(new(big.Int).Set(prime), 1)
				if want.Cmp(half) > 0 {
					want.Sub(want, prime)
				}
				actual, e := m.core.integer.Int64(&out)
				serviceMust(t, e)
				if actual != want.Int64() {
					t.Fatal("independent finite-ring evaluation", actual, want)
				}
			}
			serviceMust(t, m.core.integer.Del(&value))
			serviceMust(t, m.core.integer.Del(&out))
			serviceMust(t, vm.SetZ())
		})
		if success {
			break
		}
	}
	if failed == 0 || !success {
		t.Fatal("modular complete real prefix sweep", failed, success)
	}
}
