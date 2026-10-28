package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSearchMainBooleanSATAndUNSAT(t *testing.T) {
	for _, tc := range []struct {
		name  string
		unsat bool
	}{
		{name: "sat"},
		{name: "unsat", unsat: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, r := searchStateFixture(t)
			q, err := newSearchResolution(r)
			assignmentMust(t, err)
			t.Cleanup(func() { assignmentMust(t, q.close()) })
			b, err := r.store.atoms.mkBoolVar()
			assignmentMust(t, err)
			l := sat.MakeLiteral(b, false)
			searchClause(t, r, l)
			if tc.unsat {
				searchClause(t, r, l.Negated())
			}
			result, err := r.search(q, nil)
			assignmentMust(t, err)
			want := lbool.True
			if tc.unsat {
				want = lbool.False
			}
			if result != want {
				t.Fatal("Boolean main-search result", result, want)
			}
			if !tc.unsat {
				value, err := r.assignedValue(l)
				assignmentMust(t, err)
				if value != lbool.True {
					t.Fatal("Boolean model does not satisfy unit", value)
				}
			}
		})
	}
}

func TestSearchMainSelectsArithmeticWitness(t *testing.T) {
	e, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	lower, _ := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 0, 0})
	upper, _ := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-2, 0, 0})

	result, err := r.search(q, nil)
	assignmentMust(t, err)
	if result != lbool.True {
		t.Fatal("bounded arithmetic search result", result)
	}
	for _, literal := range []sat.Literal{lower, upper} {
		value, err := r.value(literal)
		assignmentMust(t, err)
		if value != lbool.True {
			t.Fatal("selected witness violates arithmetic clause", literal, value)
		}
	}
}

func TestSearchMainLearnsAcrossBooleanStage(t *testing.T) {
	_, r := searchStateFixture(t)
	q, err := newSearchResolution(r)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, q.close()) })
	b, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	c, err := r.store.atoms.mkBoolVar()
	assignmentMust(t, err)
	bl := sat.MakeLiteral(b, false)
	cl := sat.MakeLiteral(c, false)
	searchClause(t, r, bl, cl)
	searchClause(t, r, bl, cl.Negated())
	searchClause(t, r, bl.Negated(), cl)

	result, err := r.search(q, nil)
	assignmentMust(t, err)
	if result != lbool.True || len(r.store.learned) != 1 {
		t.Fatal("Boolean search did not learn through the main loop", result, len(r.store.learned))
	}
	for _, literal := range []sat.Literal{bl, cl} {
		value, err := r.assignedValue(literal)
		assignmentMust(t, err)
		if value != lbool.True {
			t.Fatal("learned Boolean model", literal, value)
		}
	}
}
