// Source: src/math/polynomial/polynomial.cpp:1697-1818,3264-3338,3551-3553,4600-4655.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func polynomialQueryTerms(t *testing.T, m *MultivariatePolynomialCore, as []int32, pws [][]MonomialPower) Polynomial {
	t.Helper()
	aa := make([]Int, len(as))
	ms := make([]Monomial, len(as))
	for i, a := range as {
		aa[i] = NewInt(a)
		var e error
		ms[i], e = m.monomials.MkMonomial(pws[i])
		serviceMust(t, e)
	}
	p, e := m.MkTerms(aa, ms)
	serviceMust(t, e)
	return polynomialCaller(t, m, p)
}
func polynomialQueryExpect(t *testing.T, m *MultivariatePolynomialCore, p Polynomial, want map[[3]uint32]int64) {
	t.Helper()
	n, e := m.readWord(p, 8)
	serviceMust(t, e)
	if n != uint32(len(want)) {
		t.Fatal("size", n, len(want))
	}
	seen := map[[3]uint32]bool{}
	for i := uint32(0); i < n; i++ {
		term, e := m.TermMonomial(p, i)
		serviceMust(t, e)
		var key [3]uint32
		for x := uint32(0); x < 3; x++ {
			key[x], e = m.monomials.DegreeOf(term, x)
			serviceMust(t, e)
		}
		a, e := m.Coefficient(p, i)
		serviceMust(t, e)
		v, e := m.integer.Int64(&a)
		serviceMust(t, e)
		w, ok := want[key]
		if !ok || seen[key] || v != w {
			t.Fatal(key, v, w, ok)
		}
		seen[key] = true
	}
}
func TestPolynomialQueryCoefficientIndependentMaps(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, -5, 7, 11}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 1}, {2, 1}}, nil, {{0, 3}, {1, 2}}})
	for x, w := range []uint32{3, 2, 1} {
		d, e := m.Degree(p, uint32(x))
		serviceMust(t, e)
		if d != w {
			t.Fatal(x, d, w)
		}
	}
	x, e := m.MaxVariable(p)
	serviceMust(t, e)
	if x != 2 {
		t.Fatal(x)
	}
	zero, e := m.IsZero(m.zero)
	serviceMust(t, e)
	constant, e := m.IsConstant(m.zero)
	serviceMust(t, e)
	if !zero || !constant {
		t.Fatal(zero, constant)
	}
	reduct, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, reduct.Close()) }()
	c, e := m.CoeffWithReduct(p, 1, 2, reduct)
	serviceMust(t, e)
	polynomialCaller(t, m, c)
	polynomialQueryExpect(t, m, c, map[[3]uint32]int64{{1, 0, 0}: 3, {3, 0, 0}: 11})
	r, e := reduct.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, r, map[[3]uint32]int64{{0, 1, 1}: -5, {}: 7})
	lc, e := m.LeadingCoefficient(p, 1)
	serviceMust(t, e)
	polynomialCaller(t, m, lc)
	same, e := m.Equal(c, lc)
	serviceMust(t, e)
	if !same {
		t.Fatal("LC")
	}
	out := NewInt(91)
	ok, e := m.ConstantCoefficient(p, 1, 0, &out)
	serviceMust(t, e)
	v, e := m.integer.Int64(&out)
	serviceMust(t, e)
	if !ok || v != 7 {
		t.Fatal(ok, v)
	}
	ok, e = m.ConstantCoefficient(p, 1, 2, &out)
	serviceMust(t, e)
	v, e = m.integer.Int64(&out)
	serviceMust(t, e)
	if ok || v != 0 {
		t.Fatal("reset-before-refusal", ok, v)
	}
	serviceMust(t, m.modular.Del(&out))
}
func TestPolynomialQueryPowerLinearIndependentOracle(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	// Independently fixed binomial coefficient rows, not produced by powering.
	wants := [][]int64{{1}, {1, 1}, {1, 2, 1}, {1, 3, 3, 1}, {1, 4, 6, 4, 1}, {1, 5, 10, 10, 5, 1}}
	for k, row := range wants {
		serviceMust(t, m.Power(p, uint32(k), out))
		r, e := out.Get()
		serviceMust(t, e)
		expected := map[[3]uint32]int64{}
		for i, a := range row {
			expected[[3]uint32{uint32(i), 0, 0}] = a
		}
		polynomialQueryExpect(t, m, r, expected)
	}
	// Actual input/output identity: r's old value is read by parameter value.
	serviceMust(t, out.Assign(p))
	input, e := out.Get()
	serviceMust(t, e)
	serviceMust(t, m.Power(input, 3, out))
	r, e := out.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, r, map[[3]uint32]int64{{}: 1, {1, 0, 0}: 3, {2, 0, 0}: 3, {3, 0, 0}: 1})
	// Source k0 does not dereference a raw-null input or construct result local.
	serviceMust(t, m.Power(Polynomial{}, 0, out))
	r, e = out.Get()
	serviceMust(t, e)
	if r != m.unit {
		t.Fatal(r)
	}
}
func TestPolynomialQueryEqualSourceOrderAndPermutation(t *testing.T) {
	m := polynomialCellFixture(t)
	a := polynomialQueryTerms(t, m, []int32{2, 3, 5}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	b := polynomialQueryTerms(t, m, []int32{5, 2, 3}, [][]MonomialPower{nil, {{0, 1}}, {{1, 1}}})
	ok, e := m.Equal(a, b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("permutation")
	}
	c := polynomialQueryTerms(t, m, []int32{2, 3, 7}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	ok, e = m.Equal(a, c)
	serviceMust(t, e)
	if ok {
		t.Fatal("coefficient inequality")
	}
	ok, e = m.Equal(Polynomial{}, Polynomial{})
	serviceMust(t, e)
	if !ok {
		t.Fatal("pointer short circuit")
	}
	ok, e = m.Equal(m.zero, m.unit)
	serviceMust(t, e)
	if ok {
		t.Fatal("size short circuit")
	}
	// No allocation/event is hidden after all source mapping capacity is warm.
	m.integer.budget.FailAfter(0)
	allocs := testing.AllocsPerRun(20, func() {
		same, e := m.Equal(a, b)
		if e != nil || !same {
			panic(e)
		}
	})
	m.integer.budget.FailAfter(-1)
	if allocs != 0 {
		t.Fatal("warm Eq", allocs)
	}
}

// Real borrowed event handler executes an actual allocation, whose failure
// interrupts INC-new/DEC-old before result publication. It fabricates no error.
func TestPolynomialQueryPowerAssignmentStoppedCallback(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	serviceMust(t, r.Assign(m.unit))
	h, e := NewPolynomialDeletionHandler(m.integer.budget, func(h *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
		h.State++
		if h.State != 1 {
			return nil
		}
		m.integer.budget.FailAfter(0)
		raw, e := m.memory.Allocate(1)
		if raw != (memory.Block{}) {
			t.Fatal("unexpected completed callback allocation", raw)
		}
		return e
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(h))
	defer func() {
		m.integer.budget.FailAfter(-1)
		serviceMust(t, m.CleanupPolynomialPower())
		serviceMust(t, m.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
	}()
	e = m.Power(p, 3, r)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("real callback allocation", e)
	}
	local := &m.powerLocal
	if !local.active || !local.result.decremented || !local.result.assigning || local.result.value == (Polynomial{}) || local.result.incoming == (Polynomial{}) || h.State != 1 {
		t.Fatal("stopped assignment roots", local, h.State)
	}
	got, e := r.Get()
	serviceMust(t, e)
	if got != m.unit {
		t.Fatal("caller published before last source assignment", got)
	}
	polynomialQueryExpect(t, m, local.result.value, map[[3]uint32]int64{{}: 1, {1, 0, 0}: 2, {2, 0, 0}: 1})
	polynomialQueryExpect(t, m, local.result.incoming, map[[3]uint32]int64{{}: 1, {1, 0, 0}: 3, {2, 0, 0}: 3, {3, 0, 0}: 1})
	polynomialNewtonPhysicalWithRaw(t, m, nil, []*PolynomialRef{r}, nil, nil, []Polynomial{p, local.result.value, local.result.incoming}, []*PolynomialDeletionHandler{h}, nil)
	polynomialOperationMonomialAudit(t, m, []Polynomial{p, local.result.value, local.result.incoming}, nil)
	if e = m.CloseChecked(); e != ErrPolynomialPending {
		t.Fatal("actual pending lexical owner guard", e)
	}
	m.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupPolynomialPower())
	if h.State != 2 || m.powerLocal.active {
		t.Fatal("completed callback replay or unjoined INC lost", h.State, m.powerLocal)
	}
}

func TestPolynomialQueryUnivariateLinearAndScalarCoefficient(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 7}, [][]MonomialPower{{{1, 2}}, nil})
	u, e := m.IsUnivariate(p)
	serviceMust(t, e)
	if !u {
		t.Fatal("univariate")
	}
	linear, e := m.IsLinear(p)
	serviceMust(t, e)
	if linear {
		t.Fatal("nonlinear")
	}
	total, e := m.TotalDegree(p)
	serviceMust(t, e)
	if total != 2 {
		t.Fatal(total)
	}
	c, e := m.UnivariateCoefficient(p, 2)
	serviceMust(t, e)
	v, e := m.integer.Int64(&c)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal(v)
	}
	c, e = m.UnivariateCoefficient(p, 1)
	serviceMust(t, e)
	if c != (Int{}) {
		t.Fatal("actual inline default absent coefficient", c)
	}
	p2 := polynomialQueryTerms(t, m, []int32{2, 3, 7}, [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil})
	u, e = m.IsUnivariate(p2)
	serviceMust(t, e)
	if u {
		t.Fatal("two variables")
	}
	linear, e = m.IsLinear(p2)
	serviceMust(t, e)
	if !linear {
		t.Fatal("linear")
	}
	total, e = m.TotalDegree(m.zero)
	serviceMust(t, e)
	if total != 0 {
		t.Fatal(total)
	}
}

