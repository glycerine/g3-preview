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

func TestFourierMotzkinCPPProjectionNormalForms(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"real", "real-large", "integer-unit", "integer-nonunit", "boolean-clause", "one-sided", "strict", "inconsistent", "tautology", "literal-resolution"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			forms, _, sorts, declarations := fmDriverCase(f, mode)
			quantify := func(body ast.ASTHandle) ast.ASTHandle {
				var names []usymbol.Symbol
				for i := range sorts {
					name, err := m.Symbols.Intern(fmt.Sprintf("fm-bound-%d", i))
					fmMust(t, err)
					names = append(names, name)
				}
				return f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: sorts, Names: names, Body: body}))
			}
			// Public QE-lite rotates the flattened AND's last child into the
			// parent slot. Prepare the source input for the native phase order.
			sourceForms := append([]ast.ASTHandle(nil), forms[1:]...)
			sourceForms = append(sourceForms, forms[0])
			original := quantify(f.own(m.MkAndArray(sourceForms)))
			conjs := f.conjunctions(forms...)
			d := f.eliminator()
			fmMust(t, d.Apply(conjs))
			fmMust(t, d.Close())
			var outputs []ast.ASTHandle
			for _, ref := range *conjs {
				h, err := m.RefGet(ref)
				fmMust(t, err)
				outputs = append(outputs, h)
			}
			projected := quantify(f.own(m.MkAndArray(outputs)))
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
			// SMT-LIB prints native negative numeral nodes as (- n). Parsing
			// that text creates unary-minus applications. Source FM requires
			// numeral coefficients, so simplify restores the original native
			// arithmetic normal form before the source elimination pass.
			want, got := normalize(original, "(then simplify qe-light simplify)"), normalize(projected, "simplify")
			if got != want {
				t.Fatalf("Go FM/source QE-lite projection mismatch\nGo: %s\nC++: %s", got, want)
			}
		})
	}
}
