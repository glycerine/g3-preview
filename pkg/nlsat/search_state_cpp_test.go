package nlsat

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
)

func TestSearchUnitPropagationDomainsAndReasonsPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	type constraint struct {
		kind  AtomKind
		terms []evaluatorTerm
		text  string
		neg   bool
	}
	cases := []constraint{
		{AtomGT, []evaluatorTerm{{1, 0, 1}, {2, 0, 0}}, "(> (+ x 2.0) 0.0)", false},
		{AtomLT, []evaluatorTerm{{1, 0, 1}, {-3, 0, 0}}, "(< (- x 3.0) 0.0)", false},
		{AtomLT, []evaluatorTerm{{1, 0, 2}, {-2, 0, 0}}, "(< (- (* x x) 2.0) 0.0)", false},
		{AtomGT, []evaluatorTerm{{1, 0, 3}, {2, 0, 0}}, "(> (+ (* x x x) 2.0) 0.0)", false},
		{AtomEQ, []evaluatorTerm{{1, 0, 2}, {-2, 0, 0}}, "(= (* x x) 2.0)", false},
		{AtomGT, []evaluatorTerm{{1, 0, 1}}, "(> x 0.0)", false},
		{AtomLT, []evaluatorTerm{{1, 0, 1}}, "(< x 0.0)", false},
		{AtomLT, []evaluatorTerm{{1, 0, 2}, {1, 0, 0}}, "(< (+ (* x x) 1.0) 0.0)", false},
		{AtomGT, []evaluatorTerm{{1, 0, 2}, {1, 0, 0}}, "(> (+ (* x x) 1.0) 0.0)", false},
		{AtomLT, []evaluatorTerm{{-1, 0, 2}, {2, 0, 0}}, "(< (- 2.0 (* x x)) 0.0)", false},
		{AtomGT, []evaluatorTerm{{1, 0, 1}}, "(> x 0.0)", true},
	}
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	queries, conflicts, reasons := 0, 0, 0
	random := rand.New(rand.NewPCG(983, 816))
	for world := 0; world < 48; world++ {
		e, r := searchStateFixture(t)
		searchAtRealStage(t, r)
		base := uint32(len(r.trail))
		formulas := map[sat.BoolVar]string{}
		var chosen []string
		for step := 0; step < 4; step++ {
			item := cases[random.IntN(len(cases))]
			p := evaluatorPolynomial(t, e, item.terms...)
			b, err := r.store.atoms.mkInequality(item.kind, []polynomial.Polynomial{p}, []bool{false})
			assignmentMust(t, err)
			formulas[b] = item.text
			l := sat.MakeLiteral(b, item.neg)
			formula := item.text
			if item.neg {
				formula = "(not " + formula + ")"
			}
			c := searchClause(t, r, l)
			chosen = append(chosen, formula)
			ok, err := r.processClause(c, false)
			assignmentMust(t, err)
			conjunction := "(and " + strings.Join(chosen, " ") + ")"
			if !ok {
				fmt.Fprintf(&script, "(push)(assert %s)(check-sat)(pop)\n", conjunction)
				queries++
				conflicts++
			} else {
				fmt.Fprintf(&script, "(push)(assert (not (= %s (not %s))))(check-sat)(pop)\n", conjunction, intervalSMTSet(t, r.ism, r.store.vars.infeasible[1]))
				queries++
			}
			j := r.store.atoms.bools.justifications[b]
			if j.IsLazy() {
				lits, err := j.lazy.Literals()
				assignmentMust(t, err)
				parts := make([]string, len(lits))
				for i, why := range lits {
					text, found := formulas[why.Variable()]
					if !found {
						t.Fatal("reason refers to an unconstructed atom", why)
					}
					if why.Sign() {
						text = "(not " + text + ")"
					}
					parts[i] = text
				}
				fmt.Fprintf(&script, "(push)(assert (and %s))(check-sat)(pop)\n", strings.Join(parts, " "))
				queries++
				reasons++
			}
			if !ok {
				break
			}
		}
		assignmentMust(t, r.undoUntilSize(base))
		if r.store.vars.infeasible[1] != nil || r.store.vars.equalities[1] != nil || r.level != 0 {
			t.Fatal("backtracking did not restore actual domain/equality state")
		}
	}
	cmd := exec.Command(reference, "-in", "-T:30")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	statuses := strings.Fields(string(out))
	if len(statuses) != queries {
		t.Fatal("C++ errors or surplus output", len(statuses), queries, string(out))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("domain/conflict/reason disagrees with C++", i, status)
		}
	}
	if conflicts == 0 || reasons == 0 {
		t.Fatal("oracle did not reach conflict and lazy-reason cases")
	}
	t.Logf("prebuilt C++ rejects all %d domain/conflict/reason disagreements (%d conflicts, %d lazy reasons)", queries, conflicts, reasons)
}
