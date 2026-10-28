package polynomial

import "testing"

// Use the real stopped lexical-sort path: its wrapped-degree input reaches a
// bounds diagnostic after publishing scratch, without allocating giant buckets.
func TestPolynomialCacheUniqueCleanupAcceptedInvocation(t *testing.T) {
	m, valid, _ := polynomialLexFixture(t)
	invalid := polynomialLexWrappedPolynomial(t, m)
	c, err := NewPolynomialCache(m)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	_, err = c.MkUnique(invalid)
	if err == nil || !m.matchesLexSortInvocation(&c.hashLex) || !c.hasPendingWork() {
		t.Fatal("canonicalization did not retain its accepted lexical sort", err)
	}
	serviceMust(t, c.CleanupUnique())
	if m.lexPending || c.hasPendingWork() || m.lexBuckets != nil || m.lexPermutation != nil {
		t.Fatal("accepted lexical scratch survived cleanup")
	}
	p, err := c.MkUnique(valid)
	serviceMust(t, err)
	if p != valid {
		t.Fatal("canonicalization failed after cleanup")
	}
	serviceMust(t, c.CleanupUnique())
	again, err := c.MkUnique(valid)
	serviceMust(t, err)
	if again != p {
		t.Fatal("cleanup discarded a published representative")
	}
}

func TestPolynomialCacheUniqueCleanupPreservesOlderLexicalSort(t *testing.T) {
	m, valid, _ := polynomialLexFixture(t)
	invalid := polynomialLexWrappedPolynomial(t, m)
	c, err := NewPolynomialCache(m)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	cause := m.LexSort(invalid)
	if cause == nil || !m.lexPending {
		t.Fatal("older call did not stop with scratch", cause)
	}
	stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
	generation, receipt := m.lexGeneration, c.hashLex
	_, err = c.MkUnique(valid)
	if err != ErrPolynomialPending {
		t.Fatal("younger canonicalization accepted older stopped scratch", err)
	}
	serviceMust(t, c.CleanupUnique())
	if !m.lexPending || m.lexCause != cause || m.lexGeneration != generation || c.hashLex != receipt || m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used {
		t.Fatal("younger cleanup changed the older invocation")
	}
	serviceMust(t, m.CleanupLexSort())
	_, err = c.MkUnique(valid)
	serviceMust(t, err)
}
