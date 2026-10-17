package polynomial

import (
	"errors"
	"testing"
)

func TestPolynomialCompositionHornerChildCancellationCleanupAndRetry(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{-2, 1}, [][]MonomialPower{nil, {{0, 2}}})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, full.CleanupComposition()); serviceMust(t, r.Close()) }()
	checkpoint := c.config.Checkpoint
	visits := 0
	c.config.Checkpoint = func() bool { visits++; return visits < 2 }
	body := full.ComposeXMinusY(p, 1, r)
	c.config.Checkpoint = checkpoint
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("Horner multiply/add did not stop at its second checkpoint", body)
	}
	serviceMust(t, full.CleanupComposition())
	serviceMust(t, full.ComposeXMinusY(p, 1, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{
		{2, 0, 0}: 1, {1, 1, 0}: -2, {0, 2, 0}: 1, {0, 0, 0}: -2,
	})
}
