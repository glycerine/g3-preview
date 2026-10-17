package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"math"
	"testing"
)

func TestAlgebraicDefiningNegationPreservesOlderNumericAllocation(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	qm, z := c.config.Rationals, c.config.Rationals.Integers()
	q := unumeral.NewRational()
	serviceMust(t, qm.SetInt32(&q, math.MinInt32))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	pad := algebraicCellSourceTLSBoundary(t, c)
	runtime := z.RuntimeOwner()
	var older unumeral.IntegerCopyAllocationReceipt
	destination := unumeral.NewInt(0)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &older)
	if body == nil || !z.MatchesCopyAllocationInvocation(&older) {
		t.Fatal("actual older pending integer allocation missing", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	body = c.GetPolynomial(&a, out)
	if body == nil || c.representationNeg.owner != c || unumeral.NumericAllocationPending(&c.representationNeg.roots) {
		t.Fatal("refused negation acquired older allocation", body)
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&older) || state != z.CopyAllocation || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("getter cleanup consumed older numeric allocation")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, p.Reset(out))
	serviceMust(t, c.GetPolynomial(&a, out))
	v, e := p.wholeView(out)
	serviceMust(t, e)
	first, e := p.read(v, 0)
	serviceMust(t, e)
	text, e := z.String(first)
	serviceMust(t, e)
	if text != "2147483648" {
		t.Fatal("getter retry coefficient", text)
	}
	serviceMust(t, p.DelPolynomial(&out))
	serviceMust(t, c.Del(&a))
	serviceMust(t, qm.Del(&q))
}
