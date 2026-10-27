package portfolio_test

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func enumStatus(s lbool.LBool) string {
	if s == lbool.True {
		return "sat"
	}
	if s == lbool.False {
		return "unsat"
	}
	return "unknown"
}
func enumCanonical(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
}
func TestEnum2BVSolverQFFDReference(t *testing.T) {
	for _, which := range []string{"models-scopes", "consequences", "named-core"} {
		t.Run(which, func(t *testing.T) {
			f := enumSolverFixture(t)
			s, m := f.s, f.manager
			_, h, x := enumGoal(t, f)
			script := `(set-option :produce-models true)(set-option :produce-unsat-cores true)(set-logic QF_FD)(declare-datatypes () ((E red amber green)))(declare-const x E)`
			var out strings.Builder
			switch which {
			case "models-scopes":
				enumMust(t, s.Assert(h))
				script += `(assert (= x green))`
				for phase := 0; phase < 3; phase++ {
					switch phase {
					case 1:
						enumMust(t, s.Push())
						not, e := m.MkNot(h)
						enumMust(t, e)
						enumMust(t, s.Assert(not))
						script += `(push 1)(assert (not (= x green)))`
					case 2:
						enumMust(t, s.Pop(1))
						script += `(pop 1)`
					}
					status, e := s.CheckSat(nil)
					enumMust(t, e)
					fmt.Fprintln(&out, enumStatus(status))
					script += `(check-sat)`
					if phase != 1 {
						md, e := s.AcquireModel()
						enumMust(t, e)
						if md == nil {
							t.Fatal("missing model")
						}
						d, e := m.GetDecl(x)
						enumMust(t, e)
						v, e := md.ConstantInterpretation(d)
						enumMust(t, e)
						vd, e := m.GetDecl(v)
						enumMust(t, e)
						name, e := m.GetASTName(vd)
						enumMust(t, e)
						text, e := m.Symbols.String(name)
						enumMust(t, e)
						fmt.Fprintf(&out, "((x %s))\n", text)
						enumMust(t, md.DecRef())
						script += `(get-value (x))`
					}
				}
			case "consequences":
				enumMust(t, s.Assert(h))
				script += `(assert (= x green))(get-consequences () (x))`
				status, conseq, e := s.GetConsequences(nil, []ast.ASTHandle{x})
				enumMust(t, e)
				fmt.Fprintln(&out, enumStatus(status))
				printer, e := ast.NewSimpleASTPrinterContext(m, &out, nil)
				enumMust(t, e)
				for _, h := range conseq {
					enumMust(t, printer.DisplayExpr(&out, h, 0))
					out.WriteByte('\n')
				}
			case "named-core":
				first := enumTag(t, m, "chosen")
				second := enumTag(t, m, "clash")
				enumMust(t, s.AssertTracked(h, first))
				not, e := m.MkNot(h)
				enumMust(t, e)
				enumMust(t, s.AssertTracked(not, second))
				status, e := s.CheckSat(nil)
				enumMust(t, e)
				fmt.Fprintln(&out, enumStatus(status))
				core, e := s.UnsatCore()
				enumMust(t, e)
				out.WriteByte('(')
				for i, tag := range core {
					d, e := m.GetDecl(tag)
					enumMust(t, e)
					name, e := m.GetASTName(d)
					enumMust(t, e)
					text, e := m.Symbols.String(name)
					enumMust(t, e)
					if i > 0 {
						out.WriteByte(' ')
					}
					out.WriteString(text)
				}
				out.WriteString(")\n")
				script += `(assert (! (= x green) :named chosen))(assert (! (not (= x green)) :named clash))(check-sat)(get-unsat-core)`
			}
			reference := os.Getenv("G3_CPP_REFERENCE")
			if reference == "" {
				t.Log("native checks completed; set G3_CPP_REFERENCE for the prebuilt oracle")
				return
			}
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(script)
			cpp, e := cmd.CombinedOutput()
			if e != nil {
				t.Fatal(e, string(cpp))
			}
			actual, expected := out.String(), string(cpp)
			if which == "named-core" {
				// Core membership is the contract; the two actual backends
				// discover these named assumptions in opposite orders.
				coreKey := func(s string) string {
					lines := strings.SplitN(strings.TrimSpace(s), "\n", 2)
					if len(lines) != 2 {
						return s
					}
					labels := strings.Fields(strings.Trim(strings.TrimSpace(lines[1]), "()"))
					sort.Strings(labels)
					return lines[0] + " (" + strings.Join(labels, " ") + ")"
				}
				actual, expected = coreKey(actual), coreKey(expected)
			}
			if enumCanonical(actual) != enumCanonical(expected) {
				t.Fatalf("Go enum wrapper + SMT backend: %s\nC++ QF_FD pipeline: %s", out.String(), cpp)
			}
		})
	}
}
