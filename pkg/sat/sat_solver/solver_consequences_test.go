package sat_solver_test

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func consequenceText(t *testing.T, f *encoderFixture, expressions []ast.ASTHandle) string {
	t.Helper()
	printer, err := ast.NewSimpleASTPrinterContext(f.manager, nil, nil)
	incMust(t, err)
	var out strings.Builder
	for _, expression := range expressions {
		out.WriteByte('\n')
		incMust(t, printer.DisplayExpr(&out, expression, 0))
	}
	return out.String()
}

// These vectors contain simple identifiers and numerals. Canonicalize only
// conjunction order: the source SAT antecedent hashtable has unspecified key
// iteration, while the Go port deliberately uses insertion-ordered insdict.
func canonicalConsequenceOutput(t *testing.T, text string) string {
	t.Helper()
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(text))
	index := 0
	var read func() string
	read = func() string {
		if index >= len(tokens) {
			t.Fatal("truncated consequence output")
		}
		token := tokens[index]
		index++
		if token != "(" {
			if token == ")" {
				t.Fatal("unexpected closing parenthesis")
			}
			return token
		}
		var children []string
		for index < len(tokens) && tokens[index] != ")" {
			children = append(children, read())
		}
		if index == len(tokens) {
			t.Fatal("unterminated consequence output")
		}
		index++
		if len(children) > 0 && children[0] == "and" {
			sort.Strings(children[1:])
		}
		return "(" + strings.Join(children, " ") + ")"
	}
	var forms []string
	for index < len(tokens) {
		forms = append(forms, read())
	}
	return strings.Join(forms, " ")
}

func TestIncrementalSolverConsequencesCPP(t *testing.T) {
	cases := []struct {
		name                               string
		assertions, assumptions, variables []string
		count                              int
		status                             lbool.LBool
	}{
		{"guarded", []string{`(=> p (= x #x3))`}, []string{`p`}, []string{`p`, `q`, `x`, `y`}, 2, lbool.True},
		{"fixed", []string{`(= x #x5)`}, nil, []string{`p`, `x`, `y`}, 1, lbool.True},
		{"partial", []string{`(= ((_ extract 0 0) x) #b1)`}, nil, []string{`x`, `y`}, 0, lbool.True},
		{"negative", []string{`(=> (not p) (= x #x7))`}, []string{`(not p)`}, []string{`p`, `x`}, 2, lbool.True},
		{"premises", []string{`(=> (and p q) (= x #x1))`}, []string{`p`, `q`}, []string{`x`}, 1, lbool.True},
		{"nonconstant", []string{`(= x #x5)`}, nil, []string{`(bvadd x #x1)`}, 0, lbool.True},
		{"contradictory", []string{`(= x #x1)`, `(= x #x2)`}, nil, []string{`x`}, 0, lbool.False},
		{"assumption-core", []string{`(=> p q)`}, []string{`p`, `(not q)`}, []string{`p`, `q`}, 0, lbool.False},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, s := newSolverFixture(t)
			var script strings.Builder
			script.WriteString(`(set-logic QF_BV)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-const p Bool)(declare-const q Bool)`)
			for _, term := range tc.assertions {
				incMust(t, s.Assert(solverForm(t, f, term)))
				script.WriteString("(assert " + term + ")")
			}
			var assumptions, variables []ast.ASTHandle
			for _, term := range tc.assumptions {
				assumptions = append(assumptions, solverForm(t, f, term))
			}
			for _, term := range tc.variables {
				h, err := f.context.MakeApp(term, nil, nil, ast.ASTHandle{})
				if strings.HasPrefix(term, "(") {
					h = solverForm(t, f, "(= "+term+" "+term+")")
					h, err = f.manager.GetArg(h, 0)
				}
				incMust(t, err)
				variables = append(variables, h)
			}
			status, consequences, err := s.GetConsequences(assumptions, variables)
			incMust(t, err)
			if status != tc.status || len(consequences) != tc.count {
				t.Fatal("consequences", status, consequenceText(t, f, consequences), tc.count)
			}
			if status == lbool.False && len(assumptions) > 0 {
				core, err := s.UnsatCore()
				incMust(t, err)
				if len(core) != 2 {
					t.Fatal("unsat assumptions core", core)
				}
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				script.WriteString("(get-consequences (" + strings.Join(tc.assumptions, " ") + ") (" + strings.Join(tc.variables, " ") + "))")
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script.String())
				out, err := cmd.CombinedOutput()
				incMust(t, err)
				want := map[lbool.LBool]string{lbool.True: "sat", lbool.False: "unsat"}[status] + consequenceText(t, f, consequences)
				if canonicalConsequenceOutput(t, string(out)) != canonicalConsequenceOutput(t, want) {
					t.Fatal("C++ consequence mismatch", string(out), want)
				}
			}
		})
	}
}

