// Source: src/math/polynomial/polynomial.cpp:5094-5156.
package polynomial

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialExactDivIndependentProductAndScalar(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	quotient, e := m.ExactDiv(p, q)
	serviceMust(t, e)
	polynomialCaller(t, m, quotient)
	polynomialQueryExpect(t, m, quotient, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
	p2 := polynomialQueryTerms(t, m, []int32{6, -12}, [][]MonomialPower{{{1, 2}}, nil})
	c := util.NewInt(3)
	scalar, e := m.ExactDivScalar(p2, &c)
	serviceMust(t, e)
	polynomialCaller(t, m, scalar)
	polynomialQueryExpect(t, m, scalar, map[[3]uint32]int64{{0, 2, 0}: 2, {}: -4})
	zero, e := m.ExactDiv(m.zero, Polynomial{})
	serviceMust(t, e)
	if zero != m.zero {
		t.Fatal("zero source short circuit before divisor read", zero)
	}
}

func polynomialExactDivisionPhysical(t *testing.T, m *MultivariatePolynomialCore, polys []Polynomial, public ...*util.Int) {
	t.Helper()
	// Scope roots must be named by the shared census, without caller annotations.
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, nil, polys, nil, public...)
	polynomialOperationMonomialAudit(t, m, polys, nil)
}
func TestPolynomialExactDivCheckpointAfterActualInputCopy(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	visits := uint32(0)
	m.config.Checkpoint = func() bool { visits++; return false }
	_, e := m.ExactDiv(p, q)
	if !errors.Is(e, ErrUPolynomialCanceled) || visits != 1 {
		t.Fatal("actual first checkpoint", e, visits)
	}
	n, e := m.som.ms.size(m)
	serviceMust(t, e)
	if n != 2 || m.exactDivision.active {
		t.Fatal("R.add precedes checkpoint; completed empty locals unwind", n, m.exactDivision)
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q})
	polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{2, 0, 0}: 1, {}: -1})
	polynomialQueryExpect(t, m, q, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
}
func TestPolynomialExactDivPhysicalPrefixes(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		reached := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
			q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
			b := m.integer.Budget()
			b.FailAfter(ordinal)
			result, e := m.ExactDiv(p, q)
			polys := []Polynomial{p, q}
			if result != (Polynomial{}) {
				polys = append(polys, result)
			}
			polynomialExactDivisionPhysical(t, m, polys)
			polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{2, 0, 0}: 1, {}: -1})
			polynomialQueryExpect(t, m, q, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual allocation refusal", ordinal, e)
				}
				failures++
			} else {
				polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
				reached = true
			}
			b.FailAfter(-1)
			if m.exactDivision.active {
				serviceMust(t, m.CleanupExactDivision())
			}
			if reached {
				polynomialCaller(t, m, result)
			}
		})
		if reached {
			success = true
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal(failures, success)
	}
}

func TestPolynomialExactDivVerifySourceBoundary(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	q := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 2}}})
	visits := uint32(0)
	m.config.Checkpoint = func() bool { visits++; return true }
	_, e := m.ExactDiv(p, q)
	if visits != 1 {
		t.Fatal("VERIFY after first checkpoint", visits)
	}
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("VERIFY debug assertion", e)
		}
	} else {
		if !errors.Is(e, ErrPolynomialUnreachable) {
			t.Fatal("must stop at GNU compiler unreachable before null monomial reads", e)
		}
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q})
}

