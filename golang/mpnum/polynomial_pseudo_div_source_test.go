// Source: src/math/polynomial/polynomial.cpp:4905-5091.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

func polynomialPseudoOutputs(t *testing.T, m *MultivariatePolynomialCore) (*PolynomialRef, *PolynomialRef) {
	t.Helper()
	q, e := NewPolynomialRef(m)
	serviceMust(t, e)
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()); serviceMust(t, q.Close()) })
	return q, r
}
func TestPolynomialPseudoDivisionIndependentIdentity(t *testing.T) {
	for _, exact := range []bool{false, true} {
		t.Run(map[bool]string{false: "minimal", true: "exact"}[exact], func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			divisor := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
			q, r := polynomialPseudoOutputs(t, m)
			d := uint32(91)
			var e error
			if exact {
				e = m.ExactPseudoDivide(p, divisor, 0, q, r)
			} else {
				e = m.PseudoDivide(p, divisor, 0, &d, q, r)
			}
			serviceMust(t, e)
			a, e := q.Get()
			serviceMust(t, e)
			b, e := r.Get()
			serviceMust(t, e)
			polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{1, 0, 0}: 2, {}: -1})
			polynomialQueryExpect(t, m, b, map[[3]uint32]int64{{}: 5})
			if !exact && d != 2 {
				t.Fatal("source completed iterations", d)
			}
			// Independent evaluation: 4*(x*x+1)=(2*x-1)*(2*x+1)+5.
			for x := int64(-7); x <= 7; x++ {
				if 4*(x*x+1) != (2*x-1)*(2*x+1)+5 {
					t.Fatal(x)
				}
			}
		})
	}
}
func TestPolynomialPseudoDivisionLowDegreeStillVisitsCheckpoint(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{7}, [][]MonomialPower{nil})
	divisor := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q, r := polynomialPseudoOutputs(t, m)
	d := uint32(91)
	visits := uint32(0)
	m.config.Checkpoint = func() bool { visits++; return false }
	e := m.PseudoDivide(p, divisor, 0, &d, q, r)
	if !errors.Is(e, ErrUPolynomialCanceled) || visits != 1 || d != 0 {
		t.Fatal("source continues after low degree publications", e, visits, d)
	}
	a, e := q.Get()
	serviceMust(t, e)
	b, e := r.Get()
	serviceMust(t, e)
	if a != m.zero || b != p {
		t.Fatal("actual Q0 R=A before checkpoint", a, b)
	}
}
func TestPolynomialPseudoDivisionConstantDivisorAndExactCorrection(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	divisor := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{nil})
	q, r := polynomialPseudoOutputs(t, m)
	d := uint32(91)
	serviceMust(t, m.PseudoDivide(p, divisor, 0, &d, q, r))
	a, e := q.Get()
	serviceMust(t, e)
	b, e := r.Get()
	serviceMust(t, e)
	if a != p || b != m.zero || d != 1 {
		t.Fatal("constant source no division", a, b, d)
	}
	serviceMust(t, m.ExactPseudoDivide(p, divisor, 0, q, r))
	a, e = q.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{2, 0, 0}: 4, {}: 4})
	// A single iteration drops degree by three. Exact adjustment multiplies by lc^2.
	cube := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 3}}})
	linear := polynomialQueryTerms(t, m, []int32{2}, [][]MonomialPower{{{0, 1}}})
	serviceMust(t, m.PseudoDivide(cube, linear, 0, &d, q, r))
	a, e = q.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{2, 0, 0}: 1})
	if d != 1 {
		t.Fatal(d)
	}
	serviceMust(t, m.ExactPseudoDivide(cube, linear, 0, q, r))
	a, e = q.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{2, 0, 0}: 4})
	b, e = r.Get()
	serviceMust(t, e)
	if b != m.zero {
		t.Fatal(b)
	}
}
func TestPolynomialPseudoRemainderSourceDebugIdentityAndOutputAlias(t *testing.T) {
	for _, exact := range []bool{false, true} {
		t.Run(map[bool]string{false: "minimal", true: "exact"}[exact], func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			divisor := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
			_, r := polynomialPseudoOutputs(t, m)
			serviceMust(t, r.Assign(p))
			d := uint32(91)
			if exact {
				serviceMust(t, m.ExactPseudoRemainder(p, divisor, 0, r))
			} else {
				serviceMust(t, m.PseudoRemainder(p, divisor, 0, &d, r))
			}
			a, e := r.Get()
			serviceMust(t, e)
			polynomialQueryExpect(t, m, a, map[[3]uint32]int64{{}: 5})
			if !exact && d != 2 {
				t.Fatal(d)
			}
		})
	}
}

