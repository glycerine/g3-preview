// Source: src/math/polynomial/polynomial.cpp:5114-5156;
// src/util/mpzzp.h:154,265-270; src/util/mpz.h:468-485.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialExactDivActualNUMAcquisitionUsesDirectFrame(t *testing.T) {
	saw := false
	for ordinal := int64(0); ordinal < 140; ordinal++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c := full.core
		var big util.Int
		words := make([]uint32, 80)
		words[0], words[79] = 2, 1
		serviceMust(t, c.integer.SetLimbs(&big, words))
		p, e := c.MkConstantTakeCopy(&big)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		two := util.NewInt(2)
		q, e := c.MkConstantTake(&two)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(q))
		var receipt polynomialExactDivInvocationReceipt
		c.integer.Budget().FailAfter(ordinal)
		result, body := c.exactDivWithReceipt(p, q, &receipt)
		t.Log("reached actual prefix", ordinal, body, "NUM", c.exactDivision.numeral, "phase", c.exactDivision.numericPhase, "output", c.som2.arithmeticScope.transport, "R", c.som.arithmeticScope.transport)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, result}, []*util.Int{&big}, 0)
		roots, e := c.integer.SourceAllocator().PendingRoots()
		serviceMust(t, e)
		pending := c.integer.PendingCell
		if body != nil && receipt.invocation != 0 && (pending.Object != (util.SourceCellBacking{}).Object || roots != ([3]umemory.Block{})) {
			ownedCell, ownedRaw := pending.Object == (util.SourceCellBacking{}).Object, roots == ([3]umemory.Block{})
			if c.matchesExactDivInvocation(&receipt) {
				for _, actual := range []*polynomialNumericRoots{&receipt.frame.numeric} {
					ownedCell = ownedCell || actual.Cell == pending
					ownedRaw = ownedRaw || actual.Allocator == roots
				}
				for _, actual := range []*polynomialArithmeticInvocationReceipt{&receipt.frame.output, &receipt.frame.directReceipt.prelude} {
					if c.matchesArithmeticInvocation(actual) {
						ownedCell = ownedCell || actual.frame.numeric.Cell == pending
						ownedRaw = ownedRaw || actual.frame.numeric.Allocator == roots
						if c.integer.MatchesAddMulInvocation(&actual.frame.addMul) {
							ownedCell = ownedCell || actual.frame.addMul.Frame.Numeric.Cell == pending
							ownedRaw = ownedRaw || actual.frame.addMul.Frame.Numeric.Allocator == roots
						}
					}
				}
			}
			if !ownedCell || !ownedRaw {
				t.Fatal("actual ExactDiv acquisition has no exact direct local/buffer frame", ordinal, body, pending, roots, receipt.frame.numericPhase)
			}
			if receipt.frame.numericPhase == 1 && ((pending.Object != (util.SourceCellBacking{}).Object && receipt.frame.numeric.Cell == pending) || (roots != ([3]umemory.Block{}) && receipt.frame.numeric.Allocator == roots)) {
				saw = true
				// At this actual NUM leaf, pending destruction needs no new
				// source allocation. Prove false/true ownership while the real
				// fault epoch is still exhausted, before lifting its budget.
				serviceMust(t, c.cleanupExactDivInvocation(&receipt, false))
				serviceMust(t, c.cleanupExactDivInvocation(&receipt, true))
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, result}, []*util.Int{&big}, 0)
			}
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupExactDivInvocation(&receipt, false))
		serviceMust(t, c.cleanupExactDivInvocation(&receipt, true))
		if body == nil {
			// Independent bit pattern: (2^(79*32)+2)/2.
			half := util.Int{}
			expectedWords := make([]uint32, 79)
			expectedWords[0], expectedWords[78] = 1, uint32(1)<<31
			serviceMust(t, c.integer.SetLimbs(&half, expectedWords))
			coefficient, e := c.coefficientCore(result, 0)
			serviceMust(t, e)
			equal, e := c.integer.Equal(&coefficient, &half)
			serviceMust(t, e)
			if !equal {
				t.Fatal("literal quotient coefficient")
			}
			serviceMust(t, c.integer.Del(&half))
			serviceMust(t, c.CleanupValue(&result))
		}
		serviceMust(t, c.DecRef(q))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&big))
		if body == nil {
			if !saw {
				t.Fatal("actual NUM acquisition refusal horizon not reached", ordinal)
			}
			return
		}
	}
	t.Fatal("every actual NUM prefix did not reach success")
}

// PageSize1 is a real configured descriptor topology: every new source cell
// reaches descriptor preparation after its direct payload allocation.
func polynomialExactDivDirectPageOneFixture(t *testing.T) *MultivariatePolynomialManager {
	budget := arena.NewBudget(1 << 24)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 1, Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, full.CloseChecked())
		serviceMust(t, q.CloseChecked())
		if budget.Used() != 0 {
			t.Error("actual named owner graph retained", budget.Used())
		}
	})
	return full
}

// The public nil-receipt route must use the same actual embedded acquisition
// frame; its own entry point retires the failed invocation without callee sweeps.
func TestPolynomialExactDivPublicOwnAcknowledgmentPermitsSameOwnerRetry(t *testing.T) {
	for ordinal := int64(0); ordinal < 140; ordinal++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c := full.core
		var big util.Int
		words := make([]uint32, 80)
		words[0], words[79] = 2, 1
		serviceMust(t, c.integer.SetLimbs(&big, words))
		p, e := c.MkConstantTakeCopy(&big)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		two := util.NewInt(2)
		q, e := c.MkConstantTake(&two)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(q))
		c.integer.Budget().FailAfter(ordinal)
		result, body := c.ExactDiv(p, q)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, result}, []*util.Int{&big}, 0)
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.CleanupExactDivision())
		t.Log("ack", ordinal, "NUMactive", c.exactDivision.active, "R", c.som.arithmeticScope.active, "C", c.som2.arithmeticScope.active, "generated", c.generatedActive, "prelude", c.exactDivision.directReceipt.prelude.invocation, "scratch", c.integer.Tmp)
		if body != nil {
			retry, e := c.ExactDiv(p, q)
			if e != nil {
				// The actual source scratch free-before-replacement failure leaves a
				// dangling C++ member; the Go source engine preserves a guarded hole.
				// Owning operation cleanup must not allocate a replacement or repair it.
				if (c.integer.Tmp[0] != 0 && c.integer.Tmp[1] != 0) || !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("initialized same-owner retry after exact acknowledgment", ordinal, body, e)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, retry}, []*util.Int{&big}, 0)
				serviceMust(t, c.CleanupExactDivision())
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, retry}, []*util.Int{&big}, 0)
			serviceMust(t, c.CleanupValue(&retry))
		} else {
			serviceMust(t, c.CleanupValue(&result))
		}
		serviceMust(t, c.DecRef(q))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&big))
		if body == nil {
			return
		}
	}
	t.Fatal("actual public reservation horizon did not reach success")
}