func TestPolynomialExactDivScalarRawLocalFailurePrefixes(t *testing.T) {
	failures, retained, success := 0, 0, false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		reached := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			big := util.Int{}
			serviceMust(t, m.integer.SetLimbs(&big, []uint32{2, 0, 0, 1}))
			defer func() { serviceMust(t, m.integer.Del(&big)) }()
			term, e := m.monomials.MkVariable(1, 2)
			serviceMust(t, e)
			unit, e := m.monomials.Unit()
			serviceMust(t, e)
			p, e := m.MkTerms([]util.Int{big, util.NewInt(6)}, []Monomial{term, unit})
			serviceMust(t, e)
			serviceMust(t, m.IncRef(p))
			defer func() { serviceMust(t, m.DecRef(p)) }()
			c := util.NewInt(3)
			b := m.integer.Budget()
			b.FailAfter(ordinal)
			result, e := m.ExactDivScalar(p, &c)
			named := []Polynomial{p}
			if result != (Polynomial{}) {
				named = append(named, result)
			}
			polynomialExactDivisionPhysical(t, m, named, &big, &c)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("real scalar allocation category", ordinal, e)
				}
				failures++
				if m.exactDivision.active {
					defer func() { b.FailAfter(-1); serviceMust(t, m.CleanupExactDivision()) }()
					beforeA, x := m.som.as.size(m)
					serviceMust(t, x)
					beforeM, x := m.som.ms.size(m)
					serviceMust(t, x)
					_, blocked := m.ExactDivScalar(m.zero, &c)
					if !errors.Is(blocked, ErrPolynomialPending) {
						t.Fatal("pending raw local source reuse", blocked)
					}
					afterA, x := m.som.as.size(m)
					serviceMust(t, x)
					afterM, x := m.som.ms.size(m)
					serviceMust(t, x)
					if beforeA != afterA || beforeM != afterM {
						t.Fatal("blocked source reuse mutated older R", beforeA, afterA, beforeM, afterM)
					}
					if m.exactDivision.kind != 2 || !m.matchesScalarExactDivInvocation(&m.exactDivision.scalarReceipt) {
						t.Fatal("actual scalar declaration retention", m.exactDivision)
					}
					if (m.exactDivision.numeral).Cell != 0 {
						retained++ // Retain the original positive heap-RAW assertion.
					} else if !m.exactDivDirectPending(&m.exactDivision) && !polynomialNegNumericPending(&m.exactDivision.numeric) && !m.matchesArithmeticInvocation(&m.exactDivision.scalarReceipt.prelude) {
						t.Fatal("inline RAW retained without its actual acquired child")
					}
					if !errors.Is(m.CloseChecked(), ErrPolynomialPending) {
						t.Fatal("owning abort must be explicit")
					}
				}
			} else {
				reached = true
				a, e := m.Coefficient(result, 0)
				serviceMust(t, e)
				if roundPublicationDecode(t, m.integer, &a).String() != "26409387504754779197847983446" {
					t.Fatal("independent 96-bit quotient")
				}
				a, e = m.Coefficient(result, 1)
				serviceMust(t, e)
				if a != util.NewInt(2) {
					t.Fatal("constant quotient", a)
				}
			}
			b.FailAfter(-1)
			if m.exactDivision.active {
				serviceMust(t, m.CleanupExactDivision())
			}
			if reached {
				serviceMust(t, m.IncRef(result))
				defer func() { serviceMust(t, m.DecRef(result)) }()
			}
		})
		if reached {
			success = true
			break
		}
	}
	if failures == 0 || retained == 0 || !success {
		t.Fatal(failures, retained, success)
	}
}

func TestPolynomialExactDivScalarDiagnosticsAndTruncation(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, nil})
	c := util.NewInt(2)
	quotient, e := m.ExactDivScalar(p, &c)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("divides source assertion", e)
		}
	} else {
		serviceMust(t, e)
		polynomialCaller(t, m, quotient)
		polynomialQueryExpect(t, m, quotient, map[[3]uint32]int64{{2, 0, 0}: 1, {}: 2})
	}
	zero := util.NewInt(0)
	quotient, e = m.ExactDivScalar(m.zero, &zero)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("scalar assertion before zero numerator", e)
		}
	} else {
		serviceMust(t, e)
		polynomialCaller(t, m, quotient)
		iszero, e := m.IsZero(quotient)
		serviceMust(t, e)
		if !iszero {
			t.Fatal("release empty loop before scalar arithmetic")
		}
	}
}
func TestPolynomialExactDivWarmSourceProfileAndCounts(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	visits := uint32(0)
	m.config.Checkpoint = func() bool { visits++; return true }
	operation := func() {
		result, e := m.ExactDiv(p, q)
		if e != nil {
			panic(e)
		}
		if e = m.IncRef(result); e != nil {
			panic(e)
		}
		if e = m.DecRef(result); e != nil {
			panic(e)
		}
	}
	for i := 0; i < 20; i++ {
		operation()
	}
	visits = 0
	before := numericWarmProfileSnapshot(t, m.integer, m.integer.Budget())
	monoBefore := m.monomials.allocator.Stats()
	goAllocs := testing.AllocsPerRun(20, operation)
	// AllocsPerRun includes one unmeasured primer invocation.
	if visits != 63 {
		t.Fatal("source checkpoint once per loop including empty R", visits)
	}
	direct := monoBefore.DirectObjects != 0
	numericWarmProfileAudit(t, m.integer, m.integer.Budget(), before, goAllocs, direct)
	if m.monomials.allocator.Stats() != monoBefore || m.exactDivision.active {
		t.Fatal("retained actual graph", monoBefore, m.monomials.allocator.Stats(), m.exactDivision)
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q})
}

func TestPolynomialExactDivActualModularChild(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.modular.SetZPUint64(7))
	// (3x+1)(-2x+2) = x²-3x+2 in the source's centered Z7 profile.
	p := polynomialQueryTerms(t, m, []int32{1, -3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{-2, 2}, [][]MonomialPower{{{0, 1}}, nil})
	result, e := m.ExactDiv(p, q)
	serviceMust(t, e)
	polynomialCaller(t, m, result)
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 1})
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q, result})
}

