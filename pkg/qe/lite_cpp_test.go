package qe

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestLiteCPPFullTraversalNormalForms(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"real", "real-large", "integer-unit", "integer-nonunit", "boolean-clause", "one-sided", "strict", "inconsistent", "tautology", "literal-resolution", "universal", "array", "nested-pattern"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			l := f.lite(true)
			forms, _, sorts, declarations := fmDriverCase(f, mode)
			forall := mode == "universal"
			if forall {
				x := f.variable(0, f.integer)
				sorts = []ast.ASTHandle{f.integer}
				forms = []ast.ASTHandle{f.own(m.MkNot(f.own(m.MkEq(x, f.number("3"))))), f.predicate("lite-universal", x)}
				declarations = "(declare-fun lite-universal (Int) Bool)\n"
			}
			if mode == "array" {
				sort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
				sorts = []ast.ASTHandle{sort}
				a := f.variable(0, sort)
				key, other := f.constant("lite-key", f.integer), f.constant("lite-other-key", f.integer)
				family, err := m.ArrayFamilyID()
				fmMust(t, err)
				read := func(index ast.ASTHandle) ast.ASTHandle {
					return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpSelect, ast.SliceArguments([]ast.ASTHandle{a, index})))
				}
				forms = []ast.ASTHandle{f.own(m.MkEq(read(key), f.number("7"))), f.own(m.MkEq(read(other), f.number("9")))}
				declarations = "(declare-const lite-key Int)\n(declare-const lite-other-key Int)\n"
			}
			body := f.own(m.MkAndArray(forms))
			if forall {
				body = f.own(m.MkOrArray(forms))
			}
			var names []usymbol.Symbol
			for i := range sorts {
				name, err := m.Symbols.Intern(fmt.Sprint("lite-bound-", i))
				fmMust(t, err)
				names = append(names, name)
			}
			original := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: forall, Sorts: sorts, Names: names, Body: body}))
			if mode == "nested-pattern" {
				x, y := f.variable(0, f.integer), f.variable(1, f.integer)
				innerName, err := m.Symbols.Intern("lite-inner")
				fmMust(t, err)
				outerName, err := m.Symbols.Intern("lite-outer")
				fmMust(t, err)
				pattern := f.own(m.MkPattern([]ast.ASTHandle{f.predicate("lite-trigger", x)}))
				innerBody := f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(x, y)), f.predicate("lite-p", x)}))
				inner := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{innerName}, Body: innerBody, Patterns: []ast.ASTHandle{pattern}}))
				original = f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{outerName}, Body: inner}))
				declarations = "(declare-fun lite-trigger (Int) Bool)\n(declare-fun lite-p (Int) Bool)\n"
			}
			formula := f.conjunctions(original)
			proof := f.conjunctions(ast.ASTHandle{})
			fmMust(t, l.Rewrite((*formula)[0], (*proof)[0]))
			fmMust(t, l.Close())
			projected, err := m.RefGet((*formula)[0])
			fmMust(t, err)
			normalize := func(h ast.ASTHandle, strategy string) string {
				var expression bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &expression, h))
				script := "(set-logic ALL)\n" + declarations + "(assert " + expression.String() + ")\n(apply " + strategy + ")\n"
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				var out, diagnostics bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &diagnostics
				if err := cmd.Run(); err != nil {
					t.Fatal(err, out.String(), diagnostics.String(), script)
				}
				return regexp.MustCompile(`:depth [0-9]+`).ReplaceAllString(strings.Join(strings.Fields(out.String()), " "), ":depth")
			}
			// Restore native negative numeral nodes after SMT-LIB serialization
			// before the source FM phase, without another QE pass on Go output.
			want, got := normalize(original, "(then simplify qe-light simplify)"), normalize(projected, "simplify")
			if got != want {
				t.Fatalf("native full QE-lite/source mismatch\nGo: %s\nC++: %s", got, want)
			}
		})
	}
}