// The temporary encoded monomial has real storage but only source-reached
// fields initialized. Restore the actual polynomial before any destruction;
// this probes guarded malformed-layout reads, not a constructed source value.
func TestPolynomialQueryReachedFieldReadProfiles(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{9}, [][]MonomialPower{{{1, 2}}})
	original, e := m.monomialCore(p, 0)
	serviceMust(t, e)
	partial := orderPartialValue(t, m.monomials, 1, nil)
	serviceMust(t, m.writeMonomial(p, 0, partial))
	defer func() { serviceMust(t, m.writeMonomial(p, 0, original)) }()
	univariate, e := m.IsUnivariate(p)
	serviceMust(t, e)
	if !univariate {
		t.Fatal("univariate reads VAR but not DEG or cached total")
	}
	_, e = m.IsLinear(p)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("linear reaches degree", e)
	}
	_, e = m.TotalDegree(p)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("total reaches cached total", e)
	}
	_, e = m.Degree(p, 1)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("maximal variable reaches degree", e)
	}
	serviceMust(t, m.writeMonomial(p, 0, original))
	p2 := polynomialQueryTerms(t, m, []int32{4, 3}, [][]MonomialPower{{{1, 2}}, {{0, 1}}})
	second, e := m.monomialCore(p2, 1)
	serviceMust(t, e)
	serviceMust(t, m.writeMonomial(p2, 1, Monomial{}))
	linear, e := m.IsLinear(p2)
	serviceMust(t, m.writeMonomial(p2, 1, second))
	serviceMust(t, e)
	if linear {
		t.Fatal("nonlinear first term must stop before dead second term")
	}
	got, e := m.UnivariateCoefficient(p2, 2)
	if modularSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("debug univariate assertion", e)
		}
	} else {
		serviceMust(t, e)
		n, e := m.integer.Int64(&got)
		serviceMust(t, e)
		if n != 4 {
			t.Fatal("release reads first matching total degree", n)
		}
	}
}

