package polynomial

// Source: src/math/polynomial/polynomial.cpp:5964-6068.
// Returning Runtime allocation-error transport is an explicit Go owner guard
// profile; native destructor reentry is not asserted.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSqrtActualAllocationTransportRefusesOwnerRetirement(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 0)
}

func TestPolynomialSqrtAppendAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 1)
}

func TestPolynomialSqrtScalarAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 2)
}

func TestPolynomialSqrtDivisionAllocationUsesActualChild(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 3)
}

func TestPolynomialSqrtPositiveDivisionAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 4)
}

func TestPolynomialSqrtMonomialAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 5)
}

func TestPolynomialSqrtFinalFactoryAllocationUsesActualInvocation(t *testing.T) {
	polynomialSqrtAllocationTransport(t, 6)
}

func polynomialSqrtAllocationTransport(t *testing.T, mode uint8) {
	t.Helper()
	budget := arena.NewBudget(1 << 26)
	var full *MultivariatePolynomialManager
	active := false
	calls := 0
	var rejected error
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			c := full.core
			used, stats := budget.Used(), c.memory.Stats()
			closing, stage := full.closing, full.cleanupStage
			inv := full.squareRoot.invocation
			rejected = full.CloseChecked()
			if rejected != ErrPolynomialPending || budget.Used() != used || c.memory.Stats() != stats || full.closing != closing || full.cleanupStage != stage || !full.squareRoot.executing || full.squareRoot.invocation != inv {
				t.Error("callback changed actual executing Sqrt owner", rejected, full.closing, full.cleanupStage)
			}
			if e := full.CleanupSqrt(); e != ErrPolynomialPending || budget.Used() != used || c.memory.Stats() != stats {
				t.Error("callback retired executing square-root resources", e)
			}
		}
		return f
	}})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	var mm *MonomialManager
	if mode >= 5 {
		mm, e = NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		serviceMust(t, mm.IncManagerRef())
	}
	full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	words := make([]uint32, 80)
	words[79] = 1
	var big unumeral.Int
	serviceMust(t, c.integer.SetLimbs(&big, words))
	var divisor unumeral.Int
	if mode == 3 || mode == 4 {
		digits := make([]uint32, 16)
		digits[0], digits[15] = 3, 1
		serviceMust(t, c.integer.SetLimbs(&divisor, digits))
		if mode == 3 {
			serviceMust(t, c.integer.Neg(&big))
		}
	}
	var p Polynomial
	var term Monomial
	if mode == 5 {
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		term, e = c.monomials.MkVariable(0, 2)
		serviceMust(t, e)
		p, e = c.MkTerms([]unumeral.Int{big}, []Monomial{term})
	} else {
		p, e = c.MkConstantTakeCopy(&big)
	}
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	if mode == 6 {
		coefficient := unumeral.NewInt(3)
		serviceMust(t, c.som.add(&coefficient, c.monomials.unit, false))
	}
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source frontier setup", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	var receipt polynomialSqrtInvocationReceipt
	var body error
	if mode >= 5 {
		serviceMust(t, full.beginSquareRootWithReceipt(&receipt))
		if mode == 5 {
			_, _, body = full.squareRootMonomial(0, term, Monomial{})
			r := &full.squareRoot.monomialAllocator
			if !c.monomials.matchesMonomialAllocatorInvocation(r) || r.outcome.Pending == (umemory.Block{}) {
				t.Fatal("actual monomial allocation absent from invocation receipt", body)
			}
		} else {
			_, body = c.som.mkWithFactoryReceipt(false, &full.squareRoot.factory)
			r := &full.squareRoot.factory
			if !c.matchesFactoryInvocation(r) || r.frame.allocation.Pending == (umemory.Block{}) {
				t.Fatal("actual final factory allocation absent from invocation receipt", body)
			}
		}
		body = full.finishSquareRoot(body)
	} else if mode != 0 {
		serviceMust(t, full.beginSquareRootWithReceipt(&receipt))
		if mode == 1 {
			body = full.squareRootAdd(&c.som, &big, c.monomials.unit)
		} else if mode == 2 {
			_, body = full.squareRootNumeral(2, &big, &big, &full.squareRoot.values[0])
		} else {
			_, body = full.squareRootNumeral(4, &big, &divisor, &full.squareRoot.values[0])
		}
		if mode == 3 && !c.integer.MatchesRawInvocation(&full.squareRoot.divisionRaw) {
			t.Fatal("actual division child absent from square-root invocation", body)
		}
		if mode != 3 && !polynomialNegNumericPending(&full.squareRoot.numeric) {
			t.Fatal("actual scalar/append allocation absent from invocation receipt", body)
		}
		if !full.squareRoot.numeric.NumericBound(true) {
			t.Fatal("actual executing square-root numeric policy is unbound")
		}
		body = full.finishSquareRoot(body)
		if !full.squareRoot.numeric.NumericBound(false) || full.squareRoot.numeric.NumericBound(true) {
			t.Fatal("completed numeric policy confused cleanup with execution")
		}
		foreign, e := unumeral.NewIntegerModularManager(c.integer)
		serviceMust(t, e)
		if full.squareRoot.numeric.NumericModularOwner(foreign) {
			t.Fatal("numeric policy accepted a foreign modular receiver")
		}
		serviceMust(t, foreign.Close())

	} else {
		_, body = full.sqrtWithReceipt(p, out, &receipt)
	}
	active = false
	if mode == 0 && !c.integer.MatchesPerfectSquareInvocation(&full.squareRoot.perfectSquare) {
		t.Fatal("actual perfect-square child absent from invocation", body)
	}
	if calls != 1 || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual square-root allocation transport not reached", calls, body)
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p}, []*unumeral.Int{&big, &divisor}, out.header, []umemory.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, full.cleanupSqrtInvocation(&receipt, false))
	serviceMust(t, full.cleanupSqrtInvocation(&receipt, true))
	if mode != 0 && full.squareRoot.numeric.NumericBound(false) {
		t.Fatal("retired numeric policy retained cleanup authority")
	}
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, c.integer.Del(&divisor))
	serviceMust(t, c.integer.Del(&big))
	serviceMust(t, c.DecRef(p))
	serviceMust(t, out.Close())
	serviceMust(t, full.CloseChecked())
	if mm != nil {
		serviceMust(t, mm.DecManagerRef())
	}
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Close())
	if budget.Used() != 0 {
		t.Fatal("square-root transport cleanup retained charge", budget.Used())
	}
}
