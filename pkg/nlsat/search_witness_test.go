package nlsat

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestSearchWitnessEmptyRandomIntegerAndHiddenReplacement(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, randomize := range []bool{false, true} {
			for _, seed := range []uint32{0, 1, 17, ^uint32(0)} {
				e, r := searchStateFixture(t)
				assignmentMust(t, r.newStage())
				if !integer {
					assignmentMust(t, r.newStage())
				}
				r.config.Randomize = randomize
				assignmentMust(t, r.ism.SetSeed(seed))
				// ResetVar retains an owning hidden cell. SET_CORE must return it
				// to the lexical witness for destruction, rather than leaking it.
				old := assignmentInteger(t, e.am, 2)
				assignmentMust(t, e.am.Root(old, 2, old))
				assignmentMust(t, r.assignment.Set(r.xk, old))
				assignmentMust(t, r.assignment.ResetVar(r.xk))
				assignmentMust(t, r.selectWitness())
				if r.witnessActive || r.irrationalAssignments != 0 {
					t.Fatal("lexical witness/statistic", r.witnessActive, r.irrationalAssignments)
				}
				value, err := r.assignment.Value(r.xk)
				assignmentMust(t, err)
				q, err := e.am.RationalManager()
				assignmentMust(t, err)
				want := unumeral.NewRational()
				state := seed
				if randomize {
					var sign, denK uint32
					state, sign = intervalNext(state)
					state, denK = intervalNext(state)
					num, den := int64(-1), uint64(1)
					if sign%2 == 0 {
						num = 1
					}
					if !integer {
						den = 1 << (denK % 4)
					}
					assignmentMust(t, q.SetFraction(&want, num, den))
				}
				cmp, err := e.am.CompareRational(value, &want)
				assignmentMust(t, err)
				assignmentMust(t, q.Del(&want))
				_, next := intervalNext(state)
				if cmp != 0 || r.ism.rand.Next() != next {
					t.Fatal("selected value or random draw count", integer, randomize, seed)
				}
				assignmentMust(t, r.newStage())
				assignmentMust(t, r.undoUntilEmpty())
				assigned, err := r.assignment.Contains(0)
				assignmentMust(t, err)
				if assigned {
					t.Fatal("stage undo retained assigned flag")
				}
			}
		}
	}
}

func searchWitnessRootDomain(t *testing.T, e *Evaluator, r *searchState) {
	t.Helper()
	searchAtRealStage(t, r)
	l, c := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	ok, err := r.processClause(c, false)
	assignmentMust(t, err)
	if !ok {
		t.Fatal("irrational equality conflicted", l)
	}
}

func TestSearchWitnessIntegerGapDefersIntegralityCheck(t *testing.T) {
	e, r := searchStateFixture(t)
	assignmentMust(t, r.newStage()) // x0 is an integer variable.
	_, lower := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 1, 0})
	_, upper := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	for _, c := range []*Clause{lower, upper} {
		ok, err := r.processClause(c, false)
		assignmentMust(t, err)
		if !ok {
			t.Fatal("real relaxation of integer gap rejected")
		}
	}
	assignmentMust(t, r.selectWitness())
	v, err := r.assignment.Value(0)
	assignmentMust(t, err)
	q, err := e.am.RationalManager()
	assignmentMust(t, err)
	half := unumeral.NewRational()
	assignmentMust(t, q.SetFraction(&half, 1, 2))
	cmp, err := e.am.CompareRational(v, &half)
	assignmentMust(t, err)
	assignmentMust(t, q.Del(&half))
	if cmp != 0 || r.irrationalAssignments != 0 {
		t.Fatal("integer witness prematurely rounded or rejected", cmp)
	}
}

func TestSearchWitnessFullDomainAndStageGuards(t *testing.T) {
	atomDebugFixture(t)
	_, r := searchStateFixture(t)
	if err := r.selectWitness(); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("missing stage guard", err)
	}
	assignmentMust(t, r.cleanup())
	assignmentMust(t, r.newStage())
	full, err := r.ism.Mk(true, true, nil, true, true, nil, TrueLiteral)
	assignmentMust(t, err)
	assignmentMust(t, r.ism.IncRef(full))
	// Qualified invalid source state, to reach the full-domain assertion.
	r.store.vars.infeasible[0] = full
	err = r.selectWitness()
	r.store.vars.infeasible[0] = nil
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("full-domain source guard", err)
		}
	} else if !errors.Is(err, arena.ErrBounds) {
		t.Fatal("full-domain refusal", err)
	}
	if r.irrationalAssignments != 0 {
		t.Fatal("guarded call advanced statistic")
	}
	assignmentMust(t, r.cleanup())
	assignmentMust(t, r.ism.DecRef(full))
}

