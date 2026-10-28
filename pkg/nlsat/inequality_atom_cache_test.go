package nlsat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func inequalityCacheFixture(t *testing.T) (*Evaluator, *inequalityAtomCache) {
	t.Helper()
	e := evaluatorFixture(t)
	pc, err := polynomial.NewPolynomialCache(e.pm)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, pc.Close()) })
	c, err := newInequalityAtomCache(e.pm, pc, e.ism.allocator)
	assignmentMust(t, err)
	t.Cleanup(func() {
		assignmentMust(t, c.cleanup())
		var atoms []*InequalityAtom
		for a := range c.owned {
			atoms = append(atoms, a)
		}
		for _, a := range atoms {
			if c.owned[a].inTable {
				assignmentMust(t, c.erase(a))
			}
			assignmentMust(t, c.release(a))
		}
		assignmentMust(t, c.close())
	})
	return e, c
}
func TestInequalityFactoryCanonicalIdentityParityOrderAndHeader(t *testing.T) {
	atomDebugFixture(t)
	e, c := inequalityCacheFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 0})
	duplicate := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 0})
	negative := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 2}, evaluatorTerm{-1, 1, 0})
	if p == duplicate {
		t.Fatal("fixture needs distinct equal polynomial cells")
	}
	a, fresh, err := c.mk(AtomGT, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	if !fresh || a.Kind() != AtomGT || a.BooleanVariable() != sat.NullBoolVar || a.RefCount() != 0 || a.MaxVariable() != 1 {
		t.Fatal("source inner factory/header")
	}
	a.boolVar = 17
	a.IncRef()
	before, err := e.core.Info(p)
	assignmentMust(t, err)
	for _, test := range []struct {
		k AtomKind
		p polynomial.Polynomial
	}{{AtomGT, duplicate}, {AtomLT, negative}, {AtomGT, p}} {
		found, isNew, err := c.mk(test.k, []polynomial.Polynomial{test.p}, []bool{false})
		assignmentMust(t, err)
		if isNew || found != a || found.boolVar != 17 || found.refs != 1 {
			t.Fatal("canonical identity/header replaced")
		}
	}
	after, err := e.core.Info(p)
	assignmentMust(t, err)
	if before.RefCount != after.RefCount {
		t.Fatal("duplicate acquired extra factor references", before, after)
	}
	even, isNew, err := c.mk(AtomGT, []polynomial.Polynomial{p}, []bool{true})
	assignmentMust(t, err)
	if !isNew || even == a || a.Equal(even) {
		t.Fatal("factor parity omitted from canonical equality")
	}
	h1, err := a.Hash(e.core)
	assignmentMust(t, err)
	h2, err := even.Hash(e.core)
	assignmentMust(t, err)
	if h1 != h2 {
		t.Fatal("source hash includes parity")
	}
	q := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	ab, _, err := c.mk(AtomEQ, []polynomial.Polynomial{p, q}, []bool{false, true})
	assignmentMust(t, err)
	ba, isNew, err := c.mk(AtomEQ, []polynomial.Polynomial{q, p}, []bool{true, false})
	assignmentMust(t, err)
	if !isNew || ab == ba || ab.MaxVariable() != 1 || ba.MaxVariable() != 1 {
		t.Fatal("factor order or maximal variable")
	}
	want := ErrAtomCacheLive
	if diagnostic.DebugBuild {
		want = diagnostic.ErrDebugAssertion
	}
	if err = c.erase(a); !errors.Is(err, want) || a.refs != 1 || !c.owned[a].inTable {
		t.Fatal("erased referenced atom", err)
	}
	assignmentMust(t, a.DecRef())
}

