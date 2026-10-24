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
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestFMConstraintNormalizationCPPLogicalEquivalence(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"integer-sum", "integer-ge", "integer-not-ge", "integer-not-le", "real-fraction", "real-strict", "real-large", "coerced-int-strict", "mixed", "clause"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			form, _, _, _ := fmConstraintCase(f, mode)
			s := f.store()
			c, err := s.parse(form)
			fmMust(t, err)
			fmMust(t, s.normalize(c))
			ref, err := s.expression(c)
			fmMust(t, err)
			t.Cleanup(func() { fmMust(t, m.DelRef(&ref)) })
			normalized, err := m.RefGet(ref)
			fmMust(t, err)
			fmMust(t, s.Close())
			// Ask C++ for a counterexample to the exact native constraint
			// conversion. No source QE pass repairs the native result.
			difference := f.own(m.MkNot(f.own(m.MkEq(form, normalized))))
			xsort, ysort := f.integer, f.integer
			if strings.HasPrefix(mode, "real") {
				xsort, ysort = f.real, f.real
			}
			if mode == "mixed" {
				ysort = f.real
			}
			var names []usymbol.Symbol
			for _, name := range []string{"fm-y", "fm-x"} {
				s, err := m.Symbols.Intern(name)
				fmMust(t, err)
				names = append(names, s)
			}
			query := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{ysort, xsort}, Names: names, Body: difference}))
			var expression bytes.Buffer
			fmMust(t, writeExpression(m, f.r.config, &expression, query))
			script := "(set-logic ALL)\n(declare-const fm-p Bool)\n(declare-const fm-q Bool)\n(assert " + expression.String() + ")\n(check-sat)\n"
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
			if strings.TrimSpace(out.String()) != "unsat" {
				t.Fatalf("C++ found or could not exclude a normalization counterexample: %s\n%s\n%s", out.String(), diagnostics.String(), script)
			}
		})
	}
}
