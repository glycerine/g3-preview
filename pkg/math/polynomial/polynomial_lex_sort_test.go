// Source: src/math/polynomial/polynomial.cpp:1315-1366,1440-1460,2462-2464,7022-7024;
// src/util/permutation.h:57-91.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func polynomialLexFixture(t *testing.T) (*MultivariatePolynomialManager, Polynomial, []Monomial) {
	t.Helper()
	q, b := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("source lex-sort has no checkpoint") }, Cooperation: func() error { panic("source lex-sort has no cooperation") }}})
	serviceMust(t, e)
	for i := 0; i < 2; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	powers := [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 2}}, {{1, 3}}, {{0, 1}, {1, 1}}, nil}
	ms := make([]Monomial, len(powers))
	for i := range powers {
		ms[i], e = m.core.monomials.MkMonomial(powers[i])
		serviceMust(t, e)
	}
	p, e := m.core.MkTerms([]util.Int{util.NewInt(11), util.NewInt(13), util.NewInt(17), util.NewInt(19), util.NewInt(23)}, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() {
		serviceMust(t, m.core.DecRef(p))
		serviceMust(t, m.CloseChecked())
		if b.Used() == 0 {
			t.Fatal("borrowed numeral owner unexpectedly disappeared")
		}
	})
	return m, p, ms
}

// Frozen independent order uses the highest variable first, descending degrees:
// y^3, x*y^2, x*y, x^2, 1. The coefficients stay attached to their monomials.
func TestPolynomialLexSortIndependentRecursiveOrder(t *testing.T) {
	m, p, ms := polynomialLexFixture(t)
	serviceMust(t, m.LexSort(p))
	info, e := m.core.Info(p)
	serviceMust(t, e)
	if !info.LexSorted || info.Size != 5 {
		t.Fatal("final source lex flag/size", info)
	}
	for i, index := range []int{2, 0, 3, 1, 4} {
		term, e := m.core.monomialCore(p, uint32(i))
		serviceMust(t, e)
		a, e := m.core.coefficientCore(p, uint32(i))
		serviceMust(t, e)
		if term != ms[index] || (a).Cell != 0 || a.NativeScalar() != []int32{11, 13, 17, 19, 23}[index] {
			t.Fatal("independent term ordering/coefficient identity", i, term, a)
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	before := m.core.memory.Stats()
	bytes := m.core.integer.Budget().Used()
	serviceMust(t, m.LexSort(p))
	if m.core.memory.Stats() != before || m.core.integer.Budget().Used() != bytes {
		t.Fatal("already lex-sorted source branch allocated")
	}
}

// The source permutation swaps mpz representation fields; it does not call
// numeral_manager.swap or query normalization/digits. An actual failed fresh
// Copy records a defined source-null pointer, although public Go reads reject it.
func TestPolynomialLexSortFreshNullCoefficientRawPermutation(t *testing.T) {
	m, p, ms := polynomialLexFixture(t)
	primer, e := m.core.MkTerms([]util.Int{util.NewInt(11), util.NewInt(13), util.NewInt(17), util.NewInt(19), util.NewInt(23)}, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(primer))
	serviceMust(t, m.LexSort(primer))
	serviceMust(t, m.core.DecRef(primer))
	a, e := m.core.coefficientCore(p, 1)
	serviceMust(t, e)
	target, e := m.core.monomialCore(p, 1)
	serviceMust(t, e)
	b := m.core.integer.Budget()
	b.FailAfter(0)
	e = m.core.integer.Copy(&a, &m.core.integer.Two64)
	if !numericPhysicalReservationRefusal(e) || (a).Cell != m.core.integer.FailedNullRef {
		t.Fatal("actual fresh source-null setter refusal", e, a)
	}
	serviceMust(t, m.core.writeCoefficient(p, 1, a))
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	b.FailAfter(-1)
	serviceMust(t, m.LexSort(p))
	found := false
	for i := uint32(0); i < 5; i++ {
		term, e := m.core.monomialCore(p, i)
		serviceMust(t, e)
		if term == target {
			actual, e := m.core.coefficientCore(p, i)
			serviceMust(t, e)
			if actual != a {
				t.Fatal("source raw NUM swap was replaced by guarded numeral operation", actual, a)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("actual source term disappeared")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func polynomialLexNestedFixture(t *testing.T) (*MultivariatePolynomialManager, Polynomial, []Monomial, []int32) {
	q, _ := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("source lex-sort never checkpoints") }, Cooperation: func() error { panic("source lex-sort never cooperates") }}})
	serviceMust(t, e)
	for i := 0; i < 3; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	powers := [][]MonomialPower{{{0, 17}, {2, 1}}, {{0, 3}, {2, 1}}, {{1, 2}}, {{0, 2}, {1, 2}}, {{2, 2}}, nil}
	ms := make([]Monomial, len(powers))
	values := []int32{2, 3, 5, 7, 11, 13}
	as := make([]util.Int, len(values))
	for i := range ms {
		ms[i], e = m.core.monomials.MkMonomial(powers[i])
		serviceMust(t, e)
		as[i] = util.NewInt(values[i])
	}
	p, e := m.core.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.core.DecRef(p)); serviceMust(t, m.CloseChecked()) })
	return m, p, ms, values
}

