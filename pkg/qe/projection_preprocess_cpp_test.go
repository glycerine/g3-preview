package qe

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestProjectionPreprocessCPPCubeImplication(t *testing.T) {
	for _, mode := range projectionLiteralModes {
		t.Run(mode, func(t *testing.T) {
			c := projectionLiteralFixture(t, mode)
			p := projectionPreprocessOwner(t, c.f)
			m := c.f.r.manager
			fmMust(t, p.extractLiterals(c.md.Core(), c.forms))
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, c.f.r.config, &out, h))
				return out.String()
			}
			var hs []ast.ASTHandle
			for _, r := range *c.forms {
				hs = append(hs, predicateValue(t, c.f, r))
			}
			cube := format(c.f.own(m.MkAndArray(hs)))
			original := format(c.f.own(m.MkAndArray(c.original)))
			script := "(set-logic ALL)\n(declare-const a Bool)\n(declare-const b Bool)\n(declare-const c Bool)\n(declare-const x Int)\n(declare-const y Int)\n(declare-fun P (Bool) Bool)\n(declare-fun Q (Bool) Bool)\n(assert " + cube + ")\n(push 1)\n(assert (not " + original + "))\n(check-sat)\n(pop 1)\n(check-sat)\n"
			if got := projectionCPP(t, script); got != "unsat\nsat" {
				t.Fatal("selected cube implication/nonvacuity", got, script)
			}
		})
	}
}

func TestProjectionPreprocessCPPEqualityAndBooleanProjection(t *testing.T) {
	for _, mode := range []string{"right-selected", "chain", "boolean"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			md := predicateModel(t, f)
			p := projectionPreprocessOwner(t, f)
			x, y := f.constant("x", f.integer), f.constant("y", f.integer)
			a := f.constant("a", f.boolean)
			predicateRegister(t, f, md, x, f.number("3"))
			predicateRegister(t, f, md, y, f.number("3"))
			predicateRegister(t, f, md, a, f.own(m.MkTrue()))
			var original []ast.ASTHandle
			var vars, forms *[]ast.ASTRef
			binders := "((y Int))"
			if mode == "boolean" {
				original = []ast.ASTHandle{a, f.own(m.MkOrArray([]ast.ASTHandle{f.own(m.MkNot(a)), f.own(m.MkArithmeticGE(x, f.number("1")))}))}
				vars = predicateRefs(t, f, a, x)
				forms = predicateRefs(t, f, original...)
				fmMust(t, p.projectBooleans(md.Core(), vars, forms))
				binders = "((a Bool))"
			} else {
				original = []ast.ASTHandle{f.own(m.MkEq(x, y)), f.own(m.MkArithmeticGE(x, f.number("0")))}
				if mode == "chain" {
					original = append(original, f.own(m.MkEq(y, f.number("3"))))
					binders = "((x Int) (y Int))"
				}
				vars = predicateRefs(t, f, x, y)
				forms = predicateRefs(t, f, original...)
				_, err := p.solveEqualities(vars, forms)
				fmMust(t, err)
			}
			arithmeticEvaluate(t, f, md, *forms)
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &out, h))
				return out.String()
			}
			var hs []ast.ASTHandle
			for _, r := range *forms {
				hs = append(hs, predicateValue(t, f, r))
			}
			script := "(set-logic ALL)\n(declare-const x Int)\n(declare-const y Int)\n(declare-const a Bool)\n(assert " + format(f.own(m.MkAndArray(hs))) + ")\n(push 1)\n(assert (not (exists " + binders + " " + format(f.own(m.MkAndArray(original))) + ")))\n(check-sat-using (then simplify qe smt))\n(pop 1)\n(check-sat)\n"
			if got := projectionCPP(t, script); got != "unsat\nsat" {
				t.Fatal("preprocessing projection implication/nonvacuity", got, script)
			}
		})
	}
}
