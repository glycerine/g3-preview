package nlsat

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func projectionWorklistFixture(t *testing.T) (*Evaluator, *searchState, *projectionWorklist) {
	t.Helper()
	e, r := searchStateFixture(t)
	p, err := newProjectionWorklist(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, p.close()) })
	return e, r, p
}
func projectionOutput(t *testing.T, e *Evaluator) *polynomial.PolynomialRefVector {
	t.Helper()
	v, err := polynomial.NewPolynomialRefVector(e.core)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, v.Close()) })
	return v
}
func projectionValues(t *testing.T, v *polynomial.PolynomialRefVector) []polynomial.Polynomial {
	t.Helper()
	n, err := v.Size()
	assignmentMust(t, err)
	r := make([]polynomial.Polynomial, n)
	for i := range r {
		r[i], err = v.Get(uint32(i))
		assignmentMust(t, err)
	}
	return r
}
func TestProjectionWorklistCanonicalInsertionAndStableMaximumExtraction(t *testing.T) {
	e, r, p := projectionWorklistFixture(t)
	x, err := r.store.mkVar(false)
	assignmentMust(t, err)
	if x != 2 {
		t.Fatal("actual variable registry", x)
	}
	polys := []polynomial.Polynomial{
		evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{1, 0, 0}),
		evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{2, 0, 0}),
		evaluatorPolynomial(t, e, evaluatorTerm{1, 2, 0}, evaluatorTerm{3, 0, 0}),
	}
	z, err := e.core.MkVariable(2, 1)
	assignmentMust(t, err)
	assignmentMust(t, e.core.IncRef(z))
	t.Cleanup(func() { assignmentMust(t, e.core.DecRef(z)) })
	polys = append(polys, z)
	order := []int{0, 3, 1, 2}
	for _, i := range order {
		assignmentMust(t, p.insert(polys[i]))
	}
	duplicate := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 0}, evaluatorTerm{1, 1, 0})
	assignmentMust(t, p.insert(duplicate))
	if len(projectionValues(t, p.set)) != 4 {
		t.Fatal("equal fresh polynomial did not canonicalize before membership")
	}
	out := projectionOutput(t, e)
	assignmentMust(t, out.Push(polys[0]))
	max, err := p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	if max != 2 || !reflect.DeepEqual(projectionValues(t, out), []polynomial.Polynomial{z}) || !reflect.DeepEqual(projectionValues(t, p.set), []polynomial.Polynomial{polys[0], polys[1], polys[2]}) {
		t.Fatal("maximum extraction/old output reset", max)
	}
	assignmentMust(t, p.insert(z)) // removed entries must be eligible again.
	max, err = p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	if max != 2 {
		t.Fatal("reinsertion mark", max)
	}
	max, err = p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	if max != 1 || !reflect.DeepEqual(projectionValues(t, out), []polynomial.Polynomial{polys[1]}) {
		t.Fatal("next projection level", max)
	}
	max, err = p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	if max != 0 || !reflect.DeepEqual(projectionValues(t, out), []polynomial.Polynomial{polys[0], polys[2]}) {
		t.Fatal("stable order for same maximal variable", max)
	}
	max, err = p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	empty, err2 := p.empty()
	assignmentMust(t, err2)
	if max != NullVariable || !empty || len(projectionValues(t, out)) != 0 {
		t.Fatal("empty maximum/reset", max, empty)
	}
	for _, v := range polys {
		assignmentMust(t, p.insert(v))
	}
	assignmentMust(t, p.reset())
	for _, mark := range p.inSet {
		if mark {
			t.Fatal("reset retained published membership")
		}
	}
	for _, v := range polys {
		assignmentMust(t, p.insert(v))
	}
}
func TestProjectionWorklistIndependentRandomSequenceAndReferences(t *testing.T) {
	e, _, p := projectionWorklistFixture(t)
	out := projectionOutput(t, e)
	var polys []polynomial.Polynomial
	var variables []uint32
	for x := uint32(0); x < 2; x++ {
		for degree := uint32(1); degree <= 5; degree++ {
			terms := evaluatorTerm{1, degree, 0}
			if x == 1 {
				terms = evaluatorTerm{1, 0, degree}
			}
			v := evaluatorPolynomial(t, e, terms)
			assignmentMust(t, p.insert(v))
			polys = append(polys, v)
			variables = append(variables, x)
		}
	}
	assignmentMust(t, p.reset())
	random := rand.New(rand.NewPCG(831, 19))
	var model []int
	for step := 0; step < 100; step++ {
		if random.IntN(3) != 0 {
			i := random.IntN(len(polys))
			assignmentMust(t, p.insert(polys[i]))
			found := false
			for _, j := range model {
				found = found || i == j
			}
			if !found {
				model = append(model, i)
			}
		} else {
			wantMax := uint32(NullVariable)
			for _, i := range model {
				if wantMax == NullVariable || variables[i] > wantMax {
					wantMax = variables[i]
				}
			}
			var keep []int
			var selected []polynomial.Polynomial
			for _, i := range model {
				if variables[i] == wantMax {
					selected = append(selected, polys[i])
				} else {
					keep = append(keep, i)
				}
			}
			max, err := p.removeMaxPolynomials(out)
			assignmentMust(t, err)
			actual := projectionValues(t, out)
			if max != wantMax || len(actual) != len(selected) {
				t.Fatal("independent max/group", step, max, wantMax)
			}
			for i, v := range selected {
				if actual[i] != v {
					t.Fatal("unstable group", step)
				}
			}
			model = keep
		}
		actual := projectionValues(t, p.set)
		if len(actual) != len(model) {
			t.Fatal("membership cardinality", step)
		}
		for i, j := range model {
			if actual[i] != polys[j] {
				t.Fatal("survivor compaction order", step)
			}
		}
		for _, v := range polys {
			want := uint32(2) // fixture plus canonical cache owner
			for _, j := range model {
				if polys[j] == v {
					want++
				}
			}
			for _, q := range projectionValues(t, out) {
				if q == v {
					want++
				}
			}
			info, err := e.core.Info(v)
			assignmentMust(t, err)
			if info.RefCount != want {
				t.Fatal("independent owning multiplicity", step, info.RefCount, want)
			}
		}
	}
}
func TestProjectionWorklistConstructorAndConstantGuards(t *testing.T) {
	atomDebugFixture(t)
	for _, stop := range []int64{0, 1} {
		_, r := searchStateFixture(t)
		b := r.store.atoms.ineq.budget
		before := b.Used()
		b.FailAfter(stop)
		p, err := newProjectionWorklist(r)
		b.FailAfter(-1)
		if !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("constructor reservation", stop, err)
		}
		if p != nil {
			assignmentMust(t, p.cleanup())
			if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
				t.Fatal("search accepted an incomplete projection constructor", err)
			}
			assignmentMust(t, p.close())
		}
		if r.projection != nil || b.Used() != before {
			t.Fatal("failed constructor retained actual owner", stop)
		}
	}
	e, r, p := projectionWorklistFixture(t)
	c := evaluatorPolynomial(t, e, evaluatorTerm{3, 0, 0})
	assignmentMust(t, p.insert(c))
	max, err := p.maxVariable()
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("source constant max guard", max, err)
		}
	} else {
		assignmentMust(t, err)
		if max != NullVariable {
			t.Fatal("constant max", max)
		}
	}
	assignmentMust(t, p.reset())
	assignmentMust(t, r.close())
	if !p.closed {
		t.Fatal("search did not close its projection subobject")
	}
}

