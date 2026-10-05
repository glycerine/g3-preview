// Source: src/math/polynomial/polynomial.cpp:1988-2067.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialRegularSetActualDefaultSlotNumericAcquisition(t *testing.T) {
	saw := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		big := NewInt(0)
		limbs := make([]uint32, 80)
		limbs[0], limbs[79] = 1, 1
		serviceMust(t, c.integer.SetLimbs(&big, limbs))
		p, e := c.MkConstantTakeCopy(&big)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		// Ordinary caller copies consume the actual free descriptor slots, isolating
		// a later descriptor-preparation refusal after the source payload allocation.
		var spare [16]Int
		public := []*Int{&big}
		used := 0
		for c.integer.cells.Stats().Slots > c.integer.cells.Stats().Live+c.integer.cells.Stats().Retired || c.integer.cells.Stats().Slots < c.integer.cells.Stats().Pages*2 {
			if used == len(spare) {
				t.Fatal("bounded source setup did not fill descriptor page")
			}
			serviceMust(t, c.integer.Copy(&spare[used], &big))
			public = append(public, &spare[used])
			used++
		}
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		a := NewInt(2)
		c.integer.budget.FailAfter(ordinal)
		e = c.som.addSourceInputWithArithmeticReceipt(&a, c.monomials.unit, polynomialSOMInput{polynomial: p}, false, &r)
		t.Log("actual prefix", ordinal, e, "numeric", r.frame.numeric, "pending", c.integer.pendingCell, "slab", c.integer.cells.Stats())
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, public, 0)
		if c.integer.pendingCell.object != (sourceCellBacking{}).object {
			if r.frame.numeric.cell != c.integer.pendingCell {
				t.Fatal("actual SET descriptor acquisition absent from passed frame", ordinal, e)
			}
			saw = true
		}
		if r.frame.numeric.allocator != ([3]memory.Block{}) {
			saw = true
		}
		// No lexical NUM is constructed in this source buffer method. Exact unjoined
		// numerical leaves are separately owned; rows remain persistent members.
		if r.frame.executing {
			t.Fatal("actual source return retained executing acquisition context")
		}
		serviceMust(t, r.frame.numeric.cleanup(c.integer))
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, public, 0)
		c.integer.budget.FailAfter(-1)
		serviceMust(t, c.som.close())
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&big))
		for i := 0; i < used; i++ {
			serviceMust(t, c.integer.Del(&spare[i]))
		}
		if e == nil {
			if !saw {
				t.Fatal("no actual numeric acquisition refusal reached")
			}
			return
		}
	}
	t.Fatal("real refusal campaign did not reach success")
}

// A concrete direct allocator profile isolates the reached SET leaf refusal;
// the ordinary pooled source operation remains covered separately.
func polynomialArithmeticDirectNumericFixture(t *testing.T) *MultivariatePolynomialManager {
	t.Helper()
	budget := arena.NewBudget(1 << 24)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, full.CloseChecked())
		serviceMust(t, q.CloseChecked())
		if budget.Used() != 0 {
			t.Errorf("real owning graph retained %d", budget.Used())
		}
	})
	return full
}