func TestSearchWitnessIrrationalAndStoppedAssignmentOwnership(t *testing.T) {
	assignmentStops, successes := 0, 0
	for stop := int64(0); stop < 32; stop++ {
		e, r := searchStateFixture(t)
		searchWitnessRootDomain(t, e, r)
		e.budget.FailAfter(stop)
		err := r.selectWitness()
		e.budget.FailAfter(-1)
		priorCount := r.irrationalAssignments
		if err != nil {
			if !errors.Is(err, arena.ErrInjectedFailure) && err.Error() != "out of memory" {
				t.Fatal("unexpected witness stop", stop, err)
			}
			if !r.witnessActive || priorCount > 1 {
				t.Fatal("stopped lexical witness/statistic", stop, priorCount)
			}
			if priorCount == 1 {
				assignmentStops++
			}
			if assigned, err := r.assignment.Contains(1); err != nil || assigned {
				t.Fatal("stopped SET published assignment", assigned, err)
			}
			// The actual failed algebraic COPY owns any numerical construction prefix.
			// Acknowledge that callee explicitly before retiring this lexical value.
			searchAcknowledgeNumericalCallees(t, e)
			assignmentMust(t, r.cleanup())
			zero, err := e.am.IsZero(&r.witness)
			assignmentMust(t, err)
			if r.witnessActive || !zero {
				t.Fatal("owning cleanup retained the stopped lexical witness", stop)
			}
			assignmentMust(t, r.selectWitness())
		} else {
			successes++
		}
		value, err := r.assignment.Value(1)
		assignmentMust(t, err)
		rational, err := e.am.IsRational(value)
		assignmentMust(t, err)
		want := priorCount + 1
		if successes > 0 {
			want = 1
		}
		if rational || r.irrationalAssignments != want || r.witnessActive {
			t.Fatal("irrational witness/cumulative statistic", rational, r.irrationalAssignments, want)
		}
		assignmentMust(t, r.newStage())
		assignmentMust(t, r.undoUntilEmpty())
		if r.irrationalAssignments != want {
			t.Fatal("undo rewound cumulative statistic")
		}
		if successes > 0 {
			break
		}
	}
	if assignmentStops != 2 || successes != 1 {
		t.Fatal("did not cover both actual assignment growth reservations through success", assignmentStops, successes)
	}
}

func TestSearchSelectedWitnessPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	queries := 0
	for trial := 0; trial < 32; trial++ {
		e, r := searchStateFixture(t)
		searchAtRealStage(t, r)
		r.config.Randomize = trial%2 != 0
		assignmentMust(t, r.ism.SetSeed(uint32(trial)))
		var formula string
		var literals []sat.Literal
		if trial%4 == 0 {
			l, c := searchInequality(t, e, r, AtomEQ, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
			ok, err := r.processClause(c, false)
			assignmentMust(t, err)
			if !ok {
				t.Fatal("sqrt2 conflict")
			}
			literals = append(literals, l)
			formula = "(= (* x x) 2.0)"
		} else {
			bound := int32(trial%7 - 3)
			lower, c1 := searchInequality(t, e, r, AtomGT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-bound, 0, 0})
			upper, c2 := searchInequality(t, e, r, AtomLT, evaluatorTerm{1, 0, 1}, evaluatorTerm{-(bound + 2), 0, 0})
			for _, c := range []*Clause{c1, c2} {
				ok, err := r.processClause(c, false)
				assignmentMust(t, err)
				if !ok {
					t.Fatal("nonempty interval conflicted")
				}
			}
			literals = append(literals, lower, upper)
			formula = fmt.Sprintf("(and (> x %s) (< x %s))", intervalSMTInteger(int32(bound)), intervalSMTInteger(int32(bound+2)))
		}
		assignmentMust(t, r.selectWitness())
		for _, l := range literals {
			ok, err := e.Eval(r.store.atoms.bools.atoms[l.Variable()], l.Sign())
			assignmentMust(t, err)
			if !ok {
				t.Fatal("selected witness violates propagated atom", trial, l)
			}
		}
		v, err := r.assignment.Value(1)
		assignmentMust(t, err)
		fmt.Fprintf(&script, "(push)(assert (= x %s))(assert (not %s))(check-sat)(pop)\n", intervalSMTValue(t, r.ism, v), formula)
		queries++
	}
	cmd := exec.Command(reference, "-in", "-smt2")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C++ witness oracle: %v\n%s", err, out)
	}
	answers := strings.Fields(string(out))
	if len(answers) != queries {
		t.Fatalf("C++ answers %d, want %d: %s", len(answers), queries, out)
	}
	for i, answer := range answers {
		if answer != "unsat" {
			t.Fatalf("witness counterexample %d: %s", i, answer)
		}
	}
}
