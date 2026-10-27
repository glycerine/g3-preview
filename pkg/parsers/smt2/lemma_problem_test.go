package smt2

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/smtcore"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2ContextLemmaProblemsGoAndCPP(t *testing.T) {
	for _, tc := range []struct {
		name, logic string
		sort        bool
		conclusion  int
		want        string
	}{
		{"conflict", "QF_UFLRA", false, 0, "sat"},
		{"positive", "QF_UFLRA", false, 1, "unsat"},
		{"negative", "QF_UFLRA", false, 2, "unsat"},
		{"true", "QF_UFLRA", false, 3, "unsat"},
		{"uf-sort", "QF_UF", true, 1, "unsat"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			c, err := cmd_context.NewStandaloneContext()
			must(err)
			t.Cleanup(func() { must(c.Close()) })
			m, err := c.Manager()
			must(err)
			checker, err := smtcore.NewSMTCheckerContext(m)
			must(err)
			state, err := smtcore.NewSMTInternalizerState(m, checker, nil)
			must(err)
			t.Cleanup(func() { must(state.Close()) })
			pin := func(h ast.ASTHandle, err error) ast.ASTHandle {
				t.Helper()
				must(err)
				r, err := m.NewRef(h)
				must(err)
				t.Cleanup(func() { must(m.DelRef(&r)) })
				return h
			}
			symbol := func(name string) usymbol.Symbol { v, err := m.Symbols.Intern(name); must(err); return v }
			b, err := m.MkBoolSort()
			must(err)
			p := pin(m.MkConst(symbol("lemma-p"), b))
			q := pin(m.MkConst(symbol("lemma-q"), b))
			must(state.Internalize(p, true))
			must(state.Internalize(q, true))
			pl, err := state.Literal(p)
			must(err)
			ql, err := state.Literal(q)
			must(err)
			var sort ast.ASTHandle
			if tc.sort {
				sort = pin(m.MkUninterpretedSort(symbol("lemma-S"), nil))
			} else {
				family, err := m.RegisterArithmeticPlugin()
				must(err)
				sort = pin(m.MkFamilySort(family, 0, nil))
			}
			x, y := pin(m.MkConst(symbol("lemma-x"), sort)), pin(m.MkConst(symbol("lemma-y"), sort))
			xn, err := state.MkEnode(x, false, false, true)
			must(err)
			yn, err := state.MkEnode(y, false, false, true)
			must(err)
			conclusion := smtcore.SMTFalseLiteral
			switch tc.conclusion {
			case 1:
				conclusion = pl
			case 2:
				conclusion = ql.Negated()
			case 3:
				conclusion = smtcore.SMTTrueLiteral
			}
			var dump bytes.Buffer
			must(state.DisplayLemmaAsSMTProblem(&dump, []smtcore.SMTLiteral{pl, ql.Negated()}, []smtcore.SMTEnodePair{{First: xn, Second: yn}}, conclusion, symbol(tc.logic)))
			script := dump.String() + "(check-sat)\n"
			target, err := cmd_context.NewStandaloneContext()
			must(err)
			defer func() { must(target.Close()) }()
			var out bytes.Buffer
			must(ParseCommands(target, strings.NewReader(script), &out))
			if strings.TrimSpace(out.String()) != tc.want {
				t.Fatal("Go lemma result", out.String(), tc.want, script)
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				result, err := cmd.CombinedOutput()
				must(err)
				if strings.TrimSpace(string(result)) != tc.want {
					t.Fatal("C++ lemma result", string(result), tc.want, script)
				}
			}
		})
	}
}
