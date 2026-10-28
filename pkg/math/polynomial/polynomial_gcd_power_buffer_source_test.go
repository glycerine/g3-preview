package polynomial

// Source: src/math/polynomial/polynomial.cpp:3207-3256,4391-4392;
// src/util/buffer.h.

import (
	"fmt"
	"testing"
)

func TestPolynomialGCDPowerSortIndependentUnsignedPairs(t *testing.T) {
	for _, n := range []uint32{0, 1, 2, 3, 31, 32, 33, 127, 129} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			c := polynomialCellFixture(t)
			var b polynomialGCDPowers

			t.Cleanup(func() { b = nil })
			for i := n; i > 0; i-- {
				x := i - 1
				serviceMust(t, b.push(x, x*3+7))
			}
			b.sortVariables()
			for i := uint32(0); i < n; i++ {
				x, e := b.word(i, 0)
				serviceMust(t, e)
				k, e := b.word(i, 1)
				serviceMust(t, e)
				if x != i || k != i*3+7 {
					t.Fatal("independent sorted pair", n, i, x, k)
				}
			}
			polynomialPhysicalAudit(t, c)
			b = nil
			polynomialPhysicalAudit(t, c)
		})
	}
}
func TestPolynomialGCDPowerSortUnsignedBoundary(t *testing.T) {
	c := polynomialCellFixture(t)
	var b polynomialGCDPowers

	for _, x := range []uint32{^uint32(0), 0, 1 << 31, 7, 1 << 31, 0} {
		serviceMust(t, b.push(x, x^19))
	}
	b.sortVariables()
	for i, x := range []uint32{0, 0, 7, 1 << 31, 1 << 31, ^uint32(0)} {
		got, e := b.word(uint32(i), 0)
		serviceMust(t, e)
		degree, e := b.word(uint32(i), 1)
		serviceMust(t, e)
		if got != x || degree != x^19 {
			t.Fatal("unsigned lt_var and paired payload", i, got, degree)
		}
	}
	b = nil
	polynomialPhysicalAudit(t, c)
}

func TestPolynomialGCDPowerDegreeSortSourceAdapter(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	var b polynomialGCDPowers

	for i := uint32(0); i < 41; i++ {
		serviceMust(t, b.push(i+100, 40-i))
	}
	b.sortDegrees()
	for i := uint32(0); i < 41; i++ {
		degree, e := b.word(i, 1)
		serviceMust(t, e)
		variable, e := b.word(i, 0)
		serviceMust(t, e)
		if degree != i || variable != 140-i {
			t.Fatal("source degree-only order preserves pair", i, variable, degree)
		}
	}
	b = nil
	b = nil
	polynomialImpPhysicalAudit(t, full)
}

func TestPolynomialGCDPowersNativeGrowthAndReuse(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	state, stats, used := c.memory.State(), c.memory.Stats(), c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	defer c.integer.Budget().FailAfter(-1)
	var powers polynomialGCDPowers
	for i := uint32(0); i < 4097; i++ {
		serviceMust(t, powers.push(4096-i, i%17))
	}
	powers.sortVariables()
	for i, p := range powers {
		if p.Var != uint32(i) || p.Degree != (4096-uint32(i))%17 {
			t.Fatal("paired sort", i, p)
		}
	}
	powers.sortDegrees()
	counts := [17]int{}
	for i, p := range powers {
		if i > 0 && powers[i-1].Degree > p.Degree {
			t.Fatal("degree order", i)
		}
		if p.Degree != (4096-p.Var)%17 {
			t.Fatal("degree sort lost variable", i, p)
		}
		counts[p.Degree]++
	}
	for degree, n := range counts {
		want := 4097 / 17
		if degree < 4097%17 {
			want++
		}
		if n != want {
			t.Fatal("equal-degree multiplicity", degree, n, want)
		}
	}
	backing := &powers[0]
	var err error
	allocations := testing.AllocsPerRun(20, func() {
		powers = powers[:0]
		for i := uint32(0); i < 4097; i++ {
			if err = powers.push(i, 3); err != nil {
				return
			}
		}
	})
	if err != nil || allocations != 0 || &powers[0] != backing {
		t.Fatal("native reuse", err, allocations)
	}
	if c.memory.State() != state || c.memory.Stats() != stats || c.integer.Budget().Used() != used {
		t.Fatal("power workspace entered Runtime")
	}
	polynomialImpPhysicalAudit(t, full)
}

func TestPolynomialGCDVariableDegreesNativeCache(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	for i := 0; i < 3; i++ {
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
	}
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 3}, {1, 2}}, {{0, 1}, {2, 7}}})
	q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{1, 9}}})
	state, stats, used := c.memory.State(), c.memory.Stats(), c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	defer c.integer.Budget().FailAfter(-1)
	var powers polynomialGCDPowers
	for _, maximum := range []bool{true, false, true} {
		serviceMust(t, full.variableDegrees(p, &powers, maximum))
		powers.sortVariables()
		degree := uint32(1)
		if maximum {
			degree = 3
		}
		want := [3]MonomialPower{{0, degree}, {1, 2}, {2, 7}}
		if len(powers) != len(want) {
			t.Fatal("unique variables", powers)
		}
		for i, v := range want {
			if powers[i] != v {
				t.Fatal("degree collection", maximum, i, powers, want)
			}
		}
		for _, pos := range full.varDegrees {
			if pos != ^uint32(0) {
				t.Fatal("cache position not reset", pos)
			}
		}
		serviceMust(t, full.variableDegrees(q, &powers, maximum))
		if len(powers) != 1 || powers[0] != (MonomialPower{1, 9}) {
			t.Fatal("prior variables leaked", powers)
		}
	}
	if c.memory.State() != state || c.memory.Stats() != stats || c.integer.Budget().Used() != used {
		t.Fatal("degree collection entered Runtime")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
}
