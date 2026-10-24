package qe

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
)

func TestPredicateAbstractionCPPDefinitionEquivalence(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"conjunction", "disjunction", "boolean-ite", "term-ite", "distinct", "shared", "negative-assumption"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			p := predicateOwner(t, f)
			x, y := f.constant("x", f.integer), f.constant("y", f.integer)
			b := f.constant("b", f.boolean)
			for _, h := range []ast.ASTHandle{x, y, b} {
				fmMust(t, p.SetExpressionLevel(h, MaxLevel{0, NoQuantifierLevel}))
			}
			atom := f.predicate("P", x)
			equal := f.own(m.MkEq(x, y))
			formula := f.own(m.MkAndArray([]ast.ASTHandle{atom, equal, b}))
			switch mode {
			case "disjunction":
				formula = f.own(m.MkOrArray([]ast.ASTHandle{atom, equal, b}))
			case "boolean-ite":
				formula = f.own(m.MkIte(b, atom, equal))
			case "term-ite":
				formula = f.own(m.MkEq(f.own(m.MkIte(b, x, y)), f.number("0")))
			case "distinct":
				formula = f.own(m.MkDistinct([]ast.ASTHandle{x, y, f.number("0")}))
			case "shared":
				formula = f.own(m.MkImplies(atom, f.own(m.MkOrArray([]ast.ASTHandle{atom, b}))))
			case "negative-assumption":
				formula = f.own(m.MkNot(formula))
			}
			defs := predicateRefs(t, f)
			result := predicateRefs(t, f, ast.ASTHandle{})
			level := EmptyMaxLevel()
			if mode == "negative-assumption" {
				fmMust(t, p.MakeAssumptionLiteral(formula, nil, MaxLevel{2, NoQuantifierLevel}, defs, (*result)[0]))
			} else {
				fmMust(t, p.AbstractAtoms(formula, &level, defs))
				fmMust(t, p.MakeAbstract(formula, (*result)[0]))
			}
			abstract := predicateValue(t, f, (*result)[0])
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &out, h))
				return out.String()
			}
			var script strings.Builder
			script.WriteString("(set-logic ALL)\n(declare-const x Int)\n(declare-const y Int)\n(declare-const b Bool)\n(declare-fun P (Int) Bool)\n")
			for _, r := range *defs {
				d := predicateValue(t, f, r)
				fresh, err := m.GetArg(d, 0)
				fmMust(t, err)
				script.WriteString("(declare-const " + format(fresh) + " Bool)\n")
			}
			for _, r := range *defs {
				script.WriteString("(assert " + format(predicateValue(t, f, r)) + ")\n")
			}
			script.WriteString("(push 1)\n(assert (not (= " + format(formula) + " " + format(abstract) + ")))\n(check-sat)\n(pop 1)\n(assert " + format(formula) + ")\n(assert " + format(abstract) + ")\n(check-sat)\n")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, reference, "-in")
			cmd.Stdin = strings.NewReader(script.String())
			var out, diagnostics bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &diagnostics
			if err := cmd.Run(); err != nil {
				t.Fatal(err, out.String(), diagnostics.String(), script.String())
			}
			if strings.TrimSpace(out.String()) != "unsat\nsat" || diagnostics.Len() != 0 {
				t.Fatal("definition equivalence or satisfiable extension failed", out.String(), diagnostics.String(), script.String())
			}
		})
	}
}
