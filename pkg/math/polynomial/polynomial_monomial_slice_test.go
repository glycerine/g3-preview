// Source: src/math/polynomial/polynomial.cpp:193-544.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
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
			serviceMust(t, m.tempReset(tmp))
			for i := uint32(0); i < n; i++ {
				serviceMust(t, m.tempPushBack(tmp, MonomialPower{17*i + 3, i%7 + 1}))
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
			if _, e := m.tempPower(tmp, tmp.capacity); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("source capacity bounds", e)
			}
			serviceMust(t, m.tempSetSize(tmp, n/2))
			serviceMust(t, m.tempSetSize(tmp, n))
			for i := n / 2; i < n; i++ {
				got, e := m.tempPower(tmp, i)
				serviceMust(t, e)
				if got != (MonomialPower{17*i + 3, i%7 + 1}) {
					t.Fatal("source reset/set_size lost retained storage", i, got)
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

func TestMonomialTemporaryCapacityAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			action := choice
			if choice == "disabled" {
				action = "T"
			}
			polynomialAssertionActionsFixture(t, action+action+action)
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}

			var initTemp monomialTemp
			input := make([]MonomialPower, monomialTempInitialCapacity)
			for i := range input {
				input[i] = MonomialPower{uint32(i), 1}
			}
			e := m.tempInit(&initTemp, input)
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) || initTemp.size != 0 || initTemp.capacity != monomialTempInitialCapacity {
					t.Fatal("equal-capacity init did not stop before size/copy", e, initTemp.size, initTemp.capacity)
				}
			} else {
				serviceMust(t, e)
				if initTemp.size != monomialTempInitialCapacity || initTemp.powers[127] != input[127] {
					t.Fatal("init Continue/disabled/release", initTemp.size, initTemp.powers[127])
				}
			}

			var sizeTemp monomialTemp
			e = m.tempSetSize(&sizeTemp, monomialTempInitialCapacity+1)
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) || sizeTemp.size != 0 || sizeTemp.capacity != monomialTempInitialCapacity {
					t.Fatal("set_size assertion mutation", e, sizeTemp.size, sizeTemp.capacity)
				}
			} else {
				serviceMust(t, e)
				if sizeTemp.size != monomialTempInitialCapacity+1 || sizeTemp.capacity != monomialTempInitialCapacity+1 {
					t.Fatal("safe continued set_size adaptation", sizeTemp.size, sizeTemp.capacity)
				}
			}

			var powerTemp monomialTemp
			e = m.tempSetPower(&powerTemp, monomialTempInitialCapacity, MonomialPower{9, 7})
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) || powerTemp.size != 0 || powerTemp.capacity != monomialTempInitialCapacity {
					t.Fatal("set_power assertion mutation", e, powerTemp.size, powerTemp.capacity)
				}
			} else {
				serviceMust(t, e)
				if powerTemp.size != 0 || powerTemp.capacity != 2*monomialTempInitialCapacity || powerTemp.powers[128] != (MonomialPower{9, 7}) {
					t.Fatal("safe continued set_power adaptation", powerTemp.size, powerTemp.capacity, powerTemp.powers[128])
				}
			}
			serviceMust(t, m.tempCleanup(&initTemp))
			serviceMust(t, m.tempCleanup(&sizeTemp))
			serviceMust(t, m.tempCleanup(&powerTemp))
		})
	}
}

func TestMonomialTemporaryDestructiveReserveAndPreservingExpand(t *testing.T) {
	m := monomialFixture(t)
	var tmp monomialTemp
	serviceMust(t, m.tempPushBack(&tmp, MonomialPower{3, 5}))
	serviceMust(t, m.tempPushBack(&tmp, MonomialPower{7, 11}))
	tmp.hash = 99
	serviceMust(t, m.tempReserve(&tmp, 129))
	if tmp.size != 0 || tmp.capacity != 258 || tmp.hash != 0 || tmp.powers[0] != (MonomialPower{}) {
		t.Fatal("increase_capacity did not replace the old temporary", tmp.size, tmp.capacity, tmp.hash, tmp.powers[0])
	}
	for i := uint32(0); i < tmp.capacity; i++ {
		serviceMust(t, m.tempPushBack(&tmp, MonomialPower{i + 1, i + 2}))
	}
	tmp.hash = 77
	serviceMust(t, m.tempPushBack(&tmp, MonomialPower{999, 1000}))
	if tmp.size != 259 || tmp.capacity != 516 || tmp.hash != 0 || tmp.powers[0] != (MonomialPower{1, 2}) || tmp.powers[258] != (MonomialPower{999, 1000}) {
		t.Fatal("expand_capacity did not preserve the active prefix", tmp.size, tmp.capacity, tmp.hash, tmp.powers[0], tmp.powers[258])
	}
	serviceMust(t, m.tempCleanup(&tmp))
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

func TestMonomialArithmeticReservesBeyondInitialTemporaryCapacity(t *testing.T) {
	m := monomialFixture(t)
	powers := make([]MonomialPower, 129)
	for i := range powers {
		_, e := m.MkVar()
		serviceMust(t, e)
		powers[i] = MonomialPower{uint32(i), 1}
	}
	base := monomialOwn(t, m, powers)
	squared, e := m.Mul(base, base)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(squared))
	t.Cleanup(func() { serviceMust(t, m.DecRef(squared)) })
	if m.temps[1].capacity != 4*uint32(len(powers)) {
		t.Fatal("source product reserve did not replace capacity-128 scratch", m.temps[1].capacity)
	}
	for i := range powers {
		powers[i].Degree = 2
	}
	monomialEqualPowers(t, m, squared, powers)

	powered, e := m.Power(base, 2)
	serviceMust(t, e)
	if powered != squared {
		t.Fatal("large power did not intern product", powered, squared)
	}
	root, ok, e := m.Sqrt(squared)
	serviceMust(t, e)
	if !ok || root != base {
		t.Fatal("large square root", root, ok)
	}
	quotient, ok, e := m.Divide(squared, base)
	serviceMust(t, e)
	if !ok || quotient != base {
		t.Fatal("large division", quotient, ok)
	}
	g, q1, q2, e := m.GCD(squared, base)
	serviceMust(t, e)
	if g != base || q1 != base || q2 != m.unit {
		t.Fatal("large gcd", g, q1, q2)
	}
	removed, e := m.RemoveVariable(base, 0)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(removed))
	t.Cleanup(func() { serviceMust(t, m.DecRef(removed)) })
	if info, e := m.Info(removed); e != nil || info.Size != 128 {
		t.Fatal("large variable transform", info, e)
	}
	monomialPhysicalAudit(t, m)
}