func TestPolynomialLexSortNestedGrowthIndependentFiniteModel(t *testing.T) {
	m, p, ms, values := polynomialLexNestedFixture(t)
	serviceMust(t, m.LexSort(p))
	for i, index := range []int{4, 0, 1, 3, 2, 5} {
		term, e := m.core.monomialCore(p, uint32(i))
		serviceMust(t, e)
		a, e := m.core.coefficientCore(p, uint32(i))
		serviceMust(t, e)
		if term != ms[index] || a != util.NewInt(values[index]) {
			t.Fatal("nested independent order", i, term, a)
		}
	}
	// Independent evaluation at x=2,y=3,z=5: 2*2^17*5 + 3*2^3*5
	// + 5*3^2 + 7*2^2*3^2 + 11*5^2 + 13 = 1311425.
	sum := int64(0)
	for i := uint32(0); i < 6; i++ {
		term, e := m.core.monomialCore(p, i)
		serviceMust(t, e)
		a, e := m.core.coefficientCore(p, i)
		serviceMust(t, e)
		h, e := m.core.monomials.Info(term)
		serviceMust(t, e)
		v := int64(a.NativeScalar())
		for j := uint32(0); j < h.Size; j++ {
			power, e := m.core.monomials.ReadPower(term, j)
			serviceMust(t, e)
			for k := uint32(0); k < power.Degree; k++ {
				v *= []int64{2, 3, 5}[power.Var]
			}
		}
		sum += v
	}
	if sum != 1311425 {
		t.Fatal("independent finite evaluation", sum)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

// Source d+1 is unsigned32. UINT_MAX therefore does not reserve an enormous
// bucket array: the subsequent subscript reaches the profile assertion or
// actual physical bounds guard. Native out-of-array access is a guarded domain
// boundary, not a defined C++ exception or ordinary allocation exhaustion.
func TestPolynomialLexSortWrappedDegreeReachedBucket(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("lex has no checkpoint") }, Cooperation: func() error { panic("lex has no cooperation") }}})
	serviceMust(t, e)
	for i := 0; i < 2; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	x, e := m.core.monomials.MkMonomial([]MonomialPower{{0, ^uint32(0)}})
	serviceMust(t, e)
	y, e := m.core.monomials.MkMonomial([]MonomialPower{{1, 1}})
	serviceMust(t, e)
	p, e := m.core.MkTerms([]util.Int{util.NewInt(7), util.NewInt(11)}, []Monomial{x, y})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	defer func() { serviceMust(t, m.core.DecRef(p)); serviceMust(t, m.CloseChecked()) }()
	e = m.LexSort(p)
	if polynomialSourceDebug {
		if e != ErrPolynomialAssertion {
			t.Fatal("source debug bucket subscript", e)
		}
	} else if e != arena.ErrBounds {
		t.Fatal("actual physical bucket bound", e)
	}
	if !m.lexPending || m.lexCause != e {
		t.Fatal("reached source stop not retained", e)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	// Top-level y grouping has completed, with the actual lower-x tail retained.
	first, read := m.core.monomialCore(p, 0)
	serviceMust(t, read)
	if first != y {
		t.Fatal("completed higher-variable permutation", first)
	}
	serviceMust(t, m.CleanupLexSort())
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func TestPolynomialLexSortActualBigCoefficientOwnership(t *testing.T) {
	m, _, ms := polynomialLexFixture(t)
	as := make([]util.Int, 5)
	for i := range as {
		serviceMust(t, m.core.integer.SetUint64(&as[i], uint64(1)<<40+uint64(i)))
	}
	p, e := m.core.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	for i := range as {
		serviceMust(t, m.core.integer.Del(&as[i]))
	}
	defer func() { serviceMust(t, m.core.DecRef(p)) }()
	refs := make([]util.Int, 5)
	for i := uint32(0); i < 5; i++ {
		term, e := m.core.monomialCore(p, i)
		serviceMust(t, e)
		a, e := m.core.coefficientCore(p, i)
		serviceMust(t, e)
		for j, known := range ms {
			if term == known {
				refs[j] = a
			}
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	serviceMust(t, m.LexSort(p))
	for i, index := range []int{2, 0, 3, 1, 4} {
		a, e := m.core.coefficientCore(p, uint32(i))
		serviceMust(t, e)
		if a != refs[index] || (a).Cell == 0 {
			t.Fatal("actual owning numeral token not moved exactly", i, a, refs[index])
		}
		v, e := m.core.integer.Uint64(&a)
		serviceMust(t, e)
		if v != (uint64(1)<<40)+uint64(index) {
			t.Fatal("big coefficient independent value", i, v)
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func TestPolynomialLexSortNativeScratchGrowthAndReuse(t *testing.T) {
	m, original, ms, values := polynomialLexNestedFixture(t)
	defer m.core.integer.Budget().FailAfter(-1)
	for pass := 0; pass < 2; pass++ {
		p := original
		if pass != 0 {
			as := make([]util.Int, len(values))
			for i, v := range values {
				as[i] = util.NewInt(v)
			}
			var e error
			p, e = m.core.MkTerms(as, ms)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(p))
			t.Cleanup(func() { serviceMust(t, m.core.DecRef(p)) })
		}
		stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
		m.core.integer.Budget().FailAfter(0)
		serviceMust(t, m.LexSort(p))
		m.core.integer.Budget().FailAfter(-1)
		if m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used {
			t.Fatal("native lex scratch charged Runtime")
		}
		for i, index := range []int{4, 0, 1, 3, 2, 5} {
			term, e := m.core.monomialCore(p, uint32(i))
			serviceMust(t, e)
			a, e := m.core.coefficientCore(p, uint32(i))
			serviceMust(t, e)
			if term != ms[index] || a != util.NewInt(values[index]) {
				t.Fatal("native lex growth/reuse changed term pair", pass, i)
			}
		}
		for _, b := range m.lexBuckets {
			if len(b) != 0 {
				t.Fatal("recursive sort retained bucket terms")
			}
		}
		polynomialImpPhysicalAuditValues(t, m, []Polynomial{original, p})
	}
}

// This invalid unsigned-degree fixture reaches the native bucket bounds guard
// after completing the higher-variable permutation. No giant allocation occurs.
func polynomialLexWrappedPolynomial(t *testing.T, m *MultivariatePolynomialManager) Polynomial {
	t.Helper()
	x, e := m.core.monomials.MkMonomial([]MonomialPower{{0, ^uint32(0)}})
	serviceMust(t, e)
	y, e := m.core.monomials.MkMonomial([]MonomialPower{{1, 1}})
	serviceMust(t, e)
	p, e := m.core.MkTerms([]util.Int{util.NewInt(7), util.NewInt(11)}, []Monomial{x, y})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.core.DecRef(p)) })
	return p
}

func TestPolynomialLexSortStoppedScratchOnlyGuardsReachedReuse(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	invalid := polynomialLexWrappedPolynomial(t, m)
	e := m.LexSort(invalid)
	expected := error(arena.ErrBounds)
	if polynomialSourceDebug {
		expected = ErrPolynomialAssertion
	}
	if e != expected || !m.lexPending {
		t.Fatal("fixture did not reach native bucket guard", e)
	}
	stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
	if again := m.LexSort(p); again != ErrPolynomialPending {
		t.Fatal("stopped scratch reuse", again)
	}
	serviceMust(t, m.LexSort(m.core.zero))
	serviceMust(t, m.LexSort(m.core.unit))
	if m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used || !m.lexPending {
		t.Fatal("early branches changed stopped scratch")
	}
	serviceMust(t, m.CleanupLexSort())
	if m.lexBuckets != nil || m.lexPermutation != nil {
		t.Fatal("owning cleanup retained native scratch")
	}
	serviceMust(t, m.LexSort(p))
}

func TestPolynomialLexSortNativeLargeBucketsIndependentGrid(t *testing.T) {
	m, _, _, _ := polynomialLexNestedFixture(t)
	ms := make([]Monomial, 0, 405)
	as := make([]util.Int, 0, 405)
	for z := uint32(0); z < 3; z++ {
		for y := uint32(0); y < 5; y++ {
			for x := uint32(0); x < 27; x++ {
				var powers []MonomialPower
				for v, degree := range [3]uint32{x, y, z} {
					if degree != 0 {
						powers = append(powers, MonomialPower{uint32(v), degree})
					}
				}
				term, e := m.core.monomials.MkMonomial(powers)
				serviceMust(t, e)
				ms = append(ms, term)
				as = append(as, util.NewInt(int32(len(ms))))
			}
		}
	}
	defer m.core.integer.Budget().FailAfter(-1)
	for pass := 0; pass < 2; pass++ {
		p, e := m.core.MkTerms(as, ms)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(p))
		t.Cleanup(func() { serviceMust(t, m.core.DecRef(p)) })
		stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
		m.core.integer.Budget().FailAfter(0)
		serviceMust(t, m.LexSort(p))
		m.core.integer.Budget().FailAfter(-1)
		if m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used {
			t.Fatal("native large lex scratch charged Runtime")
		}
		// Construction enumerates ascending z, then y, then x; lex order reverses it.
		for i := range ms {
			index := len(ms) - 1 - i
			term, e := m.core.monomialCore(p, uint32(i))
			serviceMust(t, e)
			a, e := m.core.coefficientCore(p, uint32(i))
			serviceMust(t, e)
			if term != ms[index] || a != as[index] {
				t.Fatal("independent grid lex order/pair mismatch", pass, i)
			}
		}
		for _, bucket := range m.lexBuckets {
			if len(bucket) != 0 {
				t.Fatal("native bucket not reset")
			}
		}
		polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	}
}
