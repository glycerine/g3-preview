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

func projectionCPP(t *testing.T, script string) string {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in")
	cmd.Stdin = strings.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diagnostics
	if err := cmd.Run(); err != nil {
		t.Fatal(err, out.String(), diagnostics.String(), script)
	}
	if diagnostics.Len() != 0 {
		t.Fatal(diagnostics.String(), script)
	}
	return strings.TrimSpace(out.String())
}

func TestArithmeticProjectionCPPModelBasedImplication(t *testing.T) {
	for _, mode := range arithmeticProjectionModes {
		t.Run(mode, func(t *testing.T) {
			c := arithmeticCase(t, mode)
			p := arithmeticProjectionOwner(t, c.f)
			m := c.f.r.manager
			fmMust(t, p.Project(c.md.Core(), c.vars, c.forms))
			format := func(h ast.ASTHandle) string {
				var s bytes.Buffer
				fmMust(t, writeExpression(m, c.f.r.config, &s, h))
				return s.String()
			}
			original := format(c.f.own(m.MkAndArray(c.original)))
			var projected []ast.ASTHandle
			for _, r := range *c.forms {
				projected = append(projected, predicateValue(t, c.f, r))
			}
			projection := format(c.f.own(m.MkAndArray(projected)))
			sort := "Real"
			if c.sort == c.f.integer {
				sort = "Int"
			}
			quantified := original
			if len(*c.vars) == 0 {
				quantified = "(exists ((x " + sort + ")) " + original + ")"
			}
			declarations := "(set-logic ALL)\n(declare-const x " + sort + ")\n(declare-const y " + sort + ")\n(declare-const z " + sort + ")\n(declare-const b Bool)\n"
			// The old reference's default SMT can return unknown on these
			// quantified arithmetic implications. Full QE is used only by
			// the C++ oracle to decide the counterexample formula; the native
			// projected constraints are submitted exactly as produced.
			script := declarations + "(assert " + projection + ")\n(push 1)\n(assert (not " + quantified + "))\n(check-sat-using (then simplify qe smt))\n(pop 1)\n(check-sat)\n"
			if got := projectionCPP(t, script); got != "unsat\nsat" {
				t.Fatal("projection implication/nonvacuity", got, script)
			}
		})
	}
}

func TestArithmeticProjectionCPPMaximumBounds(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "strict"}[strict], func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			md := predicateModel(t, f)
			p := arithmeticProjectionOwner(t, f)
			x := f.constant("x", f.real)
			predicateRegister(t, f, md, x, f.own(m.ArithmeticRealNumeralText("1")))
			three := f.own(m.ArithmeticRealNumeralText("3"))
			op := int32(2)
			if strict {
				op = 4
			}
			forms := predicateRefs(t, f, f.own(m.MkFamilyApp(f.r.arithmetic, op, nil, []ast.ASTHandle{x, three})))
			bounds := predicateRefs(t, f, ast.ASTHandle{}, ast.ASTHandle{})
			_, err := p.Maximize(md.Core(), *forms, x, (*bounds)[0], (*bounds)[1])
			fmMust(t, err)
			format := func(r ast.ASTRef) string {
				var s bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &s, predicateValue(t, f, r)))
				return s.String()
			}
			script := "(set-logic ALL)\n(declare-const x Real)\n(assert " + format((*forms)[0]) + ")\n(push 1)\n(assert " + format((*bounds)[1]) + ")\n(check-sat)\n(pop 1)\n(assert " + format((*bounds)[0]) + ")\n(check-sat)\n"
			if got := projectionCPP(t, script); got != "unsat\nsat" {
				t.Fatal("maximum improvement and achievable bounds", got, script)
			}
		})
	}
}
