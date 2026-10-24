package qe

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

const datatypeProjectionDeclarations = `(set-logic ALL)
(declare-datatypes () ((DPair (pair (first Int) (second Bool))) (DList (nil) (cons (head Int) (tail DList))) (DColor (red) (blue))))
`

func datatypeProjectionSort(d *datatypeProjectionFixture, h ast.ASTHandle) string {
	switch h {
	case d.f.integer:
		return "Int"
	case d.f.boolean:
		return "Bool"
	case d.pair:
		return "DPair"
	case d.list:
		return "DList"
	case d.color:
		return "DColor"
	}
	return "unknown-sort"
}
func TestDatatypeProjectionCPPExistentialImplication(t *testing.T) {
	datatypeProjectionCPPImplication(t, false)
}
func TestModelBasedProjectionCPPDatatypeImplication(t *testing.T) {
	datatypeProjectionCPPImplication(t, true)
}
func datatypeProjectionCPPImplication(t *testing.T, driver bool) {
	for _, mode := range datatypeProjectionModes {
		t.Run(mode, func(t *testing.T) {
			c := datatypeCase(t, mode)
			d := c.d
			f := d.f
			m := f.r.manager
			before := len(d.md.Core().ConstantDeclarations)
			ok := true
			var err error
			if driver {
				p := modelProjectionOwner(t, f)
				vars := predicateRefs(t, f, c.target)
				err = p.Project(true, vars, d.md.Core(), c.forms)
			} else {
				p := datatypeProjectionOwner(t, d)
				ok, err = p.ProjectVariable(d.md.Core(), c.target, c.additional, c.forms)
			}
			fmMust(t, err)
			if !ok {
				t.Fatal("projection refused")
			}
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &out, h))
				return out.String()
			}
			var script strings.Builder
			script.WriteString(datatypeProjectionDeclarations)
			for _, row := range []string{"dx DList", "dy DList", "dp DPair", "dc DColor", "uninterpreted-model DList"} {
				script.WriteString("(declare-const " + row + ")\n")
			}
			for _, decl := range d.md.Core().ConstantDeclarations[before:] {
				name, err := m.GetASTName(decl)
				fmMust(t, err)
				text, err := m.Symbols.String(name)
				fmMust(t, err)
				sort, err := m.GetRange(decl)
				fmMust(t, err)
				script.WriteString("(declare-const " + text + " " + datatypeProjectionSort(d, sort) + ")\n")
			}
			var hs []ast.ASTHandle
			for _, r := range *c.forms {
				hs = append(hs, predicateValue(t, f, r))
			}
			result := format(f.own(m.MkAndArray(hs)))
			original := format(f.own(m.MkAndArray(c.original)))
			sort, err := m.GetSort(c.target)
			fmMust(t, err)
			script.WriteString("(assert " + result + ")\n(push 1)\n(assert (not (exists ((" + format(c.target) + " " + datatypeProjectionSort(d, sort) + ")) " + original + ")))\n(check-sat-using (then simplify qe smt))\n(pop 1)\n(check-sat)\n")
			if got := projectionCPP(t, script.String()); got != "unsat\nsat" {
				t.Fatal("datatype projection implication/nonvacuity", got, script.String())
			}
		})
	}
}

func TestDatatypeProjectionCPPForeignEqualityEquivalence(t *testing.T) {
	for _, mode := range []string{"left", "right", "both"} {
		t.Run(mode, func(t *testing.T) {
			d := newDatatypeProjectionFixture(t)
			f := d.f
			m := f.r.manager
			p := datatypeProjectionOwner(t, d)
			x, y := f.constant("fx", f.integer), f.constant("fy", f.integer)
			b, c := f.constant("fb", f.boolean), f.constant("fc", f.boolean)
			r := f.constant("fr", d.pair)
			left, right := d.app(d.pairCtor, x, b), r
			vars := predicateRefs(t, f, x, b, r)
			if mode == "right" {
				left, right = right, left
			}
			if mode == "both" {
				right = d.app(d.pairCtor, y, c)
				vars = predicateRefs(t, f, x, y)
			}
			original := f.own(m.MkEq(left, right))
			forms := predicateRefs(t, f, original)
			changed, err := p.Solve(d.md.Core(), vars, forms)
			fmMust(t, err)
			if !changed {
				t.Fatal("foreign equality not lifted")
			}
			format := func(h ast.ASTHandle) string {
				var out bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &out, h))
				return out.String()
			}
			var hs []ast.ASTHandle
			for _, r := range *forms {
				hs = append(hs, predicateValue(t, f, r))
			}
			result := format(f.own(m.MkAndArray(hs)))
			script := datatypeProjectionDeclarations + "(declare-const fx Int)\n(declare-const fy Int)\n(declare-const fb Bool)\n(declare-const fc Bool)\n(declare-const fr DPair)\n(push 1)\n(assert (not (= " + format(original) + " " + result + ")))\n(check-sat)\n(pop 1)\n(assert " + result + ")\n(check-sat)\n"
			if got := projectionCPP(t, script); got != "unsat\nsat" {
				t.Fatal("foreign-field equivalence/nonvacuity", got, script)
			}
		})
	}
}
