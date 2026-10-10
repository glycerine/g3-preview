package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialNativeSOMTagsPreserveAcquisitionAndForeignRows(t *testing.T) {
	full := polynomialPSCFullFixtureWithLimit(t, 1<<25)
	c := full.core
	const count = 257
	terms := make([]Monomial, count+1)
	for i := range terms {
		var e error
		terms[i], e = c.monomials.MkVariable(0, uint32(i+1))
		serviceMust(t, e)
		serviceMust(t, c.monomials.IncRef(terms[i]))
	}
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cheap.reset())
		for _, term := range terms {
			serviceMust(t, c.monomials.DecRef(term))
		}
	})
	foreign := util.NewInt(17)
	serviceMust(t, c.cheap.add(&foreign, terms[0], false))
	used, storage, cells := c.integer.Budget().Used(), c.memory.Stats(), c.integer.StorageStats()
	var receipt polynomialCheapAppendInvocationReceipt
	c.integer.Budget().FailAfter(0)
	for i, term := range terms[1:] {
		a := util.NewInt(int32(i + 1))
		serviceMust(t, c.cheap.addWithAppendReceipt(&a, term, false, &receipt))
		n := i + 1
		if c.cheap.coefficientOwners[n] != receipt.invocation || c.cheap.monomialOwners[n] != receipt.invocation || c.cheap.coefficientTransferred[n] != 0 || c.cheap.monomialTransferred[n] != 0 {
			t.Fatal("native growth changed actual acquisition authority", n)
		}
	}
	if c.integer.Budget().Used() != used || c.memory.Stats() != storage || c.integer.StorageStats() != cells {
		t.Fatal("native scalar tag growth changed numerical/object storage")
	}
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, false))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, true))
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	ms, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	a, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	if as != 1 || ms != 1 || a != foreign || term != terms[0] || c.integer.Budget().Used() != used {
		t.Fatal("matched cleanup changed earlier foreign row", as, ms, a, term)
	}
	for i, term := range terms {
		refs, e := c.monomialRefCount(term)
		serviceMust(t, e)
		want := uint32(1)
		if i == 0 {
			want = 2
		}
		if refs != want {
			t.Fatal("native tags changed actual monomial ownership", i, refs, want)
		}
	}
	c.integer.Budget().FailAfter(-1)
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
	polynomialOperationMonomialAudit(t, c, nil, terms)
}
