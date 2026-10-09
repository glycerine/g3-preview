// Source: src/math/polynomial/polynomial.cpp:4745-4819.
package polynomial

import "testing"

func TestPolynomialCompositionNativeDegreeCacheNoRuntimeGrowth(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{257, 0, 0}}, {-2, [3]uint32{2, 0, 0}}, {5, [3]uint32{}}})
	stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
	m.core.integer.Budget().FailAfter(0)
	t.Cleanup(func() { m.core.integer.Budget().FailAfter(-1) })
	for pass := 0; pass < 2; pass++ {
		serviceMust(t, m.saveCompositionDegrees(p))
		expected := map[uint32]int64{257: 3, 2: -2, 0: 5}
		for degree := uint32(0); degree <= 257; degree++ {
			pos, e := m.compositionDegreePosition(degree)
			serviceMust(t, e)
			want, present := expected[degree]
			if !present {
				if pos != ^uint32(0) {
					t.Fatal("missing degree has a term position", degree, pos)
				}
				continue
			}
			a, e := m.core.coefficientCore(p, pos)
			serviceMust(t, e)
			got, e := m.core.integer.Int64(&a)
			serviceMust(t, e)
			if got != want {
				t.Fatal("degree points to wrong coefficient", degree, got, want)
			}
		}
		serviceMust(t, m.resetCompositionDegrees(p))
		for _, pos := range m.degree2pos {
			if pos != ^uint32(0) {
				t.Fatal("reset retained a degree position", pos)
			}
		}
	}
	if m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used {
		t.Fatal("native degree cache charged custom Runtime storage")
	}
}

func TestPolynomialCompositionNativeDegreeCacheSparseHornerReuse(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{}}})
	q := polynomialCompositionTerms(t, m, []polynomialEvalTerm{{1, [3]uint32{0, 2, 0}}})
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupComposition()); serviceMust(t, r.Close()) })
	// Reuse one manager with increasing and decreasing sparse degrees. Missing
	// coefficients must remain zero in Horner's loop, including on warmed runs.
	for _, degree := range []uint32{17, 65, 257, 33, 17} {
		p := polynomialCompositionTerms(t, m, []polynomialEvalTerm{{3, [3]uint32{degree, 0, 0}}, {-2, [3]uint32{2, 0, 0}}, {5, [3]uint32{}}})
		serviceMust(t, m.Compose(p, q, r))
		// p(y^2) = 3*y^(2*degree) - 2*y^4 + 5.
		want := polynomialCompositionTerms(t, m, []polynomialEvalTerm{{3, [3]uint32{0, 2 * degree, 0}}, {-2, [3]uint32{0, 4, 0}}, {5, [3]uint32{}}})
		equal, e := m.core.Equal(r.root.value, want)
		serviceMust(t, e)
		if !equal {
			t.Fatal("sparse Horner composition mismatch", degree)
		}
		for _, pos := range m.degree2pos {
			if pos != ^uint32(0) {
				t.Fatal("completed composition retained degree position", degree, pos)
			}
		}
	}
}
