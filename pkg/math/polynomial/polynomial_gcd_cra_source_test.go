// Source: src/math/polynomial/polynomial.cpp:3675-3767.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialGCDCRAIndependentBalancedCoefficientTable(t *testing.T) {
	for route := uint32(0); route < 3; route++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		a := polynomialQueryTerms(t, c, []int32{-1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		b := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
		want := map[[3]uint32]int64{{1, 0, 0}: 2, {}: 1}
		if route == 1 {
			a = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
			b = polynomialQueryTerms(t, c, []int32{-1, -2}, [][]MonomialPower{{{0, 1}}, nil})
			want = map[[3]uint32]int64{{1, 0, 0}: 4, {}: -2}
		}
		if route == 2 {
			a = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			b = polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
			want = map[[3]uint32]int64{{2, 0, 0}: -5, {0, 1, 0}: 6, {}: 7}
		}
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, r.Close()) })
		first, second := util.NewInt(3), util.NewInt(5)
		serviceMust(t, full.gcdCRA(a, &first, b, &second, r))
		if second != util.NewInt(15) || first != util.NewInt(3) {
			t.Fatal("source final bound publication", first, second)
		}
		polynomialQueryExpect(t, c, r.root.value, want)
	}
}

func TestPolynomialGCDCRAEveryActualBigNumeralPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			z := c.integer
			a := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
			b := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			var first util.Int
			second := util.NewInt(3)
			serviceMust(t, z.Copy(&first, &z.Two64))
			serviceMust(t, z.Inc(&first))
			t.Cleanup(func() { serviceMust(t, z.Del(&first)); serviceMust(t, z.Del(&second)) })
			t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, full.CleanupCRA()) })
			z.Budget().FailAfter(ordinal)
			e = full.gcdCRA(a, &first, b, &second, r)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual CRA prefix category", ordinal, e)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{a, b, r.root.value, r.root.incoming}, []*util.Int{&first, &second}, r.header)
			z.Budget().FailAfter(-1)
			serviceMust(t, full.CleanupCRA())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{a, b, r.root.value, r.root.incoming}, []*util.Int{&first, &second}, r.header)
			if e == nil {
				polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 1})
				success = true
				t.Log("actual CRA BIG endpoint", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("no completed actual CRA BIG route")
	}
}

// Guarded helper-domain fixture: real Copy frees its old cell then fails the
// replacement. It proves the reached source destructor/census boundary, not
// ordinary mathematical CRA reachability of a larger saved coefficient.
func TestPolynomialGCDCRACompletedNumeralGuardRetainsOlderRoots(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	serviceMust(t, c.pscEnter())
	s := &full.cra
	*s = polynomialCRAScope{active: true, topOwner: true, generation: c.pscExecutingOwner(), constructed: 3}
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, full.CleanupCRA()); serviceMust(t, z.Del(&large)) })
	for i := range 3 {
		serviceMust(t, z.Copy(&s.nums[i], &z.Two64))
	}
	older0, older1 := s.nums[0], s.nums[1]
	z.Budget().FailAfter(0)
	body := z.Copy(&s.nums[2], &large)
	if !numericPhysicalReservationRefusal(body) || (s.nums[2]).Cell != z.FailedRef {
		t.Fatal("actual dangling source prefix", body, s.nums[2])
	}
	source := full.craSourceEnd()
	if !errors.Is(source, arena.ErrInvalidSlot) || s.constructed != 3 || s.nums[0] != older0 || s.nums[1] != older1 {
		t.Fatal("stopped source reverse order", source, s.constructed)
	}
	c.pscLeave()
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	if again := full.craSourceEnd(); again != source {
		t.Fatal("source destructor replay", source, again)
	}
	if e := full.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("owning close mutated retained scope", e)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupCRA())
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}