func TestInequalityFactoryRetirementOwnsFactorsAcrossPolynomialReset(t *testing.T) {
	e, c := inequalityCacheFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	before, err := e.core.Info(p)
	assignmentMust(t, err)
	a, fresh, err := c.mk(AtomGT, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	if !fresh || len(c.owned) != 1 {
		t.Fatal("factory failed to publish one owned atom")
	}
	if err = c.close(); err != ErrAtomCacheLive {
		t.Fatal("closed with a published atom", err)
	}
	if err = c.release(a); err != ErrAtomCacheOwner {
		t.Fatal("released before table erasure", err)
	}
	assignmentMust(t, c.polynomials.Reset())
	retained, err := e.core.Info(p)
	assignmentMust(t, err)
	if retained.RefCount != before.RefCount+1 {
		t.Fatal("atom did not retain its own factor across cache reset", before, retained)
	}
	assignmentMust(t, c.erase(a))
	erased, err := e.core.Info(p)
	assignmentMust(t, err)
	if erased.RefCount != retained.RefCount {
		t.Fatal("table erasure retired factors before Boolean-variable deletion")
	}
	assignmentMust(t, c.release(a))
	after, err := e.core.Info(p)
	assignmentMust(t, err)
	if after.RefCount != before.RefCount || len(c.owned) != 0 {
		t.Fatal("factor or atom allocation survived retirement", before, after)
	}
}

func TestInequalityFactorySourceSignFlipAndEvaluator(t *testing.T) {
	e, c := inequalityCacheFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{-2, 0, 3}, evaluatorTerm{4, 1, 0})
	q := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 1}, evaluatorTerm{1, 0, 0})
	for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
		for count := 1; count <= 2; count++ {
			ps := []polynomial.Polynomial{p, q}[:count]
			a, _, err := c.mk(kind, ps, make([]bool, count))
			assignmentMust(t, err)
			wantKind := kind
			if count%2 == 1 {
				wantKind, err = kind.Flip()
				assignmentMust(t, err)
			}
			if a.Kind() != wantKind {
				t.Fatal("source sign flip", kind, count, a.Kind(), wantKind)
			}
			a.boolVar = 31
			for y := -1; y <= 1; y++ {
				evaluatorSetFraction(t, e, 0, int64(y), 1)
				for x := -2; x <= 2; x++ {
					evaluatorSetFraction(t, e, 1, int64(x), 1)
					product := -2*x*x*x + 4*y
					if count == 2 {
						product *= 1 - x
					}
					for _, neg := range []bool{false, true} {
						got, err := e.Eval(a, neg)
						assignmentMust(t, err)
						want := evaluatorTruth(product, kind, neg)
						if got != want {
							t.Fatal("factory/evaluator sign", x, y, kind, count, neg, got, want)
						}
					}
				}
			}
		}
	}
	// The source flips on actual polynomial identity changes even for an
	// even-tagged factor; the factory does not reinterpret this as exponent math.
	a, _, err := c.mk(AtomGT, []polynomial.Polynomial{p}, []bool{true})
	assignmentMust(t, err)
	if a.Kind() != AtomLT {
		t.Fatal("source factory sign schedule changed for parity")
	}
}
func TestInequalityFactoryAllocationCleanupAndReuse(t *testing.T) {
	for _, stop := range []int64{0, 1, 2, 3, 4, 5, 8, 12, 16, 24, 32, 64, 128, -1} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, c := inequalityCacheFixture(t)
			p := evaluatorPolynomial(t, e, evaluatorTerm{-2, 0, 3}, evaluatorTerm{4, 1, 0})
			q := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 1, 0})
			c.budget.FailAfter(stop)
			a, _, err := c.mk(AtomLT, []polynomial.Polynomial{p, q}, []bool{false, true})
			c.budget.FailAfter(-1)
			assignmentMust(t, e.pm.CleanupFlipSignIfLeadingMonomialNegative())
			assignmentMust(t, c.polynomials.CleanupUnique())
			assignmentMust(t, c.cleanup())
			var physical umemory.Failure
			if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
				t.Fatal("unexpected factory refusal", err)
			}
			if err != nil && a != nil {
				t.Fatal("failed factory published an atom")
			}
			if c.factor != nil || c.pending != nil || c.allocationFailed || c.failed {
				t.Fatal("retained factory state")
			}
			retry, _, err := c.mk(AtomLT, []polynomial.Polynomial{p, q}, []bool{false, true})
			assignmentMust(t, err)
			again, fresh, err := c.mk(AtomLT, []polynomial.Polynomial{p, q}, []bool{false, true})
			assignmentMust(t, err)
			if fresh || retry != again {
				t.Fatal("factory did not recover canonical reuse")
			}
		})
	}
}