func polynomialPseudoPhysical(t *testing.T, m *MultivariatePolynomialCore, q, r *PolynomialRef, polys []Polynomial, public ...*Int) {
	t.Helper()
	refs := []*PolynomialRef{r}
	if q != nil {
		refs = append(refs, q)
	}
	raw := []memory.Block{}
	if m.queryPositions.raw != (memory.Block{}) {
		raw = append(raw, m.queryPositions.raw)
	}
	if m.queryPositions.pending != (memory.Block{}) {
		raw = append(raw, m.queryPositions.pending)
	}
	polynomialNewtonPhysicalWithRaw(t, m, nil, refs, nil, nil, polys, nil, raw, public...)
	// Ref outputs are real named polynomial roots, not inferred directory entries.
	for _, ref := range refs {
		polys = append(polys, ref.root.value, ref.root.incoming)
	}
	polynomialOperationMonomialAudit(t, m, polys, nil)
}
func TestPolynomialPseudoDivisionPhysicalFailurePrefixes(t *testing.T) {
	for mode := uint8(0); mode < 4; mode++ {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			failures, success := 0, false
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				passed := false
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					m := polynomialCellFixture(t)
					p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
					divisor := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
					q, r := polynomialPseudoOutputs(t, m)
					serviceMust(t, q.Assign(m.unit))
					serviceMust(t, r.Assign(m.unit))
					d := uint32(91)
					m.integer.budget.FailAfter(ordinal)
					var e error
					switch mode {
					case 0:
						e = m.PseudoDivide(p, divisor, 0, &d, q, r)
					case 1:
						e = m.ExactPseudoDivide(p, divisor, 0, q, r)
					case 2:
						e = m.PseudoRemainder(p, divisor, 0, &d, r)
					case 3:
						e = m.ExactPseudoRemainder(p, divisor, 0, r)
					}
					polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor})
					polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{2, 0, 0}: 1, {}: 1})
					polynomialQueryExpect(t, m, divisor, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 1})
					if e != nil {
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal("actual allocation refusal", ordinal, e)
						}
						failures++
					} else {
						value, e := r.Get()
						serviceMust(t, e)
						polynomialQueryExpect(t, m, value, map[[3]uint32]int64{{}: 5})
						if mode < 2 {
							value, e = q.Get()
							serviceMust(t, e)
							polynomialQueryExpect(t, m, value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: -1})
						}
						passed = true
					}
					m.integer.budget.FailAfter(-1)
					if m.pseudoDivision.active {
						serviceMust(t, m.CleanupPseudoDivision())
					}
					if m.powerLocal.active {
						serviceMust(t, m.CleanupPolynomialPower())
					}
				})
				if passed {
					t.Log("first success", ordinal)
					success = true
					break
				}
			}
			if failures == 0 || !success {
				t.Fatal(failures, success)
			}
		})
	}
}
func TestPolynomialPseudoDivisionOutputIdentityAndSourceAssertions(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	divisor := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	q, r := polynomialPseudoOutputs(t, m)
	d := uint32(91)
	serviceMust(t, m.PseudoDivide(p, divisor, 0, &d, r, r))
	value, e := r.Get()
	serviceMust(t, e)
	if value != m.zero || d != 0 {
		t.Fatal("same Q/R: source R=A then Q=0", value, d)
	}
	polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor})
	if modularSourceDebug {
		visits := uint32(0)
		m.config.Checkpoint = func() bool { visits++; return true }
		e = m.PseudoDivide(Polynomial{}, Polynomial{}, ^uint32(0), &d, r, r)
		if !errors.Is(e, ErrPolynomialAssertion) || visits != 0 {
			t.Fatal("valid variable assertion before numeral reads", e, visits)
		}
	}
}

