package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNativeVariableNumeralHistoryPreservesReceipts(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{2, 1}}, nil})
	const count = 4097
	receipts := make([]polynomialVariableNumeralInvocationReceipt, count)
	n := util.NewInt(17)
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.integer.Del(&n))
	})
	used, before, cells := c.integer.Budget().Used(), c.memory.Stats(), c.integer.StorageStats()
	c.integer.Budget().FailAfter(0)
	for i := range receipts {
		x := uint32(77)
		yes, e := c.isVariableNumeralWithReceipt(p, &x, &n, &receipts[i])
		serviceMust(t, e)
		if !yes || x != 2 || n != util.NewInt(5) || receipts[i].invocation != uint64(i+1) {
			t.Fatal("independent variable-plus-constant recognition", i, yes, x, n)
		}
		if i != 0 {
			current := c.variableNumeral.invocation
			serviceMust(t, c.cleanupVariableNumeralInvocation(&receipts[i-1], true))
			if c.variableNumeral.invocation != current || n != util.NewInt(5) {
				t.Fatal("old diagnostic receipt changed later invocation")
			}
		}
	}
	if len(c.variableNumeralHistory) != count-1 || c.integer.Budget().Used() != used || c.memory.Stats() != before || c.integer.StorageStats() != cells {
		t.Fatal("native diagnostic growth changed actual numeric/object storage")
	}
	for i, receipt := range receipts {
		h, ok := c.variableNumeralDiagnostic(&receipt)
		if !ok || h.invocation != uint64(i+1) || h.bodyCause != nil || h.sourceCause != nil || h.cleanupCause != nil {
			t.Fatal("archived invocation lost diagnostic identity", i, h, ok)
		}
	}
	c.integer.Budget().FailAfter(-1)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&n}, 0)
}
