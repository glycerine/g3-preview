// Source: src/math/polynomial/polynomial.cpp:5094-5114.
package polynomial

import (
	mathbig "math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// The scalar overload's tmp is a real RAW declaration. Its direct division
// acquisitions must remain owned by that exact invocation on a refused leaf.
func TestPolynomialScalarExactDivActualNUMAllocationAuthority(t *testing.T) {
	scalarExactDivActualNUMPrefixes(t, false)
}
func TestPolynomialScalarExactDivNegativeActualRAWChildAllocationAuthority(t *testing.T) {
	scalarExactDivActualNUMPrefixes(t, true)
}
func scalarExactDivActualNUMPrefixes(t *testing.T, negative bool) {
	saw := false
	for ordinal := int64(0); ordinal < 140; ordinal++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c, z := full.core, full.core.integer
		var big util.Int
		words := make([]uint32, 80)
		words[0], words[79] = 2, 1
		serviceMust(t, z.SetLimbs(&big, words))
		if negative {
			serviceMust(t, z.Neg(&big))
		}
		p, e := c.MkConstantTakeCopy(&big)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		two := util.NewInt(2)
		z.Budget().FailAfter(ordinal)
		result, body := c.ExactDivScalar(p, &two)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*util.Int{&big}, 0)
		if body != nil && c.exactDivision.endEntered {
			t.Log("actual scalar end frontier", ordinal, "NUM stage", c.exactDivision.stage, "source stop", c.exactDivision.sourceCause)
		}
		pending := z.PendingCell
		raw, e := z.SourceAllocator().PendingRoots()
		serviceMust(t, e)
		acquired := pending.Object != (util.ObjectRef{}) || raw != ([3]util.Block{})
		owned := !acquired || c.exactDivision.active && c.exactDivision.kind == 2 &&
			(pending.Object == (util.ObjectRef{}) || c.exactDivision.numeric.Cell == pending) &&
			(raw == ([3]util.Block{}) || c.exactDivision.numeric.Allocator == raw)
		// Child RAW is allowed to own its own actual declarations, but the
		// parent's saved child tuple must identify that exact returned call.
		if acquired && z.MatchesRawInvocation(&c.exactDivision.divisionRaw) {
			owned = (pending.Object == (util.ObjectRef{}) || c.exactDivision.divisionRaw.Frame.Numeric.Cell == pending) &&
				(raw == ([3]util.Block{}) || c.exactDivision.divisionRaw.Frame.Numeric.Allocator == raw)
		}
		// R.add is an actual child after direct NUM division. Its matched
		// arithmetic receipt owns its own copy/publication acquisitions.
		r := &c.exactDivision.scalarReceipt.prelude
		if acquired && c.matchesArithmeticInvocation(r) {
			owned = owned || (pending.Object == (util.ObjectRef{}) || r.frame.numeric.Cell == pending) &&
				(raw == ([3]util.Block{}) || r.frame.numeric.Allocator == raw)
		}
		saw = saw || acquired
		if body == nil {
			a, e := c.coefficientCore(result, 0)
			serviceMust(t, e)
			want := new(mathbig.Int).Lsh(mathbig.NewInt(1), 79*32-1)
			want.Add(want, mathbig.NewInt(1))
			if negative {
				want.Neg(want)
			}
			if got := roundPublicationDecode(t, z, &a); got.Cmp(want) != 0 {
				t.Fatal("independent scalar quotient", got, want)
			}
		}
		z.Budget().FailAfter(-1)
		serviceMust(t, c.CleanupExactDivision())
		if result != (Polynomial{}) {
			serviceMust(t, c.CleanupValue(&result))
		}
		serviceMust(t, c.DecRef(p))
		serviceMust(t, z.Del(&big))
		if !owned {
			t.Fatal("scalar ExactDiv actual NUM acquisition lacks its direct frame/child", ordinal, body, "scope", c.exactDivision.active, "kind", c.exactDivision.kind)
		}
		if body == nil {
			if !saw {
				t.Fatal("actual scalar NUM refusal horizon not reached")
			}
			t.Log("actual scalar NUM and R child first success", ordinal)
			return
		}
	}
	t.Fatal("scalar actual allocation prefixes did not reach success")
}

func TestPolynomialScalarExactDivFieldActualCalleePrefixes(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c, z := full.core, full.core.integer
		var modulus util.Int
		words := make([]uint32, 17)
		for i := range words {
			words[i] = ^uint32(0)
		}
		words[16] = 511
		serviceMust(t, z.SetLimbs(&modulus, words))
		serviceMust(t, c.modular.SetZP(&modulus))
		two := util.NewInt(2)
		z.Budget().FailAfter(ordinal)
		result, body := c.ExactDivScalar(c.unit, &two)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{result}, []*util.Int{&modulus}, 0)
		if body == nil {
			a, e := c.coefficientCore(result, 0)
			serviceMust(t, e)
			want := new(mathbig.Int).Lsh(mathbig.NewInt(1), 520)
			want.Sub(want, mathbig.NewInt(1))
			want.Neg(want)
			if got := roundPublicationDecode(t, z, &a); got.Cmp(want) != 0 {
				t.Fatal("independent centered Mersenne field inverse", got, want)
			}
		} else if !numericPhysicalReservationRefusal(body) {
			t.Fatal("actual scalar field refusal", ordinal, body)
		}
		z.Budget().FailAfter(-1)
		serviceMust(t, c.CleanupExactDivision())
		if result != (Polynomial{}) {
			serviceMust(t, c.CleanupValue(&result))
		}
		serviceMust(t, z.Del(&modulus))
		if body == nil {
			t.Log("actual scalar field first success", ordinal)
			return
		}
	}
	t.Fatal("actual scalar field prefix campaign did not reach success")
}