func TestPolynomialPseudoDivisionWarmSourceProfileAndCounts(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	divisor := polynomialQueryTerms(t, m, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	q, r := polynomialPseudoOutputs(t, m)
	visits := uint32(0)
	m.config.Checkpoint = func() bool { visits++; return true }
	operation := func() {
		d := uint32(0)
		if e := m.PseudoDivide(p, divisor, 0, &d, q, r); e != nil {
			panic(e)
		}
		if d != 2 {
			panic("iteration count")
		}
	}
	for i := 0; i < 20; i++ {
		operation()
	}
	visits = 0
	before := numericWarmProfileSnapshot(t, m.integer, m.integer.budget)
	monoBefore := m.monomials.allocator.Stats()
	allocations := testing.AllocsPerRun(20, operation)
	if visits != 63 {
		t.Fatal("source checkpoints including terminal loop", visits)
	}
	numericWarmProfileAudit(t, m.integer, m.integer.budget, before, allocations, monoBefore.DirectObjects != 0)
	if m.monomials.allocator.Stats() != monoBefore || m.pseudoDivision.active {
		t.Fatal("retained graph", monoBefore, m.monomials.allocator.Stats())
	}
	polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor})
}

// Explicit malformed requested-size metadata exercises the reached source
// destructor guard. This is neither source-valid exhaustion nor pointer repair.
func TestPolynomialPseudoDivisionReverseDestructionRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	config := serviceConfig(b).Integers
	config.Memory = runtime
	config.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	rational, e := NewRationalManager(RationalConfig{Integers: config})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: runtime, Profile: memory.SmallAllocatorProfile{PointerBits: 32}})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: rational.z, Rationals: rational, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	big := Int{}
	serviceMust(t, m.integer.SetLimbs(&big, []uint32{17, 0, 0, 1}))
	x2, e := mm.MkVariable(0, 2)
	serviceMust(t, e)
	x, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	unit, e := mm.Unit()
	serviceMust(t, e)
	p, e := m.MkTerms([]Int{big, NewInt(1)}, []Monomial{x2, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	divisor, e := m.MkTerms([]Int{NewInt(2), big}, []Monomial{x, unit})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(divisor))
	q, e := NewPolynomialRef(m)
	serviceMust(t, e)
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	var token numericCellRef
	var original numericCell
	defer func() {
		if token != 0 && validNumericRef(m.integer.cells, token) {
			cell := m.integer.cell(token)
			cell.backing.requested = original.backing.requested
			m.integer.store(token, cell)
		}
		serviceMust(t, m.CleanupPseudoDivision())
		serviceMust(t, r.Close())
		serviceMust(t, q.Close())
		serviceMust(t, m.DecRef(divisor))
		serviceMust(t, m.DecRef(p))
		serviceMust(t, m.integer.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, mm.DecManagerRef())
		serviceMust(t, rational.CloseChecked())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Fatal("complete owning refund", b.Used())
		}
	}()
	visits := uint32(0)
	var leading Polynomial
	var before uint32
	var numeral Int
	m.config.Checkpoint = func() bool {
		visits++
		if visits != 2 {
			return true
		}
		s := &m.pseudoDivision
		a, e := m.Coefficient(s.refs[pseudoReductB].value, 0)
		serviceMust(t, e)
		token = a.cell
		if token == 0 {
			t.Fatal("real BIG reduct local")
		}
		original = m.integer.cell(token)
		corrupt := original
		corrupt.backing.requested = uint64(math.MaxUint32) + 1
		m.integer.store(token, corrupt)
		leading = s.refs[pseudoLeadB].value
		before, e = m.readWord(leading, 0)
		serviceMust(t, e)
		numeral = s.numeral
		if numeral.cell == 0 || m.integer.isFailureRef(numeral.cell) {
			t.Fatal("actual retained BIG NUM absent", numeral)
		}
		return false
	}
	d := uint32(0)
	e = m.PseudoDivide(p, divisor, 0, &d, q, r)
	if !errors.Is(e, ErrUPolynomialCanceled) || !errors.Is(e, arena.ErrBounds) || visits != 2 {
		t.Fatal("original plus reached destructor", e, visits)
	}
	s := &m.pseudoDivision
	if !s.active || s.depth == 0 || s.stack[s.depth-1] != pseudoReductB || s.refs[pseudoLeadB].value != leading || s.numeral != numeral {
		t.Fatal("younger reduct stop retains older leading and NUM", s)
	}
	after, e := m.readWord(leading, 0)
	serviceMust(t, e)
	if before != after {
		t.Fatal("older source destructor unreached", before, after)
	}
	polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor}, &big)
	if !errors.Is(m.CloseChecked(), ErrPolynomialPending) {
		t.Fatal("explicit pending guard")
	}
	prior := s.refs[pseudoReductB].value
	if !errors.Is(m.PseudoDivide(p, divisor, 0, &d, q, r), ErrPolynomialPending) || s.refs[pseudoReductB].value != prior {
		t.Fatal("source reuse does not acknowledge stopped cleanup")
	}
	// Fixture metadata restoration is only performed after the immediate census.
	cell := m.integer.cell(token)
	cell.backing.requested = original.backing.requested
	m.integer.store(token, cell)
	serviceMust(t, m.CleanupPseudoDivision())
	polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor}, &big)
}

