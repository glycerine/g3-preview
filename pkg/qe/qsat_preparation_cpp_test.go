package qe

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestQSATPreparationCPPPrefixEquivalence(t *testing.T) {
	for _, mode := range qsatPreparationModes {
		for _, kind := range []qsatMode{qsatSAT, qsatQE} {
			name := "sat"
			if kind == qsatQE {
				name = "qe"
			}
			t.Run(mode+"/"+name, func(t *testing.T) {
				c := qsatPrepareFixture(t, mode)
				p := qsatPrepareOwner(t, c.f)
				m := c.f.r.manager
				fmMust(t, p.hoist(kind, c.formula))
				format := func(h ast.ASTHandle) string {
					var out bytes.Buffer
					fmMust(t, writeExpression(m, c.f.r.config, &out, h))
					return out.String()
				}
				// Rebuild the prefix independently as SMT-LIB text. Original free
				// parameters stay free, including those in SAT's existential block.
				// This checks equivalence for every assignment to those parameters.
				rebuilt := format(predicateValue(t, c.f, c.formula))
				for i := len(p.variables) - 1; i >= 0; i-- {
					var binders []string
					for _, r := range p.variables[i] {
						h := predicateValue(t, c.f, r)
						if h != c.free {
							sort, err := m.GetSort(h)
							fmMust(t, err)
							ty := "Bool"
							if sort == c.f.integer {
								ty = "Int"
							} else if sort == c.f.real {
								ty = "Real"
							}
							binders = append(binders, "("+format(h)+" "+ty+")")
						}
					}
					if len(binders) > 0 {
						q := "exists"
						if i%2 == 1 {
							q = "forall"
						}
						rebuilt = "(" + q + " (" + strings.Join(binders, " ") + ") " + rebuilt + ")"
					}
				}
				declarations := "(set-logic ALL)\n(declare-const free Bool)\n(declare-fun opaque (Bool) Bool)\n"
				script := declarations + "(assert (not (= " + format(c.original) + " " + rebuilt + ")))\n(check-sat)\n"
				if got := projectionCPP(t, script); got != "unsat" {
					t.Fatal("quantifier-prefix equivalence", got, script)
				}
			})
		}
	}
}
