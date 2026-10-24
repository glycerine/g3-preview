package qe

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestArrayProjectionCPPWitnessAndRefusalEquivalence(t *testing.T) {
	for _, mode := range arrayProjectionModes {
		t.Run(mode, func(t *testing.T) {
			c := arrayCase(t, mode)
			p := arrayProjectionOwner(t, c)
			m := c.f.r.manager
			before := len(c.md.Core().ConstantDeclarations)
			ok, err := p.ProjectVariable(c.md.Core(), c.target, c.additional, c.lits)
			fmMust(t, err)
			if ok != c.success {
				t.Fatal(ok, c.success)
			}
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, c.f.r.config, &out, h))
				return out.String()
			}
			element := "Int"
			if c.boolean {
				element = "Bool"
			}
			if c.nestedRange {
				element = "(Array Int Int)"
			}
			sort := "(Array Int " + element + ")"
			if c.multi {
				sort = "(Array Int Int " + element + ")"
			}
			var script strings.Builder
			script.WriteString("(set-logic ALL)\n(declare-const A " + sort + ")\n(declare-const B " + sort + ")\n")
			for _, name := range []string{"i", "j", "k", "l"} {
				script.WriteString("(declare-const " + name + " Int)\n")
			}
			for _, decl := range c.md.Core().ConstantDeclarations[before:] {
				name, err := m.GetASTName(decl)
				fmMust(t, err)
				text, err := m.Symbols.String(name)
				fmMust(t, err)
				s, err := m.GetRange(decl)
				fmMust(t, err)
				ty := "Int"
				if s == c.f.boolean {
					ty = "Bool"
				}
				if c.nestedRange {
					ty = "(Array Int Int)"
				}
				script.WriteString("(declare-const " + text + " " + ty + ")\n")
			}
			var hs []ast.ASTHandle
			for _, r := range *c.lits {
				hs = append(hs, predicateValue(t, c.f, r))
			}
			result := format(c.f.own(m.MkAndArray(hs)))
			original := format(c.f.own(m.MkAndArray(c.original)))
			expected := original
			if ok {
				// This constructive statement implies the existential claim.
				// The source executable returns unknown on many quantified-array
				// queries, even with qe. SMT-LIB let performs the substitution
				// independently of the native replacement implementation.
				expected = "(let ((A " + format(c.witness) + ")) " + original + ")"
			}
			script.WriteString("(assert " + result + ")\n(push 1)\n(assert (not " + expected + "))\n(check-sat)\n(pop 1)\n(check-sat)\n")
			if got := projectionCPP(t, script.String()); got != "unsat\nsat" {
				t.Fatal("array projection implication/nonvacuity", got, script.String())
			}
		})
	}
}