func TestPolynomialPseudoDivisionIndependentSignedTable(t *testing.T) {
	for _, a2 := range []int32{-3, 1, 2} {
		for _, a1 := range []int32{-2, 0, 3} {
			for _, b1 := range []int32{-2, 1, 3} {
				t.Run(fmt.Sprint(a2, "/", a1, "/", b1), func(t *testing.T) {
					m := polynomialCellFixture(t)
					const a0 int32 = 2
					const b0 int32 = -1
					coefficients := []int32{a2, a0}
					powers := [][]MonomialPower{{{0, 2}}, nil}
					if a1 != 0 {
						coefficients = append(coefficients, a1)
						powers = append(powers, []MonomialPower{{0, 1}})
					}
					p := polynomialQueryTerms(t, m, coefficients, powers)
					divisor := polynomialQueryTerms(t, m, []int32{b1, b0}, [][]MonomialPower{{{0, 1}}, nil})
					q, r := polynomialPseudoOutputs(t, m)
					serviceMust(t, q.Assign(divisor))
					serviceMust(t, r.Assign(p))
					for _, exact := range []bool{false, true} {
						d := uint32(91)
						if exact {
							serviceMust(t, m.ExactPseudoDivide(p, divisor, 0, q, r))
						} else {
							serviceMust(t, m.PseudoDivide(p, divisor, 0, &d, q, r))
						}
						slope := int64(b1)*int64(a1) - int64(b0)*int64(a2)
						qx := int64(b1) * int64(a2)
						qc := slope
						rem := int64(b1)*int64(b1)*int64(a0) - int64(b1)*int64(b0)*int64(a1) + int64(b0)*int64(b0)*int64(a2)
						wantD := uint32(2)
						if slope == 0 && !exact {
							wantD = 1
							qx = int64(a2)
							qc = 0
							rem = int64(b1) * int64(a0)
						}
						if !exact && d != wantD {
							t.Fatal("minimal source loop count", d, wantD)
						}
						wantQ := map[[3]uint32]int64{{1, 0, 0}: qx}
						if qc != 0 {
							wantQ[[3]uint32{}] = qc
						}
						wantR := map[[3]uint32]int64{}
						if rem != 0 {
							wantR[[3]uint32{}] = rem
						}
						a, e := q.Get()
						serviceMust(t, e)
						b, e := r.Get()
						serviceMust(t, e)
						polynomialQueryExpect(t, m, a, wantQ)
						polynomialQueryExpect(t, m, b, wantR)
						// Independent coefficient expansion, evaluated at signed sample points.
						factor := int64(b1)
						if exact || wantD == 2 {
							factor *= int64(b1)
						}
						for x := int64(-5); x <= 5; x++ {
							left := factor * (int64(a2)*x*x + int64(a1)*x + int64(a0))
							right := (qx*x+qc)*(int64(b1)*x+int64(b0)) + rem
							if left != right {
								t.Fatal("independent identity", x, left, right)
							}
						}
						polynomialPseudoPhysical(t, m, q, r, []Polynomial{p, divisor})
					}
				})
			}
		}
	}
}