// This explicitly malformed requested-size metadata is not valid source input
// or memory exhaustion. It makes the actual 32-bit scoped numeral release
// refuse before the older monomial_ref destructor is reached.
func TestPolynomialExactDivReverseDestructionRefusal(t *testing.T) {
	t.Run("body exception", func(t *testing.T) { polynomialExactDivRefusal(t, false) })
	t.Run("evaluated return", func(t *testing.T) { polynomialExactDivRefusal(t, true) })
}
func polynomialExactDivRefusal(t *testing.T, returnExpression bool) {
	b := arena.NewBudget(1 << 24)
	runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 32}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: runtime, Profile: util.SmallAllocatorProfile{PointerBits: 32}})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	big := util.Int{}
	serviceMust(t, m.integer.SetLimbs(&big, []uint32{2, 0, 0, 1}))
	term, e := mm.MkVariable(0, 2)
	serviceMust(t, e)
	unit, e := mm.Unit()
	serviceMust(t, e)
	negative := util.Int{}
	serviceMust(t, m.integer.Copy(&negative, &big))
	serviceMust(t, m.integer.Neg(&negative))
	p, e := m.MkTerms([]util.Int{big, negative}, []Monomial{term, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	linear, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	divisor, e := m.MkTerms([]util.Int{util.NewInt(1), util.NewInt(1)}, []Monomial{linear, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(divisor))
	// Cleanup is explicit and restores only malformed fixture metadata.
	var token util.NumericCellRef
	var original util.NumericCell
	defer func() {
		if token != 0 && m.integer.IsLiveCellToken(uint64(token)) {
			cell := m.integer.Cell(token)
			cell.Backing.Requested = original.Backing.Requested
			m.integer.Store(token, cell)
		}
		serviceMust(t, m.CleanupExactDivision())
		serviceMust(t, m.DecRef(divisor))
		serviceMust(t, m.DecRef(p))
		serviceMust(t, m.integer.Del(&big))
		serviceMust(t, m.integer.Del(&negative))
	}()
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, mm.DecManagerRef())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Fatal("complete owning refund", b.Used())
		}
	})
	visits, before, resetReferences := uint32(0), uint32(0), uint32(0)
	m.config.Checkpoint = func() bool {
		visits++
		stop := uint32(2)
		if returnExpression {
			stop = 3
		}
		if visits != stop {
			return true
		}
		token = (m.exactDivision.numeral).Cell
		if token == 0 {
			t.Fatal("completed actual BIG local")
		}
		original = m.integer.Cell(token)
		corrupt := original
		corrupt.Backing.Requested = uint64(math.MaxUint32) + 1
		m.integer.Store(token, corrupt)
		before, e = m.monomialRefCount(m.exactDivision.monomial.value)
		serviceMust(t, e)
		if returnExpression {
			n, e := m.som.ms.size(m)
			serviceMust(t, e)
			for i := uint32(0); i < n; i++ {
				term, e := m.som.ms.monomial(m, i)
				serviceMust(t, e)
				if term == m.exactDivision.monomial.value {
					resetReferences++
				}
			}
		}
		return returnExpression
	}
	_, e = m.ExactDiv(p, divisor)
	if !errors.Is(e, arena.ErrBounds) || (!returnExpression && !errors.Is(e, ErrUPolynomialCanceled)) {
		t.Fatal("original and reached cleanup causes", e)
	}
	s := &m.exactDivision
	if !s.active || s.stage != 0 || (s.numeral).Cell != token || s.monomial.value == (Monomial{}) {
		t.Fatal("younger stop preserves both actual locals", s)
	}
	after, e := m.monomialRefCount(s.monomial.value)
	serviceMust(t, e)
	if after != before-resetReferences {
		t.Fatal("older destructor must remain unreached after literal R.reset", before, after, resetReferences)
	}
	if returnExpression && s.returned == (Polynomial{}) {
		t.Fatal("actual evaluated return must remain owned before failed destruction")
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, divisor}, &big, &negative)
	if returnExpression {
		a, e := m.Coefficient(s.returned, 0)
		serviceMust(t, e)
		if roundPublicationDecode(t, m.integer, &a).String() != "79228162514264337593543950338" {
			t.Fatal("unpublished result exact leading coefficient")
		}
	}
	// Reached state is audited above, before restoring the guarded fixture.
	cell := m.integer.Cell(token)
	cell.Backing.Requested = original.Backing.Requested
	m.integer.Store(token, cell)
	serviceMust(t, m.CleanupExactDivision())
}
