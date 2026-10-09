// Source: src/math/polynomial/polynomial.cpp:193-544.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMonomialNativeScratchGrowthResetAndBounds(t *testing.T) {
	m := monomialFixture(t)
	var borrowed monomialTemp
	scratch := []*monomialTemp{&m.temps[0], &m.temps[1], &m.temps[2], &m.temps[3], &borrowed}
	m.config.Budget.FailAfter(0)
	t.Cleanup(func() { m.config.Budget.FailAfter(-1) })
	before, used := m.memory.Stats(), m.config.Budget.Used()
	for _, tmp := range scratch {
		for _, n := range []uint32{0, 1, 128, 129, 4097, 33} {
			serviceMust(t, m.tempSetSize(tmp, 0))
			for i := uint32(0); i < n; i++ {
				serviceMust(t, m.tempSetPower(tmp, i, MonomialPower{17*i + 3, i%7 + 1}))
			}
			size, e := m.tempSize(tmp)
			serviceMust(t, e)
			if size != n {
				t.Fatal("native length", size, n)
			}
			for i := uint32(0); i < n; i++ {
				got, e := m.tempPower(tmp, i)
				serviceMust(t, e)
				if got != (MonomialPower{17*i + 3, i%7 + 1}) {
					t.Fatal("independent power", i, got)
				}
			}
			if _, e := m.tempPower(tmp, n); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("native bounds", e)
			}
			serviceMust(t, m.tempSetSize(tmp, n/2))
			serviceMust(t, m.tempSetSize(tmp, n))
			for i := n / 2; i < n; i++ {
				got, e := m.tempPower(tmp, i)
				serviceMust(t, e)
				if got != (MonomialPower{}) {
					t.Fatal("regrown scalar", i, got)
				}
			}
		}
		input := []MonomialPower{{3, 7}, {9, 11}}
		serviceMust(t, m.tempInit(tmp, input))
		input[0] = MonomialPower{99, 99}
		if tmp.powers[0] != (MonomialPower{3, 7}) {
			t.Fatal("scratch aliases borrowed input")
		}
		serviceMust(t, m.tempCleanup(tmp))
		if tmp.powers != nil {
			t.Fatal("cleanup retained scratch")
		}
	}
	if m.memory.Stats() != before || m.config.Budget.Used() != used {
		t.Fatal("native scratch changed Runtime accounting")
	}
	monomialPhysicalAudit(t, m)
}

func TestMonomialNativeVariableCountingAndInterning(t *testing.T) {
	m := monomialFixture(t)
	powers := make([]MonomialPower, 129)
	var xs []uint32
	for v := range powers {
		_, e := m.MkVar()
		serviceMust(t, e)
		powers[v] = MonomialPower{uint32(v), uint32(v%7 + 1)}
	}
	expected := monomialOwn(t, m, powers)
	for v := len(powers) - 1; v >= 0; v-- {
		for j := uint32(0); j < powers[v].Degree; j++ {
			xs = append(xs, uint32(v))
		}
	}
	m.config.Budget.FailAfter(0)
	t.Cleanup(func() { m.config.Budget.FailAfter(-1) })
	before, used := m.memory.Stats(), m.config.Budget.Used()
	for attempt := 0; attempt < 2; attempt++ {
		got, e := m.MkFromVariables(xs)
		serviceMust(t, e)
		if got != expected {
			t.Fatal("canonical identity", got, expected)
		}
		monomialEqualPowers(t, m, got, powers)
		if len(m.powers) != len(powers) {
			t.Fatal("counted size", len(m.powers))
		}
		for i, p := range powers {
			if m.powers[i] != p {
				t.Fatal("counted power", i, m.powers[i], p)
			}
		}
		for i := 1; i < len(xs); i++ {
			if xs[i-1] > xs[i] {
				t.Fatal("caller variables not sorted", i)
			}
		}
	}
	if m.memory.Stats() != before || m.config.Budget.Used() != used {
		t.Fatal("native counting changed Runtime accounting")
	}
	monomialPhysicalAudit(t, m)
}