func TestProjectionWorklistCleanupPreservesBorrowedOutputFailure(t *testing.T) {
	e, _, p := projectionWorklistFixture(t)
	v := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	assignmentMust(t, p.insert(v))
	out := projectionOutput(t, e)
	old, err := e.core.MkVariable(0, 7)
	assignmentMust(t, err)
	assignmentMust(t, out.Push(old))
	stop := errors.New("stop borrowed output destructor")
	calls := 0
	h, err := polynomial.NewPolynomialDeletionHandler(e.budget, func(_ *polynomial.PolynomialDeletionHandler, _ *polynomial.MultivariatePolynomialCore, value polynomial.Polynomial) error {
		if value == old {
			calls++
			return stop
		}
		return nil
	})
	assignmentMust(t, err)
	assignmentMust(t, e.core.AddDeletionHandler(h))
	t.Cleanup(func() { assignmentMust(t, e.core.RemoveDeletionHandler(h)); assignmentMust(t, h.CloseChecked()) })
	_, err = p.removeMaxPolynomials(out)
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatal("borrowed output reset did not reach source refusal", err, calls)
	}
	stats, used := p.search.ism.allocator.RuntimeOwner().Stats(), e.budget.Used()
	assignmentMust(t, p.cleanup())
	n, err := out.Size()
	assignmentMust(t, err)
	info, err := e.core.Info(old)
	assignmentMust(t, err)
	if n != 1 || info.RefCount != 0 || calls != 1 || p.search.ism.allocator.RuntimeOwner().Stats() != stats || e.budget.Used() != used {
		t.Fatal("worklist cleanup adopted caller output destruction", n, info.RefCount, calls)
	}
	if len(projectionValues(t, p.set)) != 1 {
		t.Fatal("output reset refusal mutated worklist")
	}
	assignmentMust(t, out.CleanupRetained())
	max, err := p.removeMaxPolynomials(out)
	assignmentMust(t, err)
	if max != 1 || !reflect.DeepEqual(projectionValues(t, out), []polynomial.Polynomial{v}) || calls != 1 {
		t.Fatal("caller cleanup retry replayed deletion", max, calls)
	}
}

func TestProjectionWorklistStoppedCloseBlocksSearch(t *testing.T) {
	e, r, p := projectionWorklistFixture(t)
	v, err := e.core.MkVariable(1, 7)
	assignmentMust(t, err)
	assignmentMust(t, p.insert(v))
	// Remove the cache lease through its actual owner; the worklist now owns
	// the final reference. No test counter or source ownership field is repaired.
	assignmentMust(t, r.store.atoms.ineq.polynomials.Reset())
	stop := errors.New("stop worklist owning destructor")
	calls := 0
	h, err := polynomial.NewPolynomialDeletionHandler(e.budget, func(_ *polynomial.PolynomialDeletionHandler, _ *polynomial.MultivariatePolynomialCore, value polynomial.Polynomial) error {
		if value == v {
			calls++
			return stop
		}
		return nil
	})
	assignmentMust(t, err)
	assignmentMust(t, e.core.AddDeletionHandler(h))
	t.Cleanup(func() { assignmentMust(t, e.core.RemoveDeletionHandler(h)); assignmentMust(t, h.CloseChecked()) })
	err = p.close()
	if !errors.Is(err, stop) || calls != 1 || !p.failed || r.projection != p {
		t.Fatal("owning close lost its pending borrower", err, calls)
	}
	if err := r.newLevel(); !errors.Is(err, ErrSearchPending) {
		t.Fatal("search progressed during stopped worklist close", err)
	}
	assignmentMust(t, p.close())
	if calls != 1 || r.projection != nil || !p.closed {
		t.Fatal("continued close replayed source drop", calls)
	}
}