func TestIncrementalSolverMutexesPendingAndSigned(t *testing.T) {
	for _, negative := range []bool{false, true} {
		t.Run(fmt.Sprint(negative), func(t *testing.T) {
			f, s := newSolverFixture(t)
			var variables []ast.ASTHandle
			for _, name := range []string{"p", "q", "r", "v"} {
				term := name
				if negative {
					term = "(not " + name + ")"
				}
				variables = append(variables, solverForm(t, f, term))
			}
			for _, pair := range [][2]string{{"p", "q"}, {"p", "r"}, {"q", "r"}} {
				a, b := "(not "+pair[0]+")", "(not "+pair[1]+")"
				if negative {
					a, b = pair[0], pair[1]
				}
				incMust(t, s.Assert(solverForm(t, f, "(or "+a+" "+b+")")))
			}
			status, rows, err := s.FindMutexes(variables)
			incMust(t, err)
			if status != lbool.True || len(rows) != 0 {
				t.Fatal("mutex query unexpectedly compiled pending formulas", status, rows)
			}
			incMust(t, s.Push()) // internalizes the pending queue without solving it.
			status, rows, err = s.FindMutexes(variables)
			incMust(t, err)
			if status != lbool.True || len(rows) != 1 || len(rows[0]) != 3 {
				t.Fatal("binary-clause clique", status, rows)
			}
			seen := map[ast.ASTHandle]bool{}
			for _, h := range rows[0] {
				seen[h] = true
			}
			if !seen[variables[0]] || !seen[variables[1]] || !seen[variables[2]] || seen[variables[3]] {
				t.Fatal("signed clique inverse map", rows)
			}
		})
	}
}

func TestIncrementalSolverWideConsequencesCPP(t *testing.T) {
	for _, width := range []int{1, 4, 65, 257} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			f, s := newSolverFixture(t)
			declaration := fmt.Sprintf(`(declare-const wide (_ BitVec %d))`, width)
			incMust(t, smt2.ParseCommands(f.context, strings.NewReader(declaration), io.Discard))
			var integer big.Int
			integer.Lsh(big.NewInt(1), uint(width))
			integer.Sub(&integer, big.NewInt(1))
			term := fmt.Sprintf(`(=> p (= wide (_ bv%s %d)))`, integer.String(), width)
			incMust(t, s.Assert(solverForm(t, f, term)))
			p := solverForm(t, f, `p`)
			wide, err := f.context.MakeApp("wide", nil, nil, ast.ASTHandle{})
			incMust(t, err)
			status, consequences, err := s.GetConsequences([]ast.ASTHandle{p}, []ast.ASTHandle{wide})
			incMust(t, err)
			if status != lbool.True || len(consequences) != 1 {
				t.Fatal("wide fixed consequence", status, len(consequences))
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				script := `(set-logic QF_BV)(declare-const p Bool)` + declaration + "(assert " + term + ")(get-consequences (p) (wide))"
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				out, err := cmd.CombinedOutput()
				incMust(t, err)
				want := "sat" + consequenceText(t, f, consequences)
				if canonicalConsequenceOutput(t, string(out)) != canonicalConsequenceOutput(t, want) {
					t.Fatal("C++ wide consequence mismatch", string(out), want)
				}
			}
		})
	}
}

func TestIncrementalSolverConsequencesTrackedScopes(t *testing.T) {
	f, s := newSolverFixture(t)
	q := solverForm(t, f, `q`)
	r := solverForm(t, f, `r`)
	x, err := f.context.MakeApp("x", nil, nil, ast.ASTHandle{})
	incMust(t, err)
	incMust(t, s.AssertTracked(solverForm(t, f, `(= x #x1)`), q))
	check := func() {
		t.Helper()
		status, forms, err := s.GetConsequences(nil, []ast.ASTHandle{x})
		incMust(t, err)
		if status != lbool.True || len(forms) != 1 {
			t.Fatal("tracked fixed value", status, len(forms))
		}
		expected := solverForm(t, f, `(=> q (= x #x1))`)
		if forms[0] != expected {
			t.Fatal("tracked premise lost", consequenceText(t, f, forms))
		}
	}
	check()
	incMust(t, s.Push())
	incMust(t, s.AssertTracked(solverForm(t, f, `(= x #x2)`), r))
	status, _, err := s.GetConsequences(nil, []ast.ASTHandle{x})
	incMust(t, err)
	if status != lbool.False {
		t.Fatal(status)
	}
	core, err := s.UnsatCore()
	incMust(t, err)
	seen := map[ast.ASTHandle]bool{}
	for _, h := range core {
		seen[h] = true
	}
	if len(core) != 2 || !seen[q] || !seen[r] {
		t.Fatal("tracked consequence core", core)
	}
	incMust(t, s.Pop(1))
	check()
	n, err := s.NumAssumptions()
	incMust(t, err)
	if n != 1 {
		t.Fatal("popped tracker retained", n)
	}
}

func TestIncrementalSolverConsequenceAndMutexAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"consequences", "repeat", "mutexes"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 8192 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f, s := newSolverFixture(t)
					p := solverForm(t, f, `p`)
					x, err := f.context.MakeApp("x", nil, nil, ast.ASTHandle{})
					incMust(t, err)
					incMust(t, s.Assert(solverForm(t, f, `(=> p (= x #x3))`)))
					if phase == "repeat" {
						_, _, err = s.GetConsequences([]ast.ASTHandle{p}, []ast.ASTHandle{x})
						incMust(t, err)
					}
					if phase == "mutexes" {
						incMust(t, s.Push())
					}
					f.manager.Budget().FailAfter(stop)
					if phase == "mutexes" {
						_, _, err = s.FindMutexes([]ast.ASTHandle{p})
					} else {
						_, _, err = s.GetConsequences([]ast.ASTHandle{p}, []ast.ASTHandle{x})
					}
					f.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if t.Failed() {
					t.Fatal("reservation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful reservation frontier", phase)
			}
		})
	}
}
