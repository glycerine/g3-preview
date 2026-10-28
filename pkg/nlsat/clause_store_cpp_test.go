package nlsat

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestClauseStoreConstructedTruthAndLiteralPreservationPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	e, s := clauseStoreFixture(t)
	var script strings.Builder
	script.WriteString("(declare-const x Real)(declare-const y Real)\n")
	formulas := map[sat.BoolVar]string{0: "true"}
	pool := []sat.BoolVar{0}
	for i := 0; i < 2; i++ {
		b, err := s.atoms.mkBoolVar()
		assignmentMust(t, err)
		name := fmt.Sprintf("b%d", b)
		fmt.Fprintf(&script, "(declare-const %s Bool)\n", name)
		formulas[b] = name
		pool = append(pool, b)
	}
	add := func(b sat.BoolVar, formula string) {
		formulas[b] = formula
		pool = append(pool, b)
		assignmentMust(t, s.atoms.incRef(b))
		t.Cleanup(func() { assignmentMust(t, s.atoms.decRef(b)) })
	}
	for _, item := range []struct {
		p    polynomial.Polynomial
		text string
	}{
		{evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 2}, evaluatorTerm{1, 1, 0}), "(- y (* x x))"},
		{evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 0, 0}), "(- x 1.0)"},
		{evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{2, 0, 0}), "(+ y 2.0)"},
	} {
		for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
			b, err := s.atoms.mkInequality(kind, []polynomial.Polynomial{item.p}, []bool{false})
			assignmentMust(t, err)
			add(b, "("+evaluatorSMTOperator(kind)+" "+item.text+" 0.0)")
		}
	}
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	for index := uint32(1); index <= 2; index++ {
		for _, kind := range []AtomKind{AtomRootEQ, AtomRootLT, AtomRootGT, AtomRootLE, AtomRootGE} {
			b, err := s.atoms.mkRoot(kind, 1, index, p)
			assignmentMust(t, err)
			add(b, fmt.Sprintf("(%s x (root-obj (+ (^ x 2) (- 2)) %d))", evaluatorSMTOperator(kind), index))
		}
	}
	clauseText := func(lits []sat.Literal) string {
		parts := make([]string, len(lits))
		for i, l := range lits {
			parts[i] = formulas[l.Variable()]
			if l.Sign() {
				parts[i] = "(not " + parts[i] + ")"
			}
		}
		return "(or " + strings.Join(parts, " ") + ")"
	}
	random := rand.New(rand.NewPCG(621, 816))
	queries := 0
	for world := 0; world < 64; world++ {
		lits := make([]sat.Literal, 1+random.IntN(10))
		for i := range lits {
			lits[i] = sat.MakeLiteral(pool[random.IntN(len(pool))], random.IntN(2) != 0)
		}
		original := clauseText(lits)
		c, err := s.mkClause(lits, world%2 != 0, nil)
		assignmentMust(t, err)
		fmt.Fprintf(&script, "(push)(assert (not (= %s %s)))(check-sat)(pop)\n", original, clauseText(c.Literals()))
		queries++
		for point, sample := range []struct {
			x int64
			d uint64
			y int64
		}{{-3, 2, -2}, {0, 1, 0}, {5, 3, 3}} {
			evaluatorSetFraction(t, e, 0, sample.y, 1)
			evaluatorSetFraction(t, e, 1, sample.x, sample.d)
			fmt.Fprintf(&script, "(push)(assert (= x (/ %d.0 %d.0)))(assert (= y %d.0))", sample.x, sample.d, sample.y)
			for b := sat.BoolVar(0); b <= 2; b++ {
				truth := b == 0 || (world+point+int(b))%2 == 0
				s.atoms.bools.values[b] = lbool.FromBool(truth)
				if b != 0 {
					formula := formulas[b]
					if !truth {
						formula = "(not " + formula + ")"
					}
					fmt.Fprintf(&script, "(assert %s)", formula)
				}
			}
			truth := false
			for _, l := range c.Literals() {
				a, err := s.literalAtom(l)
				assignmentMust(t, err)
				var value bool
				if a == nil {
					value = s.atoms.bools.values[l.Variable()] == lbool.True
					if l.Sign() {
						value = !value
					}
				} else {
					value, err = e.Eval(a, l.Sign())
					assignmentMust(t, err)
				}
				truth = truth || value
			}
			fmt.Fprintf(&script, "(assert (not (= %s %s)))(check-sat)(pop)\n", original, strconv.FormatBool(truth))
			queries++
		}
		assignmentMust(t, s.deleteClause(c))
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
			t.Fatal("constructed clause disagrees with C++", i, status)
		}
	}
	t.Logf("prebuilt C++ rejects all %d constructed-clause truth/literal-preservation disagreements", queries)
}