func TestPolynomialQueryCoefficientPhysicalFailurePrefixes(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		reached := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{3, -5, 7, 11}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 1}, {2, 1}}, nil, {{0, 3}, {1, 2}}})
			reduct, e := NewPolynomialRef(m)
			serviceMust(t, e)
			serviceMust(t, reduct.Assign(m.unit))
			defer func() { serviceMust(t, reduct.Close()) }()
			b := m.integer.budget
			b.FailAfter(ordinal)
			result, e := m.CoeffWithReduct(p, 1, 2, reduct)
			named := []Polynomial{p}
			if result != (Polynomial{}) {
				named = append(named, result)
			}
			// These are actual caller and retained source buffer roots before cleanup
			// or fault changes; neither failed output nor intern history is a root.
			polynomialNewtonPhysicalAudit(t, m, nil, []*PolynomialRef{reduct}, nil, nil, named, nil)
			polynomialOperationMonomialAudit(t, m, named, nil)
			polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{1, 2, 0}: 3, {0, 1, 1}: -5, {}: 7, {3, 2, 0}: 11})
			// Source publishes the complete reduct only after cheap2.mk; a later
			// cheap.mk refusal can therefore leave that complete earlier action.
			rawReduct, re := reduct.Get()
			serviceMust(t, re)
			rn, re := m.readWord(rawReduct, 8)
			serviceMust(t, re)
			if rn == 1 {
				polynomialQueryExpect(t, m, rawReduct, map[[3]uint32]int64{{}: 1})
			} else {
				polynomialQueryExpect(t, m, rawReduct, map[[3]uint32]int64{{0, 1, 1}: -5, {}: 7})
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual physical category", ordinal, e)
				}
				failures++
			} else {
				reached = true
			}
			b.FailAfter(-1)
			if reached {
				polynomialCaller(t, m, result)
				polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 3, {3, 0, 0}: 11})
				r, e := reduct.Get()
				serviceMust(t, e)
				polynomialQueryExpect(t, m, r, map[[3]uint32]int64{{0, 1, 1}: -5, {}: 7})
			}
		})
		if reached {
			success = true
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal("exhaustive finite physical prefixes", failures, success)
	}
}

func TestPolynomialQueryEqualScratchPhysicalFailures(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		reached := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{3, 7}, [][]MonomialPower{{{1, 2}}, nil})
			q := polynomialQueryTerms(t, m, []int32{5, 7}, [][]MonomialPower{{{1, 2}}, nil})
			b := m.integer.budget
			b.FailAfter(ordinal)
			equal, e := m.Equal(p, q)
			raw := []memory.Block{}
			if m.queryPositions.raw != (memory.Block{}) {
				raw = append(raw, m.queryPositions.raw)
			}
			if m.queryPositions.pending != (memory.Block{}) {
				raw = append(raw, m.queryPositions.pending)
			}
			polynomialNewtonPhysicalWithRaw(t, m, nil, nil, nil, nil, []Polynomial{p, q}, nil, raw)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p, q}, nil)
			polynomialQueryExpect(t, m, p, map[[3]uint32]int64{{0, 2, 0}: 3, {}: 7})
			polynomialQueryExpect(t, m, q, map[[3]uint32]int64{{0, 2, 0}: 5, {}: 7})
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("scratch allocation category", e)
				}
				failures++
			} else {
				if equal {
					t.Fatal("independent unequal coefficients")
				}
				reached = true
			}
			b.FailAfter(-1)
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
